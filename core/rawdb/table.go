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

package rawdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
)

// table is a wrapper around a database that prefixes each key access with a pre-
// configured string.
type table struct {
	db     ethdb.Database
	prefix string
}

// NewTable returns a database object that prefixes all keys with a given string.
func NewTable(db ethdb.Database, prefix string) ethdb.Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

// Close is a noop to implement the Database interface.
func (t *table) Close() error {
	return nil
}

// Has retrieves if a prefixed version of a key is present in the database.
func (t *table) Has(key []byte) (bool, error) {
	return t.db.Has(append([]byte(t.prefix), key...))
}

// Get retrieves the given prefixed key if it's present in the database.
func (t *table) Get(key []byte) ([]byte, error) {
	return t.db.Get(append([]byte(t.prefix), key...))
}

// Put inserts the given value into the database at a prefixed version of the
// provided key.
func (t *table) Put(key []byte, value []byte) error {
	return t.db.Put(append([]byte(t.prefix), key...), value)
}

// Delete removes the given prefixed key from the database.
func (t *table) Delete(key []byte) error {
	return t.db.Delete(append([]byte(t.prefix), key...))
}

// NewIterator creates a binary-alphabetical iterator over the entire keyspace
// contained within the database.
func (t *table) NewIterator() ethdb.Iterator {
	return t.NewIteratorWithPrefix(nil)
}

// NewIteratorWithPrefix creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix.
func (t *table) NewIteratorWithPrefix(prefix []byte) ethdb.Iterator {
	return t.db.NewIteratorWithPrefix(append([]byte(t.prefix), prefix...))
}

// Stat returns a particular internal stat of the database.
func (t *table) Stat(property string) (string, error) {
	return t.db.Stat(property)
}

// Compact flattens the underlying data store for the given key range. In essence,
// deleted and overwritten versions are discarded, and the data is rearranged to
// reduce the cost of operations needed to access them.
//
// A nil start is treated as a key before all keys in the data store; a nil limit
// is treated as a key after all keys in the data store. If both is nil then it
// will compact entire data store.
func (t *table) Compact(start []byte, limit []byte) error {
	// If no start was specified, use the table prefix as the first value
	if start == nil {
		start = []byte(t.prefix)
	}
	// If no limit was specified, use the first element not matching the prefix
	// as the limit
	if limit == nil {
		limit = []byte(t.prefix)
		for i := len(limit) - 1; i >= 0; i-- {
			// Bump the current character, stopping if it doesn't overflow
			limit[i]++
			if limit[i] > 0 {
				break
			}
			// Character overflown, proceed to the next or nil if the last
			if i == 0 {
				limit = nil
			}
		}
	}
	// Range correctly calculated based on table prefix, delegate down
	return t.db.Compact(start, limit)
}

// NewBatch creates a write-only database that buffers changes to its host db
// until a final write is called, each operation prefixing all keys with the
// pre-configured string.
func (t *table) NewBatch() ethdb.Batch {
	return &tableBatch{t.db.NewBatch(), t.prefix}
}

// tableBatch is a wrapper around a database batch that prefixes each key access
// with a pre-configured string.
type tableBatch struct {
	batch  ethdb.Batch
	prefix string
}

// Put inserts the given value into the batch for later committing.
func (b *tableBatch) Put(key, value []byte) error {
	return b.batch.Put(append([]byte(b.prefix), key...), value)
}

// Delete inserts the a key removal into the batch for later committing.
func (b *tableBatch) Delete(key []byte) error {
	return b.batch.Delete(append([]byte(b.prefix), key...))
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *tableBatch) ValueSize() int {
	return b.batch.ValueSize()
}

// Write flushes any accumulated data to disk.
func (b *tableBatch) Write() error {
	return b.batch.Write()
}

// Reset resets the batch for reuse.
func (b *tableBatch) Reset() {
	b.batch.Reset()
}

// Replay replays the batch contents.
func (b *tableBatch) Replay(w ethdb.Writer) error {
	return b.batch.Replay(w)
}
