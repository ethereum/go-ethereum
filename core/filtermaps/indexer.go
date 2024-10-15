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
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	startLvMap           = 1 << 31         // map index assigned to init block
	removedPointer       = math.MaxUint64  // used in updateBatch to signal removed items
	revertPointFrequency = 256             // frequency of revert points in database
	cachedRevertPoints   = 64              // revert points for most recent blocks in memory
	logFrequency         = time.Second * 8 // log info frequency during long indexing/unindexing process
)

// updateLoop initializes and updates the log index structure according to the
// canonical chain.
func (f *FilterMaps) updateLoop() {
	defer f.closeWg.Done()

	if f.noHistory {
		f.reset()
		return
	}

	f.indexLock.Lock()
	f.updateMapCache()
	f.indexLock.Unlock()
	if rp, err := f.newUpdateBatch().makeRevertPoint(); err == nil {
		f.revertPoints[rp.blockNumber] = rp
	} else {
		log.Error("Error creating head revert point", "error", err)
	}

	var (
		headEventCh = make(chan core.ChainEvent, 10)
		sub         = f.chain.SubscribeChainEvent(headEventCh)
		head        = f.chain.CurrentBlock()
		stop        bool
		syncMatcher *FilterMapsMatcherBackend
	)

	matcherSync := func() {
		if syncMatcher != nil && f.initialized && f.headBlockHash == head.Hash() {
			syncMatcher.synced(head)
			syncMatcher = nil
		}
	}

	defer func() {
		sub.Unsubscribe()
		matcherSync()
	}()

	wait := func() {
		matcherSync()
		if stop {
			return
		}
	loop:
		for {
			select {
			case ev := <-headEventCh:
				head = ev.Block.Header()
			case syncMatcher = <-f.matcherSyncCh:
				head = f.chain.CurrentBlock()
			case <-f.closeCh:
				stop = true
			case ch := <-f.waitIdleCh:
				head = f.chain.CurrentBlock()
				if head.Hash() == f.headBlockHash {
					ch <- true
					continue loop
				}
				ch <- false
			case <-time.After(time.Second * 20):
				// keep updating log index during syncing
				head = f.chain.CurrentBlock()
			}
			break
		}
	}
	for head == nil {
		wait()
		if stop {
			return
		}
	}

	for !stop {
		if !f.initialized {
			if !f.tryInit(head) {
				return
			}
			if !f.initialized {
				wait()
				continue
			}
		}
		// log index is initialized
		if f.headBlockHash != head.Hash() {
			// log index head need to be updated
			f.tryUpdateHead(func() *types.Header {
				// return nil if head processing needs to be stopped
				select {
				case ev := <-headEventCh:
					head = ev.Block.Header()
				case syncMatcher = <-f.matcherSyncCh:
					head = f.chain.CurrentBlock()
				case <-f.closeCh:
					stop = true
					return nil
				default:
					head = f.chain.CurrentBlock()
				}
				return head
			})
			if stop {
				return
			}
			if !f.initialized {
				continue
			}
			if f.headBlockHash != head.Hash() {
				// if head processing stopped without reaching current head then
				// something went wrong; tryUpdateHead prints an error log in
				// this case and there is nothing better to do here than retry
				// later. Wait for an event though in order to avoid the retry
				// loop spinning at full power.
				wait()
				continue
			}
		}
		// log index is synced to the latest known chain head
		matcherSync()
		// process tail blocks if possible
		if f.tryUpdateTail(head, func() bool {
			// return true if tail processing needs to be stopped
			select {
			case ev := <-headEventCh:
				head = ev.Block.Header()
			case syncMatcher = <-f.matcherSyncCh:
				head = f.chain.CurrentBlock()
			case <-f.closeCh:
				stop = true
				return true
			default:
				head = f.chain.CurrentBlock()
			}
			// stop if there is a new chain head (always prioritize head updates)
			return f.headBlockHash != head.Hash() || syncMatcher != nil
		}) && f.headBlockHash == head.Hash() {
			// if tail processing reached its final state and there is no new
			// head then wait for more events
			wait()
		}
	}
}

// WaitIdle blocks until the indexer is in an idle state while synced up to the
// latest chain head.
func (f *FilterMaps) WaitIdle() {
	if f.noHistory {
		f.closeWg.Wait()
		return
	}
	for {
		ch := make(chan bool)
		f.waitIdleCh <- ch
		if <-ch {
			return
		}
	}
}

// tryInit attempts to initialize the log index structure.
// Returns false if indexer was stopped during a database reset. In this case the
// indexer should exit and remaining parts of the old database will be removed
// at next startup.
func (f *FilterMaps) tryInit(head *types.Header) bool {
	if !f.reset() {
		return false
	}
	receipts := f.chain.GetReceiptsByHash(head.Hash())
	if receipts == nil {
		log.Error("Could not retrieve block receipts for init block", "number", head.Number, "hash", head.Hash())
		return true
	}
	update := f.newUpdateBatch()
	if err := update.initWithBlock(head, receipts); err != nil {
		log.Error("Could not initialize log index", "error", err)
	}
	f.applyUpdateBatch(update)
	log.Info("Initialized log index", "head", head.Number.Uint64())
	return true
}

// tryUpdateHead attempts to update the log index with a new head. If necessary,
// it reverts to a common ancestor with the old head before adding new block logs.
// If no suitable revert point is available (probably a reorg just after init)
// then it resets the index and tries to re-initialize with the new head.
// Returns false if indexer was stopped during a database reset. In this case the
// indexer should exit and remaining parts of the old database will be removed
// at next startup.
func (f *FilterMaps) tryUpdateHead(headFn func() *types.Header) {
	head := headFn()
	if head == nil {
		return
	}

	defer func() {
		if head.Hash() == f.headBlockHash {
			if f.loggedHeadUpdate {
				log.Info("Forward log indexing finished", "processed", f.headBlockNumber-f.ptrHeadUpdate,
					"elapsed", common.PrettyDuration(time.Since(f.lastLogHeadUpdate)))
				f.loggedHeadUpdate, f.startHeadUpdate = false, false
			}
		} else {
			if time.Since(f.lastLogHeadUpdate) > logFrequency || !f.loggedHeadUpdate {
				log.Info("Forward log indexing in progress", "processed", f.headBlockNumber-f.ptrHeadUpdate,
					"remaining", head.Number.Uint64()-f.headBlockNumber,
					"elapsed", common.PrettyDuration(time.Since(f.startedHeadUpdate)))
				f.loggedHeadUpdate = true
				f.lastLogHeadUpdate = time.Now()
			}
		}
	}()

	hc := newHeaderChain(f.chain, head.Number.Uint64(), head.Hash())
	f.revertToCommonAncestor(head.Number.Uint64(), hc)
	if !f.initialized {
		return
	}
	if f.headBlockHash == head.Hash() {
		return
	}

	if !f.startHeadUpdate {
		f.lastLogHeadUpdate = time.Now()
		f.startedHeadUpdate = f.lastLogHeadUpdate
		f.startHeadUpdate = true
		f.ptrHeadUpdate = f.headBlockNumber
	}

	// add new blocks
	update := f.newUpdateBatch()
	for update.headBlockNumber < head.Number.Uint64() {
		header := hc.getHeader(update.headBlockNumber + 1)
		if header == nil {
			log.Error("Header not found", "number", update.headBlockNumber+1)
			return
		}
		receipts := f.chain.GetReceiptsByHash(header.Hash())
		if receipts == nil {
			log.Error("Could not retrieve block receipts for new block", "number", header.Number, "hash", header.Hash())
			break
		}
		if err := update.addBlockToHead(header, receipts); err != nil {
			log.Error("Error adding new block", "number", header.Number, "hash", header.Hash(), "error", err)
			break
		}
		if update.updatedRangeLength() >= f.mapsPerEpoch {
			// limit the amount of data updated in a single batch
			f.applyUpdateBatch(update)
			newHead := headFn()
			if newHead == nil {
				return
			}
			if newHead.Hash() != head.Hash() {
				head = newHead
				hc = newHeaderChain(f.chain, head.Number.Uint64(), head.Hash())
				if hc.getBlockHash(f.headBlockNumber) != f.headBlockHash {
					f.revertToCommonAncestor(head.Number.Uint64(), hc)
					if !f.initialized {
						return
					}
				}
			}
			update = f.newUpdateBatch()
		}
	}
	f.applyUpdateBatch(update)
}

// find the latest revert point that is the ancestor of the new head
func (f *FilterMaps) revertToCommonAncestor(headNum uint64, hc *headerChain) {
	var (
		number = headNum
		rp     *revertPoint
	)
	for {
		var err error
		if rp, err = f.getRevertPoint(number); err == nil {
			if rp == nil || hc.getBlockHash(rp.blockNumber) == rp.blockHash {
				break
			}
		} else {
			log.Error("Error fetching revert point", "block number", number, "error", err)
		}
		if rp.blockNumber == 0 {
			rp = nil
			break
		}
		number = rp.blockNumber - 1
	}
	if rp == nil {
		// there are no more revert points available so we should reset and re-initialize
		log.Warn("No suitable revert point exists; re-initializing log index", "block number", headNum)
		f.setRange(f.db, filterMapsRange{})
		return
	}
	if rp.blockHash == f.headBlockHash {
		return // found the head revert point, nothing to do
	}
	// revert to the common ancestor if necessary
	if rp.blockNumber+128 <= f.headBlockNumber {
		log.Warn("Rolling back log index", "old head", f.headBlockNumber, "new head", rp.blockNumber)
	}
	if err := f.revertTo(rp); err != nil {
		log.Error("Error applying revert point", "block number", rp.blockNumber, "error", err)
	}
}

// tryUpdateTail attempts to extend or shorten the log index according to the
// current head block number and the log history settings.
// stopFn is called regularly during the process, and if it returns true, the
// latest batch is written and the function returns.
// tryUpdateTail returns true if it has reached the desired history length.
func (f *FilterMaps) tryUpdateTail(head *types.Header, stopFn func() bool) bool {
	var tailTarget uint64
	if f.history > 0 {
		if headNum := head.Number.Uint64(); headNum >= f.history {
			tailTarget = headNum + 1 - f.history
		}
	}
	tailNum := f.tailBlockNumber
	if tailNum > tailTarget {
		if !f.tryExtendTail(tailTarget, stopFn) {
			return false
		}
	}
	if tailNum+f.unindexLimit <= tailTarget {
		return f.tryUnindexTail(tailTarget, stopFn)
	}
	return true
}

// tryExtendTail attempts to extend the log index backwards until the desired
// indexed history length is achieved. Returns true if finished.
func (f *FilterMaps) tryExtendTail(tailTarget uint64, stopFn func() bool) bool {
	defer func() {
		if f.tailBlockNumber <= tailTarget {
			if f.loggedTailExtend {
				log.Info("Reverse log indexing finished", "maps", f.mapCount(f.logValuesPerMap), "history", f.headBlockNumber+1-f.tailBlockNumber,
					"processed", f.ptrTailExtend-f.tailBlockNumber, "elapsed", common.PrettyDuration(time.Since(f.startedTailExtend)))
				f.loggedTailExtend = false
			}
		}
	}()

	number, parentHash := f.tailBlockNumber, f.tailParentHash

	if !f.loggedTailExtend {
		f.lastLogTailExtend = time.Now()
		f.startedTailExtend = f.lastLogTailExtend
		f.ptrTailExtend = f.tailBlockNumber
	}

	update := f.newUpdateBatch()
	lastTailEpoch := update.tailEpoch()
	for number > tailTarget && !stopFn() {
		if tailEpoch := update.tailEpoch(); tailEpoch < lastTailEpoch {
			// limit the amount of data updated in a single batch
			f.applyUpdateBatch(update)

			if time.Since(f.lastLogTailExtend) > logFrequency || !f.loggedTailExtend {
				log.Info("Reverse log indexing in progress", "maps", update.mapCount(f.logValuesPerMap), "history", update.headBlockNumber+1-update.tailBlockNumber,
					"processed", f.ptrTailExtend-update.tailBlockNumber, "remaining", update.tailBlockNumber-tailTarget,
					"elapsed", common.PrettyDuration(time.Since(f.startedTailExtend)))
				f.loggedTailExtend = true
				f.lastLogTailExtend = time.Now()
			}

			update = f.newUpdateBatch()
			lastTailEpoch = tailEpoch
		}
		newTail := f.chain.GetHeader(parentHash, number-1)
		if newTail == nil {
			log.Error("Tail header not found", "number", number-1, "hash", parentHash)
			break
		}
		receipts := f.chain.GetReceiptsByHash(newTail.Hash())
		if receipts == nil {
			log.Error("Could not retrieve block receipts for tail block", "number", newTail.Number, "hash", newTail.Hash())
			break
		}
		if err := update.addBlockToTail(newTail, receipts); err != nil {
			log.Error("Error adding tail block", "number", newTail.Number, "hash", newTail.Hash(), "error", err)
			break
		}
		number, parentHash = newTail.Number.Uint64(), newTail.ParentHash
	}
	f.applyUpdateBatch(update)
	return number <= tailTarget
}

// tryUnindexTail attempts to prune the log index tail until the desired indexed
// history length is achieved. Returns true if finished.
func (f *FilterMaps) tryUnindexTail(tailTarget uint64, stopFn func() bool) bool {
	if !f.loggedTailUnindex {
		f.lastLogTailUnindex = time.Now()
		f.startedTailUnindex = f.lastLogTailUnindex
		f.ptrTailUnindex = f.tailBlockNumber
	}
	for {
		if f.unindexTailEpoch(tailTarget) {
			log.Info("Log unindexing finished", "maps", f.mapCount(f.logValuesPerMap), "history", f.headBlockNumber+1-f.tailBlockNumber,
				"removed", f.tailBlockNumber-f.ptrTailUnindex, "elapsed", common.PrettyDuration(time.Since(f.startedTailUnindex)))
			f.loggedTailUnindex = false
			return true
		}
		if time.Since(f.lastLogTailUnindex) > logFrequency || !f.loggedTailUnindex {
			log.Info("Log unindexing in progress", "maps", f.mapCount(f.logValuesPerMap), "history", f.headBlockNumber+1-f.tailBlockNumber,
				"removed", f.tailBlockNumber-f.ptrTailUnindex, "remaining", tailTarget-f.tailBlockNumber,
				"elapsed", common.PrettyDuration(time.Since(f.startedTailUnindex)))
			f.loggedTailUnindex = true
			f.lastLogTailUnindex = time.Now()
		}
		if stopFn() {
			return false
		}
	}
}

// unindexTailEpoch unindexes at most an epoch of tail log index data until the
// desired tail target is reached.
func (f *FilterMaps) unindexTailEpoch(tailTarget uint64) (finished bool) {
	oldRange := f.filterMapsRange
	newTailMap, changed := f.unindexTailPtr(tailTarget)
	newRange := f.filterMapsRange

	if !changed {
		return true // nothing more to do
	}
	finished = newRange.tailBlockNumber == tailTarget

	oldTailMap := uint32(oldRange.tailLvPointer >> f.logValuesPerMap)
	// remove map data [oldTailMap, newTailMap) and block data
	// [oldRange.tailBlockNumber, newRange.tailBlockNumber)
	f.indexLock.Lock()
	batch := f.db.NewBatch()
	for blockNumber := oldRange.tailBlockNumber; blockNumber < newRange.tailBlockNumber; blockNumber++ {
		f.deleteBlockLvPointer(batch, blockNumber)
		if blockNumber%revertPointFrequency == 0 {
			rawdb.DeleteRevertPoint(batch, blockNumber)
		}
	}
	for mapIndex := oldTailMap; mapIndex < newTailMap; mapIndex++ {
		f.deleteMapBlockPtr(batch, mapIndex)
	}
	for rowIndex := uint32(0); rowIndex < f.mapHeight; rowIndex++ {
		for mapIndex := oldTailMap; mapIndex < newTailMap; mapIndex++ {
			f.storeFilterMapRow(batch, mapIndex, rowIndex, emptyRow)
		}
	}
	newRange.tailLvPointer = uint64(newTailMap) << f.logValuesPerMap
	if newRange.tailLvPointer > newRange.tailBlockLvPointer {
		log.Error("Cannot unindex filter maps beyond tail block log value pointer", "tailLvPointer", newRange.tailLvPointer, "tailBlockLvPointer", newRange.tailBlockLvPointer)
		f.indexLock.Unlock()
		return
	}
	f.setRange(batch, newRange)
	f.indexLock.Unlock()

	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
	return
}

// unindexTailPtr determines the range of tail maps to be removed in the next
// batch and updates the tail block number and hash and the corresponding
// tailBlockLvPointer accordingly.
// Note that this function does not remove old index data, only marks it unused
// by updating the tail pointers, except for targetLvPointer which is not changed
// yet as it marks the tail of the log index data stored in the database and
// therefore should be updated when map data is actually removed.
// Note that this function assumes that the read/write lock is being held.
func (f *FilterMaps) unindexTailPtr(tailTarget uint64) (newTailMap uint32, changed bool) {
	// obtain target log value pointer
	if tailTarget <= f.tailBlockNumber || tailTarget > f.headBlockNumber {
		return 0, false // nothing to do
	}
	targetLvPointer, err := f.getBlockLvPointer(tailTarget)
	if err != nil {
		log.Error("Error fetching tail target log value pointer", "block number", tailTarget, "error", err)
		return 0, false
	}
	newRange := f.filterMapsRange
	tailMap := uint32(f.tailBlockLvPointer >> f.logValuesPerMap)
	nextEpochFirstMap := ((tailMap >> f.logMapsPerEpoch) + 1) << f.logMapsPerEpoch
	targetMap := uint32(targetLvPointer >> f.logValuesPerMap)
	if targetMap <= nextEpochFirstMap {
		// unindexed range is within a single epoch, do it in a single batch
		newRange.tailBlockNumber, newRange.tailBlockLvPointer, newTailMap = tailTarget, targetLvPointer, targetMap
	} else {
		// new tail map should be nextEpochFirstMap, determine new tail block
		tailBlockNumber, err := f.getMapBlockPtr(nextEpochFirstMap)
		if err != nil {
			log.Error("Error fetching tail map block pointer", "map index", nextEpochFirstMap, "error", err)
			return 0, false
		}
		tailBlockNumber++
		tailBlockLvPointer, err := f.getBlockLvPointer(tailBlockNumber)
		if err != nil {
			log.Error("Error fetching tail block log value pointer", "block number", tailBlockNumber, "error", err)
			return 0, false
		}
		newRange.tailBlockNumber, newRange.tailBlockLvPointer, newTailMap = tailBlockNumber, tailBlockLvPointer, uint32(tailBlockLvPointer>>f.logValuesPerMap)
	}
	// obtain tail target's parent hash
	if newRange.tailBlockNumber > 0 {
		newRange.tailParentHash = newHeaderChain(f.chain, f.headBlockNumber, f.headBlockHash).getBlockHash(newRange.tailBlockNumber - 1)
	}
	f.setRange(f.db, newRange)
	return newTailMap, true
}

type headerChain struct {
	chain        blockchain
	nonCanonical []*types.Header
	number       uint64
	hash         common.Hash
}

func newHeaderChain(chain blockchain, number uint64, hash common.Hash) *headerChain {
	hc := &headerChain{
		chain:  chain,
		number: number,
		hash:   hash,
	}
	hc.extendNonCanonical()
	return hc
}

func (hc *headerChain) extendNonCanonical() bool {
	for hc.hash != hc.chain.GetCanonicalHash(hc.number) {
		header := hc.chain.GetHeader(hc.hash, hc.number)
		if header == nil {
			log.Error("Header not found", "number", hc.number, "hash", hc.hash)
			return false
		}
		hc.nonCanonical = append(hc.nonCanonical, header)
		hc.number, hc.hash = hc.number-1, header.ParentHash
	}
	return true
}

func (hc *headerChain) getBlockHash(number uint64) common.Hash {
	if number <= hc.number {
		hash := hc.chain.GetCanonicalHash(number)
		if !hc.extendNonCanonical() {
			return common.Hash{}
		}
		if number <= hc.number {
			return hash
		}
	}
	if number-hc.number > uint64(len(hc.nonCanonical)) {
		return common.Hash{}
	}
	return hc.nonCanonical[len(hc.nonCanonical)+1-int(number-hc.number)].Hash()
}

func (hc *headerChain) getHeader(number uint64) *types.Header {
	if number <= hc.number {
		hash := hc.chain.GetCanonicalHash(number)
		if !hc.extendNonCanonical() {
			return nil
		}
		if number <= hc.number {
			return hc.chain.GetHeader(hash, number)
		}
	}
	if number-hc.number > uint64(len(hc.nonCanonical)) {
		return nil
	}
	return hc.nonCanonical[len(hc.nonCanonical)+1-int(number-hc.number)]
}

// updateBatch is a memory overlay collecting changes to the index log structure
// that can be written to the database in a single batch while the in-memory
// representations in FilterMaps are also updated.
type updateBatch struct {
	f *FilterMaps
	filterMapsRange
	maps                   map[uint32]filterMap // nil rows are unchanged
	blockLvPointer         map[uint64]uint64    // removedPointer means delete
	mapBlockPtr            map[uint32]uint64    // removedPointer means delete
	revertPoints           map[uint64]*revertPoint
	firstMap, afterLastMap uint32
}

// newUpdateBatch creates a new updateBatch.
func (f *FilterMaps) newUpdateBatch() *updateBatch {
	return &updateBatch{
		f:               f,
		filterMapsRange: f.filterMapsRange,
		maps:            make(map[uint32]filterMap),
		blockLvPointer:  make(map[uint64]uint64),
		mapBlockPtr:     make(map[uint32]uint64),
		revertPoints:    make(map[uint64]*revertPoint),
	}
}

// applyUpdateBatch writes creates a batch and writes all changes to the database
// and also updates the in-memory representations of log index data.
func (f *FilterMaps) applyUpdateBatch(u *updateBatch) {
	f.indexLock.Lock()

	batch := f.db.NewBatch()
	// write or remove block to log value index pointers
	for blockNumber, lvPointer := range u.blockLvPointer {
		if lvPointer != removedPointer {
			f.storeBlockLvPointer(batch, blockNumber, lvPointer)
		} else {
			f.deleteBlockLvPointer(batch, blockNumber)
		}
	}
	// write or remove filter map to block number pointers
	for mapIndex, blockNumber := range u.mapBlockPtr {
		if blockNumber != removedPointer {
			f.storeMapBlockPtr(batch, mapIndex, blockNumber)
		} else {
			f.deleteMapBlockPtr(batch, mapIndex)
		}
	}
	// write filter map rows
	for rowIndex := uint32(0); rowIndex < f.mapHeight; rowIndex++ {
		for mapIndex := u.firstMap; mapIndex < u.afterLastMap; mapIndex++ {
			if fm := u.maps[mapIndex]; fm != nil {
				if row := fm[rowIndex]; row != nil {
					f.storeFilterMapRow(batch, mapIndex, rowIndex, row)
				}
			}
		}
	}
	// delete removed revert points from the database
	if u.headBlockNumber < f.headBlockNumber {
		for b := u.headBlockNumber + 1; b <= f.headBlockNumber; b++ {
			delete(f.revertPoints, b)
			if b%revertPointFrequency == 0 {
				rawdb.DeleteRevertPoint(batch, b)
			}
		}
	}
	// delete removed revert points from the memory cache
	if u.headBlockNumber > f.headBlockNumber {
		for b := f.headBlockNumber + 1; b <= u.headBlockNumber; b++ {
			delete(f.revertPoints, b-cachedRevertPoints)
		}
	}
	// store new revert points in database and/or memory
	for b, rp := range u.revertPoints {
		if b+cachedRevertPoints > u.headBlockNumber {
			f.revertPoints[b] = rp
		}
		if b%revertPointFrequency == 0 {
			rawdb.WriteRevertPoint(batch, b, &rawdb.RevertPoint{
				BlockHash: rp.blockHash,
				MapIndex:  rp.mapIndex,
				RowLength: rp.rowLength,
			})
		}
	}
	// update filterMapsRange
	f.setRange(batch, u.filterMapsRange)
	f.indexLock.Unlock()

	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
}

// updatedRangeLength returns the length of the updated filter map range.
func (u *updateBatch) updatedRangeLength() uint32 {
	return u.afterLastMap - u.firstMap
}

// tailEpoch returns the tail epoch index.
func (u *updateBatch) tailEpoch() uint32 {
	return uint32(u.tailBlockLvPointer >> (u.f.logValuesPerMap + u.f.logMapsPerEpoch))
}

// getRowPtr returns a pointer to a FilterRow that can be modified. If the batch
// did not have a modified version of the given row yet, it is retrieved using the
// request function from the backing FilterMaps cache or database and copied
// before modification.
func (u *updateBatch) getRowPtr(mapIndex, rowIndex uint32) (*FilterRow, error) {
	fm := u.maps[mapIndex]
	if fm == nil {
		fm = make(filterMap, u.f.mapHeight)
		u.maps[mapIndex] = fm
		if mapIndex < u.firstMap || u.afterLastMap == 0 {
			u.firstMap = mapIndex
		}
		if mapIndex >= u.afterLastMap {
			u.afterLastMap = mapIndex + 1
		}
	}
	rowPtr := &fm[rowIndex]
	if *rowPtr == nil {
		if filterRow, err := u.f.getFilterMapRow(mapIndex, rowIndex); err == nil {
			// filterRow is read only, copy before write
			*rowPtr = make(FilterRow, len(filterRow), len(filterRow)+8)
			copy(*rowPtr, filterRow)
		} else {
			return nil, err
		}
	}
	return rowPtr, nil
}

// initWithBlock initializes the log index with the given block as head.
func (u *updateBatch) initWithBlock(header *types.Header, receipts types.Receipts) error {
	if u.initialized {
		return errors.New("already initialized")
	}
	u.initialized = true
	startLvPointer := uint64(startLvMap) << u.f.logValuesPerMap
	u.headLvPointer, u.tailLvPointer, u.tailBlockLvPointer = startLvPointer, startLvPointer, startLvPointer
	u.headBlockNumber, u.tailBlockNumber = header.Number.Uint64()-1, header.Number.Uint64()
	u.headBlockHash, u.tailParentHash = header.ParentHash, header.ParentHash
	u.addBlockToHead(header, receipts)
	return nil
}

// addValueToHead adds a single log value to the head of the log index.
func (u *updateBatch) addValueToHead(logValue common.Hash) error {
	mapIndex := uint32(u.headLvPointer >> u.f.logValuesPerMap)
	rowPtr, err := u.getRowPtr(mapIndex, u.f.rowIndex(mapIndex>>u.f.logMapsPerEpoch, logValue))
	if err != nil {
		return err
	}
	column := u.f.columnIndex(u.headLvPointer, logValue)
	*rowPtr = append(*rowPtr, column)
	u.headLvPointer++
	return nil
}

// addBlockToHead adds the logs of the given block to the head of the log index.
// It also adds block to log value index and filter map to block pointers and
// a new revert point.
func (u *updateBatch) addBlockToHead(header *types.Header, receipts types.Receipts) error {
	if !u.initialized {
		return errors.New("not initialized")
	}
	if header.ParentHash != u.headBlockHash {
		return errors.New("addBlockToHead parent mismatch")
	}
	number := header.Number.Uint64()
	u.blockLvPointer[number] = u.headLvPointer
	startMap := uint32((u.headLvPointer + u.f.valuesPerMap - 1) >> u.f.logValuesPerMap)
	if err := iterateReceipts(receipts, u.addValueToHead); err != nil {
		return err
	}
	stopMap := uint32((u.headLvPointer + u.f.valuesPerMap - 1) >> u.f.logValuesPerMap)
	for m := startMap; m < stopMap; m++ {
		u.mapBlockPtr[m] = number
	}
	u.headBlockNumber, u.headBlockHash = number, header.Hash()
	if (u.headBlockNumber-cachedRevertPoints)%revertPointFrequency != 0 {
		delete(u.revertPoints, u.headBlockNumber-cachedRevertPoints)
	}
	if rp, err := u.makeRevertPoint(); err != nil {
		return err
	} else if rp != nil {
		u.revertPoints[u.headBlockNumber] = rp
	}
	return nil
}

// addValueToTail adds a single log value to the tail of the log index.
func (u *updateBatch) addValueToTail(logValue common.Hash) error {
	if u.tailBlockLvPointer == 0 {
		return errors.New("tail log value pointer underflow")
	}
	if u.tailBlockLvPointer < u.tailLvPointer {
		panic("tailBlockLvPointer < tailLvPointer")
	}
	u.tailBlockLvPointer--
	if u.tailBlockLvPointer >= u.tailLvPointer {
		return nil // already added to the map
	}
	u.tailLvPointer--
	mapIndex := uint32(u.tailBlockLvPointer >> u.f.logValuesPerMap)
	rowPtr, err := u.getRowPtr(mapIndex, u.f.rowIndex(mapIndex>>u.f.logMapsPerEpoch, logValue))
	if err != nil {
		return err
	}
	column := u.f.columnIndex(u.tailBlockLvPointer, logValue)
	*rowPtr = append(*rowPtr, 0)
	copy((*rowPtr)[1:], (*rowPtr)[:len(*rowPtr)-1])
	(*rowPtr)[0] = column
	return nil
}

// addBlockToTail adds the logs of the given block to the tail of the log index.
// It also adds block to log value index and filter map to block pointers.
func (u *updateBatch) addBlockToTail(header *types.Header, receipts types.Receipts) error {
	if !u.initialized {
		return errors.New("not initialized")
	}
	if header.Hash() != u.tailParentHash {
		return errors.New("addBlockToTail parent mismatch")
	}
	number := header.Number.Uint64()
	stopMap := uint32((u.tailBlockLvPointer + u.f.valuesPerMap - 1) >> u.f.logValuesPerMap)
	var cnt int
	if err := iterateReceiptsReverse(receipts, func(lv common.Hash) error {
		cnt++
		return u.addValueToTail(lv)
	}); err != nil {
		return err
	}
	startMap := uint32(u.tailBlockLvPointer >> u.f.logValuesPerMap)
	for m := startMap; m < stopMap; m++ {
		u.mapBlockPtr[m] = number
	}
	u.blockLvPointer[number] = u.tailBlockLvPointer
	u.tailBlockNumber, u.tailParentHash = number, header.ParentHash
	return nil
}

// iterateReceipts iterates the given block receipts, generates log value hashes
// and passes them to the given callback function as a parameter.
func iterateReceipts(receipts types.Receipts, valueCb func(common.Hash) error) error {
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			if err := valueCb(addressValue(log.Address)); err != nil {
				return err
			}
			for _, topic := range log.Topics {
				if err := valueCb(topicValue(topic)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// iterateReceiptsReverse iterates the given block receipts, generates log value
// hashes in reverse order and passes them to the given callback function as a
// parameter.
func iterateReceiptsReverse(receipts types.Receipts, valueCb func(common.Hash) error) error {
	for i := len(receipts) - 1; i >= 0; i-- {
		logs := receipts[i].Logs
		for j := len(logs) - 1; j >= 0; j-- {
			log := logs[j]
			for k := len(log.Topics) - 1; k >= 0; k-- {
				if err := valueCb(topicValue(log.Topics[k])); err != nil {
					return err
				}
			}
			if err := valueCb(addressValue(log.Address)); err != nil {
				return err
			}
		}
	}
	return nil
}

// revertPoint can be used to revert the log index to a certain head block.
type revertPoint struct {
	blockNumber uint64
	blockHash   common.Hash
	mapIndex    uint32
	rowLength   []uint
}

// makeRevertPoint creates a new revertPoint.
func (u *updateBatch) makeRevertPoint() (*revertPoint, error) {
	rp := &revertPoint{
		blockNumber: u.headBlockNumber,
		blockHash:   u.headBlockHash,
		mapIndex:    uint32(u.headLvPointer >> u.f.logValuesPerMap),
		rowLength:   make([]uint, u.f.mapHeight),
	}
	if u.tailLvPointer > uint64(rp.mapIndex)<<u.f.logValuesPerMap {
		return nil, nil
	}
	for i := range rp.rowLength {
		var row FilterRow
		if m := u.maps[rp.mapIndex]; m != nil {
			row = m[i]
		}
		if row == nil {
			var err error
			row, err = u.f.getFilterMapRow(rp.mapIndex, uint32(i))
			if err != nil {
				return nil, err
			}
		}
		rp.rowLength[i] = uint(len(row))
	}
	return rp, nil
}

// getRevertPoint retrieves the latest revert point at or before the given block
// number from memory cache or from the database if available. If no such revert
// point is available then it returns no result and no error.
func (f *FilterMaps) getRevertPoint(blockNumber uint64) (*revertPoint, error) {
	if blockNumber > f.headBlockNumber {
		blockNumber = f.headBlockNumber
	}
	if rp := f.revertPoints[blockNumber]; rp != nil {
		return rp, nil
	}
	blockNumber -= blockNumber % revertPointFrequency
	rps, err := rawdb.ReadRevertPoint(f.db, blockNumber)
	if err != nil {
		return nil, err
	}
	if rps == nil {
		return nil, nil
	}
	if uint32(len(rps.RowLength)) != f.mapHeight {
		return nil, errors.New("invalid number of rows in stored revert point")
	}
	return &revertPoint{
		blockNumber: blockNumber,
		blockHash:   rps.BlockHash,
		mapIndex:    rps.MapIndex,
		rowLength:   rps.RowLength,
	}, nil
}

// revertTo reverts the log index to the given revert point.
func (f *FilterMaps) revertTo(rp *revertPoint) error {
	batch := f.db.NewBatch()
	afterLastMap := uint32((f.headLvPointer + f.valuesPerMap - 1) >> f.logValuesPerMap)
	if rp.mapIndex > afterLastMap {
		return errors.New("cannot revert (head map behind revert point)")
	}
	lvPointer := uint64(rp.mapIndex) << f.logValuesPerMap
	for rowIndex, rowLen := range rp.rowLength {
		rowIndex := uint32(rowIndex)
		row, err := f.getFilterMapRow(rp.mapIndex, rowIndex)
		if err != nil {
			return err
		}
		if uint(len(row)) < rowLen {
			return errors.New("cannot revert (row too short)")
		}
		if uint(len(row)) > rowLen {
			f.storeFilterMapRow(batch, rp.mapIndex, rowIndex, row[:rowLen])
		}
		for mapIndex := rp.mapIndex + 1; mapIndex < afterLastMap; mapIndex++ {
			f.storeFilterMapRow(batch, mapIndex, rowIndex, emptyRow)
		}
		lvPointer += uint64(rowLen)
	}
	for mapIndex := rp.mapIndex + 1; mapIndex < afterLastMap; mapIndex++ {
		f.deleteMapBlockPtr(batch, mapIndex)
	}
	for blockNumber := rp.blockNumber + 1; blockNumber <= f.headBlockNumber; blockNumber++ {
		f.deleteBlockLvPointer(batch, blockNumber)
		if blockNumber%revertPointFrequency == 0 {
			rawdb.DeleteRevertPoint(batch, blockNumber)
		}
	}
	newRange := f.filterMapsRange
	newRange.headLvPointer = lvPointer
	newRange.headBlockNumber = rp.blockNumber
	newRange.headBlockHash = rp.blockHash
	f.indexLock.Lock()
	f.setRange(batch, newRange)
	f.indexLock.Unlock()

	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
	return nil
}
