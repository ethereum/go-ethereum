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

// DatabaseSnapshot implements NodeDatabase by creating the isolated
// database snapshot. The snapshot can be mutated(revert, update) and
// the state changes will be erased once the snapshot is released.
type DatabaseSnapshot struct {
	tree     *layerTree
	released bool
	lock     sync.RWMutex
	diskdb   ethdb.Database
}

// GetSnapshot initializes the database snapshot with the given target state
// identifier. The returned snapshot should be released otherwise resource
// leak can happen. It's only supported by hash-based database.
func (db *Database) GetSnapshot(root common.Hash) (*DatabaseSnapshot, error) {
	snapDB, ok := db.backend.(*snapDatabase)
	if !ok {
		return nil, errors.New("not supported")
	}
	if snapDB.freezer == nil {
		return nil, errors.New("unrecoverable trie database")
	}
	snap, err := snapDB.tree.bottom().(*diskLayer).GetSnapshot(root, snapDB.freezer)
	if err != nil {
		return nil, err
	}
	return &DatabaseSnapshot{
		tree:   newLayerTree(snap),
		diskdb: db.diskdb,
	}, nil
}

// GetReader retrieves a snapshot belonging to the given block root.
func (snap *DatabaseSnapshot) GetReader(blockRoot common.Hash) Reader {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.released {
		return nil
	}
	return snap.tree.get(blockRoot)
}

// Commit adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all). Apart
// from that this function will flatten the extra diff layers at bottom into disk
// to only keep 128 diff layers in memory. All the mutations caused in disk can be
// erased later by invoking Release function.
func (snap *DatabaseSnapshot) Commit(root common.Hash, parent common.Hash, nodes *NodeSet) error {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.released {
		return errSnapshotReleased
	}
	if err := snap.tree.add(root, parent, nodes.nodes); err != nil {
		return err
	}
	// Keep 128 diff layers in the memory, persistent layer is 129th.
	// - head layer is paired with HEAD state
	// - head-1 layer is paired with HEAD-1 state
	// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
	// - head-128 layer(disk layer) is paired with HEAD-128 state
	return snap.tree.cap(root, 128, nil, 0)
}

// DiskDB returns the underlying database handler.
func (snap *DatabaseSnapshot) DiskDB() ethdb.Database {
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
	snap.tree.bottom().(*diskLayerSnapshot).Release()
}
