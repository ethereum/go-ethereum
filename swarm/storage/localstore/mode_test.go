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
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestModeSyncing validates internal data operations and state
// for ModeSyncing on DB with default configuration.
func TestModeSyncing(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	testModeSyncing(t, db)
}

// TestModeSyncing_withRetrievalCompositeIndex validates internal
// data operations and state for ModeSyncing on DB with
// retrieval composite index enabled.
func TestModeSyncing_withRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, WithRetrievalCompositeIndex(true))
	defer cleanupFunc()

	testModeSyncing(t, db)
}

// testModeSyncing validates ModeSyncing on the provided DB.
func testModeSyncing(t *testing.T, db *DB) {
	a := db.Accessor(ModeSyncing)

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	now = func() (t int64) {
		return wantTimestamp
	}

	wantSize, err := db.sizeCounter.Get()
	if err != nil {
		t.Fatal(err)
	}

	err = a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	wantSize++

	t.Run("retrieve indexes", testRetrieveIndexes(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("pull index", testPullIndex(db, chunk, wantTimestamp))

	t.Run("size counter", testSizeCounter(db, wantSize))
}

// TestModeUpload validates internal data operations and state
// for ModeUpload on DB with default configuration.
func TestModeUpload(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	testModeUpload(t, db)
}

// TestModeUpload_withRetrievalCompositeIndex validates internal
// data operations and state for ModeUpload on DB with
// retrieval composite index enabled.
func TestModeUpload_withRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, WithRetrievalCompositeIndex(true))
	defer cleanupFunc()

	testModeUpload(t, db)
}

// testModeUpload validates ModeUpload on the provided DB.
func testModeUpload(t *testing.T, db *DB) {
	a := db.Accessor(ModeUpload)

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	now = func() (t int64) {
		return wantTimestamp
	}

	wantSize, err := db.sizeCounter.Get()
	if err != nil {
		t.Fatal(err)
	}

	err = a.Put(context.Background(), chunk)
	if err != nil {
		t.Fatal(err)
	}

	wantSize++

	t.Run("retrieve indexes", testRetrieveIndexes(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("pull index", testPullIndex(db, chunk, wantTimestamp))

	t.Run("push index", testPullIndex(db, chunk, wantTimestamp))

	t.Run("size counter", testSizeCounter(db, wantSize))
}

// testRetrieveIndexes returns a test function that validates if the right
// chunk values are in the retrieval indexes.
func testRetrieveIndexes(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
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

// testPullIndex returns a test function that validates if the right
// chunk values are in the pull index.
func testPullIndex(db *DB, chunk storage.Chunk, storeTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pullIndex.Get(shed.IndexItem{
			Address:        chunk.Address(),
			StoreTimestamp: storeTimestamp,
		})
		if err != nil {
			t.Fatal(err)
		}
		validateItem(t, item, chunk.Address(), nil, storeTimestamp, 0)
	}
}

// testPushIndex returns a test function that validates if the right
// chunk values are in the push index.
func testPushIndex(db *DB, chunk storage.Chunk, storeTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pushIndex.Get(shed.IndexItem{
			Address:        chunk.Address(),
			StoreTimestamp: storeTimestamp,
		})
		if err != nil {
			t.Fatal(err)
		}
		validateItem(t, item, chunk.Address(), nil, storeTimestamp, 0)
	}
}

// testSizeCounter returns a test function that validates the expected
// value from sizeCounter field.
func testSizeCounter(db *DB, wantSize uint64) func(t *testing.T) {
	return func(t *testing.T) {
		got, err := db.sizeCounter.Get()
		if err != nil {
			t.Fatal(err)
		}
		if got != wantSize {
			t.Errorf("got size counter value %v, want %v", got, wantSize)
		}
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
