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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/syndtr/goleveldb/leveldb"
)

// TestModeSetAccess validates ModeSetAccess index values on the provided DB.
func TestModeSetAccess(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	ch := generateTestRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	err := db.Set(context.Background(), chunk.ModeSetAccess, ch.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("pull index", newPullIndexTest(db, ch, 1, nil))

	t.Run("pull index count", newItemsCountTest(db.pullIndex, 1))

	t.Run("gc index", newGCIndexTest(db, ch, wantTimestamp, wantTimestamp, 1))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetSync validates ModeSetSync index values on the provided DB.
func TestModeSetSync(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	ch := generateTestRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Set(context.Background(), chunk.ModeSetSync, ch.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, ch, wantTimestamp, wantTimestamp))

	t.Run("push index", newPushIndexTest(db, ch, wantTimestamp, leveldb.ErrNotFound))

	t.Run("gc index", newGCIndexTest(db, ch, wantTimestamp, wantTimestamp, 1))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetRemove validates ModeSetRemove index values on the provided DB.
func TestModeSetRemove(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	ch := generateTestRandomChunk()

	_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Set(context.Background(), chunk.ModeSetRemove, ch.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", func(t *testing.T) {
		wantErr := leveldb.ErrNotFound
		_, err := db.retrievalDataIndex.Get(addressToItem(ch.Address()))
		if err != wantErr {
			t.Errorf("got error %v, want %v", err, wantErr)
		}
		t.Run("retrieve data index count", newItemsCountTest(db.retrievalDataIndex, 0))

		// access index should not be set
		_, err = db.retrievalAccessIndex.Get(addressToItem(ch.Address()))
		if err != wantErr {
			t.Errorf("got error %v, want %v", err, wantErr)
		}
		t.Run("retrieve access index count", newItemsCountTest(db.retrievalAccessIndex, 0))
	})

	t.Run("pull index", newPullIndexTest(db, ch, 0, leveldb.ErrNotFound))

	t.Run("pull index count", newItemsCountTest(db.pullIndex, 0))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 0))

	t.Run("gc size", newIndexGCSizeTest(db))
}
