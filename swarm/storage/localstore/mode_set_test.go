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

	"github.com/syndtr/goleveldb/leveldb"
)

// TestModeSetAccess validates internal data operations and state
// for ModeSetAccess on DB with default configuration.
func TestModeSetAccess(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeSetAccessValues(t, db)
}

// TestModeSetAccess_useRetrievalCompositeIndex validates internal
// data operations and state for ModeSetAccess on DB with
// retrieval composite index enabled.
func TestModeSetAccess_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeSetAccessValues(t, db)
}

// testModeSetAccessValues validates ModeSetAccess index values on the provided DB.
func testModeSetAccessValues(t *testing.T, db *DB) {
	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	err := db.NewSetter(ModeSetAccess).Set(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))

	t.Run("pull index count", newIndexItemsCountTest(db.pullIndex, 1))

	t.Run("gc index", newGCIndexTest(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetSync validates internal data operations and state
// for ModeSetSync on DB with default configuration.
func TestModeSetSync(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeSetSyncValues(t, db)
}

// TestModeSetSync_useRetrievalCompositeIndex validates internal
// data operations and state for ModeSetSync on DB with
// retrieval composite index enabled.
func TestModeSetSync_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeSetSyncValues(t, db)
}

// testModeSetSyncValues validates ModeSetSync index values on the provided DB.
func testModeSetSyncValues(t *testing.T, db *DB) {
	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	err = db.NewSetter(ModeSetSync).Set(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("push index", newPushIndexTest(db, chunk, wantTimestamp, leveldb.ErrNotFound))

	t.Run("gc index", newGCIndexTest(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("gc index count", newIndexItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetRemoval validates internal data operations and state
// for ModeSetRemoval on DB with default configuration.
func TestModeSetRemoval(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	testModeSetRemovalValues(t, db)
}

// TestModeSetRemoval_useRetrievalCompositeIndex validates internal
// data operations and state for ModeSetRemoval on DB with
// retrieval composite index enabled.
func TestModeSetRemoval_useRetrievalCompositeIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
	defer cleanupFunc()

	testModeSetRemovalValues(t, db)
}

// testModeSetRemovalValues validates ModeSetRemoval index values on the provided DB.
func testModeSetRemovalValues(t *testing.T, db *DB) {
	chunk := generateRandomChunk()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	err = db.NewSetter(ModeSetRemove).Set(chunk.Address())
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
