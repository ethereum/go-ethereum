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
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/testutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/chunk"
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

func testDbStoreRandom(n int, mock bool, t *testing.T) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	testStoreRandom(db, n, t)
}

func testDbStoreCorrect(n int, mock bool, t *testing.T) {
	db, cleanup, err := newTestDbStore(mock, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}
	testStoreCorrect(db, n, t)
}

func TestMarkAccessed(t *testing.T) {
	db, cleanup, err := newTestDbStore(false, true)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}

	h := GenerateRandomChunk(chunk.DefaultSize)

	db.Put(context.Background(), h)

	var index dpaDBIndex
	addr := h.Address()
	idxk := getIndexKey(addr)

	idata, err := db.db.Get(idxk)
	if err != nil {
		t.Fatal(err)
	}
	decodeIndex(idata, &index)

	if index.Access != 0 {
		t.Fatalf("Expected the access index to be %d, but it is %d", 0, index.Access)
	}

	db.MarkAccessed(addr)
	db.writeCurrentBatch()

	idata, err = db.db.Get(idxk)
	if err != nil {
		t.Fatal(err)
	}
	decodeIndex(idata, &index)

	if index.Access != 1 {
		t.Fatalf("Expected the access index to be %d, but it is %d", 1, index.Access)
	}

}

func TestDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, false, t)
}

func TestDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, false, t)
}

func TestDbStoreRandom_1k(t *testing.T) {
	testDbStoreRandom(1000, false, t)
}

func TestDbStoreCorrect_1k(t *testing.T) {
	testDbStoreCorrect(1000, false, t)
}

func TestMockDbStoreRandom_1(t *testing.T) {
	testDbStoreRandom(1, true, t)
}

func TestMockDbStoreCorrect_1(t *testing.T) {
	testDbStoreCorrect(1, true, t)
}

func TestMockDbStoreRandom_1k(t *testing.T) {
	testDbStoreRandom(1000, true, t)
}

func TestMockDbStoreCorrect_1k(t *testing.T) {
	testDbStoreCorrect(1000, true, t)
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
	var i int
	var poc uint
	chunkcount := 32
	chunkkeys := NewAddressCollection(chunkcount)
	chunkkeysResults := NewAddressCollection(chunkcount)

	db, cleanup, err := newTestDbStore(mock, false)
	defer cleanup()
	if err != nil {
		t.Fatalf("init dbStore failed: %v", err)
	}

	chunks := GenerateRandomChunks(chunk.DefaultSize, chunkcount)

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
			chunkkeysResults[n] = k
			i++
			return true
		})
		if err != nil {
			t.Fatalf("Iterator call failed: %v", err)
		}
	}

	for i = 0; i < chunkcount; i++ {
		if !bytes.Equal(chunkkeys[i], chunkkeysResults[i]) {
			t.Fatalf("Chunk put #%d key '%v' does not match iterator's key '%v'", i, chunkkeys[i], chunkkeysResults[i])
		}
	}

}

func TestIterator(t *testing.T) {
	testIterator(t, false)
}
func TestMockIterator(t *testing.T) {
	testIterator(t, true)
}

func benchmarkDbStorePut(n int, mock bool, b *testing.B) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	benchmarkStorePut(db, n, b)
}

func benchmarkDbStoreGet(n int, mock bool, b *testing.B) {
	db, cleanup, err := newTestDbStore(mock, true)
	defer cleanup()
	if err != nil {
		b.Fatalf("init dbStore failed: %v", err)
	}
	benchmarkStoreGet(db, n, b)
}

func BenchmarkDbStorePut_500(b *testing.B) {
	benchmarkDbStorePut(500, false, b)
}

func BenchmarkDbStoreGet_500(b *testing.B) {
	benchmarkDbStoreGet(500, false, b)
}

func BenchmarkMockDbStorePut_500(b *testing.B) {
	benchmarkDbStorePut(500, true, b)
}

func BenchmarkMockDbStoreGet_500(b *testing.B) {
	benchmarkDbStoreGet(500, true, b)
}

// TestLDBStoreWithoutCollectGarbage tests that we can put a number of random chunks in the LevelDB store, and
// retrieve them, provided we don't hit the garbage collection
func TestLDBStoreWithoutCollectGarbage(t *testing.T) {
	capacity := 50
	n := 10

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n)
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
	initialCap := defaultMaxGCRound / 100
	cap := initialCap / 2
	t.Run(fmt.Sprintf("A/%d/%d", cap, cap*4), testLDBStoreCollectGarbage)

	if testutil.RaceEnabled {
		t.Skip("only the simplest case run as others are flaky with race")
		// Note: some tests fail consistently and even locally with `-race`
	}

	t.Run(fmt.Sprintf("B/%d/%d", cap, cap*4), testLDBStoreRemoveThenCollectGarbage)

	// at max round
	cap = initialCap
	t.Run(fmt.Sprintf("A/%d/%d", cap, cap*4), testLDBStoreCollectGarbage)
	t.Run(fmt.Sprintf("B/%d/%d", cap, cap*4), testLDBStoreRemoveThenCollectGarbage)

	// more than max around, not on threshold
	cap = initialCap + 500
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
		chunks, err := mputRandomChunks(ldb, putCount)
		if err != nil {
			t.Fatal(err.Error())
		}
		allChunks = append(allChunks, chunks...)
		ldb.lock.RLock()
		log.Debug("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt, "cap", capacity, "n", n)
		ldb.lock.RUnlock()

		waitGc(ldb)
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
	chunks, err := mputRandomChunks(ldb, n)
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
		ret, err := ldb.Get(context.TODO(), chunks[i].Address())

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
	t.Skip("flaky with -race flag")

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
		c := GenerateRandomChunk(chunk.DefaultSize)
		chunks[i] = c
		log.Trace("generate random chunk", "idx", i, "chunk", c)
	}

	for i := 0; i < n; i++ {
		err := ldb.Put(context.TODO(), chunks[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	waitGc(ldb)

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
			ldb.lock.RLock()
			log.Debug("ldbstore", "entrycnt", ldb.entryCnt, "accesscnt", ldb.accessCnt, "cap", capacity, "n", n, "puts", puts, "remaining", remaining, "roundtarget", roundTarget)
			ldb.lock.RUnlock()
			puts++
			putCount--
		}

		waitGc(ldb)
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

	capacity := defaultMaxGCRound / 100 * 2
	n := capacity - 1

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n)
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
	_, err = mputRandomChunks(ldb, 2)
	if err != nil {
		t.Fatal(err.Error())
	}

	// wait for garbage collection to kick in on the responsible actor
	waitGc(ldb)

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

func TestCleanIndex(t *testing.T) {
	capacity := 5000
	n := 3

	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(uint64(capacity))
	defer cleanup()

	chunks, err := mputRandomChunks(ldb, n)
	if err != nil {
		t.Fatal(err)
	}

	// remove the data of the first chunk
	po := ldb.po(chunks[0].Address()[:])
	dataKey := make([]byte, 10)
	dataKey[0] = keyData
	dataKey[1] = byte(po)
	// dataKey[2:10] = first chunk has storageIdx 0 on [2:10]
	if _, err := ldb.db.Get(dataKey); err != nil {
		t.Fatal(err)
	}
	if err := ldb.db.Delete(dataKey); err != nil {
		t.Fatal(err)
	}

	// remove the gc index row for the first chunk
	gcFirstCorrectKey := make([]byte, 9)
	gcFirstCorrectKey[0] = keyGCIdx
	if err := ldb.db.Delete(gcFirstCorrectKey); err != nil {
		t.Fatal(err)
	}

	// warp the gc data of the second chunk
	// this data should be correct again after the clean
	gcSecondCorrectKey := make([]byte, 9)
	gcSecondCorrectKey[0] = keyGCIdx
	binary.BigEndian.PutUint64(gcSecondCorrectKey[1:], uint64(1))
	gcSecondCorrectVal, err := ldb.db.Get(gcSecondCorrectKey)
	if err != nil {
		t.Fatal(err)
	}
	warpedGCVal := make([]byte, len(gcSecondCorrectVal)+1)
	copy(warpedGCVal[1:], gcSecondCorrectVal)
	if err := ldb.db.Delete(gcSecondCorrectKey); err != nil {
		t.Fatal(err)
	}
	if err := ldb.db.Put(gcSecondCorrectKey, warpedGCVal); err != nil {
		t.Fatal(err)
	}

	if err := ldb.CleanGCIndex(); err != nil {
		t.Fatal(err)
	}

	// the index without corresponding data should have been deleted
	idxKey := make([]byte, 33)
	idxKey[0] = keyIndex
	copy(idxKey[1:], chunks[0].Address())
	if _, err := ldb.db.Get(idxKey); err == nil {
		t.Fatalf("expected chunk 0 idx to be pruned: %v", idxKey)
	}

	// the two other indices should be present
	copy(idxKey[1:], chunks[1].Address())
	if _, err := ldb.db.Get(idxKey); err != nil {
		t.Fatalf("expected chunk 1 idx to be present: %v", idxKey)
	}

	copy(idxKey[1:], chunks[2].Address())
	if _, err := ldb.db.Get(idxKey); err != nil {
		t.Fatalf("expected chunk 2 idx to be present: %v", idxKey)
	}

	// first gc index should still be gone
	if _, err := ldb.db.Get(gcFirstCorrectKey); err == nil {
		t.Fatalf("expected gc 0 idx to be pruned: %v", idxKey)
	}

	// second gc index should still be fixed
	if _, err := ldb.db.Get(gcSecondCorrectKey); err != nil {
		t.Fatalf("expected gc 1 idx to be present: %v", idxKey)
	}

	// third gc index should be unchanged
	binary.BigEndian.PutUint64(gcSecondCorrectKey[1:], uint64(2))
	if _, err := ldb.db.Get(gcSecondCorrectKey); err != nil {
		t.Fatalf("expected gc 2 idx to be present: %v", idxKey)
	}

	c, err := ldb.db.Get(keyEntryCnt)
	if err != nil {
		t.Fatalf("expected gc 2 idx to be present: %v", idxKey)
	}

	// entrycount should now be one less
	entryCount := binary.BigEndian.Uint64(c)
	if entryCount != 2 {
		t.Fatalf("expected entrycnt to be 2, was %d", c)
	}

	// the chunks might accidentally be in the same bin
	// if so that bin counter will now be 2 - the highest added index.
	// if not, the total of them will be 3
	poBins := []uint8{ldb.po(chunks[1].Address()), ldb.po(chunks[2].Address())}
	if poBins[0] == poBins[1] {
		poBins = poBins[:1]
	}

	var binTotal uint64
	var currentBin [2]byte
	currentBin[0] = keyDistanceCnt
	if len(poBins) == 1 {
		currentBin[1] = poBins[0]
		c, err := ldb.db.Get(currentBin[:])
		if err != nil {
			t.Fatalf("expected gc 2 idx to be present: %v", idxKey)
		}
		binCount := binary.BigEndian.Uint64(c)
		if binCount != 2 {
			t.Fatalf("expected entrycnt to be 2, was %d", binCount)
		}
	} else {
		for _, bin := range poBins {
			currentBin[1] = bin
			c, err := ldb.db.Get(currentBin[:])
			if err != nil {
				t.Fatalf("expected gc 2 idx to be present: %v", idxKey)
			}
			binCount := binary.BigEndian.Uint64(c)
			binTotal += binCount

		}
		if binTotal != 3 {
			t.Fatalf("expected sum of bin indices to be 3, was %d", binTotal)
		}
	}

	// check that the iterator quits properly
	chunks, err = mputRandomChunks(ldb, 4100)
	if err != nil {
		t.Fatal(err)
	}

	po = ldb.po(chunks[4099].Address()[:])
	dataKey = make([]byte, 10)
	dataKey[0] = keyData
	dataKey[1] = byte(po)
	binary.BigEndian.PutUint64(dataKey[2:], 4099+3)
	if _, err := ldb.db.Get(dataKey); err != nil {
		t.Fatal(err)
	}
	if err := ldb.db.Delete(dataKey); err != nil {
		t.Fatal(err)
	}

	if err := ldb.CleanGCIndex(); err != nil {
		t.Fatal(err)
	}

	// entrycount should now be one less of added chunks
	c, err = ldb.db.Get(keyEntryCnt)
	if err != nil {
		t.Fatalf("expected gc 2 idx to be present: %v", idxKey)
	}
	entryCount = binary.BigEndian.Uint64(c)
	if entryCount != 4099+2 {
		t.Fatalf("expected entrycnt to be 2, was %d", c)
	}
}

// Note: waitGc does not guarantee that we wait 1 GC round; it only
// guarantees that if the GC is running we wait for that run to finish
// ticket: https://github.com/ethersphere/go-ethereum/issues/1151
func waitGc(ldb *LDBStore) {
	<-ldb.gc.runC
	ldb.gc.runC <- struct{}{}
}
