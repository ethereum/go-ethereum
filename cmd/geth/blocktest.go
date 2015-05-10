package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/tests"
)

var blocktestCmd = cli.Command{
	Action: runBlockTest,
	Name:   "blocktest",
	Usage:  `loads a block test file`,
	Description: `
The first argument should be a block test file.
The second argument is the name of a block test from the file.

The block test will be loaded into an in-memory database.
If loading succeeds, the RPC server is started. Clients will
be able to interact with the chain defined by the test.
`,
}

func runBlockTest(ctx *cli.Context) {
	var (
		file, testname string
		rpc            bool
	)
	args := ctx.Args()
	switch {
	case len(args) == 1:
		file = args[0]
	case len(args) == 2:
		file, testname = args[0], args[1]
	case len(args) == 3:
		file, testname = args[0], args[1]
		rpc = true
	default:
		utils.Fatalf(`Usage: ethereum blocktest <path-to-test-file> [ <test-name> [ "rpc" ] ]`)
	}
	bt, err := tests.LoadBlockTests(file)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	// run all tests if no test name is specified
	if testname == "" {
		ecode := 0
		for name, test := range bt {
			fmt.Printf("----------------- Running Block Test %q\n", name)
			ethereum, err := runOneBlockTest(ctx, test)
			if err != nil {
				fmt.Println(err)
				fmt.Println("FAIL")
				ecode = 1
			}
			if ethereum != nil {
				ethereum.Stop()
				ethereum.WaitForShutdown()
			}
		}
		os.Exit(ecode)
		return
	}
	// otherwise, run the given test
	test, ok := bt[testname]
	if !ok {
		utils.Fatalf("Test file does not contain test named %q", testname)
	}
	ethereum, err := runOneBlockTest(ctx, test)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	defer ethereum.Stop()
	if rpc {
		fmt.Println("Block Test post state validated, starting RPC interface.")
		startEth(ctx, ethereum)
		utils.StartRPC(ethereum, ctx)
		ethereum.WaitForShutdown()
	}
}

func runOneBlockTest(ctx *cli.Context, test *tests.BlockTest) (*eth.Ethereum, error) {
	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	cfg.NewDB = func(path string) (common.Database, error) { return ethdb.NewMemDatabase() }
	cfg.MaxPeers = 0 // disable network
	cfg.Shh = false  // disable whisper
	cfg.NAT = nil    // disable port mapping

	ethereum, err := eth.New(cfg)
	if err != nil {
		return nil, err
	}
	if err := ethereum.Start(); err != nil {
		return nil, err
	}

	// import the genesis block
	ethereum.ResetWithGenesisBlock(test.Genesis)

	// import pre accounts
	statedb, err := test.InsertPreState(ethereum)
	if err != nil {
		return ethereum, fmt.Errorf("InsertPreState: %v", err)
	}

	if err := test.TryBlocksInsert(ethereum.ChainManager()); err != nil {
		return ethereum, fmt.Errorf("Block Test load error: %v", err)
	}

	if err := test.ValidatePostState(statedb); err != nil {
		return ethereum, fmt.Errorf("post state validation failed: %v", err)
	}
	return ethereum, nil
}
