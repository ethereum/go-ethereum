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
	"sync"
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
	var chunksMu sync.Mutex

	// send chunks to workers
	go func() {
		for i := 0; i < chunkCount; i++ {
			chunk := generateRandomChunk()
			select {
			case chunkChan <- chunk:
			case <-doneChan:
				return
			}
			chunksMu.Lock()
			chunks = append(chunks, chunk)
			chunksMu.Unlock()
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

	chunksMu.Lock()
	defer chunksMu.Unlock()
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

// BenchmarkPutUpload runs a series of benchmarks that upload
// a specific number of chunks in parallel.
//
// Measurements on MacBook Pro (Retina, 15-inch, Mid 2014)
//
// # go test -benchmem -run=none github.com/ethereum/go-ethereum/swarm/storage/localstore -bench BenchmarkPutUpload -v
//
// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/swarm/storage/localstore
// BenchmarkPutUpload/count_100_parallel_1-8         	     300	   4955055 ns/op	 2061388 B/op	    1754 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-8         	     300	   5162484 ns/op	 2061452 B/op	    1755 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-8         	     300	   5260477 ns/op	 2061655 B/op	    1756 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-8         	     300	   5381812 ns/op	 2061843 B/op	    1758 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-8        	     300	   5477313 ns/op	 2062115 B/op	    1762 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-8        	     300	   5565273 ns/op	 2062965 B/op	    1775 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-8        	      20	  75632247 ns/op	25009474 B/op	   17204 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-8        	      20	  78194544 ns/op	25009064 B/op	   17205 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-8        	      20	  77413001 ns/op	25010023 B/op	   17206 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-8        	      20	  77406586 ns/op	25010968 B/op	   17206 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-8       	      20	  81943323 ns/op	25006622 B/op	   17209 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-8       	      20	  84393475 ns/op	25009450 B/op	   17222 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-8       	       2	 612973544 ns/op	214429212 B/op	  186539 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-8       	       2	 613744836 ns/op	214525364 B/op	  188857 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-8       	       2	 619848337 ns/op	214437448 B/op	  188043 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-8       	       2	 612132728 ns/op	214492440 B/op	  188061 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-8      	       2	 625959679 ns/op	214493172 B/op	  188840 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-8      	       2	 652223974 ns/op	214648080 B/op	  188916 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-8      	       1	22682989072 ns/op	2317757256 B/op	 3486655 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-8      	       1	23928779747 ns/op	2339295256 B/op	 3621696 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-8      	       1	22704591819 ns/op	2317971752 B/op	 3495423 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-8      	       1	22654015490 ns/op	2320451336 B/op	 3506505 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-8     	       1	23192344424 ns/op	2326781648 B/op	 3540538 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-8     	       1	24188298331 ns/op	2344201416 B/op	 3651945 allocs/op
// PASS
func BenchmarkPutUpload(b *testing.B) {
	for _, count := range []int{
		100,
		1000,
		10000,
		100000,
	} {
		for _, maxParallelUploads := range []int{
			1,
			2,
			4,
			8,
			16,
			32,
		} {
			name := fmt.Sprintf("count %v parallel %v", count, maxParallelUploads)
			b.Run(name, func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					benchmarkPutUpload(b, nil, count, maxParallelUploads)
				}
			})
		}
	}
}

// benchmarkPutUpload runs a benchmark by uploading a specific number
// of chunks with specified max parallel uploads.
func benchmarkPutUpload(b *testing.B, o *Options, count, maxParallelUploads int) {
	b.StopTimer()
	db, cleanupFunc := newTestDB(b, o)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)
	chunks := make([]storage.Chunk, count)
	for i := 0; i < count; i++ {
		chunks[i] = generateFakeRandomChunk()
	}
	errs := make(chan error)
	b.StartTimer()

	go func() {
		sem := make(chan struct{}, maxParallelUploads)
		for i := 0; i < count; i++ {
			sem <- struct{}{}

			go func(i int) {
				defer func() { <-sem }()

				errs <- uploader.Put(chunks[i])
			}(i)
		}
	}()

	for i := 0; i < count; i++ {
		err := <-errs
		if err != nil {
			b.Fatal(err)
		}
	}
}
