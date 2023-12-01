// Copyright 2015 The go-ethereum Authors
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

package backends

import (
	"context"
	"math"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var _ bind.ContractBackend = (*SimulatedBackend)(nil)

type SimulatedBackend struct {
	eth *eth.Ethereum
	*catalyst.SimulatedBeacon
	*ethclient.Client
}

// NewSimulatedBackend creates a new binding backend using a simulated blockchain
// for testing purposes.
// A simulated backend always uses chainID 1337.
func NewSimulatedBackend(alloc core.GenesisAlloc, gasLimit uint64) *SimulatedBackend {
	// Setup the node object
	nodeConf := node.DefaultConfig
	nodeConf.DataDir = ""
	nodeConf.P2P = p2p.Config{DiscAddr: "", ListenAddr: ""}
	stack, err := node.New(&nodeConf)
	if err != nil {
		// This should never happen, if it does, please open an issue
		panic(err)
	}

	// Setup ethereum
	genesis := core.Genesis{
		Config:   params.AllDevChainProtocolChanges,
		GasLimit: gasLimit,
		Alloc:    alloc,
	}
	conf := ethconfig.Defaults
	conf.Genesis = &genesis
	conf.SyncMode = downloader.FullSync
	sim, err := NewSimWithNode(stack, &conf, math.MaxUint64)
	if err != nil {
		// This should never happen, if it does, please open an issue
		panic(err)
	}
	return sim
}

// NewSimWithNode sets up a simulated backend on an existing node
// this allows users to do persistent simulations.
// The provided node must not be started and will be started by NewSimWithNode
func NewSimWithNode(stack *node.Node, conf *eth.Config, blockPeriod uint64) (*SimulatedBackend, error) {
	backend, err := eth.New(stack, conf)
	if err != nil {
		return nil, err
	}

	// Register the filter system
	filterSystem := filters.NewFilterSystem(backend.APIBackend, filters.Config{})
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filterSystem, false),
	}})

	// Start the node
	if err := stack.Start(); err != nil {
		return nil, err
	}

	// Set up the simulated beacon
	beacon, err := catalyst.NewSimulatedBeacon(blockPeriod, backend)
	if err != nil {
		return nil, err
	}

	// Reorg our chain back to genesis
	if err := beacon.Fork(context.Background(), backend.BlockChain().GetCanonicalHash(0)); err != nil {
		return nil, err
	}

	return &SimulatedBackend{
		eth:             backend,
		SimulatedBeacon: beacon,
		Client:          ethclient.NewClient(stack.Attach()),
	}, nil
}

func (n *SimulatedBackend) Close() error {
	if n.Client != nil {
		n.Client.Close()
		n.Client = nil
	}
	if n.SimulatedBeacon != nil {
		err := n.SimulatedBeacon.Stop()
		n.SimulatedBeacon = nil
		return err
	}
	return nil
}
