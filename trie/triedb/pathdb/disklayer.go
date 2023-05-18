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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// diskLayer is a low level persistent layer built on top of a key-value store.
type diskLayer struct {
	root  common.Hash  // Immutable, root hash of the base layer
	id    uint64       // Immutable, corresponding state id
	db    *Database    // Path-based trie database
	dirty *nodebuffer  // Dirty node cache to aggregate writes.
	stale bool         // Signals that the layer became stale (state progressed)
	lock  sync.RWMutex // Lock used to protect stale flag
}

// newDiskLayer creates a new disk layer based on the passing arguments.
func newDiskLayer(root common.Hash, id uint64, db *Database, dirty *nodebuffer) *diskLayer {
	return &diskLayer{
		root:  root,
		id:    id,
		db:    db,
		dirty: dirty,
	}
}

// Root returns root hash of corresponding state.
func (dl *diskLayer) Root() common.Hash {
	return dl.root
}

// Parent always returns nil as there's no layer below the disk.
func (dl *diskLayer) Parent() layer {
	return nil
}

// ID returns the state id of disk layer.
func (dl *diskLayer) ID() uint64 {
	return dl.id
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diskLayer) Stale() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
}

// MarkStale sets the stale flag as true.
func (dl *diskLayer) MarkStale() {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.stale {
		panic("triedb disk layer is stale") // we've committed into the same base from two children, boom
	}
	dl.stale = true
}

// Node retrieves the trie node with the provided node info. No error will be
// returned if the node is not found.
func (dl *diskLayer) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the dirty memory cache.
	// The map is lock free since it's impossible to mutate the
	// disk layer before tagging it as stale.
	n, err := dl.dirty.node(owner, path, hash)
	if err != nil {
		return nil, err
	}
	if n != nil {
		// Hit node in disk cache which resides in disk layer
		dirtyHitMeter.Mark(1)
		dirtyReadMeter.Mark(int64(len(n.Blob)))
		return n.Blob, nil
	}
	// If we're in the disk layer, all diff layers missed
	dirtyMissMeter.Mark(1)

	// Try to retrieve the trie node from the clean memory cache
	if dl.db.cleans != nil {
		if blob := dl.db.cleans.Get(nil, hash.Bytes()); len(blob) > 0 {
			cleanHitMeter.Mark(1)
			cleanReadMeter.Mark(int64(len(blob)))
			return blob, nil
		}
		cleanMissMeter.Mark(1)
	}
	// Try to retrieve the trie node from the disk.
	var (
		nBlob []byte
		nHash common.Hash
	)
	if owner == (common.Hash{}) {
		nBlob, nHash = rawdb.ReadAccountTrieNode(dl.db.diskdb, path)
	} else {
		nBlob, nHash = rawdb.ReadStorageTrieNode(dl.db.diskdb, owner, path)
	}
	if nHash != hash {
		return nil, &UnexpectedNodeError{
			typ:      "disk",
			expected: hash,
			hash:     nHash,
			owner:    owner,
			path:     path,
		}
	}
	if dl.db.cleans != nil && len(nBlob) > 0 {
		dl.db.cleans.Set(hash.Bytes(), nBlob)
		cleanWriteMeter.Mark(int64(len(nBlob)))
	}
	return nBlob, nil
}

// nodeByPath retrieves the trie node with the provided trie identifier and node
// path. No error will be returned if the node is not found.
func (dl *diskLayer) nodeByPath(owner common.Hash, path []byte) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the dirty memory cache.
	// The map is lock free since it's impossible to mutate the
	// disk layer before tagging it as stale.
	n, find := dl.dirty.nodeByPath(owner, path)
	if find {
		return n, nil
	}
	// Try to retrieve the trie node from the disk.
	var nBlob []byte
	if owner == (common.Hash{}) {
		nBlob, _ = rawdb.ReadAccountTrieNode(dl.db.diskdb, path)
	} else {
		nBlob, _ = rawdb.ReadStorageTrieNode(dl.db.diskdb, owner, path)
	}
	return nBlob, nil
}

// Update returns a new diff layer on top with the given dirty node set.
func (dl *diskLayer) Update(blockHash common.Hash, id uint64, nodes map[common.Hash]map[string]*trienode.WithPrev) *diffLayer {
	return newDiffLayer(dl, blockHash, id, nodes)
}

// commit merges the given bottom-most diff layer into the local cache
// and returns a newly constructed disk layer. Note the current disk
// layer must be tagged as stale first to prevent re-access.
func (dl *diskLayer) commit(bottom *diffLayer, force bool) (*diskLayer, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	// Construct and store the trie history first. If crash happens
	// after storing the trie history but without flushing the
	// corresponding statehashes(journal), the stored trie history will be
	// truncated in the next restart.
	if dl.db.freezer != nil {
		err := storeTrieHistory(dl.db.freezer, bottom, dl.db.config.StateLimit)
		if err != nil {
			return nil, err
		}
	}
	// Mark the diskLayer as stale before applying any mutations on top.
	dl.stale = true

	// Store the root->id lookup afterwards. All stored lookups are
	// identified by the **unique** state root. It's impossible that
	// in the same chain blocks which are not adjacent have the same
	// root.
	if dl.id == 0 {
		rawdb.WriteStateID(dl.db.diskdb, dl.root, 0)
	}
	rawdb.WriteStateID(dl.db.diskdb, bottom.Root(), bottom.ID())

	// Drop the previous value to reduce memory usage.
	slim := make(map[common.Hash]map[string]*trienode.Node)
	for owner, nodes := range bottom.nodes {
		subset := make(map[string]*trienode.Node)
		for path, n := range nodes {
			subset[path] = n.Unwrap()
		}
		slim[owner] = subset
	}
	ndl := newDiskLayer(bottom.root, bottom.id, dl.db, dl.dirty.commit(slim))

	// Persist the content in disk layer if there are too many nodes cached.
	err := ndl.dirty.flush(ndl.db.diskdb, ndl.db.cleans, ndl.id, force)
	if err != nil {
		return nil, err
	}
	return ndl, nil
}

// revert applies the given reverse diff by reverting the disk layer
// and return a newly constructed disk layer.
func (dl *diskLayer) revert(h *trieHistory) (*diskLayer, error) {
	if h.Root != dl.Root() {
		return nil, errUnexpectedTrieHistory
	}
	if dl.id == 0 {
		return nil, fmt.Errorf("%w: zero state id", errStateUnrecoverable)
	}
	// Mark the diskLayer as stale before applying any mutations on top.
	dl.lock.Lock()
	defer dl.lock.Unlock()

	dl.stale = true

	if !dl.dirty.empty() {
		// Revert embedded states in the disk set first in case
		// cache is not empty.
		err := dl.dirty.revert(h)
		if err != nil {
			return nil, err
		}
	} else {
		// The disk cache is empty, applies the state reverting
		// on disk directly.
		batch := dl.db.diskdb.NewBatch()
		if err := h.apply(batch); err != nil {
			return nil, err
		}
		rawdb.WritePersistentStateID(batch, dl.id-1)
		if err := batch.Write(); err != nil {
			log.Crit("Failed to write states", "err", err)
		}
		// Reset the clean cache in case disk state is mutated.
		if dl.db.cleans != nil {
			dl.db.cleans.Reset()
		}
	}
	return newDiskLayer(h.Parent, dl.id-1, dl.db, dl.dirty), nil
}

// setCacheSize sets the dirty cache size to the provided value.
func (dl *diskLayer) setCacheSize(size int) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return errSnapshotStale
	}
	return dl.dirty.setSize(size, dl.db.diskdb, dl.db.cleans, dl.id)
}

// size returns the approximate size of cached nodes in the disk layer.
func (dl *diskLayer) size() common.StorageSize {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return 0
	}
	return common.StorageSize(dl.dirty.size)
}
