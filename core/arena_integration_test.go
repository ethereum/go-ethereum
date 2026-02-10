// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/arena"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// TestProcessWithBumpAllocator generates a chain with several types of
// transactions, then processes each block with both HeapAllocator and
// BumpAllocator, verifying that the state roots, receipt roots, gas used,
// and logs match exactly.
func TestProcessWithBumpAllocator(t *testing.T) {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		config  = params.MergedTestChainConfig
		signer  = types.LatestSigner(config)
		gspec   = &Genesis{
			Config: config,
			Alloc: types.GenesisAlloc{
				addr1: {Balance: big.NewInt(1_000_000_000_000_000_000)},
				addr2: {Balance: big.NewInt(1_000_000_000_000_000_000)},
			},
		}
		engine = beacon.New(ethash.NewFaker())
	)

	// Generate a chain with diverse transactions: value transfers, contract
	// creates, and dynamic fee transactions.
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, 5, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			// Simple value transfer.
			tx, _ := types.SignTx(
				types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(1000), params.TxGas, gen.BaseFee(), nil),
				signer, key1,
			)
			gen.AddTx(tx)

		case 1:
			// Two value transfers in one block.
			tx1, _ := types.SignTx(
				types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(2000), params.TxGas, gen.BaseFee(), nil),
				signer, key1,
			)
			gen.AddTx(tx1)
			tx2, _ := types.SignTx(
				types.NewTransaction(gen.TxNonce(addr2), addr1, big.NewInt(500), params.TxGas, gen.BaseFee(), nil),
				signer, key2,
			)
			gen.AddTx(tx2)

		case 2:
			// Contract creation: deploy a minimal contract (PUSH1 0x42, PUSH1 0, MSTORE, PUSH1 1, PUSH1 31, RETURN).
			initCode := []byte{
				0x60, 0x42, // PUSH1 0x42
				0x60, 0x00, // PUSH1 0
				0x52,       // MSTORE
				0x60, 0x01, // PUSH1 1
				0x60, 0x1f, // PUSH1 31
				0xf3, // RETURN
			}
			tx, _ := types.SignTx(
				types.NewContractCreation(gen.TxNonce(addr1), big.NewInt(0), 100_000, gen.BaseFee(), initCode),
				signer, key1,
			)
			gen.AddTx(tx)

		case 3:
			// Dynamic fee transaction (EIP-1559).
			tx, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
				Nonce:     gen.TxNonce(addr1),
				GasTipCap: big.NewInt(1),
				GasFeeCap: new(big.Int).Add(gen.BaseFee(), big.NewInt(1)),
				Gas:       params.TxGas,
				To:        &addr2,
				Value:     big.NewInt(3000),
			}), signer, key1)
			gen.AddTx(tx)

		case 4:
			// Multiple dynamic fee txs from different senders.
			tx1, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
				Nonce:     gen.TxNonce(addr1),
				GasTipCap: big.NewInt(2),
				GasFeeCap: new(big.Int).Add(gen.BaseFee(), big.NewInt(2)),
				Gas:       params.TxGas,
				To:        &addr2,
				Value:     big.NewInt(100),
			}), signer, key1)
			gen.AddTx(tx1)
			tx2, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
				Nonce:     gen.TxNonce(addr2),
				GasTipCap: big.NewInt(1),
				GasFeeCap: new(big.Int).Add(gen.BaseFee(), big.NewInt(1)),
				Gas:       params.TxGas,
				To:        &addr1,
				Value:     big.NewInt(200),
			}), signer, key2)
			gen.AddTx(tx2)
		}
	})

	// Sanity-check that we actually generated transactions.
	totalTxs := 0
	for _, r := range receipts {
		totalTxs += len(r)
	}
	if totalTxs == 0 {
		t.Fatal("no transactions generated")
	}
	t.Logf("generated %d blocks with %d total transactions", len(blocks), totalTxs)

	// processBlocks runs the state processor on all blocks with the given
	// vm.Config and returns per-block state roots, receipt roots, gas used.
	type blockResult struct {
		stateRoot   common.Hash
		receiptRoot common.Hash
		gasUsed     uint64
		logCount    int
	}

	processBlocks := func(cfg vm.Config) ([]blockResult, error) {
		// Build a fresh blockchain from genesis.
		db := rawdb.NewMemoryDatabase()
		triedb := state.NewDatabaseForTesting()
		gspec.MustCommit(db, triedb.TrieDB())
		chain := &HeaderChain{
			config:  gspec.Config,
			chainDb: db,
			engine:  engine,
		}
		processor := NewStateProcessor(chain)

		results := make([]blockResult, len(blocks))
		parentRoot := gspec.ToBlock().Root()

		for i, block := range blocks {
			statedb, err := state.New(parentRoot, triedb)
			if err != nil {
				return nil, err
			}
			res, err := processor.Process(context.Background(), block, statedb, cfg)
			if err != nil {
				return nil, err
			}
			root := statedb.IntermediateRoot(gspec.Config.IsEIP158(block.Number()))
			receiptRoot := types.DeriveSha(res.Receipts, trie.NewStackTrie(nil))

			results[i] = blockResult{
				stateRoot:   root,
				receiptRoot: receiptRoot,
				gasUsed:     res.GasUsed,
				logCount:    len(res.Logs),
			}
			// Commit so the next block can read from it.
			statedb.Commit(block.NumberU64(), gspec.Config.IsEIP158(block.Number()), false)
			parentRoot = root
		}
		return results, nil
	}

	// Process with HeapAllocator (default).
	heapResults, err := processBlocks(vm.Config{})
	if err != nil {
		t.Fatalf("heap processing failed: %v", err)
	}

	// Process with BumpAllocator.
	slab := make([]byte, 32<<20) // 32 MiB
	bumpResults, err := processBlocks(vm.Config{
		Allocator: arena.NewBumpAllocator(slab),
	})
	if err != nil {
		t.Fatalf("bump processing failed: %v", err)
	}

	// Compare results block by block.
	for i := range heapResults {
		h, b := heapResults[i], bumpResults[i]
		if h.stateRoot != b.stateRoot {
			t.Errorf("block %d: state root mismatch: heap=%x bump=%x", i, h.stateRoot, b.stateRoot)
		}
		if h.receiptRoot != b.receiptRoot {
			t.Errorf("block %d: receipt root mismatch: heap=%x bump=%x", i, h.receiptRoot, b.receiptRoot)
		}
		if h.gasUsed != b.gasUsed {
			t.Errorf("block %d: gas used mismatch: heap=%d bump=%d", i, h.gasUsed, b.gasUsed)
		}
		if h.logCount != b.logCount {
			t.Errorf("block %d: log count mismatch: heap=%d bump=%d", i, h.logCount, b.logCount)
		}
	}
}

// TestProcessWithBumpAllocatorResetBetweenBlocks verifies that the arena is
// properly reset between blocks — i.e., processing block N+1 doesn't corrupt
// because block N's arena memory was reclaimed.
func TestProcessWithBumpAllocatorResetBetweenBlocks(t *testing.T) {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		config = params.MergedTestChainConfig
		signer = types.LatestSigner(config)
		engine = beacon.New(ethash.NewFaker())
		gspec  = &Genesis{
			Config: config,
			Alloc: types.GenesisAlloc{
				addr: {Balance: big.NewInt(1_000_000_000_000_000_000)},
			},
		}
	)

	// Generate 10 blocks, each with a transfer, to exercise repeated
	// alloc/reset cycles on the same slab.
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 10, func(i int, gen *BlockGen) {
		tx, _ := types.SignTx(
			types.NewTransaction(gen.TxNonce(addr), common.HexToAddress("0xdead"), big.NewInt(1), params.TxGas, gen.BaseFee(), nil),
			signer, key,
		)
		gen.AddTx(tx)
	})

	// Use a deliberately small slab (512 KiB) so that without proper Reset,
	// the allocator would run out of memory across 10 blocks.
	slab := make([]byte, 512<<10)
	bumpAlloc := arena.NewBumpAllocator(slab)

	db := rawdb.NewMemoryDatabase()
	triedb := state.NewDatabaseForTesting()
	gspec.MustCommit(db, triedb.TrieDB())

	chain := &HeaderChain{
		config:  gspec.Config,
		chainDb: db,
		engine:  engine,
	}
	processor := NewStateProcessor(chain)
	parentRoot := gspec.ToBlock().Root()

	for i, block := range blocks {
		statedb, err := state.New(parentRoot, triedb)
		if err != nil {
			t.Fatalf("block %d: state.New failed: %v", i, err)
		}
		cfg := vm.Config{Allocator: bumpAlloc}
		res, err := processor.Process(context.Background(), block, statedb, cfg)
		if err != nil {
			t.Fatalf("block %d: Process failed: %v", i, err)
		}
		if res.GasUsed == 0 {
			t.Fatalf("block %d: no gas used", i)
		}
		parentRoot = statedb.IntermediateRoot(gspec.Config.IsEIP158(block.Number()))
		statedb.Commit(block.NumberU64(), gspec.Config.IsEIP158(block.Number()), false)
	}
}
