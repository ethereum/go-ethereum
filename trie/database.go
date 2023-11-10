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

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

// Config defines all necessary options for database.
type Config struct {
	Cache     int  // Memory allowance (MB) to use for caching trie nodes in memory
	Preimages bool // Flag whether the preimage of trie key is recorded

	// Testing hooks
	OnCommit func(states *triestate.Set) // Hook invoked when commit is performed
}

// backend defines the methods needed to access/update trie nodes in different
// state scheme.
type backend interface {
	// Scheme returns the identifier of used storage scheme.
	Scheme() string

	// Initialized returns an indicator if the state data is already initialized
	// according to the state scheme.
	Initialized(genesisRoot common.Hash) bool

	// Size returns the current storage size of the memory cache in front of the
	// persistent database layer.
	Size() common.StorageSize

	// Update performs a state transition by committing dirty nodes contained
	// in the given set in order to update state from the specified parent to
	// the specified root.
	Update(root common.Hash, parent common.Hash, nodes *trienode.MergedNodeSet) error

	// Commit writes all relevant trie nodes belonging to the specified state
	// to disk. Report specifies whether logs will be displayed in info level.
	Commit(root common.Hash, report bool) error

	// Close closes the trie database backend and releases all held resources.
	Close() error
}

// Database is the wrapper of the underlying backend which is shared by different
// types of node backend as an entrypoint. It's responsible for all interactions
// relevant with trie nodes and node preimages.
type Database struct {
	config    *Config          // Configuration for trie database
	diskdb    ethdb.Database   // Persistent database to store the snapshot
	cleans    *fastcache.Cache // Megabytes permitted using for read caches
	preimages *preimageStore   // The store for caching preimages
	backend   backend          // The backend for managing trie nodes
}

// prepare initializes the database with provided configs, but the
// database backend is still left as nil.
func prepare(diskdb ethdb.Database, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		cleans = fastcache.New(config.Cache * 1024 * 1024)
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
// hash-based scheme by default.
func NewDatabaseWithConfig(diskdb ethdb.Database, config *Config) *Database {
	db := prepare(diskdb, config)
	db.backend = hashdb.New(diskdb, db.cleans, mptResolver{})
	return db
}

// Reader returns a reader for accessing all trie nodes with provided state root.
// An error will be returned if the requested state is not available.
func (db *Database) Reader(blockRoot common.Hash) (Reader, error) {
	return db.backend.(*hashdb.Database).Reader(blockRoot)
}

// Update performs a state transition by committing dirty nodes contained in the
// given set in order to update state from the specified parent to the specified
// root. The held pre-images accumulated up to this point will be flushed in case
// the size exceeds the threshold.
func (db *Database) Update(root common.Hash, parent common.Hash, nodes *trienode.MergedNodeSet, states *triestate.Set) error {
	if db.config != nil && db.config.OnCommit != nil {
		db.config.OnCommit(states)
	}
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

// Initialized returns an indicator if the state data is already initialized
// according to the state scheme.
func (db *Database) Initialized(genesisRoot common.Hash) bool {
	return db.backend.Initialized(genesisRoot)
}

// Scheme returns the node scheme used in the database.
func (db *Database) Scheme() string {
	return db.backend.Scheme()
}

// Close flushes the dangling preimages to disk and closes the trie database.
// It is meant to be called when closing the blockchain object, so that all
// resources held can be released correctly.
func (db *Database) Close() error {
	db.WritePreimages()
	return db.backend.Close()
}

// WritePreimages flushes all accumulated preimages to disk forcibly.
func (db *Database) WritePreimages() {
	if db.preimages != nil {
		db.preimages.commit(true)
	}
}

// Cap iteratively flushes old but still referenced trie nodes until the total
// memory usage goes below the given threshold. The held pre-images accumulated
// up to this point will be flushed in case the size exceeds the threshold.
//
// It's only supported by hash-based database and will return an error for others.
func (db *Database) Cap(limit common.StorageSize) error {
	hdb, ok := db.backend.(*hashdb.Database)
	if !ok {
		return errors.New("not supported")
	}
	if db.preimages != nil {
		db.preimages.commit(false)
	}
	return hdb.Cap(limit)
}

// Reference adds a new reference from a parent node to a child node. This function
// is used to add reference between internal trie node and external node(e.g. storage
// trie root), all internal trie nodes are referenced together by database itself.
//
// It's only supported by hash-based database and will return an error for others.
func (db *Database) Reference(root common.Hash, parent common.Hash) error {
	hdb, ok := db.backend.(*hashdb.Database)
	if !ok {
		return errors.New("not supported")
	}
	hdb.Reference(root, parent)
	return nil
}

// Dereference removes an existing reference from a root node. It's only
// supported by hash-based database and will return an error for others.
func (db *Database) Dereference(root common.Hash) error {
	hdb, ok := db.backend.(*hashdb.Database)
	if !ok {
		return errors.New("not supported")
	}
	hdb.Dereference(root)
	return nil
}

// Node retrieves the rlp-encoded node blob with provided node hash. It's
// only supported by hash-based database and will return an error for others.
// Note, this function should be deprecated once ETH66 is deprecated.
func (db *Database) Node(hash common.Hash) ([]byte, error) {
	hdb, ok := db.backend.(*hashdb.Database)
	if !ok {
		return nil, errors.New("not supported")
	}
	return hdb.Node(hash)
}
