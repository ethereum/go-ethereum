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
	"bytes"
	"errors"
	"fmt"
	"path"
	"sync"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// errSnapshotReadOnly is returned if the database is opened in read only mode
	// and mutation is requested.
	errSnapshotReadOnly = errors.New("read only")

	// errSnapshotStale is returned from data accessors if the underlying snapshot
	// layer had been invalidated due to the chain progressing forward far enough
	// to not maintain the layer's original state.
	errSnapshotStale = errors.New("snapshot stale")

	// errUnmatchedReverseDiff is returned if an unmatched reverse-diff is applied
	// to the database for state rollback.
	errUnmatchedReverseDiff = errors.New("reverse diff is not matched")

	// errStateUnrecoverable is returned if state is required to be reverted to
	// a destination without associated reverse diff available.
	errStateUnrecoverable = errors.New("state is unrecoverable")

	// errUnexpectedNode is returned if the requested node with specified path is
	// not hash matched or marked as deleted.
	errUnexpectedNode = errors.New("unexpected node")
)

// snapshot is the extension of the Reader interface which implemented by all
// layers(disklayer, disklayerSnapshot, difflayer). This interface supports
// some additional methods for internal usage.
type snapshot interface {
	Reader

	// Root returns the root hash for which this snapshot was made.
	Root() common.Hash

	// Parent returns the subsequent layer of a snapshot, or nil if the base was
	// reached.
	//
	// Note, the method is an internal helper to avoid type switching between the
	// disk and diff layers. There is no locking involved.
	Parent() snapshot

	// Update creates a new layer on top of the existing snapshot diff tree with
	// the given dirty trie node set. All dirty nodes are indexed with the storage
	// format key. The deleted trie nodes are also included with the nil as the
	// node object.
	//
	// Note, the maps are retained by the method to avoid copying everything.
	Update(blockRoot common.Hash, blockNumber uint64, nodes map[string]*nodeWithPreValue) *diffLayer

	// Journal commits an entire diff hierarchy to disk into a single journal entry.
	// This is meant to be used during shutdown to persist the snapshot without
	// flattening everything down (bad for reorgs).
	Journal(buffer *bytes.Buffer) error

	// Stale returns whether this layer has become stale (was flattened across) or
	// if it's still live.
	Stale() bool

	// ID returns the id of associated reverse diff.
	ID() uint64
}

// snapDatabase is a multiple-layered structure for maintaining in-memory trie
// nodes. It consists of one persistent base layer backed by a key-value store,
// on top of which arbitrarily many in-memory diff layers are topped. The memory
// diffs can form a tree with branching, but the disk layer is singleton and
// common to all. If a reorg goes deeper than the disk layer, a batch of reverse
// diffs should be applied. The deepest reorg can be handled depends on the amount
// of reverse diffs tracked in the disk.
type snapDatabase struct {
	// readOnly is the flag whether the mutation is allowed to be applied.
	// It will be set automatically when the database is journaled(closed)
	// during the shutdown to reject all following unexpected mutations.
	readOnly bool
	config   *Config
	diskdb   ethdb.Database
	cleans   *fastcache.Cache
	tree     *layerTree     // The group for all known layers
	freezer  *rawdb.Freezer // Freezer for storing reverse diffs, nil possible in tests
	lock     sync.RWMutex
}

// openSnapDatabase attempts to load an already existing snapshot from a
// persistent key-value store (with a number of memory layers from a journal).
// If the journal is not matched with the base persistent layer, all the
// recorded diff layers are discarded.
func openSnapDatabase(diskdb ethdb.Database, readOnly bool, cleans *fastcache.Cache, config *Config) *snapDatabase {
	db := &snapDatabase{
		readOnly: readOnly,
		config:   config,
		diskdb:   diskdb,
		cleans:   cleans,
	}
	// Open the freezer for reverse diffs if the passed database contains an
	// ancient store. Otherwise, all the relevant functionalities are disabled.
	if ancient, err := diskdb.AncientDatadir(); err == nil && ancient != "" {
		freezer, err := openFreezer(path.Join(ancient, "rdiffs"), readOnly)
		if err != nil {
			log.Crit("Failed to open reverse diff freezer", "err", err)
		}
		db.freezer = freezer
	}
	// Construct the layer tree by resolving the in-disk singleton state
	// and in-memory layer journal.
	db.tree = newLayerTree(db.loadSnapshot(diskdb, cleans))

	// Truncate the extra reverse diffs above in freezer in case it's not
	// aligned with the disk layer.
	if !readOnly && db.freezer != nil {
		layer := db.tree.bottom().(*diskLayer)
		truncateDiffs(db.freezer, diskdb, layer.diffid)
	}
	return db
}

// GetReader retrieves a snapshot belonging to the given state root.
func (db *snapDatabase) GetReader(root common.Hash) Reader {
	return db.tree.get(root)
}

// Commit adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all). Apart
// from that this function will flatten the extra diff layers at bottom into disk
// to only keep 128 diff layers in memory.
func (db *snapDatabase) Commit(root common.Hash, parentRoot common.Hash, result *NodeSet) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	if err := db.tree.add(root, parentRoot, result.nodes); err != nil {
		return err
	}
	var limit uint64
	if db.config != nil {
		limit = db.config.StateLimit
	}
	// Keep 128 diff layers in the memory, persistent layer is 129th.
	// - head layer is paired with HEAD state
	// - head-1 layer is paired with HEAD-1 state
	// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
	// - head-128 layer(disk layer) is paired with HEAD-128 state
	return db.tree.cap(root, 128, db.freezer, limit)
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards. It can be used alone and mostly for test purposes.
func (db *snapDatabase) Cap(root common.Hash, layers int) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	var limit uint64
	if db.config != nil {
		limit = db.config.StateLimit
	}
	return db.tree.cap(root, layers, db.freezer, limit)
}

// Close commits an entire diff hierarchy to disk into a single journal entry.
// This is meant to be used during shutdown to persist the snapshot without
// flattening everything down (bad for reorgs). And this function will mark the
// database as read-only to prevent all following mutation to disk.
func (db *snapDatabase) Close(root common.Hash) error {
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
	_, diskroot := rawdb.ReadTrieNode(db.diskdb, EncodeStorageKey(common.Hash{}, nil))
	if diskroot == (common.Hash{}) {
		diskroot = emptyRoot
	}
	// Secondly write out the disk layer root, ensure the
	// diff journal is continuous with disk.
	if err := rlp.Encode(journal, diskroot); err != nil {
		return err
	}
	// Finally write out the journal of each layer in reverse order.
	if err := snap.(snapshot).Journal(journal); err != nil {
		return err
	}
	// Store the journal into the database and return
	rawdb.WriteTrieJournal(db.diskdb, journal.Bytes())

	// Set the db in read only mode to reject all following mutations
	db.readOnly = true
	log.Info("Stored journal in triedb", "disk", diskroot, "size", common.StorageSize(journal.Len()))

	// Close the reverse diff freezer if it's enabled.
	if db.freezer == nil {
		return nil
	}
	return db.freezer.Close()
}

// Reset wipes all available journal from the persistent database and discard
// all caches and diff layers, rebuilds the database by using the specified
// disk layer.
func (db *snapDatabase) Reset(root common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	root = convertEmpty(root)

	// Ensure the specified state is present in disk, try to avoid
	// destroying everything.
	_, hash := rawdb.ReadTrieNode(db.diskdb, EncodeStorageKey(common.Hash{}, nil))
	if hash != root {
		if root != emptyRoot {
			blob := rawdb.ReadLegacyTrieNode(db.diskdb, root)
			if len(blob) == 0 {
				return errors.New("state is non-existent")
			}
		}
		// Fallback to legacy scheme state, wipe out all path-based
		// trie nodes. The trick is applied here that only root node
		// is nuked out, it can ensure all the children are dangling
		// and shouldn't be accessible anymore.
		rawdb.DeleteTrieNode(db.diskdb, EncodeStorageKey(common.Hash{}, nil))
	}
	// Drop the stale state journal in persistent database.
	rawdb.DeleteTrieJournal(db.diskdb)

	// Iterate over all layers and mark them as stale
	db.tree.forEach(func(_ common.Hash, layer snapshot) bool {
		switch layer := layer.(type) {
		case *diskLayer:
			// Layer should be inactive now, mark it as stale
			layer.MarkStale()
		case *diffLayer:
			// If the layer is a simple diff, simply mark as stale
			layer.MarkStale()
		default:
			panic(fmt.Sprintf("unknown layer type: %T", layer))
		}
		return true
	})
	// Clean up all reverse diffs in freezer.
	var diffid uint64
	if db.freezer != nil {
		diffid, _ = db.freezer.Tail()
		rawdb.WriteReverseDiffHead(db.diskdb, diffid)
		truncateDiffs(db.freezer, db.diskdb, diffid)
	}
	db.tree = newLayerTree(newDiskLayer(root, diffid, db.cleans, newDiskcache(nil, 0), db.diskdb))
	log.Info("Rebuild triedb", "root", root, "diffid", diffid)
	return nil
}

// revert applies the reverse diffs to the database by reverting the disk layer
// content. This function assumes the lock in db is already held.
func (db *snapDatabase) revert(diffid uint64, diff *reverseDiff) error {
	ndl, err := db.tree.bottom().(*diskLayer).revert(diff, diffid)
	if err != nil {
		return err
	}
	// Delete the lookup first to mark this reverse diff invisible.
	rawdb.DeleteReverseDiffLookup(db.diskdb, diff.Parent)

	// Truncate the reverse diff from the freezer in the last step
	_, err = truncateFromHead(db.freezer, db.diskdb, diffid-1)
	if err != nil {
		return err
	}
	// Recreate the layer tree with newly created disk layer
	db.tree = newLayerTree(ndl)
	return nil
}

// Recover rollbacks the database to a specified historical point.
// The state is supported as the rollback destination only if it's
// canonical state and the corresponding reverse diffs are existent.
func (db *snapDatabase) Recover(target common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errSnapshotReadOnly
	}
	// Short circuit if the whole state rollback functionality is disabled.
	if db.freezer == nil {
		return errors.New("state revert is disabled")
	}
	// Ensure the destination is recoverable
	target = convertEmpty(target)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, target)
	if id == nil {
		return errStateUnrecoverable
	}
	current := db.tree.bottom().(*diskLayer).ID()
	if *id > current {
		return fmt.Errorf("%w dest: %d head: %d", errors.New("immature state"), *id, current)
	}
	// Clean up the database, wipe all existent diff layers and journal as well.
	rawdb.DeleteTrieJournal(db.diskdb)

	// Iterate over all diff layers and mark them as stale. Disk layer will be
	// handled later.
	db.tree.forEach(func(hash common.Hash, layer snapshot) bool {
		dl, ok := layer.(*diffLayer)
		if ok {
			dl.MarkStale()
		}
		return true
	})
	// Apply the reverse diffs with the given order.
	for current >= *id {
		diff, err := loadReverseDiff(db.freezer, current)
		if err != nil {
			return err
		}
		if err := db.revert(current, diff); err != nil {
			return err
		}
		current -= 1
	}
	return nil
}

// Recoverable returns the indicator if the specified state is enabled to be recovered.
func (db *snapDatabase) Recoverable(root common.Hash) bool {
	// In theory all the reverse diffs starts from the given id until
	// the disk layer should be checked for presence. In practice, the
	// check is too expensive. So optimistically believe that all the
	// reverse diffs are present.
	root = convertEmpty(root)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, root)
	if id == nil {
		return false
	}
	return db.tree.bottom().(*diskLayer).ID() >= *id
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *snapDatabase) Size() (size common.StorageSize) {
	db.tree.forEach(func(_ common.Hash, layer snapshot) bool {
		if diff, ok := layer.(*diffLayer); ok {
			size += common.StorageSize(diff.memory)
		}
		if disk, ok := layer.(*diskLayer); ok {
			size += disk.size()
		}
		return true
	})
	return size
}
