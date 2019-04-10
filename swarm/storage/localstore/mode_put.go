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
	"context"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

// Put stores the Chunk to database and depending
// on the Putter mode, it updates required indexes.
// Put is required to implement chunk.Store
// interface.
func (db *DB) Put(_ context.Context, mode chunk.ModePut, ch chunk.Chunk) (exists bool, err error) {
	return db.put(mode, chunkToItem(ch))
}

// put stores Item to database and updates other
// indexes. It acquires lockAddr to protect two calls
// of this function for the same address in parallel.
// Item fields Address and Data must not be
// with their nil values.
func (db *DB) put(mode chunk.ModePut, item shed.Item) (exists bool, err error) {
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
	case chunk.ModePutRequest:
		// put to indexes: retrieve, gc; it does not enter the syncpool

		// check if the chunk already is in the database
		// as gc index is updated
		i, err := db.retrievalAccessIndex.Get(item)
		switch err {
		case nil:
			exists = true
			item.AccessTimestamp = i.AccessTimestamp
		case leveldb.ErrNotFound:
			exists = false
			// no chunk accesses
		default:
			return false, err
		}
		i, err = db.retrievalDataIndex.Get(item)
		switch err {
		case nil:
			exists = true
			item.StoreTimestamp = i.StoreTimestamp
			item.BinID = i.BinID
		case leveldb.ErrNotFound:
			// no chunk accesses
			exists = false
		default:
			return false, err
		}
		if item.AccessTimestamp != 0 {
			// delete current entry from the gc index
			db.gcIndex.DeleteInBatch(batch, item)
			gcSizeChange--
		}
		if item.StoreTimestamp == 0 {
			item.StoreTimestamp = now()
		}
		if item.BinID == 0 {
			item.BinID, err = db.binIDs.IncInBatch(batch, uint64(db.po(item.Address)))
			if err != nil {
				return false, err
			}
		}
		// update access timestamp
		item.AccessTimestamp = now()
		// update retrieve access index
		db.retrievalAccessIndex.PutInBatch(batch, item)
		// add new entry to gc index
		db.gcIndex.PutInBatch(batch, item)
		gcSizeChange++

		db.retrievalDataIndex.PutInBatch(batch, item)

	case chunk.ModePutUpload:
		// put to indexes: retrieve, push, pull

		exists, err = db.retrievalDataIndex.Has(item)
		if err != nil {
			return false, err
		}
		if !exists {
			item.StoreTimestamp = now()
			item.BinID, err = db.binIDs.IncInBatch(batch, uint64(db.po(item.Address)))
			if err != nil {
				return false, err
			}
			db.retrievalDataIndex.PutInBatch(batch, item)
			db.pullIndex.PutInBatch(batch, item)
			triggerPullFeed = true
			db.pushIndex.PutInBatch(batch, item)
			triggerPushFeed = true
		}

	case chunk.ModePutSync:
		// put to indexes: retrieve, pull

		exists, err = db.retrievalDataIndex.Has(item)
		if err != nil {
			return exists, err
		}
		if !exists {
			item.StoreTimestamp = now()
			item.BinID, err = db.binIDs.IncInBatch(batch, uint64(db.po(item.Address)))
			if err != nil {
				return false, err
			}
			db.retrievalDataIndex.PutInBatch(batch, item)
			db.pullIndex.PutInBatch(batch, item)
			triggerPullFeed = true
		}

	default:
		return false, ErrInvalidMode
	}

	err = db.incGCSizeInBatch(batch, gcSizeChange)
	if err != nil {
		return false, err
	}

	err = db.shed.WriteBatch(batch)
	if err != nil {
		return false, err
	}
	if triggerPullFeed {
		db.triggerPullSubscriptions(db.po(item.Address))
	}
	if triggerPushFeed {
		db.triggerPushSubscriptions()
	}
	return exists, nil
}
