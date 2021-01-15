// Copyright 2020 The go-ethereum Authors
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

package trie

import "bytes"

func newBinKey(key []byte) binkey {
	bits := make([]byte, 8*len(key))
	for i, kb := range key {
		// might be best to have this statement first, as compiler bounds-checking hint
		bits[8*i+7] = kb & 0x1
		bits[8*i] = (kb >> 7) & 0x1
		bits[8*i+1] = (kb >> 6) & 0x1
		bits[8*i+2] = (kb >> 5) & 0x1
		bits[8*i+3] = (kb >> 4) & 0x1
		bits[8*i+4] = (kb >> 3) & 0x1
		bits[8*i+5] = (kb >> 2) & 0x1
		bits[8*i+6] = (kb >> 1) & 0x1
	}
	return binkey(bits)
}
func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
func (b binkey) commonLength(other binkey) int {
	length := min(len(b), len(other))
	for i := 0; i < length; i++ {
		if b[i] != other[i] {
			return i
		}
	}
	return length
}

// Compare the prefix by the number of bytes; there is
// a twist for bit #254 and #255, for which 4 out of 5
// nodes are grouped into one.
func (b binkey) samePrefix(other binkey, off int) bool {
	var boundary = off + len(other)
	if boundary >= 255 && boundary <= 256 {
		boundary = 254
	}
	return bytes.Equal(b[off:boundary], other[:boundary-off])
}
