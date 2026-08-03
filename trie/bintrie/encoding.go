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

package bintrie

import (
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
)

// Database record formats. Leaf and branch blobs are exactly their EIP-8297
// hash preimages, so hashing a stored blob is a single BLAKE3 call. Group
// records (tag 0x02) pack a whole stem's values into one record and carry
// the ancestor-consumed bit count d, keeping every blob standalone-hashable.
//
//	leaf:   0x00 ‖ key (34/66B) ‖ value (32B)
//	branch: 0x01 ‖ u16BE prefix bit count ‖ packed prefix ‖ left (32B) ‖ right (32B)
//	group:  0x02 ‖ u16BE d ‖ stemLen (1B) ‖ stem ‖ bitmap (32B) ‖ values (32B × k), k ≥ 2

const bitmapSize = 32

// serializeNode encodes a resolved node at position pos into its database
// record.
func serializeNode(n binaryNode, pos int) []byte {
	switch n := n.(type) {
	case *groupNode:
		if len(n.subs) == 1 {
			blob := make([]byte, 0, 1+len(n.stem)+1+32)
			blob = append(blob, tagLeaf)
			blob = append(blob, n.stem...)
			blob = append(blob, n.subs[0])
			return append(blob, n.vals[0]...)
		}
		// Both fields below are written narrower than the values they carry, so
		// an out-of-range one would wrap rather than fail. The record would
		// then describe a different node, and the mismatch would only surface
		// on read-back - as a database that cannot reload its own state. Both
		// hold by construction, so a violation is a bug here, not bad input.
		if pos > maxPathBits {
			panic(fmt.Sprintf("bintrie: node position %d exceeds the deepest legal path", pos))
		}
		if len(n.stem) > 0xff {
			panic(fmt.Sprintf("bintrie: stem length %d does not fit the record", len(n.stem)))
		}
		blob := make([]byte, 0, 1+2+1+len(n.stem)+bitmapSize+32*len(n.subs))
		blob = append(blob, tagGroup, byte(pos>>8), byte(pos))
		blob = append(blob, byte(len(n.stem)))
		blob = append(blob, n.stem...)
		var bitmap [bitmapSize]byte
		for _, sub := range n.subs {
			bitmap[sub>>3] |= 1 << (7 - sub&7)
		}
		blob = append(blob, bitmap[:]...)
		for _, val := range n.vals {
			blob = append(blob, val...)
		}
		return blob
	case *branchNode:
		blob := make([]byte, 0, 1+2+len(n.prefix.b)+64)
		blob = append(blob, tagBranch)
		blob = appendBitPrefix(blob, n.prefix)
		child := pos + n.prefix.n + 1
		left, right := n.left.hashAt(child), n.right.hashAt(child)
		blob = append(blob, left[:]...)
		return append(blob, right[:]...)
	default:
		panic("bintrie: cannot serialize node type")
	}
}

// decodeNode parses a database record into a node. Branch children decode to
// hashedNodes; the node's cached hash is left unset (use decodeNodeWithHash
// when the record's hash is known from the lookup).
func decodeNode(blob []byte) (binaryNode, error) {
	if len(blob) == 0 {
		return empty{}, nil
	}
	switch blob[0] {
	case tagLeaf:
		keyLen := len(blob) - 1 - 32
		switch keyLen {
		case AccountKeyLength, StorageKeyLength:
		default:
			return nil, errInvalidSerializedLength
		}
		// The length on its own is not enough. Account and code keys share a
		// length, so it is the zone byte that binds a stem to its own, and that
		// binding is what keeps stems prefix-free - the property the insert and
		// delete walks assume when they index into a stem.
		if err := validateStem(blob[1:keyLen]); err != nil {
			return nil, err
		}
		g := &groupNode{
			stem: append([]byte{}, blob[1:keyLen]...),
			subs: []byte{blob[keyLen]},
			vals: [][]byte{append([]byte{}, blob[1+keyLen:]...)},
		}
		return g, nil
	case tagBranch:
		prefix, consumed, err := decodeBitPrefix(blob[1:])
		if err != nil {
			return nil, err
		}
		rest := blob[1+consumed:]
		if len(rest) != 64 {
			return nil, errInvalidSerializedLength
		}
		var left, right common.Hash
		copy(left[:], rest[:32])
		copy(right[:], rest[32:])
		if left == (common.Hash{}) || right == (common.Hash{}) {
			// Canonical branches always have two non-empty children.
			return nil, errInvalidNodeTag
		}
		return &branchNode{prefix: prefix, left: hashedNode(left), right: hashedNode(right)}, nil
	case tagGroup:
		if len(blob) < 1+2+1 {
			return nil, errInvalidSerializedLength
		}
		d := int(binary.BigEndian.Uint16(blob[1:3]))
		stemLen := int(blob[3])
		switch stemLen {
		case AccountKeyLength - 1, StorageKeyLength - 1:
		default:
			return nil, errInvalidSerializedLength
		}
		if d > 8*stemLen {
			return nil, errInvalidSerializedLength
		}
		if len(blob) < 4+stemLen+bitmapSize {
			return nil, errInvalidSerializedLength
		}
		// As above: bind the stem's length to its zone, not just to the set of
		// legal lengths.
		if err := validateStem(blob[4 : 4+stemLen]); err != nil {
			return nil, err
		}
		bitmap := blob[4+stemLen : 4+stemLen+bitmapSize]
		k := 0
		for _, b := range bitmap {
			k += bits.OnesCount8(b)
		}
		if k < 2 || len(blob) != 4+stemLen+bitmapSize+32*k {
			return nil, errInvalidSerializedLength
		}
		g := &groupNode{
			stem:     append([]byte{}, blob[4:4+stemLen]...),
			subs:     make([]byte, 0, k),
			vals:     make([][]byte, 0, k),
			cachedAt: d,
		}
		values := blob[4+stemLen+bitmapSize:]
		for sub := 0; sub < 256; sub++ {
			if bitmap[sub>>3]>>(7-sub&7)&1 == 1 {
				g.subs = append(g.subs, byte(sub))
				g.vals = append(g.vals, append([]byte{}, values[:32]...))
				values = values[32:]
			}
		}
		return g, nil
	default:
		return nil, errInvalidNodeTag
	}
}

// decodeNodeWithHash decodes a record whose hash is already known from the
// database lookup, priming the node's hash cache.
func decodeNodeWithHash(blob []byte, hash common.Hash) (binaryNode, error) {
	n, err := decodeNode(blob)
	if err != nil {
		return nil, err
	}
	switch n := n.(type) {
	case *branchNode:
		n.cachedHash = hash
	case *groupNode:
		n.cachedHash = hash // cachedAt was set from the record's d (leaves ignore it)
	}
	return n, nil
}

// DeserializeAndHash computes the hash a database record commits to. Leaf
// and branch blobs are their own preimages; group records fold at their
// stored depth. The empty blob is the empty tree, hashing to zero. This is
// pathdb's node-hasher hook for the binary trie.
func DeserializeAndHash(blob []byte) (common.Hash, error) {
	if len(blob) == 0 {
		return common.Hash{}, nil
	}
	switch blob[0] {
	case tagLeaf, tagBranch:
		return common.Hash(blake3.Sum256(blob)), nil
	case tagGroup:
		n, err := decodeNode(blob)
		if err != nil {
			return common.Hash{}, err
		}
		g := n.(*groupNode)
		return g.fold(g.cachedAt), nil
	default:
		return common.Hash{}, errInvalidNodeTag
	}
}
