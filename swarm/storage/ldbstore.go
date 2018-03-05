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

// disk storage layer for the package bzz
// DbStore implements the ChunkStore interface and is used by the DPA as
// persistent storage of chunks
// it implements purging based on access count allowing for external control of
// max capacity

package storage

import (
	"archive/tar"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

//metrics variables
var (
	gcCounter            = metrics.NewRegisteredCounter("storage.db.dbstore.gc.count", nil)
	dbStoreDeleteCounter = metrics.NewRegisteredCounter("storage.db.dbstore.rm.count", nil)
)

const (
	defaultDbCapacity = 5000000
	defaultRadius     = 0 // not yet used

	gcArraySize      = 10000
	gcArrayFreeRatio = 0.1

	// key prefixes for leveldb storage
	kpIndex = 0
	kpData  = 1
)

var (
	keyOldData     = byte(1)
	keyAccessCnt   = []byte{2}
	keyEntryCnt    = []byte{3}
	keyDataIdx     = []byte{4}
	keyGCPos       = []byte{5}
	keyData        = byte(6)
	keyDistanceCnt = byte(7)
)

type gcItem struct {
	idx    uint64
	value  uint64
	idxKey []byte
}

type LDBStore struct {
	db *LDBDatabase

	// this should be stored in db, accessed transactionally
	entryCnt, accessCnt, dataIdx, capacity uint64
	bucketCnt                              []uint64

	gcPos, gcStartPos []byte
	gcArray           []*gcItem

	hashfunc SwarmHasher
	po       func(Key) uint8

	batchC   chan bool
	batchesC chan struct{}
	batch    *leveldb.Batch
	lock     sync.RWMutex
	trusted  bool // if hash integity check is to be performed (for testing only)

	// Functions encodeDataFunc is used to bypass
	// the default functionality of DbStore with
	// mock.NodeStore for testing purposes.
	encodeDataFunc func(chunk *Chunk) []byte
	// If getDataFunc is defined, it will be used for
	// retrieving the chunk data instead from the local
	// LevelDB database.
	getDataFunc func(key Key) (data []byte, err error)
}

// TODO: Instead of passing the distance function, just pass the address from which distances are calculated
// to avoid the appearance of a pluggable distance metric and opportunities of bugs associated with providing
// a function different from the one that is actually used.
func NewLDBStore(path string, hash SwarmHasher, capacity uint64, po func(Key) uint8) (s *LDBStore, err error) {
	s = new(LDBStore)
	s.hashfunc = hash

	s.batchC = make(chan bool)
	s.batchesC = make(chan struct{}, 1)
	go s.writeBatches()
	s.batch = new(leveldb.Batch)
	// associate encodeData with default functionality
	s.encodeDataFunc = encodeData

	s.db, err = NewLDBDatabase(path)
	if err != nil {
		return nil, err
	}

	s.po = po
	s.setCapacity(capacity)

	s.gcStartPos = make([]byte, 1)
	s.gcStartPos[0] = kpIndex
	s.gcArray = make([]*gcItem, gcArraySize)

	s.bucketCnt = make([]uint64, 0x100)
	for i := 0; i < 0x100; i++ {
		k := make([]byte, 2)
		k[0] = keyDistanceCnt
		k[1] = uint8(i)
		cnt, _ := s.db.Get(k)
		s.bucketCnt[i] = BytesToU64(cnt)
		s.bucketCnt[i]++
	}
	data, _ := s.db.Get(keyEntryCnt)
	s.entryCnt = BytesToU64(data)
	s.entryCnt++
	data, _ = s.db.Get(keyAccessCnt)
	s.accessCnt = BytesToU64(data)
	s.accessCnt++
	data, _ = s.db.Get(keyDataIdx)
	s.dataIdx = BytesToU64(data)
	s.dataIdx++

	s.gcPos, _ = s.db.Get(keyGCPos)
	if s.gcPos == nil {
		s.gcPos = s.gcStartPos
	}
	return s, nil
}

// NewMockDbStore creates a new instance of DbStore with
// mockStore set to a provided value. If mockStore argument is nil,
// this function behaves exactly as NewDbStore.
func NewMockDbStore(path string, hash SwarmHasher, capacity uint64, po func(Key) uint8, mockStore *mock.NodeStore) (s *LDBStore, err error) {
	s, err = NewLDBStore(path, hash, capacity, po)
	if err != nil {
		return nil, err
	}
	// replace put and get with mock store functionality
	if mockStore != nil {
		s.encodeDataFunc = newMockEncodeDataFunc(mockStore)
		s.getDataFunc = newMockGetDataFunc(mockStore)
	}
	return
}

type dpaDBIndex struct {
	Idx    uint64
	Access uint64
}

func BytesToU64(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	//return binary.LittleEndian.Uint64(data)
	return binary.BigEndian.Uint64(data)
}

func U64ToBytes(val uint64) []byte {
	data := make([]byte, 8)
	//binary.LittleEndian.PutUint64(data, val)
	binary.BigEndian.PutUint64(data, val)
	return data
}

func getIndexGCValue(index *dpaDBIndex) uint64 {
	return index.Access
}

func (s *LDBStore) updateIndexAccess(index *dpaDBIndex) {
	index.Access = s.accessCnt
}

func getIndexKey(hash Key) []byte {
	hashSize := len(hash)
	key := make([]byte, hashSize+1)
	key[0] = 0
	copy(key[1:], hash[:])
	return key
}

func getOldDataKey(idx uint64) []byte {
	key := make([]byte, 9)
	key[0] = keyOldData
	binary.BigEndian.PutUint64(key[1:9], idx)

	return key
}

func getDataKey(idx uint64, po uint8) []byte {
	key := make([]byte, 10)
	key[0] = keyData
	key[1] = po
	binary.BigEndian.PutUint64(key[2:], idx)

	return key
}

func encodeIndex(index *dpaDBIndex) []byte {
	data, _ := rlp.EncodeToBytes(index)
	return data
}

func encodeData(chunk *Chunk) []byte {
	return append(chunk.Key[:], chunk.SData...)
}

func decodeIndex(data []byte, index *dpaDBIndex) error {
	dec := rlp.NewStream(bytes.NewReader(data), 0)
	return dec.Decode(index)

}

func decodeData(data []byte, chunk *Chunk) {
	chunk.SData = data[32:]
	chunk.Size = int64(binary.BigEndian.Uint64(data[32:40]))
}

func decodeOldData(data []byte, chunk *Chunk) {
	chunk.SData = data
	chunk.Size = int64(binary.BigEndian.Uint64(data[0:8]))
}

func gcListPartition(list []*gcItem, left int, right int, pivotIndex int) int {
	pivotValue := list[pivotIndex].value
	dd := list[pivotIndex]
	list[pivotIndex] = list[right]
	list[right] = dd
	storeIndex := left
	for i := left; i < right; i++ {
		if list[i].value < pivotValue {
			dd = list[storeIndex]
			list[storeIndex] = list[i]
			list[i] = dd
			storeIndex++
		}
	}
	dd = list[storeIndex]
	list[storeIndex] = list[right]
	list[right] = dd
	return storeIndex
}

func gcListSelect(list []*gcItem, left int, right int, n int) int {
	if left == right {
		return left
	}
	pivotIndex := (left + right) / 2
	pivotIndex = gcListPartition(list, left, right, pivotIndex)
	if n == pivotIndex {
		return n
	} else {
		if n < pivotIndex {
			return gcListSelect(list, left, pivotIndex-1, n)
		} else {
			return gcListSelect(list, pivotIndex+1, right, n)
		}
	}
}

func (s *LDBStore) collectGarbage(ratio float32) {
	it := s.db.NewIterator()
	it.Seek(s.gcPos)
	if it.Valid() {
		s.gcPos = it.Key()
	} else {
		s.gcPos = nil
	}
	gcnt := 0

	for (gcnt < gcArraySize) && (uint64(gcnt) < s.entryCnt) {

		if (s.gcPos == nil) || (s.gcPos[0] != kpIndex) {
			it.Seek(s.gcStartPos)
			if it.Valid() {
				s.gcPos = it.Key()
			} else {
				s.gcPos = nil
			}
		}

		if (s.gcPos == nil) || (s.gcPos[0] != kpIndex) {
			break
		}

		gci := new(gcItem)
		gci.idxKey = s.gcPos
		var index dpaDBIndex
		decodeIndex(it.Value(), &index)
		gci.idx = index.Idx
		// the smaller, the more likely to be gc'd
		gci.value = getIndexGCValue(&index)
		s.gcArray[gcnt] = gci
		gcnt++
		it.Next()
		if it.Valid() {
			s.gcPos = it.Key()
		} else {
			s.gcPos = nil
		}
	}
	it.Release()

	cutidx := gcListSelect(s.gcArray, 0, gcnt-1, int(float32(gcnt)*ratio))
	cutval := s.gcArray[cutidx].value

	// actual gc
	for i := 0; i < gcnt; i++ {
		if s.gcArray[i].value <= cutval {
			gcCounter.Inc(1)
			s.delete(s.gcArray[i].idx, s.gcArray[i].idxKey, s.po(Key(s.gcPos[1:])))
		}
	}

	s.db.Put(keyGCPos, s.gcPos)
}

// Export writes all chunks from the store to a tar archive, returning the
// number of chunks written.
func (s *LDBStore) Export(out io.Writer) (int64, error) {
	tw := tar.NewWriter(out)
	defer tw.Close()

	it := s.db.NewIterator()
	defer it.Release()
	var count int64
	for ok := it.Seek([]byte{kpIndex}); ok; ok = it.Next() {
		key := it.Key()
		if (key == nil) || (key[0] != kpIndex) {
			break
		}

		var index dpaDBIndex
		decodeIndex(it.Value(), &index)

		hash := key[1:]

		data, err := s.db.Get(getDataKey(index.Idx, s.po(hash)))
		if err != nil {
			log.Warn(fmt.Sprintf("Chunk %x found but could not be accessed: %v", key[:], err))
			continue
		}

		hdr := &tar.Header{
			Name: hex.EncodeToString(hash),
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return count, err
		}
		if _, err := tw.Write(data); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// of chunks read.
func (s *LDBStore) Import(in io.Reader) (int64, error) {
	tr := tar.NewReader(in)

	var count int64
	var wg sync.WaitGroup
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return count, err
		}

		if len(hdr.Name) != 64 {
			log.Warn("ignoring non-chunk file", "name", hdr.Name)
			continue
		}

		key, err := hex.DecodeString(hdr.Name)
		if err != nil {
			log.Warn("ignoring invalid chunk file", "name", hdr.Name, "err", err)
			continue
		}

		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return count, err
		}
		chunk := NewChunk(key, nil)
		chunk.SData = data
		s.Put(chunk)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-chunk.dbStored
		}()
		count++
	}
	wg.Wait()
	return count, nil
}

func (s *LDBStore) Cleanup() {
	//Iterates over the database and checks that there are no faulty chunks
	it := s.db.NewIterator()
	startPosition := []byte{kpIndex}
	it.Seek(startPosition)
	var key []byte
	var errorsFound, total int
	for it.Valid() {
		key = it.Key()
		if (key == nil) || (key[0] != kpIndex) {
			break
		}
		total++
		var index dpaDBIndex
		err := decodeIndex(it.Value(), &index)
		if err != nil {
			it.Next()
			continue
		}
		data, err := s.db.Get(getDataKey(index.Idx, s.po(Key(key[1:]))))
		if err != nil {
			log.Warn(fmt.Sprintf("Chunk %x found but could not be accessed: %v", key[:], err))
			s.delete(index.Idx, getIndexKey(key[1:]), s.po(Key(key[1:])))
			errorsFound++
		} else {
			hasher := s.hashfunc()
			hasher.Write(data[32:])
			hash := hasher.Sum(nil)
			if !bytes.Equal(hash, key[1:]) {
				log.Warn(fmt.Sprintf("Found invalid chunk. Hash mismatch. hash=%x, key=%x", hash, key[:]))
				s.delete(index.Idx, getIndexKey(key[1:]), s.po(Key(key[1:])))
			}
		}
		it.Next()
	}
	it.Release()
	log.Warn(fmt.Sprintf("Found %v errors out of %v entries", errorsFound, total))
}

func (s *LDBStore) ReIndex() {
	//Iterates over the database and checks that there are no faulty chunks
	it := s.db.NewIterator()
	startPosition := []byte{keyOldData}
	it.Seek(startPosition)
	var key []byte
	var errorsFound, total int
	for it.Valid() {
		key = it.Key()
		if (key == nil) || (key[0] != keyOldData) {
			break
		}
		data := it.Value()
		hasher := s.hashfunc()
		hasher.Write(data)
		hash := hasher.Sum(nil)

		newKey := make([]byte, 10)
		oldCntKey := make([]byte, 2)
		newCntKey := make([]byte, 2)
		oldCntKey[0] = keyDistanceCnt
		newCntKey[0] = keyDistanceCnt
		key[0] = keyData
		key[1] = s.po(Key(key[1:]))
		oldCntKey[1] = key[1]
		newCntKey[1] = s.po(Key(newKey[1:]))
		copy(newKey[2:], key[1:])
		newValue := append(hash, data...)

		batch := new(leveldb.Batch)
		batch.Delete(key)
		s.bucketCnt[oldCntKey[1]]--
		batch.Put(oldCntKey, U64ToBytes(s.bucketCnt[oldCntKey[1]]))
		batch.Put(newKey, newValue)
		s.bucketCnt[newCntKey[1]]++
		batch.Put(newCntKey, U64ToBytes(s.bucketCnt[newCntKey[1]]))
		s.db.Write(batch)
		it.Next()
	}
	it.Release()
	log.Warn(fmt.Sprintf("Found %v errors out of %v entries", errorsFound, total))
}

func (s *LDBStore) delete(idx uint64, idxKey []byte, po uint8) {
	batch := new(leveldb.Batch)
	batch.Delete(idxKey)
	batch.Delete(getDataKey(idx, po))
	dbStoreDeleteCounter.Inc(1)
	s.entryCnt--
	s.bucketCnt[po]--
	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))
	s.db.Write(batch)
}

func (s *LDBStore) CurrentBucketStorageIndex(po uint8) uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.bucketCnt[po]
}

func (s *LDBStore) Size() uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.entryCnt
}

func (s *LDBStore) CurrentStorageIndex() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.dataIdx
}

func (s *LDBStore) Put(chunk *Chunk) {
	ikey := getIndexKey(chunk.Key)
	var index dpaDBIndex

	po := s.po(chunk.Key)
	s.lock.Lock()
	defer s.lock.Unlock()

	idata, err := s.db.Get(ikey)
	if err != nil {
		s.doPut(chunk, ikey, &index, po)
		batchC := s.batchC
		go func() {
			<-batchC
			close(chunk.dbStored)
		}()
	} else {
		log.Trace(fmt.Sprintf("DbStore: chunk already exists, only update access"))
		decodeIndex(idata, &index)
		close(chunk.dbStored)
	}
	index.Access = s.accessCnt
	s.accessCnt++
	idata = encodeIndex(&index)
	s.batch.Put(ikey, idata)
	select {
	case s.batchesC <- struct{}{}:
	default:
	}
}

// force putting into db, does not check access index
func (s *LDBStore) doPut(chunk *Chunk, ikey []byte, index *dpaDBIndex, po uint8) {
	data := s.encodeDataFunc(chunk)
	s.batch.Put(getDataKey(s.dataIdx, po), data)
	index.Idx = s.dataIdx
	s.bucketCnt[po] = s.dataIdx
	s.entryCnt++
	s.dataIdx++

	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	s.batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))

}

func (s *LDBStore) writeBatches() {
	for range s.batchesC {
		s.lock.Lock()
		b := s.batch
		e := s.entryCnt
		d := s.dataIdx
		a := s.accessCnt
		c := s.batchC
		s.batchC = make(chan bool)
		s.batch = new(leveldb.Batch)
		s.lock.Unlock()
		err := s.writeBatch(b, e, d, a)
		// TODO: set this error on the batch, then tell the chunk
		if err != nil {
			log.Error(fmt.Sprintf("DbStore: spawn batch write (%d chunks): %v", b.Len(), err))
		}
		close(c)
		if e >= s.capacity {
			log.Trace(fmt.Sprintf("DbStore: collecting garbage...(%d chunks)", e))
			s.collectGarbage(gcArrayFreeRatio)
		}
	}
	log.Trace(fmt.Sprintf("DbStore: quit batch write loop"))
}

// must be called non concurrently
func (s *LDBStore) writeBatch(b *leveldb.Batch, entryCnt, dataIdx, accessCnt uint64) error {
	b.Put(keyEntryCnt, U64ToBytes(entryCnt))
	b.Put(keyDataIdx, U64ToBytes(dataIdx))
	b.Put(keyAccessCnt, U64ToBytes(accessCnt))
	l := b.Len()
	if err := s.db.Write(b); err != nil {
		return fmt.Errorf("unable to write batch: %v", err)
	}
	log.Trace(fmt.Sprintf("DbStore: batch write (%d chunks) complete", l))
	return nil
}

// newMockEncodeDataFunc returns a function that stores the chunk data
// to a mock store to bypass the default functionality encodeData.
// The constructed function always returns the nil data, as DbStore does
// not need to store the data, but still need to create the index.
func newMockEncodeDataFunc(mockStore *mock.NodeStore) func(chunk *Chunk) []byte {
	return func(chunk *Chunk) []byte {
		if err := mockStore.Put(chunk.Key, encodeData(chunk)); err != nil {
			log.Error(fmt.Sprintf("%T: Chunk %v put: %v", mockStore, chunk.Key.Log(), err))
		}
		return chunk.Key[:]
	}
}

// try to find index; if found, update access cnt and return true
func (s *LDBStore) tryAccessIdx(ikey []byte, index *dpaDBIndex) bool {
	idata, err := s.db.Get(ikey)
	if err != nil {
		return false
	}
	decodeIndex(idata, index)
	s.batch.Put(keyAccessCnt, U64ToBytes(s.accessCnt))
	s.accessCnt++
	index.Access = s.accessCnt
	idata = encodeIndex(index)
	s.batch.Put(ikey, idata)
	return true
}

func (s *LDBStore) Get(key Key) (chunk *Chunk, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.get(key)
}

func (s *LDBStore) get(key Key) (chunk *Chunk, err error) {
	var indx dpaDBIndex

	if s.tryAccessIdx(getIndexKey(key), &indx) {
		var data []byte
		if s.getDataFunc != nil {
			// if getDataFunc is defined, use it to retrieve the chunk data
			data, err = s.getDataFunc(key)
			if err != nil {
				return
			}
		} else {
			// default DbStore functionality to retrieve chunk data
			proximity := s.po(key)
			datakey := getDataKey(indx.Idx, proximity)
			data, err = s.db.Get(datakey)
			log.Trace(fmt.Sprintf("DBStore: Chunk %v indexkey %v datakey %x proximity %d", key.Log(), indx.Idx, datakey, proximity))
			if err != nil {
				log.Trace(fmt.Sprintf("DBStore: Chunk %v found but could not be accessed: %v", key.Log(), err))
				s.delete(indx.Idx, getIndexKey(key), s.po(key))
				return
			}
		}

		if !s.trusted {
			data_mod := data[32:]
			hasher := s.hashfunc()
			hasher.Write(data_mod)
			hash := hasher.Sum(nil)

			if !bytes.Equal(hash, key) {
				log.Error(fmt.Sprintf("Apparent key/hash mismatch. Hash %x, key %v", hash, key[:]))
				s.delete(indx.Idx, getIndexKey(key), s.po(key))
				log.Error("Invalid Chunk in Database. Please repair with command: 'swarm cleandb'")
			}
		}

		chunk = NewChunk(key, nil)
		decodeData(data, chunk)
	} else {
		err = ErrChunkNotFound
	}

	return
}

// newMockGetFunc returns a function that reads chunk data from
// the mock database, which is used as the value for DbStore.getFunc
// to bypass the default functionality of DbStore with a mock store.
func newMockGetDataFunc(mockStore *mock.NodeStore) func(key Key) (data []byte, err error) {
	return func(key Key) (data []byte, err error) {
		data, err = mockStore.Get(key)
		if err == mock.ErrNotFound {
			// preserve ErrChunkNotFound error
			err = ErrChunkNotFound
		}
		return data, err
	}
}

func (s *LDBStore) updateAccessCnt(key Key) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var index dpaDBIndex
	s.tryAccessIdx(getIndexKey(key), &index) // result_chn == nil, only update access cnt

}

func (s *LDBStore) setCapacity(c uint64) {

	s.lock.Lock()
	defer s.lock.Unlock()

	s.capacity = c

	if s.entryCnt > c {
		ratio := float32(1.01) - float32(c)/float32(s.entryCnt)
		if ratio < gcArrayFreeRatio {
			ratio = gcArrayFreeRatio
		}
		if ratio > 1 {
			ratio = 1
		}
		for s.entryCnt > c {
			s.collectGarbage(ratio)
		}
	}
}

func (s *LDBStore) Close() {
	s.db.Close()
}

// SyncIterator(start, stop, po, f) calls f on each hash of a bin po from start to stop
func (s *LDBStore) SyncIterator(since uint64, until uint64, po uint8, f func(Key, uint64) bool) error {
	sincekey := getDataKey(since, po)
	untilkey := getDataKey(until, po)
	it := s.db.NewIterator()
	defer it.Release()

	for ok := it.Seek(sincekey); ok; ok = it.Next() {
		dbkey := it.Key()
		if dbkey[0] != keyData || dbkey[1] != po || bytes.Compare(untilkey, dbkey) < 0 {
			break
		}
		key := make([]byte, 32)
		val := it.Value()
		copy(key, val[:32])
		if !f(Key(key), binary.BigEndian.Uint64(dbkey[2:])) {
			break
		}
	}
	return it.Error()
}

func databaseExists(path string) bool {
	o := &opt.Options{
		ErrorIfMissing: true,
	}
	tdb, err := leveldb.OpenFile(path, o)
	if err != nil {
		return false
	}
	defer tdb.Close()
	return true
}
