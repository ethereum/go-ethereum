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

// Package diskdb contains the persistent database implementation.
package diskdb

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"

	gometrics "github.com/rcrowley/go-metrics"
)

// Database is a disk backed database instance that can use various
// underlying storage engines.
type Database struct {
	folder  string         // Folder containing the database
	storage ethdb.Database // LevelDB instance

	getTimer   gometrics.Timer // Timer for measuring the database get request counts and latencies
	putTimer   gometrics.Timer // Timer for measuring the database put request counts and latencies
	delTimer   gometrics.Timer // Timer for measuring the database delete request counts and latencies
	missMeter  gometrics.Meter // Meter for measuring the missed database get requests
	readMeter  gometrics.Meter // Meter for measuring the database get request data usage
	writeMeter gometrics.Meter // Meter for measuring the database put request data usage
}

// New returns a LevelDB wrapped object.
func New(dir string, cache int, handles int) (ethdb.Database, error) {
	// Calculate the cache allowance for this particular database
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	glog.V(logger.Info).Infof("%s database: alloted %dMB cache, %d file handles", dir, cache, handles)

	// Try to open the database, in the order of storage engine preference
	/*glog.V(logger.Debug).Infof("%s database: trying to use rocksdb storage engine", dir)
	rocks, err := rocksdb.New(dir, uint64(cache)*1024*1024, handles)
	if err == nil {
		return &Database{folder: dir, storage: rocks}, nil
	}
	glog.V(logger.Warn).Infof("%s database: rocksb storage engine failed: %v", dir, err)*/

	glog.V(logger.Debug).Infof("%s database: trying to use leveldb storage engine", dir)
	level, err := leveldb.New(dir, uint64(cache)*1024*1024, handles)
	if err == nil {
		return &Database{folder: dir, storage: level}, nil
	}
	glog.V(logger.Warn).Infof("%s database: leveldb storage engine failed: %v", dir, err)

	return nil, errors.New("database open failed")
}

// Put inserts the given key/value tuple into the database.
func (db *Database) Put(key []byte, value []byte) error {
	// Measure the database put latency, if requested
	if db.putTimer != nil {
		defer db.putTimer.UpdateSince(time.Now())
	}
	// Generate the data to write to disk, update the meter and write
	if db.writeMeter != nil {
		db.writeMeter.Mark(int64(len(value)))
	}
	return db.storage.Put(key, value)
}

// Get retrieves the value of the given key if it exists.
func (db *Database) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if db.getTimer != nil {
		defer db.getTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	value, err := db.storage.Get(key)
	if err != nil {
		if db.missMeter != nil {
			db.missMeter.Mark(1)
		}
		return nil, err
	}
	// Otherwise update the actually retrieved amount of data
	if db.readMeter != nil {
		db.readMeter.Mark(int64(len(value)))
	}
	return value, nil
}

// Delete removes the key from the database if it exists.
func (db *Database) Delete(key []byte) error {
	// Measure the database delete latency, if requested
	if db.delTimer != nil {
		defer db.delTimer.UpdateSince(time.Now())
	}
	// Execute the actual operation
	return db.storage.Delete(key)
}

// Close closes the database by deallocating the underlying handle.
func (db *Database) Close() error {
	err := db.storage.Close()

	switch err {
	case nil:
		glog.V(logger.Info).Infof("closed db:", db.folder)
	default:
		glog.V(logger.Error).Infof("error closing db %s: %v", db.folder, err)
	}
	return err
}

// Meter configures the database metrics collectors.
func (db *Database) Meter(prefix string) {
	db.getTimer = metrics.NewTimer(prefix + "gets")
	db.putTimer = metrics.NewTimer(prefix + "puts")
	db.delTimer = metrics.NewTimer(prefix + "dels")
	db.missMeter = metrics.NewMeter(prefix + "misses")
	db.readMeter = metrics.NewMeter(prefix + "reads")
	db.writeMeter = metrics.NewMeter(prefix + "writes")
}

// NewBatch returns a new batch wrapping this database.
func (db *Database) NewBatch() ethdb.Batch {
	return db.storage.NewBatch()
}
