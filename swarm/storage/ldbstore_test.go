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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"

	ldberrors "github.com/syndtr/goleveldb/leveldb/errors"
)

type testDbStore struct {
	*LDBStore
	dir string
}

func newTestDbStore(mock bool, trusted bool) (*testDbStore, func(), error) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		return nil, func() {}, err
	}

	var db *LDBStore
	storeparams := NewDefaultStoreParams()
	params := NewLDBStoreParams(storeparams, dir)
	params.Po = testPoFunc

	if mock {
		globalStore := mem.NewGlobalStore()
		addr := common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
		mockStore := globalStore.NewNodeStore(addr)

		db, err = NewMockDbStore(params, mockStore)
	} else {
		db, err = NewLDBStore(params)
	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
		err = os.RemoveAll(dir)
		if err != nil {
			panic(fmt.Sprintf("db cleanup failed: %v", err))
		}
	}

	return &testDbStore{db, dir}, cleanup, err
}

func testPoFunc(k Address) (ret uint8) {
	basekey := make([]byte, 32)
	return uint8(Proximity(basekey, k[:]))
}

func (db *testDbStore) close() {
	db.Close()
	err := os.RemoveAll(db.dir)
	if err != nil {
		panic(err)
	}
}

func testDbStoreRandom(n int, chunksize int64, mock bool, t *testing.T) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	testStoreRandom(db, n, chunksize, t)
}

func testDbStoreCorrect(n int, chunksize int64, mock bool, t *testing.T) {
	db, cleanup, err := newTestDbStore(mock, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	testStoreCorrect(db, n, chunksize, t)
}

func TestDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, 0, false, t)
}

func TestDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, 4096, false, t)
}

func TestDbStoreRandom_5k(t *testing.T) {
	testDbStoreRandom(5000, 0, false, t)
}

func TestDbStoreCorrect_5k(t *testing.T) {
	testDbStoreCorrect(5000, 4096, false, t)
}

func TestMockDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, 0, true, t)
}

func TestMockDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, 4096, true, t)
}

func TestMockDbStoreRandom_5k(t *testing.T) {
	testDbStoreRandom(5000, 0, true, t)
}

func TestMockDbStoreCorrect_5k(t *testing.T) {
	testDbStoreCorrect(5000, 4096, true, t)
}

func testDbStoreNotFound(t *testing.T, mock bool) {
	db, cleanup, err := newTestDbStore(mock, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}

	_, err = db.Get(context.TODO(), ZeroAddr)
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
	chunkkeys := NewAddressCollection(chunkcount)
	chunkkeys_results := NewAddressCollection(chunkcount)

	db, cleanup, err := newTestDbStore(mock, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}

	chunks := GenerateRandomChunks(ch.DefaultSize, chunkcount)

	for i = 0; i < len(chunks); i++ {
		chunkkeys[i] = chunks[i].Address()
		err := db.Put(context.TODO(), chunks[i])
		if err != nil {
			t.Fatalf("dbStore.Put failed: %v", err)
		}
	}

	for i = 0; i < len(chunkkeys); i++ {
		log.Trace(fmt.Sprintf("Chunk array pos %d/%d: '%v'", i, chunkcount, chunkkeys[i]))
	}
	i = 0
	for poc = 0; poc <= 255; poc++ {
		err := db.SyncIterator(0, uint64(chunkkeys.Len()), uint8(poc), func(k Address, n uint64) bool {
			log.Trace(fmt.Sprintf("Got key %v number %d poc %d", k, n, uint8(poc)))
			chunkkeys_results[n] = k
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

func benchmarkDbStorePut(n int, processors int, chunksize int64, mock bool, b *testing.B) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	benchmarkStorePut(db, n, chunksize, b)
}

func benchmarkDbStoreGet(n int, processors int, chunksize int64, mock bool, b *testing.B) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	benchmarkStoreGet(db, n, chunksize, b)
}

func BenchmarkDbStorePut_1_500(b *testing.B) {
	benchmarkDbStorePut(500, 1, 4096, false, b)
}

func BenchmarkDbStorePut_8_500(b *testing.B) {
	benchmarkDbStorePut(500, 8, 4096, false, b)
}

func BenchmarkDbStoreGet_1_500(b *testing.B) {
	benchmarkDbStoreGet(500, 1, 4096, false, b)
}

func BenchmarkDbStoreGet_8_500(b *testing.B) {
	benchmarkDbStoreGet(500, 8, 4096, false, b)
}

func BenchmarkMockDbStorePut_1_500(b *testing.B) {
	benchmarkDbStorePut(500, 1, 4096, true, b)
}

func BenchmarkMockDbStorePut_8_500(b *testing.B) {
	benchmarkDbStorePut(500, 8, 4096, true, b)
}

func BenchmarkMockDbStoreGet_1_500(b *testing.B) {
	benchmarkDbStoreGet(500, 1, 4096, true, b)
}

func BenchmarkMockDbStoreGet_8_500(b *testing.B) {
	benchmarkDbStoreGet(500, 8, 4096, true, b)
}

// TestLDBStoreWithoutCollectGarbage tests that we can put a number of random chunks in the LevelDB store, and
// retrieve them, provided we don't hit the garbage collection
func TestLDBStoreWithoutCollectGarbage(t *testing.T) {
	capacity := 50
	n := 10

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n, int64(ch.DefaultSize))
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	for _, ch := range chunks {
		ret, err := ldb.Get(context.TODO(), ch.Address())
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(ret.Data(), ch.Data()) {
			t.Fatal("expected to get the same data back, but got smth else")
		}
	}

	if ldb.entryCnt != uint64(n) {
		t.Fatalf("expected entryCnt to be equal to %v, but got %v", n, ldb.entryCnt)
	}

	if ldb.accessCnt != uint64(2*n) {
		t.Fatalf("expected accessCnt to be equal to %v, but got %v", 2*n, ldb.accessCnt)
	}
}

// TestLDBStoreCollectGarbage tests that we can put more chunks than LevelDB's capacity, and
// retrieve only some of them, because garbage collection must have cleared some of them
func TestLDBStoreCollectGarbage(t *testing.T) {
	capacity := 500
	n := 2000

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n, int64(ch.DefaultSize))
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	// wait for garbage collection to kick in on the responsible actor
	time.Sleep(1 * time.Second)

	var missing int
	for _, ch := range chunks {
		ret, err := ldb.Get(context.Background(), ch.Address())
		if err == ErrChunkNotFound || err == ldberrors.ErrNotFound {
			missing++
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(ret.Data(), ch.Data()) {
			t.Fatal("expected to get the same data back, but got smth else")
		}

		log.Trace("got back chunk", "chunk", ret)
	}

	if missing < n-capacity {
		t.Fatalf("gc failure: expected to miss %v chunks, but only %v are actually missing", n-capacity, missing)
	}

	log.Info("ldbstore", "total", n, "missing", missing, "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)
}

// TestLDBStoreAddRemove tests that we can put and then delete a given chunk
func TestLDBStoreAddRemove(t *testing.T) {
	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(200)
	defer cleanup()

	n := 100
	chunks, err := mputRandomChunks(ldb, n, int64(ch.DefaultSize))
	if err != nil {
		t.Fatalf(err.Error())
	}

	for i := 0; i < n; i++ {
		// delete all even index chunks
		if i%2 == 0 {
			ldb.Delete(chunks[i].Address())
		}
	}

	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	for i := 0; i < n; i++ {
		ret, err := ldb.Get(nil, chunks[i].Address())

		if i%2 == 0 {
			// expect even chunks to be missing
			if err == nil {
				// if err != ErrChunkNotFound {
				t.Fatal("expected chunk to be missing, but got no error")
			}
		} else {
			// expect odd chunks to be retrieved successfully
			if err != nil {
				t.Fatalf("expected no error, but got %s", err)
			}

			if !bytes.Equal(ret.Data(), chunks[i].Data()) {
				t.Fatal("expected to get the same data back, but got smth else")
			}
		}
	}
}

// TestLDBStoreRemoveThenCollectGarbage tests that we can delete chunks and that we can trigger garbage collection
func TestLDBStoreRemoveThenCollectGarbage(t *testing.T) {
	capacity := 11
	surplus := 4

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))

	n := capacity

	chunks := []Chunk{}
	for i := 0; i < n+surplus; i++ {
		c := GenerateRandomChunk(ch.DefaultSize)
		chunks = append(chunks, c)
		log.Trace("generate random chunk", "idx", i, "chunk", c)
	}

	for i := 0; i < n; i++ {
		ldb.Put(context.TODO(), chunks[i])
	}

	// delete all chunks
	for i := 0; i < n; i++ {
		ldb.Delete(chunks[i].Address())
	}

	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	if ldb.entryCnt != 0 {
		t.Fatalf("ldb.entrCnt expected 0 got %v", ldb.entryCnt)
	}

	expAccessCnt := uint64(n * 2)
	if ldb.accessCnt != expAccessCnt {
		t.Fatalf("ldb.accessCnt expected %v got %v", expAccessCnt, ldb.entryCnt)
	}

	cleanup()

	ldb, cleanup = newLDBStore(t)
	capacity = 10
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	n = capacity + surplus

	for i := 0; i < n; i++ {
		ldb.Put(context.TODO(), chunks[i])
	}

	// wait for garbage collection
	time.Sleep(1 * time.Second)

	// expect first surplus chunks to be missing, because they have the smallest access value
	for i := 0; i < surplus; i++ {
		_, err := ldb.Get(context.TODO(), chunks[i].Address())
		if err == nil {
			t.Fatal("expected surplus chunk to be missing, but got no error")
		}
	}

	// expect last chunks to be present, as they have the largest access value
	for i := surplus; i < surplus+capacity; i++ {
		ret, err := ldb.Get(context.TODO(), chunks[i].Address())
		if err != nil {
			t.Fatalf("chunk %v: expected no error, but got %s", i, err)
		}
		if !bytes.Equal(ret.Data(), chunks[i].Data()) {
			t.Fatal("expected to get the same data back, but got smth else")
		}
	}
}
