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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// ModeGet enumerates different Getter modes.
type ModeGet int

// Getter modes.
const (
	// ModeGetRequest: when accessed for retrieval
	ModeGetRequest ModeGet = iota
	// ModeGetSync: when accessed for syncing or proof of custody request
	ModeGetSync
)

// Getter provides Get method to retrieve Chunks
// from database.
type Getter struct {
	db   *DB
	mode ModeGet
}

// NewGetter returns a new Getter on database
// with a specific Mode.
func (db *DB) NewGetter(mode ModeGet) *Getter {
	return &Getter{
		mode: mode,
		db:   db,
	}
}

// Get returns a chunk from the database. If the chunk is
// not found storage.ErrChunkNotFound will be returned.
// All required indexes will be updated required by the
// Getter Mode.
func (g *Getter) Get(addr storage.Address) (chunk storage.Chunk, err error) {
	out, err := g.db.get(g.mode, addr)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, storage.ErrChunkNotFound
		}
		return nil, err
	}
	return storage.NewChunk(out.Address, out.Data), nil
}

// get returns Item from the retrieval index
// and updates other indexes.
func (db *DB) get(mode ModeGet, addr storage.Address) (out shed.Item, err error) {
	item := addressToItem(addr)

	out, err = db.retrievalDataIndex.Get(item)
	if err != nil {
		return out, err
	}
	switch mode {
	// update the access timestamp and gc index
	case ModeGetRequest:
		if db.updateGCSem != nil {
			// wait before creating new goroutines
			// if updateGCSem buffer id full
			db.updateGCSem <- struct{}{}
		}
		db.updateGCWG.Add(1)
		go func() {
			defer db.updateGCWG.Done()
			if db.updateGCSem != nil {
				// free a spot in updateGCSem buffer
				// for a new goroutine
				defer func() { <-db.updateGCSem }()
			}
			err := db.updateGC(out)
			if err != nil {
				log.Error("localstore update gc", "err", err)
			}
			// if gc update hook is defined, call it
			if testHookUpdateGC != nil {
				testHookUpdateGC()
			}
		}()

	// no updates to indexes
	case ModeGetSync:
	default:
		return out, ErrInvalidMode
	}
	return out, nil
}

// updateGC updates garbage collection index for
// a single item. Provided item is expected to have
// only Address and Data fields with non zero values,
// which is ensured by the get function.
func (db *DB) updateGC(item shed.Item) (err error) {
	unlock, err := db.lockAddr(item.Address)
	if err != nil {
		return err
	}
	defer unlock()

	batch := new(leveldb.Batch)

	// update accessTimeStamp in retrieve, gc

	i, err := db.retrievalAccessIndex.Get(item)
	switch err {
	case nil:
		item.AccessTimestamp = i.AccessTimestamp
	case leveldb.ErrNotFound:
		// no chunk accesses
	default:
		return err
	}
	if item.AccessTimestamp == 0 {
		// chunk is not yet synced
		// do not add it to the gc index
		return nil
	}
	// delete current entry from the gc index
	db.gcIndex.DeleteInBatch(batch, item)
	// update access timestamp
	item.AccessTimestamp = now()
	// update retrieve access index
	db.retrievalAccessIndex.PutInBatch(batch, item)
	// add new entry to gc index
	db.gcIndex.PutInBatch(batch, item)

	return db.shed.WriteBatch(batch)
}

// testHookUpdateGC is a hook that can provide
// information when a garbage collection index is updated.
var testHookUpdateGC func()
