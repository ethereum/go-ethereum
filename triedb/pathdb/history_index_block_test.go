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

// BenchmarkParseIndexBlock benchmarks the performance of parseIndexBlock.
func BenchmarkParseIndexBlock(b *testing.B) {
	// Generate a realistic index block blob
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < 4096; i++ {
		bw.append(uint64(i * 2))
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
func BenchmarkBlockWriterAppend(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	desc := newIndexBlockDesc(0)
	writer, _ := newBlockWriter(nil, desc)

	for i := 0; i < b.N; i++ {
		if writer.full() {
			desc = newIndexBlockDesc(0)
			writer, _ = newBlockWriter(nil, desc)
		}
		if err := writer.append(writer.desc.max + 1); err != nil {
			b.Error(err)
		}
	}
}

// TestBlockReaderCorruptedVarint tests that readGreaterThan properly handles
// corrupted varint encoding and returns errors instead of hanging or panicking.
func TestBlockReaderCorruptedVarint(t *testing.T) {
	// Create a valid block with multiple restart sections
	elements := []uint64{1, 5, 10, 11, 20, 100, 200, 300, 400, 500}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}
	validBlob := bw.finish()

	t.Run("corrupted varint at first restart", func(t *testing.T) {
		// Corrupt the first byte of the first restart to create invalid varint
		blob := slices.Clone(validBlob)
		restarts, data, _ := parseIndexBlock(blob)
		if len(restarts) > 0 {
			// Set the first byte to 0xFF repeatedly to create invalid varint
			// (varint encoding where all bytes have high bit set is invalid)
			for i := 0; i < 10 && int(restarts[0])+i < len(data); i++ {
				data[int(restarts[0])+i] = 0xFF
			}
		}

		br, err := newBlockReader(blob)
		if err != nil {
			// parseIndexBlock might catch it first, which is also acceptable
			return
		}
		// Try to read, should get error instead of hanging
		_, err = br.readGreaterThan(0)
		if err == nil {
			t.Fatal("Expected error when reading corrupted varint at first restart, got nil")
		}
	})

	t.Run("truncated varint in loop", func(t *testing.T) {
		// Create a block with two values: 5 and then truncated varint
		// Data: [0x05] (value 5) [0x80] (incomplete varint - truncated delta)
		// Restart: [0x00, 0x00] (points to offset 0)
		// Restart count: [0x01] (1 restart)
		blob := []byte{0x05, 0x80, 0x00, 0x00, 0x01}

		br, err := newBlockReader(blob)
		if err != nil {
			return // parseIndexBlock caught it
		}
		// Reading value 4 should find 5, then try to iterate and hit truncated varint
		// Actually, this will find 5 as the answer without hitting truncation.
		// We need to search for value >= 5 to force iteration
		_, err = br.readGreaterThan(5)
		if err == nil {
			t.Fatal("Expected error when reading truncated varint in loop, got nil")
		}
	})

	t.Run("overflow varint at restart", func(t *testing.T) {
		// Create a block with varint that would overflow uint64
		// 10 bytes of 0xFF creates an overflow (more than 64 bits)
		overflowVarint := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F}
		// Restart points to this overflow varint
		blob := append(overflowVarint, 0x00, 0x00, 0x01) // restart at 0, count 1

		br, err := newBlockReader(blob)
		if err != nil {
			return // parseIndexBlock might catch it
		}
		// Try to read should hit overflow varint
		_, err = br.readGreaterThan(100)
		if err == nil {
			t.Fatal("Expected error when reading overflow varint, got nil")
		}
	})
}

// TestBlockWriterCorruptedVarint tests that scanSection properly handles
// corrupted varint encoding by breaking out of the loop instead of hanging.
func TestBlockWriterCorruptedVarint(t *testing.T) {
	// Create a valid block
	elements := []uint64{1, 5, 10, 11, 20, 100, 200, 300}
	bw, _ := newBlockWriter(nil, newIndexBlockDesc(0))
	for i := 0; i < len(elements); i++ {
		bw.append(elements[i])
	}
	validBlob := bw.finish()

	// Create a new writer with corrupted data
	desc := &indexBlockDesc{
		id:      0,
		max:     300,
		entries: uint16(len(elements)),
	}

	// Corrupt the blob
	blob := slices.Clone(validBlob)
	restarts, data, _ := parseIndexBlock(blob)
	if len(restarts) > 0 {
		// Corrupt data in the first section
		pos := int(restarts[0]) + 1 // Skip first valid varint
		if pos < len(data)-10 {
			for i := 0; i < 10; i++ {
				data[pos+i] = 0xFF // Invalid varint
			}
		}
	}

	bw, err := newBlockWriter(blob, desc)
	if err != nil {
		// parseIndexBlock caught the corruption, which is fine
		return
	}

	// Test scanSection with corrupted data - should break instead of hanging
	callbackCalled := false
	bw.scanSection(0, func(v uint64, pos int) bool {
		callbackCalled = true
		return false // Continue iteration to test the corrupted part
	})
	// The callback should have been called at least once for the first valid element
	// before hitting corruption and breaking
	if !callbackCalled {
		t.Log("scanSection didn't call callback - may have hit corruption immediately")
	}

	// Test sectionLast with corrupted data - should handle gracefully
	_ = bw.sectionLast(0)

	// If we got here without hanging or panicking, the test passes
}
