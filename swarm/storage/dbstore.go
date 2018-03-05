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
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
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
	keyAccessCnt = []byte{2}
	keyEntryCnt  = []byte{3}
	keyDataIdx   = []byte{4}
	keyGCPos     = []byte{5}
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

	gcPos, gcStartPos []byte
	gcArray           []*gcItem

	hashfunc SwarmHasher

	lock sync.Mutex
}

func NewDbStore(path string, hash SwarmHasher, capacity uint64, radius int) (s *DbStore, err error) {
	s = new(DbStore)

	s.hashfunc = hash

	s.db, err = NewLDBDatabase(path)
	if err != nil {
		return
	}

	s.setCapacity(capacity)

	s.gcStartPos = make([]byte, 1)
	s.gcStartPos[0] = kpIndex
	s.gcArray = make([]*gcItem, gcArraySize)

	data, _ := s.db.Get(keyEntryCnt)
	s.entryCnt = BytesToU64(data)
	data, _ = s.db.Get(keyAccessCnt)
	s.accessCnt = BytesToU64(data)
	data, _ = s.db.Get(keyDataIdx)
	s.dataIdx = BytesToU64(data)
	s.gcPos, _ = s.db.Get(keyGCPos)
	if s.gcPos == nil {
		s.gcPos = s.gcStartPos
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
	return binary.LittleEndian.Uint64(data)
}

func U64ToBytes(val uint64) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, val)
	return data
}

func getIndexGCValue(index *dpaDBIndex) uint64 {
	return index.Access
}

func (s *DbStore) updateIndexAccess(index *dpaDBIndex) {
	index.Access = s.accessCnt
}

func getIndexKey(hash Key) []byte {
	HashSize := len(hash)
	key := make([]byte, HashSize+1)
	key[0] = 0
	copy(key[1:], hash[:])
	return key
}

func getDataKey(idx uint64) []byte {
	key := make([]byte, 9)
	key[0] = 1
	binary.BigEndian.PutUint64(key[1:9], idx)

	return key
}

func encodeIndex(index *dpaDBIndex) []byte {
	data, _ := rlp.EncodeToBytes(index)
	return data
}

func encodeData(chunk *Chunk) []byte {
	return chunk.SData
}

func decodeIndex(data []byte, index *dpaDBIndex) {
	dec := rlp.NewStream(bytes.NewReader(data), 0)
	dec.Decode(index)
}

func decodeData(data []byte, chunk *Chunk) {
	chunk.SData = data
	chunk.Size = int64(binary.LittleEndian.Uint64(data[0:8]))
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

	// fmt.Print(gcnt, " ", s.entryCnt, " ")

	// actual gc
	for i := 0; i < gcnt; i++ {
		if s.gcArray[i].value <= cutval {
			gcCounter.Inc(1)
			s.delete(s.gcArray[i].idx, s.gcArray[i].idxKey)
		}
	}

	// fmt.Println(s.entryCnt)

	s.db.Put(keyGCPos, s.gcPos)
}

// Export writes all chunks from the store to a tar archive, returning the
// number of chunks written.
func (s *DbStore) Export(out io.Writer) (int64, error) {
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

		data, err := s.db.Get(getDataKey(index.Idx))
		if err != nil {
			log.Warn(fmt.Sprintf("Chunk %x found but could not be accessed: %v", key[:], err))
			continue
		}

		hdr := &tar.Header{
			Name: hex.EncodeToString(key[1:]),
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

// Import reads chunks into the store from a tar archive, returning the number
// of chunks read.
func (s *DbStore) Import(in io.Reader) (int64, error) {
	tr := tar.NewReader(in)

	var count int64
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

		s.Put(&Chunk{Key: key, SData: data})
		count++
	}

	return count, nil
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
		decodeIndex(it.Value(), &index)

		data, err := s.db.Get(getDataKey(index.Idx))
		if err != nil {
			log.Warn(fmt.Sprintf("Chunk %x found but could not be accessed: %v", key[:], err))
			s.delete(index.Idx, getIndexKey(key[1:]))
			errorsFound++
		} else {
			hasher := s.hashfunc()
			hasher.Write(data)
			hash := hasher.Sum(nil)
			if !bytes.Equal(hash, key[1:]) {
				log.Warn(fmt.Sprintf("Found invalid chunk. Hash mismatch. hash=%x, key=%x", hash, key[:]))
				s.delete(index.Idx, getIndexKey(key[1:]))
				errorsFound++
			}
		}
		it.Next()
	}
	it.Release()
	log.Warn(fmt.Sprintf("Found %v errors out of %v entries", errorsFound, total))
}

func (s *DbStore) delete(idx uint64, idxKey []byte) {
	batch := new(leveldb.Batch)
	batch.Delete(idxKey)
	batch.Delete(getDataKey(idx))
	dbStoreDeleteCounter.Inc(1)
	s.entryCnt--
	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	s.db.Write(batch)
}

func (s *DbStore) Counter() uint64 {
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
	//data := ethutil.Encode([]interface{}{entry})

	if s.entryCnt >= s.capacity {
		s.collectGarbage(gcArrayFreeRatio)
	}

	batch := new(leveldb.Batch)

	batch.Put(getDataKey(s.dataIdx), data)

	index.Idx = s.dataIdx
	s.updateIndexAccess(&index)

	idata := encodeIndex(&index)
	batch.Put(ikey, idata)

	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	s.entryCnt++
	batch.Put(keyDataIdx, U64ToBytes(s.dataIdx))
	s.dataIdx++
	batch.Put(keyAccessCnt, U64ToBytes(s.accessCnt))
	s.accessCnt++

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

	batch.Put(keyAccessCnt, U64ToBytes(s.accessCnt))
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

	var index dpaDBIndex

	if s.tryAccessIdx(getIndexKey(key), &index) {
		var data []byte
		data, err = s.db.Get(getDataKey(index.Idx))
		if err != nil {
			log.Trace(fmt.Sprintf("DBStore: Chunk %v found but could not be accessed: %v", key.Log(), err))
			s.delete(index.Idx, getIndexKey(key))
			return
		}

		hasher := s.hashfunc()
		hasher.Write(data)
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

func (s *DbStore) Close() {
	s.db.Close()
}

//  describes a section of the DbStore representing the unsynced
// domain relevant to a peer
// Start - Stop designate a continuous area Keys in an address space
// typically the addresses closer to us than to the peer but not closer
// another closer peer in between
// From - To designates a time interval typically from the last disconnect
// till the latest connection (real time traffic is relayed)
type DbSyncState struct {
	Start, Stop Key
	First, Last uint64
}

// implements the syncer iterator interface
// iterates by storage index (~ time of storage = first entry to db)
type dbSyncIterator struct {
	it iterator.Iterator
	DbSyncState
}

// initialises a sync iterator from a syncToken (passed in with the handshake)
func (self *DbStore) NewSyncIterator(state DbSyncState) (si *dbSyncIterator, err error) {
	if state.First > state.Last {
		return nil, fmt.Errorf("no entries found")
	}
	si = &dbSyncIterator{
		it:          self.db.NewIterator(),
		DbSyncState: state,
	}
	si.it.Seek(getIndexKey(state.Start))
	return si, nil
}

// walk the area from Start to Stop and returns items within time interval
// First to Last
func (self *dbSyncIterator) Next() (key Key) {
	for self.it.Valid() {
		dbkey := self.it.Key()
		if dbkey[0] != 0 {
			break
		}
		key = Key(make([]byte, len(dbkey)-1))
		copy(key[:], dbkey[1:])
		if bytes.Compare(key[:], self.Start) <= 0 {
			self.it.Next()
			continue
		}
		if bytes.Compare(key[:], self.Stop) > 0 {
			break
		}
		var index dpaDBIndex
		decodeIndex(self.it.Value(), &index)
		self.it.Next()
		if (index.Idx >= self.First) && (index.Idx < self.Last) {
			return
		}
	}
	self.it.Release()
	return nil
}
