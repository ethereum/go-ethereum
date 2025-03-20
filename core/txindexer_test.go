// Copyright 2024 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

func verifyIndexes(t *testing.T, db ethdb.Database, block *types.Block, exist bool) {
	for _, tx := range block.Transactions() {
		lookup := rawdb.ReadTxLookupEntry(db, tx.Hash())
		if exist && lookup == nil {
			t.Fatalf("missing %d %x", block.NumberU64(), tx.Hash().Hex())
		}
		if !exist && lookup != nil {
			t.Fatalf("unexpected %d %x", block.NumberU64(), tx.Hash().Hex())
		}
	}
}

func verify(t *testing.T, db ethdb.Database, blocks []*types.Block, expTail uint64) {
	tail := rawdb.ReadTxIndexTail(db)
	if tail == nil {
		t.Fatal("Failed to write tx index tail")
		return
	}
	if *tail != expTail {
		t.Fatalf("Unexpected tx index tail, want %v, got %d", expTail, *tail)
	}
	for _, b := range blocks {
		if b.Number().Uint64() < *tail {
			verifyIndexes(t, db, b, false)
		} else {
			verifyIndexes(t, db, b, true)
		}
	}
}

func verifyNoIndex(t *testing.T, db ethdb.Database, blocks []*types.Block) {
	tail := rawdb.ReadTxIndexTail(db)
	if tail != nil {
		t.Fatalf("Unexpected tx index tail %d", *tail)
	}
	for _, b := range blocks {
		verifyIndexes(t, db, b, false)
	}
}

// TestTxIndexer tests the functionalities for managing transaction indexes.
func TestTxIndexer(t *testing.T) {
	var (
		testBankKey, _  = crypto.GenerateKey()
		testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
		testBankFunds   = big.NewInt(1000000000000000000)

		gspec = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		engine    = ethash.NewFaker()
		nonce     = uint64(0)
		chainHead = uint64(128)
	)
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, int(chainHead), func(i int, gen *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(nonce, common.HexToAddress("0xdeadbeef"), big.NewInt(1000), params.TxGas, big.NewInt(10*params.InitialBaseFee), nil), types.HomesteadSigner{}, testBankKey)
		gen.AddTx(tx)
		nonce += 1
	})
	var cases = []struct {
		limits []uint64
		tails  []uint64
	}{
		{
			limits: []uint64{0, 1, 64, 129, 0},
			tails:  []uint64{0, 128, 65, 0, 0},
		},
		{
			limits: []uint64{64, 1, 64, 0},
			tails:  []uint64{65, 128, 65, 0},
		},
		{
			limits: []uint64{127, 1, 64, 0},
			tails:  []uint64{2, 128, 65, 0},
		},
		{
			limits: []uint64{128, 1, 64, 0},
			tails:  []uint64{1, 128, 65, 0},
		},
		{
			limits: []uint64{129, 1, 64, 0},
			tails:  []uint64{0, 128, 65, 0},
		},
	}
	for _, c := range cases {
		db, _ := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), "", "", false)
		rawdb.WriteAncientBlocks(db, append([]*types.Block{gspec.ToBlock()}, blocks...), append([]types.Receipts{{}}, receipts...))

		// Index the initial blocks from ancient store
		indexer := &txIndexer{
			limit:    0,
			db:       db,
			progress: make(chan chan TxIndexProgress),
		}
		for i, limit := range c.limits {
			indexer.limit = limit
			indexer.run(chainHead, make(chan struct{}), make(chan struct{}))
			verify(t, db, blocks, c.tails[i])
		}
		db.Close()
	}
}

func TestTxIndexerRepair(t *testing.T) {
	var (
		testBankKey, _  = crypto.GenerateKey()
		testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
		testBankFunds   = big.NewInt(1000000000000000000)

		gspec = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		engine    = ethash.NewFaker()
		nonce     = uint64(0)
		chainHead = uint64(128)
	)
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, int(chainHead), func(i int, gen *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(nonce, common.HexToAddress("0xdeadbeef"), big.NewInt(1000), params.TxGas, big.NewInt(10*params.InitialBaseFee), nil), types.HomesteadSigner{}, testBankKey)
		gen.AddTx(tx)
		nonce += 1
	})
	tailPointer := func(n uint64) *uint64 {
		return &n
	}
	var cases = []struct {
		limit   uint64
		head    uint64
		cutoff  uint64
		expTail *uint64
	}{
		// if *tail > head => purge indexes
		{
			limit:   0,
			head:    chainHead / 2,
			cutoff:  0,
			expTail: tailPointer(0),
		},
		{
			limit:   1,             // tail = 128
			head:    chainHead / 2, // newhead = 64
			cutoff:  0,
			expTail: nil,
		},
		{
			limit:   64,            // tail = 65
			head:    chainHead / 2, // newhead = 64
			cutoff:  0,
			expTail: nil,
		},
		{
			limit:   65,            // tail = 64
			head:    chainHead / 2, // newhead = 64
			cutoff:  0,
			expTail: tailPointer(64),
		},
		{
			limit:   66,            // tail = 63
			head:    chainHead / 2, // newhead = 64
			cutoff:  0,
			expTail: tailPointer(63),
		},

		// if tail < cutoff => remove indexes below cutoff
		{
			limit:   0,         // tail = 0
			head:    chainHead, // head = 128
			cutoff:  chainHead, // cutoff = 128
			expTail: tailPointer(chainHead),
		},
		{
			limit:   1,         // tail = 128
			head:    chainHead, // head = 128
			cutoff:  chainHead, // cutoff = 128
			expTail: tailPointer(128),
		},
		{
			limit:   2,         // tail = 127
			head:    chainHead, // head = 128
			cutoff:  chainHead, // cutoff = 128
			expTail: tailPointer(chainHead),
		},
		{
			limit:   2,             // tail = 127
			head:    chainHead,     // head = 128
			cutoff:  chainHead / 2, // cutoff = 64
			expTail: tailPointer(127),
		},

		// if head < cutoff => purge indexes
		{
			limit:   0,             // tail = 0
			head:    chainHead,     // head = 128
			cutoff:  2 * chainHead, // cutoff = 256
			expTail: nil,
		},
		{
			limit:   64,            // tail = 65
			head:    chainHead,     // head = 128
			cutoff:  chainHead / 2, // cutoff = 64
			expTail: tailPointer(65),
		},
	}
	for _, c := range cases {
		db, _ := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), "", "", false)
		rawdb.WriteAncientBlocks(db, append([]*types.Block{gspec.ToBlock()}, blocks...), append([]types.Receipts{{}}, receipts...))

		// Index the initial blocks from ancient store
		indexer := &txIndexer{
			limit:    c.limit,
			db:       db,
			progress: make(chan chan TxIndexProgress),
		}
		indexer.run(chainHead, make(chan struct{}), make(chan struct{}))

		indexer.cutoff = c.cutoff
		indexer.repair(c.head)

		if c.expTail == nil {
			verifyNoIndex(t, db, blocks)
		} else {
			verify(t, db, blocks, *c.expTail)
		}
		db.Close()
	}
}

func TestTxIndexerReport(t *testing.T) {
	var (
		testBankKey, _  = crypto.GenerateKey()
		testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
		testBankFunds   = big.NewInt(1000000000000000000)

		gspec = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		engine    = ethash.NewFaker()
		nonce     = uint64(0)
		chainHead = uint64(128)
	)
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, int(chainHead), func(i int, gen *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(nonce, common.HexToAddress("0xdeadbeef"), big.NewInt(1000), params.TxGas, big.NewInt(10*params.InitialBaseFee), nil), types.HomesteadSigner{}, testBankKey)
		gen.AddTx(tx)
		nonce += 1
	})
	tailPointer := func(n uint64) *uint64 {
		return &n
	}
	var cases = []struct {
		head         uint64
		limit        uint64
		cutoff       uint64
		tail         *uint64
		expIndexed   uint64
		expRemaining uint64
	}{
		// The entire chain is supposed to be indexed
		{
			// head = 128, limit = 0, cutoff = 0 => all: 129
			head:   chainHead,
			limit:  0,
			cutoff: 0,

			// tail = 0
			tail:         tailPointer(0),
			expIndexed:   129,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 0, cutoff = 0 => all: 129
			head:   chainHead,
			limit:  0,
			cutoff: 0,

			// tail = 1
			tail:         tailPointer(1),
			expIndexed:   128,
			expRemaining: 1,
		},
		{
			// head = 128, limit = 0, cutoff = 0 => all: 129
			head:   chainHead,
			limit:  0,
			cutoff: 0,

			// tail = 128
			tail:         tailPointer(chainHead),
			expIndexed:   1,
			expRemaining: 128,
		},
		{
			// head = 128, limit = 256, cutoff = 0 => all: 129
			head:   chainHead,
			limit:  256,
			cutoff: 0,

			// tail = 0
			tail:         tailPointer(0),
			expIndexed:   129,
			expRemaining: 0,
		},

		// The chain with specific range is supposed to be indexed
		{
			// head = 128, limit = 64, cutoff = 0 => index: [65, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 0,

			// tail = 0, part of them need to be unindexed
			tail:         tailPointer(0),
			expIndexed:   129,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 64, cutoff = 0 => index: [65, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 0,

			// tail = 64, one of them needs to be unindexed
			tail:         tailPointer(64),
			expIndexed:   65,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 64, cutoff = 0 => index: [65, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 0,

			// tail = 65, all of them have been indexed
			tail:         tailPointer(65),
			expIndexed:   64,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 64, cutoff = 0 => index: [65, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 0,

			// tail = 66, one of them has to be indexed
			tail:         tailPointer(66),
			expIndexed:   63,
			expRemaining: 1,
		},

		// The chain with configured cutoff, the chain range could be capped
		{
			// head = 128, limit = 64, cutoff = 66 => index: [66, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 66,

			// tail = 0, part of them need to be unindexed
			tail:         tailPointer(0),
			expIndexed:   129,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 64, cutoff = 66 => index: [66, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 66,

			// tail = 66, all of them have been indexed
			tail:         tailPointer(66),
			expIndexed:   63,
			expRemaining: 0,
		},
		{
			// head = 128, limit = 64, cutoff = 66 => index: [66, 128]
			head:   chainHead,
			limit:  64,
			cutoff: 66,

			// tail = 67, one of them has to be indexed
			tail:         tailPointer(67),
			expIndexed:   62,
			expRemaining: 1,
		},
		{
			// head = 128, limit = 64, cutoff = 256 => index: [66, 128]
			head:         chainHead,
			limit:        0,
			cutoff:       256,
			tail:         nil,
			expIndexed:   0,
			expRemaining: 0,
		},
	}
	for _, c := range cases {
		db, _ := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), "", "", false)
		rawdb.WriteAncientBlocks(db, append([]*types.Block{gspec.ToBlock()}, blocks...), append([]types.Receipts{{}}, receipts...))

		// Index the initial blocks from ancient store
		indexer := &txIndexer{
			limit:    c.limit,
			cutoff:   c.cutoff,
			db:       db,
			progress: make(chan chan TxIndexProgress),
		}
		if c.tail != nil {
			rawdb.WriteTxIndexTail(db, *c.tail)
		}
		p := indexer.report(c.head)
		if p.Indexed != c.expIndexed {
			t.Fatalf("Unexpected indexed: %d, expected: %d", p.Indexed, c.expIndexed)
		}
		if p.Remaining != c.expRemaining {
			t.Fatalf("Unexpected remaining: %d, expected: %d", p.Remaining, c.expRemaining)
		}
		db.Close()
	}
}
