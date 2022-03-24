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
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// diskLayerSnapshot is the snapshot of diskLayer.
type diskLayerSnapshot struct {
	prefix []byte           // Immutable, the unique identifier of snapshot to differentiate live and snap states
	root   common.Hash      // Immutable, root hash of the base snapshot
	diffid uint64           // Immutable, corresponding reverse diff id
	diskdb ethdb.Database   // Key-value store for storing temporary state changes, needs to be erased later
	snap   ethdb.Snapshot   // Key-value store snapshot created since the diskLayer snapshot is built
	clean  *fastcache.Cache // Clean node cache to avoid hitting the disk for direct access
	stale  bool             // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex     // Lock used to protect stale flag
}

// getSnapshot creates the disk layer snapshot by the given root.
// If the root is matched with the
func (dl *diskLayer) getSnapshot(nocache bool) (snap *diskLayerSnapshot, err error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return nil, errSnapshotStale
	}
	// Allocate an unique database namespace for the snapshot.
	prefix := make([]byte, 8)
	if _, err := rand.Read(prefix[:]); err != nil {
		return nil, err
	}
	// Create the disk snapshot for isolating the live database.
	db, err := dl.diskdb.NewSnapshot()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			return
		}
		// Release all held resources and cleanup the junks
		db.Release()
		rawdb.DeleteTrieNodeSnapshots(dl.diskdb, prefix)
	}()
	if !nocache {
		// The requested state is located in the embedded disk
		// cache, flushest all cached nodes into the ephemeral
		// disk area and apply the reverse diffs later. Note
		// the following operation may take a few seconds.
		batch := dl.diskdb.NewBatch()
		dl.dirty.forEach(func(key string, node *cachedNode) {
			if node.node == nil {
				rawdb.DeleteTrieNodeSnapshot(batch, prefix, []byte(key))
			} else {
				rawdb.WriteTrieNodeSnapshot(batch, prefix, []byte(key), node.rlp())
			}
		})
		if err := batch.Write(); err != nil {
			return nil, err
		}
		snap = &diskLayerSnapshot{
			prefix: prefix,
			root:   dl.root,
			diffid: dl.diffid,
			diskdb: dl.diskdb,
			snap:   db,
			clean:  fastcache.New(16 * 1024 * 1024), // tiny cache
		}
	} else {
		_, diskRoot := rawdb.ReadTrieNode(dl.diskdb, EncodeStorageKey(common.Hash{}, nil))
		snap = &diskLayerSnapshot{
			prefix: prefix,
			root:   diskRoot,
			diffid: rawdb.ReadReverseDiffHead(dl.diskdb),
			diskdb: dl.diskdb,
			snap:   db,
			clean:  fastcache.New(16 * 1024 * 1024), // tiny cache
		}
	}
	return snap, nil
}

// GetSnapshotAndRewind creates a disk layer snapshot and rewinds the snapshot
// to the specified state. In order to store the temporary mutations happened,
// the unique database namespace will be allocated for the snapshot and it's
// expected to be released after the usage.
func (dl *diskLayer) GetSnapshotAndRewind(root common.Hash) (*diskLayerSnapshot, error) {
	id := rawdb.ReadReverseDiffLookup(dl.diskdb, convertEmpty(root))
	if id == nil {
		return nil, errStateUnrecoverable
	}
	// If the requested state is even blow the disk state, then
	// the cached dirty nodes are not needed by the snapshot.
	nocache := *id <= rawdb.ReadReverseDiffHead(dl.diskdb)
	snap, err := dl.getSnapshot(nocache)
	if err != nil {
		return nil, err
	}
	// Apply the reverse diffs with the given order.
	for snap.diffid >= *id {
		diff, err := loadReverseDiff(snap.diskdb, snap.diffid)
		if err != nil {
			return nil, err
		}
		snap, err = snap.revert(diff, snap.diffid)
		if err != nil {
			return nil, err
		}
	}
	return snap, nil
}

// Root returns root hash of corresponding state.
func (snap *diskLayerSnapshot) Root() common.Hash {
	return snap.root
}

// Parent always returns nil as there's no layer below the disk.
func (snap *diskLayerSnapshot) Parent() snapshot {
	return nil
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (snap *diskLayerSnapshot) Stale() bool {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	return snap.stale
}

// ID returns the id of associated reverse diff.
func (snap *diskLayerSnapshot) ID() uint64 {
	return snap.diffid
}

// MarkStale sets the stale flag as true.
func (snap *diskLayerSnapshot) MarkStale() {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	if snap.stale == true {
		panic("triedb disk layer is stale") // we've committed into the same base from two children, boom
	}
	snap.stale = true
}

// Node retrieves the trie node associated with a particular key.
func (snap *diskLayerSnapshot) Node(storage []byte, hash common.Hash) (*cachedNode, error) {
	blob, err := snap.NodeBlob(storage, hash)
	if err != nil {
		return nil, err
	}
	if len(blob) == 0 {
		return nil, nil
	}
	return &cachedNode{node: rawNode(blob), hash: hash, size: uint16(len(blob))}, nil
}

// NodeBlob retrieves the trie node blob associated with a particular key.
func (snap *diskLayerSnapshot) NodeBlob(storage []byte, hash common.Hash) ([]byte, error) {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.stale {
		return nil, errSnapshotStale
	}
	// Try to retrieve the trie node from the memory cache
	ikey := EncodeInternalKey(storage, hash)
	if blob, found := snap.clean.HasGet(nil, ikey); found && len(blob) > 0 {
		return blob, nil
	}
	// Firstly try to retrieve the trie node from the ephemeral
	// disk area or fallback to the live disk state if it's not
	// existent.
	blob, nodeHash := rawdb.ReadTrieNodeSnapshot(snap.diskdb, snap.prefix, storage)
	if len(blob) == 0 || nodeHash != hash {
		blob, nodeHash = rawdb.ReadTrieNode(snap.snap, storage)
		if len(blob) == 0 || nodeHash != hash {
			blob = rawdb.ReadLegacyTrieNode(snap.snap, hash)
		}
	}
	if len(blob) > 0 {
		snap.clean.Set(ikey, blob)
	}
	return blob, nil
}

// Update returns a new diff layer on top with the given dirty node set.
func (snap *diskLayerSnapshot) Update(blockHash common.Hash, id uint64, nodes map[string]*nodeWithPreValue) *diffLayer {
	return newDiffLayer(snap, blockHash, id, nodes)
}

// Journal it's not supported by diskLayer snapshot.
func (snap *diskLayerSnapshot) Journal(buffer *bytes.Buffer) error {
	return errors.New("not implemented")
}

// commit flushes the dirty nodes in bottom-most diff layer into
// disk. The nodes will be stored in an ephemeral disk area and
// will be erased once the snapshot itself is released.
func (snap *diskLayerSnapshot) commit(bottom *diffLayer) (*diskLayerSnapshot, error) {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	// Mark the snapshot as stale before applying any mutations on top.
	snap.stale = true

	// Commit the dirty nodes in the diff layer.
	batch := snap.diskdb.NewBatch()
	for key, n := range bottom.nodes {
		if n.node == nil {
			rawdb.DeleteTrieNodeSnapshot(batch, snap.prefix, []byte(key))
		} else {
			blob := n.rlp()
			rawdb.WriteTrieNodeSnapshot(batch, snap.prefix, []byte(key), blob)
			snap.clean.Set(EncodeInternalKey([]byte(key), n.hash), blob)
		}
	}
	if err := batch.Write(); err != nil {
		return nil, err
	}
	return &diskLayerSnapshot{
		prefix: snap.prefix,
		root:   bottom.root,
		diffid: bottom.diffid,
		diskdb: snap.diskdb,
		snap:   snap.snap,
		clean:  snap.clean,
	}, nil
}

// revert applies the given reverse diff by reverting the disk layer
// and return a newly constructed disk layer.
func (snap *diskLayerSnapshot) revert(diff *reverseDiff, diffid uint64) (*diskLayerSnapshot, error) {
	var (
		root  = snap.Root()
		batch = snap.diskdb.NewBatch()
	)
	if diff.Root != root {
		return nil, errUnmatchedReverseDiff
	}
	if diffid != snap.diffid {
		return nil, errUnmatchedReverseDiff
	}
	if snap.diffid == 0 {
		return nil, fmt.Errorf("%w: zero reverse diff id", errStateUnrecoverable)
	}
	// Mark the snapshot as stale before applying any mutations on top.
	snap.lock.Lock()
	defer snap.lock.Unlock()

	snap.stale = true

	for _, state := range diff.States {
		if len(state.Val) > 0 {
			rawdb.WriteTrieNodeSnapshot(batch, snap.prefix, state.Key, state.Val)
		} else {
			rawdb.DeleteTrieNodeSnapshot(batch, snap.prefix, state.Key)
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write reverse diff", "err", err)
	}
	batch.Reset()

	return &diskLayerSnapshot{
		prefix: snap.prefix,
		root:   diff.Parent,
		diffid: snap.diffid - 1,
		diskdb: snap.diskdb,
		snap:   snap.snap,
		clean:  snap.clean,
	}, nil
}

func (snap *diskLayerSnapshot) Release() {
	// Release the read-only disk database snapshot.
	snap.snap.Release()

	// Clean up all the written nodes in the ephemeral disk area.
	rawdb.DeleteTrieNodeSnapshots(snap.diskdb, snap.prefix)
}
