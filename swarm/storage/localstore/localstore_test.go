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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/syndtr/goleveldb/leveldb"
)

func init() {
	// Some of the tests in localstore package rely on the same ordering of
	// items uploaded or accessed compared to the ordering of items in indexes
	// that contain StoreTimestamp or AccessTimestamp in keys. In tests
	// where the same order is required from the database as the order
	// in which chunks are put or accessed, if the StoreTimestamp or
	// AccessTimestamp are the same for two or more sequential items
	// their order in database will be based on the chunk address value,
	// in which case the ordering of items/chunks stored in a test slice
	// will not be the same. To ensure the same ordering in database on such
	// indexes on windows systems, an additional short sleep is added to
	// the now function.
	if runtime.GOOS == "windows" {
		setNow(func() int64 {
			time.Sleep(time.Microsecond)
			return time.Now().UTC().UnixNano()
		})
	}
}

// TestDB validates if the chunk can be uploaded and
// correctly retrieved.
func TestDB(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateTestRandomChunk()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	got, err := db.NewGetter(ModeGetRequest).Get(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got.Address(), chunk.Address()) {
		t.Errorf("got address %x, want %x", got.Address(), chunk.Address())
	}
	if !bytes.Equal(got.Data(), chunk.Data()) {
		t.Errorf("got data %x, want %x", got.Data(), chunk.Data())
	}
}

// TestDB_updateGCSem tests maxParallelUpdateGC limit.
// This test temporary sets the limit to a low number,
// makes updateGC function execution time longer by
// setting a custom testHookUpdateGC function with a sleep
// and a count current and maximal number of goroutines.
func TestDB_updateGCSem(t *testing.T) {
	updateGCSleep := time.Second
	var count int
	var max int
	var mu sync.Mutex
	defer setTestHookUpdateGC(func() {
		mu.Lock()
		// add to the count of current goroutines
		count++
		if count > max {
			// set maximal detected numbers of goroutines
			max = count
		}
		mu.Unlock()

		// wait for some time to ensure multiple parallel goroutines
		time.Sleep(updateGCSleep)

		mu.Lock()
		count--
		mu.Unlock()
	})()

	defer func(m int) { maxParallelUpdateGC = m }(maxParallelUpdateGC)
	maxParallelUpdateGC = 3

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateTestRandomChunk()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	getter := db.NewGetter(ModeGetRequest)

	// get more chunks then maxParallelUpdateGC
	// in time shorter then updateGCSleep
	for i := 0; i < 5; i++ {
		_, err = getter.Get(chunk.Address())
		if err != nil {
			t.Fatal(err)
		}
	}

	if max != maxParallelUpdateGC {
		t.Errorf("got max %v, want %v", max, maxParallelUpdateGC)
	}
}

// newTestDB is a helper function that constructs a
// temporary database and returns a cleanup function that must
// be called to remove the data.
func newTestDB(t testing.TB, o *Options) (db *DB, cleanupFunc func()) {
	t.Helper()

	dir, err := ioutil.TempDir("", "localstore-test")
	if err != nil {
		t.Fatal(err)
	}
	cleanupFunc = func() { os.RemoveAll(dir) }
	baseKey := make([]byte, 32)
	if _, err := rand.Read(baseKey); err != nil {
		t.Fatal(err)
	}
	db, err = New(dir, baseKey, o)
	if err != nil {
		cleanupFunc()
		t.Fatal(err)
	}
	cleanupFunc = func() {
		err := db.Close()
		if err != nil {
			t.Error(err)
		}
		os.RemoveAll(dir)
	}
	return db, cleanupFunc
}

func init() {
	// needed for generateTestRandomChunk
	rand.Seed(time.Now().UnixNano())
}

// generateTestRandomChunk generates a Chunk that is not
// valid, but it contains a random key and a random value.
// This function is faster then storage.generateTestRandomChunk
// which generates a valid chunk.
// Some tests in this package do not need valid chunks, just
// random data, and their execution time can be decreased
// using this function.
func generateTestRandomChunk() chunk.Chunk {
	data := make([]byte, chunk.DefaultSize)
	rand.Read(data)
	key := make([]byte, 32)
	rand.Read(key)
	return chunk.NewChunk(key, data)
}

// TestGenerateTestRandomChunk validates that
// generateTestRandomChunk returns random data by comparing
// two generated chunks.
func TestGenerateTestRandomChunk(t *testing.T) {
	c1 := generateTestRandomChunk()
	c2 := generateTestRandomChunk()
	addrLen := len(c1.Address())
	if addrLen != 32 {
		t.Errorf("first chunk address length %v, want %v", addrLen, 32)
	}
	dataLen := len(c1.Data())
	if dataLen != chunk.DefaultSize {
		t.Errorf("first chunk data length %v, want %v", dataLen, chunk.DefaultSize)
	}
	addrLen = len(c2.Address())
	if addrLen != 32 {
		t.Errorf("second chunk address length %v, want %v", addrLen, 32)
	}
	dataLen = len(c2.Data())
	if dataLen != chunk.DefaultSize {
		t.Errorf("second chunk data length %v, want %v", dataLen, chunk.DefaultSize)
	}
	if bytes.Equal(c1.Address(), c2.Address()) {
		t.Error("fake chunks addresses do not differ")
	}
	if bytes.Equal(c1.Data(), c2.Data()) {
		t.Error("fake chunks data bytes do not differ")
	}
}

// newRetrieveIndexesTest returns a test function that validates if the right
// chunk values are in the retrieval indexes.
func newRetrieveIndexesTest(db *DB, chunk chunk.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
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

// newRetrieveIndexesTestWithAccess returns a test function that validates if the right
// chunk values are in the retrieval indexes when access time must be stored.
func newRetrieveIndexesTestWithAccess(db *DB, chunk chunk.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
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

// newPullIndexTest returns a test function that validates if the right
// chunk values are in the pull index.
func newPullIndexTest(db *DB, chunk chunk.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pullIndex.Get(shed.Item{
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
func newPushIndexTest(db *DB, chunk chunk.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.pushIndex.Get(shed.Item{
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
func newGCIndexTest(db *DB, chunk chunk.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := db.gcIndex.Get(shed.Item{
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

// newItemsCountTest returns a test function that validates if
// an index contains expected number of key/value pairs.
func newItemsCountTest(i shed.Index, want int) func(t *testing.T) {
	return func(t *testing.T) {
		var c int
		err := i.Iterate(func(item shed.Item) (stop bool, err error) {
			c++
			return
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if c != want {
			t.Errorf("got %v items in index, want %v", c, want)
		}
	}
}

// newIndexGCSizeTest retruns a test function that validates if DB.gcSize
// value is the same as the number of items in DB.gcIndex.
func newIndexGCSizeTest(db *DB) func(t *testing.T) {
	return func(t *testing.T) {
		var want uint64
		err := db.gcIndex.Iterate(func(item shed.Item) (stop bool, err error) {
			want++
			return
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := db.gcSize.Get()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("got gc size %v, want %v", got, want)
		}
	}
}

// testIndexChunk embeds storageChunk with additional data that is stored
// in database. It is used for index values validations.
type testIndexChunk struct {
	chunk.Chunk
	storeTimestamp int64
}

// testItemsOrder tests the order of chunks in the index. If sortFunc is not nil,
// chunks will be sorted with it before validation.
func testItemsOrder(t *testing.T, i shed.Index, chunks []testIndexChunk, sortFunc func(i, j int) (less bool)) {
	newItemsCountTest(i, len(chunks))(t)

	if sortFunc != nil {
		sort.Slice(chunks, sortFunc)
	}

	var cursor int
	err := i.Iterate(func(item shed.Item) (stop bool, err error) {
		want := chunks[cursor].Address()
		got := item.Address
		if !bytes.Equal(got, want) {
			return true, fmt.Errorf("got address %x at position %v, want %x", got, cursor, want)
		}
		cursor++
		return false, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// validateItem is a helper function that checks Item values.
func validateItem(t *testing.T, item shed.Item, address, data []byte, storeTimestamp, accessTimestamp int64) {
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

// setNow replaces now function and
// returns a function that will reset it to the
// value before the change.
func setNow(f func() int64) (reset func()) {
	current := now
	reset = func() { now = current }
	now = f
	return reset
}

// TestSetNow tests if setNow function changes now function
// correctly and if its reset function resets the original function.
func TestSetNow(t *testing.T) {
	// set the current function after the test finishes
	defer func(f func() int64) { now = f }(now)

	// expected value for the unchanged function
	var original int64 = 1
	// expected value for the changed function
	var changed int64 = 2

	// define the original (unchanged) functions
	now = func() int64 {
		return original
	}

	// get the time
	got := now()

	// test if got variable is set correctly
	if got != original {
		t.Errorf("got now value %v, want %v", got, original)
	}

	// set the new function
	reset := setNow(func() int64 {
		return changed
	})

	// get the time
	got = now()

	// test if got variable is set correctly to changed value
	if got != changed {
		t.Errorf("got hook value %v, want %v", got, changed)
	}

	// set the function to the original one
	reset()

	// get the time
	got = now()

	// test if got variable is set correctly to original value
	if got != original {
		t.Errorf("got hook value %v, want %v", got, original)
	}
}
