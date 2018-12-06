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
	"encoding/hex"
	"time"

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
		// No need to get access timestamp here as it is used
		// only for some of Modes in update and access time
		// is not property of the chunk returned by the Accessor.Get.
		out, err = db.retrievalDataIndex.Get(item)
		if err != nil {
			return out, err
		}
	}
	switch mode {
	case ModeRequest, modeAccess:
		// update the access timestamp and fc index
		return out, db.updateOnAccess(mode, out)
	default:
		// all other modes are not updating the index
	}
	return out, nil
}

var (
	updateLockTimeout    = 3 * time.Second
	updateLockCheckDelay = 30 * time.Microsecond
)

// update performs different operations on fields and indexes
// depending on the provided Mode. It is called in accessor
// put function.
// It protects parallel updates of items with the same address
// with updateLocks map and waiting using a simple for loop.
func (db *DB) update(mode Mode, item shed.IndexItem) (err error) {
	// protect parallel updates
	start := time.Now()
	lockKey := hex.EncodeToString(item.Address)
	for {
		_, loaded := db.updateLocks.LoadOrStore(lockKey, struct{}{})
		if !loaded {
			break
		}
		time.Sleep(updateLockCheckDelay)
		if time.Since(start) > updateLockTimeout {
			return ErrUpdateLockTimeout
		}
	}
	defer db.updateLocks.Delete(lockKey)

	batch := new(leveldb.Batch)

	switch mode {
	case ModeSyncing:
		// put to indexes: retrieve, pull
		item.StoreTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(batch, item)
		}
		db.pullIndex.PutInBatch(batch, item)

	case ModeUpload:
		// put to indexes: retrieve, push, pull
		item.StoreTimestamp = now()
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(batch, item)
		} else {
			db.retrievalDataIndex.PutInBatch(batch, item)
		}
		db.pullIndex.PutInBatch(batch, item)
		db.pushIndex.PutInBatch(batch, item)

	case ModeRequest:
		// putting a chunk on mode request does not do anything
		return nil

	case ModeSynced:
		// delete from push, insert to gc

		// need to get access timestamp here as it is not
		// provided by the access function, and it is not
		// a property of a chunk provided to Accessor.Put.
		if db.useRetrievalCompositeIndex {
			i, err := db.retrievalCompositeIndex.Get(item)
			if err != nil {
				if err == leveldb.ErrNotFound {
					// chunk is not found,
					// no need to update gc index
					// just delete from the push index
					// if it is there
					db.pushIndex.DeleteInBatch(batch, item)
					return nil
				}
				return err
			}
			item.AccessTimestamp = i.AccessTimestamp
			item.StoreTimestamp = i.StoreTimestamp
			if item.AccessTimestamp == 0 {
				// the chunk is not accessed before
				// set access time for gc index
				item.AccessTimestamp = now()
				db.retrievalCompositeIndex.PutInBatch(batch, item)
			}
		} else {
			i, err := db.retrievalDataIndex.Get(item)
			if err != nil {
				if err == leveldb.ErrNotFound {
					// chunk is not found,
					// no need to update gc index
					// just delete from the push index
					// if it is there
					db.pushIndex.DeleteInBatch(batch, item)
					return nil
				}
				return err
			}
			item.StoreTimestamp = i.StoreTimestamp

			i, err = db.retrievalAccessIndex.Get(item)
			switch err {
			case nil:
				item.AccessTimestamp = i.AccessTimestamp
				db.gcIndex.DeleteInBatch(batch, item)
			case leveldb.ErrNotFound:
				// the chunk is not accessed before
			default:
				return err
			}
			item.AccessTimestamp = now()
			db.retrievalAccessIndex.PutInBatch(batch, item)
		}
		db.pushIndex.DeleteInBatch(batch, item)
		db.gcIndex.PutInBatch(batch, item)

	case modeAccess:
		// putting a chunk on mode access does not do anything
		return nil

	case modeRemoval:
		// delete from retrieve, pull, gc

		// need to get access timestamp here as it is not
		// provided by the access function, and it is not
		// a property of a chunk provided to Accessor.Put.
		if db.useRetrievalCompositeIndex {
			i, err := db.retrievalCompositeIndex.Get(item)
			if err != nil {
				return err
			}
			item.StoreTimestamp = i.StoreTimestamp
			item.AccessTimestamp = i.AccessTimestamp
		} else {
			i, err := db.retrievalAccessIndex.Get(item)
			switch err {
			case nil:
				item.AccessTimestamp = i.AccessTimestamp
			case leveldb.ErrNotFound:
			default:
				return err
			}
			i, err = db.retrievalDataIndex.Get(item)
			if err != nil {
				return err
			}
			item.StoreTimestamp = i.StoreTimestamp
		}
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.DeleteInBatch(batch, item)
		} else {
			db.retrievalDataIndex.DeleteInBatch(batch, item)
			db.retrievalAccessIndex.DeleteInBatch(batch, item)
		}
		db.pullIndex.DeleteInBatch(batch, item)
		db.gcIndex.DeleteInBatch(batch, item)

	default:
		return ErrInvalidMode
	}

	return db.shed.WriteBatch(batch)
}

// updateOnAccess is called in access function and performs
// different operations on fields and indexes depending on
// the provided Mode.
// This function is separated from the update function to prevent
// changes on calling accessor put function in access and request modes.
// It protects parallel updates of items with the same address
// with updateLocks map and waiting using a simple for loop.
func (db *DB) updateOnAccess(mode Mode, item shed.IndexItem) (err error) {
	// protect parallel updates
	start := time.Now()
	lockKey := hex.EncodeToString(item.Address)
	for {
		_, loaded := db.updateLocks.LoadOrStore(lockKey, struct{}{})
		if !loaded {
			break
		}
		time.Sleep(updateLockCheckDelay)
		if time.Since(start) > updateLockTimeout {
			return ErrUpdateLockTimeout
		}
	}
	defer db.updateLocks.Delete(lockKey)

	batch := new(leveldb.Batch)

	switch mode {
	case ModeRequest:
		// update accessTimeStamp in retrieve, gc

		if db.useRetrievalCompositeIndex {
			// access timestamp is already populated
			// in the provided item, passed from access function.
		} else {
			i, err := db.retrievalAccessIndex.Get(item)
			switch err {
			case nil:
				item.AccessTimestamp = i.AccessTimestamp
			case leveldb.ErrNotFound:
				// no chunk accesses
			default:
				return err
			}
		}
		if item.AccessTimestamp == 0 {
			// chunk is not yes synced
			// do not add it to the gc index
			return nil
		}
		// delete current entry from the gc index
		db.gcIndex.DeleteInBatch(batch, item)
		// update access timestamp
		item.AccessTimestamp = now()
		// update retrieve access index
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(batch, item)
		} else {
			db.retrievalAccessIndex.PutInBatch(batch, item)
		}
		// add new entry to gc index
		db.gcIndex.PutInBatch(batch, item)

	// Q: modeAccess and ModeRequest are very similar,  why do we need both?
	case modeAccess:
		// update accessTimeStamp in retrieve, pull, gc

		if db.useRetrievalCompositeIndex {
			// access timestamp is already populated
			// in the provided item, passed from access function.
		} else {
			i, err := db.retrievalAccessIndex.Get(item)
			switch err {
			case nil:
				item.AccessTimestamp = i.AccessTimestamp
			case leveldb.ErrNotFound:
				// no chunk accesses
			default:
				return err
			}
		}
		// Q: why do we need to update this index?
		db.pullIndex.PutInBatch(batch, item)
		if item.AccessTimestamp == 0 {
			// chunk is not yes synced
			// do not add it to the gc index
			return nil
		}
		// delete current entry from the gc index
		db.gcIndex.DeleteInBatch(batch, item)
		// update access timestamp
		item.AccessTimestamp = now()
		// update retrieve access index
		if db.useRetrievalCompositeIndex {
			db.retrievalCompositeIndex.PutInBatch(batch, item)
		} else {
			db.retrievalAccessIndex.PutInBatch(batch, item)
		}
		// add new entry to gc index
		db.gcIndex.PutInBatch(batch, item)

	default:
		return ErrInvalidMode
	}

	return db.shed.WriteBatch(batch)
}
