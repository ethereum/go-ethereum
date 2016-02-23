// Copyright 2016 The go-ethereum Authors
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

// Package leveldb contains the LevelDB based database storage engine.
package leveldb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Database is a LevelDB backed key/value store.
type Database struct {
	storage *leveldb.DB // LevelDB database instance
}

// New returns a LevelDB backed implementation of the ethdb.Database interface.
func New(dir string, cache uint64, handles int) (ethdb.Database, error) {
	// Open the database and recover any potential corruptions
	storage, err := leveldb.OpenFile(dir, &opt.Options{
		OpenFilesCacheCapacity:        handles,
		BlockCacheCapacity:            int(cache / 2),
		WriteBuffer:                   int(cache / 4), // Two of these are used internally
		CompactionTableSizeMultiplier: 2,
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		storage, err = leveldb.RecoverFile(dir, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	return &Database{
		storage: storage,
	}, nil
}

// Put inserts the given key/value tuple into the database.
func (db *Database) Put(key []byte, value []byte) error {
	return db.storage.Put(key, value, nil)
}

// Get retrieves the value of the given key if it exists.
func (db *Database) Get(key []byte) ([]byte, error) {
	value, err := db.storage.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Delete removes the key from the database if it exists.
func (db *Database) Delete(key []byte) error {
	return db.storage.Delete(key, nil)
}

// NewIterator returns an iterator over all keys of the database.
func (db *Database) NewIterator() iterator.Iterator {
	return db.storage.NewIterator(nil, nil)
}

// Close closes the database by deallocating the underlying handle.
func (db *Database) Close() error {
	return db.storage.Close()
}

// NewBatch returns a new batch wrapping this LevelDB database.
func (db *Database) NewBatch() ethdb.Batch {
	return &Batch{
		storage: db.storage,
		batch:   new(leveldb.Batch),
	}
}

// Batch is a write collector wrapping a LevelDB database.
type Batch struct {
	storage *leveldb.DB    // LevelDB storage engine to commit changes into
	batch   *leveldb.Batch // LevelDB batch to accumulate pending writes
}

// Put inserts the given key/value tuple into the batch.
func (b *Batch) Put(key, value []byte) error {
	b.batch.Put(key, value)
	return nil
}

// Commit atomically applies any batched updates to the underlying database.
func (b *Batch) Commit() error {
	return b.storage.Write(b.batch, nil)
}
