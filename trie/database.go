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
	"runtime"
	"sync"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	triedbCleanHitMeter   = metrics.NewRegisteredMeter("trie/triedb/clean/hit", nil)
	triedbCleanMissMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/miss", nil)
	triedbCleanReadMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/read", nil)
	triedbCleanWriteMeter = metrics.NewRegisteredMeter("trie/triedb/clean/write", nil)

	triedbFallbackHitMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/hit", nil)
	triedbFallbackReadMeter = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/read", nil)

	triedbDirtyHitMeter   = metrics.NewRegisteredMeter("trie/triedb/dirty/hit", nil)
	triedbDirtyMissMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/miss", nil)
	triedbDirtyReadMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/read", nil)
	triedbDirtyWriteMeter = metrics.NewRegisteredMeter("trie/triedb/dirty/write", nil)

	triedbDirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("trie/triedb/dirty/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	triedbCommitTimeTimer  = metrics.NewRegisteredTimer("trie/triedb/commit/time", nil)
	triedbCommitNodesMeter = metrics.NewRegisteredMeter("trie/triedb/commit/nodes", nil)
	triedbCommitSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/commit/size", nil)

	triedbDiffLayerSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/diff/size", nil)
	triedbDiffLayerNodesMeter = metrics.NewRegisteredMeter("trie/triedb/diff/nodes", nil)

	triedbReverseDiffTimeTimer = metrics.NewRegisteredTimer("trie/triedb/reversediff/time", nil)
	triedbReverseDiffSizeMeter = metrics.NewRegisteredMeter("trie/triedb/reversediff/size", nil)

	// ErrSnapshotReadOnly is returned if the database is opened in read only mode
	// and mutation is requested.
	ErrSnapshotReadOnly = errors.New("read only")

	// errSnapshotStale is returned from data accessors if the underlying snapshot
	// layer had been invalidated due to the chain progressing forward far enough
	// to not maintain the layer's original state.
	errSnapshotStale = errors.New("snapshot stale")

	// errUnexpectedNode is returned if the requested node with specified path is
	// not hash matched or marked as deleted.
	errUnexpectedNode = errors.New("unexpected node")

	// errSnapshotCycle is returned if a snapshot is attempted to be inserted
	// that forms a cycle in the snapshot tree.
	errSnapshotCycle = errors.New("snapshot cycle")

	// errUnmatchedReverseDiff is returned if an unmatched reverse-diff is applied
	// to the database for state rollback.
	errUnmatchedReverseDiff = errors.New("reverse diff is not matched")

	// errStateUnrecoverable is returned if state is required to be reverted to
	// a destination without associated reverse diff available.
	errStateUnrecoverable = errors.New("state is unrecoverable")

	// errImmatureState is returned if state is required to be reverted to an
	// immature destination.
	errImmatureState = errors.New("immature state")
)

// Snapshot represents the functionality supported by a snapshot storage layer.
type Snapshot interface {
	// Root returns the root hash for which this snapshot was made.
	Root() common.Hash

	// NodeBlob retrieves the RLP-encoded trie node blob associated with
	// a particular key and the corresponding node hash. The passed key
	// should be encoded in storage format. No error will be returned if
	// the node is not found.
	NodeBlob(storage []byte, hash common.Hash) ([]byte, error)
}

// snapshot is the internal version of the snapshot data layer that supports some
// additional methods compared to the public API.
type snapshot interface {
	Snapshot

	// Node retrieves the trie node associated with a particular key and the
	// corresponding node hash. The returned node is in a wrapper through which
	// callers can obtain the RLP-format or canonical node representation easily.
	// No error will be returned if the node is not found.
	Node(storage []byte, hash common.Hash) (*cachedNode, error)

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

// StateReader wraps the Snapshot method of a backing trie database.
type StateReader interface {
	// Snapshot retrieves a snapshot belonging to the given state root.
	Snapshot(root common.Hash) Snapshot
}

// StateWriter wraps the Update and Cap methods of a backing trie database.
type StateWriter interface {
	// Update adds a new snapshot into the tree, if that can be linked to an existing
	// old parent. It is disallowed to insert a disk layer (the origin of all).
	// The passed keys must all be encoded in the **storage** format.
	Update(root common.Hash, parentRoot common.Hash, nodes map[string]*nodeWithPreValue) error

	// Cap traverses downwards the snapshot tree from a head block hash until the
	// number of allowed layers are crossed. All layers beyond the permitted number
	// are flattened downwards.
	Cap(root common.Hash, layers int) error
}

// StateDatabase wraps all the necessary functions for accessing and persisting
// nodes. It's implemented by Database and DatabaseSnapshot.
type StateDatabase interface {
	StateReader
	StateWriter

	// DiskDB returns the underlying key-value disk store.
	DiskDB() ethdb.KeyValueStore
}

// Config defines all necessary options for database.
type Config struct {
	Cache     int    // Memory allowance (MB) to use for caching trie nodes in memory
	Journal   string // Journal of clean cache to survive node restarts
	Preimages bool   // Flag whether the preimage of trie key is recorded
	ReadOnly  bool   // Flag whether the database is opened in read only mode.
}

// Database is a multiple-layered structure for maintaining in-memory trie nodes.
// It consists of one persistent base layer backed by a key-value store, on top
// of which arbitrarily many in-memory diff layers are topped. The memory diffs
// can form a tree with branching, but the disk layer is singleton and common to
// all. If a reorg goes deeper than the disk layer, a batch of reverse diffs should
// be applied. The deepest reorg can be handled depends on the amount of reverse
// diffs tracked in the disk.
type Database struct {
	// readOnly is the flag whether the mutation is allowed to be applied.
	// It will be set automatically when the database is journaled during
	// the shutdown to reject all following unexpected mutations.
	readOnly bool
	config   *Config
	lock     sync.RWMutex     // Lock to prevent concurrent mutations and readOnly flag
	diskdb   ethdb.Database   // Persistent database to store the snapshot
	cleans   *fastcache.Cache // Megabytes permitted using for read caches
	tree     *layerTree       // The group for all known layers
	preimage *preimageStore   // The store for caching preimages
}

// NewDatabase attempts to load an already existing snapshot from a persistent
// key-value store (with a number of memory layers from a journal). If the journal
// is not matched with the base persistent layer, all the recorded diff layers
// are discarded.
func NewDatabase(diskdb ethdb.Database, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		if config.Journal == "" {
			cleans = fastcache.New(config.Cache * 1024 * 1024)
		} else {
			cleans = fastcache.LoadFromFileOrNew(config.Journal, config.Cache*1024*1024)
		}
	}
	readOnly := config != nil && config.ReadOnly
	db := &Database{
		readOnly: readOnly,
		config:   config,
		diskdb:   diskdb,
		cleans:   cleans,
		tree:     newLayerTree(loadSnapshot(diskdb, cleans, readOnly)),
	}
	if config == nil || config.Preimages {
		db.preimage = newPreimageStore(diskdb)
	}
	return db
}

// InsertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will NOT make a copy of the slice, only use if the
// preimage will NOT be changed later on.
func (db *Database) InsertPreimage(preimages map[common.Hash][]byte) {
	if db.preimage == nil {
		return
	}
	db.preimage.insertPreimage(preimages)
}

// Preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *Database) Preimage(hash common.Hash) []byte {
	if db.preimage == nil {
		return nil
	}
	return db.preimage.preimage(hash)
}

// Snapshot retrieves a snapshot belonging to the given block root.
func (db *Database) Snapshot(blockRoot common.Hash) Snapshot {
	return db.tree.get(blockRoot)
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
func (db *Database) Update(root common.Hash, parentRoot common.Hash, nodes map[string]*nodeWithPreValue) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	return db.tree.add(root, parentRoot, nodes)
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards.
func (db *Database) Cap(root common.Hash, layers int) error {
	// Hold the lock to prevent concurrent mutations.
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	if db.preimage != nil {
		db.preimage.commit()
	}
	return db.tree.cap(root, layers)
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
		return ErrSnapshotReadOnly
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
	log.Info("Stored snapshot journal in triedb", "disk", diskroot, "size", common.StorageSize(journal.Len()))
	return nil
}

// Clean wipes all available journal from the persistent database and discard
// all caches and diff layers. Using the given root to create a new disk layer.
func (db *Database) Clean(root common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	// TODO check if the given root node is existent
	// before applying any mutations.
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
	// Delete all remaining reverse diffs in disk
	head, err := purgeReverseDiffs(db.diskdb)
	if err != nil {
		return err
	}
	db.tree = newLayerTree(newDiskLayer(root, head, db.cleans, newDiskcache(nil, 0), db.diskdb))
	log.Info("Rebuild triedb", "root", root, "diffid", head)
	return nil
}

// revert applies the reverse diffs to the database by reverting the disk layer
// content. This function assumes the lock in db is already held.
func (db *Database) revert(diffid uint64, diff *reverseDiff) error {
	ndl, err := db.disklayer().revert(diff, diffid)
	if err != nil {
		return err
	}
	// Delete the lookup first to mark this reverse diff invisible.
	rawdb.DeleteReverseDiffLookup(db.diskdb, diff.Parent)

	// Truncate the reverse diff from the freezer in the last step
	_, err = truncateFromHead(db.diskdb, diffid-1)
	if err != nil {
		return err
	}
	// Recreate the disk layer with newly created clean cache
	db.tree = newLayerTree(ndl)
	return nil
}

// Rollback rollbacks the database to a specified historical point.
// The state is supported as the rollback destination only if it's
// canonical state and the corresponding reverse diffs are existent.
func (db *Database) Rollback(target common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Ensure the destination is recoverable
	target = convertEmpty(target)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, target)
	if id == nil {
		return errStateUnrecoverable
	}
	current := db.disklayer().ID()
	if *id > current {
		return fmt.Errorf("%w dest: %d head: %d", errImmatureState, *id, current)
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
		diff, err := loadReverseDiff(db.diskdb, current)
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

// StateRecoverable returns the indicator if the specified state is enabled to be recovered.
func (db *Database) StateRecoverable(root common.Hash) bool {
	root = convertEmpty(root)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, root)
	if id == nil {
		return false
	}
	if db.disklayer().ID() < *id {
		return false
	}
	// In theory all the reverse diffs starts from the given id until
	// the disk layer should be checked for presence. In practice, the
	// check is too expensive. So optimistically believe that all the
	// reverse diffs are present.
	return true
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.KeyValueStore {
	return db.diskdb
}

// disklayer is an internal helper function to return the disk layer.
// The lock of trieDB is assumed to be held already.
func (db *Database) disklayer() (ret *diskLayer) {
	db.tree.forEach(func(hash common.Hash, layer snapshot) bool {
		if dl, ok := layer.(*diskLayer); ok {
			ret = dl
			return false
		}
		return true
	})
	return ret
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *Database) Size() common.StorageSize {
	var nodes common.StorageSize
	db.tree.forEach(func(_ common.Hash, layer snapshot) bool {
		if diff, ok := layer.(*diffLayer); ok {
			nodes += common.StorageSize(diff.memory)
		}
		if disk, ok := layer.(*diskLayer); ok {
			nodes += disk.size()
		}
		return true
	})
	return nodes
}

// saveCache saves clean state cache to given directory path
// using specified CPU cores.
func (db *Database) saveCache(dir string, threads int) error {
	if db.cleans == nil {
		return nil
	}
	log.Info("Writing clean trie cache to disk", "path", dir, "threads", threads)

	start := time.Now()
	err := db.cleans.SaveToFileConcurrent(dir, threads)
	if err != nil {
		log.Error("Failed to persist clean trie cache", "error", err)
		return err
	}
	log.Info("Persisted the clean trie cache", "path", dir, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// SaveCache atomically saves fast cache data to the given dir using all
// available CPU cores.
func (db *Database) SaveCache(dir string) error {
	return db.saveCache(dir, runtime.GOMAXPROCS(0))
}

// SaveCachePeriodically atomically saves fast cache data to the given dir with
// the specified interval. All dump operation will only use a single CPU core.
func (db *Database) SaveCachePeriodically(dir string, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.saveCache(dir, 1)
		case <-stopCh:
			return
		}
	}
}

// Config returns the configures used by db.
func (db *Database) Config() *Config {
	return db.config
}

// IsEmpty returns an indicator if the state database is empty
func (db *Database) IsEmpty() bool {
	var nonEmpty bool
	db.tree.forEach(func(_ common.Hash, layer snapshot) bool {
		if layer.Root() != emptyRoot {
			nonEmpty = true
			return false
		}
		return true
	})
	return !nonEmpty
}
