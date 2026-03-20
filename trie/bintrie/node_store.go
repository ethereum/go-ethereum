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

import "slices"

const (
	chunkSize = 1024 // number of nodes per chunk
	maxDepth  = 248  // maximum trie depth guard
)

// NodeStore is an arena-based pool for trie nodes. Each node type has its own
// chunked pool. NodeRef values index into these pools.
type NodeStore struct {
	internals [][]InternalNode
	stems     [][]StemNode
	hashed    [][]HashedNode

	internalCount uint32
	stemCount     uint32
	hashedCount   uint32
}

// NewNodeStore creates a new empty node store.
func NewNodeStore() *NodeStore {
	return &NodeStore{}
}

// allocInternal allocates a new InternalNode and returns a NodeRef to it.
func (s *NodeStore) allocInternal(n InternalNode) NodeRef {
	idx := s.internalCount
	if idx >= indexMask {
		panic("bintrie: internal node pool overflow")
	}
	chunk := int(idx / chunkSize)
	offset := int(idx % chunkSize)
	if chunk >= len(s.internals) {
		s.internals = append(s.internals, make([]InternalNode, chunkSize))
	}
	s.internals[chunk][offset] = n
	s.internalCount++
	return makeRef(KindInternal, idx)
}

// allocStem allocates a new StemNode and returns a NodeRef to it.
func (s *NodeStore) allocStem(n StemNode) NodeRef {
	idx := s.stemCount
	if idx >= indexMask {
		panic("bintrie: stem node pool overflow")
	}
	chunk := int(idx / chunkSize)
	offset := int(idx % chunkSize)
	if chunk >= len(s.stems) {
		s.stems = append(s.stems, make([]StemNode, chunkSize))
	}
	s.stems[chunk][offset] = n
	s.stemCount++
	return makeRef(KindStem, idx)
}

// allocHashed allocates a new HashedNode and returns a NodeRef to it.
func (s *NodeStore) allocHashed(n HashedNode) NodeRef {
	idx := s.hashedCount
	if idx >= indexMask {
		panic("bintrie: hashed node pool overflow")
	}
	chunk := int(idx / chunkSize)
	offset := int(idx % chunkSize)
	if chunk >= len(s.hashed) {
		s.hashed = append(s.hashed, make([]HashedNode, chunkSize))
	}
	s.hashed[chunk][offset] = n
	s.hashedCount++
	return makeRef(KindHashed, idx)
}

// getInternal returns a pointer to the InternalNode at the given index.
func (s *NodeStore) getInternal(idx uint32) *InternalNode {
	return &s.internals[idx/chunkSize][idx%chunkSize]
}

// getStem returns a pointer to the StemNode at the given index.
func (s *NodeStore) getStem(idx uint32) *StemNode {
	return &s.stems[idx/chunkSize][idx%chunkSize]
}

// getHashed returns a pointer to the HashedNode at the given index.
func (s *NodeStore) getHashed(idx uint32) *HashedNode {
	return &s.hashed[idx/chunkSize][idx%chunkSize]
}

// Copy creates a deep copy of the entire node store. The returned store has
// independent copies of every node, so mutations to one will not affect the other.
func (s *NodeStore) Copy() *NodeStore {
	ns := &NodeStore{
		internalCount: s.internalCount,
		stemCount:     s.stemCount,
		hashedCount:   s.hashedCount,
	}
	// Copy internal nodes
	ns.internals = make([][]InternalNode, len(s.internals))
	for i, chunk := range s.internals {
		ns.internals[i] = make([]InternalNode, len(chunk))
		copy(ns.internals[i], chunk)
	}
	// Copy stem nodes (deep copy values)
	ns.stems = make([][]StemNode, len(s.stems))
	for i, chunk := range s.stems {
		ns.stems[i] = make([]StemNode, len(chunk))
		for j := range chunk {
			sn := &chunk[j]
			dst := &ns.stems[i][j]
			dst.Stem = slices.Clone(sn.Stem)
			dst.depth = sn.depth
			dst.hash = sn.hash
			dst.mustRecompute = sn.mustRecompute
			dst.Values = make([][]byte, len(sn.Values))
			for k, v := range sn.Values {
				dst.Values[k] = slices.Clone(v)
			}
		}
	}
	// Copy hashed nodes
	ns.hashed = make([][]HashedNode, len(s.hashed))
	for i, chunk := range s.hashed {
		ns.hashed[i] = make([]HashedNode, len(chunk))
		copy(ns.hashed[i], chunk)
	}
	return ns
}
