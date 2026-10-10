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

import "math/bits"

// indexBitset represents a set of indices using a bitmap.
type indexBitset struct {
	minIndex uint32
	maxIndex uint32
	bits     []uint64
}

// newIndexBitset creates a bitset from a list of indices.
// Returns an empty bitset if indices is empty.
func newIndexBitset(indices []uint32) *indexBitset {
	if len(indices) == 0 {
		return &indexBitset{}
	}
	// Find index range
	minIdx, maxIdx := indices[0], indices[0]
	for _, idx := range indices[1:] {
		if idx < minIdx {
			minIdx = idx
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	// Calculate number of uint64 needed
	rangeSize := maxIdx - minIdx + 1
	bitsCount := (rangeSize + 63) / 64
	bitset := &indexBitset{
		minIndex: minIdx,
		maxIndex: maxIdx,
		bits:     make([]uint64, bitsCount),
	}
	// Set all specified indices
	for _, idx := range indices {
		bitset.Set(idx)
	}
	return bitset
}

// Has checks if an index exists in the set.
func (b *indexBitset) Has(idx uint32) bool {
	if b.bits == nil || idx < b.minIndex || idx > b.maxIndex {
		return false
	}
	pos := idx - b.minIndex
	wordIdx := pos / 64
	bitIdx := pos % 64
	return (b.bits[wordIdx] & (1 << bitIdx)) != 0
}

// Set adds an index to the set.
func (b *indexBitset) Set(idx uint32) {
	if b.bits == nil || idx < b.minIndex || idx > b.maxIndex {
		return
	}
	pos := idx - b.minIndex
	wordIdx := pos / 64
	bitIdx := pos % 64
	b.bits[wordIdx] |= 1 << bitIdx
}

// Clear removes an index from the set.
func (b *indexBitset) Clear(idx uint32) {
	if b.bits == nil || idx < b.minIndex || idx > b.maxIndex {
		return
	}
	pos := idx - b.minIndex
	wordIdx := pos / 64
	bitIdx := pos % 64
	b.bits[wordIdx] &^= 1 << bitIdx
}

// Count returns the number of indices in the set.
func (b *indexBitset) Count() int {
	if b.bits == nil {
		return 0
	}
	count := 0
	for _, word := range b.bits {
		count += bits.OnesCount64(word)
	}
	return count
}

// IsEmpty checks if the set is empty.
func (b *indexBitset) IsEmpty() bool {
	if b.bits == nil {
		return true
	}
	for _, word := range b.bits {
		if word != 0 {
			return false
		}
	}
	return true
}

// Iterate traverses all indices in the set.
// The callback function fn is called with each index in the set.
// Iteration order is from smallest to largest.
func (b *indexBitset) Iterate(fn func(uint32)) {
	if b.bits == nil {
		return
	}
	for i, word := range b.bits {
		if word == 0 {
			continue
		}
		baseIdx := b.minIndex + uint32(i*64)
		for bitIdx := 0; bitIdx < 64; bitIdx++ {
			if (word & (1 << bitIdx)) != 0 {
				fn(baseIdx + uint32(bitIdx))
			}
		}
	}
}
