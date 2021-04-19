// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	genesis, halfchain, fullchain, nodekey string
)

func init() {
	if err := filepaths(); err != nil {
		panic(err)
	}
}

func TestSetupGeth(t *testing.T) {
	if _, err := setupGeth(); err != nil {
		t.Fatalf("could not create geth: %v", err)
	}
}

func TestAll(t *testing.T) {
	runAll(t)
}

func TestEth65(t *testing.T) {
	runEth65(t)
}

func TestEth66(t *testing.T) {
	runEth66(t)
}

func TestStatus(t *testing.T) {
	runTest(t, "TestStatus")
}

func TestStatus_66(t *testing.T) {
	runTest(t, "TestStatus_66")
}

func TestGetBlockHeaders(t *testing.T) {
	runTest(t, "TestGetBlockHeaders")
}

func TestGetBlockHeaders_66(t *testing.T) {
	runTest(t, "TestGetBlockHeaders_66")
}

func TestSimultaneousRequests_66(t *testing.T) {
	runTest(t, "TestSimultaneousRequests_66")
}

func TestSameRequestID_66(t *testing.T) {
	runTest(t, "TestSameRequestID_66")
}

func TestZeroRequestID_66(t *testing.T) {
	runTest(t, "TestZeroRequestID_66")
}

func TestGetBlockBodies(t *testing.T) {
	runTest(t, "TestGetBlockBodies")
}

func TestGetBlockBodies_66(t *testing.T) {
	runTest(t, "TestGetBlockBodies_66")
}

func TestBroadcast(t *testing.T) {
	runTest(t, "TestBroadcast")
}

func TestBroadcast_66(t *testing.T) {
	runTest(t, "TestBroadcast_66")
}

func TestLargeAnnounce(t *testing.T) {
	runTest(t, "TestLargeAnnounce")
}

func TestLargeAnnounce_66(t *testing.T) {
	runTest(t, "TestLargeAnnounce_66")
}

func TestOldAnnounce(t *testing.T) {
	runTest(t, "TestOldAnnounce")
}

func TestMaliciousHandshake(t *testing.T) {
	runTest(t, "TestMaliciousHandshake")
}

func TestMaliciousStatus(t *testing.T) {
	runTest(t, "TestMaliciousStatus")
}

func TestMaliciousHandshake_66(t *testing.T) {
	runTest(t, "TestMaliciousHandshake_66")
}

func TestMaliciousStatus_66(t *testing.T) {
	runTest(t, "TestMaliciousStatus_66")
}

func TestTransaction(t *testing.T) {
	runTest(t, "TestTransaction")
}

func TestTransaction_66(t *testing.T) {
	runTest(t, "TestTransaction_66")
}

func TestMaliciousTx(t *testing.T) {
	runTest(t, "TestMaliciousTx")
}

func TestMaliciousTx_66(t *testing.T) {
	runTest(t, "TestMaliciousTx_66")
}

func runAll(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	// wait for geth to start up
	time.Sleep(time.Second * 5)

	suite := newTestSuite(t, geth.Server().Self())
	failures := make(map[string]string)
	for _, test := range suite.AllEthTests() {
		failed, output := utesting.Run(test)
		if failed {
			failures[test.Name] = output
		}
	}
	if len(failures) > 0 {
		for name, failure := range failures {
			t.Logf("%s FAILED: %s", name, failure)
		}
		t.Fatalf("%d out of %d tests failed", len(failures), len(suite.AllEthTests()))
	}
}

func runEth65(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	// wait for geth to start up
	time.Sleep(time.Second * 5)

	suite := newTestSuite(t, geth.Server().Self())
	failures := make(map[string]string)
	for _, test := range suite.EthTests() {
		failed, output := utesting.Run(test)
		if failed {
			failures[test.Name] = output
		}
	}
	if len(failures) > 0 {
		for name, failure := range failures {
			t.Logf("%s FAILED: %s", name, failure)
		}
		t.Fatalf("%d out of %d tests failed", len(failures), len(suite.EthTests()))
	}
}

func runEth66(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	// wait for geth to start up
	time.Sleep(time.Second * 5)

	suite := newTestSuite(t, geth.Server().Self())
	failures := make(map[string]string)
	for _, test := range suite.Eth66Tests() {
		failed, output := utesting.Run(test)
		if failed {
			failures[test.Name] = output
		}
	}
	if len(failures) > 0 {
		for name, failure := range failures {
			t.Logf("%s FAILED: %s", name, failure)
		}
		t.Fatalf("%d out of %d tests failed", len(failures), len(suite.Eth66Tests()))
	}
}


func runTest(t *testing.T, test string) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	// wait for geth to start up
	time.Sleep(time.Second * 5)

	suite := newTestSuite(t, geth.Server().Self())
	fn := testFn(test, suite)
	if fn == nil {
		t.Fatalf("could not find test function for %s", test)
	}
	failed, output := utesting.Run(utesting.Test{Name: test, Fn: fn})
	if failed {
		t.Fatalf("test failed: \n%s", output)
	}
}

func testFn(name string, suite *Suite) func(t *utesting.T) {
	switch name {
	case "TestStatus":
		return suite.TestStatus
	case "TestStatus_66":
		return suite.TestStatus_66
	case "TestGetBlockHeaders":
		return suite.TestGetBlockHeaders
	case "TestGetBlockHeaders_66":
		return suite.TestGetBlockHeaders_66
	case "TestSimultaneousRequests_66":
		return suite.TestSimultaneousRequests_66
	case "TestSameRequestID_66":
		return suite.TestSameRequestID_66
	case "TestZeroRequestID_66":
		return suite.TestZeroRequestID_66
	case "TestGetBlockBodies":
		return suite.TestGetBlockBodies
	case "TestGetBlockBodies_66":
		return suite.TestGetBlockBodies_66
	case "TestBroadcast":
		return suite.TestBroadcast
	case "TestBroadcast_66":
		return suite.TestBroadcast_66
	case "TestLargeAnnounce":
		return suite.TestLargeAnnounce
	case "TestLargeAnnounce_66":
		return suite.TestLargeAnnounce_66
	case "TestOldAnnounce":
		return suite.TestOldAnnounce
	case "TestOldAnnounce_66":
		return suite.TestOldAnnounce_66
	case "TestMaliciousHandshake":
		return suite.TestMaliciousHandshake
	case "TestMaliciousStatus":
		return suite.TestMaliciousStatus
	case "TestMaliciousStatus_66":
		return suite.TestMaliciousStatus_66
	case "TestMaliciousHandshake_66":
		return suite.TestMaliciousHandshake_66
	case "TestTransaction":
		return suite.TestTransaction
	case "TestTransaction_66":
		return suite.TestTransaction_66
	case "TestMaliciousTx":
		return suite.TestMaliciousTx
	case "TestMaliciousTx_66":
		return suite.TestMaliciousTx_66
	}
	return nil
}

// runGeth creates and starts a geth node
func runGeth() (*node.Node, error) {
	geth, err := setupGeth()
	if err != nil {
		return nil, err
	}
	if err := geth.Start(); err != nil {
		return nil, err
	}
	return geth, nil
}

func newTestSuite(t *testing.T, enodeID *enode.Node) *Suite {
	suite, err := NewSuite(enodeID, fullchain, genesis)
	if err != nil {
		t.Fatalf("could not create test suite: %v", err)
	}
	return suite
}

func filepaths() error {
	var err error

	genesis, err = filepath.Abs("./testdata/genesis.json")
	if err != nil {
		return err
	}

	halfchain, err = filepath.Abs("./testdata/halfchain.rlp")
	if err != nil {
		return err
	}

	fullchain, err = filepath.Abs("./testdata/chain.rlp")
	if err != nil {
		return err
	}

	nodekey, err = filepath.Abs("./testdata/nodekey")
	return err
}

func setupGeth() (*node.Node, error) {
	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:30303",
			NoDiscovery: true,
			MaxPeers:    20, // TODO arbitrary
			NoDial:      true,
		},
	})
	if err != nil {
		return nil, err
	}
	// get genesis
	gen, err := gen()
	if err != nil {
		return nil, err
	}
	genBlock := gen.ToBlock(nil)

	backend, err := eth.New(stack, &ethconfig.Config{
		Genesis:   gen,
		NetworkId: gen.Config.ChainID.Uint64(), // 19763
	})
	if err != nil {
		return nil, err
	}

	blocks, err := blocksFromFile(halfchain, genBlock)
	if err != nil {
		return nil, err
	}

	_, err = backend.BlockChain().InsertChain(blocks[1:])
	if err != nil {
		return nil, err
	}

	return stack, nil
}

func gen() (*core.Genesis, error) {
	chainConfig, err := ioutil.ReadFile(genesis)
	if err != nil {
		return nil, err
	}
	var gen core.Genesis
	if err := json.Unmarshal(chainConfig, &gen); err != nil {
		return nil, err
	}
	return &gen, nil
}
