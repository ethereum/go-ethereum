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
	//"strconv"
	//"strings"
	"sync"
	"time"
	
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/ethereum/go-ethereum/common"
	
)



type BadgerDatabase struct {
	fn 				string      // filename for reporting
	db				*badger.DB 
	badgerCache		*BadgerCache
	getTimer       metrics.Timer // Timer for measuring the database get request counts and latencies
	putTimer       metrics.Timer // Timer for measuring the database put request counts and latencies
	delTimer       metrics.Timer // Timer for measuring the database delete request counts and latencies
	missMeter      metrics.Meter // Meter for measuring the missed database get requests
	readMeter      metrics.Meter // Meter for measuring the database get request data usage
	writeMeter     metrics.Meter // Meter for measuring the database put request data usage

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	log log.Logger // Contextual logger tracking the database path
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewBadgerDatabase(file string) (*BadgerDatabase, error) {
	logger := log.New("database", file)
	
	opts := badger.DefaultOptions
	opts.Dir = file
	opts.ValueDir = file
	opts.SyncWrites = false
	opts.ValueLogFileSize = 1 << 30
	opts.TableLoadingMode = options.MemoryMap
	db, err := badger.Open(opts)

	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	ret := &BadgerDatabase{
		fn:  file,
		db:  db,
		log: logger,
	}
	
	ret.badgerCache = &BadgerCache{db: ret, c: make(map[string][]byte), size: 0, limit: 500000000}
	return ret, nil
}

// Path returns the path to the database directory.
func (db *BadgerDatabase) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *BadgerDatabase) Put(key []byte, value []byte) error {

	db.badgerCache.lock.Lock()
	db.badgerCache.c[string(key)] = common.CopyBytes(value)
	db.badgerCache.size += len(value)+len(key)
	db.badgerCache.lock.Unlock()
	
	if db.badgerCache.size >= db.badgerCache.limit {
		return db.badgerCache.Flush()
	}
	
	return nil
}

func (db *BadgerDatabase) Has(key []byte) (ret bool, err error) {
	db.badgerCache.lock.RLock()
	defer db.badgerCache.lock.RUnlock()
	if db.badgerCache.c[string(key)] != nil {
		return true, nil
	}
	
	err = db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if item != nil {
			ret = true
		}
		if err == badger.ErrKeyNotFound {
			ret = false
			err = nil
		}
		return err
	})
	return ret, err
}

type BadgerCache struct {
	db		*BadgerDatabase
	c	 	map[string][]byte
	size	int
	limit	int
	lock 	sync.RWMutex
}

func (badgerCache *BadgerCache) Flush() (err error) {
	badgerCache.lock.Lock()
	defer badgerCache.lock.Unlock()
	
	txn := badgerCache.db.db.NewTransaction(true)
	for key, value := range badgerCache.c {
		putStartTime := time.Now()
		if badgerCache.db.writeMeter != nil {
			badgerCache.db.writeMeter.Mark(int64(len(value)))
		}
		err = txn.Set([]byte(key), value)
		if err == badger.ErrTxnTooBig {
		    txn.Commit(nil)
		    txn = badgerCache.db.db.NewTransaction(true)
		    err = txn.Set([]byte(key), value)
		}
		if badgerCache.db.putTimer != nil {
			badgerCache.db.putTimer.UpdateSince(putStartTime)
		}
	}
	err = txn.Commit(nil)
	log.Info("Badger flushed to disk", "badgerCache size", badgerCache.size)
	badgerCache.size = 0
	badgerCache.c = make(map[string][]byte)
	return err
}

// Get returns the given key if it's present.
func (db *BadgerDatabase) Get(key []byte) (dat []byte, err error) {
	db.badgerCache.lock.RLock()
	dat = db.badgerCache.c[string(key)]
	db.badgerCache.lock.RUnlock()
	if dat == nil {
		// Measure the database get latency, if requested
		if db.getTimer != nil {
			defer db.getTimer.UpdateSince(time.Now())
		}
		err = db.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			val, err := item.Value()
			if err != nil {
				return err
			}
			dat = common.CopyBytes(val)
			return nil
		})
		//Update the actually retrieved amount of data
		if db.readMeter != nil {
			db.readMeter.Mark(int64(len(dat)))
		}
	}
	if err != nil {
		if db.missMeter != nil {
			db.missMeter.Mark(1)
		}
		return nil, err
	}
	
	return dat, nil
}

// Delete deletes the key from the queue and database
func (db *BadgerDatabase) Delete(key []byte) error {
	
	// Execute the actual operation
	db.badgerCache.lock.Lock()
	delete(db.badgerCache.c, string(key))
	
	//TODO: also subtract len(value)
	db.badgerCache.size-=len(key)
	db.badgerCache.lock.Unlock()
	
	// Measure the database delete latency, if requested
	if db.delTimer != nil {
		defer db.delTimer.UpdateSince(time.Now())
	}
	return db.db.Update(func(txn *badger.Txn) error {
  		err := txn.Delete(key)
		if err == badger.ErrKeyNotFound {
			err = nil
		}
  		return err
	})
}

type badgerIterator struct {
	txn 				*badger.Txn
	internIterator		*badger.Iterator
	released			bool
	initialised			bool
}

func (it *badgerIterator) Release() {
	it.internIterator.Close()
	it.txn.Discard()
	it.released = true
}

func (it *badgerIterator) Released() bool {
	return it.released
}

func (it *badgerIterator) Next() bool {
	if(!it.initialised) {
		it.internIterator.Rewind()
		it.initialised = true
	} else {
		it.internIterator.Next()
	}
	return it.internIterator.Valid()
}

func (it *badgerIterator) Seek(key []byte) {
	it.internIterator.Seek(key)
}

func (it *badgerIterator) Key() []byte {
	return it.internIterator.Item().Key()
}

func (it *badgerIterator) Value() []byte {
	value, err := it.internIterator.Item().Value()
	if err != nil {
		return nil
	}
	return value
}

func (db *BadgerDatabase) NewIterator() badgerIterator {
	txn := db.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	internIterator := txn.NewIterator(opts)
	return badgerIterator{txn: txn, internIterator: internIterator, released: false, initialised: false}
}

func (db *BadgerDatabase) Close() {
	
	db.badgerCache.Flush()
	// Stop the metrics collection to avoid internal database races
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			db.log.Error("Metrics collection failed", "err", err)
		}
	}
	err := db.db.Close()
	if err == nil {
		db.log.Info("Database closed")
	} else {
		db.log.Error("Failed to close database", "err", err)
	}
}

// Meter configures the database metrics collectors and
func (db *BadgerDatabase) Meter(prefix string) {
	// Short circuit metering if the metrics system is disabled
	if !metrics.Enabled {
		return
	}
	// Initialize all the metrics collector at the requested prefix
	db.getTimer = metrics.NewRegisteredTimer(prefix+"user/gets", nil)
	db.putTimer = metrics.NewRegisteredTimer(prefix+"user/puts", nil)
	db.delTimer = metrics.NewRegisteredTimer(prefix+"user/dels", nil)
	db.missMeter = metrics.NewRegisteredMeter(prefix+"user/misses", nil)
	db.readMeter = metrics.NewRegisteredMeter(prefix+"user/reads", nil)
	db.writeMeter = metrics.NewRegisteredMeter(prefix+"user/writes", nil)

	// Create a quit channel for the periodic collector and run it
	db.quitLock.Lock()
	db.quitChan = make(chan chan error)
	db.quitLock.Unlock()

	//go db.meter(3 * time.Second)
}

func (db *BadgerDatabase) NewBatch() Batch {
	return &badgerBatch{db: db}
}

type badgerBatch struct {
	db		*BadgerDatabase
	size int
}

func (b *badgerBatch) Put(key, value []byte) error {
	b.db.badgerCache.lock.Lock()
	b.db.badgerCache.c[string(key)] = common.CopyBytes(value)
	b.db.badgerCache.size += len(value)+len(key)
	b.db.badgerCache.lock.Unlock()
	if b.db.badgerCache.size >= b.db.badgerCache.limit {
		b.db.badgerCache.Flush()
	}
	b.size += len(value)
	return nil
}

func (b *badgerBatch) Write() error {
	b.size = 0
	if b.db.badgerCache.size >= b.db.badgerCache.limit {
		return b.db.badgerCache.Flush()
	}
	return nil
}

func (b *badgerBatch) Discard() {
	b.size = 0
}

func (b *badgerBatch) ValueSize() int {
	return b.size
}

func (b *badgerBatch) Reset() {
	b.size = 0
}

type table struct {
	db     Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

type tableBatch struct {
	batch  Batch
	prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
	tb.batch.Reset()
}