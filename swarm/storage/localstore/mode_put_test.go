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

	"github.com/ethereum/go-ethereum/swarm/chunk"
)

// TestModePutRequest validates ModePutRequest index values on the provided DB.
func TestModePutRequest(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	putter := db.NewPutter(ModePutRequest)

	chunk := generateTestRandomChunk()

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

	chunk := generateTestRandomChunk()

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

	chunk := generateTestRandomChunk()

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

	chunkChan := make(chan chunk.Chunk)
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

	chunks := make([]chunk.Chunk, 0)
	var chunksMu sync.Mutex

	// send chunks to workers
	go func() {
		for i := 0; i < chunkCount; i++ {
			chunk := generateTestRandomChunk()
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
// BenchmarkPutUpload/count_100_parallel_1-8         	     300	   5107704 ns/op	 2081461 B/op	    2374 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-8         	     300	   5411742 ns/op	 2081608 B/op	    2364 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-8         	     500	   3704964 ns/op	 2081696 B/op	    2324 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-8         	     500	   2932663 ns/op	 2082594 B/op	    2295 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-8        	     500	   3117157 ns/op	 2085438 B/op	    2282 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-8        	     500	   3449122 ns/op	 2089721 B/op	    2286 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-8        	      20	  79784470 ns/op	25211240 B/op	   23225 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-8        	      20	  75422164 ns/op	25210730 B/op	   23187 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-8        	      20	  70698378 ns/op	25206522 B/op	   22692 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-8        	      20	  71285528 ns/op	25213436 B/op	   22345 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-8       	      20	  71301826 ns/op	25205040 B/op	   22090 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-8       	      30	  57713506 ns/op	25219781 B/op	   21848 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-8       	       2	 656719345 ns/op	216792908 B/op	  248940 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-8       	       2	 646301962 ns/op	216730800 B/op	  248270 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-8       	       2	 532784228 ns/op	216667080 B/op	  241910 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-8       	       3	 494290188 ns/op	216297749 B/op	  236247 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-8      	       3	 483485315 ns/op	216060384 B/op	  231090 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-8      	       3	 434461294 ns/op	215371280 B/op	  224800 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-8      	       1	22767894338 ns/op	2331372088 B/op	 4049876 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-8      	       1	25347872677 ns/op	2344140160 B/op	 4106763 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-8      	       1	23580460174 ns/op	2338582576 B/op	 4027452 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-8      	       1	22197559193 ns/op	2321803496 B/op	 3877553 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-8     	       1	22527046476 ns/op	2327854800 B/op	 3885455 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-8     	       1	21332243613 ns/op	2299654568 B/op	 3697181 allocs/op
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
	chunks := make([]chunk.Chunk, count)
	for i := 0; i < count; i++ {
		chunks[i] = generateTestRandomChunk()
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
