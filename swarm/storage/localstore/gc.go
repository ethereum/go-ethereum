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

/*
Counting number of items in garbage collection index

The number of items in garbage collection index is not the same as the number of
chunks in retrieval index (total number of stored chunks). Chunk can be garbage
collected only when it is set to a synced state by ModSetSync, and only then can
be counted into garbage collection size, which determines whether a number of
chunk should be removed from the storage by the garbage collection. This opens a
possibility that the storage size exceeds the limit if files are locally
uploaded and the node is not connected to other nodes or there is a problem with
syncing.

Tracking of garbage collection size (gcSize) is focused on performance. Key
points:

 1. counting the number of key/value pairs in LevelDB takes around 0.7s for 1e6
    on a very fast ssd (unacceptable long time in reality)
 2. locking leveldb batch writes with a global mutex (serial batch writes) is
    not acceptable, we should use locking per chunk address

Because of point 1. we cannot count the number of items in garbage collection
index in New constructor as it could last very long for realistic scenarios
where limit is 5e6 and nodes are running on slower hdd disks or cloud providers
with low IOPS.

Point 2. is a performance optimization to allow parallel batch writes with
getters, putters and setters. Every single batch that they create contain only
information related to a single chunk, no relations with other chunks or shared
statistical data (like gcSize). This approach avoids race conditions on writing
batches in parallel, but creates a problem of synchronizing statistical data
values like gcSize. With global mutex lock, any data could be written by any
batch, but would not use utilize the full potential of leveldb parallel writes.

To mitigate this two problems, the implementation of counting and persisting
gcSize is split into two parts. One is the in-memory value (gcSize) that is fast
to read and write with a dedicated mutex (gcSizeMu) if the batch which adds or
removes items from garbage collection index is successful. The second part is
the reliable persistence of this value to leveldb database, as storedGCSize
field. This database field is saved by writeGCSizeWorker and writeGCSize
functions when in-memory gcSize variable is changed, but no too often to avoid
very frequent database writes. This database writes are triggered by
writeGCSizeTrigger when a call is made to function incGCSize. Trigger ensures
that no database writes are done only when gcSize is changed (contrary to a
simpler periodic writes or checks). A backoff of 10s in writeGCSizeWorker
ensures that no frequent batch writes are made. Saving the storedGCSize on
database Close function ensures that in-memory gcSize is persisted when database
is closed.

This persistence must be resilient to failures like panics. For this purpose, a
collection of hashes that are added to the garbage collection index, but still
not persisted to storedGCSize, must be tracked to count them in when DB is
constructed again with New function after the failure (swarm node restarts). On
every batch write that adds a new item to garbage collection index, the same
hash is added to gcUncountedHashesIndex. This ensures that there is a persisted
information which hashes were added to the garbage collection index. But, when
the storedGCSize is saved by writeGCSize function, this values are removed in
the same batch in which storedGCSize is changed to ensure consistency. When the
panic happen, or database Close method is not saved. The database storage
contains all information to reliably and efficiently get the correct number of
items in garbage collection index. This is performed in the New function when
all hashes in gcUncountedHashesIndex are counted, added to the storedGCSize and
saved to the disk before the database is constructed again. Index
gcUncountedHashesIndex is acting as dirty bit for recovery that provides
information what needs to be corrected. With a simple dirty bit, the whole
garbage collection index should me counted on recovery instead only the items in
gcUncountedHashesIndex. Because of the triggering mechanizm of writeGCSizeWorker
and relatively short backoff time, the number of hashes in
gcUncountedHashesIndex should be low and it should take a very short time to
recover from the previous failure. If there was no failure and
gcUncountedHashesIndex is empty, which is the usual case, New function will take
the minimal time to return.
*/

package localstore

import (
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
	// garbage collection runs.
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
			// run a single collect garbage run and
			// if done is false, gcBatchSize is reached and
			// another collect garbage run is needed
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

		gcSize := db.getGCSize()
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

	db.gcSizeMu.Lock()
	new := db.gcSize + count
	db.gcSize = new
	db.gcSizeMu.Unlock()

	select {
	case db.writeGCSizeTrigger <- struct{}{}:
	default:
	}
	if new >= db.capacity {
		db.triggerGarbageCollection()
	}
}

// getGCSize returns gcSize value by locking it
// with gcSizeMu mutex.
func (db *DB) getGCSize() (count int64) {
	db.gcSizeMu.RLock()
	count = db.gcSize
	db.gcSizeMu.RUnlock()
	return count
}

// triggerGarbageCollection signals collectGarbageWorker
// to call collectGarbage.
func (db *DB) triggerGarbageCollection() {
	select {
	case db.collectGarbageTrigger <- struct{}{}:
	case <-db.close:
	default:
	}
}

// writeGCSizeWorker writes gcSize on trigger event
// and waits writeGCSizeDelay after each write.
// It implements a linear backoff with delay of
// writeGCSizeDelay duration to avoid very frequent
// database operations.
func (db *DB) writeGCSizeWorker() {
	for {
		select {
		case <-db.writeGCSizeTrigger:
			err := db.writeGCSize(db.getGCSize())
			if err != nil {
				log.Error("localstore write gc size", "err", err)
			}
			// Wait some time before writing gc size in the next
			// iteration. This prevents frequent I/O operations.
			select {
			case <-time.After(10 * time.Second):
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
// not to include them on the next DB initialization
// (New function) when gcSize is counted.
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
