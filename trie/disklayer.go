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
	"sync"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// diskLayer is a low level persistent snapshot built on top of a key-value store.
type diskLayer struct {
	// Immutables
	root common.Hash // Root hash of the base snapshot
	rid  uint64      // Corresponding reverse diff id

	diskdb ethdb.Database   // Key-value store containing the base snapshot
	cache  *fastcache.Cache // Cache to avoid hitting the disk for direct access
	stale  bool             // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex     // Lock used to prevent stale flag
}

// newDiskLayer creates a new disk layer based on the passing arguments.
func newDiskLayer(root common.Hash, rid uint64, cache *fastcache.Cache, diskdb ethdb.Database) *diskLayer {
	dl := &diskLayer{
		diskdb: diskdb,
		cache:  cache,
		root:   root,
		rid:    rid,
	}
	return dl
}

// Root returns root hash of corresponding state.
func (dl *diskLayer) Root() common.Hash {
	return dl.root
}

// Parent always returns nil as there's no layer below the disk.
func (dl *diskLayer) Parent() snapshot {
	return nil
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diskLayer) Stale() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
}

// ID returns the id of associated reverse diff.
func (dl *diskLayer) ID() uint64 {
	return dl.rid
}

// MarkStale sets the stale flag as true.
func (dl *diskLayer) MarkStale() {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.stale == true {
		panic("triedb disk layer is stale") // we've committed into the same base from two children, boom
	}
	dl.stale = true
}

// Node retrieves the trie node associated with a particular key.
func (dl *diskLayer) Node(storage []byte, hash common.Hash) (node, error) {
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If we're in the disk layer, all diff layers missed
	triedbDirtyMissMeter.Mark(1)

	// Try to retrieve the trie node from the memory cache
	ikey := EncodeInternalKey(storage, hash)
	if dl.cache != nil {
		if blob, found := dl.cache.HasGet(nil, ikey); found && len(blob) > 0 {
			triedbCleanHitMeter.Mark(1)
			triedbCleanReadMeter.Mark(int64(len(blob)))
			return mustDecodeNode(hash.Bytes(), blob), nil
		}
		triedbCleanMissMeter.Mark(1)
	}
	blob, nodeHash := rawdb.ReadTrieNode(dl.diskdb, storage)
	if len(blob) == 0 || nodeHash != hash {
		blob = rawdb.ReadArchiveTrieNode(dl.diskdb, hash)
		if len(blob) != 0 {
			triedbFallbackHitMeter.Mark(1)
			triedbFallbackReadMeter.Mark(int64(len(blob)))
		}
	}
	if dl.cache != nil && len(blob) > 0 {
		dl.cache.Set(ikey, blob)
		triedbCleanWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) > 0 {
		return mustDecodeNode(hash.Bytes(), blob), nil
	}
	return nil, nil
}

// NodeBlob retrieves the trie node blob associated with a particular key.
func (dl *diskLayer) NodeBlob(storage []byte, hash common.Hash) ([]byte, error) {
	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	// If we're in the disk layer, all diff layers missed
	triedbDirtyMissMeter.Mark(1)

	// Try to retrieve the trie node from the memory cache
	ikey := EncodeInternalKey(storage, hash)
	if dl.cache != nil {
		if blob, found := dl.cache.HasGet(nil, ikey); found && len(blob) > 0 {
			triedbCleanHitMeter.Mark(1)
			triedbCleanReadMeter.Mark(int64(len(blob)))
			return blob, nil
		}
		triedbCleanMissMeter.Mark(1)
	}
	blob, nodeHash := rawdb.ReadTrieNode(dl.diskdb, storage)
	if len(blob) == 0 || nodeHash != hash {
		blob = rawdb.ReadArchiveTrieNode(dl.diskdb, hash)
		if len(blob) != 0 {
			triedbFallbackHitMeter.Mark(1)
			triedbFallbackReadMeter.Mark(int64(len(blob)))
		}
	}
	if dl.cache != nil && len(blob) > 0 {
		dl.cache.Set(ikey, blob)
		triedbCleanWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) > 0 {
		return blob, nil
	}
	return nil, nil
}

func (dl *diskLayer) Update(blockHash common.Hash, id uint64, nodes map[string]*cachedNode) *diffLayer {
	return newDiffLayer(dl, blockHash, id, nodes)
}
