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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

type mockBackend struct {
	bc     *core.BlockChain
	txPool *txpool.TxPool
}

func NewMockBackend(bc *core.BlockChain, txPool *txpool.TxPool) *mockBackend {
	return &mockBackend{
		bc:     bc,
		txPool: txPool,
	}
}

func (m *mockBackend) BlockChain() *core.BlockChain {
	return m.bc
}

func (m *mockBackend) TxPool() *txpool.TxPool {
	return m.txPool
}

type testBlockChain struct {
	root          common.Hash
	config        *params.ChainConfig
	statedb       *state.StateDB
	gasLimit      uint64
	chainHeadFeed *event.Feed
}

func (bc *testBlockChain) Config() *params.ChainConfig {
	return bc.config
}

func (bc *testBlockChain) CurrentBlock() *types.Header {
	return &types.Header{
		Number:   new(big.Int),
		GasLimit: bc.gasLimit,
	}
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return types.NewBlock(bc.CurrentBlock(), nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) StateAt(root common.Hash, number *big.Int, time uint64) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) Genesis() *types.Block {
	return types.NewBlock(bc.CurrentBlock(), nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) HasState(root common.Hash) bool {
	return bc.root == root
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return bc.chainHeadFeed.Subscribe(ch)
}

func TestBuildPendingBlocks(t *testing.T) {
	miner := createMiner(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		block, _, _ := miner.Pending()
		if block == nil {
			t.Error("Pending failed")
		}
	}()
	wg.Wait()
}

// TestBuildPendingBlockAheadOfClock checks the pending block when the chain
// head is ahead of the local clock. The block is then built at parent.Time+1,
// so its fork-specific fields must follow that timestamp, not the wall clock.
func TestBuildPendingBlockAheadOfClock(t *testing.T) {
	forkTime := uint64(time.Now().Add(time.Hour).Unix())

	tests := []struct {
		name       string
		parentTime uint64
		amsterdam  bool
		slot       uint64
	}{
		// The chain head is already past the fork, the local clock is not.
		// The Amsterdam genesis has slot 0, so the pending block has slot 1.
		{"head-past-fork", forkTime + 12, true, 1},
		// The pending block is the first block of the fork. The parent has
		// no slot, so the pending block starts at slot 0.
		{"head-at-fork-boundary", forkTime - 1, true, 0},
		// The chain head and the pending block are both before the fork.
		{"head-before-fork", forkTime - 12, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := *params.MergedTestChainConfig
			config.AmsterdamTime = &forkTime

			alloc := core.SystemContractAllocs()
			alloc[testBankAddress] = types.Account{Balance: testBankFunds}
			gspec := &core.Genesis{
				Config:    &config,
				Timestamp: tt.parentTime,
				Alloc:     alloc,
			}
			engine := beacon.New(ethash.NewFaker())
			chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
			if err != nil {
				t.Fatalf("can't create new chain: %v", err)
			}
			defer chain.Stop()

			pool := legacypool.New(testTxPoolConfig, chain)
			txpool, _ := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{pool})
			defer txpool.Close()

			miner := New(&testWorkerBackend{chain: chain, txPool: txpool}, testConfig, engine)
			block, _, _ := miner.Pending()
			if block == nil {
				t.Fatal("pending block is not available")
			}
			if block.Time() != tt.parentTime+1 {
				t.Errorf("pending block time mismatch: have %d, want %d", block.Time(), tt.parentTime+1)
			}
			slot := block.Header().SlotNumber
			if !tt.amsterdam {
				if slot != nil {
					t.Errorf("unexpected slot number %d pre-amsterdam", *slot)
				}
				return
			}
			if slot == nil {
				t.Fatal("missing slot number post-amsterdam")
			}
			if *slot != tt.slot {
				t.Errorf("slot number mismatch: have %d, want %d", *slot, tt.slot)
			}
		})
	}
}

func minerTestGenesisBlock(period uint64, gasLimit uint64, faucet common.Address) *core.Genesis {
	config := *params.AllCliqueProtocolChanges
	config.Clique = &params.CliqueConfig{
		Period: period,
		Epoch:  config.Clique.Epoch,
	}

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &core.Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:   gasLimit,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]types.Account{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func createMiner(t *testing.T) *Miner {
	// Create Ethash config
	config := Config{
		PendingFeeRecipient: common.HexToAddress("123456789"),
	}
	// Create chainConfig
	chainDB := rawdb.NewMemoryDatabase()
	triedb := triedb.NewDatabase(chainDB, nil)
	genesis := minerTestGenesisBlock(15, 11_500_000, common.HexToAddress("12345"))
	chainConfig, _, _, err := core.SetupGenesisBlock(chainDB, triedb, genesis)
	if err != nil {
		t.Fatalf("can't create new chain config: %v", err)
	}
	// Create consensus engine
	engine := clique.New(chainConfig.Clique, chainDB)
	// Create Ethereum backend
	bc, err := core.NewBlockChain(chainDB, genesis, engine, nil)
	if err != nil {
		t.Fatalf("can't create new chain %v", err)
	}
	statedb, _ := state.New(bc.Genesis().Root(), state.NewDatabase(bc.TrieDB(), bc.CodeDB()))
	blockchain := &testBlockChain{bc.Genesis().Root(), chainConfig, statedb, 10000000, new(event.Feed)}

	pool := legacypool.New(testTxPoolConfig, blockchain)
	txpool, _ := txpool.New(testTxPoolConfig.PriceLimit, blockchain, []txpool.SubPool{pool})

	// Create Miner
	backend := NewMockBackend(bc, txpool)
	miner := New(backend, config, engine)
	return miner
}
