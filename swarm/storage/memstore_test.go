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
	"crypto/rand"
	"encoding/binary"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/log"
)

func newTestMemStore() *MemStore {
	storeparams := NewDefaultStoreParams()
	return NewMemStore(storeparams, nil)
}

func testMemStoreRandom(n int, processors int, chunksize int64, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreRandom(m, processors, n, chunksize, t)
}

func testMemStoreCorrect(n int, processors int, chunksize int64, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreCorrect(m, processors, n, chunksize, t)
}

func TestMemStoreRandom_1(t *testing.T) {
	testMemStoreRandom(1, 1, 0, t)
}

func TestMemStoreCorrect_1(t *testing.T) {
	testMemStoreCorrect(1, 1, 4104, t)
}

func TestMemStoreRandom_1_1k(t *testing.T) {
	testMemStoreRandom(1, 1000, 0, t)
}

func TestMemStoreCorrect_1_1k(t *testing.T) {
	testMemStoreCorrect(1, 100, 4096, t)
}

func TestMemStoreRandom_8_1k(t *testing.T) {
	testMemStoreRandom(8, 1000, 0, t)
}

func TestMemStoreCorrect_8_1k(t *testing.T) {
	testMemStoreCorrect(8, 1000, 4096, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := newTestMemStore()
	defer m.Close()

	_, err := m.Get(ZeroAddr)
	if err != ErrChunkNotFound {
		t.Errorf("Expected ErrChunkNotFound, got %v", err)
	}
}

func benchmarkMemStorePut(n int, processors int, chunksize int64, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStorePut(m, processors, n, chunksize, b)
}

func benchmarkMemStoreGet(n int, processors int, chunksize int64, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStoreGet(m, processors, n, chunksize, b)
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

func newLDBStore(t *testing.T) (*LDBStore, func()) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	log.Trace("memstore.tempdir", "dir", dir)

	ldbparams := NewLDBStoreParams(NewDefaultStoreParams(), dir)
	db, err := NewLDBStore(ldbparams)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		db.Close()
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}

	return db, cleanup
}

func TestMemStoreAndLDBStore(t *testing.T) {
	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(4000)
	defer cleanup()

	cacheCap := 200
	requestsCap := 200
	memStore := NewMemStore(NewStoreParams(4000, 200, 200, nil, nil), nil)

	tests := []struct {
		n         int    // number of chunks to push to memStore
		chunkSize uint64 // size of chunk (by default in Swarm - 4096)
		request   bool   // whether or not to set the ReqC channel on the random chunks
	}{
		{
			n:         1,
			chunkSize: 4096,
			request:   false,
		},
		{
			n:         201,
			chunkSize: 4096,
			request:   false,
		},
		{
			n:         501,
			chunkSize: 4096,
			request:   false,
		},
		{
			n:         3100,
			chunkSize: 4096,
			request:   false,
		},
		{
			n:         100,
			chunkSize: 4096,
			request:   true,
		},
	}

	for i, tt := range tests {
		log.Info("running test", "idx", i, "tt", tt)
		var chunks []*Chunk

		for i := 0; i < tt.n; i++ {
			var c *Chunk
			if tt.request {
				c = NewRandomRequestChunk(tt.chunkSize)
			} else {
				c = NewRandomChunk(tt.chunkSize)
			}

			chunks = append(chunks, c)
		}

		for i := 0; i < tt.n; i++ {
			go ldb.Put(chunks[i])
			memStore.Put(chunks[i])

			if got := memStore.cache.Len(); got > cacheCap {
				t.Fatalf("expected to get cache capacity less than %v, but got %v", cacheCap, got)
			}

			if got := memStore.requests.Len(); got > requestsCap {
				t.Fatalf("expected to get requests capacity less than %v, but got %v", requestsCap, got)
			}
		}

		for i := 0; i < tt.n; i++ {
			_, err := memStore.Get(chunks[i].Addr)
			if err != nil {
				if err == ErrChunkNotFound {
					_, err := ldb.Get(chunks[i].Addr)
					if err != nil {
						t.Fatalf("couldn't get chunk %v from ldb, got error: %v", i, err)
					}
				} else {
					t.Fatalf("got error from memstore: %v", err)
				}
			}
		}

		// wait for all chunks to be stored before ending the test are cleaning up
		for i := 0; i < tt.n; i++ {
			<-chunks[i].dbStoredC
		}
	}
}

func NewRandomChunk(chunkSize uint64) *Chunk {
	c := &Chunk{
		Addr:       make([]byte, 32),
		ReqC:       nil,
		SData:      make([]byte, chunkSize+8), // SData should be chunkSize + 8 bytes reserved for length
		dbStoredC:  make(chan bool),
		dbStoredMu: &sync.Mutex{},
	}

	rand.Read(c.SData)

	binary.LittleEndian.PutUint64(c.SData[:8], chunkSize)

	hasher := MakeHashFunc(SHA3Hash)()
	hasher.Write(c.SData)
	copy(c.Addr, hasher.Sum(nil))

	return c
}

func NewRandomRequestChunk(chunkSize uint64) *Chunk {
	c := NewRandomChunk(chunkSize)
	c.ReqC = make(chan bool)

	return c
}
