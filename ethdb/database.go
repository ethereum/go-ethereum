// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"

	gometrics "github.com/rcrowley/go-metrics"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	getTimer       gometrics.Timer // Timer for measuring the database get request counts and latencies
	putTimer       gometrics.Timer // Timer for measuring the database put request counts and latencies
	delTimer       gometrics.Timer // Timer for measuring the database delete request counts and latencies
	missMeter      gometrics.Meter // Meter for measuring the missed database get requests
	readMeter      gometrics.Meter // Meter for measuring the database get request data usage
	writeMeter     gometrics.Meter // Meter for measuring the database put request data usage
	compTimeMeter  gometrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter  gometrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter gometrics.Meter // Meter for measuring the data written during compaction

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

// NewLDBDatabase returns a LevelDB wrapped object. LDBDatabase does not persist data by
// it self but requires a background poller which syncs every X. `Flush` should be called
// when data needs to be stored and written to disk.
func NewLDBDatabase(file string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(file, &opt.Options{OpenFilesCacheCapacity: OpenFileLimit})
	// check for corruption and attempt to recover
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	// (re) check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	return &LDBDatabase{
		fn: file,
		db: db,
	}, nil
}

// Put puts the given key / value to the queue
func (self *LDBDatabase) Put(key []byte, value []byte) error {
	// Measure the database put latency, if requested
	if self.putTimer != nil {
		defer self.putTimer.UpdateSince(time.Now())
	}
	// Generate the data to write to disk, update the meter and write
	dat := rle.Compress(value)

	if self.writeMeter != nil {
		self.writeMeter.Mark(int64(len(dat)))
	}
	return self.db.Put(key, dat, nil)
}

// Get returns the given key if it's present.
func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if self.getTimer != nil {
		defer self.getTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	dat, err := self.db.Get(key, nil)
	if err != nil {
		if self.missMeter != nil {
			self.missMeter.Mark(1)
		}
		return nil, err
	}
	// Otherwise update the actually retrieved amount of data
	if self.readMeter != nil {
		self.readMeter.Mark(int64(len(dat)))
	}
	return rle.Decompress(dat)
}

// Delete deletes the key from the queue and database
func (self *LDBDatabase) Delete(key []byte) error {
	// Measure the database delete latency, if requested
	if self.delTimer != nil {
		defer self.delTimer.UpdateSince(time.Now())
	}
	// Execute the actual operation
	return self.db.Delete(key, nil)
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.db.NewIterator(nil, nil)
}

// Flush flushes out the queue to leveldb
func (self *LDBDatabase) Flush() error {
	return nil
}

func (self *LDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	self.quitLock.Lock()
	defer self.quitLock.Unlock()

	if self.quitChan != nil {
		errc := make(chan error)
		self.quitChan <- errc
		if err := <-errc; err != nil {
			glog.V(logger.Error).Infof("metrics failure in '%s': %v\n", self.fn, err)
		}
	}
	// Flush and close the database
	if err := self.Flush(); err != nil {
		glog.V(logger.Error).Infof("flushing '%s' failed: %v\n", self.fn, err)
	}
	self.db.Close()
	glog.V(logger.Error).Infoln("flushed and closed db:", self.fn)
}

func (self *LDBDatabase) LDB() *leveldb.DB {
	return self.db
}

// Meter configures the database metrics collectors and
func (self *LDBDatabase) Meter(prefix string) {
	// Initialize all the metrics collector at the requested prefix
	self.getTimer = metrics.NewTimer(prefix + "user/gets")
	self.putTimer = metrics.NewTimer(prefix + "user/puts")
	self.delTimer = metrics.NewTimer(prefix + "user/dels")
	self.missMeter = metrics.NewMeter(prefix + "user/misses")
	self.readMeter = metrics.NewMeter(prefix + "user/reads")
	self.writeMeter = metrics.NewMeter(prefix + "user/writes")
	self.compTimeMeter = metrics.NewMeter(prefix + "compact/time")
	self.compReadMeter = metrics.NewMeter(prefix + "compact/input")
	self.compWriteMeter = metrics.NewMeter(prefix + "compact/output")

	// Create a quit channel for the periodic collector and run it
	self.quitLock.Lock()
	self.quitChan = make(chan chan error)
	self.quitLock.Unlock()

	go self.meter(3 * time.Second)
}

// meter periodically retrieves internal leveldb counters and reports them to
// the metrics subsystem.
//
// This is how a stats table look like (currently):
//   Compactions
//    Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)
//   -------+------------+---------------+---------------+---------------+---------------
//      0   |          0 |       0.00000 |       1.27969 |       0.00000 |      12.31098
//      1   |         85 |     109.27913 |      28.09293 |     213.92493 |     214.26294
//      2   |        523 |    1000.37159 |       7.26059 |      66.86342 |      66.77884
//      3   |        570 |    1113.18458 |       0.00000 |       0.00000 |       0.00000
func (self *LDBDatabase) meter(refresh time.Duration) {
	// Create the counters to store current and previous values
	counters := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		counters[i] = make([]float64, 3)
	}
	// Iterate ad infinitum and collect the stats
	for i := 1; ; i++ {
		// Retrieve the database stats
		stats, err := self.db.GetProperty("leveldb.stats")
		if err != nil {
			glog.V(logger.Error).Infof("failed to read database stats: %v", err)
			return
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			glog.V(logger.Error).Infof("compaction table not found")
			return
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(counters[i%2]); j++ {
			counters[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				if value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64); err != nil {
					glog.V(logger.Error).Infof("compaction entry parsing failed: %v", err)
					return
				} else {
					counters[i%2][idx] += value
				}
			}
		}
		// Update all the requested meters
		if self.compTimeMeter != nil {
			self.compTimeMeter.Mark(int64((counters[i%2][0] - counters[(i-1)%2][0]) * 1000 * 1000 * 1000))
		}
		if self.compReadMeter != nil {
			self.compReadMeter.Mark(int64((counters[i%2][1] - counters[(i-1)%2][1]) * 1024 * 1024))
		}
		if self.compWriteMeter != nil {
			self.compWriteMeter.Mark(int64((counters[i%2][2] - counters[(i-1)%2][2]) * 1024 * 1024))
		}
		// Sleep a bit, then repeat the stats collection
		select {
		case errc := <-self.quitChan:
			// Quit requesting, stop hammering the database
			errc <- nil
			return

		case <-time.After(refresh):
			// Timeout, gather a new set of stats
		}
	}
}
