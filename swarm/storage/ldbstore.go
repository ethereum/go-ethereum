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
// DbStore implements the ChunkStore interface and is used by the FileStore as
// persistent storage of chunks
// it implements purging based on access count allowing for external control of
// max capacity

package storage

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	gcArrayFreeRatio = 0.1
	maxGCitems       = 5000 // max number of items to be gc'd per call to collectGarbage()
)

var (
	keyIndex       = byte(0)
	keyOldData     = byte(1)
	keyAccessCnt   = []byte{2}
	keyEntryCnt    = []byte{3}
	keyDataIdx     = []byte{4}
	keyData        = byte(6)
	keyDistanceCnt = byte(7)
)

type gcItem struct {
	idx    uint64
	value  uint64
	idxKey []byte
	po     uint8
}

type LDBStoreParams struct {
	*StoreParams
	Path string
	Po   func(Address) uint8
}

// NewLDBStoreParams constructs LDBStoreParams with the specified values.
func NewLDBStoreParams(storeparams *StoreParams, path string) *LDBStoreParams {
	return &LDBStoreParams{
		StoreParams: storeparams,
		Path:        path,
		Po:          func(k Address) (ret uint8) { return uint8(Proximity(storeparams.BaseKey[:], k[:])) },
	}
}

type LDBStore struct {
	db *LDBDatabase

	// this should be stored in db, accessed transactionally
	entryCnt  uint64 // number of items in the LevelDB
	accessCnt uint64 // ever-accumulating number increased every time we read/access an entry
	dataIdx   uint64 // similar to entryCnt, but we only increment it
	capacity  uint64
	bucketCnt []uint64

	hashfunc SwarmHasher
	po       func(Address) uint8

	batchC   chan bool
	batchesC chan struct{}
	batch    *leveldb.Batch
	lock     sync.RWMutex
	quit     chan struct{}

	// Functions encodeDataFunc is used to bypass
	// the default functionality of DbStore with
	// mock.NodeStore for testing purposes.
	encodeDataFunc func(chunk *Chunk) []byte
	// If getDataFunc is defined, it will be used for
	// retrieving the chunk data instead from the local
	// LevelDB database.
	getDataFunc func(addr Address) (data []byte, err error)
}

// TODO: Instead of passing the distance function, just pass the address from which distances are calculated
// to avoid the appearance of a pluggable distance metric and opportunities of bugs associated with providing
// a function different from the one that is actually used.
func NewLDBStore(params *LDBStoreParams) (s *LDBStore, err error) {
	s = new(LDBStore)
	s.hashfunc = params.Hash
	s.quit = make(chan struct{})

	s.batchC = make(chan bool)
	s.batchesC = make(chan struct{}, 1)
	go s.writeBatches()
	s.batch = new(leveldb.Batch)
	// associate encodeData with default functionality
	s.encodeDataFunc = encodeData

	s.db, err = NewLDBDatabase(params.Path)
	if err != nil {
		return nil, err
	}

	s.po = params.Po
	s.setCapacity(params.DbCapacity)

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

	return s, nil
}

// NewMockDbStore creates a new instance of DbStore with
// mockStore set to a provided value. If mockStore argument is nil,
// this function behaves exactly as NewDbStore.
func NewMockDbStore(params *LDBStoreParams, mockStore *mock.NodeStore) (s *LDBStore, err error) {
	s, err = NewLDBStore(params)
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
	return binary.BigEndian.Uint64(data)
}

func U64ToBytes(val uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, val)
	return data
}

func (s *LDBStore) updateIndexAccess(index *dpaDBIndex) {
	index.Access = s.accessCnt
}

func getIndexKey(hash Address) []byte {
	hashSize := len(hash)
	key := make([]byte, hashSize+1)
	key[0] = keyIndex
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
	// Always create a new underlying array for the returned byte slice.
	// The chunk.Key array may be used in the returned slice which
	// may be changed later in the code or by the LevelDB, resulting
	// that the Key is changed as well.
	return append(append([]byte{}, chunk.Addr[:]...), chunk.SData...)
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

func (s *LDBStore) collectGarbage(ratio float32) {
	metrics.GetOrRegisterCounter("ldbstore.collectgarbage", nil).Inc(1)

	it := s.db.NewIterator()
	defer it.Release()

	garbage := []*gcItem{}
	gcnt := 0

	for ok := it.Seek([]byte{keyIndex}); ok && (gcnt < maxGCitems) && (uint64(gcnt) < s.entryCnt); ok = it.Next() {
		itkey := it.Key()

		if (itkey == nil) || (itkey[0] != keyIndex) {
			break
		}

		// it.Key() contents change on next call to it.Next(), so we must copy it
		key := make([]byte, len(it.Key()))
		copy(key, it.Key())

		val := it.Value()

		var index dpaDBIndex

		hash := key[1:]
		decodeIndex(val, &index)
		po := s.po(hash)

		gci := &gcItem{
			idxKey: key,
			idx:    index.Idx,
			value:  index.Access, // the smaller, the more likely to be gc'd. see sort comparator below.
			po:     po,
		}

		garbage = append(garbage, gci)
		gcnt++
	}

	sort.Slice(garbage[:gcnt], func(i, j int) bool { return garbage[i].value < garbage[j].value })

	cutoff := int(float32(gcnt) * ratio)
	metrics.GetOrRegisterCounter("ldbstore.collectgarbage.delete", nil).Inc(int64(cutoff))

	for i := 0; i < cutoff; i++ {
		s.delete(garbage[i].idx, garbage[i].idxKey, garbage[i].po)
	}
}

// Export writes all chunks from the store to a tar archive, returning the
// number of chunks written.
func (s *LDBStore) Export(out io.Writer) (int64, error) {
	tw := tar.NewWriter(out)
	defer tw.Close()

	it := s.db.NewIterator()
	defer it.Release()
	var count int64
	for ok := it.Seek([]byte{keyIndex}); ok; ok = it.Next() {
		key := it.Key()
		if (key == nil) || (key[0] != keyIndex) {
			break
		}

		var index dpaDBIndex

		hash := key[1:]
		decodeIndex(it.Value(), &index)
		po := s.po(hash)
		datakey := getDataKey(index.Idx, po)
		log.Trace("store.export", "dkey", fmt.Sprintf("%x", datakey), "dataidx", index.Idx, "po", po)
		data, err := s.db.Get(datakey)
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

		keybytes, err := hex.DecodeString(hdr.Name)
		if err != nil {
			log.Warn("ignoring invalid chunk file", "name", hdr.Name, "err", err)
			continue
		}

		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return count, err
		}
		key := Address(keybytes)
		chunk := NewChunk(key, nil)
		chunk.SData = data[32:]
		s.Put(context.TODO(), chunk)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-chunk.dbStoredC
		}()
		count++
	}
	wg.Wait()
	return count, nil
}

func (s *LDBStore) Cleanup() {
	//Iterates over the database and checks that there are no chunks bigger than 4kb
	var errorsFound, removed, total int

	it := s.db.NewIterator()
	defer it.Release()
	for ok := it.Seek([]byte{keyIndex}); ok; ok = it.Next() {
		key := it.Key()
		if (key == nil) || (key[0] != keyIndex) {
			break
		}
		total++
		var index dpaDBIndex
		err := decodeIndex(it.Value(), &index)
		if err != nil {
			log.Warn("Cannot decode")
			errorsFound++
			continue
		}
		hash := key[1:]
		po := s.po(hash)
		datakey := getDataKey(index.Idx, po)
		data, err := s.db.Get(datakey)
		if err != nil {
			found := false

			// highest possible proximity is 255
			for po = 1; po <= 255; po++ {
				datakey = getDataKey(index.Idx, po)
				data, err = s.db.Get(datakey)
				if err == nil {
					found = true
					break
				}
			}

			if !found {
				log.Warn(fmt.Sprintf("Chunk %x found but count not be accessed with any po", key[:]))
				errorsFound++
				continue
			}
		}

		c := &Chunk{}
		ck := data[:32]
		decodeData(data, c)

		cs := int64(binary.LittleEndian.Uint64(c.SData[:8]))
		log.Trace("chunk", "key", fmt.Sprintf("%x", key[:]), "ck", fmt.Sprintf("%x", ck), "dkey", fmt.Sprintf("%x", datakey), "dataidx", index.Idx, "po", po, "len data", len(data), "len sdata", len(c.SData), "size", cs)

		if len(c.SData) > chunk.DefaultSize+8 {
			log.Warn("chunk for cleanup", "key", fmt.Sprintf("%x", key[:]), "ck", fmt.Sprintf("%x", ck), "dkey", fmt.Sprintf("%x", datakey), "dataidx", index.Idx, "po", po, "len data", len(data), "len sdata", len(c.SData), "size", cs)
			s.delete(index.Idx, getIndexKey(key[1:]), po)
			removed++
			errorsFound++
		}
	}

	log.Warn(fmt.Sprintf("Found %v errors out of %v entries. Removed %v chunks.", errorsFound, total, removed))
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
		key[1] = s.po(Address(key[1:]))
		oldCntKey[1] = key[1]
		newCntKey[1] = s.po(Address(newKey[1:]))
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
	metrics.GetOrRegisterCounter("ldbstore.delete", nil).Inc(1)

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

func (s *LDBStore) Put(ctx context.Context, chunk *Chunk) {
	metrics.GetOrRegisterCounter("ldbstore.put", nil).Inc(1)
	log.Trace("ldbstore.put", "key", chunk.Addr)

	ikey := getIndexKey(chunk.Addr)
	var index dpaDBIndex

	po := s.po(chunk.Addr)
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Trace("ldbstore.put: s.db.Get", "key", chunk.Addr, "ikey", fmt.Sprintf("%x", ikey))
	idata, err := s.db.Get(ikey)
	if err != nil {
		s.doPut(chunk, &index, po)
		batchC := s.batchC
		go func() {
			<-batchC
			chunk.markAsStored()
		}()
	} else {
		log.Trace("ldbstore.put: chunk already exists, only update access", "key", chunk.Addr)
		decodeIndex(idata, &index)
		chunk.markAsStored()
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
func (s *LDBStore) doPut(chunk *Chunk, index *dpaDBIndex, po uint8) {
	data := s.encodeDataFunc(chunk)
	dkey := getDataKey(s.dataIdx, po)
	s.batch.Put(dkey, data)
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
mainLoop:
	for {
		select {
		case <-s.quit:
			break mainLoop
		case <-s.batchesC:
			s.lock.Lock()
			b := s.batch
			e := s.entryCnt
			d := s.dataIdx
			a := s.accessCnt
			c := s.batchC
			s.batchC = make(chan bool)
			s.batch = new(leveldb.Batch)
			err := s.writeBatch(b, e, d, a)
			// TODO: set this error on the batch, then tell the chunk
			if err != nil {
				log.Error(fmt.Sprintf("spawn batch write (%d entries): %v", b.Len(), err))
			}
			close(c)
			for e > s.capacity {
				// Collect garbage in a separate goroutine
				// to be able to interrupt this loop by s.quit.
				done := make(chan struct{})
				go func() {
					s.collectGarbage(gcArrayFreeRatio)
					close(done)
				}()

				e = s.entryCnt
				select {
				case <-s.quit:
					s.lock.Unlock()
					break mainLoop
				case <-done:
				}
			}
			s.lock.Unlock()
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
	log.Trace(fmt.Sprintf("batch write (%d entries)", l))
	return nil
}

// newMockEncodeDataFunc returns a function that stores the chunk data
// to a mock store to bypass the default functionality encodeData.
// The constructed function always returns the nil data, as DbStore does
// not need to store the data, but still need to create the index.
func newMockEncodeDataFunc(mockStore *mock.NodeStore) func(chunk *Chunk) []byte {
	return func(chunk *Chunk) []byte {
		if err := mockStore.Put(chunk.Addr, encodeData(chunk)); err != nil {
			log.Error(fmt.Sprintf("%T: Chunk %v put: %v", mockStore, chunk.Addr.Log(), err))
		}
		return chunk.Addr[:]
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
	select {
	case s.batchesC <- struct{}{}:
	default:
	}
	return true
}

func (s *LDBStore) Get(ctx context.Context, addr Address) (chunk *Chunk, err error) {
	metrics.GetOrRegisterCounter("ldbstore.get", nil).Inc(1)
	log.Trace("ldbstore.get", "key", addr)

	s.lock.Lock()
	defer s.lock.Unlock()
	return s.get(addr)
}

func (s *LDBStore) get(addr Address) (chunk *Chunk, err error) {
	var indx dpaDBIndex

	if s.tryAccessIdx(getIndexKey(addr), &indx) {
		var data []byte
		if s.getDataFunc != nil {
			// if getDataFunc is defined, use it to retrieve the chunk data
			log.Trace("ldbstore.get retrieve with getDataFunc", "key", addr)
			data, err = s.getDataFunc(addr)
			if err != nil {
				return
			}
		} else {
			// default DbStore functionality to retrieve chunk data
			proximity := s.po(addr)
			datakey := getDataKey(indx.Idx, proximity)
			data, err = s.db.Get(datakey)
			log.Trace("ldbstore.get retrieve", "key", addr, "indexkey", indx.Idx, "datakey", fmt.Sprintf("%x", datakey), "proximity", proximity)
			if err != nil {
				log.Trace("ldbstore.get chunk found but could not be accessed", "key", addr, "err", err)
				s.delete(indx.Idx, getIndexKey(addr), s.po(addr))
				return
			}
		}

		chunk = NewChunk(addr, nil)
		chunk.markAsStored()
		decodeData(data, chunk)
	} else {
		err = ErrChunkNotFound
	}

	return
}

// newMockGetFunc returns a function that reads chunk data from
// the mock database, which is used as the value for DbStore.getFunc
// to bypass the default functionality of DbStore with a mock store.
func newMockGetDataFunc(mockStore *mock.NodeStore) func(addr Address) (data []byte, err error) {
	return func(addr Address) (data []byte, err error) {
		data, err = mockStore.Get(addr)
		if err == mock.ErrNotFound {
			// preserve ErrChunkNotFound error
			err = ErrChunkNotFound
		}
		return data, err
	}
}

func (s *LDBStore) updateAccessCnt(addr Address) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var index dpaDBIndex
	s.tryAccessIdx(getIndexKey(addr), &index) // result_chn == nil, only update access cnt

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
	close(s.quit)
	s.db.Close()
}

// SyncIterator(start, stop, po, f) calls f on each hash of a bin po from start to stop
func (s *LDBStore) SyncIterator(since uint64, until uint64, po uint8, f func(Address, uint64) bool) error {
	metrics.GetOrRegisterCounter("ldbstore.synciterator", nil).Inc(1)

	sincekey := getDataKey(since, po)
	untilkey := getDataKey(until, po)
	it := s.db.NewIterator()
	defer it.Release()

	for ok := it.Seek(sincekey); ok; ok = it.Next() {
		metrics.GetOrRegisterCounter("ldbstore.synciterator.seek", nil).Inc(1)

		dbkey := it.Key()
		if dbkey[0] != keyData || dbkey[1] != po || bytes.Compare(untilkey, dbkey) < 0 {
			break
		}
		key := make([]byte, 32)
		val := it.Value()
		copy(key, val[:32])
		if !f(Address(key), binary.BigEndian.Uint64(dbkey[2:])) {
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
