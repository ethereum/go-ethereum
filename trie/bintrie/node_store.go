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
const storeChunkSize = 4096

// nodeStore is a GC-friendly arena for binary trie nodes. Nodes are packed
// into typed chunked pools so pointer-free types (InternalNode, HashedNode)
// land in noscan spans the GC skips entirely.
type nodeStore struct {
	internalChunks []*[storeChunkSize]InternalNode
	internalCount  uint32

	stemChunks []*[storeChunkSize]StemNode
	stemCount  uint32

	hashedChunks []*[storeChunkSize]HashedNode
	hashedCount  uint32

	root nodeRef

	// Free list for recycling hashed-node slots after resolve. Internal and
	// stem nodes are never freed under current semantics (no delete path,
	// stem-split keeps the old stem at a deeper position), so they don't
	// have free lists.
	freeHashed []uint32
}

func newNodeStore() *nodeStore {
	return &nodeStore{root: emptyRef}
}

func (s *nodeStore) allocInternal() uint32 {
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

func (s *nodeStore) getInternal(idx uint32) *InternalNode {
	return &s.internalChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *nodeStore) newInternalRef(depth int) nodeRef {
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

func (s *nodeStore) allocStem() uint32 {
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

func (s *nodeStore) getStem(idx uint32) *StemNode {
	return &s.stemChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *nodeStore) newStemRef(stem []byte, depth int) nodeRef {
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

func (s *nodeStore) allocHashed() uint32 {
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

func (s *nodeStore) getHashed(idx uint32) *HashedNode {
	return &s.hashedChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *nodeStore) freeHashedNode(idx uint32) {
	s.freeHashed = append(s.freeHashed, idx)
}

func (s *nodeStore) newHashedRef(hash common.Hash) nodeRef {
	idx := s.allocHashed()
	*s.getHashed(idx) = HashedNode(hash)
	return makeRef(kindHashed, idx)
}

func (s *nodeStore) Copy() *nodeStore {
	ns := &nodeStore{
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
	if len(s.freeHashed) > 0 {
		ns.freeHashed = make([]uint32, len(s.freeHashed))
		copy(ns.freeHashed, s.freeHashed)
	}

	return ns
}
