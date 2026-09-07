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

// Package ssz implements the Simple Serialize (SSZ) specification of the
// Ethereum consensus layer, following consensus-specs ssz/simple-serialize.md.
//
// The package currently covers merkleization: packing serialized basic values
// into 32-byte chunks and folding those chunks into a hash_tree_root. Types
// build their roots from the primitives here through hand-written code; there
// is no reflection and no code generation.
package ssz

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// depthOf returns the depth of the padded tree for a chunk limit:
// log2(next_pow_of_two(limit)), with limits 0 and 1 both mapping to depth 0.
func depthOf(limit uint64) int {
	if limit <= 1 {
		return 0
	}
	return bits.Len64(limit - 1)
}

// Merkleize computes merkleize(chunks, limit) of the SSZ spec: the root of a
// binary Merkle tree over the chunks, virtually zero-padded to
// next_pow_of_two(limit) leaves. Padding costs O(depth), not O(limit): a
// level's odd trailing node pairs with the zero-subtree hash of that level,
// and once real nodes are exhausted the accumulator folds against zero
// hashes only.
//
// A limit smaller than the chunk count is a programming error on the caller's
// side (types enforce their limits before merkleizing), so it panics rather than
// returning an error.
//
// For "no limit" semantics (containers, plain merkleize(chunks)), pass
// uint64(len(chunks)).
func Merkleize(chunks [][32]byte, limit uint64) [32]byte {
	if uint64(len(chunks)) > limit {
		panic(fmt.Sprintf("ssz: merkleize called with %d chunks over limit %d", len(chunks), limit))
	}
	depth := depthOf(limit)
	if len(chunks) == 0 {
		// Empty input: the whole tree is one zero subtree.
		return zeroHashes[depth]
	}
	// Fold the tree level by level, in place. The scratch buffer caps the
	// mutation to this function; chunk slices handed in by callers are only
	// read.
	layer := make([][32]byte, len(chunks), (len(chunks)+1)/2*2)
	copy(layer, chunks)
	for d := 0; d < depth; d++ {
		if len(layer) == 1 {
			// Real nodes exhausted: fold with zero subtrees the rest of the way.
			root := layer[0]
			for ; d < depth; d++ {
				root = hashPair(root, zeroHashes[d])
			}
			return root
		}
		if len(layer)%2 == 1 {
			layer = append(layer, zeroHashes[d])
		}
		hashPairs(layer[:len(layer)/2], layer)
		layer = layer[:len(layer)/2]
	}
	return layer[0]
}

// MixInLength returns hash(root ‖ uint256_le(length)): the length mix-in that
// distinguishes lists of different element counts (and empty lists from lists
// of zero values).
func MixInLength(root [32]byte, length uint64) [32]byte {
	var word [32]byte
	binary.LittleEndian.PutUint64(word[:8], length)
	return hashPair(root, word)
}

// MixInSelector returns hash(root ‖ uint256_le(selector)): the union variant
// mix-in.
func MixInSelector(root [32]byte, selector uint8) [32]byte {
	var word [32]byte
	word[0] = selector
	return hashPair(root, word)
}
