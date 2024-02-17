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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package core

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

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

	// verifyIndexes checks if the transaction indexes are present or not
	// of the specified block.
	verifyIndexes := func(db ethdb.Database, number uint64, exist bool) {
		if number == 0 {
			return
		}
		block := blocks[number-1]
		for _, tx := range block.Transactions() {
			lookup := rawdb.ReadTxLookupEntry(db, tx.Hash())
			if exist && lookup == nil {
				t.Fatalf("missing %d %x", number, tx.Hash().Hex())
			}
			if !exist && lookup != nil {
				t.Fatalf("unexpected %d %x", number, tx.Hash().Hex())
			}
		}
	}
	verify := func(db ethdb.Database, expTail uint64, indexer *txIndexer) {
		tail := rawdb.ReadTxIndexTail(db)
		if tail == nil {
			t.Fatal("Failed to write tx index tail")
		}
		if *tail != expTail {
			t.Fatalf("Unexpected tx index tail, want %v, got %d", expTail, *tail)
		}
		if *tail != 0 {
			for number := uint64(0); number < *tail; number += 1 {
				verifyIndexes(db, number, false)
			}
		}
		for number := *tail; number <= chainHead; number += 1 {
			verifyIndexes(db, number, true)
		}
		progress := indexer.report(chainHead, tail)
		if !progress.Done() {
			t.Fatalf("Expect fully indexed")
		}
	}

	var cases = []struct {
		limitA uint64
		tailA  uint64
		limitB uint64
		tailB  uint64
		limitC uint64
		tailC  uint64
	}{
		{
			// LimitA: 0
			// TailA:  0
			//
			// all blocks are indexed
			limitA: 0,
			tailA:  0,

			// LimitB: 1
			// TailB:  128
			//
			// block-128 is indexed
			limitB: 1,
			tailB:  128,

			// LimitB: 64
			// TailB:  65
			//
			// block [65, 128] are indexed
			limitC: 64,
			tailC:  65,
		},
		{
			// LimitA: 64
			// TailA:  65
			//
			// block [65, 128] are indexed
			limitA: 64,
			tailA:  65,

			// LimitB: 1
			// TailB:  128
			//
			// block-128 is indexed
			limitB: 1,
			tailB:  128,

			// LimitB: 64
			// TailB:  65
			//
			// block [65, 128] are indexed
			limitC: 64,
			tailC:  65,
		},
		{
			// LimitA: 127
			// TailA:  2
			//
			// block [2, 128] are indexed
			limitA: 127,
			tailA:  2,

			// LimitB: 1
			// TailB:  128
			//
			// block-128 is indexed
			limitB: 1,
			tailB:  128,

			// LimitB: 64
			// TailB:  65
			//
			// block [65, 128] are indexed
			limitC: 64,
			tailC:  65,
		},
		{
			// LimitA: 128
			// TailA:  1
			//
			// block [2, 128] are indexed
			limitA: 128,
			tailA:  1,

			// LimitB: 1
			// TailB:  128
			//
			// block-128 is indexed
			limitB: 1,
			tailB:  128,

			// LimitB: 64
			// TailB:  65
			//
			// block [65, 128] are indexed
			limitC: 64,
			tailC:  65,
		},
		{
			// LimitA: 129
			// TailA:  0
			//
			// block [0, 128] are indexed
			limitA: 129,
			tailA:  0,

			// LimitB: 1
			// TailB:  128
			//
			// block-128 is indexed
			limitB: 1,
			tailB:  128,

			// LimitB: 64
			// TailB:  65
			//
			// block [65, 128] are indexed
			limitC: 64,
			tailC:  65,
		},
	}
	for _, c := range cases {
		frdir := t.TempDir()
		db, _ := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), frdir, "", false)
		rawdb.WriteAncientBlocks(db, append([]*types.Block{gspec.ToBlock()}, blocks...), append([]types.Receipts{{}}, receipts...), big.NewInt(0))

		// Index the initial blocks from ancient store
		indexer := &txIndexer{
			limit:    c.limitA,
			db:       db,
			progress: make(chan chan TxIndexProgress),
		}
		indexer.run(nil, 128, make(chan struct{}), make(chan struct{}))
		verify(db, c.tailA, indexer)

		indexer.limit = c.limitB
		indexer.run(rawdb.ReadTxIndexTail(db), 128, make(chan struct{}), make(chan struct{}))
		verify(db, c.tailB, indexer)

		indexer.limit = c.limitC
		indexer.run(rawdb.ReadTxIndexTail(db), 128, make(chan struct{}), make(chan struct{}))
		verify(db, c.tailC, indexer)

		// Recover all indexes
		indexer.limit = 0
		indexer.run(rawdb.ReadTxIndexTail(db), 128, make(chan struct{}), make(chan struct{}))
		verify(db, 0, indexer)

		db.Close()
		os.RemoveAll(frdir)
	}
}
