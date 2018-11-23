// Copyright 2018 The go-ethereum Authors
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

// Package shed provides a simple abstraction components to compose
// more complex operations on storage data organized in fields and indexes.
//
// Only type which holds logical information about swarm storage chunks data
// and metadata is IndexItem. This part is not generalized mostly for
// performance reasons.
package shed

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// The limit for LevelDB OpenFilesCacheCapacity.
const openFileLimit = 128

// DB provides abstractions over LevelDB in order to
// implement complex structures using fields and ordered indexes.
// It provides a schema functionality to store fields and indexes
// information about naming and types.
type DB struct {
	ldb *leveldb.DB
}

// NewDB constructs a new DB and validates the schema
// if it exists in database on the given path.
func NewDB(path string) (db *DB, err error) {
	ldb, err := leveldb.OpenFile(path, &opt.Options{
		OpenFilesCacheCapacity: openFileLimit,
	})
	if err != nil {
		return nil, err
	}
	db = &DB{
		ldb: ldb,
	}

	if _, err = db.getSchema(); err != nil {
		if err == leveldb.ErrNotFound {
			// save schema with initialized default fields
			if err = db.putSchema(schema{
				Fields:  make(map[string]fieldSpec),
				Indexes: make(map[byte]indexSpec),
			}); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return db, nil
}

// Put wraps LevelDB Put method to increment metrics counter.
func (db *DB) Put(key []byte, value []byte) (err error) {
	err = db.ldb.Put(key, value, nil)
	if err != nil {
		metrics.GetOrRegisterCounter("DB.putFail", nil).Inc(1)
		return err
	}
	metrics.GetOrRegisterCounter("DB.put", nil).Inc(1)
	return nil
}

// Get wraps LevelDB Get method to increment metrics counter.
func (db *DB) Get(key []byte) (value []byte, err error) {
	value, err = db.ldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			metrics.GetOrRegisterCounter("DB.getNotFound", nil).Inc(1)
		} else {
			metrics.GetOrRegisterCounter("DB.getFail", nil).Inc(1)
		}
		return nil, err
	}
	metrics.GetOrRegisterCounter("DB.get", nil).Inc(1)
	return value, nil
}

// Delete wraps LevelDB Delete method to increment metrics counter.
func (db *DB) Delete(key []byte) (err error) {
	err = db.ldb.Delete(key, nil)
	if err != nil {
		metrics.GetOrRegisterCounter("DB.deleteFail", nil).Inc(1)
		return err
	}
	metrics.GetOrRegisterCounter("DB.delete", nil).Inc(1)
	return nil
}

// NewIterator wraps LevelDB NewIterator method to increment metrics counter.
func (db *DB) NewIterator() iterator.Iterator {
	metrics.GetOrRegisterCounter("DB.newiterator", nil).Inc(1)

	return db.ldb.NewIterator(nil, nil)
}

// WriteBatch wraps LevelDB Write method to increment metrics counter.
func (db *DB) WriteBatch(batch *leveldb.Batch) (err error) {
	err = db.ldb.Write(batch, nil)
	if err != nil {
		metrics.GetOrRegisterCounter("DB.writebatchFail", nil).Inc(1)
		return err
	}
	metrics.GetOrRegisterCounter("DB.writebatch", nil).Inc(1)
	return nil
}

// Close closes LevelDB database.
func (db *DB) Close() (err error) {
	return db.ldb.Close()
}
