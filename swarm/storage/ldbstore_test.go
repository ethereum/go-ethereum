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
	"strconv"
	"strings"
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

func TestDbStoreRandom_1k(t *testing.T) {
	testDbStoreRandom(1000, 0, false, t)
}

func TestDbStoreCorrect_1k(t *testing.T) {
	testDbStoreCorrect(1000, 4096, false, t)
}

func TestMockDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, 0, true, t)
}

func TestMockDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, 4096, true, t)
}

func TestMockDbStoreRandom_1k(t *testing.T) {
	testDbStoreRandom(1000, 0, true, t)
}

func TestMockDbStoreCorrect_1k(t *testing.T) {
	testDbStoreCorrect(1000, 4096, true, t)
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
// retrieve only some of them, because garbage collection must have partially cleared the store
// Also tests that we can delete chunks and that we can trigger garbage collection
func TestLDBStoreCollectGarbage(t *testing.T) {

	// below max ronud
	cap := defaultMaxGCRound / 2
	t.Run(fmt.Sprintf("A/%d/%d", cap, cap*4), testLDBStoreCollectGarbage)
	t.Run(fmt.Sprintf("B/%d/%d", cap, cap*4), testLDBStoreRemoveThenCollectGarbage)

	// at max round
	cap = defaultMaxGCRound
	t.Run(fmt.Sprintf("A/%d/%d", cap, cap*4), testLDBStoreCollectGarbage)
	t.Run(fmt.Sprintf("B/%d/%d", cap, cap*4), testLDBStoreRemoveThenCollectGarbage)

	// more than max around, not on threshold
	cap = defaultMaxGCRound * 1.1
	t.Run(fmt.Sprintf("A/%d/%d", cap, cap*4), testLDBStoreCollectGarbage)
	t.Run(fmt.Sprintf("B/%d/%d", cap, cap*4), testLDBStoreRemoveThenCollectGarbage)

}

func testLDBStoreCollectGarbage(t *testing.T) {
	params := strings.Split(t.Name(), "/")
	capacity, err := strconv.Atoi(params[2])
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(params[3])
	if err != nil {
		t.Fatal(err)
	}

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	// retrieve the gc round target count for the db capacity
	ldb.startGC(capacity)
	roundTarget := ldb.gc.target

	// split put counts to gc target count threshold, and wait for gc to finish in between
	var allChunks []Chunk
	remaining := n
	for remaining > 0 {
		var putCount int
		if remaining < roundTarget {
			putCount = remaining
		} else {
			putCount = roundTarget
		}
		remaining -= putCount
		chunks, err := mputRandomChunks(ldb, putCount, int64(ch.DefaultSize))
		if err != nil {
			t.Fatal(err.Error())
		}
		allChunks = append(allChunks, chunks...)
		log.Debug("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt, "cap", capacity, "n", n)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		waitGc(ctx, ldb)
	}

	// attempt gets on all put chunks
	var missing int
	for _, ch := range allChunks {
		ret, err := ldb.Get(context.TODO(), ch.Address())
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

	// all surplus chunks should be missing
	expectMissing := roundTarget + (((n - capacity) / roundTarget) * roundTarget)
	if missing != expectMissing {
		t.Fatalf("gc failure: expected to miss %v chunks, but only %v are actually missing", expectMissing, missing)
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

func testLDBStoreRemoveThenCollectGarbage(t *testing.T) {

	params := strings.Split(t.Name(), "/")
	capacity, err := strconv.Atoi(params[2])
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(params[3])
	if err != nil {
		t.Fatal(err)
	}

	ldb, cleanup := newLDBStore(t)
	defer cleanup()
	ldb.setCapacity(uint64(capacity))

	// put capacity count number of chunks
	chunks := make([]Chunk, n)
	for i := 0; i < n; i++ {
		c := GenerateRandomChunk(ch.DefaultSize)
		chunks[i] = c
		log.Trace("generate random chunk", "idx", i, "chunk", c)
	}

	for i := 0; i < n; i++ {
		err := ldb.Put(context.TODO(), chunks[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	waitGc(ctx, ldb)

	// delete all chunks
	// (only count the ones actually deleted, the rest will have been gc'd)
	deletes := 0
	for i := 0; i < n; i++ {
		if ldb.Delete(chunks[i].Address()) == nil {
			deletes++
		}
	}

	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	if ldb.entryCnt != 0 {
		t.Fatalf("ldb.entrCnt expected 0 got %v", ldb.entryCnt)
	}

	// the manual deletes will have increased accesscnt, so we need to add this when we verify the current count
	expAccessCnt := uint64(n)
	if ldb.accessCnt != expAccessCnt {
		t.Fatalf("ldb.accessCnt expected %v got %v", expAccessCnt, ldb.accessCnt)
	}

	// retrieve the gc round target count for the db capacity
	ldb.startGC(capacity)
	roundTarget := ldb.gc.target

	remaining := n
	var puts int
	for remaining > 0 {
		var putCount int
		if remaining < roundTarget {
			putCount = remaining
		} else {
			putCount = roundTarget
		}
		remaining -= putCount
		for putCount > 0 {
			ldb.Put(context.TODO(), chunks[puts])
			log.Debug("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt, "cap", capacity, "n", n, "puts", puts, "remaining", remaining, "roundtarget", roundTarget)
			puts++
			putCount--
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		waitGc(ctx, ldb)
	}

	// expect first surplus chunks to be missing, because they have the smallest access value
	expectMissing := roundTarget + (((n - capacity) / roundTarget) * roundTarget)
	for i := 0; i < expectMissing; i++ {
		_, err := ldb.Get(context.TODO(), chunks[i].Address())
		if err == nil {
			t.Fatalf("expected surplus chunk %d to be missing, but got no error", i)
		}
	}

	// expect last chunks to be present, as they have the largest access value
	for i := expectMissing; i < n; i++ {
		ret, err := ldb.Get(context.TODO(), chunks[i].Address())
		if err != nil {
			t.Fatalf("chunk %v: expected no error, but got %s", i, err)
		}
		if !bytes.Equal(ret.Data(), chunks[i].Data()) {
			t.Fatal("expected to get the same data back, but got smth else")
		}
	}
}

// TestLDBStoreCollectGarbageAccessUnlikeIndex tests garbage collection where accesscount differs from indexcount
func TestLDBStoreCollectGarbageAccessUnlikeIndex(t *testing.T) {

	capacity := defaultMaxGCRound * 2
	n := capacity - 1

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n, int64(ch.DefaultSize))
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Info("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)

	// set first added capacity/2 chunks to highest accesscount
	for i := 0; i < capacity/2; i++ {
		_, err := ldb.Get(context.TODO(), chunks[i].Address())
		if err != nil {
			t.Fatalf("fail add chunk #%d - %s: %v", i, chunks[i].Address(), err)
		}
	}
	_, err = mputRandomChunks(ldb, 2, int64(ch.DefaultSize))
	if err != nil {
		t.Fatal(err.Error())
	}

	// wait for garbage collection to kick in on the responsible actor
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	waitGc(ctx, ldb)

	var missing int
	for i, ch := range chunks[2 : capacity/2] {
		ret, err := ldb.Get(context.TODO(), ch.Address())
		if err == ErrChunkNotFound || err == ldberrors.ErrNotFound {
			t.Fatalf("fail find chunk #%d - %s: %v", i, ch.Address(), err)
		}

		if !bytes.Equal(ret.Data(), ch.Data()) {
			t.Fatal("expected to get the same data back, but got smth else")
		}
		log.Trace("got back chunk", "chunk", ret)
	}

	log.Info("ldbstore", "total", n, "missing", missing, "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt)
}

func waitGc(ctx context.Context, ldb *LDBStore) {
	<-ldb.gc.runC
	ldb.gc.runC <- struct{}{}
}
