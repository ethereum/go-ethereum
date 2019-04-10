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

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

// Get returns a chunk from the database. If the chunk is
// not found chunk.ErrChunkNotFound will be returned.
// All required indexes will be updated required by the
// Getter Mode. Get is required to implement chunk.Store
// interface.
func (db *DB) Get(_ context.Context, mode chunk.ModeGet, addr chunk.Address) (ch chunk.Chunk, err error) {
	out, err := db.get(mode, addr)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, chunk.ErrChunkNotFound
		}
		return nil, err
	}
	return chunk.NewChunk(out.Address, out.Data), nil
}

// get returns Item from the retrieval index
// and updates other indexes.
func (db *DB) get(mode chunk.ModeGet, addr chunk.Address) (out shed.Item, err error) {
	item := addressToItem(addr)

	out, err = db.retrievalDataIndex.Get(item)
	if err != nil {
		return out, err
	}
	switch mode {
	// update the access timestamp and gc index
	case chunk.ModeGetRequest:
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
	case chunk.ModeGetSync:
	case chunk.ModeGetLookup:
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
	db.batchMu.Lock()
	defer db.batchMu.Unlock()

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
