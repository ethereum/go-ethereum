// Copyright 2021 The go-ethereum Authors
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

package trie

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
)

var (
	// cacheSizeLimit is the maximum size of the disk cache that aggregates
	// the writes from above until it's flushed into the disk. Do not
	// increase the cache size arbitrarily, otherwise the system pause
	// time will increase when the database writes happen.
	cacheSizeLimit = uint64(256 * 1024 * 1024)
)

// diskcache is a collection of dirty trie nodes to aggregate the disk
// write. It can act as an additional cache to avoid hitting disk too much.
// diskcache is not thread-safe, callers must manage concurrency issues
// by themselves.
type diskcache struct {
	seq   uint64                 // The number of state transitions contained
	nodes map[string]*cachedNode // The dirty node set, mapped by storage key
	size  uint64                 // The approximate size of cached nodes
}

// newDiskcache initializes the dirty node cache with the given node set.
func newDiskcache(nodes map[string]*cachedNode, seq uint64) *diskcache {
	if nodes == nil {
		nodes = make(map[string]*cachedNode)
	}
	var size uint64
	for key, node := range nodes {
		size += uint64(node.memorySize(len(key)))
	}
	return &diskcache{seq: seq, nodes: nodes, size: size}
}

// node retrieves the node with given storage key and hash.
func (cache *diskcache) node(storage []byte, hash common.Hash) (*cachedNode, error) {
	n, ok := cache.nodes[string(storage)]
	if ok {
		if n.hash != hash {
			owner, path := DecodeStorageKey(storage)
			return nil, fmt.Errorf("%w %x(%x %v)", errUnexpectedNode, hash, owner, path)
		}
		triedbDirtyHitMeter.Mark(1)
		triedbDirtyNodeHitDepthHist.Update(int64(128))
		triedbDirtyReadMeter.Mark(int64(n.size))
		return n, nil
	}
	return nil, nil
}

// commit merges the given dirty nodes into the cache and bumps seq as well
// to complete the state transition.
func (cache *diskcache) commit(nodes map[string]*cachedNode) *diskcache {
	cache.seq += 1

	var size int64
	for storage, node := range nodes {
		if orig, exist := cache.nodes[storage]; exist {
			size += int64(node.size) - int64(orig.size)
		} else {
			size += int64(node.memorySize(len(storage)))
		}
		cache.nodes[storage] = node
	}
	if final := int64(cache.size) + size; final < 0 {
		log.Error("Negative disk cache size", "previous", common.StorageSize(cache.size), "diff", common.StorageSize(size))
		cache.size = 0
	} else {
		cache.size = uint64(final)
	}
	return cache
}

// revert applies the reverse diff to the local dirty node set.
func (cache *diskcache) revert(diff *reverseDiff) error {
	if cache.seq == 0 {
		return errStateUnrecoverable
	}
	cache.seq -= 1

	// If all the embedded state transitions are reverted,
	// reset the cache entirely.
	if cache.seq == 0 {
		cache.reset()
		return nil
	}
	for _, state := range diff.States {
		_, ok := cache.nodes[string(state.Key)]
		if !ok {
			// TODO it should never happen, perhaps panic here.
			owner, path := DecodeStorageKey(state.Key)
			return fmt.Errorf("non-existent node (%x %v)", owner, path)
		}
		if len(state.Val) == 0 {
			cache.nodes[string(state.Key)] = &cachedNode{
				node: nil,
				size: 0,
				hash: common.Hash{},
			}
		} else {
			cache.nodes[string(state.Key)] = &cachedNode{
				node: rawNode(state.Val),
				size: uint16(len(state.Val)),
				hash: crypto.Keccak256Hash(state.Val),
			}
		}
	}
	return nil
}

// reset cleans up the disk cache.
func (cache *diskcache) reset() {
	cache.seq = 0
	cache.nodes = make(map[string]*cachedNode)
	cache.size = 0
}

// empty returns an indicator if diskcache contains any state transition inside.
func (cache *diskcache) empty() bool {
	return cache.seq == 0
}

// forEach iterates all the cached nodes and applies the given callback on them
func (cache *diskcache) forEach(callback func(key string, node *cachedNode)) {
	for storage, n := range cache.nodes {
		callback(storage, n)
	}
}

// mayFlush persists the in-memory dirty trie node into the disk if the predefined
// memory threshold is reached. Note, all data must be written to disk atomically.
// This function should never be called simultaneously with other map accessors.
func (cache *diskcache) mayFlush(db ethdb.KeyValueStore, clean *fastcache.Cache, diffid uint64, force bool) error {
	if cache.size <= cacheSizeLimit && !force {
		return nil
	}
	// Ensure the given reverse diff identifier is aligned
	// with the internal counter.
	head := rawdb.ReadReverseDiffHead(db)
	if head+cache.seq != diffid {
		return errors.New("invalid reverse diff id")
	}
	var (
		start = time.Now()
		batch = db.NewBatchWithSize(int(cacheSizeLimit))
	)
	for storage, n := range cache.nodes {
		if n.node == nil {
			rawdb.DeleteTrieNode(batch, []byte(storage))
			continue
		}
		blob := n.rlp()
		rawdb.WriteTrieNode(batch, []byte(storage), blob)
		if clean != nil {
			clean.Set(n.hash.Bytes(), blob)
		}
	}
	rawdb.WriteReverseDiffHead(batch, diffid)

	if err := batch.Write(); err != nil {
		return err
	}
	cache.reset()

	triedbCommitSizeMeter.Mark(int64(batch.ValueSize()))
	triedbCommitNodesMeter.Mark(int64(len(cache.nodes)))
	triedbCommitTimeTimer.UpdateSince(start)

	log.Debug("Persisted uncommitted nodes",
		"nodes", len(cache.nodes),
		"size", common.StorageSize(batch.ValueSize()),
		"elapsed", common.PrettyDuration(time.Since(start)),
	)
	return nil
}
