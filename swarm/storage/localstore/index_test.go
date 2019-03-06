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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/chunk"
)

// TestDB_pullIndex validates the ordering of keys in pull index.
// Pull index key contains PO prefix which is calculated from
// DB base key and chunk address. This is not an Item field
// which are checked in Mode tests.
// This test uploads chunks, sorts them in expected order and
// validates that pull index iterator will iterate it the same
// order.
func TestDB_pullIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	chunkCount := 50

	chunks := make([]testIndexChunk, chunkCount)

	// upload random chunks
	for i := 0; i < chunkCount; i++ {
		chunk := generateTestRandomChunk()

		err := uploader.Put(chunk)
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

	testItemsOrder(t, db.pullIndex, chunks, func(i, j int) (less bool) {
		poi := chunk.Proximity(db.baseKey, chunks[i].Address())
		poj := chunk.Proximity(db.baseKey, chunks[j].Address())
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

// TestDB_gcIndex validates garbage collection index by uploading
// a chunk with and performing operations using synced, access and
// request modes.
func TestDB_gcIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	chunkCount := 50

	chunks := make([]testIndexChunk, chunkCount)

	// upload random chunks
	for i := 0; i < chunkCount; i++ {
		chunk := generateTestRandomChunk()

		err := uploader.Put(chunk)
		if err != nil {
			t.Fatal(err)
		}

		chunks[i] = testIndexChunk{
			Chunk: chunk,
		}
	}

	// check if all chunks are stored
	newItemsCountTest(db.pullIndex, chunkCount)(t)

	// check that chunks are not collectable for garbage
	newItemsCountTest(db.gcIndex, 0)(t)

	// set update gc test hook to signal when
	// update gc goroutine is done by sending to
	// testHookUpdateGCChan channel, which is
	// used to wait for indexes change verifications
	testHookUpdateGCChan := make(chan struct{})
	defer setTestHookUpdateGC(func() {
		testHookUpdateGCChan <- struct{}{}
	})()

	t.Run("request unsynced", func(t *testing.T) {
		chunk := chunks[1]

		_, err := db.NewGetter(ModeGetRequest).Get(chunk.Address())
		if err != nil {
			t.Fatal(err)
		}
		// wait for update gc goroutine to be done
		<-testHookUpdateGCChan

		// the chunk is not synced
		// should not be in the garbace collection index
		newItemsCountTest(db.gcIndex, 0)(t)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("sync one chunk", func(t *testing.T) {
		chunk := chunks[0]

		err := db.NewSetter(ModeSetSync).Set(chunk.Address())
		if err != nil {
			t.Fatal(err)
		}

		// the chunk is synced and should be in gc index
		newItemsCountTest(db.gcIndex, 1)(t)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("sync all chunks", func(t *testing.T) {
		setter := db.NewSetter(ModeSetSync)

		for i := range chunks {
			err := setter.Set(chunks[i].Address())
			if err != nil {
				t.Fatal(err)
			}
		}

		testItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("request one chunk", func(t *testing.T) {
		i := 6

		_, err := db.NewGetter(ModeGetRequest).Get(chunks[i].Address())
		if err != nil {
			t.Fatal(err)
		}
		// wait for update gc goroutine to be done
		<-testHookUpdateGCChan

		// move the chunk to the end of the expected gc
		c := chunks[i]
		chunks = append(chunks[:i], chunks[i+1:]...)
		chunks = append(chunks, c)

		testItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("random chunk request", func(t *testing.T) {
		requester := db.NewGetter(ModeGetRequest)

		rand.Shuffle(len(chunks), func(i, j int) {
			chunks[i], chunks[j] = chunks[j], chunks[i]
		})

		for _, chunk := range chunks {
			_, err := requester.Get(chunk.Address())
			if err != nil {
				t.Fatal(err)
			}
			// wait for update gc goroutine to be done
			<-testHookUpdateGCChan
		}

		testItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})

	t.Run("remove one chunk", func(t *testing.T) {
		i := 3

		err := db.NewSetter(modeSetRemove).Set(chunks[i].Address())
		if err != nil {
			t.Fatal(err)
		}

		// remove the chunk from the expected chunks in gc index
		chunks = append(chunks[:i], chunks[i+1:]...)

		testItemsOrder(t, db.gcIndex, chunks, nil)

		newIndexGCSizeTest(db)(t)
	})
}
