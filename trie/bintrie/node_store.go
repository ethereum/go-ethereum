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

// storeChunkSize is the number of nodes per chunk in each typed pool.
// Using fixed-size array chunks ensures that pointers to nodes within
// existing chunks remain valid when new chunks are added (no reallocation
// of the backing data, only the outer pointer slice grows).
const storeChunkSize = 4096

// NodeStore is a GC-friendly arena for binary trie nodes.
//
// Instead of allocating each node as a separate heap object with interface
// pointers (which the GC must scan), NodeStore packs nodes into typed chunked
// pools. InternalNode and HashedNode contain ZERO Go pointers, so their pool
// backing arrays are allocated in noscan spans — the GC skips them entirely.
// StemNode has one pointer (valueData []byte) per node.
//
// For a trie with 25K InternalNodes, this reduces GC-scanned pointer-words
// from ~125K (with interface-based nodes) to ~25K (just StemNode valueData),
// an ~80% reduction. At mainnet scale (millions of nodes), this prevents
// multi-second GC pauses.
type NodeStore struct {
	// InternalNode pool — NOSCAN: InternalNode contains zero Go pointers.
	// Children are NodeRef (uint32), hash is [32]byte.
	internalChunks []*[storeChunkSize]InternalNode
	internalCount  uint32

	// StemNode pool — each StemNode has one pointer (valueData []byte).
	// Still much better than the old design where each InternalNode had
	// two BinaryNode interface pointers (4 pointer-words each).
	stemChunks []*[storeChunkSize]StemNode
	stemCount  uint32

	// HashedNode pool — NOSCAN: HashedNode is just [32]byte.
	hashedChunks []*[storeChunkSize]HashedNode
	hashedCount  uint32

	root NodeRef

	// Free lists for recycling deleted node slots.
	freeInternals []uint32
	freeStems     []uint32
	freeHashed    []uint32
}

// NewNodeStore creates a new empty NodeStore.
func NewNodeStore() *NodeStore {
	return &NodeStore{root: EmptyRef}
}

// Root returns the root NodeRef.
func (s *NodeStore) Root() NodeRef { return s.root }

// SetRoot sets the root NodeRef.
func (s *NodeStore) SetRoot(ref NodeRef) { s.root = ref }

// --- InternalNode allocation ---

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

func (s *NodeStore) freeInternal(idx uint32) {
	s.freeInternals = append(s.freeInternals, idx)
}

// newInternalRef allocates an InternalNode and returns its NodeRef.
func (s *NodeStore) newInternalRef(depth int) NodeRef {
	if depth > 248 {
		panic("node depth exceeds maximum binary trie depth")
	}
	idx := s.allocInternal()
	n := s.getInternal(idx)
	n.depth = uint8(depth)
	n.mustRecompute = true
	return MakeRef(KindInternal, idx)
}

// --- StemNode allocation ---

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
		panic("internal node pool overflow")
	}
	return idx
}

func (s *NodeStore) getStem(idx uint32) *StemNode {
	return &s.stemChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *NodeStore) freeStem(idx uint32) {
	s.freeStems = append(s.freeStems, idx)
}

// newStemRef allocates a StemNode with the given stem/depth and returns its NodeRef.
func (s *NodeStore) newStemRef(stem []byte, depth int) NodeRef {
	if depth > 248 {
		panic("node depth exceeds maximum binary trie depth")
	}
	idx := s.allocStem()
	sn := s.getStem(idx)
	copy(sn.Stem[:], stem[:StemSize])
	sn.depth = uint8(depth)
	sn.mustRecompute = true
	return MakeRef(KindStem, idx)
}

// --- HashedNode allocation ---

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
		panic("internal node pool overflow")
	}
	return idx
}

func (s *NodeStore) getHashed(idx uint32) *HashedNode {
	return &s.hashedChunks[idx/storeChunkSize][idx%storeChunkSize]
}

func (s *NodeStore) freeHashedNode(idx uint32) {
	s.freeHashed = append(s.freeHashed, idx)
}

// newHashedRef allocates a HashedNode and returns its NodeRef.
func (s *NodeStore) newHashedRef(hash common.Hash) NodeRef {
	idx := s.allocHashed()
	*s.getHashed(idx) = HashedNode{hash: hash}
	return MakeRef(KindHashed, idx)
}

// --- Utility ---

// nodeHash returns the hash for any NodeRef without computing (reads cached or stored hash).
func (s *NodeStore) nodeHash(ref NodeRef) common.Hash {
	switch ref.Kind() {
	case KindInternal:
		return s.getInternal(ref.Index()).hash
	case KindStem:
		return s.getStem(ref.Index()).hash
	case KindHashed:
		return s.getHashed(ref.Index()).hash
	case KindEmpty:
		return common.Hash{}
	default:
		return common.Hash{}
	}
}

// Copy creates a deep copy of the NodeStore and all its nodes.
func (s *NodeStore) Copy() *NodeStore {
	ns := &NodeStore{
		root:          s.root,
		internalCount: s.internalCount,
		stemCount:     s.stemCount,
		hashedCount:   s.hashedCount,
	}

	// Deep copy internal chunks
	ns.internalChunks = make([]*[storeChunkSize]InternalNode, len(s.internalChunks))
	for i, chunk := range s.internalChunks {
		cp := *chunk
		ns.internalChunks[i] = &cp
	}

	// Deep copy stem chunks (need to deep copy valueData)
	ns.stemChunks = make([]*[storeChunkSize]StemNode, len(s.stemChunks))
	for i, chunk := range s.stemChunks {
		cp := *chunk
		ns.stemChunks[i] = &cp
	}
	// Deep copy pointer fields for each active stem
	for i := uint32(0); i < s.stemCount; i++ {
		src := s.getStem(i)
		dst := ns.getStem(i)
		if len(src.valueData) > 0 {
			dst.valueData = make([]byte, len(src.valueData))
			copy(dst.valueData, src.valueData)
		}
	}

	// Deep copy hashed chunks
	ns.hashedChunks = make([]*[storeChunkSize]HashedNode, len(s.hashedChunks))
	for i, chunk := range s.hashedChunks {
		cp := *chunk
		ns.hashedChunks[i] = &cp
	}

	// Copy free lists
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
