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
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

// maxDiffLayers is the maximum diff layers allowed in the layer tree.
const maxDiffLayers = 128

// layer is the interface implemented by all state layers which includes some
// public methods and some additional methods for internal usage.
type layer interface {
	// Node retrieves the trie node with the node info. An error will be returned
	// if the read operation exits abnormally. For example, if the layer is already
	// stale, or the associated state is regarded as corrupted. Notably, no error
	// will be returned if the requested node is not found in database.
	Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error)

	// rootHash returns the root hash for which this layer was made.
	rootHash() common.Hash

	// stateID returns the associated state id of layer.
	stateID() uint64

	// parentLayer returns the subsequent layer of it, or nil if the disk was reached.
	parentLayer() layer

	// update creates a new layer on top of the existing layer diff tree with
	// the provided dirty trie nodes along with the state change set.
	//
	// Note, the maps are retained by the method to avoid copying everything.
	update(root common.Hash, id uint64, block uint64, nodes map[common.Hash]map[string]*trienode.Node, states *triestate.Set) *diffLayer

	// journal commits an entire diff hierarchy to disk into a single journal entry.
	// This is meant to be used during shutdown to persist the layer without
	// flattening everything down (bad for reorgs).
	journal(w io.Writer) error
}

// Config contains the settings for database.
type Config struct {
	StateLimit uint64 // Number of recent blocks to maintain state history for
	CleanSize  int    // Maximum memory allowance (in bytes) for caching clean nodes
	DirtySize  int    // Maximum memory allowance (in bytes) for caching dirty nodes
	ReadOnly   bool   // Flag whether the database is opened in read only mode.
}

var (
	// defaultCleanSize is the default memory allowance of clean cache.
	defaultCleanSize = 16 * 1024 * 1024

	// defaultBufferSize is the default memory allowance of node buffer
	// that aggregates the writes from above until it's flushed into the
	// disk. Do not increase the buffer size arbitrarily, otherwise the
	// system pause time will increase when the database writes happen.
	defaultBufferSize = 128 * 1024 * 1024
)

// Defaults contains default settings for Ethereum mainnet.
var Defaults = &Config{
	StateLimit: params.FullImmutabilityThreshold,
	CleanSize:  defaultCleanSize,
	DirtySize:  defaultBufferSize,
}

// Database is a multiple-layered structure for maintaining in-memory trie nodes.
// It consists of one persistent base layer backed by a key-value store, on top
// of which arbitrarily many in-memory diff layers are stacked. The memory diffs
// can form a tree with branching, but the disk layer is singleton and common to
// all. If a reorg goes deeper than the disk layer, a batch of reverse diffs can
// be applied to rollback. The deepest reorg that can be handled depends on the
// amount of state histories tracked in the disk.
//
// At most one readable and writable database can be opened at the same time in
// the whole system which ensures that only one database writer can operate disk
// state. Unexpected open operations can cause the system to panic.
type Database struct {
	// readOnly is the flag whether the mutation is allowed to be applied.
	// It will be set automatically when the database is journaled during
	// the shutdown to reject all following unexpected mutations.
	readOnly   bool                     // Indicator if database is opened in read only mode
	bufferSize int                      // Memory allowance (in bytes) for caching dirty nodes
	config     *Config                  // Configuration for database
	diskdb     ethdb.Database           // Persistent storage for matured trie nodes
	tree       *layerTree               // The group for all known layers
	freezer    *rawdb.ResettableFreezer // Freezer for storing trie histories, nil possible in tests
	lock       sync.RWMutex             // Lock to prevent mutations from happening at the same time
}

// New attempts to load an already existing layer from a persistent key-value
// store (with a number of memory layers from a journal). If the journal is not
// matched with the base persistent layer, all the recorded diff layers are discarded.
func New(diskdb ethdb.Database, config *Config) *Database {
	if config == nil {
		config = Defaults
	}
	db := &Database{
		readOnly:   config.ReadOnly,
		bufferSize: config.DirtySize,
		config:     config,
		diskdb:     diskdb,
	}
	// Construct the layer tree by resolving the in-disk singleton state
	// and in-memory layer journal.
	db.tree = newLayerTree(db.loadLayers())

	// Open the freezer for state history if the passed database contains an
	// ancient store. Otherwise, all the relevant functionalities are disabled.
	//
	// Because the freezer can only be opened once at the same time, this
	// mechanism also ensures that at most one **non-readOnly** database
	// is opened at the same time to prevent accidental mutation.
	if ancient, err := diskdb.AncientDatadir(); err == nil && ancient != "" && !db.readOnly {
		freezer, err := rawdb.NewStateHistoryFreezer(ancient, false)
		if err != nil {
			log.Crit("Failed to open state history freezer", "err", err)
		}
		db.freezer = freezer

		// Truncate the extra state histories above in freezer in case
		// it's not aligned with the disk layer.
		pruned, err := truncateFromHead(db.diskdb, freezer, db.tree.bottom().stateID())
		if err != nil {
			log.Crit("Failed to truncate extra state histories", "err", err)
		}
		if pruned != 0 {
			log.Warn("Truncated extra state histories", "number", pruned)
		}
	}
	log.Warn("Path-based state scheme is an experimental feature")
	return db
}

// Reader retrieves a layer belonging to the given state root.
func (db *Database) Reader(root common.Hash) (layer, error) {
	l := db.tree.get(root)
	if l == nil {
		return nil, fmt.Errorf("state %#x is not available", root)
	}
	return l, nil
}

// Update adds a new layer into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all). Apart
// from that this function will flatten the extra diff layers at bottom into disk
// to only keep 128 diff layers in memory by default.
//
// The passed in maps(nodes, states) will be retained to avoid copying everything.
// Therefore, these maps must not be changed afterwards.
func (db *Database) Update(root common.Hash, parentRoot common.Hash, block uint64, nodes *trienode.MergedNodeSet, states *triestate.Set) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	if err := db.tree.add(root, parentRoot, block, nodes, states); err != nil {
		return err
	}
	// Keep 128 diff layers in the memory, persistent layer is 129th.
	// - head layer is paired with HEAD state
	// - head-1 layer is paired with HEAD-1 state
	// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
	// - head-128 layer(disk layer) is paired with HEAD-128 state
	return db.tree.cap(root, maxDiffLayers)
}

// Commit traverses downwards the layer tree from a specified layer with the
// provided state root and all the layers below are flattened downwards. It
// can be used alone and mostly for test purposes.
func (db *Database) Commit(root common.Hash, report bool) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	return db.tree.cap(root, 0)
}

// Reset rebuilds the database with the specified state as the base.
//
//   - if target state is empty, clear the stored state and all layers on top
//   - if target state is non-empty, ensure the stored state matches with it
//     and clear all other layers on top.
func (db *Database) Reset(root common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	batch := db.diskdb.NewBatch()
	root = types.TrieRootHash(root)
	if root == types.EmptyRootHash {
		// Empty state is requested as the target, nuke out
		// the root node and leave all others as dangling.
		rawdb.DeleteAccountTrieNode(batch, nil)
	} else {
		// Ensure the requested state is existent before any
		// action is applied.
		_, hash := rawdb.ReadAccountTrieNode(db.diskdb, nil)
		if hash != root {
			return fmt.Errorf("state is mismatched, local: %x, target: %x", hash, root)
		}
	}
	// Mark the disk layer as stale before applying any mutation.
	db.tree.bottom().markStale()

	// Drop the stale state journal in persistent database and
	// reset the persistent state id back to zero.
	rawdb.DeleteTrieJournal(batch)
	rawdb.WritePersistentStateID(batch, 0)
	if err := batch.Write(); err != nil {
		return err
	}
	// Clean up all state histories in freezer. Theoretically
	// all root->id mappings should be removed as well. Since
	// mappings can be huge and might take a while to clear
	// them, just leave them in disk and wait for overwriting.
	if db.freezer != nil {
		if err := db.freezer.Reset(); err != nil {
			return err
		}
	}
	// Re-construct a new disk layer backed by persistent state
	// with **empty clean cache and node buffer**.
	dl := newDiskLayer(root, 0, db, nil, newNodeBuffer(db.bufferSize, nil, 0))
	db.tree.reset(dl)
	log.Info("Rebuilt trie database", "root", root)
	return nil
}

// Recover rollbacks the database to a specified historical point.
// The state is supported as the rollback destination only if it's
// canonical state and the corresponding trie histories are existent.
func (db *Database) Recover(root common.Hash, loader triestate.TrieLoader) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if rollback operation is not supported.
	if db.readOnly || db.freezer == nil {
		return errors.New("state rollback is non-supported")
	}
	// Short circuit if the target state is not recoverable.
	root = types.TrieRootHash(root)
	if !db.Recoverable(root) {
		return errStateUnrecoverable
	}
	// Apply the state histories upon the disk layer in order.
	var (
		start = time.Now()
		dl    = db.tree.bottom()
	)
	for dl.rootHash() != root {
		h, err := readHistory(db.freezer, dl.stateID())
		if err != nil {
			return err
		}
		dl, err = dl.revert(h, loader)
		if err != nil {
			return err
		}
		// reset layer with newly created disk layer. It must be
		// done after each revert operation, otherwise the new
		// disk layer won't be accessible from outside.
		db.tree.reset(dl)
	}
	rawdb.DeleteTrieJournal(db.diskdb)
	_, err := truncateFromHead(db.diskdb, db.freezer, dl.stateID())
	if err != nil {
		return err
	}
	log.Debug("Recovered state", "root", root, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// Recoverable returns the indicator if the specified state is recoverable.
func (db *Database) Recoverable(root common.Hash) bool {
	// Ensure the requested state is a known state.
	root = types.TrieRootHash(root)
	id := rawdb.ReadStateID(db.diskdb, root)
	if id == nil {
		return false
	}
	// Recoverable state must below the disk layer. The recoverable
	// state only refers the state that is currently not available,
	// but can be restored by applying state history.
	dl := db.tree.bottom()
	if *id >= dl.stateID() {
		return false
	}
	// Ensure the requested state is a canonical state and all state
	// histories in range [id+1, disklayer.ID] are present and complete.
	parent := root
	return checkHistories(db.freezer, *id+1, dl.stateID()-*id, func(m *meta) error {
		if m.parent != parent {
			return errors.New("unexpected state history")
		}
		if len(m.incomplete) > 0 {
			return errors.New("incomplete state history")
		}
		parent = m.root
		return nil
	}) == nil
}

// Close closes the trie database and the held freezer.
func (db *Database) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.readOnly = true
	if db.freezer == nil {
		return nil
	}
	return db.freezer.Close()
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *Database) Size() (size common.StorageSize) {
	db.tree.forEach(func(layer layer) {
		if diff, ok := layer.(*diffLayer); ok {
			size += common.StorageSize(diff.memory)
		}
		if disk, ok := layer.(*diskLayer); ok {
			size += disk.size()
		}
	})
	return size
}

// Initialized returns an indicator if the state data is already
// initialized in path-based scheme.
func (db *Database) Initialized(genesisRoot common.Hash) bool {
	var inited bool
	db.tree.forEach(func(layer layer) {
		if layer.rootHash() != types.EmptyRootHash {
			inited = true
		}
	})
	return inited
}

// SetBufferSize sets the node buffer size to the provided value(in bytes).
func (db *Database) SetBufferSize(size int) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.bufferSize = size
	return db.tree.bottom().setBufferSize(db.bufferSize)
}

// Scheme returns the node scheme used in the database.
func (db *Database) Scheme() string {
	return rawdb.PathScheme
}
