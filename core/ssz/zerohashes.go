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

package ssz

import "crypto/sha256"

// zeroHashes[d] is the root of a depth-d Merkle tree whose leaves are all
// zero chunks. Every node on a level of such a tree is the same value, since
// all its children are, so zeroHashes[0] is the zero chunk and
// zeroHashes[d+1] = H(zeroHashes[d] ‖ zeroHashes[d]). Merkleize substitutes
// these for empty subtrees, so padding to a limit costs one lookup per level
// rather than one hash per missing leaf. Depth 64 covers any uint64 chunk count.
var zeroHashes = func() [65][32]byte {
	var zh [65][32]byte
	for d := 1; d <= 64; d++ {
		var buf [64]byte
		copy(buf[:32], zh[d-1][:])
		copy(buf[32:], zh[d-1][:])
		zh[d] = sha256.Sum256(buf[:])
	}
	return zh
}()
