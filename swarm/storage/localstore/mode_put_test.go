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
// BenchmarkPutUpload/count_100_parallel_1-addr_lock-8         	     300	   5075184 ns/op	 2081455 B/op	    2374 allocs/op
// BenchmarkPutUpload/count_100_parallel_1-glob_lock-8         	     300	   5032374 ns/op	 2061207 B/op	    1772 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-addr_lock-8         	     300	   5079732 ns/op	 2081731 B/op	    2370 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-glob_lock-8         	     300	   5179478 ns/op	 2061380 B/op	    1773 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-addr_lock-8         	     500	   3748581 ns/op	 2081535 B/op	    2323 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-glob_lock-8         	     300	   5367513 ns/op	 2061337 B/op	    1774 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-addr_lock-8         	     500	   3311724 ns/op	 2082696 B/op	    2297 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-glob_lock-8         	     300	   5677622 ns/op	 2061636 B/op	    1776 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-addr_lock-8        	     500	   3606605 ns/op	 2085559 B/op	    2282 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-glob_lock-8        	     300	   6057814 ns/op	 2062032 B/op	    1780 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-addr_lock-8        	     500	   3720995 ns/op	 2089247 B/op	    2280 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-glob_lock-8        	     200	   6186910 ns/op	 2062744 B/op	    1789 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-addr_lock-8        	      20	  84397760 ns/op	25210142 B/op	   23222 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-glob_lock-8        	      20	  83432699 ns/op	25011813 B/op	   17222 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-addr_lock-8        	      20	  80471064 ns/op	25208653 B/op	   23182 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-glob_lock-8        	      20	  87841819 ns/op	25008899 B/op	   17223 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-addr_lock-8        	      20	  71364750 ns/op	25206981 B/op	   22704 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-glob_lock-8        	      20	  91491913 ns/op	25013307 B/op	   17225 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-addr_lock-8        	      20	  67776485 ns/op	25210323 B/op	   22315 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-glob_lock-8        	      20	  88658733 ns/op	25008864 B/op	   17228 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-addr_lock-8       	      20	  61599020 ns/op	25213746 B/op	   22000 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-glob_lock-8       	      20	  92734980 ns/op	25012744 B/op	   17228 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-addr_lock-8       	      20	  57465216 ns/op	25224471 B/op	   21844 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-glob_lock-8       	      20	  92420562 ns/op	25013237 B/op	   17244 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-addr_lock-8       	       2	 611387455 ns/op	216747724 B/op	  248218 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-glob_lock-8       	       2	 616212255 ns/op	214871528 B/op	  188983 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-addr_lock-8       	       2	 576871975 ns/op	216552736 B/op	  246849 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-glob_lock-8       	       2	 601008305 ns/op	214713748 B/op	  188931 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-addr_lock-8       	       2	 551001371 ns/op	216701032 B/op	  241935 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-glob_lock-8       	       2	 605576690 ns/op	214719292 B/op	  188949 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-addr_lock-8       	       2	 504949238 ns/op	216431280 B/op	  236326 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-glob_lock-8       	       2	 611631748 ns/op	214809276 B/op	  188957 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-addr_lock-8      	       3	 510030296 ns/op	216088080 B/op	  231171 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-glob_lock-8      	       2	 611416284 ns/op	214855916 B/op	  189724 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-addr_lock-8      	       3	 481631118 ns/op	215341840 B/op	  224716 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-glob_lock-8      	       2	 633612977 ns/op	214904164 B/op	  189775 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-addr_lock-8      	       1	23289076334 ns/op	2354337552 B/op	 4190917 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-glob_lock-8      	       1	22155535580 ns/op	2312803760 B/op	 3455566 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-addr_lock-8      	       1	21908455154 ns/op	2328191128 B/op	 4014009 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-glob_lock-8      	       1	22956308053 ns/op	2325078528 B/op	 3530270 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-addr_lock-8      	       1	22334786914 ns/op	2338677488 B/op	 4028700 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-glob_lock-8      	       1	23222406988 ns/op	2334153480 B/op	 3580197 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-addr_lock-8      	       1	21569685948 ns/op	2322310120 B/op	 3880022 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-glob_lock-8      	       1	22730998001 ns/op	2318311616 B/op	 3494378 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-addr_lock-8     	       1	22005406658 ns/op	2324345744 B/op	 3862100 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-glob_lock-8     	       1	24246335163 ns/op	2341373784 B/op	 3626749 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-addr_lock-8     	       1	22764682771 ns/op	2332867552 B/op	 3896808 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-glob_lock-8     	       1	24617688531 ns/op	2343609240 B/op	 3647404 allocs/op
// PASS
//
// As expected, global lock introduces performance penalty, but in much less degree then expected.
// Higher levels of parallelization do not give high level of performance boost. For 8 parallel
// uploads on 8 core benchmark, the speedup is only ~1.72x at best. There is no significant difference
// when a larger number of chunks is uploaded.
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
			b.Run(name+"-addr_lock", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					benchmarkPutUpload(b, nil, count, maxParallelUploads)
				}
			})
			b.Run(name+"-glob_lock", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					benchmarkPutUpload(b, &Options{useGlobalLock: true}, count, maxParallelUploads)
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
