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
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	genesisFile   = "./testdata/genesis.json"
	halfchainFile = "./testdata/halfchain.rlp"
	fullchainFile = "./testdata/chain.rlp"
)

func TestEthSuite(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	defer geth.Close()

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
	suite, err := NewSuite(enodeID, fullchainFile, genesisFile)
	if err != nil {
		t.Fatalf("could not create test suite: %v", err)
	}
	return suite
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
	chain, err := loadChain(halfchainFile, genesisFile)
	if err != nil {
		return nil, err
	}

	backend, err := eth.New(stack, &ethconfig.Config{
		Genesis:   &chain.genesis,
		NetworkId: chain.genesis.Config.ChainID.Uint64(), // 19763
	})
	if err != nil {
		return nil, err
	}

	_, err = backend.BlockChain().InsertChain(chain.blocks[1:])
	if err != nil {
		return nil, err
	}

	return stack, nil
}
