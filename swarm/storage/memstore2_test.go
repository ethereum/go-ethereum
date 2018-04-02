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
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func newLDBStore(t *testing.T) (*LDBStore, func()) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	log.Trace("memstore.tempdir", "dir", dir)
	fmt.Println(dir)

	db, err := NewLDBStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, testPoFunc)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}

	return db, cleanup
}

func newMemStore(t *testing.T) (*MemStore, func()) {
	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(singletonSwarmDbCapacity)

	memstore := NewMemStore(ldb, defaultCacheCapacity)
	return memstore, cleanup
}

func TestMemStore(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	memStore, cleanup := newMemStore(t)
	defer cleanup()

	tests := []struct {
		n         int   // number of chunks to push to memStore
		chunkSize int64 // size of chunk (by default in Swarm - 4096)
	}{
		{
			n:         10001,
			chunkSize: 4096,
		},
	}

	for _, tt := range tests {
		var chunks []*Chunk

		for i := 0; i < tt.n; i++ {
			chunks = append(chunks, NewChunk(nil, make(chan bool)))
		}

		FakeChunk(tt.chunkSize, tt.n, chunks)

		for i := 0; i < tt.n; i++ {
			memStore.Put(chunks[i])
		}

		for i := 0; i < tt.n; i++ {
			_, err := memStore.Get(chunks[i].Key)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
