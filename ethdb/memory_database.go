// Copyright 2014 The go-ethereum Authors
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

package ethdb

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// MemDatabase is an in-memory key-value store.
type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

// NewMemDatabase creates an in-memory key-value store.
func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte),
	}
}

// NewMemDatabaseWithCap creates an in-memory key-value store with a specific
// starting capacity item.
func NewMemDatabaseWithCap(size int) *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte, size),
	}
}

// Put sets the value of the key.
func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

// Has checks if a given key exists.
func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

// Get retrieves the value associated with a given key.
func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

// Keys returns a list of all the keys present in the database.
func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

// Delete removes the specified entry from the database.
func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

// Close implements the closer interface. For the memory database, this is a noop.
func (db *MemDatabase) Close() {}

// NewBatch creates a memory buffer to group together database writes and flush
// them out in one go. Batches are not atomic, just an performance optimization.
func (db *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}

// Len returns the number of entries in the database.
func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct{ k, v []byte }

type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
