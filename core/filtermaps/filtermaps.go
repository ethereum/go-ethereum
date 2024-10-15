// Copyright 2024 The go-ethereum Authors
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

package filtermaps

import (
	"errors"
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

const headCacheSize = 8 // maximum number of recent filter maps cached in memory

// blockchain defines functions required by the FilterMaps log indexer.
type blockchain interface {
	CurrentBlock() *types.Header
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetCanonicalHash(number uint64) common.Hash
	GetReceiptsByHash(hash common.Hash) types.Receipts
}

// FilterMaps is the in-memory representation of the log index structure that is
// responsible for building and updating the index according to the canonical
// chain.
// Note that FilterMaps implements the same data structure as proposed in EIP-7745
// without the tree hashing and consensus changes:
// https://eips.ethereum.org/EIPS/eip-7745
type FilterMaps struct {
	closeCh               chan struct{}
	closeWg               sync.WaitGroup
	history, unindexLimit uint64
	noHistory             bool
	Params
	chain         blockchain
	matcherSyncCh chan *FilterMapsMatcherBackend

	db ethdb.KeyValueStore

	// fields written by the indexer and read by matcher backend. Indexer can
	// read them without a lock and write them under indexLock write lock.
	// Matcher backend can read them under indexLock read lock.
	indexLock sync.RWMutex
	filterMapsRange
	// filterMapCache caches certain filter maps (headCacheSize most recent maps
	// and one tail map) that are expected to be frequently accessed and modified
	// while updating the structure. Note that the set of cached maps depends
	// only on filterMapsRange and rows of other maps are not cached here.
	filterMapCache map[uint32]filterMap

	// also accessed by indexer and matcher backend but no locking needed.
	blockPtrCache  *lru.Cache[uint32, uint64]
	lvPointerCache *lru.Cache[uint64, uint64]

	// the matchers set and the fields of FilterMapsMatcherBackend instances are
	// read and written both by exported functions and the indexer.
	// Note that if both indexLock and matchersLock needs to be locked then
	// indexLock should be locked first.
	matchersLock sync.Mutex
	matchers     map[*FilterMapsMatcherBackend]struct{}

	// fields only accessed by the indexer (no mutex required).
	revertPoints                                                           map[uint64]*revertPoint
	startHeadUpdate, loggedHeadUpdate, loggedTailExtend, loggedTailUnindex bool
	startedHeadUpdate, startedTailExtend, startedTailUnindex               time.Time
	lastLogHeadUpdate, lastLogTailExtend, lastLogTailUnindex               time.Time
	ptrHeadUpdate, ptrTailExtend, ptrTailUnindex                           uint64

	waitIdleCh chan chan bool
}

// filterMap is a full or partial in-memory representation of a filter map where
// rows are allowed to have a nil value meaning the row is not stored in the
// structure. Note that therefore a known empty row should be represented with
// a zero-length slice.
// It can be used as a memory cache or an overlay while preparing a batch of
// changes to the structure. In either case a nil value should be interpreted
// as transparent (uncached/unchanged).
type filterMap []FilterRow

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
// Note that tailBlockLvPointer points to the earliest log value index belonging
// to the tail block while tailLvPointer points to the earliest log value index
// added to the corresponding filter map. The latter might point to an earlier
// index after tail blocks have been unindexed because we do not remove tail
// values one by one, rather delete entire maps when all blocks that had log
// values in those maps are unindexed.
type filterMapsRange struct {
	initialized                                      bool
	headLvPointer, tailLvPointer, tailBlockLvPointer uint64
	headBlockNumber, tailBlockNumber                 uint64
	headBlockHash, tailParentHash                    common.Hash
}

// NewFilterMaps creates a new FilterMaps and starts the indexer in order to keep
// the structure in sync with the given blockchain.
func NewFilterMaps(db ethdb.KeyValueStore, chain blockchain, params Params, history, unindexLimit uint64, noHistory bool) *FilterMaps {
	rs, err := rawdb.ReadFilterMapsRange(db)
	if err != nil {
		log.Error("Error reading log index range", "error", err)
	}
	params.deriveFields()
	fm := &FilterMaps{
		db:           db,
		chain:        chain,
		closeCh:      make(chan struct{}),
		waitIdleCh:   make(chan chan bool),
		history:      history,
		noHistory:    noHistory,
		unindexLimit: unindexLimit,
		Params:       params,
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
		filterMapCache: make(map[uint32]filterMap),
		blockPtrCache:  lru.NewCache[uint32, uint64](1000),
		lvPointerCache: lru.NewCache[uint64, uint64](1000),
		revertPoints:   make(map[uint64]*revertPoint),
	}
	fm.tailBlockLvPointer, err = fm.getBlockLvPointer(fm.tailBlockNumber)
	if err != nil {
		log.Error("Error fetching tail block pointer, resetting log index", "error", err)
		fm.filterMapsRange = filterMapsRange{} // updateLoop resets the database
	}
	return fm
}

// Start starts the indexer.
func (f *FilterMaps) Start() {
	f.closeWg.Add(2)
	go f.removeBloomBits()
	go f.updateLoop()
}

// Stop ensures that the indexer is fully stopped before returning.
func (f *FilterMaps) Stop() {
	close(f.closeCh)
	f.closeWg.Wait()
}

// reset un-initializes the FilterMaps structure and removes all related data from
// the database. The function returns true if everything was successfully removed.
func (f *FilterMaps) reset() bool {
	f.indexLock.Lock()
	f.filterMapsRange = filterMapsRange{}
	f.filterMapCache = make(map[uint32]filterMap)
	f.revertPoints = make(map[uint64]*revertPoint)
	f.blockPtrCache.Purge()
	f.lvPointerCache.Purge()
	f.indexLock.Unlock()
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
// Note that this function assumes that the index write lock is being held.
func (f *FilterMaps) setRange(batch ethdb.KeyValueWriter, newRange filterMapsRange) {
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
// Note that this function assumes that the index write lock is being held.
func (f *FilterMaps) updateMapCache() {
	if !f.initialized {
		return
	}
	newFilterMapCache := make(map[uint32]filterMap)
	firstMap, afterLastMap := uint32(f.tailBlockLvPointer>>f.logValuesPerMap), uint32((f.headLvPointer+f.valuesPerMap-1)>>f.logValuesPerMap)
	headCacheFirst := firstMap + 1
	if afterLastMap > headCacheFirst+headCacheSize {
		headCacheFirst = afterLastMap - headCacheSize
	}
	fm := f.filterMapCache[firstMap]
	if fm == nil {
		fm = make(filterMap, f.mapHeight)
	}
	newFilterMapCache[firstMap] = fm
	for mapIndex := headCacheFirst; mapIndex < afterLastMap; mapIndex++ {
		fm := f.filterMapCache[mapIndex]
		if fm == nil {
			fm = make(filterMap, f.mapHeight)
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
// Note that this function assumes that the indexer read lock is being held when
// called from outside the updateLoop goroutine.
func (f *FilterMaps) getLogByLvIndex(lvIndex uint64) (*types.Log, error) {
	if lvIndex < f.tailBlockLvPointer || lvIndex > f.headLvPointer {
		return nil, nil
	}
	// find possible block range based on map to block pointers
	mapIndex := uint32(lvIndex >> f.logValuesPerMap)
	firstBlockNumber, err := f.getMapBlockPtr(mapIndex)
	if err != nil {
		return nil, err
	}
	if firstBlockNumber < f.tailBlockNumber {
		firstBlockNumber = f.tailBlockNumber
	}
	var lastBlockNumber uint64
	if mapIndex+1 < uint32((f.headLvPointer+f.valuesPerMap-1)>>f.logValuesPerMap) {
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
	receipts := f.chain.GetReceiptsByHash(f.chain.GetCanonicalHash(firstBlockNumber))
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
// Note that the function assumes that the indexLock is not being held (should
// only be called from the updateLoop goroutine).
func (f *FilterMaps) getFilterMapRow(mapIndex, rowIndex uint32) (FilterRow, error) {
	fm := f.filterMapCache[mapIndex]
	if fm != nil && fm[rowIndex] != nil {
		return fm[rowIndex], nil
	}
	row, err := rawdb.ReadFilterMapRow(f.db, f.mapRowIndex(mapIndex, rowIndex))
	if err != nil {
		return nil, err
	}
	if fm != nil {
		f.indexLock.Lock()
		fm[rowIndex] = FilterRow(row)
		f.indexLock.Unlock()
	}
	return FilterRow(row), nil
}

// getFilterMapRowUncached returns the given row of the given map. If the row is
// empty then a non-nil zero length row is returned.
// This function bypasses the memory cache which is mostly useful for processing
// the head and tail maps during the indexing process and should be used by the
// matcher backend which rarely accesses the same row twice and therefore does
// not really benefit from caching anyways.
// The function is unaffected by the indexLock mutex.
func (f *FilterMaps) getFilterMapRowUncached(mapIndex, rowIndex uint32) (FilterRow, error) {
	row, err := rawdb.ReadFilterMapRow(f.db, f.mapRowIndex(mapIndex, rowIndex))
	return FilterRow(row), err
}

// storeFilterMapRow stores a row at the given row index of the given map and also
// caches it in filterMapCache if the given map is cached.
// Note that empty rows are not stored in the database and therefore there is no
// separate delete function; deleting a row is the same as storing an empty row.
// Note that this function assumes that the indexer write lock is being held.
func (f *FilterMaps) storeFilterMapRow(batch ethdb.Batch, mapIndex, rowIndex uint32, row FilterRow) {
	if fm := f.filterMapCache[mapIndex]; fm != nil {
		fm[rowIndex] = row
	}
	rawdb.WriteFilterMapRow(batch, f.mapRowIndex(mapIndex, rowIndex), []uint32(row))
}

// mapRowIndex calculates the unified storage index where the given row of the
// given map is stored. Note that this indexing scheme is the same as the one
// proposed in EIP-7745 for tree-hashing the filter map structure and for the
// same data proximity reasons it is also suitable for database representation.
// See also:
// https://eips.ethereum.org/EIPS/eip-7745#hash-tree-structure
func (f *FilterMaps) mapRowIndex(mapIndex, rowIndex uint32) uint64 {
	epochIndex, mapSubIndex := mapIndex>>f.logMapsPerEpoch, mapIndex&(f.mapsPerEpoch-1)
	return (uint64(epochIndex)<<f.logMapHeight+uint64(rowIndex))<<f.logMapsPerEpoch + uint64(mapSubIndex)
}

// getBlockLvPointer returns the starting log value index where the log values
// generated by the given block are located. If blockNumber is beyond the current
// head then the first unoccupied log value index is returned.
// Note that this function assumes that the indexer read lock is being held when
// called from outside the updateLoop goroutine.
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
