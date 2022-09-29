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
	"runtime"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// NodeReader warps all the necessary functions for accessing trie node.
type NodeReader interface {
	// GetReader returns a reader for accessing all trie nodes with provided
	// state root. Nil is returned in case the state is not available.
	GetReader(root common.Hash) Reader
}

// Config defines all necessary options for database.
type Config struct {
	Cache     int    // Memory allowance (MB) to use for caching trie nodes in memory
	Journal   string // Journal of clean cache to survive node restarts
	Preimages bool   // Flag whether the preimage of trie key is recorded

	// Configs for experimental path-based scheme, not used yet.
	Scheme     string // Disk scheme for reading/writing trie nodes, hash-based as default
	StateLimit uint64 // Number of recent blocks to maintain state history for
	DirtySize  int    // Maximum memory allowance (MB) for caching dirty nodes
	ReadOnly   bool   // Flag whether the database is opened in read only mode.
}

// nodeBackend defines the methods needed to access/update trie nodes in
// different state scheme. It's implemented by hashDatabase and snapDatabase.
type nodeBackend interface {
	// GetReader returns a reader for accessing all trie nodes with provided
	// state root. Nil is returned in case the state is not available.
	GetReader(root common.Hash) Reader

	// Update performs a state transition from specified parent to root by committing
	// dirty nodes provided in the nodeset.
	Update(root common.Hash, parent common.Hash, nodes *MergedNodeSet) error

	// Commit writes all relevant trie nodes belonging to the specified state to disk.
	// Report specifies whether logs will be displayed in info level.
	Commit(root common.Hash, report bool) error

	// IsEmpty returns an indicator if the node database is empty.
	IsEmpty() bool

	// Size returns the current storage size of the memory cache in front of the
	// persistent database layer.
	Size() common.StorageSize

	// Scheme returns the node scheme used in the database.
	Scheme() string

	// Close closes the trie database backend and releases all held resources.
	Close() error
}

// Database is the wrapper of the underlying nodeBackend which is shared by
// different types of nodeBackends as an entrypoint. It's responsible for all
// interactions relevant with trie nodes and the node preimages.
type Database struct {
	config    *Config          // Configuration for trie database.
	diskdb    ethdb.Database   // Persistent database to store the snapshot
	cleans    *fastcache.Cache // Megabytes permitted using for read caches
	preimages *preimageStore   // The store for caching preimages
	backend   nodeBackend      // The backend for managing trie nodes
}

// prepare initializes the database with provided configs, but the
// database backend is still left as nil.
func prepare(diskdb ethdb.Database, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		if config.Journal == "" {
			cleans = fastcache.New(config.Cache * 1024 * 1024)
		} else {
			cleans = fastcache.LoadFromFileOrNew(config.Journal, config.Cache*1024*1024)
		}
	}
	var preimages *preimageStore
	if config != nil && config.Preimages {
		preimages = newPreimageStore(diskdb)
	}
	return &Database{
		config:    config,
		diskdb:    diskdb,
		cleans:    cleans,
		preimages: preimages,
	}
}

// NewDatabase initializes the trie database with default settings, namely
// the legacy hash-based scheme is used by default.
func NewDatabase(diskdb ethdb.Database) *Database {
	return NewDatabaseWithConfig(diskdb, nil)
}

// NewDatabaseWithConfig initializes the trie database with provided configs.
// The path-based scheme is not activated yet, always initialized with legacy
// hash-based scheme.
func NewDatabaseWithConfig(diskdb ethdb.Database, config *Config) *Database {
	db := prepare(diskdb, config)
	db.backend = openHashDatabase(diskdb, db.cleans)
	return db
}

// GetReader returns a reader for accessing all trie nodes with provided
// state root. Nil is returned in case the state is not available.
func (db *Database) GetReader(blockRoot common.Hash) Reader {
	return db.backend.GetReader(blockRoot)
}

// Update performs a state transition by committing dirty nodes contained
// in the given set in order to update state from the specified parent to
// the specified root. The held pre-images accumulated up to this point
// will be flushed in case the size exceeds the threshold.
func (db *Database) Update(root common.Hash, parent common.Hash, nodes *MergedNodeSet) error {
	if db.preimages != nil {
		db.preimages.commit(false)
	}
	return db.backend.Update(root, parent, nodes)
}

// Commit iterates over all the children of a particular node, writes them out
// to disk. As a side effect, all pre-images accumulated up to this point are
// also written.
func (db *Database) Commit(root common.Hash, report bool) error {
	if db.preimages != nil {
		db.preimages.commit(true)
	}
	return db.backend.Commit(root, report)
}

// Size returns the storage size of dirty trie nodes in front of the persistent
// database and the size of cached preimages.
func (db *Database) Size() (common.StorageSize, common.StorageSize) {
	var (
		storages  common.StorageSize
		preimages common.StorageSize
	)
	storages = db.backend.Size()
	if db.preimages != nil {
		preimages = db.preimages.size()
	}
	return storages, preimages
}

// IsEmpty returns an indicator if the node database is empty.
func (db *Database) IsEmpty() bool {
	return db.backend.IsEmpty()
}

// Scheme returns the node scheme used in the database.
func (db *Database) Scheme() string {
	return db.backend.Scheme()
}

// Close flushes the dangling preimages to disk and closes the trie database.
// It is meant to be called when closing the blockchain object, so that all
// resources held can be released correctly.
func (db *Database) Close() error {
	if db.preimages != nil {
		db.preimages.commit(true)
	}
	return db.backend.Close()
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
