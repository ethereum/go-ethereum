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

package filtermaps

import (
	"testing"
)

func TestBitsetBasic(t *testing.T) {
	indices := []uint32{100, 101, 105, 110, 115}
	bs := newIndexBitset(indices)
	// Test Has - existing indices
	for _, idx := range indices {
		if !bs.Has(idx) {
			t.Errorf("Expected Has(%d) = true", idx)
		}
	}
	// Test Has - non-existing indices
	notInSet := []uint32{99, 102, 103, 104, 106, 107, 108, 109, 111, 116}
	for _, idx := range notInSet {
		if bs.Has(idx) {
			t.Errorf("Expected Has(%d) = false", idx)
		}
	}
	// Test Count
	if count := bs.Count(); count != len(indices) {
		t.Errorf("Expected Count = %d, got %d", len(indices), count)
	}
}

func TestBitsetClear(t *testing.T) {
	indices := []uint32{100, 101, 105, 110, 115}
	bs := newIndexBitset(indices)
	// Clear an index
	bs.Clear(105)
	if bs.Has(105) {
		t.Error("Expected Has(105) = false after Clear")
	}
	// Count should decrease
	if count := bs.Count(); count != 4 {
		t.Errorf("Expected Count = 4, got %d", count)
	}
	// Other indices should remain unaffected
	if !bs.Has(100) || !bs.Has(101) || !bs.Has(110) || !bs.Has(115) {
		t.Error("Other indices should remain")
	}
}

func TestBitsetSet(t *testing.T) {
	indices := []uint32{100, 105, 110}
	bs := newIndexBitset(indices)
	// Set a new index
	bs.Set(102)
	if !bs.Has(102) {
		t.Error("Expected Has(102) = true after Set")
	}
	// Count should increase
	if count := bs.Count(); count != 4 {
		t.Errorf("Expected Count = 4, got %d", count)
	}
}

func TestBitsetIterate(t *testing.T) {
	indices := []uint32{100, 101, 105, 110, 115}
	bs := newIndexBitset(indices)
	// Collect iterated indices
	var collected []uint32
	bs.Iterate(func(idx uint32) {
		collected = append(collected, idx)
	})
	// Verify count
	if len(collected) != len(indices) {
		t.Errorf("Expected %d indices, got %d", len(indices), len(collected))
	}
	// Verify all indices are present
	for _, idx := range indices {
		found := false
		for _, c := range collected {
			if c == idx {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Index %d not found in iteration", idx)
		}
	}
}

// Benchmark: Bitset vs Map
func BenchmarkBitsetVsMap(b *testing.B) {
	// Generate test data: 1000 consecutive indices
	indices := make([]uint32, 1000)
	for i := range indices {
		indices[i] = uint32(5000 + i)
	}

	b.Run("Bitset_Has", func(b *testing.B) {
		bs := newIndexBitset(indices)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bs.Has(5500)
		}
	})

	b.Run("Map_Has", func(b *testing.B) {
		m := make(map[uint32]struct{})
		for _, idx := range indices {
			m[idx] = struct{}{}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = m[5500]
		}
	})

	b.Run("Bitset_Set", func(b *testing.B) {
		bs := newIndexBitset(indices)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs.Set(5500)
		}
	})

	b.Run("Map_Set", func(b *testing.B) {
		m := make(map[uint32]struct{})
		for _, idx := range indices {
			m[idx] = struct{}{}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m[5500] = struct{}{}
		}
	})

	b.Run("Bitset_Clear", func(b *testing.B) {
		bs := newIndexBitset(indices)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs.Clear(5500)
			bs.Set(5500) // Reset for next iteration
		}
	})

	b.Run("Map_Delete", func(b *testing.B) {
		m := make(map[uint32]struct{})
		for _, idx := range indices {
			m[idx] = struct{}{}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			delete(m, 5500)
			m[5500] = struct{}{} // Re-add for next iteration
		}
	})
}
