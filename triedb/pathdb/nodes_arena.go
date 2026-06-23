// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// arenaChunkSize is the size of each arena block. The arena grows one block at a
// time, so a store never has to copy previously stored bytes and never triggers
// a single large allocation/zeroing spike.
const arenaChunkSize = 4 * 1024 * 1024

// nodeRef is a pointer-free reference into the chunked arena of an arenaNodes.
// The referenced slot stores hash(32 bytes) || blob within a single chunk. A
// zero size denotes a deleted node, which occupies no arena bytes.
type nodeRef struct {
	chunk uint32 // index of the arena chunk holding the entry
	off   uint32 // start offset of the (hash||blob) entry within the chunk
	size  uint32 // blob length; 0 means the node is deleted
}

// arenaNodes is an arena-backed, pointer-free store of trie nodes used by the
// write buffer.
//
// Node blobs are appended to a single contiguous byte arena and the index maps
// hold only offsets (nodeRef). Because the (large, long-lived) index maps carry
// no pointers, inserts incur no GC write barrier and the garbage collector never
// scans into the node data; the arena itself is a single []byte scanned as one
// object. This keeps the buffer cheap to merge into and light on the GC while it
// accumulates across many blocks — the exact opposite of a map[string]*Node,
// whose millions of pointers dominate both write-barrier traffic and GC marking.
//
// It is the buffer-only counterpart of nodeSet (which the diff layers keep using
// in pointer form). The two share the journal wire format (journalNodes) so the
// on-disk layout is unchanged.
type arenaNodes struct {
	size   uint64   // aggregated live database size of the trie nodes (matches nodeSet.size)
	chunks [][]byte // arena blocks; each entry is hash(32B) || blob, never split across chunks
	alloc  uint64   // total bytes appended across all chunks (live + dead)
	dead   uint64   // arena bytes no longer referenced (overwritten/reverted), reclaimed on compaction

	accountNodes map[string]nodeRef                 // account trie nodes, keyed by path
	storageNodes map[common.Hash]map[string]nodeRef // storage trie nodes, keyed by owner and path
}

// newArenaNodes constructs an empty arena-backed node store. The arena grows
// on demand, one chunk at a time, so no capacity hint is required.
func newArenaNodes() *arenaNodes {
	return &arenaNodes{
		accountNodes: make(map[string]nodeRef),
		storageNodes: make(map[common.Hash]map[string]nodeRef),
	}
}

// reserve returns the index of a chunk with room for n bytes, allocating a new
// one if the current tail chunk is too full. Entries are never split across
// chunks; an oversized entry gets its own dedicated chunk.
func (s *arenaNodes) reserve(n int) int {
	if ci := len(s.chunks) - 1; ci >= 0 && cap(s.chunks[ci])-len(s.chunks[ci]) >= n {
		return ci
	}
	size := arenaChunkSize
	if n > size {
		size = n
	}
	s.chunks = append(s.chunks, make([]byte, 0, size))
	return len(s.chunks) - 1
}

// store appends the given node into the arena and returns its reference. A node
// with an empty blob is treated as deleted and consumes no arena space.
func (s *arenaNodes) store(hash common.Hash, blob []byte) nodeRef {
	if len(blob) == 0 {
		return nodeRef{}
	}
	n := common.HashLength + len(blob)
	ci := s.reserve(n)
	off := uint32(len(s.chunks[ci]))
	s.chunks[ci] = append(s.chunks[ci], hash[:]...)
	s.chunks[ci] = append(s.chunks[ci], blob...)
	s.alloc += uint64(n)
	return nodeRef{chunk: uint32(ci), off: off, size: uint32(len(blob))}
}

// blob returns the node blob referenced by ref, aliasing the arena (do not
// modify). It returns nil for a deleted node.
func (s *arenaNodes) blob(ref nodeRef) []byte {
	if ref.size == 0 {
		return nil
	}
	start := ref.off + common.HashLength
	return s.chunks[ref.chunk][start : start+ref.size]
}

// hash returns the node hash referenced by ref (zero for a deleted node).
func (s *arenaNodes) hash(ref nodeRef) common.Hash {
	if ref.size == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(s.chunks[ref.chunk][ref.off : ref.off+common.HashLength])
}

// updateSize adjusts the tracked live size by the given delta.
func (s *arenaNodes) updateSize(delta int64) {
	size := int64(s.size) + delta
	if size >= 0 {
		s.size = uint64(size)
		return
	}
	log.Error("Nodeset size underflow", "prev", common.StorageSize(s.size), "delta", common.StorageSize(delta))
	s.size = 0
}

// computeSize recomputes the live size from scratch (used after decode).
func (s *arenaNodes) computeSize() {
	var size uint64
	for path, ref := range s.accountNodes {
		size += uint64(int(ref.size) + len(path))
	}
	for _, subset := range s.storageNodes {
		for path, ref := range subset {
			size += uint64(common.HashLength + int(ref.size) + len(path))
		}
	}
	s.size = size
}

// node retrieves the trie node blob and hash for the given owner and path.
func (s *arenaNodes) node(owner common.Hash, path []byte) ([]byte, common.Hash, bool) {
	if owner == (common.Hash{}) {
		ref, ok := s.accountNodes[string(path)]
		if !ok {
			return nil, common.Hash{}, false
		}
		return s.blob(ref), s.hash(ref), true
	}
	subset, ok := s.storageNodes[owner]
	if !ok {
		return nil, common.Hash{}, false
	}
	ref, ok := subset[string(path)]
	if !ok {
		return nil, common.Hash{}, false
	}
	return s.blob(ref), s.hash(ref), true
}

// merge integrates the provided (pointer-form) node set into the arena store,
// copying each blob into the arena. The provided set is left unchanged. It
// returns a breakdown of the merge for per-block diagnostics.
func (s *arenaNodes) merge(src *nodeSet) nodeMergeStats {
	var (
		delta     int64
		overwrite counter
	)
	// Account trie nodes (single flat index).
	accountStart := time.Now()
	for path, n := range src.accountNodes {
		if ref, exist := s.accountNodes[path]; !exist {
			delta += int64(len(n.Blob) + len(path))
		} else {
			delta += int64(len(n.Blob)) - int64(ref.size)
			if ref.size != 0 {
				s.dead += uint64(common.HashLength) + uint64(ref.size)
				overwrite.add(int(ref.size) + len(path))
			}
		}
		s.accountNodes[path] = s.store(n.Hash, n.Blob)
	}
	accountDur := time.Since(accountStart)

	// Storage trie nodes (per-owner index).
	var storageCount int
	storageStart := time.Now()
	for owner, subset := range src.storageNodes {
		current, exist := s.storageNodes[owner]
		if !exist {
			current = make(map[string]nodeRef, len(subset))
			s.storageNodes[owner] = current
		}
		for path, n := range subset {
			storageCount++
			if ref, ok := current[path]; !ok {
				delta += int64(common.HashLength + len(n.Blob) + len(path))
			} else {
				delta += int64(len(n.Blob)) - int64(ref.size)
				if ref.size != 0 {
					s.dead += uint64(common.HashLength) + uint64(ref.size)
					overwrite.add(common.HashLength + int(ref.size) + len(path))
				}
			}
			current[path] = s.store(n.Hash, n.Blob)
		}
	}
	storageDur := time.Since(storageStart)

	overwrite.report(gcTrieNodeMeter, gcTrieNodeBytesMeter)
	s.updateSize(delta)
	s.maybeCompact()

	return nodeMergeStats{
		accountDur:   accountDur,
		storageDur:   storageDur,
		owners:       len(src.storageNodes),
		accountNodes: len(src.accountNodes),
		storageNodes: storageCount,
	}
}

// revertTo merges the provided trie nodes into the store, reversing the changes
// made by the most recent state transition. See nodeSet.revertTo.
func (s *arenaNodes) revertTo(db ethdb.KeyValueReader, nodes map[common.Hash]map[string]*trienode.Node) {
	var delta int64
	for owner, subset := range nodes {
		if owner == (common.Hash{}) {
			for path, n := range subset {
				ref, ok := s.accountNodes[path]
				if !ok {
					blob := rawdb.ReadAccountTrieNode(db, []byte(path))
					if bytes.Equal(blob, n.Blob) {
						continue
					}
					panic(fmt.Sprintf("non-existent account node (%v) blob: %v", path, crypto.Keccak256Hash(n.Blob).Hex()))
				}
				if ref.size != 0 {
					s.dead += uint64(common.HashLength) + uint64(ref.size)
				}
				delta += int64(len(n.Blob)) - int64(ref.size)
				s.accountNodes[path] = s.store(n.Hash, n.Blob)
			}
		} else {
			current, ok := s.storageNodes[owner]
			if !ok {
				panic(fmt.Sprintf("non-existent subset (%x)", owner))
			}
			for path, n := range subset {
				ref, ok := current[path]
				if !ok {
					blob := rawdb.ReadStorageTrieNode(db, owner, []byte(path))
					if bytes.Equal(blob, n.Blob) {
						continue
					}
					panic(fmt.Sprintf("non-existent storage node (%x %v) blob: %v", owner, path, crypto.Keccak256Hash(n.Blob).Hex()))
				}
				if ref.size != 0 {
					s.dead += uint64(common.HashLength) + uint64(ref.size)
				}
				delta += int64(len(n.Blob)) - int64(ref.size)
				current[path] = s.store(n.Hash, n.Blob)
			}
		}
	}
	s.updateSize(delta)
	s.maybeCompact()
}

// maybeCompact rebuilds the arena to drop dead bytes once they dominate it,
// bounding the buffer's physical memory under heavy node overwriting.
func (s *arenaNodes) maybeCompact() {
	if s.alloc < 8*1024*1024 || s.dead*2 <= s.alloc {
		return
	}
	old := s.chunks
	s.chunks = nil
	s.alloc = 0
	s.dead = 0
	relocate := func(m map[string]nodeRef) {
		for path, ref := range m {
			if ref.size == 0 {
				continue // deleted: no arena bytes
			}
			n := common.HashLength + int(ref.size)
			ci := s.reserve(n)
			off := uint32(len(s.chunks[ci]))
			s.chunks[ci] = append(s.chunks[ci], old[ref.chunk][ref.off:ref.off+uint32(n)]...)
			s.alloc += uint64(n)
			m[path] = nodeRef{chunk: uint32(ci), off: off, size: ref.size}
		}
	}
	relocate(s.accountNodes)
	for _, subset := range s.storageNodes {
		relocate(subset)
	}
}

// write flushes the held nodes into the provided database batch.
func (s *arenaNodes) write(batch ethdb.Batch, clean *fastcache.Cache) int {
	var total int
	for path, ref := range s.accountNodes {
		if ref.size == 0 {
			rawdb.DeleteAccountTrieNode(batch, []byte(path))
			if clean != nil {
				clean.Del(nodeCacheKey(common.Hash{}, []byte(path)))
			}
		} else {
			blob := s.blob(ref)
			rawdb.WriteAccountTrieNode(batch, []byte(path), blob)
			if clean != nil {
				clean.Set(nodeCacheKey(common.Hash{}, []byte(path)), blob)
			}
		}
		total++
	}
	for owner, subset := range s.storageNodes {
		for path, ref := range subset {
			if ref.size == 0 {
				rawdb.DeleteStorageTrieNode(batch, owner, []byte(path))
				if clean != nil {
					clean.Del(nodeCacheKey(owner, []byte(path)))
				}
			} else {
				blob := s.blob(ref)
				rawdb.WriteStorageTrieNode(batch, owner, []byte(path), blob)
				if clean != nil {
					clean.Set(nodeCacheKey(owner, []byte(path)), blob)
				}
			}
			total++
		}
	}
	return total
}

// reset clears all cached trie node data.
func (s *arenaNodes) reset() {
	s.accountNodes = make(map[string]nodeRef)
	s.storageNodes = make(map[common.Hash]map[string]nodeRef)
	s.chunks = nil
	s.alloc = 0
	s.dead = 0
	s.size = 0
}

// dbsize returns the approximate size of the resulting database write.
func (s *arenaNodes) dbsize() int {
	m := len(s.accountNodes) * len(rawdb.TrieNodeAccountPrefix)
	for _, subset := range s.storageNodes {
		m += len(subset) * len(rawdb.TrieNodeStoragePrefix)
	}
	return m + int(s.size)
}

// encode serializes the held trie nodes into the provided writer using the same
// wire format as nodeSet.encode.
func (s *arenaNodes) encode(w io.Writer) error {
	nodes := make([]journalNodes, 0, len(s.storageNodes)+1)
	if len(s.accountNodes) > 0 {
		entry := journalNodes{Owner: common.Hash{}}
		for path, ref := range s.accountNodes {
			entry.Nodes = append(entry.Nodes, journalNode{Path: []byte(path), Blob: s.blob(ref)})
		}
		nodes = append(nodes, entry)
	}
	for owner, subset := range s.storageNodes {
		entry := journalNodes{Owner: owner}
		for path, ref := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{Path: []byte(path), Blob: s.blob(ref)})
		}
		nodes = append(nodes, entry)
	}
	return rlp.Encode(w, nodes)
}

// decode deserializes node content from the rlp stream into the arena store.
func (s *arenaNodes) decode(r *rlp.Stream) error {
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return fmt.Errorf("load nodes: %v", err)
	}
	s.accountNodes = make(map[string]nodeRef)
	s.storageNodes = make(map[common.Hash]map[string]nodeRef)
	s.chunks = nil
	s.alloc = 0
	s.dead = 0

	for _, entry := range encoded {
		if entry.Owner == (common.Hash{}) {
			for _, n := range entry.Nodes {
				s.accountNodes[string(n.Path)] = s.storeBlob(n.Blob)
			}
		} else {
			subset := make(map[string]nodeRef)
			for _, n := range entry.Nodes {
				subset[string(n.Path)] = s.storeBlob(n.Blob)
			}
			s.storageNodes[entry.Owner] = subset
		}
	}
	s.computeSize()
	return nil
}

// storeBlob appends a blob (computing its hash) into the arena. An empty blob is
// treated as a deleted node.
func (s *arenaNodes) storeBlob(blob []byte) nodeRef {
	if len(blob) == 0 {
		return nodeRef{}
	}
	return s.store(crypto.Keccak256Hash(blob), blob)
}
