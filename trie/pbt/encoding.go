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

package pbt

import (
	"encoding/binary"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/blake3"
)

// serializeNode returns the database record of n at bit position pos. Leaf and
// branch records are their hash preimages.
func serializeNode(n node, pos int) []byte {
	switch n := n.(type) {
	case *groupNode:
		if len(n.subs) == 1 {
			blob := append(make([]byte, 0, 2+len(n.stem)+len(n.vals[0])), tagLeaf)
			return append(append(append(blob, n.stem...), n.subs[0]), n.vals[0]...)
		}
		blob := make([]byte, 4, 4+len(n.stem)+32+32*len(n.vals))
		blob[0] = tagGroup
		binary.BigEndian.PutUint16(blob[1:], uint16(pos))
		blob[3] = byte(len(n.stem))
		blob = append(blob, n.stem...)
		var bitmap [32]byte
		for _, sub := range n.subs {
			bitmap[sub>>3] |= 1 << (7 - sub&7)
		}
		blob = append(blob, bitmap[:]...)
		for _, v := range n.vals {
			blob = append(blob, v...)
		}
		return blob
	case *branchNode:
		child := pos + n.prefix.n + 1
		left, right := n.left.hashAt(child), n.right.hashAt(child)
		blob := appendBitPrefix(append(make([]byte, 0, 3+len(n.prefix.b)+64), tagBranch), n.prefix)
		return append(append(blob, left[:]...), right[:]...)
	}
	panic("pbt: unexpected node type")
}

// decodeNode parses a record written by serializeNode, taking hash as the
// node's hash without recomputing it.
func decodeNode(blob []byte, hash common.Hash) (node, error) {
	if len(blob) == 0 {
		return nil, errInvalidRecord
	}
	switch blob[0] {
	case tagLeaf:
		end := len(blob) - 33
		if end < 1 || validateStem(blob[1:end]) != nil {
			return nil, errInvalidRecord
		}
		return &groupNode{
			stem: common.CopyBytes(blob[1:end]),
			subs: []byte{blob[end]},
			vals: [][]byte{common.CopyBytes(blob[end+1:])},
			hash: hash,
		}, nil
	case tagBranch:
		prefix, size, err := decodeBitPrefix(blob[1:])
		if err != nil || len(blob) != 1+size+64 {
			return nil, errInvalidRecord
		}
		left, right := common.BytesToHash(blob[1+size:1+size+32]), common.BytesToHash(blob[1+size+32:])
		if left == (common.Hash{}) || right == (common.Hash{}) {
			return nil, errInvalidRecord
		}
		return &branchNode{prefix: prefix, left: hashedNode(left), right: hashedNode(right), hash: hash}, nil
	case tagGroup:
		if len(blob) < 4 {
			return nil, errInvalidRecord
		}
		pos, size := int(binary.BigEndian.Uint16(blob[1:])), int(blob[3])
		if len(blob) < 4+size+32 || pos > 8*size || validateStem(blob[4:4+size]) != nil {
			return nil, errInvalidRecord
		}
		bitmap, values := blob[4+size:4+size+32], blob[4+size+32:]
		count := 0
		for _, b := range bitmap {
			count += bits.OnesCount8(b)
		}
		if count < 2 || len(values) != 32*count {
			return nil, errInvalidRecord
		}
		g := &groupNode{stem: common.CopyBytes(blob[4 : 4+size]), hash: hash, hashPos: pos}
		for sub := range 256 {
			if bitmap[sub>>3]>>(7-sub&7)&1 == 1 {
				g.subs = append(g.subs, byte(sub))
				g.vals = append(g.vals, common.CopyBytes(values[:32]))
				values = values[32:]
			}
		}
		return g, nil
	}
	return nil, errInvalidRecord
}

// DeserializeAndHash returns the hash committed to by a database record.
func DeserializeAndHash(blob []byte) (common.Hash, error) {
	if len(blob) == 0 {
		return common.Hash{}, nil
	}
	if blob[0] == tagLeaf || blob[0] == tagBranch {
		return blake3.Sum256(blob), nil
	}
	n, err := decodeNode(blob, common.Hash{})
	if err != nil {
		return common.Hash{}, err
	}
	g := n.(*groupNode)
	return g.rootHash(g.hashPos), nil
}
