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

import (
	"crypto/sha256"

	"github.com/prysmaticlabs/gohashtree"
)

// hashPair hashes one 64-byte node: H(a ‖ b).
func hashPair(a, b [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], a[:])
	copy(buf[32:], b[:])
	return sha256.Sum256(buf[:])
}

// hashPairs hashes 2n 32-byte chunks pairwise into n digests:
// digests[i] = H(chunks[2i] ‖ chunks[2i+1]). The call takes a whole tree
// level at once rather than one pair, because gohashtree hashes several
// independent nodes in parallel (SHA-NI or AVX on amd64, SHA2 instructions
// on arm64) and falls back to a portable implementation elsewhere. Digests
// may overlap the front of chunks, which is how the merkleization loop folds
// a level onto itself: digest i is written strictly behind the two chunks it
// reads.
func hashPairs(digests, chunks [][32]byte) {
	if len(chunks)%2 != 0 {
		panic("ssz: hashPairs called with an odd number of chunks")
	}
	if err := gohashtree.Hash(digests, chunks); err != nil {
		// Only reachable through a caller bug (digests too short).
		panic("ssz: " + err.Error())
	}
}
