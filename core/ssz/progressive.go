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

// progressiveBase returns the first chunk index and the leaf capacity of
// spine subtree i: base(0) = 0 and base(i+1) = base(i) + 4^i, so the bases
// run 0, 1, 5, 21, 85, 341, ... and the capacities 1, 4, 16, 64, 256, ...
func progressiveBase(i int) (base, size uint64) {
	size = 1
	for ; i > 0; i-- {
		base += size
		size *= 4
	}
	return base, size
}

// MerkleizeProgressive computes merkleize_progressive(chunks) of EIP-7916,
// the tree shape behind ProgressiveList, ProgressiveByteList and
// ProgressiveBitlist. The EIP defines it recursively:
//
//	merkleize_progressive(chunks, num_leaves=1):
//	    if len(chunks) == 0: return Bytes32()
//	    a = merkleize(chunks[:num_leaves], num_leaves)
//	    b = merkleize_progressive(chunks[num_leaves:], num_leaves * 4)
//	    return hash(a, b)
//
// The result is a chain of classic Merkle subtrees of 1, 4, 16, 64, ...
// leaves hung off a spine that descends to the right and ends in a zero
// chunk:
//
//	     root
//	     /  \
//	sub(1)   •            chunks[0:1]
//	        /  \
//	   sub(4)   •         chunks[1:5]
//	           /  \
//	     sub(16)   0      chunks[5:21], then the terminator
//
// A chunk's position depends only on its index, never on the list length,
// which is what keeps generalized indices stable as a list grows.
//
// The orientation is consensus-critical: the subtree is the left hash input
// and the spine continuation the right. EIPs PR 11032 and consensus-specs
// PR 4813, both merged 2026-01-15, flipped the shape to this form. Text,
// diagrams and test vectors older than that have it mirrored and give
// plausible, stable, wrong roots. TestProgressiveOrientation pins the current
// form to published roots.
//
// The recursion is replaced by an equivalent fold from the deepest subtree
// back to the root; the recursion itself is kept in the tests as
// naiveMerkleizeProgressive and the two are fuzzed against each other. Each
// subtree is merkleized straight out of the caller's slice, which Merkleize
// never writes to.
func MerkleizeProgressive(chunks [][32]byte) [32]byte {
	var acc [32]byte // the zero terminator of the spine
	if len(chunks) == 0 {
		return acc
	}
	// Count the subtrees the chunks occupy. The capacity quadruples per
	// subtree, so a chunk count below the SSZ size envelope needs at most
	// seventeen of them and the arithmetic cannot overflow.
	n := 0
	for base, _ := progressiveBase(n); base < uint64(len(chunks)); base, _ = progressiveBase(n) {
		n++
	}
	for i := n - 1; i >= 0; i-- {
		base, size := progressiveBase(i)
		end := min(base+size, uint64(len(chunks)))
		acc = hashPair(Merkleize(chunks[base:end], size), acc)
	}
	return acc
}

// HashTreeRootProgressiveBytes computes the root of a ProgressiveByteList,
// the EIP-7916 alias of ProgressiveList[byte]:
// mix_in_length(merkleize_progressive(pack(data)), len(data)). Lists of
// wider basic types are rooted the same way from their serialization, mixing
// in the element count rather than the byte count.
func HashTreeRootProgressiveBytes(data []byte) [32]byte {
	return MixInLength(MerkleizeProgressive(Pack(data)), uint64(len(data)))
}

// HashTreeRootProgressiveRoots computes the root of a ProgressiveList of
// composite elements from their hash tree roots, one chunk per element:
// mix_in_length(merkleize_progressive(roots), len(roots)).
func HashTreeRootProgressiveRoots(roots [][32]byte) [32]byte {
	return MixInLength(MerkleizeProgressive(roots), uint64(len(roots)))
}

// HashTreeRootProgressiveBitlist computes the root of a ProgressiveBitlist
// from its packed data bits, delimiter excluded as DecodeBitlist returns
// them, and its bit length:
// mix_in_length(merkleize_progressive(pack_bits(bits)), nbits).
func HashTreeRootProgressiveBitlist(bits []byte, nbits uint64) [32]byte {
	return MixInLength(MerkleizeProgressive(PackBits(bits)), nbits)
}
