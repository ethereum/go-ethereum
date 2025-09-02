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

type mapStorage struct {
	params               *Params
	mapDb                *mapDatabase
	triggerCh, closeCh   chan struct{}
	closeWg              sync.WaitGroup
	mtForceWrite, mtBusy uint32

	lock                       sync.RWMutex
	initialized                bool
	knownEpochs                uint32 // epochs initialized with last map block pointer and corresponding reverse block lv pointer
	knownEpochBlocks           uint64
	valid, dirty               rangeSet[uint32] // valid and dirty maps in database
	overlay                    rangeSet[uint32] // memory maps
	overlayCount               uint32
	validBlocks, overlayBlocks rangeSet[uint64]
	writeEpochs                rangeSet[uint32]
	maps                       map[uint32]*finishedMap
	suspended                  uint32
}

func newMapStorage(params *Params, mapDb *mapDatabase) *mapStorage {
	m := &mapStorage{
		params:    params,
		mapDb:     mapDb,
		triggerCh: make(chan struct{}, 1),
		closeCh:   make(chan struct{}),
		maps:      make(map[uint32]*finishedMap),

		mtForceWrite: params.rowGroupSize[0] * 9 / 8,
		mtBusy:       params.rowGroupSize[0] * 10 / 8,
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
		if err := m.validUpdated(); err != nil {
			m.resetWithError(fmt.Sprintf("could not initialize valid block range: %v", err))
		}
	}
	//fmt.Println("newMapStorage")
	//fmt.Println(" valid:", m.valid)
	//fmt.Println(" validBlocks:", m.validBlocks)
	//fmt.Println(" dirty:", m.dirty)
	//fmt.Println(" knownEpochs:", m.knownEpochs)
	//fmt.Println(" knownEpochBlocks:", m.knownEpochBlocks)
	m.closeWg.Add(1)
	go m.eventLoop()
	return m
}

func (m *mapStorage) stop() {
	close(m.closeCh)
	m.closeWg.Wait()
}

func (m *mapStorage) isReady() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.overlayCount < m.mtBusy
}

func (m *mapStorage) tailEpoch() uint32 {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.valid) > 0 && m.params.mapEpoch(m.valid[len(m.valid)-1].AfterLast()) >= m.knownEpochs {
		return min(m.knownEpochs, m.params.mapEpoch(m.valid[len(m.valid)-1].First()+m.params.mapsPerEpoch-1))
	}
	return m.knownEpochs
}

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

func (m *mapStorage) canExtendKnownEpochs(cpList checkpointList) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return uint32(len(cpList)) > m.knownEpochs
}

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

func (m *mapStorage) addKnownEpochs(cpList checkpointList) error {
	//fmt.Println("/addKnownEpochs", m.knownEpochs, len(cpList))
	//defer fmt.Println("\\addKnownEpochs")

	m.lock.Lock()
	defer m.lock.Unlock()

	if uint32(len(cpList)) <= m.knownEpochs {
		return errors.New("checkpoint init list has no new epochs")
	}
	if m.knownEpochs > 0 {
		lastNumber, lastHash, err := m.mapDb.getLastBlockOfMap(m.params.lastEpochMap(m.knownEpochs - 1))
		if err != nil {
			return err //TODO fmt.Errorf
		}
		lvPointer, err := m.mapDb.getBlockLvPointer(lastNumber)
		if err != nil {
			return err //TODO fmt.Errorf
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

func (m *mapStorage) addMap(mapIndex uint32, fm *finishedMap, forceCommit bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if fm == nil {
		panic("trying to add nil map")
	}
	//fmt.Println("addMap", mapIndex, forceCommit, m.overlay.count())
	if m.valid.includes(mapIndex) || m.overlay.includes(mapIndex) {
		panic("addMap to non-empty map index")
	}
	epoch := m.params.mapEpoch(mapIndex)
	if (epoch > m.knownEpochs || mapIndex != m.params.firstEpochMap(epoch)) &&
		!m.valid.includes(mapIndex-1) && !m.overlay.includes(mapIndex-1) {
		panic("addMap to map index with no known boundary")
	}
	m.overlay = m.overlay.union(rangeSet[uint32]{common.NewRange[uint32](mapIndex, 1)})
	m.maps[mapIndex] = fm
	m.overlayUpdated()
	if epoch >= m.knownEpochs && mapIndex == m.params.lastEpochMap(epoch) {
		m.knownEpochs = epoch + 1
		m.knownEpochBlocks = fm.lastBlock.number + 1
	}
	if forceCommit || (mapIndex+1)%m.params.rowGroupSize[0] == 0 {
		m.writeEpochs = m.writeEpochs.union(rangeSet[uint32]{common.NewRange[uint32](epoch, 1)})
	}
	m.trigger()
}

func (m *mapStorage) deleteMaps(maps common.Range[uint32]) {
	m.lock.Lock()
	defer m.lock.Unlock()

	//fmt.Println("deleteMaps", maps)
	dr := rangeSet[uint32]{maps}
	for i := range dr.intersection(m.overlay).iter() {
		delete(m.maps, i)
	}
	//fmt.Println(" overlay before", m.overlay)
	m.overlay = m.overlay.exclude(dr)
	//fmt.Println(" overlay after", m.overlay)
	m.overlayUpdated()
	m.dirty = m.dirty.union(dr.intersection(m.valid))
	m.valid = m.valid.exclude(dr)
	if m.params.mapEpoch(maps.AfterLast()) >= m.knownEpochs {
		if epochs := m.params.mapEpoch(maps.First()); epochs < m.knownEpochs {
			m.knownEpochs = epochs
			if epochs > 0 {
				last, _, err := m.mapDb.getLastBlockOfMap(m.params.lastEpochMap(epochs - 1))
				if err != nil {
					m.resetWithError(fmt.Sprintf("could not revert valid block range: %v", err))
					m.trigger()
					return
				}
				m.knownEpochBlocks = last + 1
			} else {
				m.knownEpochBlocks = 0
			}
		}
	}
	if err := m.validUpdated(); err != nil {
		m.resetWithError(fmt.Sprintf("could not revert valid block range: %v", err))
	}
	m.trigger()
}

func (m *mapStorage) suspendOrResume(suspend bool) {
	//fmt.Println("suspendOrResume", suspend)
	var suspendU32 uint32
	if suspend {
		suspendU32 = 1
	}
	old := atomic.SwapUint32(&m.suspended, suspendU32)
	if suspendU32 < old {
		m.trigger()
	}
}

func (m *mapStorage) eventLoop() {
	defer m.closeWg.Done()

	var stopped bool

	blockingSelect := func() {
		select {
		case <-m.closeCh:
			stopped = true
			//fmt.Println("STOPPED")
		case <-m.triggerCh:
		}
	}

	nonBlockingSelect := func() {
		select {
		case <-m.closeCh:
			stopped = true
			//fmt.Println("STOPPED")
		case <-m.triggerCh:
		default:
		}
	}

	stopCallback := func() bool {
		if atomic.LoadUint32(&m.suspended) == 1 {
			blockingSelect()
		} else {
			nonBlockingSelect()
		}
		return stopped
	}

	for !stopped {
		//fmt.Println("e1")
		if !m.initialized {
			if done, _ := m.mapDb.reset(stopCallback); !done {
				return // node stopped before cleaning old database
			}
			m.lock.Lock()
			m.initialized = true
			m.lock.Unlock()
		}
		//fmt.Println("e2")
		done, err := m.doWriteCycle(stopCallback)
		//fmt.Println("e3", done, err)
		if err != nil {
			m.resetWithError(fmt.Sprintf("could not read last known epoch boundary: %v", err))
			continue
		}
		//fmt.Println("e4")
		if !done && !stopped { // wait for next event if no changes done
			blockingSelect()
		} else {
			nonBlockingSelect()
		}
		//fmt.Println("e5")
	}
	//fmt.Println("e6")
}

func (m *mapStorage) trigger() {
	select {
	case m.triggerCh <- struct{}{}:
	default:
	}
}

func (m *mapStorage) getBlockLvPointer(blockNumber uint64) (uint64, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlayBlocks.includes(blockNumber) {
		for _, fm := range m.maps { //TODO ??optimize with binary search?
			if fm.blocks().Includes(blockNumber) {
				return fm.blockPtrs[blockNumber-fm.firstBlock()], nil
			}
		}
		return 0, errors.New("memory overlay block pointer not found")
	}
	if blockNumber < m.knownEpochBlocks || m.validBlocks.includes(blockNumber) {
		return m.mapDb.getBlockLvPointer(blockNumber)
	}
	return 0, errors.New("block log value pointer not found")
}

func (m *mapStorage) getLastBlockOfMap(mapIndex uint32) (uint64, common.Hash, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlay.includes(mapIndex) {
		fm := m.maps[mapIndex]
		if fm == nil {
			return 0, common.Hash{}, errors.New("memory overlay map not found")
		}
		return fm.lastBlock.number, fm.lastBlock.hash, nil
	}
	if mapIndex < m.params.firstEpochMap(m.knownEpochs) || m.valid.includes(mapIndex) || m.valid.includes(mapIndex+1) {
		return m.mapDb.getLastBlockOfMap(mapIndex)
	}
	return 0, common.Hash{}, errors.New("last block of map not found")
}

func (m *mapStorage) getFilterMapRows(mapIndices []uint32, rowIndex, layers uint32) ([]FilterRow, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	rows := make([]FilterRow, len(mapIndices))
	dbMaps := make([]uint32, 0, len(mapIndices))
	for i, mapIndex := range mapIndices {
		if m.overlay.includes(mapIndex) {
			fm := m.maps[mapIndex]
			if fm == nil {
				return nil, errors.New("memory overlay map not found") //TODO fmt.Errorf...
			}
			rows[i] = fm.getRow(rowIndex, m.params.getMaxRowLength(layers))
		} else {
			dbMaps = append(dbMaps, mapIndex)
		}
	}
	dbRows, err := m.mapDb.getFilterMapRows(dbMaps, rowIndex, layers)
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
	if j != len(mapIndices) {
		panic("rows length mismatch")
	}
	return rows, nil
}

// returns nil, nil if map is unknown
func (m *mapStorage) getFilterMap(mapIndex uint32) (*finishedMap, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.overlay.includes(mapIndex) {
		fm := m.maps[mapIndex]
		if fm == nil {
			return nil, errors.New("memory overlay map not found") //TODO fmt.Errorf...
		}
		return fm, nil
	}
	if m.valid.includes(mapIndex) {
		return m.mapDb.getFilterMap(mapIndex)
	}
	return nil, nil
}

func (m *mapStorage) extendPointerRange(deleteRange common.Range[uint32]) common.Range[uint32] {
	epoch := deleteRange.First() >> m.params.logMapsPerEpoch
	if deleteRange.Last()>>m.params.logMapsPerEpoch != epoch {
		panic("deleted map range crosses epoch boundary")
	}
	first := min(m.knownEpochs, epoch) << m.params.logMapsPerEpoch
	if deleteRange.First() > 0 {
		if c, ok := m.valid.closestLte(deleteRange.First() - 1); ok {
			first = max(first, c+1)
		}
	}
	afterLast := uint32(math.MaxUint32)
	if epoch < m.knownEpochs {
		afterLast = (epoch + 1) << m.params.logMapsPerEpoch
	}
	if fa, ok := m.valid.closestGte(deleteRange.AfterLast()); ok {
		afterLast = min(afterLast, fa)
	}
	return common.NewRange[uint32](first, afterLast-first)
}

func (m *mapStorage) selectEpochTriggeredWrite() (uint32, bool) {
	if len(m.writeEpochs) == 0 {
		return 0, false
	}
	//fmt.Println("selectEpochTriggeredWrite", m.writeEpochs)
	epoch := m.writeEpochs[len(m.writeEpochs)-1].First()
	m.writeEpochs = m.writeEpochs.exclude(rangeSet[uint32]{common.NewRange[uint32](epoch, 1)})
	return epoch, true
}

func (m *mapStorage) selectEpochForcedWrite() (uint32, bool) {
	if m.overlayCount < m.mtForceWrite {
		return 0, false
	}
	var longest common.Range[uint32]
	for _, r := range m.overlay {
		if r.Count() > longest.Count() {
			longest = r
		}
	}
	//fmt.Println("selectEpochForcedWrite", m.overlay)
	return m.params.mapEpoch(longest.First()), true
}

func (m *mapStorage) mapToEpochRange(mapRange rangeSet[uint32]) rangeSet[uint32] {
	vb := make(rangeBoundaries[uint32], 0, len(mapRange)*2)
	for _, r := range mapRange {
		first := m.params.mapEpoch(r.First())
		last := m.params.mapEpoch(r.Last())
		vb.add(common.NewRange[uint32](first, last+1-first), 1)
	}
	return vb.makeSet(1)
}

func (m *mapStorage) selectEpochOnlyDirty() (uint32, bool) {
	epochs := m.mapToEpochRange(m.dirty).exclude(m.mapToEpochRange(m.overlay))
	if len(epochs) == 0 {
		return 0, false
	}
	//fmt.Println("selectEpochForcedWrite", epochs, m.dirty, m.overlay)
	return epochs[0].First(), true
}

func (m *mapStorage) doWriteCycle(stopCallback func() bool) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// always operate on a single epoch
	epoch, ok := m.selectEpochTriggeredWrite()
	if !ok {
		epoch, ok = m.selectEpochForcedWrite()
		if !ok {
			epoch, ok = m.selectEpochOnlyDirty()
			if !ok {
				return true, nil
			}
		}
	}
	//TODO wait for group boundary unless last map is added
	epochRange := rangeSet[uint32]{common.NewRange[uint32](m.params.firstEpochMap(epoch), m.params.mapsPerEpoch)}
	writeMaps := epochRange.intersection(m.overlay).singleRange()
	validInEpoch := epochRange.intersection(m.valid).singleRange()
	dirtyInEpoch := epochRange.intersection(m.dirty).singleRange()
	keepEmptyFrom := max(m.params.firstEpochMap(epoch), writeMaps.First(), validInEpoch.First(), dirtyInEpoch.First())
	keepEmptyInEpoch := common.NewRange[uint32](keepEmptyFrom, m.params.firstEpochMap(epoch+1)-keepEmptyFrom)
	//fmt.Println("* epoch", epoch, epochRange)
	//fmt.Println("* writeMaps", writeMaps, m.overlay)
	//fmt.Println("* validInEpoch", validInEpoch, m.valid)
	//fmt.Println("* dirtyInEpoch", dirtyInEpoch, m.dirty)
	// delete old pointers
	if !dirtyInEpoch.IsEmpty() {
		m.mapDb.deletePointers(m.extendPointerRange(dirtyInEpoch), stopCallback)
	}
	if writeMaps.IsEmpty() && validInEpoch.IsEmpty() {
		// delete map rows of entire epoch if nothing to write or keep
		m.lock.Unlock()
		done, err := m.mapDb.deleteEpochRows(epoch, stopCallback)
		m.lock.Lock()
		if done {
			if err := m.updateRange(m.valid, m.dirty.exclude(epochRange), m.overlay); err != nil {
				return false, err
			}
		}
		return done, err
	}
	maps := make([]*finishedMap, writeMaps.Count())
	for i := range writeMaps.Iter() {
		maps[i-writeMaps.First()] = m.maps[i]
	}
	// temporarily mark newly written maps as dirty (replaced/deleted maps are already dirty)
	if err := m.updateRange(m.valid, m.dirty.union(rangeSet[uint32]{writeMaps}), m.overlay); err != nil {
		return false, err
	}
	m.lock.Unlock()
	// write/overwrite map rows and delete dirty map data, write new pointers
	done, err := m.mapDb.writeMaps(writeMaps, dirtyInEpoch, keepEmptyInEpoch, maps, stopCallback)
	m.lock.Lock()
	if !done {
		return false, err
	}
	// check if newly written maps are still valid according to the current memory
	// map overlay and shorten range if some maps have been invalidated
	for mapIndex := range writeMaps.Iter() {
		if m.maps[mapIndex] != maps[mapIndex-writeMaps.First()] {
			writeMaps = common.NewRange[uint32](writeMaps.First(), mapIndex-writeMaps.First())
			break
		}
		delete(m.maps, mapIndex)
	}
	writeMapsRs := rangeSet[uint32]{writeMaps}
	if err := m.updateRange(m.valid.union(writeMapsRs), m.dirty.exclude(writeMapsRs), m.overlay.exclude(writeMapsRs)); err != nil {
		return false, err
	}
	return true, nil
}

func (m *mapStorage) updateRange(valid, dirty, overlay rangeSet[uint32]) error {
	if !valid.equal(m.valid) {
		m.valid = valid
		if err := m.validUpdated(); err != nil {
			return err
		}
	}
	m.dirty = dirty
	if !overlay.equal(m.overlay) {
		m.overlay = overlay
		m.overlayUpdated()
	}
	m.mapDb.storeMapRange(valid, dirty, m.knownEpochs)
	return nil
}

func (m *mapStorage) resetWithError(errStr string) {
	log.Error("Resetting invalid log index database", "error", errStr)
	m.uninitialize()
}

func (m *mapStorage) uninitialize() {
	m.valid, m.dirty, m.overlay, m.knownEpochs, m.initialized = nil, nil, nil, 0, false
	m.mapDb.deleteMapRange()
	m.trigger()
}

func (m *mapStorage) validUpdated() error {
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

func (m *mapStorage) overlayUpdated() {
	ob := make(rangeBoundaries[uint64], 0, len(m.overlay)*2)
	for _, or := range m.overlay {
		first := m.maps[or.First()].firstBlock()
		last := m.maps[or.Last()].lastBlock.number
		ob.add(common.NewRange[uint64](first, last+1-first), 1)
	}
	m.overlayBlocks = ob.makeSet(1)
	m.overlayCount = m.overlay.count()
}
