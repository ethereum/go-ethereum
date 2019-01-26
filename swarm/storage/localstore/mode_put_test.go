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
// BenchmarkPutUpload/count_100_parallel_1-addr_lock-8         	     300         5955129 ns/op     2500357 B/op	    2672 allocs/op
// BenchmarkPutUpload/count_100_parallel_1-glob_lock-8         	     300         5693210 ns/op     2480057 B/op	    2070 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-addr_lock-8         	     300         5147344 ns/op     2500580 B/op	    2673 allocs/op
// BenchmarkPutUpload/count_100_parallel_2-glob_lock-8         	     300         5801207 ns/op     2480237 B/op	    2072 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-addr_lock-8         	     500         3900634 ns/op     2500283 B/op	    2630 allocs/op
// BenchmarkPutUpload/count_100_parallel_4-glob_lock-8         	     300         5956225 ns/op     2480160 B/op	    2071 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-addr_lock-8         	     500         3204571 ns/op     2500840 B/op	    2604 allocs/op
// BenchmarkPutUpload/count_100_parallel_8-glob_lock-8         	     200         5804689 ns/op     2480354 B/op	    2073 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-addr_lock-8        	     500         3209578 ns/op     2502570 B/op	    2609 allocs/op
// BenchmarkPutUpload/count_100_parallel_16-glob_lock-8        	     300         5868150 ns/op     2480533 B/op	    2076 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-addr_lock-8        	     500         3091060 ns/op     2503923 B/op	    2634 allocs/op
// BenchmarkPutUpload/count_100_parallel_32-glob_lock-8        	     300         5620684 ns/op     2481332 B/op	    2087 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-addr_lock-8        	      20        83724617 ns/op    29397827 B/op	   26226 allocs/op
// BenchmarkPutUpload/count_1000_parallel_1-glob_lock-8        	      20        79737650 ns/op    29202973 B/op	   20228 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-addr_lock-8        	      20        73382431 ns/op    29405901 B/op	   26234 allocs/op
// BenchmarkPutUpload/count_1000_parallel_2-glob_lock-8        	      20        87743895 ns/op    29200106 B/op	   20230 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-addr_lock-8        	      20        59550383 ns/op    29397483 B/op	   25761 allocs/op
// BenchmarkPutUpload/count_1000_parallel_4-glob_lock-8        	      20        80713765 ns/op    29195823 B/op	   20232 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-addr_lock-8        	      30        54826082 ns/op    29405468 B/op	   25448 allocs/op
// BenchmarkPutUpload/count_1000_parallel_8-glob_lock-8        	      20        82545759 ns/op    29205908 B/op	   20233 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-addr_lock-8       	      30        53334438 ns/op    29406540 B/op	   25332 allocs/op
// BenchmarkPutUpload/count_1000_parallel_16-glob_lock-8       	      20        81493550 ns/op    29205267 B/op	   20233 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-addr_lock-8       	      30        51840371 ns/op    29411834 B/op	   25336 allocs/op
// BenchmarkPutUpload/count_1000_parallel_32-glob_lock-8       	      20        80898167 ns/op    29209452 B/op	   20234 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-addr_lock-8       	       2       668323148 ns/op   259038900 B/op	  280705 allocs/op
// BenchmarkPutUpload/count_10000_parallel_1-glob_lock-8       	       2       679351952 ns/op   257010124 B/op	  219969 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-addr_lock-8       	       2       666368239 ns/op   258808396 B/op	  278026 allocs/op
// BenchmarkPutUpload/count_10000_parallel_2-glob_lock-8       	       2       670005612 ns/op   256970316 B/op	  219983 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-addr_lock-8       	       2       551150500 ns/op   258527680 B/op	  272697 allocs/op
// BenchmarkPutUpload/count_10000_parallel_4-glob_lock-8       	       2       685501375 ns/op   256762796 B/op	  219901 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-addr_lock-8       	       2       518875154 ns/op   258491000 B/op	  268423 allocs/op
// BenchmarkPutUpload/count_10000_parallel_8-glob_lock-8       	       2       692095806 ns/op   256747644 B/op	  219858 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-addr_lock-8      	       2       528648421 ns/op   257939932 B/op	  264513 allocs/op
// BenchmarkPutUpload/count_10000_parallel_16-glob_lock-8      	       2       716251691 ns/op   256762568 B/op	  219120 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-addr_lock-8      	       3       473578608 ns/op   257253077 B/op	  259673 allocs/op
// BenchmarkPutUpload/count_10000_parallel_32-glob_lock-8      	       2       676274817 ns/op   256824384 B/op	  219168 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-addr_lock-8      	       1     24740576226 ns/op  2778786256 B/op	 4525586 allocs/op
// BenchmarkPutUpload/count_100000_parallel_1-glob_lock-8      	       1     24704378905 ns/op  2760701208 B/op	 3930715 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-addr_lock-8      	       1     24391650224 ns/op  2778239744 B/op	 4501266 allocs/op
// BenchmarkPutUpload/count_100000_parallel_2-glob_lock-8      	       1     25900543952 ns/op  2750693384 B/op	 3870144 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-addr_lock-8      	       1     23036622183 ns/op  2756547704 B/op	 4316307 allocs/op
// BenchmarkPutUpload/count_100000_parallel_4-glob_lock-8      	       1     25068711098 ns/op  2761207392 B/op	 3935577 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-addr_lock-8      	       1     21948692932 ns/op  2742785760 B/op	 4196817 allocs/op
// BenchmarkPutUpload/count_100000_parallel_8-glob_lock-8      	       1     24591707861 ns/op  2760381320 B/op	 3929831 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-addr_lock-8     	       1     22399527760 ns/op  2750030272 B/op	 4218608 allocs/op
// BenchmarkPutUpload/count_100000_parallel_16-glob_lock-8     	       1     24758066757 ns/op  2749799200 B/op	 3864641 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-addr_lock-8     	       1     23118686208 ns/op  2762324560 B/op	 4283463 allocs/op
// BenchmarkPutUpload/count_100000_parallel_32-glob_lock-8     	       1     25448525628 ns/op  2771420720 B/op	 3998428 allocs/op
// PASS
//
// As expected, global lock introduces performance penalty, but in much less degree then expected.
// Higher levels of parallelization do not give high level of performance boost. For 8 parallel
// uploads on 8 core benchmark, the speedup is only ~1.5x at best.
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
	errs := make(chan error)
	b.StartTimer()

	go func() {
		sem := make(chan struct{}, maxParallelUploads)
		for i := 0; i < count; i++ {
			sem <- struct{}{}

			go func() {
				defer func() { <-sem }()

				chunk := generateFakeRandomChunk()
				errs <- uploader.Put(chunk)
			}()
		}
	}()

	for i := 0; i < count; i++ {
		err := <-errs
		if err != nil {
			b.Fatal(err)
		}
	}
}
