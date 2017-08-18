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

const BloomLength = 2048

// BloomBitsCreator takes SectionSize number of header bloom filters and calculates the bloomBits vectors of the section
type BloomBitsCreator struct {
	blooms              [BloomLength][]byte
	sectionSize, bitIndex uint64
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
	if b.bitIndex >= b.sectionSize {
		panic("too many header blooms added")
	}

	byteIdx := b.bitIndex / 8
	bitMask := byte(1) << byte(7-b.bitIndex%8)
	for bloomBitIdx, _ := range b.blooms {
		bloomByteIdx := BloomLength/8 - 1 - bloomBitIdx/8
		bloomBitMask := byte(1) << byte(bloomBitIdx%8)
		if (bloom[bloomByteIdx] & bloomBitMask) != 0 {
			b.blooms[bloomBitIdx][byteIdx] |= bitMask
		}
	}
	b.bitIndex++
}

// GetBitVector returns the bit vector belonging to the given bit index after header blooms have been added
func (b *BloomBitsCreator) GetBitVector(idx uint) []byte {
	if b.bitIndex != b.sectionSize {
		panic("not enough header blooms added")
	}

	return b.blooms[idx][:]
}
