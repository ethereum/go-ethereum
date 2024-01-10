// Copyright 2023 The go-ethereum Authors
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

package simulated

import (
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

// Backend is a simulated blockchain. You can use it to test your contracts or
// other code that interacts with the Ethereum chain.
type Backend struct {
	eth    *eth.Ethereum
	beacon *catalyst.SimulatedBeacon
	client simClient
}

// simClient wraps ethclient. This exists to prevent extracting ethclient.Client
// from the Client interface returned by Backend.
type simClient struct {
	*ethclient.Client
}

// Client exposes the methods provided by the Ethereum RPC client.
type Client interface {
	ethereum.BlockNumberReader
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer
	ethereum.GasPricer1559
	ethereum.FeeHistoryReader
	ethereum.LogFilterer
	ethereum.PendingStateReader
	ethereum.PendingContractCaller
	ethereum.TransactionReader
	ethereum.TransactionSender
	ethereum.ChainIDReader
}

// New creates a new binding backend using a simulated blockchain
// for testing purposes.
// A simulated backend always uses chainID 1337.
func New(alloc core.GenesisAlloc, gasLimit uint64) *Backend {
	// Setup the node object
	nodeConf := node.DefaultConfig
	nodeConf.DataDir = ""
	nodeConf.P2P = p2p.Config{NoDiscovery: true}
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
	conf.TxPool.NoLocals = true
	sim, err := newWithNode(stack, &conf, 0)
	if err != nil {
		// This should never happen, if it does, please open an issue
		panic(err)
	}
	return sim
}

// newWithNode sets up a simulated backend on an existing node
// this allows users to do persistent simulations.
// The provided node must not be started and will be started by newWithNode
func newWithNode(stack *node.Node, conf *eth.Config, blockPeriod uint64) (*Backend, error) {
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
	if err := beacon.Fork(backend.BlockChain().GetCanonicalHash(0)); err != nil {
		return nil, err
	}

	return &Backend{
		eth:    backend,
		beacon: beacon,
		client: simClient{ethclient.NewClient(stack.Attach())},
	}, nil
}

// Close shuts down the simBackend.
// The simulated backend can't be used afterwards.
func (n *Backend) Close() error {
	if n.client.Client != nil {
		n.client.Close()
		n.client = simClient{}
	}
	if n.beacon != nil {
		err := n.beacon.Stop()
		n.beacon = nil
		return err
	}
	return nil
}

// Commit seals a block and moves the chain forward to a new empty block.
func (n *Backend) Commit() common.Hash {
	return n.beacon.Commit()
}

// Rollback removes all pending transactions, reverting to the last committed state.
func (n *Backend) Rollback() {
	n.beacon.Rollback()
}

// Fork creates a side-chain that can be used to simulate reorgs.
//
// This function should be called with the ancestor block where the new side
// chain should be started. Transactions (old and new) can then be applied on
// top and Commit-ed.
//
// Note, the side-chain will only become canonical (and trigger the events) when
// it becomes longer. Until then CallContract will still operate on the current
// canonical chain.
//
// There is a % chance that the side chain becomes canonical at the same length
// to simulate live network behavior.
func (n *Backend) Fork(parentHash common.Hash) error {
	return n.beacon.Fork(parentHash)
}

// AdjustTime changes the block timestamp and creates a new block.
// It can only be called on empty blocks.
func (n *Backend) AdjustTime(adjustment time.Duration) error {
	return n.beacon.AdjustTime(adjustment)
}

// Client returns a client that accesses the simulated chain.
func (n *Backend) Client() Client {
	return n.client
}
