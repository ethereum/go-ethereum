// Copyright 2025 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"math"
	"math/rand"
	"slices"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestIndexReaderBasic(t *testing.T) {
	elements := []uint64{
		1, 5, 10, 11, 20,
	}
	db := rawdb.NewMemoryDatabase()
	bw, _ := newIndexWriter(db, newAccountIdent(common.Hash{0xa}))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}
	batch := db.NewBatch()
	bw.finish(batch)
	batch.Write()

	br, err := newIndexReader(db, newAccountIdent(common.Hash{0xa}))
	if err != nil {
		t.Fatalf("Failed to construct the index reader, %v", err)
	}
	cases := []struct {
		value  uint64
		result uint64
	}{
		{0, 1},
		{1, 5},
		{10, 11},
		{19, 20},
		{20, math.MaxUint64},
		{21, math.MaxUint64},
	}
	for _, c := range cases {
		got, err := br.readGreaterThan(c.value)
		if err != nil {
			t.Fatalf("Unexpected error, got %v", err)
		}
		if got != c.result {
			t.Fatalf("Unexpected result, got %v, wanted %v", got, c.result)
		}
	}
}

func TestIndexReaderLarge(t *testing.T) {
	var elements []uint64
	for i := 0; i < 10*indexBlockEntriesCap; i++ {
		elements = append(elements, rand.Uint64())
	}
	slices.Sort(elements)

	db := rawdb.NewMemoryDatabase()
	bw, _ := newIndexWriter(db, newAccountIdent(common.Hash{0xa}))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}
	batch := db.NewBatch()
	bw.finish(batch)
	batch.Write()

	br, err := newIndexReader(db, newAccountIdent(common.Hash{0xa}))
	if err != nil {
		t.Fatalf("Failed to construct the index reader, %v", err)
	}
	for i := 0; i < 100; i++ {
		value := rand.Uint64()
		pos := sort.Search(len(elements), func(i int) bool {
			return elements[i] > value
		})
		got, err := br.readGreaterThan(value)
		if err != nil {
			t.Fatalf("Unexpected error, got %v", err)
		}
		if pos == len(elements) {
			if got != math.MaxUint64 {
				t.Fatalf("Unexpected result, got %d, wanted math.MaxUint64", got)
			}
		} else if got != elements[pos] {
			t.Fatalf("Unexpected result, got %d, wanted %d", got, elements[pos])
		}
	}
}

func TestEmptyIndexReader(t *testing.T) {
	br, err := newIndexReader(rawdb.NewMemoryDatabase(), newAccountIdent(common.Hash{0xa}))
	if err != nil {
		t.Fatalf("Failed to construct the index reader, %v", err)
	}
	res, err := br.readGreaterThan(100)
	if err != nil {
		t.Fatalf("Failed to query, %v", err)
	}
	if res != math.MaxUint64 {
		t.Fatalf("Unexpected result, got %d, wanted math.MaxUint64", res)
	}
}

func TestIndexWriterBasic(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	iw, _ := newIndexWriter(db, newAccountIdent(common.Hash{0xa}))
	iw.append(2)
	if err := iw.append(1); err == nil {
		t.Fatal("out-of-order insertion is not expected")
	}
	for i := 0; i < 10; i++ {
		iw.append(uint64(i + 3))
	}
	batch := db.NewBatch()
	iw.finish(batch)
	batch.Write()

	iw, err := newIndexWriter(db, newAccountIdent(common.Hash{0xa}))
	if err != nil {
		t.Fatalf("Failed to construct the block writer, %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := iw.append(uint64(i + 100)); err != nil {
			t.Fatalf("Failed to append item, %v", err)
		}
	}
	iw.finish(db.NewBatch())
}

func TestIndexWriterDelete(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	iw, _ := newIndexWriter(db, newAccountIdent(common.Hash{0xa}))
	for i := 0; i < indexBlockEntriesCap*4; i++ {
		iw.append(uint64(i + 1))
	}
	batch := db.NewBatch()
	iw.finish(batch)
	batch.Write()

	// Delete unknown id, the request should be rejected
	id, _ := newIndexDeleter(db, newAccountIdent(common.Hash{0xa}))
	if err := id.pop(indexBlockEntriesCap * 5); err == nil {
		t.Fatal("Expect error to occur for unknown id")
	}
	for i := indexBlockEntriesCap * 4; i >= 1; i-- {
		if err := id.pop(uint64(i)); err != nil {
			t.Fatalf("Unexpected error for element popping, %v", err)
		}
		if id.lastID != uint64(i-1) {
			t.Fatalf("Unexpected lastID, want: %d, got: %d", uint64(i-1), iw.lastID)
		}
		if rand.Intn(10) == 0 {
			batch := db.NewBatch()
			id.finish(batch)
			batch.Write()
		}
	}
}

func TestBatchIndexerWrite(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		batch     = newBatchIndexer(db, false, typeStateHistory)
		histories = makeStateHistories(10)
	)
	for i, h := range histories {
		if err := batch.process(h, uint64(i+1)); err != nil {
			t.Fatalf("Failed to process history, %v", err)
		}
	}
	if err := batch.finish(true); err != nil {
		t.Fatalf("Failed to finish batch indexer, %v", err)
	}
	metadata := loadIndexMetadata(db, typeStateHistory)
	if metadata == nil || metadata.Last != uint64(10) {
		t.Fatal("Unexpected index position")
	}
	var (
		accounts = make(map[common.Hash][]uint64)
		storages = make(map[common.Hash]map[common.Hash][]uint64)
	)
	for i, h := range histories {
		for _, addr := range h.accountList {
			addrHash := crypto.Keccak256Hash(addr.Bytes())
			accounts[addrHash] = append(accounts[addrHash], uint64(i+1))

			if _, ok := storages[addrHash]; !ok {
				storages[addrHash] = make(map[common.Hash][]uint64)
			}
			for _, slot := range h.storageList[addr] {
				storages[addrHash][slot] = append(storages[addrHash][slot], uint64(i+1))
			}
		}
	}
	for addrHash, indexes := range accounts {
		ir, _ := newIndexReader(db, newAccountIdent(addrHash))
		for i := 0; i < len(indexes)-1; i++ {
			n, err := ir.readGreaterThan(indexes[i])
			if err != nil {
				t.Fatalf("Failed to read index, %v", err)
			}
			if n != indexes[i+1] {
				t.Fatalf("Unexpected result, want %d, got %d", indexes[i+1], n)
			}
		}
		n, err := ir.readGreaterThan(indexes[len(indexes)-1])
		if err != nil {
			t.Fatalf("Failed to read index, %v", err)
		}
		if n != math.MaxUint64 {
			t.Fatalf("Unexpected result, want math.MaxUint64, got %d", n)
		}
	}
	for addrHash, slots := range storages {
		for slotHash, indexes := range slots {
			ir, _ := newIndexReader(db, newStorageIdent(addrHash, slotHash))
			for i := 0; i < len(indexes)-1; i++ {
				n, err := ir.readGreaterThan(indexes[i])
				if err != nil {
					t.Fatalf("Failed to read index, %v", err)
				}
				if n != indexes[i+1] {
					t.Fatalf("Unexpected result, want %d, got %d", indexes[i+1], n)
				}
			}
			n, err := ir.readGreaterThan(indexes[len(indexes)-1])
			if err != nil {
				t.Fatalf("Failed to read index, %v", err)
			}
			if n != math.MaxUint64 {
				t.Fatalf("Unexpected result, want math.MaxUint64, got %d", n)
			}
		}
	}
}

func TestBatchIndexerDelete(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		bw        = newBatchIndexer(db, false, typeStateHistory)
		histories = makeStateHistories(10)
	)
	// Index histories
	for i, h := range histories {
		if err := bw.process(h, uint64(i+1)); err != nil {
			t.Fatalf("Failed to process history, %v", err)
		}
	}
	if err := bw.finish(true); err != nil {
		t.Fatalf("Failed to finish batch indexer, %v", err)
	}

	// Unindex histories
	bd := newBatchIndexer(db, true, typeStateHistory)
	for i := len(histories) - 1; i >= 0; i-- {
		if err := bd.process(histories[i], uint64(i+1)); err != nil {
			t.Fatalf("Failed to process history, %v", err)
		}
	}
	if err := bd.finish(true); err != nil {
		t.Fatalf("Failed to finish batch indexer, %v", err)
	}

	metadata := loadIndexMetadata(db, typeStateHistory)
	if metadata != nil {
		t.Fatal("Unexpected index position")
	}
	it := db.NewIterator(rawdb.StateHistoryIndexPrefix, nil)
	for it.Next() {
		t.Fatal("Leftover history index data")
	}
	it.Release()
}
