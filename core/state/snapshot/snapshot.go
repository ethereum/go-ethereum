// Copyright 2019 The go-ethereum Authors
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

// Package snapshot implements a journalled, dynamic state dump.
package snapshot

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	snapshotCleanHitMeter   = metrics.NewRegisteredMeter("state/snapshot/clean/hit", nil)
	snapshotCleanMissMeter  = metrics.NewRegisteredMeter("state/snapshot/clean/miss", nil)
	snapshotCleanReadMeter  = metrics.NewRegisteredMeter("state/snapshot/clean/read", nil)
	snapshotCleanWriteMeter = metrics.NewRegisteredMeter("state/snapshot/clean/write", nil)

	// ErrSnapshotStale is returned from data accessors if the underlying snapshot
	// layer had been invalidated due to the chain progressing forward far enough
	// to not maintain the layer's original state.
	ErrSnapshotStale = errors.New("snapshot stale")
)

// Snapshot represents the functionality supported by a snapshot storage layer.
type Snapshot interface {
	// Info returns the block number and root hash for which this snapshot was made.
	Info() (uint64, common.Hash)

	// Account directly retrieves the account associated with a particular hash in
	// the snapshot slim data format.
	Account(hash common.Hash) (*Account, error)

	// AccountRLP directly retrieves the account RLP associated with a particular
	// hash in the snapshot slim data format.
	AccountRLP(hash common.Hash) ([]byte, error)

	// Storage directly retrieves the storage data associated with a particular hash,
	// within a particular account.
	Storage(accountHash, storageHash common.Hash) ([]byte, error)
}

// snapshot is the internal version of the snapshot data layer that supports some
// additional methods compared to the public API.
type snapshot interface {
	Snapshot

	// Update creates a new layer on top of the existing snapshot diff tree with
	// the specified data items. Note, the maps are retained by the method to avoid
	// copying everything.
	Update(blockRoot common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer

	// Cap traverses downwards the diff tree until the number of allowed layers are
	// crossed. All diffs beyond the permitted number are flattened downwards. The
	// block numbers for the disk layer and first diff layer are returned for GC.
	Cap(layers int, memory uint64) (uint64, uint64)

	// Journal commits an entire diff hierarchy to disk into a single journal file.
	// This is meant to be used during shutdown to persist the snapshot without
	// flattening everything down (bad for reorgs).
	Journal() error
}

// SnapshotTree is an Ethereum state snapshot tree. It consists of one persistent
// base layer backed by a key-value store, on top of which arbitrarily many in-
// memory diff layers are topped. The memory diffs can form a tree with branching,
// but the disk layer is singleton and common to all. If a reorg goes deeper than
// the disk layer, everything needs to be deleted.
//
// The goal of a state snapshot is twofold: to allow direct access to account and
// storage data to avoid expensive multi-level trie lookups; and to allow sorted,
// cheap iteration of the account/storage tries for sync aid.
type SnapshotTree struct {
	layers map[common.Hash]snapshot // Collection of all known layers // TODO(karalabe): split Clique overlaps
	lock   sync.RWMutex
}

// New attempts to load an already existing snapshot from a persistent key-value
// store (with a number of memory layers from a journal), ensuring that the head
// of the snapshot matches the expected one.
//
// If the snapshot is missing or inconsistent, the entirety is deleted and will
// be reconstructed from scratch based on the tries in the key-value store.
func New(db ethdb.KeyValueStore, journal string, headNumber uint64, headRoot common.Hash) (*SnapshotTree, error) {
	// Attempt to load a previously persisted snapshot
	head, err := loadSnapshot(db, journal, headNumber, headRoot)
	if err != nil {
		log.Warn("Failed to load snapshot, regenerating", "err", err)
		if head, err = generateSnapshot(db, journal, headNumber, headRoot); err != nil {
			return nil, err
		}
	}
	// Existing snapshot loaded or one regenerated, seed all the layers
	snap := &SnapshotTree{
		layers: make(map[common.Hash]snapshot),
	}
	for head != nil {
		_, root := head.Info()
		snap.layers[root] = head

		switch self := head.(type) {
		case *diffLayer:
			head = self.parent
		case *diskLayer:
			head = nil
		default:
			panic(fmt.Sprintf("unknown data layer: %T", self))
		}
	}
	return snap, nil
}

// Snapshot retrieves a snapshot belonging to the given block root, or nil if no
// snapshot is maintained for that block.
func (st *SnapshotTree) Snapshot(blockRoot common.Hash) Snapshot {
	st.lock.RLock()
	defer st.lock.RUnlock()

	return st.layers[blockRoot]
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
func (st *SnapshotTree) Update(blockRoot common.Hash, parentRoot common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) error {
	// Generate a new snapshot on top of the parent
	parent := st.Snapshot(parentRoot).(snapshot)
	if parent == nil {
		return fmt.Errorf("parent [%#x] snapshot missing", parentRoot)
	}
	snap := parent.Update(blockRoot, accounts, storage)

	// Save the new snapshot for later
	st.lock.Lock()
	defer st.lock.Unlock()

	st.layers[snap.root] = snap
	return nil
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards.
func (st *SnapshotTree) Cap(blockRoot common.Hash, layers int, memory uint64) error {
	// Retrieve the head snapshot to cap from
	snap := st.Snapshot(blockRoot).(snapshot)
	if snap == nil {
		return fmt.Errorf("snapshot [%#x] missing", blockRoot)
	}
	// Run the internal capping and discard all stale layers
	st.lock.Lock()
	defer st.lock.Unlock()

	diskNumber, diffNumber := snap.Cap(layers, memory)
	for root, snap := range st.layers {
		if number, _ := snap.Info(); number != diskNumber && number < diffNumber {
			delete(st.layers, root)
		}
	}
	return nil
}

// Journal commits an entire diff hierarchy to disk into a single journal file.
// This is meant to be used during shutdown to persist the snapshot without
// flattening everything down (bad for reorgs).
func (st *SnapshotTree) Journal(blockRoot common.Hash) error {
	// Retrieve the head snapshot to journal from
	snap := st.Snapshot(blockRoot).(snapshot)
	if snap == nil {
		return fmt.Errorf("snapshot [%#x] missing", blockRoot)
	}
	// Run the journaling
	st.lock.Lock()
	defer st.lock.Unlock()

	return snap.Journal()
}

// loadSnapshot loads a pre-existing state snapshot backed by a key-value store.
func loadSnapshot(db ethdb.KeyValueStore, journal string, headNumber uint64, headRoot common.Hash) (snapshot, error) {
	// Retrieve the block number and hash of the snapshot, failing if no snapshot
	// is present in the database (or crashed mid-update).
	number, root := rawdb.ReadSnapshotBlock(db)
	if root == (common.Hash{}) {
		return nil, errors.New("missing or corrupted snapshot")
	}
	cache, _ := bigcache.NewBigCache(bigcache.Config{ // TODO(karalabe): dedup
		Shards:             1024,
		LifeWindow:         time.Hour,
		MaxEntriesInWindow: 512 * 1024,
		MaxEntrySize:       512,
		HardMaxCacheSize:   512,
	})
	base := &diskLayer{
		journal: journal,
		db:      db,
		cache:   cache,
		number:  number,
		root:    root,
	}
	// Load all the snapshot diffs from the journal, failing if their chain is broken
	// or does not lead from the disk snapshot to the specified head.
	if _, err := os.Stat(journal); os.IsNotExist(err) {
		// Journal doesn't exist, don't worry if it's not supposed to
		if number != headNumber || root != headRoot {
			return nil, fmt.Errorf("snapshot journal missing, head doesn't match snapshot: #%d [%#x] vs. #%d [%#x]",
				headNumber, headRoot, number, root)
		}
		return base, nil
	}
	file, err := os.Open(journal)
	if err != nil {
		return nil, err
	}
	snapshot, err := loadDiffLayer(base, rlp.NewStream(file, 0))
	if err != nil {
		return nil, err
	}
	// Entire snapshot journal loaded, sanity check the head and return
	// Journal doesn't exist, don't worry if it's not supposed to
	number, root = snapshot.Info()
	if number != headNumber || root != headRoot {
		return nil, fmt.Errorf("head doesn't match snapshot: #%d [%#x] vs. #%d [%#x]",
			headNumber, headRoot, number, root)
	}
	return snapshot, nil
}
