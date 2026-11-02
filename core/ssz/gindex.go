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
	"fmt"
	"math/bits"
)

// Generalized indices name the nodes of a Merkle tree: the root is 1 and the
// children of node g are 2g and 2g+1, so an index written in binary is the
// path to its node, one bit per level after the leading 1, 0 for left and 1
// for right. The arithmetic and the conventions here are those of
// ethereum/ssz-specs src/ssz/gindex.py (gindex_depth, gindex_concat,
// progressive_chunk_gindex), which is where the progressive shapes got their
// indices; consensus-specs' merkle-proofs.md never had them.
//
// Positions under a type root, as the specification lays them out: a list,
// bitlist, progressive list or progressive bitlist has its data tree at 2 and
// its length word at 3; a progressive container has its field tree at 2 and
// its active_fields word at 3; a vector or container mixes nothing in, so its
// chunks hang off the root directly.

// maxProgressiveChunk bounds the chunk indices ProgressiveChunkGindex accepts.
// The SSZ size envelope caps any chunk count at 2^32; indices below 2^40 stay
// within spine level 20, whose indices fit a uint64, so the bound costs no
// real position and turns an overflow into a panic.
const maxProgressiveChunk = 1 << 40

// GindexDepth returns the number of levels index sits below the root, which
// is the number of siblings a proof of it carries. The root has depth 0. Zero
// names no node and panics.
func GindexDepth(index uint64) int {
	if index == 0 {
		panic("ssz: zero generalized index")
	}
	return bits.Len64(index) - 1
}

// ConcatGindex is concat_generalized_indices of the specification: it rebases
// inner, an index measured from the root of a nested value, onto outer, the
// position of that value in the enclosing tree. The inner path replaces the
// outer index's leading bit, so the result splices rather than multiplies:
// ConcatGindex(2, 24) is 40, not 48. Zero names no node and panics.
func ConcatGindex(outer, inner uint64) uint64 {
	if outer == 0 {
		panic("ssz: zero generalized index")
	}
	depth := GindexDepth(inner)
	return outer<<depth | (inner - 1<<depth)
}

// ChunkGindex returns the index of chunk index in a merkleize(chunks, limit)
// tree, measured from that tree's root: 2^depth + index for the depth of the
// padded tree, so a one-chunk capacity puts its chunk at the root itself. A
// limit of 0 rounds to one leaf, as merkleize rounds it (List[T, 0] is
// legal). Every leaf of the padded tree is a node, padding past the limit
// included, so any index below next_pow_of_two(limit) names one; an index
// beyond that names nothing and panics, both arguments being a type's own
// numbers. Whether a type DECLARES a position (an element below the limit, a
// field) is the caller's question, as it is the path layer's in the
// reference, and whether a value reaches it is the prover's.
func ChunkGindex(limit, index uint64) uint64 {
	width := uint64(1) << depthOf(limit)
	if index >= width {
		panic(fmt.Sprintf("ssz: chunk %d outside the %d-leaf tree of limit %d", index, width, limit))
	}
	return width + index
}

// ProgressiveChunkGindex returns the index of chunk index of a progressive
// list, bitlist or container, measured from the TYPE root, as
// progressive_chunk_gindex of the specification does: chunk 0 is 4, chunks 1
// to 4 are 40 to 43, chunks 5 to 20 are 352 to 367, chunk 21 is 2944. The
// first turn below the root is the left step past the mix-in word into the
// data tree at 2; every progressive shape has such a word, so this is the
// index a caller composes with, and the one the light-client constants are
// made of. It never depends on the length, which is the property EIP-7916
// exists for, so the function is total over the chunks a value can hold.
//
// Measured from the data tree's own root the chunk sits at
// progressiveLeafGindex, the form Nimbus, Lodestar and Teku compute and then
// compose under 2. ProveProgressive reports that form, since the root it
// proves against is the data tree; MixLength and MixActiveFields lift it here.
func ProgressiveChunkGindex(index uint64) uint64 {
	return ConcatGindex(2, progressiveLeafGindex(index))
}

// progressiveLevel locates a chunk on the spine of merkleize_progressive: the
// level k whose subtree holds it, that subtree's first chunk and its capacity
// 4^k. It is the inverse of progressiveBase.
func progressiveLevel(index uint64) (k int, base, size uint64) {
	if index >= maxProgressiveChunk {
		panic(fmt.Sprintf("ssz: progressive chunk %d beyond the addressable range", index))
	}
	size = 1
	for index >= base+size {
		base += size
		size *= 4
		k++
	}
	return k, base, size
}

// progressiveLeafGindex returns the index of a chunk measured from the root
// of its merkleize_progressive tree: k right turns down the spine, one left
// turn into the subtree of level k, then 2k levels down to the leaf. Chunk 0
// is 2, chunk 1 is 24, chunk 5 is 224.
func progressiveLeafGindex(index uint64) uint64 {
	k, base, size := progressiveLevel(index)
	spine := uint64(1)<<(k+1) - 1 // the leading 1 followed by k right turns
	return (spine<<1)*size + (index - base)
}
