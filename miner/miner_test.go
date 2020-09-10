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

// Package miner implements Ethereum block creation and mining.
package miner

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type mockBackend struct {
	bc     *core.BlockChain
	txPool *core.TxPool
}

func NewMockBackend(bc *core.BlockChain, txPool *core.TxPool) *mockBackend {
	return &mockBackend{
		bc:     bc,
		txPool: txPool,
	}
}

func (m *mockBackend) BlockChain() *core.BlockChain {
	return m.bc
}

func (m *mockBackend) TxPool() *core.TxPool {
	return m.txPool
}

type testBlockChain struct {
	statedb       *state.StateDB
	gasLimit      uint64
	chainHeadFeed *event.Feed
}

func (bc *testBlockChain) CurrentBlock() *types.Block {
	return types.NewBlock(&types.Header{
		GasLimit: bc.gasLimit,
	}, nil, nil, nil, new(trie.Trie))
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return bc.CurrentBlock()
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return bc.chainHeadFeed.Subscribe(ch)
}

func TestMiner(t *testing.T) {
	miner, mux := createMiner(t)
	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)
	// Start the downloader
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)
	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.DoneEvent{})
	waitForMiningState(t, miner, true)
	// Start the downloader and wait for the update loop to run
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)
	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.FailedEvent{})
	waitForMiningState(t, miner, true)
}

func TestStartStopMiner(t *testing.T) {
	miner, _ := createMiner(t)
	waitForMiningState(t, miner, false)
	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)
	miner.Stop()
	waitForMiningState(t, miner, false)
}

func TestCloseMiner(t *testing.T) {
	miner, _ := createMiner(t)
	waitForMiningState(t, miner, false)
	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)
	// Terminate the miner and wait for the update loop to run
	miner.Close()
	waitForMiningState(t, miner, false)
}

// waitForMiningState waits until either
// * the desired mining state was reached
// * a timeout was reached which fails the test
func waitForMiningState(t *testing.T, m *Miner, mining bool) {
	t.Helper()

	var state bool
	for i := 0; i < 100; i++ {
		if state = m.Mining(); state == mining {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Mining() == %t, want %t", state, mining)
}

func createMiner(t *testing.T) (*Miner, *event.TypeMux) {
	// Create Ethash config
	config := Config{
		Etherbase: common.HexToAddress("123456789"),
	}
	// Create chainConfig
	memdb := memorydb.New()
	chainDB := rawdb.NewDatabase(memdb)
	genesis := core.DeveloperGenesisBlock(15, common.HexToAddress("12345"))
	chainConfig, _, err := core.SetupGenesisBlock(chainDB, genesis)
	if err != nil {
		t.Fatalf("can't create new chain config: %v", err)
	}
	// Create event Mux
	mux := new(event.TypeMux)
	// Create consensus engine
	engine := ethash.New(ethash.Config{}, []string{}, false)
	engine.SetThreads(-1)
	// Create isLocalBlock
	isLocalBlock := func(block *types.Block) bool {
		return true
	}
	// Create Ethereum backend
	limit := uint64(1000)
	bc, err := core.NewBlockChain(chainDB, new(core.CacheConfig), chainConfig, engine, vm.Config{}, isLocalBlock, &limit)
	if err != nil {
		t.Fatalf("can't create new chain %v", err)
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 10000000, new(event.Feed)}

	pool := core.NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	backend := NewMockBackend(bc, pool)
	// Create Miner
	return New(backend, &config, chainConfig, mux, engine, isLocalBlock), mux
}
