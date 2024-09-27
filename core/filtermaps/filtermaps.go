package filtermaps

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

const (
	logMapHeight    = 12                   // log2(mapHeight)
	mapHeight       = 1 << logMapHeight    // filter map height (number of rows)
	logMapsPerEpoch = 6                    // log2(mmapsPerEpochapsPerEpoch)
	mapsPerEpoch    = 1 << logMapsPerEpoch // number of maps in an epoch
	logValuesPerMap = 16                   // log2(logValuesPerMap)
	valuesPerMap    = 1 << logValuesPerMap // number of log values marked on each filter map

	headCacheSize = 8 // maximum number of recent filter maps cached in memory
)

// blockchain defines functions required by the FilterMaps log indexer.
type blockchain interface {
	CurrentBlock() *types.Header
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetCanonicalHash(number uint64) common.Hash
}

// FilterMaps is the in-memory representation of the log index structure that is
// responsible for building and updating the index according to the canonical
// chain.
// Note that FilterMaps implements the same data structure as proposed in EIP-7745
// without the tree hashing and consensus changes:
// https://eips.ethereum.org/EIPS/eip-7745
type FilterMaps struct {
	lock    sync.RWMutex
	db      ethdb.Database
	closeCh chan struct{}
	closeWg sync.WaitGroup
	filterMapsRange
	chain         blockchain
	matcherSyncCh chan *FilterMapsMatcherBackend
	matchers      map[*FilterMapsMatcherBackend]struct{}
	// filterMapCache caches certain filter maps (headCacheSize most recent maps
	// and one tail map) that are expected to be frequently accessed and modified
	// while updating the structure. Note that the set of cached maps depends
	// only on filterMapsRange and rows of other maps are not cached here.
	filterMapLock  sync.Mutex
	filterMapCache map[uint32]*filterMap
	blockPtrCache  *lru.Cache[uint32, uint64]
	lvPointerCache *lru.Cache[uint64, uint64]
	revertPoints   map[uint64]*revertPoint
}

// filterMap is a full or partial in-memory representation of a filter map where
// rows are allowed to have a nil value meaning the row is not stored in the
// structure. Note that therefore a known empty row should be represented with
// a zero-length slice.
// It can be used as a memory cache or an overlay while preparing a batch of
// changes to the structure. In either case a nil value should be interpreted
// as transparent (uncached/unchanged).
type filterMap [mapHeight]FilterRow

// FilterRow encodes a single row of a filter map as a list of column indices.
// Note that the values are always stored in the same order as they were added
// and if the same column index is added twice, it is also stored twice.
// Order of column indices and potential duplications do not matter when searching
// for a value but leaving the original order makes reverting to a previous state
// simpler.
type FilterRow []uint32

// emptyRow represents an empty FilterRow. Note that in case of decoded FilterRows
// nil has a special meaning (transparent; not stored in the cache/overlay map)
// and therefore an empty row is represented by a zero length slice.
var emptyRow = FilterRow{}

// filterMapsRange describes the block range that has been indexed and the log
// value index range it has been mapped to.
type filterMapsRange struct {
	initialized                      bool
	headLvPointer, tailLvPointer     uint64
	headBlockNumber, tailBlockNumber uint64
	headBlockHash, tailParentHash    common.Hash
}

// NewFilterMaps creates a new FilterMaps and starts the indexer in order to keep
// the structure in sync with the given blockchain.
func NewFilterMaps(db ethdb.Database, chain blockchain) *FilterMaps {
	rs, err := rawdb.ReadFilterMapsRange(db)
	if err != nil {
		log.Error("Error reading log index range", "error", err)
	}
	fm := &FilterMaps{
		db:      db,
		chain:   chain,
		closeCh: make(chan struct{}),
		filterMapsRange: filterMapsRange{
			initialized:     rs.Initialized,
			headLvPointer:   rs.HeadLvPointer,
			tailLvPointer:   rs.TailLvPointer,
			headBlockNumber: rs.HeadBlockNumber,
			tailBlockNumber: rs.TailBlockNumber,
			headBlockHash:   rs.HeadBlockHash,
			tailParentHash:  rs.TailParentHash,
		},
		matcherSyncCh:  make(chan *FilterMapsMatcherBackend),
		matchers:       make(map[*FilterMapsMatcherBackend]struct{}),
		filterMapCache: make(map[uint32]*filterMap),
		blockPtrCache:  lru.NewCache[uint32, uint64](1000),
		lvPointerCache: lru.NewCache[uint64, uint64](1000),
		revertPoints:   make(map[uint64]*revertPoint),
	}
	fm.closeWg.Add(2)
	go fm.removeBloomBits()
	go fm.updateLoop()
	return fm
}

// Close ensures that the indexer is fully stopped before returning.
func (f *FilterMaps) Close() {
	close(f.closeCh)
	f.closeWg.Wait()
}

// reset un-initializes the FilterMaps structure and removes all related data from
// the database. The function returns true if everything was successfully removed.
func (f *FilterMaps) reset() bool {
	f.lock.Lock()
	f.filterMapsRange = filterMapsRange{}
	f.filterMapCache = make(map[uint32]*filterMap)
	f.revertPoints = make(map[uint64]*revertPoint)
	f.blockPtrCache.Purge()
	f.lvPointerCache.Purge()
	f.lock.Unlock()
	// deleting the range first ensures that resetDb will be called again at next
	// startup and any leftover data will be removed even if it cannot finish now.
	rawdb.DeleteFilterMapsRange(f.db)
	return f.removeDbWithPrefix(rawdb.FilterMapsPrefix, "Resetting log index database")
}

// removeBloomBits removes old bloom bits data from the database.
func (f *FilterMaps) removeBloomBits() {
	f.removeDbWithPrefix(rawdb.BloomBitsPrefix, "Removing old bloom bits database")
	f.removeDbWithPrefix(rawdb.BloomBitsIndexPrefix, "Removing old bloom bits chain index")
	f.closeWg.Done()
}

// removeDbWithPrefix removes data with the given prefix from the database and
// returns true if everything was successfully removed.
func (f *FilterMaps) removeDbWithPrefix(prefix []byte, action string) bool {
	var (
		logged     bool
		lastLogged time.Time
		removed    uint64
	)
	for {
		select {
		case <-f.closeCh:
			return false
		default:
		}
		it := f.db.NewIterator(prefix, nil)
		batch := f.db.NewBatch()
		var count int
		for ; count < 10000 && it.Next(); count++ {
			batch.Delete(it.Key())
			removed++
		}
		it.Release()
		if count == 0 {
			break
		}
		if !logged {
			log.Info(action + "...")
			logged = true
			lastLogged = time.Now()
		}
		if time.Since(lastLogged) >= time.Second*10 {
			log.Info(action+" in progress", "removed keys", removed)
			lastLogged = time.Now()
		}
		batch.Write()
	}
	if logged {
		log.Info(action + " finished")
	}
	return true
}

// setRange updates the covered range and also adds the changes to the given batch.
// Note that this function assumes that the read/write lock is being held.
func (f *FilterMaps) setRange(batch ethdb.Batch, newRange filterMapsRange) {
	f.filterMapsRange = newRange
	rs := rawdb.FilterMapsRange{
		Initialized:     newRange.initialized,
		HeadLvPointer:   newRange.headLvPointer,
		TailLvPointer:   newRange.tailLvPointer,
		HeadBlockNumber: newRange.headBlockNumber,
		TailBlockNumber: newRange.tailBlockNumber,
		HeadBlockHash:   newRange.headBlockHash,
		TailParentHash:  newRange.tailParentHash,
	}
	rawdb.WriteFilterMapsRange(batch, rs)
	f.updateMapCache()
	f.updateMatchersValidRange()
}

// updateMapCache updates the maps covered by the filterMapCache according to the
// covered range.
// Note that this function assumes that the read lock is being held.
func (f *FilterMaps) updateMapCache() {
	if !f.initialized {
		return
	}
	f.filterMapLock.Lock()
	defer f.filterMapLock.Unlock()

	newFilterMapCache := make(map[uint32]*filterMap)
	firstMap, afterLastMap := uint32(f.tailLvPointer>>logValuesPerMap), uint32((f.headLvPointer+valuesPerMap-1)>>logValuesPerMap)
	headCacheFirst := firstMap + 1
	if afterLastMap > headCacheFirst+headCacheSize {
		headCacheFirst = afterLastMap - headCacheSize
	}
	fm := f.filterMapCache[firstMap]
	if fm == nil {
		fm = new(filterMap)
	}
	newFilterMapCache[firstMap] = fm
	for mapIndex := headCacheFirst; mapIndex < afterLastMap; mapIndex++ {
		fm := f.filterMapCache[mapIndex]
		if fm == nil {
			fm = new(filterMap)
		}
		newFilterMapCache[mapIndex] = fm
	}
	f.filterMapCache = newFilterMapCache
}

// getLogByLvIndex returns the log at the given log value index. If the index does
// not point to the first log value entry of a log then no log and no error are
// returned as this can happen when the log value index was a false positive.
// Note that this function assumes that the log index structure is consistent
// with the canonical chain at the point where the given log value index points.
// If this is not the case then an invalid result or an error may be returned.
// Note that this function assumes that the read lock is being held.
func (f *FilterMaps) getLogByLvIndex(lvIndex uint64) (*types.Log, error) {
	if lvIndex < f.tailLvPointer || lvIndex > f.headLvPointer {
		return nil, nil
	}
	// find possible block range based on map to block pointers
	mapIndex := uint32(lvIndex >> logValuesPerMap)
	firstBlockNumber, err := f.getMapBlockPtr(mapIndex)
	if err != nil {
		return nil, err
	}
	var lastBlockNumber uint64
	if mapIndex+1 < uint32((f.headLvPointer+valuesPerMap-1)>>logValuesPerMap) {
		lastBlockNumber, err = f.getMapBlockPtr(mapIndex + 1)
		if err != nil {
			return nil, err
		}
	} else {
		lastBlockNumber = f.headBlockNumber
	}
	// find block with binary search based on block to log value index pointers
	for firstBlockNumber < lastBlockNumber {
		midBlockNumber := (firstBlockNumber + lastBlockNumber + 1) / 2
		midLvPointer, err := f.getBlockLvPointer(midBlockNumber)
		if err != nil {
			return nil, err
		}
		if lvIndex < midLvPointer {
			lastBlockNumber = midBlockNumber - 1
		} else {
			firstBlockNumber = midBlockNumber
		}
	}
	// get block receipts
	hash := f.chain.GetCanonicalHash(firstBlockNumber)
	receipts := rawdb.ReadRawReceipts(f.db, hash, firstBlockNumber) //TODO small cache
	if receipts == nil {
		return nil, errors.New("receipts not found")
	}
	lvPointer, err := f.getBlockLvPointer(firstBlockNumber)
	if err != nil {
		return nil, err
	}
	// iterate through receipts to find the exact log starting at lvIndex
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			if lvPointer > lvIndex {
				// lvIndex does not point to the first log value (address value)
				// generated by a log as true matches should always do, so it
				// is considered a false positive (no log and no error returned).
				return nil, nil
			}
			if lvPointer == lvIndex {
				return log, nil // potential match
			}
			lvPointer += uint64(len(log.Topics) + 1)
		}
	}
	return nil, nil
}

// getFilterMapRow returns the given row of the given map. If the row is empty
// then a non-nil zero length row is returned.
// Note that the returned slices should not be modified, they should be copied
// on write.
func (f *FilterMaps) getFilterMapRow(mapIndex, rowIndex uint32) (FilterRow, error) {
	f.filterMapLock.Lock()
	defer f.filterMapLock.Unlock()

	fm := f.filterMapCache[mapIndex]
	if fm != nil && fm[rowIndex] != nil {
		return fm[rowIndex], nil
	}
	row, err := rawdb.ReadFilterMapRow(f.db, mapRowIndex(mapIndex, rowIndex))
	if err != nil {
		return nil, err
	}
	if fm != nil {
		fm[rowIndex] = FilterRow(row)
	}
	return FilterRow(row), nil
}

// storeFilterMapRow stores a row at the given row index of the given map and also
// caches it in filterMapCache if the given map is cached.
// Note that empty rows are not stored in the database and therefore there is no
// separate delete function; deleting a row is the same as storing an empty row.
func (f *FilterMaps) storeFilterMapRow(batch ethdb.Batch, mapIndex, rowIndex uint32, row FilterRow) {
	f.filterMapLock.Lock()
	defer f.filterMapLock.Unlock()

	if fm := f.filterMapCache[mapIndex]; fm != nil {
		(*fm)[rowIndex] = row
	}
	rawdb.WriteFilterMapRow(batch, mapRowIndex(mapIndex, rowIndex), []uint32(row))
}

// mapRowIndex calculates the unified storage index where the given row of the
// given map is stored. Note that this indexing scheme is the same as the one
// proposed in EIP-7745 for tree-hashing the filter map structure and for the
// same data proximity reasons it is also suitable for database representation.
// See also:
// https://eips.ethereum.org/EIPS/eip-7745#hash-tree-structure
func mapRowIndex(mapIndex, rowIndex uint32) uint64 {
	epochIndex, mapSubIndex := mapIndex>>logMapsPerEpoch, mapIndex%mapsPerEpoch
	return (uint64(epochIndex)<<logMapHeight+uint64(rowIndex))<<logMapsPerEpoch + uint64(mapSubIndex)
}

// getBlockLvPointer returns the starting log value index where the log values
// generated by the given block are located. If blockNumber is beyond the current
// head then the first unoccupied log value index is returned.
// Note that this function assumes that the read lock is being held.
func (f *FilterMaps) getBlockLvPointer(blockNumber uint64) (uint64, error) {
	if blockNumber > f.headBlockNumber {
		return f.headLvPointer, nil
	}
	if lvPointer, ok := f.lvPointerCache.Get(blockNumber); ok {
		return lvPointer, nil
	}
	lvPointer, err := rawdb.ReadBlockLvPointer(f.db, blockNumber)
	if err != nil {
		return 0, err
	}
	f.lvPointerCache.Add(blockNumber, lvPointer)
	return lvPointer, nil
}

// storeBlockLvPointer stores the starting log value index where the log values
// generated by the given block are located.
func (f *FilterMaps) storeBlockLvPointer(batch ethdb.Batch, blockNumber, lvPointer uint64) {
	f.lvPointerCache.Add(blockNumber, lvPointer)
	rawdb.WriteBlockLvPointer(batch, blockNumber, lvPointer)
}

// deleteBlockLvPointer deletes the starting log value index where the log values
// generated by the given block are located.
func (f *FilterMaps) deleteBlockLvPointer(batch ethdb.Batch, blockNumber uint64) {
	f.lvPointerCache.Remove(blockNumber)
	rawdb.DeleteBlockLvPointer(batch, blockNumber)
}

// getMapBlockPtr returns the number of the block that generated the first log
// value entry of the given map.
func (f *FilterMaps) getMapBlockPtr(mapIndex uint32) (uint64, error) {
	if blockPtr, ok := f.blockPtrCache.Get(mapIndex); ok {
		return blockPtr, nil
	}
	blockPtr, err := rawdb.ReadFilterMapBlockPtr(f.db, mapIndex)
	if err != nil {
		return 0, err
	}
	f.blockPtrCache.Add(mapIndex, blockPtr)
	return blockPtr, nil
}

// storeMapBlockPtr stores the number of the block that generated the first log
// value entry of the given map.
func (f *FilterMaps) storeMapBlockPtr(batch ethdb.Batch, mapIndex uint32, blockPtr uint64) {
	f.blockPtrCache.Add(mapIndex, blockPtr)
	rawdb.WriteFilterMapBlockPtr(batch, mapIndex, blockPtr)
}

// deleteMapBlockPtr deletes the number of the block that generated the first log
// value entry of the given map.
func (f *FilterMaps) deleteMapBlockPtr(batch ethdb.Batch, mapIndex uint32) {
	f.blockPtrCache.Remove(mapIndex)
	rawdb.DeleteFilterMapBlockPtr(batch, mapIndex)
}

// addressValue returns the log value hash of a log emitting address.
func addressValue(address common.Address) common.Hash {
	var result common.Hash
	hasher := sha256.New()
	hasher.Write(address[:])
	hasher.Sum(result[:0])
	return result
}

// topicValue returns the log value hash of a log topic.
func topicValue(topic common.Hash) common.Hash {
	var result common.Hash
	hasher := sha256.New()
	hasher.Write(topic[:])
	hasher.Sum(result[:0])
	return result
}

// rowIndex returns the row index in which the given log value should be marked
// during the given epoch. Note that row assignments are re-shuffled in every
// epoch in order to ensure that even though there are always a few more heavily
// used rows due to very popular addresses and topics, these will not make search
// for other log values very expensive. Even if certain values are occasionally
// sorted into these heavy rows, in most of the epochs they are placed in average
// length rows.
func rowIndex(epochIndex uint32, logValue common.Hash) uint32 {
	hasher := sha256.New()
	hasher.Write(logValue[:])
	var indexEnc [4]byte
	binary.LittleEndian.PutUint32(indexEnc[:], epochIndex)
	hasher.Write(indexEnc[:])
	var hash common.Hash
	hasher.Sum(hash[:0])
	return binary.LittleEndian.Uint32(hash[:4]) % mapHeight
}

// columnIndex returns the column index that should be added to the appropriate
// row in order to place a mark for the next log value.
func columnIndex(lvIndex uint64, logValue common.Hash) uint32 {
	x := uint32(lvIndex % valuesPerMap) // log value sub-index
	transformHash := transformHash(uint32(lvIndex/valuesPerMap), logValue)
	// apply column index transformation function
	x += binary.LittleEndian.Uint32(transformHash[0:4])
	x *= binary.LittleEndian.Uint32(transformHash[4:8])*2 + 1
	x ^= binary.LittleEndian.Uint32(transformHash[8:12])
	x *= binary.LittleEndian.Uint32(transformHash[12:16])*2 + 1
	x += binary.LittleEndian.Uint32(transformHash[16:20])
	x *= binary.LittleEndian.Uint32(transformHash[20:24])*2 + 1
	x ^= binary.LittleEndian.Uint32(transformHash[24:28])
	x *= binary.LittleEndian.Uint32(transformHash[28:32])*2 + 1
	return x
}

// transformHash calculates a hash specific to a given map and log value hash
// that defines a bijective function on the uint32 range. This function is used
// to transform the log value sub-index (distance from the first index of the map)
// into a 32 bit column index, then applied in reverse when searching for potential
// matches for a given log value.
func transformHash(mapIndex uint32, logValue common.Hash) (result common.Hash) {
	hasher := sha256.New()
	hasher.Write(logValue[:])
	var indexEnc [4]byte
	binary.LittleEndian.PutUint32(indexEnc[:], mapIndex)
	hasher.Write(indexEnc[:])
	hasher.Sum(result[:0])
	return
}

// potentialMatches returns the list of log value indices potentially matching
// the given log value hash in the range of the filter map the row belongs to.
// Note that the list of indices is always sorted and potential duplicates are
// removed. Though the column indices are stored in the same order they were
// added and therefore the true matches are automatically reverse transformed
// in the right order, false positives can ruin this property. Since these can
// only be separated from true matches after the combined pattern matching of the
// outputs of individual log value matchers and this pattern matcher assumes a
// sorted and duplicate-free list of indices, we should ensure these properties
// here.
func (row FilterRow) potentialMatches(mapIndex uint32, logValue common.Hash) potentialMatches {
	results := make(potentialMatches, 0, 8)
	transformHash := transformHash(mapIndex, logValue)
	sub1 := binary.LittleEndian.Uint32(transformHash[0:4])
	mul1 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[4:8])*2 + 1)
	xor1 := binary.LittleEndian.Uint32(transformHash[8:12])
	mul2 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[12:16])*2 + 1)
	sub2 := binary.LittleEndian.Uint32(transformHash[16:20])
	mul3 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[20:24])*2 + 1)
	xor2 := binary.LittleEndian.Uint32(transformHash[24:28])
	mul4 := uint32ModInverse(binary.LittleEndian.Uint32(transformHash[28:32])*2 + 1)
	// perform reverse column index transformation on all column indices of the row.
	// if a column index was added by the searched log value then the reverse
	// transform will yield a valid log value sub-index of the given map.
	// Column index is 32 bits long while there are 2**16 valid log value indices
	// in the map's range, so this can also happen by accident with 1 in 2**16
	// chance, in which case we have a false positive.
	for _, columnIndex := range row {
		if potentialSubIndex := (((((((columnIndex * mul4) ^ xor2) * mul3) - sub2) * mul2) ^ xor1) * mul1) - sub1; potentialSubIndex < valuesPerMap {
			results = append(results, uint64(mapIndex)*valuesPerMap+uint64(potentialSubIndex))
		}
	}
	sort.Sort(results)
	// remove duplicates
	j := 0
	for i, match := range results {
		if i == 0 || match != results[i-1] {
			results[j] = results[i]
			j++
		}
	}
	return results[:j]
}

// potentialMatches is a strictly monotonically increasing list of log value
// indices in the range of a filter map that are potential matches for certain
// filter criteria.
// Note that nil is used as a wildcard and therefore means that all log value
// indices in the filter map range are potential matches. If there are no
// potential matches in the given map's range then an empty slice should be used.
type potentialMatches []uint64

// noMatches means there are no potential matches in a given filter map's range.
var noMatches = potentialMatches{}

func (p potentialMatches) Len() int           { return len(p) }
func (p potentialMatches) Less(i, j int) bool { return p[i] < p[j] }
func (p potentialMatches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// uint32ModInverse takes an odd 32 bit number and returns its modular
// multiplicative inverse (mod 2**32), meaning that for any uint32 x and odd y
// x * y *  uint32ModInverse(y) == 1.
func uint32ModInverse(v uint32) uint32 {
	if v&1 == 0 {
		panic("uint32ModInverse called with even argument")
	}
	m := int64(1) << 32
	m0 := m
	a := int64(v)
	x, y := int64(1), int64(0)
	for a > 1 {
		q := a / m
		m, a = a%m, m
		x, y = y, x-q*y
	}
	if x < 0 {
		x += m0
	}
	return uint32(x)
}
