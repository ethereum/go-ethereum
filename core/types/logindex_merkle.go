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

package types

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

// zeroHashes is the zero-subtree ladder of the index table tree: zeroHashes[0]
// is 32 zero bytes and zeroHashes[k] = sha256(zeroHashes[k-1] || zeroHashes[k-1]).
// An empty subtree of height k hashes to zeroHashes[k], so a level with an odd
// node count pads its last node with zeroHashes[level]. 64 entries cover any
// tree with a uint64 leaf count.
var zeroHashes = func() (ladder [64][32]byte) {
	for i := 1; i < len(ladder); i++ {
		ladder[i] = sha256.Sum256(append(ladder[i-1][:], ladder[i-1][:]...))
	}
	return ladder
}()

// merkleizeEntries computes the EIP-8304 index table root. Per the spec, the
// root is "calculated as the root of the List[Hash32, entry_count] SSZ list
// containing the SHA2-256 hashes of the binary encoded entries":
//
//	leaf_i     = sha256(entry_i)                       (raw 38/42/50-byte encoding)
//	tree_root  = binary merkle tree over the n leaves, right-padded with zeroHashes
//	table_root = sha256(tree_root || uint64_le(n) || 0x00*24)
//
// The count is always mixed in, including single-entry tables: at depth 0 the
// tree root is the leaf itself. An empty table has no entries to hash and no
// root; the zero hash is returned for n == 0.
func merkleizeEntries(entries []IndexEntry) common.Hash {
	n := len(entries)
	if n == 0 {
		return common.Hash{}
	}
	hasher := sha256.New()
	layer := make([][32]byte, n)
	for i, entry := range entries {
		hasher.Reset()
		hasher.Write(entry)
		hasher.Sum(layer[i][:0])
	}
	count := n
	for level := 0; count > 1; level++ {
		// Drop the stale nodes past the live layer before growing: after a
		// fold, len(layer) still covers the previous level, and an appended
		// zero-subtree sibling must land at index count, not at len(layer).
		layer = layer[:count]
		if count%2 == 1 {
			layer = append(layer, zeroHashes[level])
			count++
		}
		// Fold adjacent pairs in place: outputs land at layer[i] while every
		// input index (2i, 2i+1) is still unread at the time it is hashed.
		for i := 0; i < count/2; i++ {
			hasher.Reset()
			hasher.Write(layer[2*i][:])
			hasher.Write(layer[2*i+1][:])
			hasher.Sum(layer[i][:0])
		}
		count /= 2
	}
	var countEnc [32]byte
	binary.LittleEndian.PutUint64(countEnc[:8], uint64(n))
	hasher.Reset()
	hasher.Write(layer[0][:])
	hasher.Write(countEnc[:])
	var root common.Hash
	hasher.Sum(root[:0])
	return root
}
