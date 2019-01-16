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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	defaultGCRatio    = 10
	defaultMaxGCRound = 10000
	defaultMaxGCBatch = 5000

	wEntryCnt  = 1 << 0
	wIndexCnt  = 1 << 1
	wAccessCnt = 1 << 2
)

var (
	dbEntryCount = metrics.NewRegisteredCounter("ldbstore.entryCnt", nil)
)

var (
	keyIndex       = byte(0)
	keyAccessCnt   = []byte{2}
	keyEntryCnt    = []byte{3}
	keyDataIdx     = []byte{4}
	keyData        = byte(6)
	keyDistanceCnt = byte(7)
	keySchema      = []byte{8}
	keyGCIdx       = byte(9) // access to chunk data index, used by garbage collection in ascending order from first entry
)

var (
	ErrDBClosed = errors.New("LDBStore closed")
)

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
		Po:          func(k Address) (ret uint8) { return uint8(Proximity(storeparams.BaseKey, k[:])) },
	}
}

type garbage struct {
	maxRound int           // maximum number of chunks to delete in one garbage collection round
	maxBatch int           // maximum number of chunks to delete in one db request batch
	ratio    int           // 1/x ratio to calculate the number of chunks to gc on a low capacity db
	count    int           // number of chunks deleted in running round
	target   int           // number of chunks to delete in running round
	batch    *dbBatch      // the delete batch
	runC     chan struct{} // struct in chan means gc is NOT running
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

	batchesC chan struct{}
	closed   bool
	batch    *dbBatch
	lock     sync.RWMutex
	quit     chan struct{}
	gc       *garbage

	// Functions encodeDataFunc is used to bypass
	// the default functionality of DbStore with
	// mock.NodeStore for testing purposes.
	encodeDataFunc func(chunk Chunk) []byte
	// If getDataFunc is defined, it will be used for
	// retrieving the chunk data instead from the local
	// LevelDB database.
	getDataFunc func(key Address) (data []byte, err error)
}

type dbBatch struct {
	*leveldb.Batch
	err error
	c   chan struct{}
}

func newBatch() *dbBatch {
	return &dbBatch{Batch: new(leveldb.Batch), c: make(chan struct{})}
}

// TODO: Instead of passing the distance function, just pass the address from which distances are calculated
// to avoid the appearance of a pluggable distance metric and opportunities of bugs associated with providing
// a function different from the one that is actually used.
func NewLDBStore(params *LDBStoreParams) (s *LDBStore, err error) {
	s = new(LDBStore)
	s.hashfunc = params.Hash
	s.quit = make(chan struct{})

	s.batchesC = make(chan struct{}, 1)
	go s.writeBatches()
	s.batch = newBatch()
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
	}
	data, _ := s.db.Get(keyEntryCnt)
	s.entryCnt = BytesToU64(data)
	data, _ = s.db.Get(keyAccessCnt)
	s.accessCnt = BytesToU64(data)
	data, _ = s.db.Get(keyDataIdx)
	s.dataIdx = BytesToU64(data)

	// set up garbage collection
	s.gc = &garbage{
		maxBatch: defaultMaxGCBatch,
		maxRound: defaultMaxGCRound,
		ratio:    defaultGCRatio,
	}

	s.gc.runC = make(chan struct{}, 1)
	s.gc.runC <- struct{}{}

	return s, nil
}

// MarkAccessed increments the access counter as a best effort for a chunk, so
// the chunk won't get garbage collected.
func (s *LDBStore) MarkAccessed(addr Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return
	}

	proximity := s.po(addr)
	s.tryAccessIdx(addr, proximity)
}

// initialize and set values for processing of gc round
func (s *LDBStore) startGC(c int) {

	s.gc.count = 0
	// calculate the target number of deletions
	if c >= s.gc.maxRound {
		s.gc.target = s.gc.maxRound
	} else {
		s.gc.target = c / s.gc.ratio
	}
	s.gc.batch = newBatch()
	log.Debug("startgc", "requested", c, "target", s.gc.target)
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

func getIndexKey(hash Address) []byte {
	hashSize := len(hash)
	key := make([]byte, hashSize+1)
	key[0] = keyIndex
	copy(key[1:], hash[:])
	return key
}

func getDataKey(idx uint64, po uint8) []byte {
	key := make([]byte, 10)
	key[0] = keyData
	key[1] = po
	binary.BigEndian.PutUint64(key[2:], idx)

	return key
}

func getGCIdxKey(index *dpaDBIndex) []byte {
	key := make([]byte, 9)
	key[0] = keyGCIdx
	binary.BigEndian.PutUint64(key[1:], index.Access)
	return key
}

func getGCIdxValue(index *dpaDBIndex, po uint8, addr Address) []byte {
	val := make([]byte, 41) // po = 1, index.Index = 8, Address = 32
	val[0] = po
	binary.BigEndian.PutUint64(val[1:], index.Idx)
	copy(val[9:], addr)
	return val
}

func parseIdxKey(key []byte) (byte, []byte) {
	return key[0], key[1:]
}

func parseGCIdxEntry(accessCnt []byte, val []byte) (index *dpaDBIndex, po uint8, addr Address) {
	index = &dpaDBIndex{
		Idx:    binary.BigEndian.Uint64(val[1:]),
		Access: binary.BigEndian.Uint64(accessCnt),
	}
	po = val[0]
	addr = val[9:]
	return
}

func encodeIndex(index *dpaDBIndex) []byte {
	data, _ := rlp.EncodeToBytes(index)
	return data
}

func encodeData(chunk Chunk) []byte {
	// Always create a new underlying array for the returned byte slice.
	// The chunk.Address array may be used in the returned slice which
	// may be changed later in the code or by the LevelDB, resulting
	// that the Address is changed as well.
	return append(append([]byte{}, chunk.Address()[:]...), chunk.Data()...)
}

func decodeIndex(data []byte, index *dpaDBIndex) error {
	dec := rlp.NewStream(bytes.NewReader(data), 0)
	return dec.Decode(index)
}

func decodeData(addr Address, data []byte) (*chunk, error) {
	return NewChunk(addr, data[32:]), nil
}

func (s *LDBStore) collectGarbage() error {

	// prevent duplicate gc from starting when one is already running
	select {
	case <-s.gc.runC:
	default:
		return nil
	}

	s.lock.Lock()
	entryCnt := s.entryCnt
	s.lock.Unlock()

	metrics.GetOrRegisterCounter("ldbstore.collectgarbage", nil).Inc(1)

	// calculate the amount of chunks to collect and reset counter
	s.startGC(int(entryCnt))
	log.Debug("collectGarbage", "target", s.gc.target, "entryCnt", entryCnt)

	var totalDeleted int
	for s.gc.count < s.gc.target {
		it := s.db.NewIterator()
		ok := it.Seek([]byte{keyGCIdx})
		var singleIterationCount int

		// every batch needs a lock so we avoid entries changing accessidx in the meantime
		s.lock.Lock()
		for ; ok && (singleIterationCount < s.gc.maxBatch); ok = it.Next() {

			// quit if no more access index keys
			itkey := it.Key()
			if (itkey == nil) || (itkey[0] != keyGCIdx) {
				break
			}

			// get chunk data entry from access index
			val := it.Value()
			index, po, hash := parseGCIdxEntry(itkey[1:], val)
			keyIdx := make([]byte, 33)
			keyIdx[0] = keyIndex
			copy(keyIdx[1:], hash)

			// add delete operation to batch
			s.delete(s.gc.batch.Batch, index, keyIdx, po)
			singleIterationCount++
			s.gc.count++
			log.Trace("garbage collect enqueued chunk for deletion", "key", hash)

			// break if target is not on max garbage batch boundary
			if s.gc.count >= s.gc.target {
				break
			}
		}

		s.writeBatch(s.gc.batch, wEntryCnt)
		s.lock.Unlock()
		it.Release()
		log.Trace("garbage collect batch done", "batch", singleIterationCount, "total", s.gc.count)
	}

	s.gc.runC <- struct{}{}
	log.Debug("garbage collect done", "c", s.gc.count)

	metrics.GetOrRegisterCounter("ldbstore.collectgarbage.delete", nil).Inc(int64(totalDeleted))
	return nil
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
			log.Warn(fmt.Sprintf("Chunk %x found but could not be accessed: %v", key, err))
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	countC := make(chan int64)
	errC := make(chan error)
	var count int64
	go func() {
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				select {
				case errC <- err:
				case <-ctx.Done():
				}
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
				select {
				case errC <- err:
				case <-ctx.Done():
				}
			}
			key := Address(keybytes)
			chunk := NewChunk(key, data[32:])

			go func() {
				select {
				case errC <- s.Put(ctx, chunk):
				case <-ctx.Done():
				}
			}()

			count++
		}
		countC <- count
	}()

	// wait for all chunks to be stored
	i := int64(0)
	var total int64
	for {
		select {
		case err := <-errC:
			if err != nil {
				return count, err
			}
			i++
		case total = <-countC:
		case <-ctx.Done():
			return i, ctx.Err()
		}
		if total > 0 && i == total {
			return total, nil
		}
	}
}

// Cleanup iterates over the database and deletes chunks if they pass the `f` condition
func (s *LDBStore) Cleanup(f func(*chunk) bool) {
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
				log.Warn(fmt.Sprintf("Chunk %x found but count not be accessed with any po", key))
				errorsFound++
				continue
			}
		}

		ck := data[:32]
		c, err := decodeData(ck, data)
		if err != nil {
			log.Error("decodeData error", "err", err)
			continue
		}

		cs := int64(binary.LittleEndian.Uint64(c.sdata[:8]))
		log.Trace("chunk", "key", fmt.Sprintf("%x", key), "ck", fmt.Sprintf("%x", ck), "dkey", fmt.Sprintf("%x", datakey), "dataidx", index.Idx, "po", po, "len data", len(data), "len sdata", len(c.sdata), "size", cs)

		// if chunk is to be removed
		if f(c) {
			log.Warn("chunk for cleanup", "key", fmt.Sprintf("%x", key), "ck", fmt.Sprintf("%x", ck), "dkey", fmt.Sprintf("%x", datakey), "dataidx", index.Idx, "po", po, "len data", len(data), "len sdata", len(c.sdata), "size", cs)
			s.deleteNow(&index, getIndexKey(key[1:]), po)
			removed++
			errorsFound++
		}
	}

	log.Warn(fmt.Sprintf("Found %v errors out of %v entries. Removed %v chunks.", errorsFound, total, removed))
}

// CleanGCIndex rebuilds the garbage collector index from scratch, while
// removing inconsistent elements, e.g., indices with missing data chunks.
// WARN: it's a pretty heavy, long running function.
func (s *LDBStore) CleanGCIndex() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	batch := leveldb.Batch{}

	var okEntryCount uint64
	var totalEntryCount uint64

	// throw out all gc indices, we will rebuild from cleaned index
	it := s.db.NewIterator()
	it.Seek([]byte{keyGCIdx})
	var gcDeletes int
	for it.Valid() {
		rowType, _ := parseIdxKey(it.Key())
		if rowType != keyGCIdx {
			break
		}
		batch.Delete(it.Key())
		gcDeletes++
		it.Next()
	}
	log.Debug("gc", "deletes", gcDeletes)
	if err := s.db.Write(&batch); err != nil {
		return err
	}
	batch.Reset()

	it.Release()

	// corrected po index pointer values
	var poPtrs [256]uint64

	// set to true if chunk count not on 4096 iteration boundary
	var doneIterating bool

	// last key index in previous iteration
	lastIdxKey := []byte{keyIndex}

	// counter for debug output
	var cleanBatchCount int

	// go through all key index entries
	for !doneIterating {
		cleanBatchCount++
		var idxs []dpaDBIndex
		var chunkHashes [][]byte
		var pos []uint8
		it := s.db.NewIterator()

		it.Seek(lastIdxKey)

		// 4096 is just a nice number, don't look for any hidden meaning here...
		var i int
		for i = 0; i < 4096; i++ {

			// this really shouldn't happen unless database is empty
			// but let's keep it to be safe
			if !it.Valid() {
				doneIterating = true
				break
			}

			// if it's not keyindex anymore we're done iterating
			rowType, chunkHash := parseIdxKey(it.Key())
			if rowType != keyIndex {
				doneIterating = true
				break
			}

			// decode the retrieved index
			var idx dpaDBIndex
			err := decodeIndex(it.Value(), &idx)
			if err != nil {
				return fmt.Errorf("corrupt index: %v", err)
			}
			po := s.po(chunkHash)
			lastIdxKey = it.Key()

			// if we don't find the data key, remove the entry
			// if we find it, add to the array of new gc indices to create
			dataKey := getDataKey(idx.Idx, po)
			_, err = s.db.Get(dataKey)
			if err != nil {
				log.Warn("deleting inconsistent index (missing data)", "key", chunkHash)
				batch.Delete(it.Key())
			} else {
				idxs = append(idxs, idx)
				chunkHashes = append(chunkHashes, chunkHash)
				pos = append(pos, po)
				okEntryCount++
				if idx.Idx > poPtrs[po] {
					poPtrs[po] = idx.Idx
				}
			}
			totalEntryCount++
			it.Next()
		}
		it.Release()

		// flush the key index corrections
		err := s.db.Write(&batch)
		if err != nil {
			return err
		}
		batch.Reset()

		// add correct gc indices
		for i, okIdx := range idxs {
			gcIdxKey := getGCIdxKey(&okIdx)
			gcIdxData := getGCIdxValue(&okIdx, pos[i], chunkHashes[i])
			batch.Put(gcIdxKey, gcIdxData)
			log.Trace("clean ok", "key", chunkHashes[i], "gcKey", gcIdxKey, "gcData", gcIdxData)
		}

		// flush them
		err = s.db.Write(&batch)
		if err != nil {
			return err
		}
		batch.Reset()

		log.Debug("clean gc index pass", "batch", cleanBatchCount, "checked", i, "kept", len(idxs))
	}

	log.Debug("gc cleanup entries", "ok", okEntryCount, "total", totalEntryCount, "batchlen", batch.Len())

	// lastly add updated entry count
	var entryCount [8]byte
	binary.BigEndian.PutUint64(entryCount[:], okEntryCount)
	batch.Put(keyEntryCnt, entryCount[:])

	// and add the new po index pointers
	var poKey [2]byte
	poKey[0] = keyDistanceCnt
	for i, poPtr := range poPtrs {
		poKey[1] = uint8(i)
		if poPtr == 0 {
			batch.Delete(poKey[:])
		} else {
			var idxCount [8]byte
			binary.BigEndian.PutUint64(idxCount[:], poPtr)
			batch.Put(poKey[:], idxCount[:])
		}
	}

	// if you made it this far your harddisk has survived. Congratulations
	return s.db.Write(&batch)
}

// Delete is removes a chunk and updates indices.
// Is thread safe
func (s *LDBStore) Delete(addr Address) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	ikey := getIndexKey(addr)

	idata, err := s.db.Get(ikey)
	if err != nil {
		return err
	}

	var idx dpaDBIndex
	decodeIndex(idata, &idx)
	proximity := s.po(addr)
	return s.deleteNow(&idx, ikey, proximity)
}

// executes one delete operation immediately
// see *LDBStore.delete
func (s *LDBStore) deleteNow(idx *dpaDBIndex, idxKey []byte, po uint8) error {
	batch := new(leveldb.Batch)
	s.delete(batch, idx, idxKey, po)
	return s.db.Write(batch)
}

// adds a delete chunk operation to the provided batch
// if called directly, decrements entrycount regardless if the chunk exists upon deletion. Risk of wrap to max uint64
func (s *LDBStore) delete(batch *leveldb.Batch, idx *dpaDBIndex, idxKey []byte, po uint8) {
	metrics.GetOrRegisterCounter("ldbstore.delete", nil).Inc(1)

	gcIdxKey := getGCIdxKey(idx)
	batch.Delete(gcIdxKey)
	dataKey := getDataKey(idx.Idx, po)
	batch.Delete(dataKey)
	batch.Delete(idxKey)
	s.entryCnt--
	dbEntryCount.Dec(1)
	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	batch.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))
}

func (s *LDBStore) BinIndex(po uint8) uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.bucketCnt[po]
}

// Put adds a chunk to the database, adding indices and incrementing global counters.
// If it already exists, it merely increments the access count of the existing entry.
// Is thread safe
func (s *LDBStore) Put(ctx context.Context, chunk Chunk) error {
	metrics.GetOrRegisterCounter("ldbstore.put", nil).Inc(1)
	log.Trace("ldbstore.put", "key", chunk.Address())

	ikey := getIndexKey(chunk.Address())
	var index dpaDBIndex

	po := s.po(chunk.Address())

	s.lock.Lock()

	if s.closed {
		s.lock.Unlock()
		return ErrDBClosed
	}
	batch := s.batch

	log.Trace("ldbstore.put: s.db.Get", "key", chunk.Address(), "ikey", fmt.Sprintf("%x", ikey))
	_, err := s.db.Get(ikey)
	if err != nil {
		s.doPut(chunk, &index, po)
	}
	idata := encodeIndex(&index)
	s.batch.Put(ikey, idata)

	// add the access-chunkindex index for garbage collection
	gcIdxKey := getGCIdxKey(&index)
	gcIdxData := getGCIdxValue(&index, po, chunk.Address())
	s.batch.Put(gcIdxKey, gcIdxData)
	s.lock.Unlock()

	select {
	case s.batchesC <- struct{}{}:
	default:
	}

	select {
	case <-batch.c:
		return batch.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// force putting into db, does not check or update necessary indices
func (s *LDBStore) doPut(chunk Chunk, index *dpaDBIndex, po uint8) {
	data := s.encodeDataFunc(chunk)
	dkey := getDataKey(s.dataIdx, po)
	s.batch.Put(dkey, data)
	index.Idx = s.dataIdx
	s.bucketCnt[po] = s.dataIdx
	s.entryCnt++
	dbEntryCount.Inc(1)
	s.dataIdx++
	index.Access = s.accessCnt
	s.accessCnt++
	cntKey := make([]byte, 2)
	cntKey[0] = keyDistanceCnt
	cntKey[1] = po
	s.batch.Put(cntKey, U64ToBytes(s.bucketCnt[po]))
}

func (s *LDBStore) writeBatches() {
	for {
		select {
		case <-s.quit:
			log.Debug("DbStore: quit batch write loop")
			return
		case <-s.batchesC:
			err := s.writeCurrentBatch()
			if err != nil {
				log.Debug("DbStore: quit batch write loop", "err", err.Error())
				return
			}
		}
	}

}

func (s *LDBStore) writeCurrentBatch() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	b := s.batch
	l := b.Len()
	if l == 0 {
		return nil
	}
	s.batch = newBatch()
	b.err = s.writeBatch(b, wEntryCnt|wAccessCnt|wIndexCnt)
	close(b.c)
	if s.entryCnt >= s.capacity {
		go s.collectGarbage()
	}
	return nil
}

// must be called non concurrently
func (s *LDBStore) writeBatch(b *dbBatch, wFlag uint8) error {
	if wFlag&wEntryCnt > 0 {
		b.Put(keyEntryCnt, U64ToBytes(s.entryCnt))
	}
	if wFlag&wIndexCnt > 0 {
		b.Put(keyDataIdx, U64ToBytes(s.dataIdx))
	}
	if wFlag&wAccessCnt > 0 {
		b.Put(keyAccessCnt, U64ToBytes(s.accessCnt))
	}
	l := b.Len()
	if err := s.db.Write(b.Batch); err != nil {
		return fmt.Errorf("unable to write batch: %v", err)
	}
	log.Trace(fmt.Sprintf("batch write (%d entries)", l))
	return nil
}

// newMockEncodeDataFunc returns a function that stores the chunk data
// to a mock store to bypass the default functionality encodeData.
// The constructed function always returns the nil data, as DbStore does
// not need to store the data, but still need to create the index.
func newMockEncodeDataFunc(mockStore *mock.NodeStore) func(chunk Chunk) []byte {
	return func(chunk Chunk) []byte {
		if err := mockStore.Put(chunk.Address(), encodeData(chunk)); err != nil {
			log.Error(fmt.Sprintf("%T: Chunk %v put: %v", mockStore, chunk.Address().Log(), err))
		}
		return chunk.Address()[:]
	}
}

// tryAccessIdx tries to find index entry. If found then increments the access
// count for garbage collection and returns the index entry and true for found,
// otherwise returns nil and false.
func (s *LDBStore) tryAccessIdx(addr Address, po uint8) (*dpaDBIndex, bool) {
	ikey := getIndexKey(addr)
	idata, err := s.db.Get(ikey)
	if err != nil {
		return nil, false
	}

	index := new(dpaDBIndex)
	decodeIndex(idata, index)
	oldGCIdxKey := getGCIdxKey(index)
	s.batch.Put(keyAccessCnt, U64ToBytes(s.accessCnt))
	index.Access = s.accessCnt
	idata = encodeIndex(index)
	s.accessCnt++
	s.batch.Put(ikey, idata)
	newGCIdxKey := getGCIdxKey(index)
	newGCIdxData := getGCIdxValue(index, po, ikey[1:])
	s.batch.Delete(oldGCIdxKey)
	s.batch.Put(newGCIdxKey, newGCIdxData)
	select {
	case s.batchesC <- struct{}{}:
	default:
	}
	return index, true
}

// GetSchema is returning the current named schema of the datastore as read from LevelDB
func (s *LDBStore) GetSchema() (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := s.db.Get(keySchema)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return DbSchemaNone, nil
		}
		return "", err
	}

	return string(data), nil
}

// PutSchema is saving a named schema to the LevelDB datastore
func (s *LDBStore) PutSchema(schema string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.db.Put(keySchema, []byte(schema))
}

// Get retrieves the chunk matching the provided key from the database.
// If the chunk entry does not exist, it returns an error
// Updates access count and is thread safe
func (s *LDBStore) Get(_ context.Context, addr Address) (chunk Chunk, err error) {
	metrics.GetOrRegisterCounter("ldbstore.get", nil).Inc(1)
	log.Trace("ldbstore.get", "key", addr)

	s.lock.Lock()
	defer s.lock.Unlock()
	return s.get(addr)
}

// TODO: To conform with other private methods of this object indices should not be updated
func (s *LDBStore) get(addr Address) (chunk *chunk, err error) {
	if s.closed {
		return nil, ErrDBClosed
	}
	proximity := s.po(addr)
	index, found := s.tryAccessIdx(addr, proximity)
	if found {
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
			datakey := getDataKey(index.Idx, proximity)
			data, err = s.db.Get(datakey)
			log.Trace("ldbstore.get retrieve", "key", addr, "indexkey", index.Idx, "datakey", fmt.Sprintf("%x", datakey), "proximity", proximity)
			if err != nil {
				log.Trace("ldbstore.get chunk found but could not be accessed", "key", addr, "err", err)
				s.deleteNow(index, getIndexKey(addr), s.po(addr))
				return
			}
		}

		return decodeData(addr, data)
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

func (s *LDBStore) setCapacity(c uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.capacity = c

	for s.entryCnt > c {
		s.collectGarbage()
	}
}

func (s *LDBStore) Close() {
	close(s.quit)
	s.lock.Lock()
	s.closed = true
	s.lock.Unlock()
	// force writing out current batch
	s.writeCurrentBatch()
	close(s.batchesC)
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
