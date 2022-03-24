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
	"runtime"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// Reader wraps the Node and NodeBlob method of a backing trie store.
type Reader interface {
	// Node retrieves the trie node associated with a particular trie node path
	// and the corresponding node hash. The returned node is in a wrapper through
	// which callers can obtain the RLP-format or canonical node representation
	// easily.
	// No error will be returned if the node is not found.
	Node(path []byte, hash common.Hash) (*cachedNode, error)

	// NodeBlob retrieves the RLP-encoded trie node blob associated with
	// a particular trie node path and the corresponding node hash.
	// No error will be returned if the node is not found.
	NodeBlob(path []byte, hash common.Hash) ([]byte, error)
}

// NodeReader warps all the necessary functions for accessing trie node.
type NodeReader interface {
	// GetReader returns a reader for accessing all trie nodes whose state root
	// is the specified root. Nil is returned in case the state is not available.
	GetReader(root common.Hash) Reader
}

// NodeDatabase wraps all the necessary functions for accessing and persisting
// nodes. It's implemented by Database and DatabaseSnapshot.
type NodeDatabase interface {
	NodeReader

	// Commit performs a state transition by committing dirty nodes contained
	// in the given set in order to update state from the specified parent to
	// the given root.
	Commit(root common.Hash, parent common.Hash, nodes *NodeSet) error

	// DiskDB returns the underlying disk store.
	DiskDB() ethdb.Database
}

// Config defines all necessary options for database.
type Config struct {
	Cache      int    // Memory allowance (MB) to use for caching trie nodes in memory
	Journal    string // Journal of clean cache to survive node restarts
	Preimages  bool   // Flag whether the preimage of trie key is recorded
	ReadOnly   bool   // Flag whether the database is opened in read only mode.
	StateLimit uint64 // Number of recent blocks to maintain state history for
	Legacy     bool   // Flag whether legacy state scheme is used(hash-based)
}

// nodeBackend defines the methods needed to access/update trie nodes in
// different state scheme.
type nodeBackend interface {
	// GetReader returns a reader for accessing all trie nodes whose state root
	// is the specified root.
	GetReader(root common.Hash) Reader

	// Commit performs a state transition by committing dirty nodes contained
	// in the given result object in order to update state from the specified
	// parent root to the specified root
	Commit(root common.Hash, parentRoot common.Hash, commit *NodeSet) error
}

// Database is a multiple-layered structure for maintaining in-memory trie nodes.
// It consists of one persistent base layer backed by a key-value store, on top
// of which arbitrarily many in-memory diff layers are topped. The memory diffs
// can form a tree with branching, but the disk layer is singleton and common to
// all. If a reorg goes deeper than the disk layer, a batch of reverse diffs should
// be applied. The deepest reorg can be handled depends on the amount of reverse
// diffs tracked in the disk.
type Database struct {
	config   *Config          // Configuration for trie database.
	diskdb   ethdb.Database   // Persistent database to store the snapshot
	cleans   *fastcache.Cache // Megabytes permitted using for read caches
	preimage *preimageStore   // The store for caching preimages
	backend  nodeBackend      // The backend for managing trie nodes
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
	var preimage *preimageStore
	if config != nil && config.Preimages {
		preimage = newPreimageStore(diskdb)
	}
	db := &Database{
		config:   config,
		diskdb:   diskdb,
		cleans:   cleans,
		preimage: preimage,
	}
	if config != nil && config.Legacy {
		db.backend = openHashDatabase(diskdb, config != nil && config.ReadOnly, cleans)
	} else {
		db.backend = openSnapDatabase(diskdb, config != nil && config.ReadOnly, cleans, config)
	}
	return db
}

// GetReader returns a reader for accessing all trie nodes whose state root
// is the specified root. Nil is returned if the state is not existent.
func (db *Database) GetReader(blockRoot common.Hash) Reader {
	return db.backend.GetReader(blockRoot)
}

// Commit performs a state transition by committing dirty nodes contained
// in the given set in order to update state from the specified parent to
// the specified root.
func (db *Database) Commit(root common.Hash, parentRoot common.Hash, nodes *NodeSet) error {
	return db.backend.Commit(root, parentRoot, nodes)
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards. It's only supported by snap database and it's noop
// for hash database.
func (db *Database) Cap(root common.Hash, layers int) error {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Cap(root, layers)
	}
	return nil // not supported
}

// Recover rollbacks the database to a specified historical point. The state is
// supported as the rollback destination only if it's canonical state and the
// corresponding reverse diffs are existent. It's only supported by snap database
// and it's noop for hash database.
func (db *Database) Recover(target common.Hash) error {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Recover(target)
	}
	return nil // not supported
}

// Recoverable returns the indicator if the specified state is enabled to be
// recovered. It's only supported by snap database.
func (db *Database) Recoverable(root common.Hash) bool {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Recoverable(root)
	}
	return false // not supported
}

// Close commits an entire diff hierarchy to disk into a single journal entry.
// This is meant to be used during shutdown to persist the snapshot without
// flattening everything down (bad for reorgs). And this function will mark the
// database as read-only to prevent all following mutation to disk.
// It's only supported by snap database and it's noop for hash database.
func (db *Database) Close(root common.Hash) error {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Close(root)
	}
	return nil // not supported
}

// Reset wipes all available journal from the persistent database and discard
// all caches and diff layers. Using the given root to create a new disk layer.
// It's only supported by snap database and it's noop for hash database.
func (db *Database) Reset(root common.Hash) error {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Reset(root)
	}
	return nil // not supported
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer. It's only supported by snap database and will
// always return 0 for other implementations.
func (db *Database) Size() (size common.StorageSize) {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.Size()
	}
	return 0 // not supported
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.Database {
	return db.diskdb
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

// CloseFreezer closes the freezer instance of snap database. It's only for tests.
func (db *Database) CloseFreezer() error {
	if snapDB, ok := db.backend.(*snapDatabase); ok {
		return snapDB.freezer.Close()
	}
	return nil
}
