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

package trie

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

var (
	// errSnapshotReleased is returned if callers want to access a released
	// database snapshot.
	errSnapshotReleased = errors.New("database snapshot released")
)

// DatabaseSnapshot implements StateDatabase by creating the isolated
// database snapshot. The snapshot can be mutated(revert, update) and
// the state changes will be erased once the snapshot is released.
type DatabaseSnapshot struct {
	tree     *layerTree
	released bool
	lock     sync.RWMutex
	diskdb   ethdb.Database
}

// NewDatabaseSnapshot initializes the database snapshot with the given
// live database and the target state identifier. The returned snapshot
// should be released otherwise resource leak will happen.
func NewDatabaseSnapshot(db *Database, root common.Hash) (*DatabaseSnapshot, error) {
	snap, err := db.disklayer().GetSnapshotAndRewind(root)
	if err != nil {
		return nil, err
	}
	return &DatabaseSnapshot{
		tree:   newLayerTree(snap),
		diskdb: db.diskdb,
	}, nil
}

// Snapshot retrieves a snapshot belonging to the given block root.
func (snap *DatabaseSnapshot) Snapshot(blockRoot common.Hash) Snapshot {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.released {
		return nil
	}
	return snap.tree.get(blockRoot)
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
// The passed keys must all be encoded in the **storage** format.
func (snap *DatabaseSnapshot) Update(root common.Hash, parentRoot common.Hash, nodes map[string]*nodeWithPreValue) error {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.released {
		return errSnapshotReleased
	}
	return snap.tree.add(root, parentRoot, nodes)
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards.
func (snap *DatabaseSnapshot) Cap(root common.Hash, layers int) error {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	if snap.released {
		return errSnapshotReleased
	}
	return snap.tree.cap(root, layers)
}

// DiskDB returns the underlying database handler.
func (snap *DatabaseSnapshot) DiskDB() ethdb.KeyValueStore {
	return snap.diskdb
}

// Release releases the snapshot and all relevant resources held.
// It's safe to call Release multiple times.
func (snap *DatabaseSnapshot) Release() {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	if snap.released {
		return
	}
	snap.released = true

	snap.tree.forEach(func(hash common.Hash, layer snapshot) bool {
		if dl, ok := layer.(*diskLayerSnapshot); ok {
			dl.Release()
			return false
		}
		return true
	})
}
