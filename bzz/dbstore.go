// disk storage layer for the package bzz
// dbStore implements the ChunkStore interface and is used by the DPA as
// persistent storage of chunks
// it implements purging based on access count allowing for external control of
// max capacity

package bzz

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
)

const gcArraySize = 10000
const gcArrayFreeRatio = 0.1

// key prefixes for leveldb storage
const kpIndex = 0
const kpData = 1

var keyAccessCnt = []byte{2}
var keyEntryCnt = []byte{3}
var keyDataIdx = []byte{4}
var keyGCPos = []byte{5}

type gcItem struct {
	idx    uint64
	value  uint64
	idxKey []byte
}

type dbStore struct {
	db *LDBDatabase

	// this should be stored in db, accessed transactionally
	entryCnt, accessCnt, dataIdx, capacity uint64

	gcPos, gcStartPos []byte
	gcArray           []*gcItem

	lock sync.Mutex
}

type dpaDBIndex struct {
	Idx    uint64
	Access uint64
}

func bytesToU64(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(data)
}

func u64ToBytes(val uint64) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, val)
	return data
}

func getIndexGCValue(index *dpaDBIndex) uint64 {
	return index.Access
}

func (s *dbStore) updateIndexAccess(index *dpaDBIndex) {
	index.Access = s.accessCnt
}

func getIndexKey(hash Key) []byte {
	HashSize := len(hash)
	key := make([]byte, HashSize+1)
	key[0] = 0
	// db keys derived from hash:
	// two halves swapped for uniformly distributed prefix
	copy(key[1:HashSize/2+1], hash[HashSize/2:HashSize])
	copy(key[HashSize/2+1:HashSize+1], hash[0:HashSize/2])

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

func (s *dbStore) collectGarbage(ratio float32) {
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

	cutidx := gcListSelect(s.gcArray, 0, gcnt-1, int(float32(gcnt)*ratio))
	cutval := s.gcArray[cutidx].value

	// fmt.Print(gcnt, " ", s.entryCnt, " ")

	// actual gc
	for i := 0; i < gcnt; i++ {
		if s.gcArray[i].value <= cutval {
			batch := new(leveldb.Batch)
			batch.Delete(s.gcArray[i].idxKey)
			batch.Delete(getDataKey(s.gcArray[i].idx))
			s.entryCnt--
			batch.Put(keyEntryCnt, u64ToBytes(s.entryCnt))
			s.db.Write(batch)
		}
	}

	// fmt.Println(s.entryCnt)

	s.db.Put(keyGCPos, s.gcPos)
}

func (s *dbStore) Put(chunk *Chunk) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ikey := getIndexKey(chunk.Key)
	var index dpaDBIndex

	if s.tryAccessIdx(ikey, &index) {
		if chunk.dbStored != nil {
			close(chunk.dbStored)
		}
		return // already exists, only update access
	}

	data := encodeData(chunk)
	//data := ethutil.Encode([]interface{}{entry})

	if s.entryCnt >= s.capacity {
		s.collectGarbage(gcArrayFreeRatio)
	}

	batch := new(leveldb.Batch)

	s.entryCnt++
	batch.Put(keyEntryCnt, u64ToBytes(s.entryCnt))
	s.dataIdx++
	batch.Put(keyDataIdx, u64ToBytes(s.dataIdx))
	s.accessCnt++
	batch.Put(keyAccessCnt, u64ToBytes(s.accessCnt))

	batch.Put(getDataKey(s.dataIdx), data)

	index.Idx = s.dataIdx
	s.updateIndexAccess(&index)

	idata := encodeIndex(&index)
	batch.Put(ikey, idata)

	s.db.Write(batch)
	if chunk.dbStored != nil {
		close(chunk.dbStored)
	}
}

// try to find index; if found, update access cnt and return true
func (s *dbStore) tryAccessIdx(ikey []byte, index *dpaDBIndex) bool {
	idata, err := s.db.Get(ikey)
	if err != nil {
		return false
	}
	decodeIndex(idata, index)

	batch := new(leveldb.Batch)

	s.accessCnt++
	batch.Put(keyAccessCnt, u64ToBytes(s.accessCnt))
	s.updateIndexAccess(index)
	idata = encodeIndex(index)
	batch.Put(ikey, idata)

	s.db.Write(batch)

	return true
}

func (s *dbStore) Get(key Key) (chunk *Chunk, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var index dpaDBIndex

	if s.tryAccessIdx(getIndexKey(key), &index) {
		var data []byte
		data, err = s.db.Get(getDataKey(index.Idx))
		if err != nil {
			return
		}

		hasher := hasherfunc.New()
		hasher.Write(data)
		if bytes.Compare(hasher.Sum(nil), key) != 0 {
			s.db.Delete(getDataKey(index.Idx))
			err = notFound
			return
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

func (s *dbStore) updateAccessCnt(key Key) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var index dpaDBIndex
	s.tryAccessIdx(getIndexKey(key), &index) // result_chn == nil, only update access cnt

}

func (s *dbStore) setCapacity(c uint64) {

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

func (s *dbStore) getEntryCnt() uint64 {
	return s.entryCnt
}

func newDbStore(path string) (s *dbStore, err error) {
	s = new(dbStore)

	s.db, err = NewLDBDatabase(path)
	if err != nil {
		return
	}

	s.setCapacity(5000000) // TODO define default capacity as constant

	s.gcStartPos = make([]byte, 1)
	s.gcStartPos[0] = kpIndex
	s.gcArray = make([]*gcItem, gcArraySize)

	data, _ := s.db.Get(keyEntryCnt)
	s.entryCnt = bytesToU64(data)
	data, _ = s.db.Get(keyAccessCnt)
	s.accessCnt = bytesToU64(data)
	data, _ = s.db.Get(keyDataIdx)
	s.dataIdx = bytesToU64(data)
	s.gcPos, _ = s.db.Get(keyGCPos)
	if s.gcPos == nil {
		s.gcPos = s.gcStartPos
	}
	return
}

func (s *dbStore) close() {
	s.db.Close()
}
