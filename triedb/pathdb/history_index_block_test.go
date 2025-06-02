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
)

func TestBlockReaderBasic(t *testing.T) {
	elements := []uint64{
		1, 5, 10, 11, 20,
	}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}

	br, err := newBlockReader(bw.finish())
	if err != nil {
		t.Fatalf("Failed to construct the block reader, %v", err)
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

func TestBlockReaderLarge(t *testing.T) {
	var elements []uint64
	for i := 0; i < 1000; i++ {
		elements = append(elements, rand.Uint64())
	}
	slices.Sort(elements)

	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}

	br, err := newBlockReader(bw.finish())
	if err != nil {
		t.Fatalf("Failed to construct the block reader, %v", err)
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

func TestBlockWriterBasic(t *testing.T) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	if !bw.empty() {
		t.Fatal("expected empty block")
	}
	bw.append(2)
	if err := bw.append(1); err == nil {
		t.Fatal("out-of-order insertion is not expected")
	}
	for i := 0; i < 10; i++ {
		bw.append(uint64(i + 3))
	}

	bw, err := newBlockWriter(bw.finish(), newIndexBlockDesc(0))
	if err != nil {
		t.Fatalf("Failed to construct the block writer, %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := bw.append(uint64(i + 100)); err != nil {
			t.Fatalf("Failed to append value %d: %v", i, err)
		}
	}
	bw.finish()
}

func TestBlockWriterDelete(t *testing.T) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < 10; i++ {
		bw.append(uint64(i + 1))
	}
	// Pop unknown id, the request should be rejected
	if err := bw.pop(100); err == nil {
		t.Fatal("Expect error to occur for unknown id")
	}
	for i := 10; i >= 1; i-- {
		if err := bw.pop(uint64(i)); err != nil {
			t.Fatalf("Unexpected error for element popping, %v", err)
		}
		empty := i == 1
		if empty != bw.empty() {
			t.Fatalf("Emptiness is not matched, want: %T, got: %T", empty, bw.empty())
		}
		newMax := uint64(i - 1)
		if bw.desc.max != newMax {
			t.Fatalf("Maxmium element is not matched, want: %d, got: %d", newMax, bw.desc.max)
		}
	}
}

func TestBlcokWriterDeleteWithData(t *testing.T) {
	elements := []uint64{
		1, 5, 10, 11, 20,
	}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}

	// Re-construct the block writer with data
	desc := &indexBlockDesc{
		id:      0,
		max:     20,
		entries: 5,
	}
	bw, err := newBlockWriter(bw.finish(), desc)
	if err != nil {
		t.Fatalf("Failed to construct block writer %v", err)
	}
	for i := len(elements) - 1; i > 0; i-- {
		if err := bw.pop(elements[i]); err != nil {
			t.Fatalf("Failed to pop element, %v", err)
		}
		newTail := elements[i-1]

		// Ensure the element can still be queried with no issue
		br, err := newBlockReader(bw.finish())
		if err != nil {
			t.Fatalf("Failed to construct the block reader, %v", err)
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
			want := c.result
			if c.value >= newTail {
				want = math.MaxUint64
			}
			got, err := br.readGreaterThan(c.value)
			if err != nil {
				t.Fatalf("Unexpected error, got %v", err)
			}
			if got != want {
				t.Fatalf("Unexpected result, got %v, wanted %v", got, want)
			}
		}
	}
}

func TestCorruptedIndexBlock(t *testing.T) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < 10; i++ {
		bw.append(uint64(i + 1))
	}
	buf := bw.finish()

	// Mutate the buffer manually
	buf[len(buf)-1]++
	_, err := newBlockWriter(buf, newIndexBlockDesc(0))
	if err == nil {
		t.Fatal("Corrupted index block data is not detected")
	}
}
