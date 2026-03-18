// Copyright 2026 The go-ethereum Authors
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

package types

import (
	"math/bits"
	"math/rand"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// `CustodyBitmap` is a bitmap to represent which custody index to store (little endian)
type CustodyBitmap [16]byte

var (
	CustodyBitmapAll = func() *CustodyBitmap {
		var result CustodyBitmap
		for i := range result {
			result[i] = 0xFF
		}
		return &result
	}()

	CustodyBitmapData = func() *CustodyBitmap {
		var result CustodyBitmap
		for i := 0; i < kzg4844.DataPerBlob/8; i++ {
			result[i] = 0xFF
		}
		return &result
	}()
)

func NewCustodyBitmap(indices []uint64) CustodyBitmap {
	var result CustodyBitmap
	for _, i := range indices {
		if i >= uint64(kzg4844.CellsPerBlob) {
			panic("CustodyBitmap: bit index out of range")
		}
		result[i/8] |= 1 << (i % 8)
	}
	return result
}

// NewRandomCustodyBitmap creates a CustodyBitmap with n randomly selected indices.
// This should be used only for tests.
func NewRandomCustodyBitmap(n int) CustodyBitmap {
	if n <= 0 || n > kzg4844.CellsPerBlob {
		panic("CustodyBitmap: invalid number of indices")
	}
	indices := make([]uint64, 0, n)
	used := make(map[uint64]bool)
	for len(indices) < n {
		idx := uint64(rand.Intn(kzg4844.CellsPerBlob))
		if !used[idx] {
			used[idx] = true
			indices = append(indices, idx)
		}
	}
	return NewCustodyBitmap(indices)
}

// IsSet returns whether bit i is set.
func (b CustodyBitmap) IsSet(i uint64) bool {
	if i >= uint64(kzg4844.CellsPerBlob) {
		return false
	}
	return (b[i/8]>>(i%8))&1 == 1
}

// OneCount returns the number of bits set to 1.
func (b CustodyBitmap) OneCount() int {
	total := 0
	for _, v := range b {
		total += bits.OnesCount8(v)
	}
	return total
}

// Indices returns the bit positions set to 1, in ascending order.
func (b CustodyBitmap) Indices() []uint64 {
	out := make([]uint64, 0, b.OneCount())
	for byteIdx, val := range b {
		v := val
		for v != 0 {
			tz := bits.TrailingZeros8(v)
			out = append(out, uint64(byteIdx*8+tz))
			v &^= 1 << tz
		}
	}
	return out
}

// Difference returns b AND NOT set (bits in b but not in set).
func (b CustodyBitmap) Difference(set *CustodyBitmap) *CustodyBitmap {
	var out CustodyBitmap
	for i := range b {
		out[i] = b[i] &^ set[i]
	}
	return &out
}

// Intersection returns b AND set.
func (b CustodyBitmap) Intersection(set *CustodyBitmap) *CustodyBitmap {
	var out CustodyBitmap
	for i := range b {
		out[i] = b[i] & set[i]
	}
	return &out
}

// Union returns b OR set.
func (b CustodyBitmap) Union(set *CustodyBitmap) *CustodyBitmap {
	var out CustodyBitmap
	for i := range b {
		out[i] = b[i] | set[i]
	}
	return &out
}
