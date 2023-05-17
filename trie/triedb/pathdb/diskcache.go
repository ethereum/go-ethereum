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
	"errors"
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

// diskcache is a collection of dirty trie nodes to aggregate the disk
// write. It can act as an additional cache to avoid hitting disk too much.
// diskcache is not thread-safe, callers must manage concurrency issues
// by themselves.
type diskcache struct {
	layers uint64                                    // The number of diff layers aggregated inside
	size   uint64                                    // The size of aggregated writes
	limit  uint64                                    // The maximum memory allowance in bytes for cache
	nodes  map[common.Hash]map[string]*trienode.Node // The dirty node set, mapped by owner and path
}

// newDiskcache initializes the dirty node cache with the given information.
func newDiskcache(limit int, nodes map[common.Hash]map[string]*trienode.Node, layers uint64) *diskcache {
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
	return &diskcache{
		layers: layers,
		nodes:  nodes,
		size:   size,
		limit:  uint64(limit),
	}
}

// node retrieves the trie node with given node info.
func (cache *diskcache) node(owner common.Hash, path []byte, hash common.Hash) (*trienode.Node, error) {
	subset, ok := cache.nodes[owner]
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
func (cache *diskcache) nodeByPath(owner common.Hash, path []byte) ([]byte, bool) {
	subset, ok := cache.nodes[owner]
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

// commit merges the dirty node belonging to the bottom-most diff layer
// into the disk cache.
func (cache *diskcache) commit(nodes map[common.Hash]map[string]*trienode.Node) *diskcache {
	var (
		delta          int64
		overwrites     int64
		overwriteSizes int64
	)
	for owner, subset := range nodes {
		current, exist := cache.nodes[owner]
		if !exist {
			cache.nodes[owner] = subset
			for path, n := range subset {
				delta += int64(len(n.Blob) + len(path))
			}
		} else {
			for path, n := range subset {
				if orig, exist := current[path]; !exist {
					delta += int64(len(n.Blob) + len(path))
				} else {
					delta += int64(len(n.Blob) - len(orig.Blob))
					overwrites += 1
					overwriteSizes += int64(len(orig.Blob) + len(path))
				}
				cache.nodes[owner][path] = n
			}
		}
	}
	cache.updateSize(delta)
	cache.layers += 1
	gcNodesMeter.Mark(overwrites)
	gcSizeMeter.Mark(overwriteSizes)
	return cache
}

// revert applies the reverse diff to the disk cache.
func (cache *diskcache) revert(h *trieHistory) error {
	if cache.layers == 0 {
		return errStateUnrecoverable
	}
	cache.layers -= 1
	if cache.layers == 0 {
		cache.reset()
		return nil
	}
	var delta int64
	for _, entry := range h.Tries {
		subset, ok := cache.nodes[entry.Owner]
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
	cache.updateSize(delta)
	return nil
}

// updateSize updates the total cache size by the given delta.
func (cache *diskcache) updateSize(delta int64) {
	size := int64(cache.size) + delta
	if size >= 0 {
		cache.size = uint64(size)
		return
	}
	s := cache.size
	cache.size = 0
	log.Error("Invalid cache size", "prev", common.StorageSize(s), "delta", common.StorageSize(delta))
}

// reset cleans up the disk cache.
func (cache *diskcache) reset() {
	cache.layers = 0
	cache.size = 0
	cache.nodes = make(map[common.Hash]map[string]*trienode.Node)
}

// empty returns an indicator if diskcache contains any state transition inside.
func (cache *diskcache) empty() bool {
	return cache.layers == 0
}

// setSize sets the cache size to the provided limit. Schedule flush operation
// if the current memory usage exceeds the new limit.
func (cache *diskcache) setSize(size int, db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64) error {
	cache.limit = uint64(size)
	return cache.mayFlush(db, clean, id, false)
}

// mayFlush persists the in-memory dirty trie node into the disk if the predefined
// memory threshold is reached. Note, all data must be written to disk atomically.
func (cache *diskcache) mayFlush(db ethdb.KeyValueStore, clean *fastcache.Cache, id uint64, force bool) error {
	if cache.size <= cache.limit && !force {
		return nil
	}
	// Ensure the given target state id is aligned with the internal counter.
	head := rawdb.ReadPersistentStateID(db)
	if head+cache.layers != id {
		return errors.New("invalid state id")
	}
	var (
		start = time.Now()
		batch = db.NewBatchWithSize(int(cache.size))
	)
	for owner, subset := range cache.nodes {
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
	commitNodesMeter.Mark(int64(len(cache.nodes)))
	commitTimeTimer.UpdateSince(start)

	log.Debug("Persisted uncommitted nodes",
		"nodes", len(cache.nodes),
		"size", common.StorageSize(batch.ValueSize()),
		"elapsed", common.PrettyDuration(time.Since(start)),
	)
	cache.reset()
	return nil
}
