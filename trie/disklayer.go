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
	diskdb ethdb.KeyValueStore // Key-value store containing the base snapshot
	cache  *fastcache.Cache    // Cache to avoid hitting the disk for direct access

	root  common.Hash // Root hash of the base snapshot
	stale bool        // Signals that the layer became stale (state progressed)
	lock  sync.RWMutex
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

// Node retrieves the trie node associated with a particular key.
// The given key must be the internal format node key.
func (dl *diskLayer) Node(key []byte) (node, error) {
	blob, err := dl.NodeBlob(key)
	if err != nil {
		return nil, err
	}
	if len(blob) == 0 {
		return nil, nil
	}
	_, hash := DecodeInternalKey(key)
	return mustDecodeNode(hash[:], blob), nil
}

// NodeBlob retrieves the trie node blob associated with a particular key.
// The given key must be the internal format node key.
func (dl *diskLayer) NodeBlob(key []byte) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, ErrSnapshotStale
	}
	// If we're in the disk layer, all diff layers missed
	triedbDirtyMissMeter.Mark(1)

	// Try to retrieve the trie node from the memory cache
	if dl.cache != nil {
		if blob, found := dl.cache.HasGet(nil, key); found {
			triedbCleanHitMeter.Mark(1)
			triedbCleanReadMeter.Mark(int64(len(blob)))
			return blob, nil
		}
		triedbCleanMissMeter.Mark(1)
	}
	path, hash := DecodeInternalKey(key)
	blob, nodeHash := rawdb.ReadTrieNode(dl.diskdb, path)
	if len(blob) == 0 || nodeHash != hash {
		blob = rawdb.ReadArchiveTrieNode(dl.diskdb, hash)
		if len(blob) != 0 {
			triedbFallbackHitMeter.Mark(1)
			triedbFallbackReadMeter.Mark(int64(len(blob)))
		}
	}
	if dl.cache != nil {
		dl.cache.Set(key, blob)
		triedbCleanWriteMeter.Mark(int64(len(blob)))
	}
	if len(blob) > 0 {
		return blob, nil
	}
	return nil, nil
}

func (dl *diskLayer) Update(blockHash common.Hash, nodes map[string]*cachedNode) *diffLayer {
	return newDiffLayer(dl, blockHash, nodes)
}
