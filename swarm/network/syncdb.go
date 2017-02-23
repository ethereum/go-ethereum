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

package network

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

const counterKeyPrefix = 0x01

/*
syncDb is a queueing service for outgoing deliveries.
One instance per priority queue for each peer

a syncDb instance maintains an in-memory buffer (of capacity bufferSize)
once its in-memory buffer is full it switches to persisting in db
and dbRead iterator iterates through the items keeping their order
once the db read catches up (there is no more items in the db) then
it switches back to in-memory buffer.

when syncdb is stopped all items in the buffer are saved to the db
*/
type syncDb struct {
	start          []byte               // this syncdb starting index in requestdb
	key            storage.Key          // remote peers address key
	counterKey     []byte               // db key to persist counter
	priority       uint                 // priotity High|Medium|Low
	buffer         chan interface{}     // incoming request channel
	db             *storage.LDBDatabase // underlying db (TODO should be interface)
	done           chan bool            // chan to signal goroutines finished quitting
	quit           chan bool            // chan to signal quitting to goroutines
	total, dbTotal int                  // counts for one session
	batch          chan chan int        // channel for batch requests
	dbBatchSize    uint                 // number of items before batch is saved
}

// constructor needs a shared request db (leveldb)
// priority is used in the index key
// uses a buffer and a leveldb for persistent storage
// bufferSize, dbBatchSize are config parameters
func newSyncDb(db *storage.LDBDatabase, key storage.Key, priority uint, bufferSize, dbBatchSize uint, deliver func(interface{}, chan bool) bool) *syncDb {
	start := make([]byte, 42)
	start[1] = byte(priorities - priority)
	copy(start[2:34], key)

	counterKey := make([]byte, 34)
	counterKey[0] = counterKeyPrefix
	copy(counterKey[1:], start[1:34])

	syncdb := &syncDb{
		start:       start,
		key:         key,
		counterKey:  counterKey,
		priority:    priority,
		buffer:      make(chan interface{}, bufferSize),
		db:          db,
		done:        make(chan bool),
		quit:        make(chan bool),
		batch:       make(chan chan int),
		dbBatchSize: dbBatchSize,
	}
	log.Trace(fmt.Sprintf("syncDb[peer: %v, priority: %v] - initialised", key.Log(), priority))

	// starts the main forever loop reading from buffer
	go syncdb.bufferRead(deliver)
	return syncdb
}

/*
bufferRead is a forever iterator loop that takes care of delivering
outgoing store requests reads from incoming buffer

its argument is the deliver function taking the item as first argument
and a quit channel as second.
Closing of this channel is supposed to abort all waiting for delivery
(typically network write)

The iteration switches between 2 modes,
* buffer mode reads the in-memory buffer and delivers the items directly
* db mode reads from the buffer and writes to the db, parallelly another
routine is started that reads from the db and delivers items

If there is buffer contention in buffer mode (slow network, high upload volume)
syncdb switches to db mode and starts dbRead
Once db backlog is delivered, it reverts back to in-memory buffer

It is automatically started when syncdb is initialised.

It saves the buffer to db upon receiving quit signal. syncDb#stop()
*/
func (self *syncDb) bufferRead(deliver func(interface{}, chan bool) bool) {
	var buffer, db chan interface{} // channels representing the two read modes
	var more bool
	var req interface{}
	var entry *syncDbEntry
	var inBatch, inDb int
	batch := new(leveldb.Batch)
	var dbSize chan int
	quit := self.quit
	counterValue := make([]byte, 8)

	// counter is used for keeping the items in order, persisted to db
	// start counter where db was at, 0 if not found
	data, err := self.db.Get(self.counterKey)
	var counter uint64
	if err == nil {
		counter = binary.BigEndian.Uint64(data)
		log.Trace(fmt.Sprintf("syncDb[%v/%v] - counter read from db at %v", self.key.Log(), self.priority, counter))
	} else {
		log.Trace(fmt.Sprintf("syncDb[%v/%v] - counter starts at %v", self.key.Log(), self.priority, counter))
	}

LOOP:
	for {
		// waiting for item next in the buffer, or quit signal or batch request
		select {
		// buffer only closes when writing to db
		case req = <-buffer:
			// deliver request : this is blocking on network write so
			// it is passed the quit channel as argument, so that it returns
			// if syncdb is stopped. In this case we need to save the item to the db
			more = deliver(req, self.quit)
			if !more {
				log.Debug(fmt.Sprintf("syncDb[%v/%v] quit: switching to db. session tally (db/total): %v/%v", self.key.Log(), self.priority, self.dbTotal, self.total))
				// received quit signal, save request currently waiting delivery
				// by switching to db mode and closing the buffer
				buffer = nil
				db = self.buffer
				close(db)
				quit = nil // needs to block the quit case in select
				break      // break from select, this item will be written to the db
			}
			self.total++
			log.Trace(fmt.Sprintf("syncDb[%v/%v] deliver (db/total): %v/%v", self.key.Log(), self.priority, self.dbTotal, self.total))
			// by the time deliver returns, there were new writes to the buffer
			// if buffer contention is detected, switch to db mode which drains
			// the buffer so no process will block on pushing store requests
			if len(buffer) == cap(buffer) {
				log.Debug(fmt.Sprintf("syncDb[%v/%v] buffer full %v: switching to db. session tally (db/total): %v/%v", self.key.Log(), self.priority, cap(buffer), self.dbTotal, self.total))
				buffer = nil
				db = self.buffer
			}
			continue LOOP

			// incoming entry to put into db
		case req, more = <-db:
			if !more {
				// only if quit is called, saved all the buffer
				binary.BigEndian.PutUint64(counterValue, counter)
				batch.Put(self.counterKey, counterValue) // persist counter in batch
				self.writeSyncBatch(batch)               // save batch
				log.Trace(fmt.Sprintf("syncDb[%v/%v] quitting: save current batch to db", self.key.Log(), self.priority))
				break LOOP
			}
			self.dbTotal++
			self.total++
			// otherwise break after select
		case dbSize = <-self.batch:
			// explicit request for batch
			if inBatch == 0 && quit != nil {
				// there was no writes since the last batch so db depleted
				// switch to buffer mode
				log.Debug(fmt.Sprintf("syncDb[%v/%v] empty db: switching to buffer", self.key.Log(), self.priority))
				db = nil
				buffer = self.buffer
				dbSize <- 0 // indicates to 'caller' that batch has been written
				inDb = 0
				continue LOOP
			}
			binary.BigEndian.PutUint64(counterValue, counter)
			batch.Put(self.counterKey, counterValue)
			log.Debug(fmt.Sprintf("syncDb[%v/%v] write batch %v/%v - %x - %x", self.key.Log(), self.priority, inBatch, counter, self.counterKey, counterValue))
			batch = self.writeSyncBatch(batch)
			dbSize <- inBatch // indicates to 'caller' that batch has been written
			inBatch = 0
			continue LOOP

			// closing syncDb#quit channel is used to signal to all goroutines to quit
		case <-quit:
			// need to save backlog, so switch to db mode
			db = self.buffer
			buffer = nil
			quit = nil
			log.Trace(fmt.Sprintf("syncDb[%v/%v] quitting: save buffer to db", self.key.Log(), self.priority))
			close(db)
			continue LOOP
		}

		// only get here if we put req into db
		entry, err = self.newSyncDbEntry(req, counter)
		if err != nil {
			log.Warn(fmt.Sprintf("syncDb[%v/%v] saving request %v (#%v/%v) failed: %v", self.key.Log(), self.priority, req, inBatch, inDb, err))
			continue LOOP
		}
		batch.Put(entry.key, entry.val)
		log.Trace(fmt.Sprintf("syncDb[%v/%v] to batch %v '%v' (#%v/%v/%v)", self.key.Log(), self.priority, req, entry, inBatch, inDb, counter))
		// if just switched to db mode and not quitting, then launch dbRead
		// in a parallel go routine to send deliveries from db
		if inDb == 0 && quit != nil {
			log.Trace(fmt.Sprintf("syncDb[%v/%v] start dbRead", self.key.Log(), self.priority))
			go self.dbRead(true, counter, deliver)
		}
		inDb++
		inBatch++
		counter++
		// need to save the batch if it gets too large (== dbBatchSize)
		if inBatch%int(self.dbBatchSize) == 0 {
			batch = self.writeSyncBatch(batch)
		}
	}
	log.Info(fmt.Sprintf("syncDb[%v:%v]: saved %v keys (saved counter at %v)", self.key.Log(), self.priority, inBatch, counter))
	close(self.done)
}

// writes the batch to the db and returns a new batch object
func (self *syncDb) writeSyncBatch(batch *leveldb.Batch) *leveldb.Batch {
	err := self.db.Write(batch)
	if err != nil {
		log.Warn(fmt.Sprintf("syncDb[%v/%v] saving batch to db failed: %v", self.key.Log(), self.priority, err))
		return batch
	}
	return new(leveldb.Batch)
}

// abstract type for db entries (TODO could be a feature of Receipts)
type syncDbEntry struct {
	key, val []byte
}

func (self syncDbEntry) String() string {
	return fmt.Sprintf("key: %x, value: %x", self.key, self.val)
}

/*
	dbRead is iterating over store requests to be sent over to the peer
	this is mainly to prevent crashes due to network output buffer contention (???)
	as well as to make syncronisation resilient to disconnects
	the messages are supposed to be sent in the p2p priority queue.

	the request DB is shared between peers, but domains for each syncdb
	are disjoint. dbkeys (42 bytes) are structured:
	* 0: 0x00 (0x01 reserved for counter key)
	* 1: priorities - priority (so that high priority can be replayed first)
	* 2-33: peers address
	* 34-41: syncdb counter to preserve order (this field is missing for the counter key)

	values (40 bytes) are:
	* 0-31: key
	* 32-39: request id

dbRead needs a boolean to indicate if on first round all the historical
record is synced. Second argument to indicate current db counter
The third is the function to apply
*/
func (self *syncDb) dbRead(useBatches bool, counter uint64, fun func(interface{}, chan bool) bool) {
	key := make([]byte, 42)
	copy(key, self.start)
	binary.BigEndian.PutUint64(key[34:], counter)
	var batches, n, cnt, total int
	var more bool
	var entry *syncDbEntry
	var it iterator.Iterator
	var del *leveldb.Batch
	batchSizes := make(chan int)

	for {
		// if useBatches is false, cnt is not set
		if useBatches {
			// this could be called before all cnt items sent out
			// so that loop is not blocking while delivering
			// only relevant if cnt is large
			select {
			case self.batch <- batchSizes:
			case <-self.quit:
				return
			}
			// wait for the write to finish and get the item count in the next batch
			cnt = <-batchSizes
			batches++
			if cnt == 0 {
				// empty
				return
			}
		}
		it = self.db.NewIterator()
		it.Seek(key)
		if !it.Valid() {
			copy(key, self.start)
			useBatches = true
			continue
		}
		del = new(leveldb.Batch)
		log.Trace(fmt.Sprintf("syncDb[%v/%v]: new iterator: %x (batch %v, count %v)", self.key.Log(), self.priority, key, batches, cnt))

		for n = 0; !useBatches || n < cnt; it.Next() {
			copy(key, it.Key())
			if len(key) == 0 || key[0] != 0 {
				copy(key, self.start)
				useBatches = true
				break
			}
			val := make([]byte, 40)
			copy(val, it.Value())
			entry = &syncDbEntry{key, val}
			// log.Trace(fmt.Sprintf("syncDb[%v/%v] - %v, batches: %v, total: %v, session total from db: %v/%v", self.key.Log(), self.priority, self.key.Log(), batches, total, self.dbTotal, self.total))
			more = fun(entry, self.quit)
			if !more {
				// quit received when waiting to deliver entry, the entry will not be deleted
				log.Trace(fmt.Sprintf("syncDb[%v/%v] batch %v quit after %v/%v items", self.key.Log(), self.priority, batches, n, cnt))
				break
			}
			// since subsequent batches of the same db session are indexed incrementally
			// deleting earlier batches can be delayed and parallelised
			// this could be batch delete when db is idle (but added complexity esp when quitting)
			del.Delete(key)
			n++
			total++
		}
		log.Debug(fmt.Sprintf("syncDb[%v/%v] - db session closed, batches: %v, total: %v, session total from db: %v/%v", self.key.Log(), self.priority, batches, total, self.dbTotal, self.total))
		self.db.Write(del) // this could be async called only when db is idle
		it.Release()
	}
}

//
func (self *syncDb) stop() {
	close(self.quit)
	<-self.done
}

// calculate a dbkey for the request, for the db to work
// see syncdb for db key structure
// polimorphic: accepted types, see syncer#addRequest
func (self *syncDb) newSyncDbEntry(req interface{}, counter uint64) (entry *syncDbEntry, err error) {
	var key storage.Key
	var chunk *storage.Chunk
	var id uint64
	var ok bool
	var sreq *storeRequestMsgData

	if key, ok = req.(storage.Key); ok {
		id = generateId()
	} else if chunk, ok = req.(*storage.Chunk); ok {
		key = chunk.Key
		id = generateId()
	} else if sreq, ok = req.(*storeRequestMsgData); ok {
		key = sreq.Key
		id = sreq.Id
	} else if entry, ok = req.(*syncDbEntry); !ok {
		return nil, fmt.Errorf("type not allowed: %v (%T)", req, req)
	}

	// order by peer > priority > seqid
	// value is request id if exists
	if entry == nil {
		dbkey := make([]byte, 42)
		dbval := make([]byte, 40)

		// encode key
		copy(dbkey[:], self.start[:34]) // db  peer
		binary.BigEndian.PutUint64(dbkey[34:], counter)
		// encode value
		copy(dbval, key[:])
		binary.BigEndian.PutUint64(dbval[32:], id)

		entry = &syncDbEntry{dbkey, dbval}
	}
	return
}
