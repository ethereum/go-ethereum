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
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// ModeSet enumerates different Setter modes.
type ModeSet int

// Setter modes.
const (
	// ModeSetAccess: when an update request is received for a chunk or chunk is retrieved for delivery
	ModeSetAccess ModeSet = iota
	// ModeSetSync: when push sync receipt is received
	ModeSetSync
	// modeSetRemove: when GC-d
	// unexported as no external packages should remove chunks from database
	modeSetRemove
)

// Setter sets the state of a particular
// Chunk in database by changing indexes.
type Setter struct {
	db   *DB
	mode ModeSet
}

// NewSetter returns a new Setter on database
// with a specific Mode.
func (db *DB) NewSetter(mode ModeSet) *Setter {
	return &Setter{
		mode: mode,
		db:   db,
	}
}

// Set updates database indexes for a specific
// chunk represented by the address.
func (s *Setter) Set(addr storage.Address) (err error) {
	return s.db.set(s.mode, addr)
}

// set updates database indexes for a specific
// chunk represented by the address.
// It acquires lockAddr to protect two calls
// of this function for the same address in parallel.
func (db *DB) set(mode ModeSet, addr storage.Address) (err error) {
	// protect parallel updates
	unlock, err := db.lockAddr(addr)
	if err != nil {
		return err
	}
	defer unlock()

	batch := new(leveldb.Batch)

	// variables that provide information for operations
	// to be done after write batch function successfully executes
	var gcSizeChange int64   // number to add or subtract from gcSize
	var triggerPullFeed bool // signal pull feed subscriptions to iterate

	item := addressToItem(addr)

	switch mode {
	case ModeSetAccess:
		// add to pull, insert to gc

		// need to get access timestamp here as it is not
		// provided by the access function, and it is not
		// a property of a chunk provided to Accessor.Put.

		i, err := db.retrievalDataIndex.Get(item)
		switch err {
		case nil:
			item.StoreTimestamp = i.StoreTimestamp
		case leveldb.ErrNotFound:
			db.pushIndex.DeleteInBatch(batch, item)
			item.StoreTimestamp = now()
		default:
			return err
		}

		i, err = db.retrievalAccessIndex.Get(item)
		switch err {
		case nil:
			item.AccessTimestamp = i.AccessTimestamp
			db.gcIndex.DeleteInBatch(batch, item)
			gcSizeChange--
		case leveldb.ErrNotFound:
			// the chunk is not accessed before
		default:
			return err
		}
		item.AccessTimestamp = now()
		db.retrievalAccessIndex.PutInBatch(batch, item)
		db.pullIndex.PutInBatch(batch, item)
		triggerPullFeed = true
		db.gcIndex.PutInBatch(batch, item)
		db.gcUncountedHashesIndex.PutInBatch(batch, item)
		gcSizeChange++

	case ModeSetSync:
		// delete from push, insert to gc

		// need to get access timestamp here as it is not
		// provided by the access function, and it is not
		// a property of a chunk provided to Accessor.Put.
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
			gcSizeChange--
		case leveldb.ErrNotFound:
			// the chunk is not accessed before
		default:
			return err
		}
		item.AccessTimestamp = now()
		db.retrievalAccessIndex.PutInBatch(batch, item)
		db.pushIndex.DeleteInBatch(batch, item)
		db.gcIndex.PutInBatch(batch, item)
		db.gcUncountedHashesIndex.PutInBatch(batch, item)
		gcSizeChange++

	case modeSetRemove:
		// delete from retrieve, pull, gc

		// need to get access timestamp here as it is not
		// provided by the access function, and it is not
		// a property of a chunk provided to Accessor.Put.

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

		db.retrievalDataIndex.DeleteInBatch(batch, item)
		db.retrievalAccessIndex.DeleteInBatch(batch, item)
		db.pullIndex.DeleteInBatch(batch, item)
		db.gcIndex.DeleteInBatch(batch, item)
		db.gcUncountedHashesIndex.DeleteInBatch(batch, item)
		// a check is needed for decrementing gcSize
		// as delete is not reporting if the key/value pair
		// is deleted or not
		if _, err := db.gcIndex.Get(item); err == nil {
			gcSizeChange = -1
		}

	default:
		return ErrInvalidMode
	}

	err = db.shed.WriteBatch(batch)
	if err != nil {
		return err
	}
	if gcSizeChange != 0 {
		db.incGCSize(gcSizeChange)
	}
	if triggerPullFeed {
		db.triggerPullSubscriptions(db.po(item.Address))
	}
	return nil
}
