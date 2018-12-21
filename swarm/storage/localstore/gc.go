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

package localstore

import (
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	// gcTargetRatio defines the target number of items
	// in garbage collection index that will not be removed
	// on garbage collection. The target number of items
	// is calculated by gcTarget function. This value must be
	// in range (0,1]. For example, with 0.9 value,
	// garbage collection will leave 90% of defined capacity
	// in database after its run. This prevents frequent
	// garbage collection runt.
	gcTargetRatio = 0.9
	// gcBatchSize limits the number of chunks in a single
	// leveldb batch on garbage collection.
	gcBatchSize int64 = 1000
)

// collectGarbageWorker is a long running function that waits for
// collectGarbageTrigger channel to signal a garbage collection
// run. GC run iterates on gcIndex and removes older items
// form retrieval and other indexes.
func (db *DB) collectGarbageWorker() {
	for {
		select {
		case <-db.collectGarbageTrigger:
			// TODO: Add comment about done
			collectedCount, done, err := db.collectGarbage()
			if err != nil {
				log.Error("localstore collect garbage", "err", err)
			}
			// check if another gc run is needed
			if !done {
				db.triggerGarbageCollection()
			}

			if testHookCollectGarbage != nil {
				testHookCollectGarbage(collectedCount)
			}
		case <-db.close:
			return
		}
	}
}

// collectGarbage removes chunks from retrieval and other
// indexes if maximal number of chunks in database is reached.
// This function returns the number of removed chunks. If done
// is false, another call to this function is needed to collect
// the rest of the garbage as the batch size limit is reached.
// This function is called in collectGarbageWorker.
func (db *DB) collectGarbage() (collectedCount int64, done bool, err error) {
	batch := new(leveldb.Batch)
	target := db.gcTarget()

	done = true
	err = db.gcIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		// protect parallel updates
		unlock, err := db.lockAddr(item.Address)
		if err != nil {
			return false, err
		}
		defer unlock()

		gcSize := atomic.LoadInt64(&db.gcSize)
		if gcSize-collectedCount <= target {
			return true, nil
		}
		// delete from retrieve, pull, gc
		db.retrievalDataIndex.DeleteInBatch(batch, item)
		db.retrievalAccessIndex.DeleteInBatch(batch, item)
		db.pullIndex.DeleteInBatch(batch, item)
		db.gcIndex.DeleteInBatch(batch, item)
		collectedCount++
		if collectedCount >= gcBatchSize {
			// bach size limit reached,
			// another gc run is needed
			done = false
			return true, nil
		}
		return false, nil
	}, nil)
	if err != nil {
		return 0, false, err
	}

	err = db.shed.WriteBatch(batch)
	if err != nil {
		return 0, false, err
	}
	// batch is written, decrement gcSize
	db.incGCSize(-collectedCount)
	return collectedCount, done, nil
}

// gcTrigger retruns the absolute value for garbage collection
// target value, calculated from db.capacity and gcTargetRatio.
func (db *DB) gcTarget() (target int64) {
	return int64(float64(db.capacity) * gcTargetRatio)
}

// incGCSize increments gcSize by the provided number.
// If count is negative, it will decrement gcSize.
func (db *DB) incGCSize(count int64) {
	if count == 0 {
		return
	}
	new := atomic.AddInt64(&db.gcSize, count)
	select {
	case db.writeGCSizeTrigger <- struct{}{}:
	default:
	}
	if new >= db.capacity {
		db.triggerGarbageCollection()
	}
}

// triggerGarbageCollection signals collectGarbageWorker
// to call collectGarbage.
func (db *DB) triggerGarbageCollection() {
	select {
	case db.collectGarbageTrigger <- struct{}{}:
	default:
	}
}

var writeGCSizeDelay = 10 * time.Second

// writeGCSizeWorker writes gcSize on trigger event
// and waits writeGCSizeDelay after each write.
// It implements a linear backoff with delay of
// writeGCSizeDelay duration to avoid very frequent
// database operations.
func (db *DB) writeGCSizeWorker() {
	for {
		select {
		case <-db.writeGCSizeTrigger:
			err := db.writeGCSize(atomic.LoadInt64(&db.gcSize))
			if err != nil {
				log.Error("localstore write gc size", "err", err)
			}
			// Wait some time before writing gc size in the next
			// iteration. This prevents frequent I/O operations.
			select {
			case <-time.After(writeGCSizeDelay):
			case <-db.close:
				return
			}
		case <-db.close:
			return
		}
	}
}

// writeGCSize stores the number of items in gcIndex.
// It removes all hashes from gcUncountedHashesIndex
// not to include them on the next database initialization
// when gcSize is counted.
func (db *DB) writeGCSize(gcSize int64) (err error) {
	const maxBatchSize = 1000

	batch := new(leveldb.Batch)
	db.storedGCSize.PutInBatch(batch, uint64(gcSize))
	batchSize := 1

	// use only one iterator as it acquires its snapshot
	// not to remove hashes from index that are added
	// after stored gc size is written
	err = db.gcUncountedHashesIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		db.gcUncountedHashesIndex.DeleteInBatch(batch, item)
		batchSize++
		if batchSize >= maxBatchSize {
			err = db.shed.WriteBatch(batch)
			if err != nil {
				return false, err
			}
			batch.Reset()
			batchSize = 0
		}
		return false, nil
	}, nil)
	if err != nil {
		return err
	}
	return db.shed.WriteBatch(batch)
}

// testHookCollectGarbage is a hook that can provide
// information when a garbage collection run is done
// and how many items it removed.
var testHookCollectGarbage func(collectedCount int64)
