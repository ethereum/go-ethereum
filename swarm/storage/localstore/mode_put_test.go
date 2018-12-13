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
	"testing"
	"time"
)

// TestModePutRequest validates internal data operations and state
// for ModePutRequest on DB with default configuration.
func TestModePutRequest(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModePutRequestValues(t, db)
}

// TestModePutRequest_useRetrievalCompositeIndex validates internal
// data operations and state for ModePutRequest on DB with
// retrieval composite index enabled.
func TestModePutRequest_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModePutRequestValues(t, db)
}

// testModePutRequestValues validates ModePutRequest index values on the provided DB.
func testModePutRequestValues(t *testing.T, db *DB) {
	putter := db.NewPutter(ModePutRequest)

	chunk := generateRandomChunk()

	// keep the record when the chunk is stored
	var storeTimestamp int64

	t.Run("first put", func(t *testing.T) {
		wantTimestamp := time.Now().UTC().UnixNano()
		defer setNow(func() (t int64) {
			return wantTimestamp
		})()

		storeTimestamp = wantTimestamp

		err := putter.Put(chunk)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, wantTimestamp, wantTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})

	t.Run("second put", func(t *testing.T) {
		wantTimestamp := time.Now().UTC().UnixNano()
		defer setNow(func() (t int64) {
			return wantTimestamp
		})()

		err := putter.Put(chunk)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, storeTimestamp, wantTimestamp))

		t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})
}

// TestModePutSync validates internal data operations and state
// for ModePutSync on DB with default configuration.
func TestModePutSync(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModePutSyncValues(t, db)
}

// TestModePutSync_useRetrievalCompositeIndex validates internal
// data operations and state for ModePutSync on DB with
// retrieval composite index enabled.
func TestModePutSync_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModePutSyncValues(t, db)
}

// testModePutSyncValues validates ModePutSync index values on the provided DB.
func testModePutSyncValues(t *testing.T, db *DB) {
	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	chunk := generateRandomChunk()

	err := db.NewPutter(ModePutSync).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTest(db, chunk, wantTimestamp, 0))

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))
}

// TestModePutUpload validates internal data operations and state
// for ModePutUpload on DB with default configuration.
func TestModePutUpload(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModePutUploadValues(t, db)
}

// TestModePutUpload_useRetrievalCompositeIndex validates internal
// data operations and state for ModePutUpload on DB with
// retrieval composite index enabled.
func TestModePutUpload_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModePutUploadValues(t, db)
}

// testModePutUploadValues validates ModePutUpload index values on the provided DB.
func testModePutUploadValues(t *testing.T, db *DB) {
	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	chunk := generateRandomChunk()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTest(db, chunk, wantTimestamp, 0))

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))

	t.Run("push index", newPushIndexTest(db, chunk, wantTimestamp, nil))
}
