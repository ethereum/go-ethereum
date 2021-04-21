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
	"os"
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

func TestEthSuite(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	// wait for geth to start up
	time.Sleep(time.Second * 5)

	suite := newTestSuite(t, geth.Server().Self())

	for _, test := range suite.AllEthTests() {
		t.Run(test.Name, func(t *testing.T) {
			result := utesting.RunTAP([]utesting.Test{{Name: test.Name, Fn: test.Fn}}, os.Stdout)
			if result[0].Failed {
				t.Fatal()
			}
		})
	}
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
			MaxPeers:    10, // TODO arbitrary
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
