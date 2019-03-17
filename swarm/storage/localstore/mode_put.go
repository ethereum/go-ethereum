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
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

// ModePut enumerates different Putter modes.
type ModePut int

// Putter modes.
const (
	// ModePutRequest: when a chunk is received as a result of retrieve request and delivery
	ModePutRequest ModePut = iota
	// ModePutSync: when a chunk is received via syncing
	ModePutSync
	// ModePutUpload: when a chunk is created by local upload
	ModePutUpload
)

// Putter provides Put method to store Chunks
// to database.
type Putter struct {
	db   *DB
	mode ModePut
}

// NewPutter returns a new Putter on database
// with a specific Mode.
func (db *DB) NewPutter(mode ModePut) *Putter {
	return &Putter{
		mode: mode,
		db:   db,
	}
}

// Put stores the Chunk to database and depending
// on the Putter mode, it updates required indexes.
func (p *Putter) Put(ch chunk.Chunk) (err error) {
	return p.db.put(p.mode, chunkToItem(ch))
}

// put stores Item to database and updates other
// indexes. It acquires lockAddr to protect two calls
// of this function for the same address in parallel.
// Item fields Address and Data must not be
// with their nil values.
func (db *DB) put(mode ModePut, item shed.Item) (err error) {
	// protect parallel updates
	db.batchMu.Lock()
	defer db.batchMu.Unlock()

	batch := new(leveldb.Batch)

	// variables that provide information for operations
	// to be done after write batch function successfully executes
	var gcSizeChange int64   // number to add or subtract from gcSize
	var triggerPullFeed bool // signal pull feed subscriptions to iterate
	var triggerPushFeed bool // signal push feed subscriptions to iterate

	switch mode {
	case ModePutRequest:
		// put to indexes: retrieve, gc; it does not enter the syncpool

		// check if the chunk already is in the database
		// as gc index is updated
		i, err := db.retrievalAccessIndex.Get(item)
		switch err {
		case nil:
			item.AccessTimestamp = i.AccessTimestamp
		case leveldb.ErrNotFound:
			// no chunk accesses
		default:
			return err
		}
		i, err = db.retrievalDataIndex.Get(item)
		switch err {
		case nil:
			item.StoreTimestamp = i.StoreTimestamp
		case leveldb.ErrNotFound:
			// no chunk accesses
		default:
			return err
		}
		if item.AccessTimestamp != 0 {
			// delete current entry from the gc index
			db.gcIndex.DeleteInBatch(batch, item)
			gcSizeChange--
		}
		if item.StoreTimestamp == 0 {
			item.StoreTimestamp = now()
		}
		// update access timestamp
		item.AccessTimestamp = now()
		// update retrieve access index
		db.retrievalAccessIndex.PutInBatch(batch, item)
		// add new entry to gc index
		db.gcIndex.PutInBatch(batch, item)
		gcSizeChange++

		db.retrievalDataIndex.PutInBatch(batch, item)

	case ModePutUpload:
		// put to indexes: retrieve, push, pull

		item.StoreTimestamp = now()
		db.retrievalDataIndex.PutInBatch(batch, item)
		db.pullIndex.PutInBatch(batch, item)
		triggerPullFeed = true
		db.pushIndex.PutInBatch(batch, item)
		triggerPushFeed = true

	case ModePutSync:
		// put to indexes: retrieve, pull

		item.StoreTimestamp = now()
		db.retrievalDataIndex.PutInBatch(batch, item)
		db.pullIndex.PutInBatch(batch, item)
		triggerPullFeed = true

	default:
		return ErrInvalidMode
	}

	err = db.incGCSizeInBatch(batch, gcSizeChange)
	if err != nil {
		return err
	}

	err = db.shed.WriteBatch(batch)
	if err != nil {
		return err
	}
	if triggerPullFeed {
		db.triggerPullSubscriptions(db.po(item.Address))
	}
	if triggerPushFeed {
		db.triggerPushSubscriptions()
	}
	return nil
}
