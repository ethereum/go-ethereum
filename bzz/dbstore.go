// disk storage layer for the package blockhash
// inefficient work-in-progress version

package bzz

import (
	//	"crypto/sha256"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/ethdb"
	//	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	//	"path"
)

const dbMaxEntries = 5000 // max number of stored (cached) blocks

const gcArraySize = 500
const gcArrayFreeRatio = 10

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

type dpaDBStorage struct {
	dpaStorage
	db *ethdb.LDBDatabase

	// this should be stored in db, accessed transactionally
	entryCnt, accessCnt, dataIdx uint64

	gcPos, gcStartPos []byte
	gcArray           []*gcItem
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

func (s *dpaDBStorage) updateIndexAccess(index *dpaDBIndex) {

	index.Access = s.accessCnt

}

func getIndexKey(hash HashType) []byte {

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

func encodeData(entry *dpaNode) []byte {

	var rlpEntry struct {
		Data []byte
		Size uint64
	}

	rlpEntry.Data = entry.data
	rlpEntry.Size = uint64(entry.size)

	data, _ := rlp.EncodeToBytes(rlpEntry)
	return data

}

func decodeIndex(data []byte, index *dpaDBIndex) {

	dec := rlp.NewStream(bytes.NewReader(data))
	dec.Decode(index)

}

func decodeData(data []byte, entry *dpaNode) {

	var rlpEntry struct {
		Data []byte
		Size uint64
	}

	dec := rlp.NewStream(bytes.NewReader(data))
	dec.Decode(&rlpEntry)
	entry.data = rlpEntry.Data
	entry.size = int64(rlpEntry.Size)
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

func (s *dpaDBStorage) collectGarbage() {

	it := s.db.NewIterator()
	it.Seek(s.gcPos)
	if it.Valid() {
		s.gcPos = it.Key()
	} else {
		s.gcPos = nil
	}
	gcnt := 0

	for gcnt < gcArraySize {

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

	cutidx := gcListSelect(s.gcArray, 0, gcnt-1, gcnt/gcArrayFreeRatio)
	cutval := s.gcArray[cutidx].value

	//fmt.Print(s.entryCnt, " ")

	// actual gc
	for i := 0; i < gcnt; i++ {
		if s.gcArray[i].value < cutval {
			batch := new(leveldb.Batch)
			batch.Delete(s.gcArray[i].idxKey)
			batch.Delete(getDataKey(s.gcArray[i].idx))
			s.entryCnt--
			batch.Put(keyEntryCnt, u64ToBytes(s.entryCnt))
			s.db.Write(batch)
		}
	}

	//fmt.Println(s.entryCnt)

	s.db.Put(keyGCPos, s.gcPos)

}

func (s *dpaDBStorage) add(entry *dpaStoreReq) {

	ikey := getIndexKey(entry.hash)
	var index dpaDBIndex

	if s.tryAccessIdx(ikey, &index) {
		return // already exists, only update access
	}

	data := encodeData(&entry.dpaNode)
	//data := ethutil.Encode([]interface{}{entry})

	if s.entryCnt >= dbMaxEntries {
		s.collectGarbage()
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

}

// try to find index; if found, update access cnt and return true
func (s *dpaDBStorage) tryAccessIdx(ikey []byte, index *dpaDBIndex) bool {

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

func (s *dpaDBStorage) find(hash HashType) (entry dpaNode) {

	key := getIndexKey(hash)
	var index dpaDBIndex

	if s.tryAccessIdx(key, &index) {
		data, _ := s.db.Get(getDataKey(index.Idx))
		decodeData(data, &entry)
	}

	return

}

func (s *dpaDBStorage) process_store(req *dpaStoreReq) {

	s.add(req)

	if s.chain != nil {
		s.chain.store_chn <- req
	}

}

func (s *dpaDBStorage) process_retrieve(req *dpaRetrieveReq) {

	if req.result_chn == nil {

		key := getIndexKey(req.hash)
		var index dpaDBIndex
		s.tryAccessIdx(key, &index) // result_chn == nil, only update access cnt

		return
	}

	entry := s.find(req.hash)

	if entry.data == nil {
		if s.chain != nil {
			s.chain.retrieve_chn <- req
			return
		}
	}

	res := new(dpaRetrieveRes)
	if entry.data != nil {
		res.dpaNode = entry
	}
	res.req_id = req.req_id
	req.result_chn <- res

}

func (s *dpaDBStorage) Init(ch *dpaStorage) {

	s.dpaStorage.Init()

	var err error
	s.db, err = ethdb.NewLDBDatabase("/tmp/bzz")
	if err != nil {
		fmt.Println("/tmp/bzz error:")
		fmt.Println(err)
	}
	if s.db == nil {
		fmt.Println("LDBDatabase is nil")
	}

	s.gcStartPos = make([]byte, HashSize+1)
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

	//	fmt.Println(s.entryCnt)
	//	fmt.Println(s.accessCnt)
	//	fmt.Println(s.dataIdx)

}

func (s *dpaDBStorage) Run() {

	for {
		bb := true
		for bb {
			select {
			case store := <-s.store_chn:
				s.process_store(store)
			default:
				bb = false
			}
		}
		select {
		case store := <-s.store_chn:
			s.process_store(store)
		case retrv := <-s.retrieve_chn:
			s.process_retrieve(retrv)
		}
	}

}
