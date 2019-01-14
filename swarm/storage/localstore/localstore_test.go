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
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// TestDB validates if the chunk can be uploaded and
// correctly retrieved.
func TestDB(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateRandomChunk()

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

	chunk := generateRandomChunk()

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

// BenchmarkNew measures the time that New function
// needs to initialize and count the number of key/value
// pairs in GC index.
// This benchmark generates a number of chunks, uploads them,
// sets them to synced state for them to enter the GC index,
// and measures the execution time of New function by creating
// new databases with the same data directory.
//
// This benchmark takes significant amount of time.
//
// Measurements on MacBook Pro (Retina, 15-inch, Mid 2014) show
// that New function executes around 1s for database with 1M chunks.
//
// # go test -benchmem -run=none github.com/ethereum/go-ethereum/swarm/storage/localstore -bench BenchmarkNew -v -timeout 20m
// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/swarm/storage/localstore
// BenchmarkNew/1000-8         	     200	  11672414 ns/op	 9570960 B/op	   10008 allocs/op
// BenchmarkNew/10000-8        	     100	  14890609 ns/op	10490118 B/op	    7759 allocs/op
// BenchmarkNew/100000-8       	      20	  58334080 ns/op	17763157 B/op	   22978 allocs/op
// BenchmarkNew/1000000-8      	       2	 748595153 ns/op	45297404 B/op	  253242 allocs/op
// PASS
func BenchmarkNew(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}
	for _, count := range []int{
		1000,
		10000,
		100000,
		1000000,
	} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			dir, err := ioutil.TempDir("", "localstore-new-benchmark")
			if err != nil {
				b.Fatal(err)
			}
			defer os.RemoveAll(dir)
			baseKey := make([]byte, 32)
			if _, err := rand.Read(baseKey); err != nil {
				b.Fatal(err)
			}
			db, err := New(dir, baseKey, nil)
			if err != nil {
				b.Fatal(err)
			}
			uploader := db.NewPutter(ModePutUpload)
			syncer := db.NewSetter(ModeSetSync)
			for i := 0; i < count; i++ {
				chunk := generateFakeRandomChunk()
				err := uploader.Put(chunk)
				if err != nil {
					b.Fatal(err)
				}
				err = syncer.Set(chunk.Address())
				if err != nil {
					b.Fatal(err)
				}
			}
			err = db.Close()
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				b.StartTimer()
				db, err := New(dir, baseKey, nil)
				b.StopTimer()

				if err != nil {
					b.Fatal(err)
				}
				err = db.Close()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
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

// generateRandomChunk generates a valid Chunk with
// data size of default chunk size.
func generateRandomChunk() storage.Chunk {
	return storage.GenerateRandomChunk(ch.DefaultSize)
}

func init() {
	// needed for generateFakeRandomChunk
	rand.Seed(time.Now().UnixNano())
}

// generateFakeRandomChunk generates a Chunk that is not
// valid, but it contains a random key and a random value.
// This function is faster then storage.GenerateRandomChunk
// which generates a valid chunk.
// Some tests in this package do not need valid chunks, just
// random data, and their execution time can be decreased
// using this function.
func generateFakeRandomChunk() storage.Chunk {
	data := make([]byte, ch.DefaultSize)
	rand.Read(data)
	key := make([]byte, 32)
	rand.Read(key)
	return storage.NewChunk(key, data)
}

// TestGenerateFakeRandomChunk validates that
// generateFakeRandomChunk returns random data by comparing
// two generated chunks.
func TestGenerateFakeRandomChunk(t *testing.T) {
	c1 := generateFakeRandomChunk()
	c2 := generateFakeRandomChunk()
	addrLen := len(c1.Address())
	if addrLen != 32 {
		t.Errorf("first chunk address length %v, want %v", addrLen, 32)
	}
	dataLen := len(c1.Data())
	if dataLen != ch.DefaultSize {
		t.Errorf("first chunk data length %v, want %v", dataLen, ch.DefaultSize)
	}
	addrLen = len(c2.Address())
	if addrLen != 32 {
		t.Errorf("second chunk address length %v, want %v", addrLen, 32)
	}
	dataLen = len(c2.Data())
	if dataLen != ch.DefaultSize {
		t.Errorf("second chunk data length %v, want %v", dataLen, ch.DefaultSize)
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
func newRetrieveIndexesTest(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
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
func newRetrieveIndexesTestWithAccess(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
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
func newPullIndexTest(db *DB, chunk storage.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
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
func newPushIndexTest(db *DB, chunk storage.Chunk, storeTimestamp int64, wantError error) func(t *testing.T) {
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
func newGCIndexTest(db *DB, chunk storage.Chunk, storeTimestamp, accessTimestamp int64) func(t *testing.T) {
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
		var want int64
		err := db.gcIndex.Iterate(func(item shed.Item) (stop bool, err error) {
			want++
			return
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := db.getGCSize()
		if got != want {
			t.Errorf("got gc size %v, want %v", got, want)
		}
	}
}

// testIndexChunk embeds storageChunk with additional data that is stored
// in database. It is used for index values validations.
type testIndexChunk struct {
	storage.Chunk
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
