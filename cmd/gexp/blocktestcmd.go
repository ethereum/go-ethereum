// Copyright 2015 The go-expanse Authors
// This file is part of go-expanse.
//
// go-expanse is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-expanse is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-expanse. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/expanse-project/go-expanse/cmd/utils"
	"github.com/expanse-project/go-expanse/eth"
	"github.com/expanse-project/go-expanse/ethdb"
	"github.com/expanse-project/go-expanse/tests"
)

var blocktestCommand = cli.Command{
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
		utils.Fatalf(`Usage: expanse blocktest <path-to-test-file> [ <test-name> [ "rpc" ] ]`)
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
			expanse, err := runOneBlockTest(ctx, test)
			if err != nil {
				fmt.Println(err)
				fmt.Println("FAIL")
				ecode = 1
			}
			if expanse != nil {
				expanse.Stop()
				expanse.WaitForShutdown()
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
	expanse, err := runOneBlockTest(ctx, test)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	if rpc {
		fmt.Println("Block Test post state validated, starting RPC interface.")
		startEth(ctx, expanse)
		utils.StartRPC(expanse, ctx)
		expanse.WaitForShutdown()
	}
}

func runOneBlockTest(ctx *cli.Context, test *tests.BlockTest) (*exp.Expanse, error) {
	cfg := utils.MakeEthConfig(ClientIdentifier, Version, ctx)
	cfg.NewDB = func(path string) (ethdb.Database, error) { return ethdb.NewMemDatabase() }
	cfg.MaxPeers = 0 // disable network
	cfg.Shh = false  // disable whisper
	cfg.NAT = nil    // disable port mapping

	expanse, err := exp.New(cfg)
	if err != nil {
		return nil, err
	}

	// import the genesis block
	expanse.ResetWithGenesisBlock(test.Genesis)
	// import pre accounts
	_, err = test.InsertPreState(expanse)

	if err != nil {
		return expanse, fmt.Errorf("InsertPreState: %v", err)
	}

	cm := expanse.ChainManager()
	validBlocks, err := test.TryBlocksInsert(cm)
	if err != nil {
		return expanse, fmt.Errorf("Block Test load error: %v", err)
	}
	newDB := cm.State()
	if err := test.ValidatePostState(newDB); err != nil {
		return expanse, fmt.Errorf("post state validation failed: %v", err)
	}
	return expanse, test.ValidateImportedHeaders(cm, validBlocks)

}
