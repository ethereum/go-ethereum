// Copyright 2025 The go-ethereum Authors
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
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var ErrOutOfRange = errors.New("pointer out of indexed range")

// mapStorage implements a filter map storage layer over mapDatabase that ensures
// efficient database usage while also providing a low latency interface to the
// indexer. It uses a memory layer over mapDatabase allowing consistently quick
// addMap and deleteMaps calls while doing the actual database updates in the
// background asynchronously.
type mapStorage struct {
	params               *Params
	mapDb                *mapDatabase
	triggerCh, closeCh   chan struct{}
	closeWg              sync.WaitGroup
	mtForceWrite, mtBusy uint32

	lock                              sync.RWMutex
	initialized                       bool
	knownEpochs                       uint32 // epochs initialized with last map block pointer and corresponding reverse block lv pointer
	knownEpochBlocks                  uint64
	valid, dirty                      rangeSet[uint32] // valid and dirty maps in database
	writeInProgress, deleteInProgress rangeSet[uint32] // write cycle in progress
	overlay                           rangeSet[uint32] // memory maps
	overlayCount                      uint32
	overlayMaps                       map[uint32]*finishedMap
	validBlocks, overlayBlocks        rangeSet[uint64]
	epochTrigger                      rangeSet[uint32]
	suspended                         uint32

	testHookCh chan bool
}

// newMapStorage creates a new mapStorage layer over the given mapDatabase.
func newMapStorage(params *Params, mapDb *mapDatabase, testHookCh chan bool) *mapStorage {
	m := &mapStorage{
		params:      params,
		mapDb:       mapDb,
		triggerCh:   make(chan struct{}, 1),
		closeCh:     make(chan struct{}),
		overlayMaps: make(map[uint32]*finishedMap),
		testHookCh:  testHookCh,

		mtForceWrite: params.rowGroupSize[0] * 9 / 8,
		mtBusy:       params.rowGroupSize[0] * 17 / 8,
	}
	if valid, dirty, knownEpochs, ok := m.mapDb.loadMapRange(); ok {
		m.valid, m.dirty, m.knownEpochs, m.initialized = valid, dirty, knownEpochs, true
		if knownEpochs > 0 {
			if lastBlock, _, err := m.mapDb.getLastBlockOfMap(m.params.lastEpochMap(knownEpochs - 1)); err == nil {
				m.knownEpochBlocks = lastBlock + 1
			} else {
				m.resetWithError(fmt.Sprintf("could not initialize known epoch range: %v", err))
			}
		}
		if err := m.updateValidBlocks(); err != nil {
			m.resetWithError(fmt.Sprintf("could not initialize valid block range: %v", err))
		}
	}
	m.closeWg.Add(1)
	go m.eventLoop()
	return m
}

// stop stops mapStorage.
func (m *mapStorage) stop() {
	close(m.closeCh)
	m.closeWg.Wait()
}

// isReady returns false if there are too many memory overlay maps at the moment.
// In this case the caller should suspend the indexing process until some maps
// are written to the database.
func (m *mapStorage) isReady() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.overlayCount < m.mtBusy {
		m.trigger()
		return true
	}
	return false
}

// tailEpoch returns the first epoch of the continuous rendered range. If it is
// larger than zero then the tail renderer can start indexing the previous epoch
// if necessary. After checkpoint initialization it points right after the last
// known epoch boundary  (in this case the tail epoch can be empty).
func (m *mapStorage) tailEpoch() uint32 {
	m.lock.Lock()
	defer m.lock.Unlock()

	mapRange := m.valid.union(m.overlay)
	if len(mapRange) > 0 && m.params.mapEpoch(mapRange[len(mapRange)-1].AfterLast()) >= m.knownEpochs {
		return min(m.knownEpochs, m.params.mapEpoch(mapRange[len(mapRange)-1].First()+m.params.mapsPerEpoch-1))
	}
	return m.knownEpochs
}

// tailNumberOfEpoch returns the number of the first block that starts in the
// given epoch.
func (m *mapStorage) tailNumberOfEpoch(epoch uint32) (uint64, error) {
	if epoch == 0 {
		return 0, nil
	}
	number, _, err := m.getLastBlockOfMap(m.params.lastEpochMap(epoch - 1))
	if err != nil {
		return 0, err
	}
	return number + 1, nil
}

// lastBoundaryBefore returns the most recent map index that is less than
// or equal to the given mapIndex parameter and is either located after a stored
// map or after a known epoch boundary.
// The returned map index position may or may not contain a stored map but if it
// is empty then it is always suitable to start a rendering process.
func (m *mapStorage) lastBoundaryBefore(mapIndex uint32) uint32 {
	m.lock.Lock()
	defer m.lock.Unlock()

	if mapIndex == 0 {
		return 0
	}
	lastBoundary := m.params.firstEpochMap(min(m.params.mapEpoch(mapIndex), m.knownEpochs))
	if m, ok := m.valid.closestLte(mapIndex - 1); ok {
		lastBoundary = max(lastBoundary, m+1)
	}
	if m, ok := m.overlay.closestLte(mapIndex - 1); ok {
		lastBoundary = max(lastBoundary, m+1)
	}
	return lastBoundary
}

// matchKnownEpochs returns true if the given list of checkpoints matches the
// checkpoints already stored in the database (always true with an empty database).
func (m *mapStorage) matchKnownEpochs(cpList checkpointList) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.knownEpochs == 0 || len(cpList) == 0 {
		return true
	}
	epoch := min(m.knownEpochs, uint32(len(cpList))) - 1
	number, hash, err := m.getLastBlockOfMap(m.params.lastEpochMap(epoch))
	if err != nil {
		m.resetWithError(fmt.Sprintf("could not read last known epoch boundary: %v", err))
		return true
	}
	return number == cpList[epoch].BlockNumber && hash == cpList[epoch].BlockHash
}

// addKnownEpochs adds the known epoch boundaries based on the given checkpoints.
func (m *mapStorage) addKnownEpochs(cpList checkpointList) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if uint32(len(cpList)) <= m.knownEpochs {
		return errors.New("checkpoint init list has no new epochs")
	}
	if m.knownEpochs > 0 {
		lastNumber, lastHash, err := m.mapDb.getLastBlockOfMap(m.params.lastEpochMap(m.knownEpochs - 1))
		if err != nil {
			return err
		}
		lvPointer, err := m.mapDb.getBlockLvPointer(lastNumber)
		if err != nil {
			return err
		}
		if cp := cpList[m.knownEpochs-1]; cp.BlockNumber != lastNumber || cp.BlockHash != lastHash || cp.FirstIndex != lvPointer {
			return errors.New("checkpoint init list does not match known epochs")
		}
	}

	m.mapDb.storeCheckpointList(m.knownEpochs, cpList[m.knownEpochs:])
	m.knownEpochs = uint32(len(cpList))
	m.knownEpochBlocks = cpList[len(cpList)-1].BlockNumber + 1
	m.mapDb.storeMapRange(m.valid, m.dirty, m.knownEpochs)
	return nil
}

// addMap adds a new map to the storage. If forceCommit is true then a write is
// always triggered. If it is false then a write is only triggered when a row
// group boundary is reached or if the total number of memory maps reaches a limit.
// addMap always returns right after adding the new map to the memory layer.
func (m *mapStorage) addMap(mapIndex uint32, fm *finishedMap, forceCommit bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if fm == nil {
		panic("trying to add nil map")
	}
	if m.valid.includes(mapIndex) || m.overlay.includes(mapIndex) {
		panic("addMap to non-empty map index")
	}
	epoch := m.params.mapEpoch(mapIndex)
	if (epoch > m.knownEpochs || mapIndex != m.params.firstEpochMap(epoch)) &&
		!m.valid.includes(mapIndex-1) && !m.overlay.includes(mapIndex-1) {
		panic("addMap to map index with no known boundary")
	}
	mapRs := singleRangeSet[uint32](common.NewRange[uint32](mapIndex, 1))
	m.overlay = m.overlay.union(mapRs)
	m.writeInProgress = m.writeInProgress.exclude(mapRs)
	m.overlayMaps[mapIndex] = fm
	m.updateOverlayBlocks()
	if forceCommit || (mapIndex+1)%m.params.rowGroupSize[0] == 0 {
		m.epochTrigger = m.epochTrigger.union(singleRangeSet[uint32](common.NewRange[uint32](epoch, 1)))
		m.trigger()
	}
}

// deleteMaps deletes the given map range. Note that similarly to addMap, it only
// performs memory operations before returning. The deleted map range is marked
// dirty, then later the database update goroutine will actually delete or
// overwrite the dirty maps.
func (m *mapStorage) deleteMaps(maps common.Range[uint32]) {
	m.lock.Lock()
	defer m.lock.Unlock()

	dr := singleRangeSet[uint32](maps)
	for i := range dr.intersection(m.overlay).iter() {
		delete(m.overlayMaps, i)
	}
	m.writeInProgress = m.writeInProgress.exclude(dr)
	knownEpochs := m.knownEpochs
	if maps.Includes(m.params.firstEpochMap(knownEpochs)) {
		knownEpochs = m.params.mapEpoch(maps.First())
	}
	if err := m.updateRange(m.valid.exclude(dr), m.dirty.union(dr.intersection(m.valid)), m.overlay.exclude(dr), knownEpochs); err != nil {
		m.resetWithError(fmt.Sprintf("could not revert valid block range: %v", err))
	}
	m.trigger()
}

// suspendOrResume suspends or resumes the background database update operations.
func (m *mapStorage) suspendOrResume(suspend bool) {
	var suspendU32 uint32
	if suspend {
		suspendU32 = 1
	}
	old := atomic.SwapUint32(&m.suspended, suspendU32)
	if suspendU32 < old {
		m.trigger()
	}
}

// eventLoop is the main event loop of the database update operations.
func (m *mapStorage) eventLoop() {
	defer m.closeWg.Done()

	var stopped bool

	selectEvent := func(blocking bool) {
		if m.testHookCh != nil {
			select {
			case <-m.closeCh:
				stopped = true
				return
			case m.testHookCh <- blocking:
			}
		}
		if blocking {
			select {
			case <-m.closeCh:
				stopped = true
				return
			case <-m.triggerCh:
			}
		} else {
			select {
			case <-m.closeCh:
				stopped = true
				return
			case <-m.triggerCh:
			default:
			}
		}
		if m.testHookCh != nil {
			select {
			case <-m.closeCh:
				stopped = true
				return
			case <-m.testHookCh:
			}
		}
	}

	stopCallback := func() bool {
		selectEvent(atomic.LoadUint32(&m.suspended) == 1)
		return stopped
	}

	for !stopped {
		if !m.initialized {
			if done, _ := m.mapDb.reset(stopCallback); !done {
				return // node stopped before cleaning old database
			}
			m.lock.Lock()
			m.initialized = true
			m.lock.Unlock()
		}
		more, err := m.doWriteCycle(stopCallback)
		if err != nil {
			m.resetWithError(fmt.Sprintf("write cycle failed: %v", err))
			continue
		}
		selectEvent(!more && !stopped) // wait for next event if no changes done
	}
}

// trigger ensures that the database update cycle will restart if it was in a
// waiting state.
func (m *mapStorage) trigger() {
	select {
	case m.triggerCh <- struct{}{}:
	default:
	}
}

// getBlockLvPointer returns the starting log value index where the log values
// generated by the given block are located.
func (m *mapStorage) getBlockLvPointer(blockNumber uint64) (uint64, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlayBlocks.includes(blockNumber) {
		for _, fm := range m.overlayMaps {
			if fm.blocks().Includes(blockNumber) {
				return fm.blockPtrs[blockNumber-fm.firstBlock()], nil
			}
		}
		return 0, errors.New("memory overlay block pointer not found")
	}
	if blockNumber < m.knownEpochBlocks || m.validBlocks.includes(blockNumber) {
		return m.mapDb.getBlockLvPointer(blockNumber)
	}
	return 0, ErrOutOfRange
}

// getLastBlockOfMap returns the number and hash of the block that generated the
// last log value entry of the given map.
func (m *mapStorage) getLastBlockOfMap(mapIndex uint32) (uint64, common.Hash, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlay.includes(mapIndex) {
		fm := m.overlayMaps[mapIndex]
		if fm == nil {
			return 0, common.Hash{}, errors.New("memory overlay map not found")
		}
		return fm.lastBlock.number, fm.lastBlock.hash, nil
	}
	if mapIndex < m.params.firstEpochMap(m.knownEpochs) || m.valid.includes(mapIndex) || m.valid.includes(mapIndex+1) {
		return m.mapDb.getLastBlockOfMap(mapIndex)
	}
	return 0, common.Hash{}, ErrOutOfRange
}

// getFilterMapRows returns a batch of filter maps rows from the same row index,
// each truncated to the length limit of the specified mapping layer.
// The function assumes that the map indices are in strictly ascending order.
func (m *mapStorage) getFilterMapRows(mapIndices []uint32, rowIndex, layerIndex uint32) ([]FilterRow, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rows := make([]FilterRow, len(mapIndices))
	dbMaps := make([]uint32, 0, len(mapIndices))
	for i, mapIndex := range mapIndices {
		if m.overlay.includes(mapIndex) {
			fm := m.overlayMaps[mapIndex]
			if fm == nil {
				return nil, errors.New("memory overlay map not found")
			}
			rows[i] = fm.getRow(rowIndex, m.params.getMaxRowLength(layerIndex))
		} else {
			dbMaps = append(dbMaps, mapIndex)
		}
	}
	dbRows, err := m.mapDb.getFilterMapRows(dbMaps, rowIndex, layerIndex)
	if err != nil {
		return nil, err
	}
	var j int
	for i, row := range rows {
		if row == nil { // zero length row is represented as zero length slice
			rows[i] = dbRows[j]
			j++
		}
	}
	if j != len(dbMaps) {
		panic("rows length mismatch")
	}
	return rows, nil
}

// getFilterMap returns the filter map at the specified index.
func (m *mapStorage) getFilterMap(mapIndex uint32) (*finishedMap, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlay.includes(mapIndex) {
		fm := m.overlayMaps[mapIndex]
		if fm == nil {
			return nil, errors.New("memory overlay map not found")
		}
		return fm, nil
	}
	if m.valid.includes(mapIndex) {
		return m.mapDb.getFilterMap(mapIndex)
	}
	return nil, nil
}

// extendDeletedPointerRange takes a map range where pointers need to be deleted
// and extends it so that on both ends there is a neighboring stored map, a
// known epoch boundary or one end of the map index range. This is required
// in order to determine the block range covered by the deleted map range.
func (m *mapStorage) extendDeletedPointerRange(deleteRange common.Range[uint32]) common.Range[uint32] {
	epoch := m.params.mapEpoch(deleteRange.First())
	if m.params.mapEpoch(deleteRange.Last()) != epoch {
		panic("deleted map range crosses epoch boundary")
	}
	first := m.params.firstEpochMap(min(m.knownEpochs, epoch))
	if deleteRange.First() > 0 {
		if c, ok := m.valid.closestLte(deleteRange.First() - 1); ok {
			first = max(first, c+1)
		}
	}
	afterLast := uint32(math.MaxUint32)
	if epoch < m.knownEpochs {
		afterLast = m.params.firstEpochMap(epoch + 1)
	}
	if fa, ok := m.valid.closestGte(deleteRange.AfterLast()); ok {
		afterLast = min(afterLast, fa)
	}
	return common.NewRange[uint32](first, afterLast-first)
}

// selectEpochTriggeredWrite selects the next epoch to update if one of the epochs
// in the canSelect list has been triggered by addMap.
func (m *mapStorage) selectEpochTriggeredWrite(canSelect []uint32) (uint32, bool) {
	for _, epoch := range canSelect {
		if !m.epochTrigger.includes(epoch) {
			continue
		}
		m.epochTrigger = m.epochTrigger.exclude(singleRangeSet[uint32](common.NewRange[uint32](epoch, 1)))
		epochRange := common.NewRange[uint32](m.params.firstEpochMap(epoch), m.params.mapsPerEpoch)
		if len(m.overlay.intersection(singleRangeSet[uint32](epochRange))) > 0 {
			return epoch, true
		}
	}
	return 0, false
}

// selectEpochForcedWrite selects the next epoch to update if the total number
// of memory maps has reached a threshold.
func (m *mapStorage) selectEpochForcedWrite(canSelect []uint32) (uint32, bool) {
	if m.overlayCount < m.mtForceWrite {
		return 0, false
	}
	var best, count uint32
	for _, epoch := range canSelect {
		epochRange := common.NewRange[uint32](m.params.firstEpochMap(epoch), m.params.mapsPerEpoch)
		if c := m.overlay.intersection(singleRangeSet[uint32](epochRange)).count(); c > count {
			best, count = epoch, c
		}
	}
	if count == 0 {
		return 0, false
	}
	return best, true
}

// mapToEpochRange returns the set of epochs that are either fully or partially
// covered by the specified mapRange.
func (m *mapStorage) mapToEpochRange(mapRange rangeSet[uint32]) rangeSet[uint32] {
	vb := make(rangeBoundaries[uint32], 0, len(mapRange)*2)
	for _, r := range mapRange {
		first := m.params.mapEpoch(r.First())
		last := m.params.mapEpoch(r.Last())
		vb.add(common.NewRange[uint32](first, last+1-first), 1)
	}
	return vb.makeSet(1)
}

// selectEpochDeleteOnly selects an epoch to update where there are no memory
// maps to write, only dirty maps to clean up.
func (m *mapStorage) selectEpochDeleteOnly() (uint32, bool) {
	epochs := m.mapToEpochRange(m.dirty).exclude(m.mapToEpochRange(m.overlay))
	if len(epochs) == 0 {
		return 0, false
	}
	return epochs[0].First(), true
}

// doWriteCycle selects an epoch where there are memory overlay maps and/or
// dirty maps in the underlying database, cleans up dirty data and writes the
// new maps to the database.
// If an epoch has been successfully updated then the function returns (true, nil).
// In this case a new write cycle can be attempted immediately until there is
// nothing more left to do.
// If there was nothing to do or if stopCallback aborted the operation then the
// function returns (false, nil). In case of an error it returns (false, err).
// Note that if the operation is aborted or failed then the partially updated
// maps are left as marked dirty and the database remains consistent.
func (m *mapStorage) doWriteCycle(stopCallback func() bool) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// always operate on a single epoch
	var canSelect []uint32
	if len(m.valid) == 0 {
		canSelect = []uint32{m.knownEpochs}
	} else {
		lastValid := m.valid[len(m.valid)-1]
		tailEpoch := m.params.mapEpoch(lastValid.First())
		headEpoch := m.params.mapEpoch(lastValid.AfterLast())
		if tailEpoch > 0 {
			canSelect = []uint32{headEpoch, tailEpoch - 1}
		} else {
			canSelect = []uint32{headEpoch}
		}
	}

	epoch, ok := m.selectEpochTriggeredWrite(canSelect)
	if !ok {
		epoch, ok = m.selectEpochForcedWrite(canSelect)
		if !ok {
			epoch, ok = m.selectEpochDeleteOnly()
			if !ok {
				return false, nil
			}
		}
	}
	epochRange := singleRangeSet[uint32](common.NewRange[uint32](m.params.firstEpochMap(epoch), m.params.mapsPerEpoch))
	writeMaps := epochRange.intersection(m.overlay).singleRange()
	validInEpoch := epochRange.intersection(m.valid).singleRange()
	dirtyInEpoch := epochRange.intersection(m.dirty).singleRange()
	keepEmptyFrom := max(m.params.firstEpochMap(epoch), writeMaps.AfterLast(), validInEpoch.AfterLast(), dirtyInEpoch.AfterLast())
	keepEmptyInEpoch := common.NewRange[uint32](keepEmptyFrom, m.params.firstEpochMap(epoch+1)-keepEmptyFrom)
	// delete old pointers
	if !dirtyInEpoch.IsEmpty() {
		m.mapDb.deletePointers(m.extendDeletedPointerRange(dirtyInEpoch), stopCallback)
	}
	if writeMaps.IsEmpty() && validInEpoch.IsEmpty() {
		// delete map rows of entire epoch if nothing to write or keep
		m.lock.Unlock()
		done, err := m.mapDb.deleteEpochRows(epoch, stopCallback)
		m.lock.Lock()
		if done {
			if err := m.updateRange(m.valid, m.dirty.exclude(epochRange), m.overlay, m.knownEpochs); err != nil {
				return false, err
			}
		}
		return done, err
	}
	maps := make([]*finishedMap, writeMaps.Count())
	for i := range writeMaps.Iter() {
		maps[i-writeMaps.First()] = m.overlayMaps[i]
	}
	// temporarily mark newly written maps as dirty (replaced/deleted maps are already dirty)
	writeInProgress := singleRangeSet[uint32](writeMaps)
	deleteInProgress := singleRangeSet[uint32](dirtyInEpoch).exclude(writeInProgress)
	if err := m.updateRange(m.valid, m.dirty.union(writeInProgress), m.overlay, m.knownEpochs); err != nil {
		return false, err
	}
	m.writeInProgress = writeInProgress
	m.deleteInProgress = deleteInProgress
	m.lock.Unlock()
	// write/overwrite map rows and delete dirty map data, write new pointers
	done, err := m.mapDb.writeMapRows(writeMaps, dirtyInEpoch, keepEmptyInEpoch, maps, stopCallback)
	if done {
		done, err = m.mapDb.writePointers(writeMaps, maps, stopCallback)
	}
	m.lock.Lock()
	writeInProgress = m.writeInProgress
	m.writeInProgress, m.deleteInProgress = nil, nil
	if !done {
		return false, err
	}
	for mapIndex := range writeInProgress.iter() {
		delete(m.overlayMaps, mapIndex)
	}
	knownEpochs := m.knownEpochs
	if len(writeInProgress) > 0 {
		knownEpochs = max(knownEpochs, m.params.mapEpoch(writeInProgress[len(writeInProgress)-1].Last()))
	}
	if err := m.updateRange(m.valid.union(writeInProgress), m.dirty.exclude(writeInProgress.union(deleteInProgress)), m.overlay.exclude(writeInProgress), knownEpochs); err != nil {
		return false, err
	}
	return true, nil
}

// updateRange updates the stored valid, dirty and memory overlay map ranges and
// the number of known epoch boundaries. It also stores these ranges in the
// database and determines the corresponding block ranges.
func (m *mapStorage) updateRange(valid, dirty, overlay rangeSet[uint32], knownEpochs uint32) error {
	if !valid.equal(m.valid) {
		m.valid = valid
		if err := m.updateValidBlocks(); err != nil {
			return err
		}
	}
	if knownEpochs != m.knownEpochs {
		m.knownEpochs = knownEpochs
		if err := m.updateKnownEpochBlocks(); err != nil {
			return err
		}
	}
	m.dirty = dirty
	if !overlay.equal(m.overlay) {
		m.overlay = overlay
		m.updateOverlayBlocks()
	}
	m.mapDb.storeMapRange(valid, dirty, m.knownEpochs)
	return nil
}

// resetWithError resets an invalid log index database after an unrecoverable error.
func (m *mapStorage) resetWithError(errStr string) {
	log.Error("Resetting invalid log index database", "error", errStr)
	m.uninitialize()
}

// uninitialize resets the map storage and returns to its non-initialized state.
func (m *mapStorage) uninitialize() {
	m.valid, m.dirty, m.overlay, m.knownEpochs, m.initialized = nil, nil, nil, 0, false
	m.mapDb.deleteMapRange()
	m.trigger()
}

// updateKnownEpochBlocks determines the block number belonging to the last known
// epoch boundary.
func (m *mapStorage) updateKnownEpochBlocks() error {
	if m.knownEpochs > 0 {
		if lastBlock, _, err := m.mapDb.getLastBlockOfMap(m.params.lastEpochMap(m.knownEpochs - 1)); err == nil {
			m.knownEpochBlocks = lastBlock + 1
		} else {
			return err
		}
	} else {
		m.knownEpochBlocks = 0
	}
	return nil
}

// updateValidBlocks determines the the set of blocks where the starting log value
// pointer points into the valid map set.
func (m *mapStorage) updateValidBlocks() error {
	if len(m.valid) > 2 || (len(m.valid) == 2 && m.valid[0].Count() >= m.params.mapsPerEpoch) {
		panic("invalid mapStorage.valid")
		return errors.New("invalid mapStorage.valid range set")
	}
	vb := make(rangeBoundaries[uint64], 0, len(m.valid)*2)
	for _, vr := range m.valid {
		var first uint64
		if vr.First() > 0 {
			lb, _, err := m.mapDb.getLastBlockOfMap(vr.First() - 1)
			if err != nil {
				return err
			}
			first = lb + 1
		}
		last, _, err := m.mapDb.getLastBlockOfMap(vr.Last())
		if err != nil {
			return err
		}
		vb.add(common.NewRange[uint64](first, last+1-first), 1)
	}
	m.validBlocks = vb.makeSet(1)
	return nil
}

// updateOverlayBlocks determines the the set of blocks where the starting log
// value pointer points into the memory overlay map set.
func (m *mapStorage) updateOverlayBlocks() {
	ob := make(rangeBoundaries[uint64], 0, len(m.overlay)*2)
	for _, or := range m.overlay {
		first := m.overlayMaps[or.First()].firstBlock()
		last := m.overlayMaps[or.Last()].lastBlock.number
		ob.add(common.NewRange[uint64](first, last+1-first), 1)
	}
	m.overlayBlocks = ob.makeSet(1)
	m.overlayCount = m.overlay.count()
}
