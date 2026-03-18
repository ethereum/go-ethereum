// Copyright 2025 go-ethereum Authors
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

package bintrie

import (
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
)

// StemNode represents a group of `StemNodeWidth` values sharing the same stem.
// It uses a packed representation: bitmap indicates which of the 256 positions
// have values, and valueData stores the values contiguously in bitmap order.
type StemNode struct {
	Stem      [StemSize]byte       // Stem path to get to StemNodeWidth values
	bitmap    [StemBitmapSize]byte // bitmap indicating which positions have values
	valueData []byte              // packed value data (count * HashSize bytes)
	count     uint16              // number of values present
	depth     uint8               // Depth of the node
	shared    bool                // true if valueData is shared with serialized input

	mustRecompute bool        // true if the hash needs to be recomputed
	hash          common.Hash // cached hash when mustRecompute == false
}

// posInData returns the index within valueData for the given suffix.
// Returns -1 if the suffix is not present.
func (sn *StemNode) posInData(suffix byte) int {
	idx := int(suffix)
	if sn.bitmap[idx/8]>>(7-(idx%8))&1 == 0 {
		return -1
	}
	// Count the bits set before this position to determine the offset
	pos := 0
	byteIdx := idx / 8
	for i := 0; i < byteIdx; i++ {
		pos += bits.OnesCount8(sn.bitmap[i])
	}
	// Count bits in the partial byte
	mask := byte(0xFF) << (8 - (idx % 8))
	pos += bits.OnesCount8(sn.bitmap[byteIdx] & mask)
	return pos
}

// getValue returns the value at the given suffix, or nil if not present.
func (sn *StemNode) getValue(suffix byte) []byte {
	pos := sn.posInData(suffix)
	if pos < 0 {
		return nil
	}
	start := pos * HashSize
	return sn.valueData[start : start+HashSize]
}

// hasValue returns true if the given suffix has a value.
func (sn *StemNode) hasValue(suffix byte) bool {
	idx := int(suffix)
	return sn.bitmap[idx/8]>>(7-(idx%8))&1 == 1
}

// allValues returns all 256 values (nil for absent positions).
func (sn *StemNode) allValues() [][]byte {
	values := make([][]byte, StemNodeWidth)
	dataIdx := 0
	for i := range StemNodeWidth {
		if sn.bitmap[i/8]>>(7-(i%8))&1 == 1 {
			values[i] = sn.valueData[dataIdx*HashSize : (dataIdx+1)*HashSize]
			dataIdx++
		}
	}
	return values
}

// ensureWritable makes the valueData writable (copies if shared with serialized input).
func (sn *StemNode) ensureWritable() {
	if sn.shared || cap(sn.valueData)-len(sn.valueData) < HashSize {
		newData := make([]byte, len(sn.valueData), len(sn.valueData)+HashSize*4)
		copy(newData, sn.valueData)
		sn.valueData = newData
		sn.shared = false
	}
}

// setValue sets or inserts a value at the given suffix.
func (sn *StemNode) setValue(suffix byte, value []byte) {
	idx := int(suffix)
	pos := sn.posInData(suffix)
	if pos >= 0 {
		// Overwrite existing value
		copy(sn.valueData[pos*HashSize:], value[:HashSize])
		return
	}
	// New value: insert into bitmap and valueData at the correct position.
	sn.bitmap[idx/8] |= 1 << (7 - (idx % 8))
	sn.count++

	// Find the correct position in valueData (count bits before this position).
	insertPos := 0
	byteIdx := idx / 8
	for i := 0; i < byteIdx; i++ {
		insertPos += bits.OnesCount8(sn.bitmap[i])
	}
	mask := byte(0xFF) << (8 - (idx % 8))
	insertPos += bits.OnesCount8(sn.bitmap[byteIdx] & mask)

	// Insert value at the correct position in valueData.
	insertOffset := insertPos * HashSize
	// Grow the slice
	sn.valueData = append(sn.valueData, make([]byte, HashSize)...)
	// Shift data after insertion point
	copy(sn.valueData[insertOffset+HashSize:], sn.valueData[insertOffset:len(sn.valueData)-HashSize])
	// Copy the new value
	copy(sn.valueData[insertOffset:], value[:HashSize])
}

// Hash returns the hash of the node.
func (sn *StemNode) Hash() common.Hash {
	if !sn.mustRecompute {
		return sn.hash
	}

	var data [StemNodeWidth]common.Hash
	h := newSha256()
	defer returnSha256(h)

	// Hash each present value
	dataIdx := 0
	for i := range StemNodeWidth {
		if sn.bitmap[i/8]>>(7-(i%8))&1 == 1 {
			v := sn.valueData[dataIdx*HashSize : (dataIdx+1)*HashSize]
			h.Reset()
			h.Write(v)
			h.Sum(data[i][:0])
			dataIdx++
		}
	}
	h.Reset()

	for level := 1; level <= 8; level++ {
		for i := range StemNodeWidth / (1 << level) {
			h.Reset()

			if data[i*2] == (common.Hash{}) && data[i*2+1] == (common.Hash{}) {
				data[i] = common.Hash{}
				continue
			}

			h.Write(data[i*2][:])
			h.Write(data[i*2+1][:])
			data[i] = common.Hash(h.Sum(nil))
		}
	}

	h.Reset()
	h.Write(sn.Stem[:])
	h.Write([]byte{0})
	h.Write(data[0][:])
	sn.hash = common.BytesToHash(h.Sum(nil))
	sn.mustRecompute = false
	return sn.hash
}

// Key returns the full key for the given index.
func (sn *StemNode) Key(i int) []byte {
	var ret [HashSize]byte
	copy(ret[:], sn.Stem[:])
	ret[StemSize] = byte(i)
	return ret[:]
}
