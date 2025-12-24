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
	"bytes"
	"math"
	"math/rand"
	"slices"
	"sort"
	"testing"
)

func randomExt(bitmapSize int, n int) []uint16 {
	if bitmapSize == 0 {
		return nil
	}
	var (
		limit   = bitmapSize * 8
		extList []uint16
	)
	for i := 0; i < n; i++ {
		extList = append(extList, uint16(rand.Intn(limit+1)))
	}
	return extList
}

func TestBlockReaderBasic(t *testing.T) {
	testBlockReaderBasic(t, 0)
	testBlockReaderBasic(t, 2)
	testBlockReaderBasic(t, 34)
}

func testBlockReaderBasic(t *testing.T, bitmapSize int) {
	elements := []uint64{
		1, 5, 10, 11, 20,
	}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i], randomExt(bitmapSize, 5))
	}

	br, err := newBlockReader(bw.finish(), bitmapSize != 0)
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
	testBlockReaderLarge(t, 0)
	testBlockReaderLarge(t, 2)
	testBlockReaderLarge(t, 34)
}

func testBlockReaderLarge(t *testing.T, bitmapSize int) {
	var elements []uint64
	for i := 0; i < 1000; i++ {
		elements = append(elements, rand.Uint64())
	}
	slices.Sort(elements)

	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i], randomExt(bitmapSize, 5))
	}

	br, err := newBlockReader(bw.finish(), bitmapSize != 0)
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
	testBlockWriteBasic(t, 0)
	testBlockWriteBasic(t, 2)
	testBlockWriteBasic(t, 34)
}

func testBlockWriteBasic(t *testing.T, bitmapSize int) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
	if !bw.empty() {
		t.Fatal("expected empty block")
	}
	bw.append(2, randomExt(bitmapSize, 5))
	if err := bw.append(1, randomExt(bitmapSize, 5)); err == nil {
		t.Fatal("out-of-order insertion is not expected")
	}
	var maxElem uint64
	for i := 0; i < 10; i++ {
		bw.append(uint64(i+3), randomExt(bitmapSize, 5))
		maxElem = uint64(i + 3)
	}

	bw, err := newBlockWriter(bw.finish(), newIndexBlockDesc(0, bitmapSize), maxElem, bitmapSize != 0)
	if err != nil {
		t.Fatalf("Failed to construct the block writer, %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := bw.append(uint64(i+100), randomExt(bitmapSize, 5)); err != nil {
			t.Fatalf("Failed to append value %d: %v", i, err)
		}
	}
	bw.finish()
}

func TestBlockWriterWithLimit(t *testing.T) {
	testBlockWriterWithLimit(t, 0)
	testBlockWriterWithLimit(t, 2)
	testBlockWriterWithLimit(t, 34)
}

func testBlockWriterWithLimit(t *testing.T, bitmapSize int) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)

	var bitmaps [][]byte
	for i := 0; i < indexBlockRestartLen+2; i++ {
		bw.append(uint64(i+1), randomExt(bitmapSize, 5))
		bitmaps = append(bitmaps, bytes.Clone(bw.desc.extBitmap))
	}
	for i := 0; i < indexBlockRestartLen+2; i++ {
		limit := uint64(i + 1)

		desc := bw.desc.copy()
		block, err := newBlockWriter(bytes.Clone(bw.finish()), desc, limit, bitmapSize != 0)
		if err != nil {
			t.Fatalf("Failed to construct the block writer, %v", err)
		}
		if block.desc.max != limit {
			t.Fatalf("Test %d, unexpected max value, got %d, want %d", i, block.desc.max, limit)
		}
		if !bytes.Equal(desc.extBitmap, bitmaps[i]) {
			t.Fatalf("Test %d, unexpected bitmap, got: %v, want: %v", i, block.desc.extBitmap, bitmaps[i])
		}

		// Re-fill the elements
		var maxElem uint64
		for elem := limit + 1; elem < indexBlockRestartLen+4; elem++ {
			if err := block.append(elem, randomExt(bitmapSize, 5)); err != nil {
				t.Fatalf("Failed to append value %d: %v", elem, err)
			}
			maxElem = elem
		}
		if block.desc.max != maxElem {
			t.Fatalf("Test %d, unexpected max value, got %d, want %d", i, block.desc.max, maxElem)
		}
	}
}

func TestBlockWriterDelete(t *testing.T) {
	testBlockWriterDelete(t, 0)
	testBlockWriterDelete(t, 2)
	testBlockWriterDelete(t, 34)
}

func testBlockWriterDelete(t *testing.T, bitmapSize int) {
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
	for i := 0; i < 10; i++ {
		bw.append(uint64(i+1), randomExt(bitmapSize, 5))
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
	testBlcokWriterDeleteWithData(t, 0)
	testBlcokWriterDeleteWithData(t, 2)
	testBlcokWriterDeleteWithData(t, 34)
}

func testBlcokWriterDeleteWithData(t *testing.T, bitmapSize int) {
	elements := []uint64{
		1, 5, 10, 11, 20,
	}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, bitmapSize), 0, bitmapSize != 0)
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i], randomExt(bitmapSize, 5))
	}

	// Re-construct the block writer with data
	desc := &indexBlockDesc{
		id:      0,
		max:     20,
		entries: 5,
	}
	if bitmapSize > 0 {
		desc.extBitmap = make([]byte, bitmapSize)
	}
	bw, err := newBlockWriter(bw.finish(), desc, elements[len(elements)-1], bitmapSize != 0)
	if err != nil {
		t.Fatalf("Failed to construct block writer %v", err)
	}
	for i := len(elements) - 1; i > 0; i-- {
		if err := bw.pop(elements[i]); err != nil {
			t.Fatalf("Failed to pop element, %v", err)
		}
		newTail := elements[i-1]

		// Ensure the element can still be queried with no issue
		br, err := newBlockReader(bw.finish(), bitmapSize != 0)
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
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, 0), 0, false)

	var maxElem uint64
	for i := 0; i < 10; i++ {
		bw.append(uint64(i+1), nil)
		maxElem = uint64(i + 1)
	}
	buf := bw.finish()

	// Mutate the buffer manually
	buf[len(buf)-1]++
	_, err := newBlockWriter(buf, newIndexBlockDesc(0, 0), maxElem, false)
	if err == nil {
		t.Fatal("Corrupted index block data is not detected")
	}
}

// BenchmarkParseIndexBlock benchmarks the performance of parseIndexBlock.
//
// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/triedb/pathdb
// cpu: Apple M1 Pro
// BenchmarkParseIndexBlock
// BenchmarkParseIndexBlock-8   	35829495	        34.16 ns/op
func BenchmarkParseIndexBlock(b *testing.B) {
	// Generate a realistic index block blob
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, 0), 0, false)
	for i := 0; i < 4096; i++ {
		bw.append(uint64(i*2), nil)
	}
	blob := bw.finish()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parseIndexBlock(blob)
		if err != nil {
			b.Fatalf("parseIndexBlock failed: %v", err)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/triedb/pathdb
// cpu: Apple M1 Pro
// BenchmarkParseIndexBlockWithExt
// BenchmarkParseIndexBlockWithExt-8   	35773242	        33.72 ns/op
func BenchmarkParseIndexBlockWithExt(b *testing.B) {
	// Generate a realistic index block blob
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0, 34), 0, true)
	for i := 0; i < 4096; i++ {
		id, ext := uint64(i*2), randomExt(34, 3)
		bw.append(id, ext)
	}
	blob := bw.finish()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parseIndexBlock(blob)
		if err != nil {
			b.Fatalf("parseIndexBlock failed: %v", err)
		}
	}
}

// BenchmarkBlockWriterAppend benchmarks the performance of indexblock.writer
//
// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/triedb/pathdb
// cpu: Apple M1 Pro
// BenchmarkBlockWriterAppend
// BenchmarkBlockWriterAppend-8   	293611083	         4.113 ns/op	       3 B/op	       0 allocs/op
func BenchmarkBlockWriterAppend(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	var blockID uint32
	desc := newIndexBlockDesc(blockID, 0)
	writer, _ := newBlockWriter(nil, desc, 0, false)

	for i := 0; i < b.N; i++ {
		if writer.estimateFull(nil) {
			blockID += 1
			desc = newIndexBlockDesc(blockID, 0)
			writer, _ = newBlockWriter(nil, desc, 0, false)
		}
		if err := writer.append(writer.desc.max+1, nil); err != nil {
			b.Error(err)
		}
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/triedb/pathdb
// cpu: Apple M1 Pro
// BenchmarkBlockWriterAppendWithExt
// BenchmarkBlockWriterAppendWithExt-8   	11123844	       103.6 ns/op	      42 B/op	       2 allocs/op
func BenchmarkBlockWriterAppendWithExt(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	var (
		bitmapSize = 34
		blockID    uint32
	)
	desc := newIndexBlockDesc(blockID, bitmapSize)
	writer, _ := newBlockWriter(nil, desc, 0, true)

	for i := 0; i < b.N; i++ {
		ext := randomExt(bitmapSize, 3)
		if writer.estimateFull(ext) {
			blockID += 1
			desc = newIndexBlockDesc(blockID, bitmapSize)
			writer, _ = newBlockWriter(nil, desc, 0, true)
		}
		if err := writer.append(writer.desc.max+1, ext); err != nil {
			b.Error(err)
		}
	}
}
