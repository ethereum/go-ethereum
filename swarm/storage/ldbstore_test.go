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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
)

type testDbStore struct {
	*LDBStore
	dir string
}

func newTestDbStore(mock bool) (*testDbStore, error) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		return nil, err
	}

	var db *LDBStore
	if mock {
		globalStore := mem.NewGlobalStore()
		addr := common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
		mockStore := globalStore.NewNodeStore(addr)

		db, err = NewMockDbStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, testPoFunc, mockStore)
	} else {
		db, err = NewLDBStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, testPoFunc)
	}

	return &testDbStore{db, dir}, err
}

func testPoFunc(k Key) (ret uint8) {
	basekey := make([]byte, 32)
	return uint8(Proximity(basekey[:], k[:]))
}

func (db *testDbStore) close() {
	db.Close()
	err := os.RemoveAll(db.dir)
	if err != nil {
		panic(err)
	}
}

func testDbStoreRandom(n int, processors int, chunksize int, mock bool, t *testing.T) {
	db, err := newTestDbStore(mock)
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()
	db.trusted = true
	testStoreRandom(db, processors, n, chunksize, t)
}

func testDbStoreCorrect(n int, processors int, chunksize int, mock bool, t *testing.T) {
	db, err := newTestDbStore(mock)
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()
	testStoreCorrect(db, processors, n, chunksize, t)
}

func TestDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, 1, 0, false, t)
}

func TestDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, 1, 4096, false, t)
}

func TestDbStoreRandom_1_5k(t *testing.T) {
	testDbStoreRandom(8, 5000, 0, false, t)
}

func TestDbStoreRandom_8_5k(t *testing.T) {
	testDbStoreRandom(8, 5000, 0, false, t)
}

func TestDbStoreCorrect_1_5k(t *testing.T) {
	testDbStoreCorrect(1, 5000, 4096, false, t)
}

func TestDbStoreCorrect_8_5k(t *testing.T) {
	testDbStoreCorrect(8, 5000, 4096, false, t)
}

func TestMockDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, 1, 0, true, t)
}

func TestMockDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, 1, 4096, true, t)
}

func TestMockDbStoreRandom_1_5k(t *testing.T) {
	testDbStoreRandom(8, 5000, 0, true, t)
}

func TestMockDbStoreRandom_8_5k(t *testing.T) {
	testDbStoreRandom(8, 5000, 0, true, t)
}

func TestMockDbStoreCorrect_1_5k(t *testing.T) {
	testDbStoreCorrect(1, 5000, 4096, true, t)
}

func TestMockDbStoreCorrect_8_5k(t *testing.T) {
	testDbStoreCorrect(8, 5000, 4096, true, t)
}

func testDbStoreNotFound(t *testing.T, mock bool) {
	db, err := newTestDbStore(mock)
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()

	_, err = db.Get(ZeroKey)
	if err != ErrChunkNotFound {
		t.Errorf("Expected ErrChunkNotFound, got %v", err)
	}
}

func TestDbStoreNotFound(t *testing.T) {
	testDbStoreNotFound(t, false)
}
func TestMockDbStoreNotFound(t *testing.T) {
	testDbStoreNotFound(t, true)
}

func testIterator(t *testing.T, mock bool) {
	var chunkcount int = 32
	var i int
	var poc uint
	chunkkeys := NewKeyCollection(chunkcount)
	chunkkeys_results := NewKeyCollection(chunkcount)
	var chunks []*Chunk

	for i := 0; i < chunkcount; i++ {
		chunks = append(chunks, NewChunk(nil, nil))
	}

	db, err := newTestDbStore(mock)
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()

	FakeChunk(DefaultChunkSize, chunkcount, chunks)

	wg := &sync.WaitGroup{}
	wg.Add(len(chunks))
	for i = 0; i < len(chunks); i++ {
		db.Put(chunks[i])
		chunkkeys[i] = chunks[i].Key
		j := i
		go func() {
			defer wg.Done()
			<-chunks[j].dbStoredC
		}()
	}

	//testSplit(m, l, 128, chunkkeys, t)

	for i = 0; i < len(chunkkeys); i++ {
		log.Trace(fmt.Sprintf("Chunk array pos %d/%d: '%v'", i, chunkcount, chunkkeys[i]))
	}
	wg.Wait()
	i = 0
	for poc = 0; poc <= 255; poc++ {
		err := db.SyncIterator(0, uint64(chunkkeys.Len()), uint8(poc), func(k Key, n uint64) bool {
			log.Trace(fmt.Sprintf("Got key %v number %d poc %d", k, n, uint8(poc)))
			chunkkeys_results[n-1] = k
			i++
			return true
		})
		if err != nil {
			t.Fatalf("Iterator call failed: %v", err)
		}
	}

	for i = 0; i < chunkcount; i++ {
		if !bytes.Equal(chunkkeys[i], chunkkeys_results[i]) {
			t.Fatalf("Chunk put #%d key '%v' does not match iterator's key '%v'", i, chunkkeys[i], chunkkeys_results[i])
		}
	}

}

func TestIterator(t *testing.T) {
	testIterator(t, false)
}
func TestMockIterator(t *testing.T) {
	testIterator(t, true)
}

func benchmarkDbStorePut(n int, processors int, chunksize int, mock bool, b *testing.B) {
	db, err := newTestDbStore(mock)
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()
	db.trusted = true
	benchmarkStorePut(db, processors, n, chunksize, b)
}

func benchmarkDbStoreGet(n int, processors int, chunksize int, mock bool, b *testing.B) {
	db, err := newTestDbStore(mock)
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	defer db.close()
	db.trusted = true
	benchmarkStoreGet(db, processors, n, chunksize, b)
}

func BenchmarkDbStorePut_1_5k(b *testing.B) {
	benchmarkDbStorePut(5000, 1, 4096, false, b)
}

func BenchmarkDbStorePut_8_5k(b *testing.B) {
	benchmarkDbStorePut(5000, 8, 4096, false, b)
}

func BenchmarkDbStoreGet_1_5k(b *testing.B) {
	benchmarkDbStoreGet(5000, 1, 4096, false, b)
}

func BenchmarkDbStoreGet_8_5k(b *testing.B) {
	benchmarkDbStoreGet(5000, 8, 4096, false, b)
}

func BenchmarkMockDbStorePut_1_5k(b *testing.B) {
	benchmarkDbStorePut(5000, 1, 4096, true, b)
}

func BenchmarkMockDbStorePut_8_5k(b *testing.B) {
	benchmarkDbStorePut(5000, 8, 4096, true, b)
}

func BenchmarkMockDbStoreGet_1_5k(b *testing.B) {
	benchmarkDbStoreGet(5000, 1, 4096, true, b)
}

func BenchmarkMockDbStoreGet_8_5k(b *testing.B) {
	benchmarkDbStoreGet(5000, 8, 4096, true, b)
}
