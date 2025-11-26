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
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func makeTestIndexBlock(count int) ([]byte, []uint64) {
	var (
		marks    = make(map[uint64]bool)
		elements []uint64
	)
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < count; i++ {
		n := uint64(rand.Uint32())
		if marks[n] {
			continue
		}
		marks[n] = true
		elements = append(elements, n)
	}
	sort.Slice(elements, func(i, j int) bool { return elements[i] < elements[j] })

	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}
	data := bw.finish()

	return data, elements
}

func makeTestIndexBlocks(db ethdb.KeyValueStore, stateIdent stateIdent, count int) []uint64 {
	var (
		marks    = make(map[uint64]bool)
		elements []uint64
	)
	for i := 0; i < count; i++ {
		n := uint64(rand.Uint32())
		if marks[n] {
			continue
		}
		marks[n] = true
		elements = append(elements, n)
	}
	sort.Slice(elements, func(i, j int) bool { return elements[i] < elements[j] })

	iw, _ := newIndexWriter(db, stateIdent)
	for i := 0; i < len(elements); i++ {
		iw.append(elements[i])
	}
	batch := db.NewBatch()
	iw.finish(batch)
	batch.Write()

	return elements
}

func checkSeekGT(it HistoryIndexIterator, input uint64, exp bool, expVal uint64) error {
	found := it.SeekGT(input)
	if it.Error() != nil {
		return it.Error()
	}
	if !exp {
		if found {
			return fmt.Errorf("unexpected returned value: %d", it.ID())
		}
		return nil
	}
	if !found {
		return fmt.Errorf("element grearter than %d is not found", input)
	}
	if it.ID() != expVal {
		return fmt.Errorf("unexpected returned value, want: %d, got: %d", expVal, it.ID())
	}
	return nil
}

func checkNext(it HistoryIndexIterator, values []uint64) error {
	for _, value := range values {
		if !it.Next() {
			return errors.New("iterator is exhausted")
		}
		if it.ID() != value {
			return fmt.Errorf("unexpected iterator ID, want: %v, got: %v", value, it.ID())
		}
	}
	if it.Next() {
		return errors.New("iterator is not exhausted yet")
	}
	return it.Error()
}

func TestBlockIteratorSeekGT(t *testing.T) {
	/* 0-size index block is not allowed

	data, elements := makeTestIndexBlock(0)
	testBlockIterator(t, data, elements)
	*/

	data, elements := makeTestIndexBlock(1)
	testBlockIterator(t, data, elements)

	data, elements = makeTestIndexBlock(indexBlockRestartLen)
	testBlockIterator(t, data, elements)

	data, elements = makeTestIndexBlock(3 * indexBlockRestartLen)
	testBlockIterator(t, data, elements)

	data, elements = makeTestIndexBlock(indexBlockEntriesCap)
	testBlockIterator(t, data, elements)
}

func testBlockIterator(t *testing.T, data []byte, elements []uint64) {
	br, err := newBlockReader(data)
	if err != nil {
		t.Fatalf("Failed to open the block for reading, %v", err)
	}
	it := newBlockIterator(br.data, br.restarts)

	for i := 0; i < 128; i++ {
		var input uint64
		if rand.Intn(2) == 0 {
			input = elements[rand.Intn(len(elements))]
		} else {
			input = uint64(rand.Uint32())
		}
		index := sort.Search(len(elements), func(i int) bool {
			return elements[i] > input
		})
		var (
			exp     bool
			expVal  uint64
			remains []uint64
		)
		if index == len(elements) {
			exp = false
		} else {
			exp = true
			expVal = elements[index]
			if index < len(elements) {
				remains = elements[index+1:]
			}
		}
		if err := checkSeekGT(it, input, exp, expVal); err != nil {
			t.Fatal(err)
		}
		if exp {
			if err := checkNext(it, remains); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestIndexIteratorSeekGT(t *testing.T) {
	ident := newAccountIdent(common.Hash{0x1})

	dbA := rawdb.NewMemoryDatabase()
	testIndexIterator(t, ident, dbA, makeTestIndexBlocks(dbA, ident, 1))

	dbB := rawdb.NewMemoryDatabase()
	testIndexIterator(t, ident, dbB, makeTestIndexBlocks(dbB, ident, 3*indexBlockEntriesCap))

	dbC := rawdb.NewMemoryDatabase()
	testIndexIterator(t, ident, dbC, makeTestIndexBlocks(dbC, ident, indexBlockEntriesCap-1))

	dbD := rawdb.NewMemoryDatabase()
	testIndexIterator(t, ident, dbD, makeTestIndexBlocks(dbD, ident, indexBlockEntriesCap+1))
}

func testIndexIterator(t *testing.T, stateIdent stateIdent, db ethdb.Database, elements []uint64) {
	ir, err := newIndexReader(db, stateIdent)
	if err != nil {
		t.Fatalf("Failed to open the index reader, %v", err)
	}
	it := newIndexIterator(ir.descList, func(id uint32) (*blockReader, error) {
		return newBlockReader(readStateIndexBlock(stateIdent, db, id))
	})

	for i := 0; i < 128; i++ {
		var input uint64
		if rand.Intn(2) == 0 {
			input = elements[rand.Intn(len(elements))]
		} else {
			input = uint64(rand.Uint32())
		}
		index := sort.Search(len(elements), func(i int) bool {
			return elements[i] > input
		})
		var (
			exp     bool
			expVal  uint64
			remains []uint64
		)
		if index == len(elements) {
			exp = false
		} else {
			exp = true
			expVal = elements[index]
			if index < len(elements) {
				remains = elements[index+1:]
			}
		}
		if err := checkSeekGT(it, input, exp, expVal); err != nil {
			t.Fatal(err)
		}
		if exp {
			if err := checkNext(it, remains); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestBlockIteratorTraversal(t *testing.T) {
	/* 0-size index block is not allowed

	data, elements := makeTestIndexBlock(0)
	testBlockIterator(t, data, elements)
	*/

	data, elements := makeTestIndexBlock(1)
	testBlockIteratorTraversal(t, data, elements)

	data, elements = makeTestIndexBlock(indexBlockRestartLen)
	testBlockIteratorTraversal(t, data, elements)

	data, elements = makeTestIndexBlock(3 * indexBlockRestartLen)
	testBlockIteratorTraversal(t, data, elements)

	data, elements = makeTestIndexBlock(indexBlockEntriesCap)
	testBlockIteratorTraversal(t, data, elements)
}

func testBlockIteratorTraversal(t *testing.T, data []byte, elements []uint64) {
	br, err := newBlockReader(data)
	if err != nil {
		t.Fatalf("Failed to open the block for reading, %v", err)
	}
	it := newBlockIterator(br.data, br.restarts)

	if err := checkNext(it, elements); err != nil {
		t.Fatal(err)
	}
}

func TestIndexIteratorTraversal(t *testing.T) {
	ident := newAccountIdent(common.Hash{0x1})

	dbA := rawdb.NewMemoryDatabase()
	testIndexIteratorTraversal(t, ident, dbA, makeTestIndexBlocks(dbA, ident, 1))

	dbB := rawdb.NewMemoryDatabase()
	testIndexIteratorTraversal(t, ident, dbB, makeTestIndexBlocks(dbB, ident, 3*indexBlockEntriesCap))

	dbC := rawdb.NewMemoryDatabase()
	testIndexIteratorTraversal(t, ident, dbC, makeTestIndexBlocks(dbC, ident, indexBlockEntriesCap-1))

	dbD := rawdb.NewMemoryDatabase()
	testIndexIteratorTraversal(t, ident, dbD, makeTestIndexBlocks(dbD, ident, indexBlockEntriesCap+1))
}

func testIndexIteratorTraversal(t *testing.T, stateIdent stateIdent, db ethdb.KeyValueReader, elements []uint64) {
	ir, err := newIndexReader(db, stateIdent)
	if err != nil {
		t.Fatalf("Failed to open the index reader, %v", err)
	}
	it := newIndexIterator(ir.descList, func(id uint32) (*blockReader, error) {
		return newBlockReader(readStateIndexBlock(stateIdent, db, id))
	})
	if err := checkNext(it, elements); err != nil {
		t.Fatal(err)
	}
}
