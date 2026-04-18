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

import "github.com/ethereum/go-ethereum/common"

// storeChunkSize is the number of nodes per chunk in each typed pool.
// Using fixed-size array chunks ensures that pointers to nodes within
// existing chunks remain valid when new chunks are added (no reallocation
// of the backing data, only the outer pointer slice grows).
const storeChunkSize = 4096

// NodeStore is a GC-friendly arena for binary trie nodes. Nodes are packed
// into typed chunked pools so pointer-free types (InternalNode, HashedNode)
// land in noscan spans the GC skips entirely.
type NodeStore struct {
	internalChunks []*[storeChunkSize]InternalNode
	internalCount  uint32

	stemChunks []*[storeChunkSize]StemNode
	stemCount  uint32

	hashedChunks []*[storeChunkSize]HashedNode
	hashedCount  uint32

	root nodeRef

	// Free lists for recycling deleted node slots.
	freeInternals []uint32
	freeStems     []uint32
	freeHashed    []uint32
}

func NewNodeStore() *NodeStore {
	return &NodeStore{root: emptyRef}
}

func (s *NodeStore) allocInternal() uint32 {
	if n := len(s.freeInternals); n > 0 {
		idx := s.freeInternals[n-1]
		s.freeInternals = s.freeInternals[:n-1]
		*s.getInternal(idx) = InternalNode{}
		return idx
	}
	idx := s.internalCount
	chunkIdx := idx / storeChunkSize
	if uint32(len(s.internalChunks)) <= chunkIdx {
		s.internalChunks = append(s.internalChunks, new([storeChunkSize]InternalNode))
	}
	s.internalCount++
	if s.internalCount > indexMask {
		panic("internal node pool overflow")
	}
	return idx
}

func (s *NodeStore) getInternal(idx uint32) *InternalNode {
	return &s.internalChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *NodeStore) newInternalRef(depth int) nodeRef {
	if depth > 248 {
		panic("node depth exceeds maximum binary trie depth")
	}
	idx := s.allocInternal()
	n := s.getInternal(idx)
	n.depth = uint8(depth)
	n.mustRecompute = true
	n.dirty = true
	return makeRef(kindInternal, idx)
}

func (s *NodeStore) allocStem() uint32 {
	if n := len(s.freeStems); n > 0 {
		idx := s.freeStems[n-1]
		s.freeStems = s.freeStems[:n-1]
		*s.getStem(idx) = StemNode{}
		return idx
	}
	idx := s.stemCount
	chunkIdx := idx / storeChunkSize
	if uint32(len(s.stemChunks)) <= chunkIdx {
		s.stemChunks = append(s.stemChunks, new([storeChunkSize]StemNode))
	}
	s.stemCount++
	if s.stemCount > indexMask {
		panic("stem node pool overflow")
	}
	return idx
}

func (s *NodeStore) getStem(idx uint32) *StemNode {
	return &s.stemChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *NodeStore) newStemRef(stem []byte, depth int) nodeRef {
	if depth > 248 {
		panic("node depth exceeds maximum binary trie depth")
	}
	idx := s.allocStem()
	sn := s.getStem(idx)
	copy(sn.Stem[:], stem[:StemSize])
	sn.depth = uint8(depth)
	sn.mustRecompute = true
	sn.dirty = true
	return makeRef(kindStem, idx)
}

func (s *NodeStore) allocHashed() uint32 {
	if n := len(s.freeHashed); n > 0 {
		idx := s.freeHashed[n-1]
		s.freeHashed = s.freeHashed[:n-1]
		*s.getHashed(idx) = HashedNode{}
		return idx
	}
	idx := s.hashedCount
	chunkIdx := idx / storeChunkSize
	if uint32(len(s.hashedChunks)) <= chunkIdx {
		s.hashedChunks = append(s.hashedChunks, new([storeChunkSize]HashedNode))
	}
	s.hashedCount++
	if s.hashedCount > indexMask {
		panic("hashed node pool overflow")
	}
	return idx
}

func (s *NodeStore) getHashed(idx uint32) *HashedNode {
	return &s.hashedChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *NodeStore) freeHashedNode(idx uint32) {
	s.freeHashed = append(s.freeHashed, idx)
}

func (s *NodeStore) newHashedRef(hash common.Hash) nodeRef {
	idx := s.allocHashed()
	*s.getHashed(idx) = HashedNode(hash)
	return makeRef(kindHashed, idx)
}

func (s *NodeStore) Copy() *NodeStore {
	ns := &NodeStore{
		root:          s.root,
		internalCount: s.internalCount,
		stemCount:     s.stemCount,
		hashedCount:   s.hashedCount,
	}
	ns.internalChunks = make([]*[storeChunkSize]InternalNode, len(s.internalChunks))
	for i, chunk := range s.internalChunks {
		cp := *chunk
		ns.internalChunks[i] = &cp
	}
	ns.stemChunks = make([]*[storeChunkSize]StemNode, len(s.stemChunks))
	for i, chunk := range s.stemChunks {
		cp := *chunk
		ns.stemChunks[i] = &cp
	}
	// Deep-copy each stem's value slots — they may alias serialized buffers,
	// so we can't rely on the chunk-wise struct copy above.
	for i := uint32(0); i < s.stemCount; i++ {
		src := s.getStem(i)
		dst := ns.getStem(i)
		for j, v := range src.values {
			if v == nil {
				continue
			}
			cp := make([]byte, len(v))
			copy(cp, v)
			dst.values[j] = cp
		}
	}
	ns.hashedChunks = make([]*[storeChunkSize]HashedNode, len(s.hashedChunks))
	for i, chunk := range s.hashedChunks {
		cp := *chunk
		ns.hashedChunks[i] = &cp
	}
	if len(s.freeInternals) > 0 {
		ns.freeInternals = make([]uint32, len(s.freeInternals))
		copy(ns.freeInternals, s.freeInternals)
	}
	if len(s.freeStems) > 0 {
		ns.freeStems = make([]uint32, len(s.freeStems))
		copy(ns.freeStems, s.freeStems)
	}
	if len(s.freeHashed) > 0 {
		ns.freeHashed = make([]uint32, len(s.freeHashed))
		copy(ns.freeHashed, s.freeHashed)
	}

	return ns
}
