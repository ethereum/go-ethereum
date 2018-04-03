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

package storage

import (
	"crypto/rand"
	"encoding/binary"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func newLDBStore(t *testing.T) (*LDBStore, func()) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	log.Trace("memstore.tempdir", "dir", dir)

	db, err := NewLDBStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, testPoFunc)
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
	ldb.setCapacity(50000)
	defer cleanup()

	memStore := NewMemStore(ldb, defaultCacheCapacity)

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
			n:         60001,
			chunkSize: 4096,
			request:   false,
		},
		{
			n:         60001,
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
		}

		for i := 0; i < tt.n; i++ {
			_, err := memStore.Get(chunks[i].Key)
			if err != nil {
				if err == ErrChunkNotFound {
					_, err := ldb.Get(chunks[i].Key)
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
		Key:        make([]byte, 32),
		ReqC:       nil,
		SData:      make([]byte, chunkSize),
		dbStoredC:  make(chan bool),
		dbStoredMu: &sync.Mutex{},
	}

	rand.Read(c.SData)

	binary.LittleEndian.PutUint64(c.SData[:8], chunkSize)

	hasher := MakeHashFunc(SHA3Hash)()
	hasher.Write(c.SData)
	copy(c.Key, hasher.Sum(nil))

	return c
}

func NewRandomRequestChunk(chunkSize uint64) *Chunk {
	c := NewRandomChunk(chunkSize)
	c.ReqC = make(chan bool)

	return c
}
