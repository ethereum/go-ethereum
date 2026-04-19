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

// nodeKind identifies the type of a trie node stored in a nodeRef.
type nodeKind uint8

const (
	kindEmpty nodeKind = iota
	kindInternal
	kindStem // up to 256 values per stem
	kindHashed
)

// nodeRef is a compact, GC-invisible reference to a node in a nodeStore.
// It packs a 2-bit type tag (bits 31-30) and a 30-bit index (bits 29-0)
// into a single uint32. Because nodeRef contains no Go pointers, slices
// of structs containing nodeRef fields are allocated in noscan spans —
// the garbage collector never examines them.
type nodeRef uint32

const (
	kindShift uint32 = 30
	indexMask uint32 = (1 << kindShift) - 1

	// emptyRef represents an empty node.
	emptyRef nodeRef = 0
)

func makeRef(kind nodeKind, idx uint32) nodeRef {
	if idx > indexMask {
		panic("nodeRef index overflow")
	}
	return nodeRef(uint32(kind)<<kindShift | idx)
}

func (r nodeRef) Kind() nodeKind { return nodeKind(uint32(r) >> kindShift) }

// Index within the typed pool.
func (r nodeRef) Index() uint32 { return uint32(r) & indexMask }

func (r nodeRef) IsEmpty() bool { return r.Kind() == kindEmpty }
