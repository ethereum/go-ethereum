package main

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/tests"
)

var blocktestCmd = cli.Command{
	Action: runblocktest,
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

func runblocktest(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		utils.Fatalf("Usage: ethereum blocktest <path-to-test-file> <test-name> {rpc, norpc}")
	}
	file, testname, startrpc := ctx.Args()[0], ctx.Args()[1], ctx.Args()[2]

	bt, err := tests.LoadBlockTests(file)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	test, ok := bt[testname]
	if !ok {
		utils.Fatalf("Test file does not contain test named %q", testname)
	}

	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	cfg.NewDB = func(path string) (common.Database, error) { return ethdb.NewMemDatabase() }
	cfg.MaxPeers = 0 // disable network
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	// import the genesis block
	ethereum.ResetWithGenesisBlock(test.Genesis)

	// import pre accounts
	if err := test.InsertPreState(ethereum.StateDb()); err != nil {
		utils.Fatalf("could not insert genesis accounts: %v", err)
	}

	// insert the test blocks, which will execute all transactions
	chain := ethereum.ChainManager()
	if err := chain.InsertChain(test.Blocks); err != nil {
		utils.Fatalf("Block Test load error: %v", err)
	} else {
		fmt.Println("Block Test chain loaded, starting ethereum.")
	}
	if startrpc == "rpc" {
		startEth(ctx, ethereum)
		utils.StartRPC(ethereum, ctx)
		ethereum.WaitForShutdown()
	} else {
		startEth(ctx, ethereum)
	}
}
