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
	"maps"
	"math/rand"
	"slices"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func checkExt(f *extFilter, ext []uint16) bool {
	if f == nil {
		return true
	}
	fn := uint16(*f)

	for _, n := range ext {
		if n == fn {
			return true
		}
		if isAncestor(fn, n) {
			return true
		}
	}
	return false
}

func makeTestIndexBlock(count int, bitmapSize int) ([]byte, []uint64, [][]uint16) {
	var (
		marks    = make(map[uint64]bool)
		elements = make([]uint64, 0, count)
		extList  = make([][]uint16, 0, count)
	)
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
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
		ext := randomExt(bitmapSize, 5)
		extList = append(extList, ext)
		bw.append(elements[i], ext)
	}
	data := bw.finish()

	return data, elements, extList
}

func makeTestIndexBlocks(db ethdb.KeyValueStore, stateIdent stateIdent, count int, bitmapSize int) ([]uint64, [][]uint16) {
	var (
		marks    = make(map[uint64]bool)
		elements []uint64
		extList  [][]uint16
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

	iw, _ := newIndexWriter(db, stateIdent, 0, bitmapSize)
	for i := 0; i < len(elements); i++ {
		ext := randomExt(bitmapSize, 5)
		extList = append(extList, ext)
		iw.append(elements[i], ext)
	}
	batch := db.NewBatch()
	iw.finish(batch)
	batch.Write()

	return elements, extList
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

func verifySeekGT(t *testing.T, elements []uint64, ext [][]uint16, newIter func(filter *extFilter) HistoryIndexIterator) {
	set := make(map[extFilter]bool)
	for _, extList := range ext {
		for _, f := range extList {
			set[extFilter(f)] = true
		}
	}
	filters := slices.Collect(maps.Keys(set))

	for i := 0; i < 128; i++ {
		var filter *extFilter
		if rand.Intn(2) == 0 && len(filters) > 0 {
			filter = &filters[rand.Intn(len(filters))]
		} else {
			filter = nil
		}

		var input uint64
		if rand.Intn(2) == 0 {
			input = elements[rand.Intn(len(elements))]
		} else {
			input = uint64(rand.Uint32())
		}

		index := sort.Search(len(elements), func(i int) bool {
			return elements[i] > input
		})
		for index < len(elements) {
			if checkExt(filter, ext[index]) {
				break
			}
			index++
		}

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

			index++
			for index < len(elements) {
				if checkExt(filter, ext[index]) {
					remains = append(remains, elements[index])
				}
				index++
			}
		}

		it := newIter(filter)
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

func verifyTraversal(t *testing.T, elements []uint64, ext [][]uint16, newIter func(filter *extFilter) HistoryIndexIterator) {
	set := make(map[extFilter]bool)
	for _, extList := range ext {
		for _, f := range extList {
			set[extFilter(f)] = true
		}
	}
	filters := slices.Collect(maps.Keys(set))

	for i := 0; i < 16; i++ {
		var filter *extFilter
		if len(filters) > 0 {
			filter = &filters[rand.Intn(len(filters))]
		} else {
			filter = nil
		}
		it := newIter(filter)

		var (
			pos int
			exp []uint64
		)
		for pos < len(elements) {
			if checkExt(filter, ext[pos]) {
				exp = append(exp, elements[pos])
			}
			pos++
		}
		if err := checkNext(it, exp); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBlockIteratorSeekGT(t *testing.T) {
	for _, size := range []int{0, 2, 34} {
		for _, n := range []int{1, indexBlockRestartLen, 3 * indexBlockRestartLen} {
			data, elements, ext := makeTestIndexBlock(n, size)

			verifySeekGT(t, elements, ext, func(filter *extFilter) HistoryIndexIterator {
				br, err := newBlockReader(data, size != 0)
				if err != nil {
					t.Fatalf("Failed to open the block for reading, %v", err)
				}
				return br.newIterator(filter)
			})
		}
	}
}

func TestIndexIteratorSeekGT(t *testing.T) {
	ident := newAccountIdent(common.Hash{0x1})

	for _, size := range []int{0, 2, 34} {
		for _, n := range []int{1, 4096, 3 * 4096} {
			db := rawdb.NewMemoryDatabase()
			elements, ext := makeTestIndexBlocks(db, ident, n, size)

			verifySeekGT(t, elements, ext, func(filter *extFilter) HistoryIndexIterator {
				ir, err := newIndexReader(db, ident, size)
				if err != nil {
					t.Fatalf("Failed to open the index reader, %v", err)
				}
				return ir.newIterator(filter)
			})
		}
	}
}

func TestBlockIteratorTraversal(t *testing.T) {
	/* 0-size index block is not allowed

	data, elements := makeTestIndexBlock(0)
	testBlockIterator(t, data, elements)
	*/

	for _, size := range []int{0, 2, 34} {
		for _, n := range []int{1, indexBlockRestartLen, 3 * indexBlockRestartLen} {
			data, elements, ext := makeTestIndexBlock(n, size)

			verifyTraversal(t, elements, ext, func(filter *extFilter) HistoryIndexIterator {
				br, err := newBlockReader(data, size != 0)
				if err != nil {
					t.Fatalf("Failed to open the block for reading, %v", err)
				}
				return br.newIterator(filter)
			})
		}
	}
}

func TestIndexIteratorTraversal(t *testing.T) {
	ident := newAccountIdent(common.Hash{0x1})

	for _, size := range []int{0, 2, 34} {
		for _, n := range []int{1, 4096, 3 * 4096} {
			db := rawdb.NewMemoryDatabase()
			elements, ext := makeTestIndexBlocks(db, ident, n, size)

			verifyTraversal(t, elements, ext, func(filter *extFilter) HistoryIndexIterator {
				ir, err := newIndexReader(db, ident, size)
				if err != nil {
					t.Fatalf("Failed to open the index reader, %v", err)
				}
				return ir.newIterator(filter)
			})
		}
	}
}

func TestSeqIterBasicIteration(t *testing.T) {
	it := newSeqIter(5) // iterates over [1..5]
	it.SeekGT(0)

	var (
		got      []uint64
		expected = []uint64{1, 2, 3, 4, 5}
	)
	got = append(got, it.ID())
	for it.Next() {
		got = append(got, it.ID())
	}
	if len(got) != len(expected) {
		t.Fatalf("iteration length mismatch: got %v, expected %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("element mismatch at %d: got %d, expected %d", i, got[i], expected[i])
		}
	}
}

func TestSeqIterSeekGT(t *testing.T) {
	it := newSeqIter(5)

	tests := []struct {
		input    uint64
		ok       bool
		expected uint64
	}{
		{0, true, 1},
		{1, true, 2},
		{4, true, 5},
		{5, false, 0}, // 6 is out of range
	}
	for _, tt := range tests {
		ok := it.SeekGT(tt.input)
		if ok != tt.ok {
			t.Fatalf("SeekGT(%d) ok mismatch: got %v, expected %v", tt.input, ok, tt.ok)
		}
		if ok && it.ID() != tt.expected {
			t.Fatalf("SeekGT(%d) positioned at %d, expected %d", tt.input, it.ID(), tt.expected)
		}
	}
}
