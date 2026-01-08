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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// commonPrefixLen returns the length of the common prefix shared by a and b.
func commonPrefixLen(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// encodeIDs sorts the given list of uint16 IDs and encodes them into a
// compact byte slice using variable-length unsigned integer encoding.
func encodeIDs(ids []uint16) []byte {
	slices.Sort(ids)
	buf := make([]byte, 0, len(ids))
	for _, id := range ids {
		buf = binary.AppendUvarint(buf, uint64(id))
	}
	return buf
}

// decodeIDs decodes a sequence of variable-length encoded uint16 IDs from the
// given byte slice and returns them as a set.
//
// Returns an error if the input buffer does not contain a complete Uvarint value.
func decodeIDs(buf []byte) ([]uint16, error) {
	var res []uint16
	for len(buf) > 0 {
		id, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("too short for decoding node id, %v", buf)
		}
		buf = buf[n:]
		res = append(res, uint16(id))
	}
	return res, nil
}

// isAncestor reports whether node x is the ancestor of node y.
func isAncestor(x, y uint16) bool {
	for y > x {
		y = (y - 1) / 16 // parentID(y) = (y - 1) / 16
		if y == x {
			return true
		}
	}
	return false
}

// isBitSet reports whether the bit at `index` in the byte slice `b` is set.
func isBitSet(b []byte, index int) bool {
	return b[index/8]&(1<<(7-index%8)) != 0
}

// setBit sets the bit at `index` in the byte slice `b` to 1.
func setBit(b []byte, index int) {
	b[index/8] |= 1 << (7 - index%8)
}
