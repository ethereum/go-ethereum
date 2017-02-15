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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
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

type DbStore struct {
	db *LDBDatabase

	// this should be stored in db, accessed transactionally
	entryCnt, accessCnt, dataIdx, capacity uint64
	bucketCnt                              []uint64

	gcPos, gcStartPos []byte
	gcArray           []*gcItem

	hashfunc Hasher
	po       func(Key) uint8
	lock     sync.Mutex
}

// TODO: Instead of passing the distance function, just pass the address from which distances are calculated
// to avoid the appearance of a pluggable distance metric and opportunities of bugs associated with providing
// a function diferent from the one that is actually used.
func NewDbStore(path string, hash Hasher, capacity uint64, po func(Key) uint8) (*DbStore, error) {

	db, err := NewLDBDatabase(path)
	if err != nil {
		return nil, err
	}

	s := &DbStore{
		hashfunc: hash,
		db:       db,
	}

	s.po = po
	s.setCapacity(capacity)

	s.gcStartPos = make([]byte, 1)
	s.gcStartPos[0] = kpIndex
	s.gcArray = make([]*gcItem, gcArraySize)

	data, _ := s.db.Get(keyEntryCnt)
	s.entryCnt = BytesToU64(data)
	s.bucketCnt = make([]uint64, 0x100)
	for i := 0; i < 0x100; i++ {
		k := make([]byte, 2)
		k[0] = keyDistanceCnt
		k[1] = byte(uint8(i))
		cnt, _ := s.db.Get(k)
		s.bucketCnt[i] = BytesToU64(cnt)
	}
	data, _ = s.db.Get(keyAccessCnt)
	//s.accessCnt = BytesToU64(data)
	if len(data) == 8 {
		s.accessCnt = binary.LittleEndian.Uint64(data)
	}
	data, _ = s.db.Get(keyDataIdx)
	s.dataIdx = BytesToU64(data)
	s.gcPos, _ = s.db.Get(keyGCPos)
	if s.gcPos == nil {
		s.gcPos = s.gcStartPos
	}
	return s, nil
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

func (s *DbStore) updateIndexAccess(index *dpaDBIndex) {
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
	key[1] = byte(po)
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

func (s *DbStore) collectGarbage(ratio float32) {
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
			s.delete(s.gcArray[i].idx, s.gcArray[i].idxKey, s.po(Key(s.gcPos[1:])))
		}
	}

	s.db.Put(keyGCPos, s.gcPos)
}

func (s *DbStore) Cleanup() {
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

func (s *DbStore) Dump() {
	//Iterates over the database and checks that there are no faulty chunks
	it := s.db.NewIterator()
	startPosition := []byte{kpIndex}
	it.Seek(startPosition)
	var key []byte
	var total int
	for it.Valid() {
		key = it.Key()
		if (key == nil) || (key[0] != kpIndex) {
			break
		}
		total++
		fmt.Printf("%x\n", key[1:])
		it.Next()
	}
	it.Release()
	log.Warn(fmt.Sprintf("logged %v chunks", total))
}

func (s *DbStore) ReIndex() {
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
		key[1] = byte(s.po(Key(key[1:])))
		oldCntKey[1] = key[1]
		newCntKey[1] = byte(s.po(Key(newKey[1:])))
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

func (s *DbStore) delete(idx uint64, idxKey []byte, po uint8) {
	batch := new(leveldb.Batch)
	batch.Delete(idxKey)
	batch.Delete(getDataKey(idx, po))
	s.entryCnt--
	s.bucketCnt[po]--
	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))
	s.db.Write(batch)
}

func (s *DbStore) Size() uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.entryCnt
}

func (s *DbStore) CurrentStorageIndex() uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.dataIdx
}

func (s *DbStore) Put(chunk *Chunk) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ikey := getIndexKey(chunk.Key)
	var index dpaDBIndex

	if s.tryAccessIdx(ikey, &index) {
		if chunk.dbStored != nil {
			close(chunk.dbStored)
		}
		log.Trace(fmt.Sprintf("Storing to DB: chunk already exists, only update access"))
		return // already exists, only update access
	}

	data := encodeData(chunk)

	if s.entryCnt >= s.capacity {
		s.collectGarbage(gcArrayFreeRatio)
	}

	batch := new(leveldb.Batch)

	po := s.po(chunk.Key)
	t_datakey := getDataKey(s.dataIdx, po)
	batch.Put(t_datakey, data)
	log.Trace(fmt.Sprintf("batch put: datai		dx %v prox %v chunkkey %v datakey %v data %v", s.dataIdx, s.po(chunk.Key), hex.EncodeToString(chunk.Key), t_datakey, hex.EncodeToString(data[0:64])))

	index.Idx = s.dataIdx
	s.updateIndexAccess(&index)

	idata := encodeIndex(&index)
	batch.Put(ikey, idata)

	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	s.entryCnt++
	batch.Put(keyDataIdx, U64ToBytes(s.dataIdx))
	s.dataIdx++
	accesscnt := make([]byte, 8)
	binary.LittleEndian.PutUint64(accesscnt, s.accessCnt)
	batch.Put(keyAccessCnt, accesscnt)
	s.accessCnt++

	s.bucketCnt[po]++
	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))

	s.db.Write(batch)
	if chunk.dbStored != nil {
		close(chunk.dbStored)
	}
	log.Trace(fmt.Sprintf("DbStore.Put: %v. db storage counter: %v ", chunk.Key.Log(), s.dataIdx))
}

// try to find index; if found, update access cnt and return true
func (s *DbStore) tryAccessIdx(ikey []byte, index *dpaDBIndex) bool {
	idata, err := s.db.Get(ikey)
	if err != nil {
		return false
	}
	decodeIndex(idata, index)

	batch := new(leveldb.Batch)

	accesscnt := make([]byte, 8)
	binary.LittleEndian.PutUint64(accesscnt, s.accessCnt)
	batch.Put(keyAccessCnt, accesscnt)

	s.accessCnt++
	s.updateIndexAccess(index)
	idata = encodeIndex(index)
	batch.Put(ikey, idata)

	s.db.Write(batch)

	return true
}

func (s *DbStore) Get(key Key) (chunk *Chunk, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.get(key)
}

func (s *DbStore) get(key Key) (chunk *Chunk, err error) {
	var indx dpaDBIndex

	if s.tryAccessIdx(getIndexKey(key), &indx) {
		var data []byte

		proximity := s.po(key)
		datakey := getDataKey(indx.Idx, proximity)
		data, err = s.db.Get(datakey)
		log.Trace(fmt.Sprintf("DBStore: Chunk %v indexkey %x datakey %x proximity %d", key.Log(), indx.Idx, datakey, proximity))
		if err != nil {
			log.Trace(fmt.Sprintf("DBStore: Chunk %v found but could not be accessed: %v", key.Log(), err))
			s.delete(indx.Idx, getIndexKey(key), s.po(key))
			return
		}

		//
		data_mod := data[32:]

		hasher := s.hashfunc()
		hasher.Write(data_mod)
		hash := hasher.Sum(nil)

		if !bytes.Equal(hash, key) {
			s.delete(index.Idx, getIndexKey(key))
			log.Warn("Invalid Chunk in Database. Please repair with command: 'swarm cleandb'")
		}

		chunk = &Chunk{
			Key: key,
		}
		decodeData(data, chunk)
	} else {
		err = notFound
	}

	return

}

func (s *DbStore) updateAccessCnt(key Key) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var index dpaDBIndex
	s.tryAccessIdx(getIndexKey(key), &index) // result_chn == nil, only update access cnt

}

func (s *DbStore) setCapacity(c uint64) {

	s.lock.Lock()
	defer s.lock.Unlock()

	s.capacity = c

	if s.entryCnt > c {
		var ratio float32
		ratio = float32(1.01) - float32(c)/float32(s.entryCnt)
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

func (s *DbStore) getEntryCnt() uint64 {
	return s.entryCnt
}

func (s *DbStore) Close() {
	s.db.Close()
}

// initialises a sync iterator from a syncToken (passed in with the handshake)
func (s *DbStore) SyncIterator(since uint64, until uint64, po uint8, f func(Key, uint64) bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	untilkey := getDataKey(until, po)

	it := s.db.NewIterator()
	it.Seek(getDataKey(since, po))
	defer it.Release()
	for it.Valid() {
		dbkey := it.Key()
		if dbkey[0] != keyData || dbkey[1] != byte(po) || bytes.Compare(untilkey, dbkey) < 0 {
			break
		}

		key := make([]byte, 32)
		copy(key, it.Value()[:32])
		if !f(Key(key), binary.BigEndian.Uint64(dbkey[2:])) {
			break
		}
		it.Next()
	}
	return nil
}

func Import(sourcepath string, targetpath string, sourceaccountkey string, targetaccountkey string) (uint64, error) {
	chunkcount := uint64(0)
	var j uint64
	var maxcount uint64 = 0
	var poc uint16
	var err error
	maxcount--

	var chunks_in KeyCollection
	var chunks_out KeyCollection

	sourceaccountkeyhash := common.HexToHash(sourceaccountkey[2:])
	targetaccountkeyhash := common.HexToHash(targetaccountkey[2:])

	log.Trace(fmt.Sprintf("srckey %x targetkey %x", sourceaccountkeyhash, targetaccountkeyhash))

	pofunc_source := func(k Key) (ret uint8) {
		return uint8(Proximity(sourceaccountkeyhash[:], k[:]))
	}

	pofunc_target := func(k Key) (ret uint8) {
		return uint8(Proximity(targetaccountkeyhash[:], k[:]))
	}

	if !databaseExists(sourcepath) {
		return 0, fmt.Errorf("sourcepath '%s' does not exist or is unavailable (someone else using it?)", sourcepath)
	}
	if !databaseExists(targetpath) {
		return 0, fmt.Errorf("targetpath '%s' does not exist or is unavailable (someone else using it?)", targetpath)
	}

	store_source, err := NewDbStore(sourcepath, MakeHashFunc(defaultHash), defaultDbCapacity, pofunc_source)
	if err != nil {
		return 0, err
	}
	store_target, err := NewDbStore(targetpath, MakeHashFunc(defaultHash), defaultDbCapacity, pofunc_target)
	if err != nil {
		return 0, err
	}

	// why does this have to be +1? should not be necessary
	// if not +1, the arrays in the iterator overflow
	chunks_in = NewKeyCollection(int(store_source.Size()) + 1)
	chunks_out = NewKeyCollection(int(store_source.Size()) + 1)
	bins := make([]int8, int(store_source.Size())+1)

	log.Trace(fmt.Sprintf("Source db count: %v, Target db count: %v ", store_source.Size(), store_target.Size()))

	for poc = 0; poc <= 255; poc++ {
		err := store_source.SyncIterator(0, store_source.CurrentStorageIndex(), uint8(poc), func(k Key, n uint64) bool {
			chunks_in[n] = make(Key, 32)
			copy(chunks_in[n], k)
			bins[n] = int8(poc)
			log.Trace(fmt.Sprintf("Iterator sc #%d '%v' (array stored: '%v')", n, k, chunks_in[n]))
			chunkcount++
			return true
		})
		if err != nil {
			return 0, fmt.Errorf("Iterator error, import aborted: %v", err)
		}
	}

	for j = 0; j < chunkcount; j++ {
		chunk, err := store_source.Get(chunks_in[j])
		if err != nil {
			log.Trace(fmt.Sprintf("Chunk get sc %d bin %d key '%v' FAIL: %v", j, bins[j], chunks_in[j], err))
		} else {
			log.Trace(fmt.Sprintf("Chunk get sc %d bin %d key '%v' OK", j, bins[j], chunks_in[j]))
			store_target.Put(chunk)
			chunks_out[j] = make(Key, 32)
			copy(chunks_out[j], chunk.Key)

		}
	}
	return chunkcount, nil
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
