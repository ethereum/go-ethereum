// Copyright 2019 The go-ethereum Authors
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

// +build !js

// Package pogreb implements the key-value database layer based on Pogreb.
package pogreb

import (
	"fmt"
	"github.com/akrylysov/pogreb"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const ()

// Database is a persistent key-value store. Apart from basic data storage
// functionality it also supports batch writes and iterating over the keyspace in
// binary-alphabetical order.
type Database struct {
	fn string     // filename for reporting
	db *pogreb.DB // Pogreb instance

	//compTimeMeter    metrics.Meter // Meter for measuring the total time spent in database compaction
	//compReadMeter    metrics.Meter // Meter for measuring the data read during compaction
	//compWriteMeter   metrics.Meter // Meter for measuring the data written during compaction
	//writeDelayNMeter metrics.Meter // Meter for measuring the write delay number due to database compaction
	//writeDelayMeter  metrics.Meter // Meter for measuring the write delay duration due to database compaction
	//diskSizeGauge    metrics.Gauge // Gauge for tracking the size of all the levels in the database
	//diskReadMeter    metrics.Meter // Meter for measuring the effective amount of data read
	//diskWriteMeter   metrics.Meter // Meter for measuring the effective amount of data written

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	log log.Logger // Contextual logger tracking the database path
}

// New returns a wrapped LevelDB object. The namespace is the prefix that the
// metrics reporting should use for surfacing internal stats.
func New(file string, cache int) (*Database, error) {
	// Ensure we have some minimal caching and file guarantees
	//if cache < minCache {
	//	cache = minCache
	//}
	//if handles < minHandles {
	//	handles = minHandles
	//}
	file = fmt.Sprintf("%v/pogreb.db", file)
	logger := log.New("database", file)
	//logger.Info("Allocated cache and file handles", "cache", common.StorageSize(cache*1024*1024), "handles", handles)

	// Open the db and recover any potential corruptions
	db, err := pogreb.Open(file, &pogreb.Options{
		BackgroundSyncInterval: 0,
		FileSystem:             nil,
	})
	//
	//if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
	//	db, err = leveldb.RecoverFile(file, nil)
	//}
	if err != nil {
		return nil, err
	}
	// Assemble the wrapper with all the registered metrics
	pdb := &Database{
		fn:       file,
		db:       db,
		log:      logger,
		//quitChan: make(chan chan error),
	}
	//ldb.compTimeMeter = metrics.NewRegisteredMeter(namespace+"compact/time", nil)
	//ldb.compReadMeter = metrics.NewRegisteredMeter(namespace+"compact/input", nil)
	//ldb.compWriteMeter = metrics.NewRegisteredMeter(namespace+"compact/output", nil)
	//ldb.diskSizeGauge = metrics.NewRegisteredGauge(namespace+"disk/size", nil)
	//ldb.diskReadMeter = metrics.NewRegisteredMeter(namespace+"disk/read", nil)
	//ldb.diskWriteMeter = metrics.NewRegisteredMeter(namespace+"disk/write", nil)
	//ldb.writeDelayMeter = metrics.NewRegisteredMeter(namespace+"compact/writedelay/duration", nil)
	//ldb.writeDelayNMeter = metrics.NewRegisteredMeter(namespace+"compact/writedelay/counter", nil)

	// Start up the metrics gathering and return
	//go ldb.meter(metricsGatheringInterval)
	return pdb, nil
}

// Close stops the metrics collection, flushes any pending data to disk and closes
// all io accesses to the underlying key-value store.
func (db *Database) Close() error {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			db.log.Error("Metrics collection failed", "err", err)
		}
		db.quitChan = nil
	}
	return db.db.Close()
}

// Has retrieves if a key is present in the key-value store.
func (db *Database) Has(key []byte) (bool, error) {
	return db.db.Has(key)
}

// Get retrieves the given key if it's present in the key-value store.
func (db *Database) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

// Put inserts the given value into the key-value store.
func (db *Database) Put(key []byte, value []byte) error {
	return db.db.Put(key, value)
}

// Delete removes the key from the key-value store.
func (db *Database) Delete(key []byte) error {
	return db.db.Delete(key)
}

type emptyIt struct{}

func (emptyIt) Next() bool {
	return false
}

func (emptyIt) Error() error {
	return nil
}

func (emptyIt) Key() []byte {
	return nil
}

func (emptyIt) Value() []byte {
	return nil
}

func (emptyIt) Release() {

}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *Database) NewBatch() ethdb.Batch {
	return &batch{
		db: db,
	}
}


// NewIterator creates a binary-alphabetical iterator over the entire keyspace
// contained within the leveldb database.
func (db *Database) NewIterator() ethdb.Iterator {
	return emptyIt{}
}

// NewIteratorWithStart creates a binary-alphabetical iterator over a subset of
// database content starting at a particular initial key (or after, if it does
// not exist).
func (db *Database) NewIteratorWithStart(start []byte) ethdb.Iterator {
	return emptyIt{}
}

// NewIteratorWithPrefix creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix.
func (db *Database) NewIteratorWithPrefix(prefix []byte) ethdb.Iterator {
	return emptyIt{}
}

// Stat returns a particular internal stat of the database.
func (db *Database) Stat(property string) (string, error) {
	return "n/a", nil
}

// Compact flattens the underlying data store for the given key range. In essence,
// deleted and overwritten versions are discarded, and the data is rearranged to
// reduce the cost of operations needed to access them.
//
// A nil start is treated as a key before all keys in the data store; a nil limit
// is treated as a key after all keys in the data store. If both is nil then it
// will compact entire data store.
func (db *Database) Compact(start []byte, limit []byte) error {
	return nil
	//return db.db.CompactRange(util.Range{Start: start, Limit: limit})
}

// Path returns the path to the database directory.
func (db *Database) Path() string {
	return db.fn
}

// keyvalue is a key-value tuple tagged with a deletion field to allow creating
// memory-database write batches.
type keyvalue struct {
	key    []byte
	value  []byte
	delete bool
}

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	db     *Database
	writes []keyvalue
	size   int
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	b.writes = append(b.writes, keyvalue{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	b.writes = append(b.writes, keyvalue{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to the memory database.
func (b *batch) Write() error {
	//b.db.Lock()
	//defer b.db.lock.Unlock()

	for _, keyvalue := range b.writes {
		if keyvalue.delete {
			b.db.db.Delete(keyvalue.key)
			//delete(b.db.db, string(keyvalue.key))
			continue
		}
		b.db.db.Put(keyvalue.key, keyvalue.value)
		//b.db.db[string(keyvalue.key)] = keyvalue.value
	}
	return nil
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	for _, keyvalue := range b.writes {
		if keyvalue.delete {
			if err := w.Delete(keyvalue.key); err != nil {
				return err
			}
			continue
		}
		if err := w.Put(keyvalue.key, keyvalue.value); err != nil {
			return err
		}
	}
	return nil
}

// iterator can walk over the (potentially partial) keyspace of a memory key
// value store. Internally it is a deep copy of the entire iterated state,
// sorted by keys.
type iterator struct {
	inited bool
	keys   []string
	values [][]byte
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (it *iterator) Next() bool {
	// If the iterator was not yet initialized, do it now
	if !it.inited {
		it.inited = true
		return len(it.keys) > 0
	}
	// Iterator already initialize, advance it
	if len(it.keys) > 0 {
		it.keys = it.keys[1:]
		it.values = it.values[1:]
	}
	return len(it.keys) > 0
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error. A memory iterator cannot encounter errors.
func (it *iterator) Error() error {
	return nil
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (it *iterator) Key() []byte {
	if len(it.keys) > 0 {
		return []byte(it.keys[0])
	}
	return nil
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (it *iterator) Value() []byte {
	if len(it.values) > 0 {
		return it.values[0]
	}
	return nil
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (it *iterator) Release() {
	it.keys, it.values = nil, nil
}
