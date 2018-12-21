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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestModePutRequest validates ModePutRequest index values on the provided DB.
func TestModePutRequest(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

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

		t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

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

		t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

		t.Run("gc size", newIndexGCSizeTest(db))
	})
}

// TestModePutSync validates ModePutSync index values on the provided DB.
func TestModePutSync(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

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

// TestModePutUpload validates ModePutUpload index values on the provided DB.
func TestModePutUpload(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

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

// TestModePutUpload_parallel uploads chunks in parallel
// and validates if all chunks can be retrieved with correct data.
func TestModePutUpload_parallel(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunkCount := 1000
	workerCount := 100

	chunkChan := make(chan storage.Chunk)
	errChan := make(chan error)
	doneChan := make(chan struct{})
	defer close(doneChan)

	// start uploader workers
	for i := 0; i < workerCount; i++ {
		go func(i int) {
			uploader := db.NewPutter(ModePutUpload)
			for {
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						return
					}
					err := uploader.Put(chunk)
					select {
					case errChan <- err:
					case <-doneChan:
					}
				case <-doneChan:
					return
				}
			}
		}(i)
	}

	chunks := make([]storage.Chunk, 0)

	// send chunks to workers
	go func() {
		for i := 0; i < chunkCount; i++ {
			chunk := generateRandomChunk()
			select {
			case chunkChan <- chunk:
			case <-doneChan:
				return
			}
			chunks = append(chunks, chunk)
		}

		close(chunkChan)
	}()

	// validate every error from workers
	for i := 0; i < chunkCount; i++ {
		err := <-errChan
		if err != nil {
			t.Fatal(err)
		}
	}

	// get every chunk and validate its data
	getter := db.NewGetter(ModeGetRequest)
	for _, chunk := range chunks {
		got, err := getter.Get(chunk.Address())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Data(), chunk.Data()) {
			t.Fatalf("got chunk %s data %x, want %x", chunk.Address().Hex(), got.Data(), chunk.Data())
		}
	}
}
