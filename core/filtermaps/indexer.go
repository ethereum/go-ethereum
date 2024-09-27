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
	startLvPointer       = valuesPerMap << 31 // log value index assigned to init block
	removedPointer       = math.MaxUint64     // used in updateBatch to signal removed items
	revertPointFrequency = 256                // frequency of revert points in database
	cachedRevertPoints   = 64                 // revert points for most recent blocks in memory
)

// updateLoop initializes and updates the log index structure according to the
// canonical chain.
func (f *FilterMaps) updateLoop() {
	defer f.closeWg.Done()
	f.updateMapCache()
	if rp, err := f.newUpdateBatch().makeRevertPoint(); err == nil {
		f.revertPoints[rp.blockNumber] = rp
	} else {
		log.Error("Error creating head revert point", "error", err)
	}

	var (
		headEventCh = make(chan core.ChainHeadEvent)
		sub         = f.chain.SubscribeChainHeadEvent(headEventCh)
		head        *types.Header
		stop        bool
		syncMatcher *FilterMapsMatcherBackend
	)

	defer func() {
		sub.Unsubscribe()
		if syncMatcher != nil {
			syncMatcher.synced(head)
			syncMatcher = nil
		}
	}()

	wait := func() {
		if syncMatcher != nil {
			syncMatcher.synced(head)
			syncMatcher = nil
		}
		if stop {
			return
		}
		select {
		case ev := <-headEventCh:
			head = ev.Block.Header()
		case syncMatcher = <-f.matcherSyncCh:
			head = f.chain.CurrentBlock()
		case <-time.After(time.Second * 20):
			// keep updating log index during syncing
			head = f.chain.CurrentBlock()
		case <-f.closeCh:
			stop = true
		}
	}
	for head == nil {
		wait()
		if stop {
			return
		}
	}
	fmr := f.getRange()

	for !stop {
		if !fmr.initialized {
			if !f.tryInit(head) {
				return
			}

			if syncMatcher != nil {
				syncMatcher.synced(head)
				syncMatcher = nil
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
		if syncMatcher != nil {
			syncMatcher.synced(head)
			syncMatcher = nil
		}
		// log index head is at latest chain head; process tail blocks if possible
		f.tryExtendTail(func() bool {
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
		})
		if fmr.headBlockHash == head.Hash() {
			// if tail processing exited while there is no new head then no more
			// tail blocks can be processed
			wait()
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
	receipts := rawdb.ReadRawReceipts(f.db, head.Hash(), head.Number.Uint64())
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
		receipts := rawdb.ReadRawReceipts(f.db, newHeader.Hash(), newHeader.Number.Uint64())
		if receipts == nil {
			log.Error("Could not retrieve block receipts for new block", "number", newHeader.Number, "hash", newHeader.Hash())
			break
		}
		if err := update.addBlockToHead(newHeader, receipts); err != nil {
			log.Error("Error adding new block", "number", newHeader.Number, "hash", newHeader.Hash(), "error", err)
			break
		}
		if update.updatedRangeLength() >= mapsPerEpoch {
			// limit the amount of data updated in a single batch
			f.applyUpdateBatch(update)
			update = f.newUpdateBatch()
		}
	}
	f.applyUpdateBatch(update)
	return true
}

// tryExtendTail attempts to extend the log index backwards until it indexes the
// genesis block or cannot find more block receipts. Since this is a long process,
// stopFn is called after adding each tail block and if it returns true, the
// latest batch is written and the function returns.
func (f *FilterMaps) tryExtendTail(stopFn func() bool) {
	fmr := f.getRange()
	number, parentHash := fmr.tailBlockNumber, fmr.tailParentHash
	if number == 0 {
		return
	}
	update := f.newUpdateBatch()
	lastTailEpoch := update.tailEpoch()
	for number > 0 && !stopFn() {
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
		receipts := rawdb.ReadRawReceipts(f.db, newTail.Hash(), newTail.Number.Uint64())
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
}

// updateBatch is a memory overlay collecting changes to the index log structure
// that can be written to the database in a single batch while the in-memory
// representations in FilterMaps are also updated.
type updateBatch struct {
	filterMapsRange
	maps                   map[uint32]*filterMap // nil rows are unchanged
	getFilterMapRow        func(mapIndex, rowIndex uint32) (FilterRow, error)
	blockLvPointer         map[uint64]uint64 // removedPointer means delete
	mapBlockPtr            map[uint32]uint64 // removedPointer means delete
	revertPoints           map[uint64]*revertPoint
	firstMap, afterLastMap uint32
}

// newUpdateBatch creates a new updateBatch.
func (f *FilterMaps) newUpdateBatch() *updateBatch {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return &updateBatch{
		filterMapsRange: f.filterMapsRange,
		maps:            make(map[uint32]*filterMap),
		getFilterMapRow: f.getFilterMapRow,
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
	for rowIndex := uint32(0); rowIndex < mapHeight; rowIndex++ {
		for mapIndex := u.firstMap; mapIndex < u.afterLastMap; mapIndex++ {
			if fm := u.maps[mapIndex]; fm != nil {
				if row := (*fm)[rowIndex]; row != nil {
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
				RowLength: rp.rowLength[:],
			})
		}
	}
	// update filterMapsRange
	f.setRange(batch, u.filterMapsRange)
	if err := batch.Write(); err != nil {
		log.Crit("Could not write update batch", "error", err)
	}
	log.Info("Log index block range updated", "tail", u.tailBlockNumber, "head", u.headBlockNumber, "log values", u.headLvPointer-u.tailLvPointer)
}

// updatedRangeLength returns the lenght of the updated filter map range.
func (u *updateBatch) updatedRangeLength() uint32 {
	return u.afterLastMap - u.firstMap
}

// tailEpoch returns the tail epoch index.
func (u *updateBatch) tailEpoch() uint32 {
	return uint32(u.tailLvPointer >> (logValuesPerMap + logMapsPerEpoch))
}

// getRowPtr returns a pointer to a FilterRow that can be modified. If the batch
// did not have a modified version of the given row yet, it is retrieved using the
// request function from the backing FilterMaps cache or database and copied
// before modification.
func (u *updateBatch) getRowPtr(mapIndex, rowIndex uint32) (*FilterRow, error) {
	fm := u.maps[mapIndex]
	if fm == nil {
		fm = new(filterMap)
		u.maps[mapIndex] = fm
		if mapIndex < u.firstMap || u.afterLastMap == 0 {
			u.firstMap = mapIndex
		}
		if mapIndex >= u.afterLastMap {
			u.afterLastMap = mapIndex + 1
		}
	}
	rowPtr := &(*fm)[rowIndex]
	if *rowPtr == nil {
		if filterRow, err := u.getFilterMapRow(mapIndex, rowIndex); err == nil {
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
	u.headLvPointer, u.tailLvPointer = startLvPointer, startLvPointer
	u.headBlockNumber, u.tailBlockNumber = header.Number.Uint64()-1, header.Number.Uint64() //TODO genesis?
	u.headBlockHash, u.tailParentHash = header.ParentHash, header.ParentHash
	u.addBlockToHead(header, receipts)
	return nil
}

// addValueToHead adds a single log value to the head of the log index.
func (u *updateBatch) addValueToHead(logValue common.Hash) error {
	mapIndex := uint32(u.headLvPointer >> logValuesPerMap)
	rowPtr, err := u.getRowPtr(mapIndex, rowIndex(mapIndex>>logMapsPerEpoch, logValue))
	if err != nil {
		return err
	}
	column := columnIndex(u.headLvPointer, logValue)
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
	startMap := uint32((u.headLvPointer + valuesPerMap - 1) >> logValuesPerMap)
	if err := iterateReceipts(receipts, u.addValueToHead); err != nil {
		return err
	}
	stopMap := uint32((u.headLvPointer + valuesPerMap - 1) >> logValuesPerMap)
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
	if u.tailLvPointer == 0 {
		return errors.New("tail log value pointer underflow")
	}
	u.tailLvPointer--
	mapIndex := uint32(u.tailLvPointer >> logValuesPerMap)
	rowPtr, err := u.getRowPtr(mapIndex, rowIndex(mapIndex>>logMapsPerEpoch, logValue))
	if err != nil {
		return err
	}
	column := columnIndex(u.tailLvPointer, logValue)
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
	stopMap := uint32((u.tailLvPointer + valuesPerMap - 1) >> logValuesPerMap)
	var cnt int
	if err := iterateReceiptsReverse(receipts, func(lv common.Hash) error {
		cnt++
		return u.addValueToTail(lv)
	}); err != nil {
		return err
	}
	startMap := uint32(u.tailLvPointer >> logValuesPerMap)
	for m := startMap; m < stopMap; m++ {
		u.mapBlockPtr[m] = number
	}
	u.blockLvPointer[number] = u.tailLvPointer
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
	rowLength   [mapHeight]uint
}

// makeRevertPoint creates a new revertPoint.
func (u *updateBatch) makeRevertPoint() (*revertPoint, error) {
	rp := &revertPoint{
		blockNumber: u.headBlockNumber,
		blockHash:   u.headBlockHash,
		mapIndex:    uint32(u.headLvPointer >> logValuesPerMap),
	}
	if u.tailLvPointer > uint64(rp.mapIndex)<<logValuesPerMap {
		return nil, nil
	}
	for i := range rp.rowLength[:] {
		var row FilterRow
		if m := u.maps[rp.mapIndex]; m != nil {
			row = (*m)[i]
		}
		if row == nil {
			var err error
			row, err = u.getFilterMapRow(rp.mapIndex, uint32(i))
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
	if len(rps.RowLength) != mapHeight {
		return nil, errors.New("invalid number of rows in stored revert point")
	}
	rp := &revertPoint{
		blockNumber: blockNumber,
		blockHash:   rps.BlockHash,
		mapIndex:    rps.MapIndex,
	}
	copy(rp.rowLength[:], rps.RowLength)
	return rp, nil
}

// revertTo reverts the log index to the given revert point.
func (f *FilterMaps) revertTo(rp *revertPoint) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	batch := f.db.NewBatch()
	afterLastMap := uint32((f.headLvPointer + valuesPerMap - 1) >> logValuesPerMap)
	if rp.mapIndex >= afterLastMap {
		return errors.New("cannot revert (head map behind revert point)")
	}
	lvPointer := uint64(rp.mapIndex) << logValuesPerMap
	for rowIndex, rowLen := range rp.rowLength[:] {
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
