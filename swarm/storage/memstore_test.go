// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/log"
)

func newTestMemStore() *MemStore {
	storeparams := NewDefaultStoreParams()
	return NewMemStore(storeparams, nil)
}

func testMemStoreRandom(n int, chunksize int64, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreRandom(m, n, chunksize, t)
}

func testMemStoreCorrect(n int, chunksize int64, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreCorrect(m, n, chunksize, t)
}

func TestMemStoreRandom_1(t *testing.T) {
	testMemStoreRandom(1, 0, t)
}

func TestMemStoreCorrect_1(t *testing.T) {
	testMemStoreCorrect(1, 4104, t)
}

func TestMemStoreRandom_1k(t *testing.T) {
	testMemStoreRandom(1000, 0, t)
}

func TestMemStoreCorrect_1k(t *testing.T) {
	testMemStoreCorrect(100, 4096, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := newTestMemStore()
	defer m.Close()

	_, err := m.Get(context.TODO(), ZeroAddr)
	if err != ErrChunkNotFound {
		t.Errorf("Expected ErrChunkNotFound, got %v", err)
	}
}

func benchmarkMemStorePut(n int, processors int, chunksize int64, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStorePut(m, n, chunksize, b)
}

func benchmarkMemStoreGet(n int, processors int, chunksize int64, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStoreGet(m, n, chunksize, b)
}

func BenchmarkMemStorePut_1_500(b *testing.B) {
	benchmarkMemStorePut(500, 1, 4096, b)
}

func BenchmarkMemStorePut_8_500(b *testing.B) {
	benchmarkMemStorePut(500, 8, 4096, b)
}

func BenchmarkMemStoreGet_1_500(b *testing.B) {
	benchmarkMemStoreGet(500, 1, 4096, b)
}

func BenchmarkMemStoreGet_8_500(b *testing.B) {
	benchmarkMemStoreGet(500, 8, 4096, b)
}

func TestMemStoreAndLDBStore(t *testing.T) {
	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(4000)
	defer cleanup()

	cacheCap := 200
	memStore := NewMemStore(NewStoreParams(4000, 200, nil, nil), nil)

	tests := []struct {
		n         int   // number of chunks to push to memStore
		chunkSize int64 // size of chunk (by default in Swarm - 4096)
	}{
		{
			n:         1,
			chunkSize: 4096,
		},
		{
			n:         101,
			chunkSize: 4096,
		},
		{
			n:         501,
			chunkSize: 4096,
		},
		{
			n:         1100,
			chunkSize: 4096,
		},
	}

	for i, tt := range tests {
		log.Info("running test", "idx", i, "tt", tt)
		var chunks []Chunk

		for i := 0; i < tt.n; i++ {
			c := GenerateRandomChunk(tt.chunkSize)
			chunks = append(chunks, c)
		}

		for i := 0; i < tt.n; i++ {
			err := ldb.Put(context.TODO(), chunks[i])
			if err != nil {
				t.Fatal(err)
			}
			err = memStore.Put(context.TODO(), chunks[i])
			if err != nil {
				t.Fatal(err)
			}

			if got := memStore.cache.Len(); got > cacheCap {
				t.Fatalf("expected to get cache capacity less than %v, but got %v", cacheCap, got)
			}

		}

		for i := 0; i < tt.n; i++ {
			_, err := memStore.Get(context.TODO(), chunks[i].Address())
			if err != nil {
				if err == ErrChunkNotFound {
					_, err := ldb.Get(context.TODO(), chunks[i].Address())
					if err != nil {
						t.Fatalf("couldn't get chunk %v from ldb, got error: %v", i, err)
					}
				} else {
					t.Fatalf("got error from memstore: %v", err)
				}
			}
		}
	}
}
