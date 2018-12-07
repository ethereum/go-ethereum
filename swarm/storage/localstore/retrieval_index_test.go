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
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// BenchmarkRetrievalIndexes compares two different retrieval
// index schemas:
// - single retrieval composite index retrievalCompositeIndex
// - two separated indexes for data and access time
//   - retrievalDataIndex
//   - retrievalAccessIndex
// This benchmark uploads a number of chunks in order to measure
// total time of updating their retrieval indexes by setting them
// to synced state and requesting them.
//
// This benchmark takes significant amount of time.
//
// Measurements on MacBook Pro (Retina, 15-inch, Mid 2014) show
// that two separated indexes perform better.
//
// # go test -benchmem -run=none github.com/ethereum/go-ethereum/swarm/storage/localstore -bench BenchmarkRetrievalIndexes -v
// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/swarm/storage/localstore
// BenchmarkRetrievalIndexes/1000-split-8         	      20       57035332 ns/op      18150318 B/op       78152 allocs/op
// BenchmarkRetrievalIndexes/1000-composite-8     	      10      145093830 ns/op      66965899 B/op       68621 allocs/op
// BenchmarkRetrievalIndexes/10000-split-8        	       1     1023919551 ns/op     376620048 B/op     1384874 allocs/op
// BenchmarkRetrievalIndexes/10000-composite-8    	       1     2612845197 ns/op    1006614104 B/op     1492380 allocs/op
// BenchmarkRetrievalIndexes/100000-split-8       	       1    14168164804 ns/op    2868944816 B/op    12425362 allocs/op
// BenchmarkRetrievalIndexes/100000-composite-8   	       1    65995988337 ns/op   12387004776 B/op    22376909 allocs/op
// PASS
func BenchmarkRetrievalIndexes(b *testing.B) {
	for _, count := range []int{
		1000,
		10000,
		100000,
	} {
		b.Run(strconv.Itoa(count)+"-split", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				benchmarkRetrievalIndexes(b, nil, count)
			}
		})
		b.Run(strconv.Itoa(count)+"-composite", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				benchmarkRetrievalIndexes(b, &Options{UseRetrievalCompositeIndex: true}, count)
			}
		})
	}
}

// benchmarkRetrievalIndexes is used in BenchmarkRetrievalIndexes
// to do benchmarks with a specific number of chunks and different
// database options.
func benchmarkRetrievalIndexes(b *testing.B, o *Options, count int) {
	b.StopTimer()
	db, cleanupFunc := newTestDB(b, o)
	defer cleanupFunc()
	uploader := db.Accessor(ModeUpload)
	syncer := db.Accessor(ModeSynced)
	requester := db.Accessor(ModeRequest)
	ctx := context.Background()
	chunks := make([]storage.Chunk, count)
	for i := 0; i < count; i++ {
		chunk := generateFakeRandomChunk()
		err := uploader.Put(ctx, chunk)
		if err != nil {
			b.Fatal(err)
		}
		chunks[i] = chunk
	}
	b.StartTimer()

	for i := 0; i < count; i++ {
		err := syncer.Put(ctx, chunks[i])
		if err != nil {
			b.Fatal(err)
		}

		_, err = requester.Get(ctx, chunks[i].Address())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUpload compares uploading speed for different
// retrieval indexes and various number of chunks.
//
// Measurements on MacBook Pro (Retina, 15-inch, Mid 2014).
//
// go test -benchmem -run=none github.com/ethereum/go-ethereum/swarm/storage/localstore -bench BenchmarkUpload -v
// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/swarm/storage/localstore
// BenchmarkUpload/1000-split-8         	      20	  99501623 ns/op      25164178 B/op    22202 allocs/op
// BenchmarkUpload/1000-composite-8     	      20	 103449118 ns/op      25177986 B/op    22204 allocs/op
// BenchmarkUpload/10000-split-8        	       2	 670290376 ns/op     216382840 B/op   239645 allocs/op
// BenchmarkUpload/10000-composite-8    	       2	 667137525 ns/op     216377176 B/op   238854 allocs/op
// BenchmarkUpload/100000-split-8       	       1	26074429894 ns/op   2326850952 B/op  3932893 allocs/op
// BenchmarkUpload/100000-composite-8   	       1	26242346728 ns/op   2331055096 B/op  3957569 allocs/op
// PASS
func BenchmarkUpload(b *testing.B) {
	for _, count := range []int{
		1000,
		10000,
		100000,
	} {
		b.Run(strconv.Itoa(count)+"-split", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				benchmarkUpload(b, nil, count)
			}
		})
		b.Run(strconv.Itoa(count)+"-composite", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				benchmarkUpload(b, &Options{UseRetrievalCompositeIndex: true}, count)
			}
		})
	}
}

// benchmarkUpload is used in BenchmarkUpload
// to do benchmarks with a specific number of chunks and different
// database options.
func benchmarkUpload(b *testing.B, o *Options, count int) {
	b.StopTimer()
	db, cleanupFunc := newTestDB(b, o)
	defer cleanupFunc()
	uploader := db.Accessor(ModeUpload)
	ctx := context.Background()
	chunks := make([]storage.Chunk, count)
	for i := 0; i < count; i++ {
		chunk := generateFakeRandomChunk()
		chunks[i] = chunk
	}
	b.StartTimer()

	for i := 0; i < count; i++ {
		err := uploader.Put(ctx, chunks[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}
