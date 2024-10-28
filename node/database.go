// Copyright 2024 The go-ethereum Authors
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

package node

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/log"
)

// openOptions contains the options to apply when opening a database.
// OBS: If AncientsDirectory is empty, it indicates that no freezer is to be used.
type openOptions struct {
	Type              string // "leveldb" | "pebble"
	Directory         string // the datadir
	AncientsDirectory string // the ancients-dir
	Namespace         string // the namespace for database relevant metrics
	Cache             int    // the capacity(in megabytes) of the data caching
	Handles           int    // number of files to be open simultaneously
	ReadOnly          bool
}

// openDatabase opens both a disk-based key-value database such as leveldb or pebble, but also
// integrates it with a freezer database -- if the AncientDir option has been
// set on the provided OpenOptions.
// The passed o.AncientDir indicates the path of root ancient directory where
// the chain freezer can be opened.
func openDatabase(o openOptions) (ethdb.Database, error) {
	kvdb, err := openKeyValueDatabase(o)
	if err != nil {
		return nil, err
	}
	if len(o.AncientsDirectory) == 0 {
		return kvdb, nil
	}
	frdb, err := rawdb.NewDatabaseWithFreezer(kvdb, o.AncientsDirectory, o.Namespace, o.ReadOnly)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}

// openKeyValueDatabase opens a disk-based key-value database, e.g. leveldb or pebble.
//
//	                      type == null          type != null
//	                   +----------------------------------------
//	db is non-existent |  pebble default  |  specified type
//	db is existent     |  from db         |  specified type (if compatible)
func openKeyValueDatabase(o openOptions) (ethdb.Database, error) {
	// Reject any unsupported database type
	if len(o.Type) != 0 && o.Type != rawdb.DBLeveldb && o.Type != rawdb.DBPebble {
		return nil, fmt.Errorf("unknown db.engine %v", o.Type)
	}
	// Retrieve any pre-existing database's type and use that or the requested one
	// as long as there's no conflict between the two types
	existingDb := rawdb.PreexistingDatabase(o.Directory)
	if len(existingDb) != 0 && len(o.Type) != 0 && o.Type != existingDb {
		return nil, fmt.Errorf("db.engine choice was %v but found pre-existing %v database in specified data directory", o.Type, existingDb)
	}
	if o.Type == rawdb.DBPebble || existingDb == rawdb.DBPebble {
		log.Info("Using pebble as the backing database")
		return newPebbleDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
	}
	if o.Type == rawdb.DBLeveldb || existingDb == rawdb.DBLeveldb {
		log.Info("Using leveldb as the backing database")
		return newLevelDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
	}
	// No pre-existing database, no user-requested one either. Default to Pebble.
	log.Info("Defaulting to pebble as the backing database")
	return newPebbleDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
}

// newLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func newLevelDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	log.Info("Using LevelDB as the backing database")
	return rawdb.NewDatabase(db), nil
}

// newPebbleDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func newPebbleDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := pebble.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	return rawdb.NewDatabase(db), nil
}
