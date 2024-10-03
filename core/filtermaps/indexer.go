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
	startLvMap           = 1 << 31        // map index assigned to init block
	removedPointer       = math.MaxUint64 // used in updateBatch to signal removed items
	revertPointFrequency = 256            // frequency of revert points in database
	cachedRevertPoints   = 64             // revert points for most recent blocks in memory
)

// updateLoop initializes and updates the log index structure according to the
// canonical chain.
func (f *FilterMaps) updateLoop() {
	defer f.closeWg.Done()

	if f.noHistory {
		f.reset()
		return
	}
	f.updateMapCache()
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
		fmr         = f.getRange()
	)

	matcherSync := func() {
		if syncMatcher != nil && fmr.initialized && fmr.headBlockHash == head.Hash() {
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
				if head.Hash() == f.getRange().headBlockHash {
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
	fmr = f.getRange()

	for !stop {
		if !fmr.initialized {
			if !f.tryInit(head) {
				return
			}

			fmr = f.getRange()
			if !fmr.initialized {
				wait()
				continue
			}
		}
		// log index is initialized
		if fmr.headBlockHash != head.Hash() {
			if !f.tryUpdateHead(head) {
				return
			}
			fmr = f.getRange()
			if fmr.headBlockHash != head.Hash() {
				wait()
				continue
			}
		}
		matcherSync()
		// log index head is at latest chain head; process tail blocks if possible
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
			return fmr.headBlockHash != head.Hash()
		}) && fmr.headBlockHash == head.Hash() {
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

// getRange returns the current filterMapsRange.
func (f *FilterMaps) getRange() filterMapsRange {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.filterMapsRange
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
	return true
}

// tryUpdateHead attempts to update the log index with a new head. If necessary,
// it reverts to a common ancestor with the old head before adding new block logs.
// If no suitable revert point is available (probably a reorg just after init)
// then it resets the index and tries to re-initialize with the new head.
// Returns false if indexer was stopped during a database reset. In this case the
// indexer should exit and remaining parts of the old database will be removed
// at next startup.
func (f *FilterMaps) tryUpdateHead(newHead *types.Header) bool {
	// iterate back from new head until the log index head or a revert point and
	// collect headers of blocks to be added
	var (
		newHeaders []*types.Header
		chainPtr   = newHead
		rp         *revertPoint
	)
	for {
		if rp == nil || chainPtr.Number.Uint64() < rp.blockNumber {
			var err error
			rp, err = f.getRevertPoint(chainPtr.Number.Uint64())
			if err != nil {
				log.Error("Error fetching revert point", "block number", chainPtr.Number.Uint64(), "error", err)
				return true
			}
			if rp == nil {
				// there are no more revert points available so we should reset and re-initialize
				log.Warn("No suitable revert point exists; re-initializing log index", "block number", newHead.Number.Uint64())
				return f.tryInit(newHead)
			}
		}
		if chainPtr.Hash() == rp.blockHash {
			// revert point found at an ancestor of the new head
			break
		}
		// keep iterating backwards and collecting headers
		newHeaders = append(newHeaders, chainPtr)
		chainPtr = f.chain.GetHeader(chainPtr.ParentHash, chainPtr.Number.Uint64()-1)
		if chainPtr == nil {
			log.Error("Canonical header not found", "number", chainPtr.Number.Uint64()-1, "hash", chainPtr.ParentHash)
			return true
		}
	}
	if rp.blockHash != f.headBlockHash {
		if rp.blockNumber+128 <= f.headBlockNumber {
			log.Warn("Rolling back log index", "old head", f.headBlockNumber, "new head", chainPtr.Number.Uint64())
		}
		if err := f.revertTo(rp); err != nil {
			log.Error("Error applying revert point", "block number", chainPtr.Number.Uint64(), "error", err)
			return true
		}
	}

	if newHeaders == nil {
		return true
	}
	// add logs of new blocks in reverse order
	update := f.newUpdateBatch()
	for i := len(newHeaders) - 1; i >= 0; i-- {
		newHeader := newHeaders[i]
		receipts := f.chain.GetReceiptsByHash(newHeader.Hash())
		if receipts == nil {
			log.Error("Could not retrieve block receipts for new block", "number", newHeader.Number, "hash", newHeader.Hash())
			break
		}
		if err := update.addBlockToHead(newHeader, receipts); err != nil {
			log.Error("Error adding new block", "number", newHeader.Number, "hash", newHeader.Hash(), "error", err)
			break
		}
		if update.updatedRangeLength() >= f.mapsPerEpoch {
			// limit the amount of data updated in a single batch
			f.applyUpdateBatch(update)
			update = f.newUpdateBatch()
		}
	}
	f.applyUpdateBatch(update)
	return true
}

// tryUpdateTail attempts to extend or prune the log index according to the
// current head block number and the log history settings.
// stopFn is called regularly during the process, and if it returns true, the
// latest batch is written and the function returns.
func (f *FilterMaps) tryUpdateTail(head *types.Header, stopFn func() bool) bool {
	var tailTarget uint64
	if f.history > 0 {
		if headNum := head.Number.Uint64(); headNum >= f.history {
			tailTarget = headNum + 1 - f.history
		}
	}
	tailNum := f.getRange().tailBlockNumber
	if tailNum > tailTarget {
		if !f.tryExtendTail(tailTarget, stopFn) {
			return false
		}
	}
	if tailNum+f.unindexLimit <= tailTarget {
		f.unindexTailPtr(tailTarget)
	}
	return f.tryUnindexTailMaps(tailTarget, stopFn)
}

// tryExtendTail attempts to extend the log index backwards until it indexes the
// tail target block or cannot find more block receipts.
func (f *FilterMaps) tryExtendTail(tailTarget uint64, stopFn func() bool) bool {
	fmr := f.getRange()
	number, parentHash := fmr.tailBlockNumber, fmr.tailParentHash
	update := f.newUpdateBatch()
	lastTailEpoch := update.tailEpoch()
	for number > tailTarget && !stopFn() {
		if tailEpoch := update.tailEpoch(); tailEpoch < lastTailEpoch {
			// limit the amount of data updated in a single batch
			f.applyUpdateBatch(update)
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

// unindexTailPtr updates the tail block number and hash and the corresponding
// tailBlockLvPointer according to the given tail target block number.
// Note that this function does not remove old index data, only marks it unused
// by updating the tail pointers, except for targetLvPointer which is unchanged
// as it marks the tail of the log index data stored in the database.
func (f *FilterMaps) unindexTailPtr(tailTarget uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// obtain target log value pointer
	if tailTarget <= f.tailBlockNumber || tailTarget > f.headBlockNumber {
		return // nothing to do
	}
	targetLvPointer, err := f.getBlockLvPointer(tailTarget)
	fmr := f.filterMapsRange

	if err != nil {
		log.Error("Error fetching tail target log value pointer", "block number", tailTarget, "error", err)
	}

	// obtain tail target's parent hash
	var tailParentHash common.Hash
	if tailTarget > 0 {
		if f.chain.GetCanonicalHash(fmr.headBlockNumber) != fmr.headBlockHash {
			return // if a reorg is happening right now then try again later
		}
		tailParentHash = f.chain.GetCanonicalHash(tailTarget - 1)
		if f.chain.GetCanonicalHash(fmr.headBlockNumber) != fmr.headBlockHash {
			return // check again to make sure that tailParentHash is consistent with the indexed chain
		}
	}

	fmr.tailBlockNumber, fmr.tailParentHash = tailTarget, tailParentHash
	fmr.tailBlockLvPointer = targetLvPointer
	f.setRange(f.db, fmr)
}

// tryUnindexTailMaps removes unused filter maps and corresponding log index
// pointers from the database. This function also updates targetLvPointer.
func (f *FilterMaps) tryUnindexTailMaps(tailTarget uint64, stopFn func() bool) bool {
	fmr := f.getRange()
	tailMap := uint32(fmr.tailLvPointer >> f.logValuesPerMap)
	targetMap := uint32(fmr.tailBlockLvPointer >> f.logValuesPerMap)
	if tailMap >= targetMap {
		return true
	}
	lastEpoch := (targetMap - 1) >> f.logMapsPerEpoch
	removeLvPtr, err := f.getMapBlockPtr(tailMap)
	if err != nil {
		log.Error("Error fetching tail map block pointer", "map index", tailMap, "error", err)
		removeLvPtr = math.MaxUint64 // do not remove anything
	}
	var (
		logged     bool
		lastLogged time.Time
	)
	for tailMap < targetMap && !stopFn() {
		tailEpoch := tailMap >> f.logMapsPerEpoch
		if tailEpoch == lastEpoch {
			f.unindexMaps(tailMap, targetMap, &removeLvPtr)
			break
		}
		nextTailMap := (tailEpoch + 1) << f.logMapsPerEpoch
		f.unindexMaps(tailMap, nextTailMap, &removeLvPtr)
		tailMap = nextTailMap
		if !logged || time.Since(lastLogged) >= time.Second*10 {
			log.Info("Pruning log index tail...", "filter maps left", targetMap-tailMap)
			logged, lastLogged = true, time.Now()
		}
	}
	if logged {
		log.Info("Finished pruning log index tail", "filter maps left", targetMap-tailMap)
	}
	return tailMap >= targetMap
}

// unindexMaps removes filter maps and corresponding log index pointers in the
// specified range in a single batch.
func (f *FilterMaps) unindexMaps(first, afterLast uint32, removeLvPtr *uint64) {
	nextBlockNumber, err := f.getMapBlockPtr(afterLast)
	if err != nil {
		log.Error("Error fetching next map block pointer", "map index", afterLast, "error", err)
		nextBlockNumber = 0 // do not remove anything
	}
	batch := f.db.NewBatch()
	for *removeLvPtr < nextBlockNumber {
		f.deleteBlockLvPointer(batch, *removeLvPtr)
		if (*removeLvPtr)%revertPointFrequency == 0 {
			rawdb.DeleteRevertPoint(batch, *removeLvPtr)
		}
		(*removeLvPtr)++
	}
	for mapIndex := first; mapIndex < afterLast; mapIndex++ {
		f.deleteMapBlockPtr(batch, mapIndex)
	}
	for rowIndex := uint32(0); rowIndex < f.mapHeight; rowIndex++ {
		for mapIndex := first; mapIndex < afterLast; mapIndex++ {
			f.storeFilterMapRow(batch, mapIndex, rowIndex, emptyRow)
		}
	}
	fmr := f.getRange()
	fmr.tailLvPointer = uint64(afterLast) << f.logValuesPerMap
	if fmr.tailLvPointer > fmr.tailBlockLvPointer {
		log.Error("Cannot unindex filter maps beyond tail block log value pointer", "tailLvPointer", fmr.tailLvPointer, "tailBlockLvPointer", fmr.tailBlockLvPointer)
		return
	}
	f.setRange(batch, fmr)
	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
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
	f.lock.RLock()
	defer f.lock.RUnlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

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
	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
	log.Info("Log index block range updated", "tail", u.tailBlockNumber, "head", u.headBlockNumber, "log values", u.headLvPointer-u.tailBlockLvPointer)
}

// updatedRangeLength returns the lenght of the updated filter map range.
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
	f.lock.RLock()
	defer f.lock.RUnlock()

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
	f.lock.Lock()
	defer f.lock.Unlock()

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
	f.setRange(batch, newRange)
	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
	return nil
}
