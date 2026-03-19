// Copyright 2026 go-ethereum Authors
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

// NodeRef is a compact 32-bit reference to a node in the arena.
// Layout: [kind:2 bits][index:30 bits]
type NodeRef uint32

// NodeKind identifies the type of node a NodeRef points to.
type NodeKind uint8

const (
	KindEmpty    NodeKind = 0 // Empty/nil node
	KindInternal NodeKind = 1 // InternalNode
	KindStem     NodeKind = 2 // StemNode
	KindHashed   NodeKind = 3 // HashedNode
)

const (
	kindShift = 30
	kindMask  = 0x3 // two bits
	indexMask = (1 << kindShift) - 1
)

// EmptyRef is the zero-value NodeRef representing an empty node.
var EmptyRef = NodeRef(0) // kind=0 (KindEmpty), index=0

// makeRef creates a NodeRef from a kind and an index.
func makeRef(kind NodeKind, index uint32) NodeRef {
	return NodeRef(uint32(kind)<<kindShift | (index & indexMask))
}

// Kind returns the node kind encoded in this reference.
func (r NodeRef) Kind() NodeKind {
	return NodeKind(uint32(r) >> kindShift & kindMask)
}

// Index returns the pool index encoded in this reference.
func (r NodeRef) Index() uint32 {
	return uint32(r) & indexMask
}

// IsEmpty returns true if the reference points to an empty node.
func (r NodeRef) IsEmpty() bool {
	return r.Kind() == KindEmpty
}
