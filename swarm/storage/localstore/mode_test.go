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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestModeSyncing validates internal data operations and state
// for ModeSyncing on DB with default configuration.
func TestModeSyncing(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeSyncingValues(t, db)
}

// TestModeSyncing_useRetrievalCompositeIndex validates internal
// data operations and state for ModeSyncing on DB with
// retrieval composite index enabled.
func TestModeSyncing_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeSyncingValues(t, db)
}

// testModeSyncingValues validates ModeSyncing index values on the provided DB.
func testModeSyncingValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeSyncing)

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer func(n func() int64) { now = n }(now)
	now = func() (t int64) {
		return wantTimestamp
	}

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTest(db, chunk, wantTimestamp, 0))

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))
}

// TestModeUpload validates internal data operations and state
// for ModeUpload on DB with default configuration.
func TestModeUpload(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeUploadValues(t, db)
}

// TestModeUpload_useRetrievalCompositeIndex validates internal
// data operations and state for ModeUpload on DB with
// retrieval composite index enabled.
func TestModeUpload_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeUploadValues(t, db)
}

// testModeUploadValues validates ModeUpload index values on the provided DB.
func testModeUploadValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer func(n func() int64) { now = n }(now)
	now = func() (t int64) {
		return wantTimestamp
	}

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTest(db, chunk, wantTimestamp, 0))

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))

	t.Run("push index", newPushIndexTest(db, chunk, wantTimestamp, nil))
}

// TestModeRequest validates internal data operations and state
// for ModeRequest on DB with default configuration.
func TestModeRequest(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeRequestValues(t, db)
}

// TestModeRequest_useRetrievalCompositeIndex validates internal
// data operations and state for ModeRequest on DB with
// retrieval composite index enabled.
func TestModeRequest_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeRequestValues(t, db)
}

// testModeRequestValues validates ModeRequest index values on the provided DB.
func testModeRequestValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunk := generateRandomChunk()

	uploadTimestamp := time.Now().UTC().UnixNano()
	defer func(n func() int64) { now = n }(now)
	now = func() (t int64) {
		return uploadTimestamp
	}

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	a = db.Accessor(ModeRequest)

	t.Run("get unsynced", func(t *testing.T) {
		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, 0))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 0))

		t.Run("gc size", newIndexGCSizeTest(db))
	})

	// set chunk to synced state
	err = db.Accessor(ModeSynced).Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("first get", func(t *testing.T) {
		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, uploadTimestamp))

		t.Run("gc index", newGCIndexTest(db, chunk, uploadTimestamp, uploadTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})

	t.Run("second get", func(t *testing.T) {
		accessTimestamp := time.Now().UTC().UnixNano()
		now = func() (t int64) {
			return accessTimestamp
		}

		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, accessTimestamp))

		t.Run("gc index", newGCIndexTest(db, chunk, uploadTimestamp, accessTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})
}

// TestModeSynced validates internal data operations and state
// for ModeSynced on DB with default configuration.
func TestModeSynced(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeSyncedValues(t, db)
}

// TestModeSynced_useRetrievalCompositeIndex validates internal
// data operations and state for ModeSynced on DB with
// retrieval composite index enabled.
func TestModeSynced_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeSyncedValues(t, db)
}

// testModeSyncedValues validates ModeSynced index values on the provided DB.
func testModeSyncedValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeSyncing)

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer func(n func() int64) { now = n }(now)
	now = func() (t int64) {
		return wantTimestamp
	}

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	a = db.Accessor(ModeSynced)

	err = a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("push index", newPushIndexTest(db, chunk, wantTimestamp, leveldb.ErrNotFound))

	t.Run("gc index", newGCIndexTest(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeAccess validates internal data operations and state
// for ModeAccess on DB with default configuration.
func TestModeAccess(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeAccessValues(t, db)
}

// TestModeAccess_useRetrievalCompositeIndex validates internal
// data operations and state for ModeAccess on DB with
// retrieval composite index enabled.
func TestModeAccess_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeAccessValues(t, db)
}

// testModeAccessValues validates ModeAccess index values on the provided DB.
func testModeAccessValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunk := generateRandomChunk()

	uploadTimestamp := time.Now().UTC().UnixNano()
	defer func(n func() int64) { now = n }(now)
	now = func() (t int64) {
		return uploadTimestamp
	}

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	a = db.Accessor(modeAccess)

	t.Run("get unsynced", func(t *testing.T) {
		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, 0))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 0))

		t.Run("gc size", newIndexGCSizeTest(db))
	})

	// set chunk to synced state
	err = db.Accessor(ModeSynced).Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("first get", func(t *testing.T) {
		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, uploadTimestamp))

		t.Run("gc index", newGCIndexTest(db, chunk, uploadTimestamp, uploadTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})

	t.Run("second get", func(t *testing.T) {
		accessTimestamp := time.Now().UTC().UnixNano()
		now = func() (t int64) {
			return accessTimestamp
		}

		got, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(chunk.Address(), got.Address()) {
			t.Errorf("got chunk address %x, want %s", chunk.Address(), got.Address())
		}

		if !bytes.Equal(chunk.Data(), got.Data()) {
			t.Errorf("got chunk data %x, want %s", chunk.Data(), got.Data())
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, uploadTimestamp, accessTimestamp))

		t.Run("gc index", newGCIndexTest(db, chunk, uploadTimestamp, accessTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})
}

// TestModeRemoval validates internal data operations and state
// for ModeRemoval on DB with default configuration.
func TestModeRemoval(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeRemovalValues(t, db)
}

// TestModeRemoval_useRetrievalCompositeIndex validates internal
// data operations and state for ModeRemoval on DB with
// retrieval composite index enabled.
func TestModeRemoval_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeRemovalValues(t, db)
}

// testModeRemovalValues validates ModeRemoval index values on the provided DB.
func testModeRemovalValues(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunk := generateRandomChunk()

	err := a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	a = db.Accessor(modeRemoval)

	err = a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", func(t *testing.T) {
		wantErr := leveldb.ErrNotFound
		if db.useRetrievalCompositeIndex {
			_, err := db.retrievalCompositeIndex.Get(addressToItem(chunk.Address()))
			if err != wantErr {
				t.Errorf("got error %v, want %v", err, wantErr)
			}
			t.Run("retrieve index count", newIndexItemsCountTest(db.retrievalCompositeIndex, 0))
		} else {
			_, err := db.retrievalDataIndex.Get(addressToItem(chunk.Address()))
			if err != wantErr {
				t.Errorf("got error %v, want %v", err, wantErr)
			}
			t.Run("retrieve data index count", newIndexItemsCountTest(db.retrievalDataIndex, 0))

			// access index should not be set
			_, err = db.retrievalAccessIndex.Get(addressToItem(chunk.Address()))
			if err != wantErr {
				t.Errorf("got error %v, want %v", err, wantErr)
			}
			t.Run("retrieve access index count", newIndexItemsCountTest(db.retrievalAccessIndex, 0))
		}
	})

	t.Run("pull index", newPullIndexTest(db, chunk, 0, leveldb.ErrNotFound))

	t.Run("pull index count", newIndexItemsCountTest(db.pullIndex, 0))

	t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 0))

	t.Run("gc size", newIndexGCSizeTest(db))

}

// TestDB_pullIndex validates the ordering of keys in pull index.
// Pull index key contains PO prefix which is calculated from
// DB base key and chunk address. This is not an IndexItem field
// which are checked in Mode tests.
// This test uploads chunks, sorts them in expected order and
// validates that pull index iterator will iterate it the same
// order.
func TestDB_pullIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	a := db.Accessor(ModeUpload)

	chunkCount := 50

	chunks := make([]testIndexChunk, chunkCount)

	// upload random chunks
	for i := 0; i < chunkCount; i++ {
		chunk := generateRandomChunk()

		err := a.Put(context.Background(), chunk)
		if err != nil {
			t.Fatal(err)
		}

		chunks[i] = testIndexChunk{
			Chunk: chunk,
			// this timestamp is not the same as in
			// the index, but given that uploads
			// are sequential and that only ordering
			// of events matter, this information is
			// sufficient
			storeTimestamp: now(),
		}
	}

	testIndexItemsOrder(t, db.pullIndex, chunks, func(i, j int) (less bool) {
		poi := storage.Proximity(db.baseKey, chunks[i].Address())
		poj := storage.Proximity(db.baseKey, chunks[j].Address())
		if poi < poj {
			return true
		}
		if poi > poj {
			return false
		}
		if chunks[i].storeTimestamp < chunks[j].storeTimestamp {
			return true
		}
		if chunks[i].storeTimestamp > chunks[j].storeTimestamp {
			return false
		}
		return bytes.Compare(chunks[i].Address(), chunks[j].Address()) == -1
	})
}

func TestDB_gcIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testDB_gcIndex(t, db)
}

func TestDB_gcIndex_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testDB_gcIndex(t, db)
}

// testDB_gcIndex validates garbage collection index by uploading
// a chunk with and performing operations using synced, access and
// request modes.
func testDB_gcIndex(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunkCount := 50

	chunks := make([]testIndexChunk, chunkCount)

	// upload random chunks
	for i := 0; i < chunkCount; i++ {
		chunk := generateRandomChunk()

		err := a.Put(context.Background(), chunk)
		if err != nil {
			t.Fatal(err)
		}

		chunks[i] = testIndexChunk{
			Chunk: chunk,
		}
	}

	// check if all chunks are stored
	newIndexItemsCountTest(db.pullIndex, chunkCount)(t)

	// check that chunks are not collectable for garbage
	newIndexItemsCountTest(db.gcIndex, 0)(t)

	t.Run("access unsynced", func(t *testing.T) {
		chunk := chunks[0]

		a := db.Accessor(modeAccess)

		_, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		// the chunk is not synced
		// should not be in the garbace collection index
		newIndexItemsCountTest(db.gcIndex, 0)(t)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("request unsynced", func(t *testing.T) {
		chunk := chunks[1]

		a := db.Accessor(ModeRequest)

		_, err := a.Get(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		// the chunk is not synced
		// should not be in the garbace collection index
		newIndexItemsCountTest(db.gcIndex, 0)(t)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("sync one chunk", func(t *testing.T) {
		chunk := chunks[0]

		a := db.Accessor(ModeSynced)

		err := a.Put(context.Background(), chunk)
		if err != nil {
			t.Fatal(err)
		}

		// the chunk is synced and should be in gc index
		newIndexItemsCountTest(db.gcIndex, 1)(t)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("sync all chunks", func(t *testing.T) {
		a := db.Accessor(ModeSynced)

		for i := range chunks {
			err := a.Put(context.Background(), chunks[i])
			if err != nil {
				t.Fatal(err)
			}
		}

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("access one chunk", func(t *testing.T) {
		a := db.Accessor(modeAccess)

		i := 5

		_, err := a.Get(context.Background(), chunks[i].Address())
		if err != nil {
			t.Fatal(err)
		}

		// move the chunk to the end of the expected gc
		c := chunks[i]
		chunks = append(chunks[:i], chunks[i+1:]...)
		chunks = append(chunks, c)

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("request one chunk", func(t *testing.T) {
		a := db.Accessor(ModeRequest)

		i := 6

		_, err := a.Get(context.Background(), chunks[i].Address())
		if err != nil {
			t.Fatal(err)
		}

		// move the chunk to the end of the expected gc
		c := chunks[i]
		chunks = append(chunks[:i], chunks[i+1:]...)
		chunks = append(chunks, c)

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("random chunk access", func(t *testing.T) {
		a := db.Accessor(modeAccess)

		rand.Shuffle(len(chunks), func(i, j int) {
			chunks[i], chunks[j] = chunks[j], chunks[i]
		})

		for _, chunk := range chunks {
			_, err := a.Get(context.Background(), chunk.Address())
			if err != nil {
				t.Fatal(err)
			}
		}

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("random chunk request", func(t *testing.T) {
		a := db.Accessor(ModeRequest)

		rand.Shuffle(len(chunks), func(i, j int) {
			chunks[i], chunks[j] = chunks[j], chunks[i]
		})

		for _, chunk := range chunks {
			_, err := a.Get(context.Background(), chunk.Address())
			if err != nil {
				t.Fatal(err)
			}
		}

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("remove one chunk", func(t *testing.T) {
		a := db.Accessor(modeRemoval)

		i := 3

		err := a.Put(context.Background(), chunks[i])
		if err != nil {
			t.Fatal(err)
		}

		// remove the chunk from the expected chunks in gc index
		chunks = append(chunks[:i], chunks[i+1:]...)

		testIndexItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})
}

// newRetrieveIndexesTest returns a test function that validates if the right
// chunk values are in the retrieval indexes.
func newRetrieveIndexesTest(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		if db.useRetrievalCompositeIndex {
			item, err := db.retrievalCompositeIndex.Get(addressToItem(chunk.Address()))
			if err != nil {
				t.Fatal(err)
			}
			validateItem(t, item, chunk.Address(), chunk.Data(), storeTimestamp, accessTimestamp)
		} else {
			item, err := db.retrievalDataIndex.Get(addressToItem(chunk.Address()))
			if err != nil {
				t.Fatal(err)
			}
			validateItem(t, item, chunk.Address(), chunk.Data(), storeTimestamp, 0)

			// access index should not be set
			wantErr := leveldb.ErrNotFound
			item, err = db.retrievalAccessIndex.Get(addressToItem(chunk.Address()))
			if err != wantErr {
				t.Errorf("got error %v, want %v", err, wantErr)
			}
		}
	}
}

// newRetrieveIndexesTestWithAccess returns a test function that validates if the right
// chunk values are in the retrieval indexes when access time must be stored.
func newRetrieveIndexesTestWithAccess(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		if db.useRetrievalCompositeIndex {
			item, err := db.retrievalCompositeIndex.Get(addressToItem(chunk.Address()))
			if err != nil {
				t.Fatal(err)
			}
			validateItem(t, item, chunk.Address(), chunk.Data(), storeTimestamp, accessTimestamp)
		} else {
			item, err := db.retrievalDataIndex.Get(addressToItem(chunk.Address()))
			if err != nil {
				t.Fatal(err)
			}
			validateItem(t, item, chunk.Address(), chunk.Data(), storeTimestamp, 0)

			if accessTimestamp > 0 {
				item, err = db.retrievalAccessIndex.Get(addressToItem(chunk.Address()))
				if err != nil {
					t.Fatal(err)
				}
				validateItem(t, item, chunk.Address(), nil, 0, accessTimestamp)
			}
		}
	}
}

// newPullIndexTest returns a test function that validates if the right
// chunk values are in the pull index.
func newPullIndexTest(db *DB, chunk storage.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pullIndex.Get(shed.IndexItem{
			Address:        chunk.Address(),
			StoreTimestamp: storeTimestamp,
		})
		if err != wantError {
			t.Errorf("got error %v, want %v", err, wantError)
		}
		if err == nil {
			validateItem(t, item, chunk.Address(), nil, storeTimestamp, 0)
		}
	}
}

// newPushIndexTest returns a test function that validates if the right
// chunk values are in the push index.
func newPushIndexTest(db *DB, chunk storage.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pushIndex.Get(shed.IndexItem{
			Address:        chunk.Address(),
			StoreTimestamp: storeTimestamp,
		})
		if err != wantError {
			t.Errorf("got error %v, want %v", err, wantError)
		}
		if err == nil {
			validateItem(t, item, chunk.Address(), nil, storeTimestamp, 0)
		}
	}
}

// newGCIndexTest returns a test function that validates if the right
// chunk values are in the push index.
func newGCIndexTest(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.gcIndex.Get(shed.IndexItem{
			Address:         chunk.Address(),
			StoreTimestamp:  storeTimestamp,
			AccessTimestamp: accessTimestamp,
		})
		if err != nil {
			t.Fatal(err)
		}
		validateItem(t, item, chunk.Address(), nil, storeTimestamp, accessTimestamp)
	}
}

// newIndexItemsCountTest returns a test function that validates if
// an index contains expected number of key/value pairs.
func newIndexItemsCountTest(i shed.Index, want int) func(t *testing.T) {
	return func(t *testing.T) {
		var c int
		i.IterateAll(func(item shed.IndexItem) (stop bool, err error) {
			c++
			return
		})
		if c != want {
			t.Errorf("got %v items in index, want %v", c, want)
		}
	}
}

func newIndexGCSizeTest(db *DB) func(t *testing.T) {
	return func(t *testing.T) {
		var want int64
		db.gcIndex.IterateAll(func(item shed.IndexItem) (stop bool, err error) {
			want++
			return
		})
		got := atomic.LoadInt64(&db.gcSize)
		if got != want {
			t.Errorf("got gc size %v, want %v", got, want)
		}
	}
}

type testIndexChunk struct {
	storage.Chunk
	storeTimestamp  int64
	accessTimestamp int64
}

func testIndexItemsOrder(t *testing.T, i shed.Index, chunks []testIndexChunk, sortFunc func(i, j int) (less bool)) {
	newIndexItemsCountTest(i, len(chunks))(t)

	if sortFunc != nil {
		sort.Slice(chunks, sortFunc)
	}

	var cursor int
	err := i.IterateAll(func(item shed.IndexItem) (stop bool, err error) {
		want := chunks[cursor].Address()
		got := item.Address
		if !bytes.Equal(got, want) {
			return true, fmt.Errorf("got address %x at position %v, want %x", got, cursor, want)
		}
		cursor++
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// validateItem is a helper function that checks IndexItem values.
func validateItem(t *testing.T, item shed.IndexItem, address, data []byte, storeTimestamp, accessTimestamp int64) {
	t.Helper()

	if !bytes.Equal(item.Address, address) {
		t.Errorf("got item address %x, want %x", item.Address, address)
	}
	if !bytes.Equal(item.Data, data) {
		t.Errorf("got item data %x, want %x", item.Data, data)
	}
	if item.StoreTimestamp != storeTimestamp {
		t.Errorf("got item store timestamp %v, want %v", item.StoreTimestamp, storeTimestamp)
	}
	if item.AccessTimestamp != accessTimestamp {
		t.Errorf("got item access timestamp %v, want %v", item.AccessTimestamp, accessTimestamp)
	}
}
