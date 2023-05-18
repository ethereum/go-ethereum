// Copyright 2022 The go-ethereum Authors
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
	"fmt"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

var (
	// defaultCacheSize is the default memory limitation of the disk cache
	// that aggregates the writes from above until it's flushed into the disk.
	// Do not increase the cache size arbitrarily, otherwise the system pause
	// time will increase when the database writes happen.
	defaultCacheSize = 128 * 1024 * 1024
)

// nodebuffer is a collection of modified trie nodes to aggregate the disk
// write.
// The content of the nodebuffer must be checked before the disk data (since
// it basically is not-yet-written data).
// nodebuffer is not thread-safe, callers must manage concurrency issues
// by themselves.
type nodebuffer struct {
	layers uint64                                    // The number of diff layers aggregated inside
	size   uint64                                    // The size of aggregated writes
	limit  uint64                                    // The maximum memory allowance in bytes for cache
	nodes  map[common.Hash]map[string]*trienode.Node // The dirty node set, mapped by owner and path
}

// newNodeBuffer initializes the dirty node cache with the given information.
func newNodeBuffer(limit int, nodes map[common.Hash]map[string]*trienode.Node, layers uint64) *nodebuffer {
	// Don't panic for lazy callers.
	if nodes == nil {
		nodes = make(map[common.Hash]map[string]*trienode.Node)
	}
	var size uint64
	for _, subset := range nodes {
		for path, n := range subset {
			size += uint64(len(n.Blob) + len(path))
		}
	}
	return &nodebuffer{
		layers: layers,
		nodes:  nodes,
		size:   size,
		limit:  uint64(limit),
	}
}

// node retrieves the trie node with given node info.
func (b *nodebuffer) node(owner common.Hash, path []byte, hash common.Hash) (*trienode.Node, error) {
	subset, ok := b.nodes[owner]
	if !ok {
		return nil, nil
	}
	n, ok := subset[string(path)]
	if !ok {
		return nil, nil
	}
	if n.Hash != hash {
		return nil, &UnexpectedNodeError{
			typ:      "cache",
			expected: hash,
			hash:     n.Hash,
			owner:    owner,
			path:     path,
		}
	}
	return n, nil
}

// nodeByPath retrieves the trie node with given node info.
// It returns the blob and whether the path was present in the nodebuffer or not.
// If the second return-param is true then the caller should look no further.
func (b *nodebuffer) nodeByPath(owner common.Hash, path []byte) ([]byte, bool) {
	subset, ok := b.nodes[owner]
	if !ok {
		return nil, false
	}
	n, ok := subset[string(path)]
	if !ok {
		return nil, false
	}
	if n.IsDeleted() {
		return nil, true
	}
	return n.Blob, true
}

// commit merges the dirty nodes into the nodebuffer b.
// (The nodes typically belongs to the bottom-most difflayer)
// This operation takes ownership of the nodes map, and the caller must no longer
// use it.
func (b *nodebuffer) commit(nodes map[common.Hash]map[string]*trienode.Node) *nodebuffer {
	var (
		delta          int64 // size
		overwrites     int64
		overwriteSizes int64
	)
	for owner, subset := range nodes {
		current, exist := b.nodes[owner]
		if !exist {
			b.nodes[owner] = subset
			for path, n := range subset {
				delta += int64(len(n.Blob) + len(path))
			}
			continue
		}
		for path, n := range subset {
			if orig, exist := current[path]; !exist {
				delta += int64(len(n.Blob) + len(path))
			} else {
				delta += int64(len(n.Blob) - len(orig.Blob))
				overwrites += 1
				overwriteSizes += int64(len(orig.Blob) + len(path))
			}
			b.nodes[owner][path] = n
		}
	}
	b.updateSize(delta)
	b.layers += 1
	gcNodesMeter.Mark(overwrites)
	gcSizeMeter.Mark(overwriteSizes)
	return b
}

// revert applies the reverse diff to the disk cache.
func (b *nodebuffer) revert(h *trieHistory) error {
	if b.layers == 0 {
		return errStateUnrecoverable
	}
	b.layers -= 1
	if b.layers == 0 {
		b.reset()
		return nil
	}
	var delta int64
	for _, entry := range h.Tries {
		subset, ok := b.nodes[entry.Owner]
		if !ok {
			panic(fmt.Sprintf("non-existent node (%x)", entry.Owner))
		}
		for _, state := range entry.Nodes {
			cur, ok := subset[string(state.Path)]
			if !ok {
				panic(fmt.Sprintf("non-existent node (%x %v)", entry.Owner, state.Path))
			}
			if len(state.Prev) == 0 {
				subset[string(state.Path)] = trienode.New(common.Hash{}, nil)
				delta -= int64(len(cur.Blob))
			} else {
				subset[string(state.Path)] = trienode.New(crypto.Keccak256Hash(state.Prev), state.Prev)
				delta += int64(len(state.Prev)) - int64(len(cur.Blob))
			}
		}
	}
	b.updateSize(delta)
	return nil
}

// updateSize updates the total cache size by the given delta.
func (b *nodebuffer) updateSize(delta int64) {
	size := int64(b.size) + delta
	if size >= 0 {
		b.size = uint64(size)
		return
	}
	s := b.size
	b.size = 0
	log.Error("Invalid cache size", "prev", common.StorageSize(s), "delta", common.StorageSize(delta))
}

// reset cleans up the disk cache.
func (b *nodebuffer) reset() {
	b.layers = 0
	b.size = 0
	b.nodes = make(map[common.Hash]map[string]*trienode.Node)
}

// empty returns an indicator if nodebuffer contains any state transition inside.
func (b *nodebuffer) empty() bool {
	return b.layers == 0
}

// setSize sets the cache size to the provided limit, and invokes a flush operation
// if the current memory usage exceeds the new limit.
func (b *nodebuffer) setSize(size int, db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64) error {
	b.limit = uint64(size)
	return b.flush(db, clean, id, false)
}

// flush persists the in-memory dirty trie node into the disk if the predefined
// memory threshold is reached. Note, all data must be written to disk atomically.
func (b *nodebuffer) flush(db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64, force bool) error {
	if b.size <= b.limit && !force {
		return nil
	}
	// Ensure the given target state id is aligned with the internal counter.
	head := rawdb.ReadPersistentStateID(db)
	if head+b.layers != id {
		return fmt.Errorf("disk has invalid state id %d, want %d", id, head+b.layers)
	}
	var (
		start = time.Now()
		batch = db.NewBatchWithSize(int(b.size))
	)
	for owner, subset := range b.nodes {
		for path, n := range subset {
			if n.IsDeleted() {
				if owner == (common.Hash{}) {
					rawdb.DeleteAccountTrieNode(batch, []byte(path))
				} else {
					rawdb.DeleteStorageTrieNode(batch, owner, []byte(path))
				}
			} else {
				if owner == (common.Hash{}) {
					rawdb.WriteAccountTrieNode(batch, []byte(path), n.Blob)
				} else {
					rawdb.WriteStorageTrieNode(batch, owner, []byte(path), n.Blob)
				}
				if clean != nil {
					clean.Set(n.Hash.Bytes(), n.Blob)
				}
			}
		}
	}
	rawdb.WritePersistentStateID(batch, id)
	if err := batch.Write(); err != nil {
		return err
	}
	commitSizeMeter.Mark(int64(batch.ValueSize()))
	commitNodesMeter.Mark(int64(len(b.nodes)))
	commitTimeTimer.UpdateSince(start)

	log.Debug("Persisted uncommitted nodes",
		"nodes", len(b.nodes),
		"size", common.StorageSize(batch.ValueSize()),
		"elapsed", common.PrettyDuration(time.Since(start)),
	)
	b.reset()
	return nil
}
