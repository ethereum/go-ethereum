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
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// maxDiffLayers is the maximum diff layers allowed in the layer tree.
const maxDiffLayers = 128

// snapshot is the interface implemented by all state layers which includes some
// public methods and some additional methods for internal usage.
type snapshot interface {
	// nodeByPath retrieves the trie node with the provided trie identifier and
	// node path regardless what's the node hash. No error will be returned if
	// the node is not found.
	nodeByPath(owner common.Hash, path []byte) ([]byte, error)

	// Node retrieves the trie node with the node info. No error will be returned
	// if the node is not found.
	//
	// TODO(rjl493456442) remove this function. Hash is essentially not required
	// for accessing a trie node, nodeByPath is enough as the accessor. Keep it
	// for a while to make initial version easier.
	Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error)

	// Root returns the root hash for which this snapshot was made.
	Root() common.Hash

	// Parent returns the subsequent layer of a snapshot, or nil if the base was
	// reached.
	//
	// Note, the method is an internal helper to avoid type switching between the
	// disk and diff layers. There is no locking involved.
	Parent() snapshot

	// Update creates a new layer on top of the existing snapshot diff tree with
	// the given dirty trie node set. The deleted trie nodes are also included.
	//
	// Note, the maps are retained by the method to avoid copying everything.
	Update(blockRoot common.Hash, id uint64, nodes map[common.Hash]map[string]*trienode.WithPrev) *diffLayer

	// Journal commits an entire diff hierarchy to disk into a single journal entry.
	// This is meant to be used during shutdown to persist the snapshot without
	// flattening everything down (bad for reorgs).
	Journal(buffer *bytes.Buffer) error

	// Stale returns whether this layer has become stale (was flattened across) or
	// if it's still live.
	Stale() bool

	// ID returns the associated state id.
	ID() uint64
}

// Config contains the settings for database.
type Config struct {
	StateLimit uint64 // Number of recent blocks to maintain state history for
	DirtySize  int    // Maximum memory allowance (in bytes) for caching dirty nodes
	ReadOnly   bool   // Flag whether the database is opened in read only mode.
}

// Defaults contains default settings for use on the Ethereum main net.
var Defaults = &Config{
	StateLimit: params.FullImmutabilityThreshold,
	DirtySize:  defaultCacheSize,
}

// Database is a multiple-layered structure for maintaining in-memory trie
// nodes. It consists of one persistent base layer backed by a key-value store,
// on top of which arbitrarily many in-memory diff layers are topped. The memory
// diffs can form a tree with branching, but the disk layer is singleton and
// common to all. If a reorg goes deeper than the disk layer, a batch of reverse
// diffs can be applied to rollback. The deepest reorg can be handled depends on
// the amount of trie histories tracked in the disk.
//
// At most one readable and writable snap database can be opened at the same time
// in the whole system which ensures that only one database writer can operate
// disk state. Unexpected open operations can cause the system to panic.
type Database struct {
	// readOnly is the flag whether the mutation is allowed to be applied.
	// It will be set automatically when the database is journaled during
	// the shutdown to reject all following unexpected mutations.
	readOnly  bool                     // Indicator if database is opened in read only mode
	dirtySize int                      // Memory allowance (in bytes) for caching dirty nodes
	config    *Config                  // Configuration for database
	diskdb    ethdb.Database           // Persistent storage for matured trie nodes
	cleans    *fastcache.Cache         // GC friendly memory cache of clean node RLPs
	tree      *layerTree               // The group for all known layers
	freezer   *rawdb.ResettableFreezer // Freezer for storing trie histories, nil possible in tests
	lock      sync.RWMutex             // Lock to prevent mutations from happening at the same time
}

// New attempts to load an already existing snapshot from a persistent key-value
// store (with a number of memory layers from a journal). If the journal is not
// matched with the base persistent layer, all the recorded diff layers are discarded.
func New(diskdb ethdb.Database, cleans *fastcache.Cache, config *Config) *Database {
	if config == nil {
		config = Defaults
	}
	db := &Database{
		readOnly:  config.ReadOnly,
		dirtySize: config.DirtySize,
		config:    config,
		diskdb:    diskdb,
		cleans:    cleans,
	}
	// Construct the layer tree by resolving the in-disk singleton state
	// and in-memory layer journal.
	db.tree = newLayerTree(db.loadSnapshot())

	// Open the freezer for trie history if the passed database contains an
	// ancient store. Otherwise, all the relevant functionalities are disabled.
	//
	// Because the freezer can only be opened once at the same time, this
	// mechanism also ensures that at most one **non-readOnly** snap database
	// is opened at the same time to prevent accidental mutation.
	if ancient, err := diskdb.AncientDatadir(); err == nil && ancient != "" && !db.readOnly {
		freezer, err := rawdb.NewTrieHistoryFreezer(ancient, false)
		if err != nil {
			log.Crit("Failed to open trie history freezer", "err", err)
		}
		db.freezer = freezer

		// Truncate the extra trie histories above in freezer in case
		// it's not aligned with the disk layer.
		pruned, err := truncateFromHead(freezer, db.tree.bottom().ID())
		if err != nil {
			log.Crit("Failed to truncate extra trie histories", "err", err)
		}
		if pruned != 0 {
			log.Info("Truncated extra trie histories", "number", pruned)
		}
	}
	log.Warn("Path-based trie scheme is an experimental feature")
	return db
}

// Reader retrieves a snapshot belonging to the given state root.
func (db *Database) Reader(root common.Hash) (snapshot, error) {
	l := db.tree.get(root)
	if l == nil {
		return nil, fmt.Errorf("state %#x is not available", root)
	}
	return l, nil
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all). Apart
// from that this function will flatten the extra diff layers at bottom into disk
// to only keep 128 diff layers in memory.
func (db *Database) Update(root common.Hash, parentRoot common.Hash, nodes *trienode.MergedNodeSet) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	if err := db.tree.add(root, parentRoot, nodes); err != nil {
		return err
	}
	// Keep 128 diff layers in the memory, persistent layer is 129th.
	// - head layer is paired with HEAD state
	// - head-1 layer is paired with HEAD-1 state
	// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
	// - head-128 layer(disk layer) is paired with HEAD-128 state
	return db.tree.cap(root, maxDiffLayers)
}

// Commit traverses downwards the snapshot tree from a specified layer with the
// provided state root and all the layers below are flattened downwards. It can
// be used alone and mostly for test purposes.
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

// Journal commits an entire diff hierarchy to disk into a single journal entry.
// This is meant to be used during shutdown to persist the snapshot without
// flattening everything down (bad for reorgs). And this function will mark the
// database as read-only to prevent all following mutation to disk.
func (db *Database) Journal(root common.Hash) error {
	// Retrieve the head snapshot to journal from var snap snapshot
	snap := db.tree.get(root)
	if snap == nil {
		return fmt.Errorf("triedb snapshot [%#x] missing", root)
	}
	// Run the journaling
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	// Firstly write out the metadata of journal
	journal := new(bytes.Buffer)
	if err := rlp.Encode(journal, journalVersion); err != nil {
		return err
	}
	// The stored state in disk might be empty, convert the
	// root to emptyRoot in this case.
	_, diskroot := rawdb.ReadAccountTrieNode(db.diskdb, nil)
	diskroot = types.TrieRootHash(diskroot)

	// Secondly write out the disk layer root, ensure the
	// diff journal is continuous with disk.
	if err := rlp.Encode(journal, diskroot); err != nil {
		return err
	}
	// Finally write out the journal of each layer in reverse order.
	if err := snap.Journal(journal); err != nil {
		return err
	}
	// Store the journal into the database and return
	rawdb.WriteTrieJournal(db.diskdb, journal.Bytes())

	// Set the db in read only mode to reject all following mutations
	db.readOnly = true
	log.Info("Stored journal in triedb", "disk", diskroot, "size", common.StorageSize(journal.Len()))
	return nil
}

// Reset rebuilds the snap database with the specified state from scratch.
// If the target state is non-empty, then the stored state must be matched
// with provided state root.
func (db *Database) Reset(root common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	root = types.TrieRootHash(root)

	batch := db.diskdb.NewBatch()
	if root == types.EmptyRootHash {
		// Empty state is requested as the target, nuke out
		// the root node and leave all others as dangling.
		rawdb.DeleteAccountTrieNode(batch, nil)
	} else {
		// Ensure the requested state is existent before any
		// action is applied.
		_, hash := rawdb.ReadAccountTrieNode(db.diskdb, nil)
		if hash != root {
			return fmt.Errorf("state is mismatched, local %x target %x", hash, root)
		}
	}
	// Iterate over all layers and mark them as stale
	db.tree.forEach(func(layer snapshot) {
		switch layer := layer.(type) {
		case *diskLayer:
			layer.MarkStale()
		case *diffLayer:
			layer.MarkStale()
		default:
			panic(fmt.Sprintf("unknown layer type: %T", layer))
		}
	})
	// Drop the stale state journal in persistent database
	// and revert the head state indicator back to zero.
	rawdb.DeleteTrieJournal(batch)
	rawdb.WritePersistentStateID(batch, 0)
	if err := batch.Write(); err != nil {
		return err
	}
	// Clean up all trie histories in freezer.
	if db.freezer != nil {
		if err := db.freezer.Reset(); err != nil {
			return err
		}
	}
	db.tree = newLayerTree(newDiskLayer(root, 0, db, newDiskcache(db.dirtySize, nil, 0)))
	log.Info("Rebuilt trie database", "root", root)
	return nil
}

// Recover rollbacks the database to a specified historical point.
// The state is supported as the rollback destination only if it's
// canonical state and the corresponding trie histories are existent.
func (db *Database) Recover(root common.Hash) error {
	root = types.TrieRootHash(root)
	if !db.Recoverable(root) {
		return errStateUnrecoverable
	}
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if rollback operation is not supported.
	if db.readOnly || db.freezer == nil {
		return errors.New("state revert is non-supported")
	}
	// Iterate over all diff layers and mark them as stale.
	// Disk layer will be handled later.
	var (
		dl    *diskLayer
		batch = db.diskdb.NewBatch()
		start = time.Now()
	)
	db.tree.forEach(func(layer snapshot) {
		switch layer := layer.(type) {
		case *diskLayer:
			dl = layer
		case *diffLayer:
			layer.MarkStale()
		default:
			panic(fmt.Sprintf("unknown layer type: %T", layer))
		}
	})
	// Apply the trie histories upon the current disk layer in order.
	for {
		h, err := loadTrieHistory(db.freezer, dl.id)
		if err != nil {
			return err
		}
		dl, err = dl.revert(h)
		if err != nil {
			return err
		}
		rawdb.DeleteStateID(batch, h.Root)

		if dl.Root() == root {
			break
		}
	}
	rawdb.DeleteTrieJournal(batch)
	if err := batch.Write(); err != nil {
		return err
	}
	_, err := truncateFromHead(db.freezer, dl.id)
	if err != nil {
		return err
	}
	// Recreate the layer tree with newly created disk layer
	db.tree = newLayerTree(dl)
	log.Debug("Recovered state", "root", root, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// Recoverable returns the indicator if the specified state is recoverable.
func (db *Database) Recoverable(root common.Hash) bool {
	// Ensure the requested state is a known state.
	root = types.TrieRootHash(root)
	id, exist := rawdb.ReadStateID(db.diskdb, root)
	if !exist {
		return false
	}
	// Ensure the requested state is a canonical state.
	h, err := loadTrieHistory(db.freezer, id+1)
	if err != nil {
		return false
	}
	if h.Parent != root {
		return false
	}
	// Recoverable state must below the disk layer. The recoverable
	// state only refers the state that is currently not available,
	// but can be restored by applying trie history.
	if id >= db.tree.bottom().ID() {
		return false
	}
	// In theory all the trie histories starts from the id+1 until
	// the disk layer should be checked for presence. In practice,
	// the check is non-trivial. So optimistically believe that all
	// the trie histories above are present.
	return true
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
	db.tree.forEach(func(layer snapshot) {
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
	db.tree.forEach(func(layer snapshot) {
		if layer.Root() != types.EmptyRootHash {
			inited = true
		}
	})
	return inited
}

// SetCacheSize sets the dirty cache size to the provided value(in mega-bytes).
func (db *Database) SetCacheSize(size int) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.dirtySize = size * 1024 * 1024
	return db.tree.bottom().(*diskLayer).setCacheSize(db.dirtySize)
}

// Scheme returns the node scheme used in the database.
func (db *Database) Scheme() string {
	return rawdb.PathScheme
}
