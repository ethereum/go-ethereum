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

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

// Mode enumerates different modes of access and update
// operations on a database.
type Mode int

// Modes of access and update.
const (
	ModeSyncing Mode = iota
	ModeUpload
	ModeRequest
	ModeSynced
	// this modes are internal only
	// they can be removed completely
	// if accessors are not used internally
	modeAccess
	modeRemoval
)

// ModeName returns a descriptive name of a Mode.
// If the Mode is not known, a blank string is returned.
func ModeName(m Mode) (name string) {
	switch m {
	case ModeSyncing:
		return "syncing"
	case ModeUpload:
		return "upload"
	case ModeRequest:
		return "request"
	case ModeSynced:
		return "synced"
	case modeAccess:
		return "access"
	case modeRemoval:
		return "removal"
	}
	return ""
}

// access is called by an Accessor with a specific Mode.
// This function utilizes different indexes depending on
// the Mode.
func (db *DB) access(mode Mode, item shed.IndexItem) (out shed.IndexItem, err error) {
	if db.useRetrievalCompositeIndex {
		out, err = db.retrievalCompositeIndex.Get(item)
		if err != nil {
			return out, err
		}
	} else {
		out, err = db.retrievalDataIndex.Get(item)
		if err != nil {
			return out, err
		}
	}
	switch mode {
	case ModeRequest:
		// update the access counter
		// Q: can we do this asynchronously
		return out, db.update(context.TODO(), mode, item)
	default:
		// all other modes are not updating the index
	}
	return out, nil
}

// update is called by an Accessor with a specific Mode.
// This function calls updateBatch to perform operations
// on indexes and fields within a single batch.
func (db *DB) update(ctx context.Context, mode Mode, item shed.IndexItem) error {
	db.mu.RLock()
	b := db.batch
	db.mu.RUnlock()

	// check if the database is not closed
	select {
	case <-db.close:
		return ErrDBClosed
	default:
	}

	// call the update with the provided mode
	err := db.updateBatch(b, mode, item)
	if err != nil {
		return err
	}
	// trigger the writeBatches loop
	select {
	case db.writeTrigger <- struct{}{}:
	default:
	}
	// wait for batch to be written and return batch error
	// this is in order for Put calls to be synchronous
	select {
	case <-b.Done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return b.Err
}

// batch wraps leveldb.Batch extending it with a done channel.
type batch struct {
	*leveldb.Batch
	Done chan struct{} // to signal when batch is written
	Err  error         // error resulting from write
}

// newBatch constructs a new batch.
func newBatch() *batch {
	return &batch{
		Batch: new(leveldb.Batch),
		Done:  make(chan struct{}),
	}
}

// updateBatch performs different operations on fields and indexes
// depending on the provided Mode.
func (db *DB) updateBatch(b *batch, mode Mode, item shed.IndexItem) (err error) {
	switch mode {
	case ModeSyncing:
		// put to indexes: retrieve, pull
		item.StoreTimestamp = now()
		item.AccessTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(b.Batch, item)
		}
		db.pullIndex.PutInBatch(b.Batch, item)
		db.sizeCounter.IncInBatch(b.Batch)

	case ModeUpload:
		// put to indexes: retrieve, push, pull
		item.StoreTimestamp = now()
		item.AccessTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(b.Batch, item)
		}
		db.pullIndex.PutInBatch(b.Batch, item)
		db.pushIndex.PutInBatch(b.Batch, item)

	case ModeRequest:
		// put to indexes: retrieve, gc
		item.StoreTimestamp = now()
		item.AccessTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(b.Batch, item)
			db.retrievalAccessIndex.PutInBatch(b.Batch, item)
		}
		db.gcIndex.PutInBatch(b.Batch, item)

	case ModeSynced:
		// delete from push, insert to gc
		item.StoreTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(b.Batch, item)
			db.retrievalAccessIndex.PutInBatch(b.Batch, item)
		}
		db.pushIndex.DeleteInBatch(b.Batch, item)
		db.gcIndex.PutInBatch(b.Batch, item)

	case modeAccess:
		// update accessTimeStamp in retrieve, gc
		db.gcIndex.DeleteInBatch(b.Batch, item)
		item.AccessTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(b.Batch, item)
			db.retrievalAccessIndex.PutInBatch(b.Batch, item)
		}
		db.gcIndex.PutInBatch(b.Batch, item)

	case modeRemoval:
		// delete from retrieve, pull, gc
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.DeleteInBatch(b.Batch, item)
		} else {
			db.retrievalDataIndex.DeleteInBatch(b.Batch, item)
			db.retrievalAccessIndex.DeleteInBatch(b.Batch, item)
		}
		db.pullIndex.DeleteInBatch(b.Batch, item)
		db.gcIndex.DeleteInBatch(b.Batch, item)
		db.sizeCounter.DecInBatch(b.Batch)

	default:
		return ErrInvalidMode
	}
	return nil
}
