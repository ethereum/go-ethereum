// Copyright 2025 go-ethereum Authors
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

import "github.com/ethereum/go-ethereum/common"

// zero is the zero value for a 32-byte array.
var zero [32]byte

const (
	StemNodeWidth  = 256 // Number of children per leaf node
	StemSize       = 31  // Number of bytes to travel before reaching a group of leaves
	NodeTypeBytes  = 1   // Size of node type prefix in serialization
	HashSize       = 32  // Size of a hash in bytes
	StemBitmapSize = 32  // Size of the bitmap in a stem node (256 values = 32 bytes)
)

const (
	nodeTypeStem = iota + 1
	nodeTypeInternal
)

// DeserializeAndHash deserializes a node from bytes and returns its hash.
// This is a convenience function for external callers that need to compute
// the hash of a serialized node without maintaining a nodeStore.
func DeserializeAndHash(blob []byte, depth int) (common.Hash, error) {
	s := newNodeStore()
	ref, err := s.deserializeNode(blob, depth)
	if err != nil {
		return common.Hash{}, err
	}
	return s.computeHash(ref), nil
}
