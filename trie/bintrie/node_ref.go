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

// NodeKind identifies the type of a trie node stored in a NodeRef.
type NodeKind uint8

const (
	KindEmpty    NodeKind = 0 // no node
	KindInternal NodeKind = 1 // internal binary branching node
	KindStem     NodeKind = 2 // leaf group containing up to 256 values
	KindHashed   NodeKind = 3 // unresolved node (hash only)
)

// NodeRef is a compact, GC-invisible reference to a node in a NodeStore.
// It packs a 2-bit type tag (bits 31-30) and a 30-bit index (bits 29-0)
// into a single uint32. Because NodeRef contains no Go pointers, slices
// of structs containing NodeRef fields are allocated in noscan spans —
// the garbage collector never examines them.
type NodeRef uint32

const (
	kindShift uint32 = 30
	indexMask uint32 = (1 << kindShift) - 1

	// EmptyRef is the zero-value NodeRef, representing an empty node.
	EmptyRef NodeRef = 0
)

// MakeRef creates a NodeRef from a kind and pool index.
func MakeRef(kind NodeKind, idx uint32) NodeRef {
	return NodeRef(uint32(kind)<<kindShift | (idx & indexMask))
}

// Kind returns the node type tag.
func (r NodeRef) Kind() NodeKind { return NodeKind(uint32(r) >> kindShift) }

// Index returns the pool index within the node's typed pool.
func (r NodeRef) Index() uint32 { return uint32(r) & indexMask }

// IsEmpty returns true if this ref represents an empty node.
func (r NodeRef) IsEmpty() bool { return r == EmptyRef }
