// Copyright 2017 The go-ethereum Authors
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
package bloombits

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type (
	BitVector  []byte
	CompVector []byte
)

// bvAnd binary ANDs b to a
func bvAnd(a, b BitVector) {
	for i, bb := range b {
		a[i] &= bb
	}
}

// bvOr binary ORs b to a
func bvOr(a, b BitVector) {
	for i, bb := range b {
		a[i] |= bb
	}
}

// bvZero returns an all-zero bit vector
func bvZero(sectionSize int) BitVector {
	return make(BitVector, sectionSize/8)
}

// bvCopy creates a copy of the given bit vector
// If the source vector is nil, returns an all-zero bit vector
func bvCopy(a BitVector, sectionSize int) BitVector {
	c := make(BitVector, sectionSize/8)
	copy(c, a)
	return c
}

// bvIsNonZero returns true if the bit vector has at least one "1" bit
func bvIsNonZero(a BitVector) bool {
	for _, b := range a {
		if b != 0 {
			return true
		}
	}
	return false
}

// CompressBloomBits compresses a bit vector for storage/network transfer purposes
func CompressBloomBits(bits BitVector, sectionSize int) CompVector {
	if len(bits) != sectionSize/8 {
		panic(nil)
	}
	c := compressBits(bits)
	if len(c) >= sectionSize/8 {
		// make a copy so that output is always detached from input
		return CompVector(bvCopy(bits, sectionSize))
	}
	return CompVector(c)
}

func compressBits(bits []byte) []byte {
	l := len(bits)
	ll := l / 8
	if ll == 0 {
		ll = 1
	}
	b := make([]byte, ll)
	c := make([]byte, l)
	cl := 0
	for i, v := range bits {
		if v != 0 {
			c[cl] = v
			cl++
			b[i/8] |= 1 << byte(7-i%8)
		}
	}
	if cl == 0 {
		return nil
	}
	if ll > 1 {
		b = compressBits(b)
	}
	return append(b, c[0:cl]...)
}

// DeompressBloomBits decompresses a bit vector
func DecompressBloomBits(bits CompVector, sectionSize int) BitVector {
	if len(bits) == sectionSize/8 {
		// make a copy so that output is always detached from input
		return bvCopy(BitVector(bits), sectionSize)
	}
	dc, ofs := decompressBits(bits, sectionSize/8)
	if ofs != len(bits) {
		panic(nil)
	}
	return dc
}

func decompressBits(bits []byte, targetLen int) ([]byte, int) {
	lb := len(bits)
	dc := make([]byte, targetLen)
	if lb == 0 {
		return dc, 0
	}

	l := targetLen / 8
	var (
		b   []byte
		ofs int
	)
	if l <= 1 {
		b = bits[0:1]
		ofs = 1
	} else {
		b, ofs = decompressBits(bits, l)
	}
	for i, _ := range dc {
		if b[i/8]&(1<<byte(7-i%8)) != 0 {
			if ofs == lb {
				panic(nil)
			}
			dc[i] = bits[ofs]
			ofs++
		}
	}
	return dc, ofs
}

const BloomLength = 2048

// BloomBitsCreator takes SectionSize number of header bloom filters and calculates the bloomBits vectors of the section
type BloomBitsCreator struct {
	blooms              [BloomLength][]byte
	sectionSize, bitIdx uint64
}

func NewBloomBitsCreator(sectionSize uint64) *BloomBitsCreator {
	b := &BloomBitsCreator{sectionSize: sectionSize}
	for i, _ := range b.blooms {
		b.blooms[i] = make([]byte, sectionSize/8)
	}
	return b
}

// AddHeaderBloom takes a single bloom filter and sets the corresponding bit column in memory accordingly
func (b *BloomBitsCreator) AddHeaderBloom(bloom types.Bloom) {
	if b.bitIdx >= b.sectionSize {
		panic("too many header blooms added")
	}

	byteIdx := b.bitIdx / 8
	bitMask := byte(1) << byte(7-b.bitIdx%8)
	for bloomBitIdx, _ := range b.blooms {
		bloomByteIdx := BloomLength/8 - 1 - bloomBitIdx/8
		bloomBitMask := byte(1) << byte(bloomBitIdx%8)
		if (bloom[bloomByteIdx] & bloomBitMask) != 0 {
			b.blooms[bloomBitIdx][byteIdx] |= bitMask
		}
	}
	b.bitIdx++
}

// GetBitVector returns the bit vector belonging to the given bit index after header blooms have been added
func (b *BloomBitsCreator) GetBitVector(idx uint) BitVector {
	if b.bitIdx != b.sectionSize {
		panic("not enough header blooms added")
	}

	return BitVector(b.blooms[idx][:])
}
