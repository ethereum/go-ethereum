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
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestDB_useRetrievalCompositeIndex checks if optional argument
// WithRetrievalCompositeIndex to New constructor is setting the
// correct state.
func TestDB_useRetrievalCompositeIndex(t *testing.T) {
	t.Run("set true", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
		defer cleanupFunc()

		if !db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to true")
		}
	})
	t.Run("set false", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: false})
		defer cleanupFunc()

		if db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to false")
		}
	})
	t.Run("unset", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, nil)
		defer cleanupFunc()

		if db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to false")
		}
	})
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
// BenchmarkNew/1000-8         	     200	  12020231 ns/op	 9556077 B/op	    9999 allocs/op
// BenchmarkNew/10000-8        	     100	  15475883 ns/op	10493071 B/op	    7781 allocs/op
// BenchmarkNew/100000-8       	      20	  64046466 ns/op	17823841 B/op	   23375 allocs/op
// BenchmarkNew/1000000-8      	       1	1011464203 ns/op	51024688 B/op	  310599 allocs/op
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
			uploader := db.Accessor(ModeUpload)
			syncer := db.Accessor(ModeSynced)
			ctx := context.Background()
			for i := 0; i < count; i++ {
				chunk := generateFakeRandomChunk()
				err := uploader.Put(ctx, chunk)
				if err != nil {
					b.Fatal(err)
				}
				err = syncer.Put(ctx, chunk)
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
	if bytes.Equal(c1.Address(), c2.Address()) {
		t.Error("fake chunks addresses do not differ")
	}
	if bytes.Equal(c1.Data(), c2.Data()) {
		t.Error("fake chunks data bytes do not differ")
	}
}
