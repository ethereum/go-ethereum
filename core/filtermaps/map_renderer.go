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
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	maxMapsPerBatch   = 32    // maximum number of maps rendered in memory
	valuesPerCallback = 1024  // log values processed per event process callback
	cachedRowMappings = 10000 // log value to row mappings cached during rendering

	// Number of rows written to db in a single batch.
	// The map renderer splits up writes like this to ensure that regular
	// block processing latency is not affected by large batch writes.
	rowsPerBatch = 1024
)

var (
	errChainUpdate = errors.New("rendered section of chain updated")
)

// mapRenderer represents a process that renders filter maps in a specified
// range according to the actual targetView.
type mapRenderer struct {
	f                                *FilterMaps
	afterLastMap                     uint32
	currentMap                       *renderedMap
	finishedMaps                     map[uint32]*renderedMap
	firstFinished, afterLastFinished uint32
	iterator                         *logIterator
}

// renderedMap represents a single filter map that is being rendered in memory.
type renderedMap struct {
	filterMap     filterMap
	mapIndex      uint32
	lastBlock     uint64
	lastBlockId   common.Hash
	blockLvPtrs   []uint64 // start pointers of blocks starting in this map; last one is lastBlock
	finished      bool     // iterator finished; all values rendered
	headDelimiter uint64   // if finished then points to the future block delimiter of the head block
}

// firstBlock returns the first block number that starts in the given map.
func (r *renderedMap) firstBlock() uint64 {
	return r.lastBlock + 1 - uint64(len(r.blockLvPtrs))
}

// renderMapsBefore creates a mapRenderer that renders the log index until the
// specified map index boundary, starting from the latest available starting
// point that is consistent with the current targetView.
// The renderer ensures that filterMapsRange, indexedView and the actual map
// data are always consistent with each other. If afterLastMap is greater than
// the latest existing rendered map then indexedView is updated to targetView,
// otherwise it is checked that the rendered range is consistent with both
// views.
func (f *FilterMaps) renderMapsBefore(afterLastMap uint32) (*mapRenderer, error) {
	nextMap, startBlock, startLvPtr, err := f.lastCanonicalMapBoundaryBefore(afterLastMap)
	if err != nil {
		return nil, err
	}
	if snapshot := f.lastCanonicalSnapshotBefore(afterLastMap); snapshot != nil && snapshot.mapIndex >= nextMap {
		return f.renderMapsFromSnapshot(snapshot)
	}
	if nextMap >= afterLastMap {
		return nil, nil
	}
	return f.renderMapsFromMapBoundary(nextMap, afterLastMap, startBlock, startLvPtr)
}

// renderMapsFromSnapshot creates a mapRenderer that starts rendering from a
// snapshot made at a block boundary.
func (f *FilterMaps) renderMapsFromSnapshot(cp *renderedMap) (*mapRenderer, error) {
	f.testSnapshotUsed = true
	iter, err := f.newLogIteratorFromBlockDelimiter(cp.lastBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create log iterator from block delimiter %d: %v", cp.lastBlock, err)
	}
	return &mapRenderer{
		f: f,
		currentMap: &renderedMap{
			filterMap:   cp.filterMap.copy(),
			mapIndex:    cp.mapIndex,
			lastBlock:   cp.lastBlock,
			blockLvPtrs: cp.blockLvPtrs,
		},
		finishedMaps:      make(map[uint32]*renderedMap),
		firstFinished:     cp.mapIndex,
		afterLastFinished: cp.mapIndex,
		afterLastMap:      math.MaxUint32,
		iterator:          iter,
	}, nil
}

// renderMapsFromMapBoundary creates a mapRenderer that starts rendering at a
// map boundary.
func (f *FilterMaps) renderMapsFromMapBoundary(firstMap, afterLastMap uint32, startBlock, startLvPtr uint64) (*mapRenderer, error) {
	iter, err := f.newLogIteratorFromMapBoundary(firstMap, startBlock, startLvPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to create log iterator from map boundary %d: %v", firstMap, err)
	}
	return &mapRenderer{
		f: f,
		currentMap: &renderedMap{
			filterMap: f.emptyFilterMap(),
			mapIndex:  firstMap,
			lastBlock: iter.blockNumber,
		},
		finishedMaps:      make(map[uint32]*renderedMap),
		firstFinished:     firstMap,
		afterLastFinished: firstMap,
		afterLastMap:      afterLastMap,
		iterator:          iter,
	}, nil
}

// lastCanonicalSnapshotBefore returns the latest cached snapshot that matches
// the current targetView.
func (f *FilterMaps) lastCanonicalSnapshotBefore(afterLastMap uint32) *renderedMap {
	var best *renderedMap
	for _, blockNumber := range f.renderSnapshots.Keys() {
		if cp, _ := f.renderSnapshots.Get(blockNumber); cp != nil && blockNumber < f.indexedRange.afterLastIndexedBlock &&
			blockNumber <= f.targetView.headNumber && f.targetView.getBlockId(blockNumber) == cp.lastBlockId &&
			cp.mapIndex < afterLastMap && (best == nil || blockNumber > best.lastBlock) {
			best = cp
		}
	}
	return best
}

// lastCanonicalMapBoundaryBefore returns the latest map boundary before the
// specified map index that matches the current targetView. This can either
// be a checkpoint (hardcoded or left from a previously unindexed tail epoch)
// or the boundary of a currently rendered map.
// Along with the next map index where the rendering can be started, the number
// and starting log value pointer of the last block is also returned.
func (f *FilterMaps) lastCanonicalMapBoundaryBefore(afterLastMap uint32) (nextMap uint32, startBlock, startLvPtr uint64, err error) {
	if !f.indexedRange.initialized {
		return 0, 0, 0, nil
	}
	mapIndex := afterLastMap
	for {
		var ok bool
		if mapIndex, ok = f.lastMapBoundaryBefore(mapIndex); !ok {
			return 0, 0, 0, nil
		}
		lastBlock, lastBlockId, err := f.getLastBlockOfMap(mapIndex)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to retrieve last block of reverse iterated map %d: %v", mapIndex, err)
		}
		if lastBlock >= f.indexedView.headNumber || lastBlock >= f.targetView.headNumber ||
			lastBlockId != f.targetView.getBlockId(lastBlock) {
			// map is not full or inconsistent with targetView; roll back
			continue
		}
		lvPtr, err := f.getBlockLvPointer(lastBlock)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to retrieve log value pointer of last canonical boundary block %d: %v", lastBlock, err)
		}
		return mapIndex + 1, lastBlock, lvPtr, nil
	}
}

// lastMapBoundaryBefore returns the latest map boundary before the specified
// map index.
func (f *FilterMaps) lastMapBoundaryBefore(mapIndex uint32) (uint32, bool) {
	if !f.indexedRange.initialized || f.indexedRange.afterLastRenderedMap == 0 {
		return 0, false
	}
	if mapIndex > f.indexedRange.afterLastRenderedMap {
		mapIndex = f.indexedRange.afterLastRenderedMap
	}
	if mapIndex > f.indexedRange.firstRenderedMap {
		return mapIndex - 1, true
	}
	if mapIndex+f.mapsPerEpoch > f.indexedRange.firstRenderedMap {
		if mapIndex > f.indexedRange.firstRenderedMap-f.mapsPerEpoch+f.indexedRange.tailPartialEpoch {
			mapIndex = f.indexedRange.firstRenderedMap - f.mapsPerEpoch + f.indexedRange.tailPartialEpoch
		}
	} else {
		mapIndex = (mapIndex >> f.logMapsPerEpoch) << f.logMapsPerEpoch
	}
	if mapIndex == 0 {
		return 0, false
	}
	return mapIndex - 1, true
}

// emptyFilterMap returns an empty filter map.
func (f *FilterMaps) emptyFilterMap() filterMap {
	return make(filterMap, f.mapHeight)
}

// loadHeadSnapshot loads the last rendered map from the database and creates
// a snapshot.
func (f *FilterMaps) loadHeadSnapshot() error {
	fm, err := f.getFilterMap(f.indexedRange.afterLastRenderedMap - 1)
	if err != nil {
		return fmt.Errorf("failed to load head snapshot map %d: %v", f.indexedRange.afterLastRenderedMap-1, err)
	}
	lastBlock, _, err := f.getLastBlockOfMap(f.indexedRange.afterLastRenderedMap - 1)
	if err != nil {
		return fmt.Errorf("failed to retrieve last block of head snapshot map %d: %v", f.indexedRange.afterLastRenderedMap-1, err)
	}
	var firstBlock uint64
	if f.indexedRange.afterLastRenderedMap > 1 {
		prevLastBlock, _, err := f.getLastBlockOfMap(f.indexedRange.afterLastRenderedMap - 2)
		if err != nil {
			return fmt.Errorf("failed to retrieve last block of map %d before head snapshot: %v", f.indexedRange.afterLastRenderedMap-2, err)
		}
		firstBlock = prevLastBlock + 1
	}
	lvPtrs := make([]uint64, lastBlock+1-firstBlock)
	for i := range lvPtrs {
		lvPtrs[i], err = f.getBlockLvPointer(firstBlock + uint64(i))
		if err != nil {
			return fmt.Errorf("failed to retrieve log value pointer of head snapshot block %d: %v", firstBlock+uint64(i), err)
		}
	}
	f.renderSnapshots.Add(f.indexedRange.afterLastIndexedBlock-1, &renderedMap{
		filterMap:     fm,
		mapIndex:      f.indexedRange.afterLastRenderedMap - 1,
		lastBlock:     f.indexedRange.afterLastIndexedBlock - 1,
		lastBlockId:   f.indexedView.getBlockId(f.indexedRange.afterLastIndexedBlock - 1),
		blockLvPtrs:   lvPtrs,
		finished:      true,
		headDelimiter: f.indexedRange.headBlockDelimiter,
	})
	return nil
}

// makeSnapshot creates a snapshot of the current state of the rendered map.
func (r *mapRenderer) makeSnapshot() {
	r.f.renderSnapshots.Add(r.iterator.blockNumber, &renderedMap{
		filterMap:     r.currentMap.filterMap.copy(),
		mapIndex:      r.currentMap.mapIndex,
		lastBlock:     r.iterator.blockNumber,
		lastBlockId:   r.f.targetView.getBlockId(r.currentMap.lastBlock),
		blockLvPtrs:   r.currentMap.blockLvPtrs,
		finished:      true,
		headDelimiter: r.iterator.lvIndex,
	})
}

// run does the actual map rendering. It periodically calls the stopCb callback
// and if it returns true the process is interrupted an can be resumed later
// by calling run again. The writeCb callback is called after new maps have
// been written to disk and the index range has been updated accordingly.
func (r *mapRenderer) run(stopCb func() bool, writeCb func()) (bool, error) {
	for {
		if done, err := r.renderCurrentMap(stopCb); !done {
			return done, err // stopped or failed
		}
		// map finished
		r.finishedMaps[r.currentMap.mapIndex] = r.currentMap
		r.afterLastFinished++
		if len(r.finishedMaps) >= maxMapsPerBatch || r.afterLastFinished&(r.f.baseRowGroupLength-1) == 0 {
			if err := r.writeFinishedMaps(stopCb); err != nil {
				return false, err
			}
			writeCb()
		}
		if r.afterLastFinished == r.afterLastMap || r.iterator.finished {
			if err := r.writeFinishedMaps(stopCb); err != nil {
				return false, err
			}
			writeCb()
			return true, nil
		}
		r.currentMap = &renderedMap{
			filterMap: r.f.emptyFilterMap(),
			mapIndex:  r.afterLastFinished,
		}
	}
}

// renderCurrentMap renders a single map.
func (r *mapRenderer) renderCurrentMap(stopCb func() bool) (bool, error) {
	if !r.iterator.updateChainView(r.f.targetView) {
		return false, errChainUpdate
	}
	var waitCnt int

	if r.iterator.lvIndex == 0 {
		r.currentMap.blockLvPtrs = []uint64{0}
	}
	type lvPos struct{ rowIndex, layerIndex uint32 }
	rowMappingCache := lru.NewCache[common.Hash, lvPos](cachedRowMappings)
	defer rowMappingCache.Purge()

	for r.iterator.lvIndex < uint64(r.currentMap.mapIndex+1)<<r.f.logValuesPerMap && !r.iterator.finished {
		waitCnt++
		if waitCnt >= valuesPerCallback {
			if stopCb() {
				return false, nil
			}
			if !r.iterator.updateChainView(r.f.targetView) {
				return false, errChainUpdate
			}
			waitCnt = 0
		}
		r.currentMap.lastBlock = r.iterator.blockNumber
		if r.iterator.delimiter {
			r.currentMap.lastBlock++
			r.currentMap.blockLvPtrs = append(r.currentMap.blockLvPtrs, r.iterator.lvIndex+1)
		}
		if logValue := r.iterator.getValueHash(); logValue != (common.Hash{}) {
			lvp, cached := rowMappingCache.Get(logValue)
			if !cached {
				lvp = lvPos{rowIndex: r.f.rowIndex(r.currentMap.mapIndex, 0, logValue)}
			}
			for uint32(len(r.currentMap.filterMap[lvp.rowIndex])) >= r.f.maxRowLength(lvp.layerIndex) {
				lvp.layerIndex++
				lvp.rowIndex = r.f.rowIndex(r.currentMap.mapIndex, lvp.layerIndex, logValue)
				cached = false
			}
			r.currentMap.filterMap[lvp.rowIndex] = append(r.currentMap.filterMap[lvp.rowIndex], r.f.columnIndex(r.iterator.lvIndex, &logValue))
			if !cached {
				rowMappingCache.Add(logValue, lvp)
			}
		}
		if err := r.iterator.next(); err != nil {
			return false, fmt.Errorf("failed to advance log iterator at %d while rendering map %d: %v", r.iterator.lvIndex, r.currentMap.mapIndex, err)
		}
		if !r.f.testDisableSnapshots && r.afterLastMap >= r.f.indexedRange.afterLastRenderedMap &&
			(r.iterator.delimiter || r.iterator.finished) {
			r.makeSnapshot()
		}
	}
	if r.iterator.finished {
		r.currentMap.finished = true
		r.currentMap.headDelimiter = r.iterator.lvIndex
	}
	r.currentMap.lastBlockId = r.f.targetView.getBlockId(r.currentMap.lastBlock)
	return true, nil
}

// writeFinishedMaps writes rendered maps to the database and updates
// filterMapsRange and indexedView accordingly.
func (r *mapRenderer) writeFinishedMaps(pauseCb func() bool) error {
	if len(r.finishedMaps) == 0 {
		return nil
	}
	r.f.indexLock.Lock()
	defer r.f.indexLock.Unlock()

	oldRange := r.f.indexedRange
	tempRange, err := r.getTempRange()
	if err != nil {
		return fmt.Errorf("failed to get temporary rendered range: %v", err)
	}
	newRange, err := r.getUpdatedRange()
	if err != nil {
		return fmt.Errorf("failed to get updated rendered range: %v", err)
	}
	renderedView := r.f.targetView // stopCb callback might still change targetView while writing finished maps

	batch := r.f.db.NewBatch()
	var writeCnt int
	checkWriteCnt := func() {
		writeCnt++
		if writeCnt == rowsPerBatch {
			writeCnt = 0
			if err := batch.Write(); err != nil {
				log.Crit("Error writing log index update batch", "error", err)
			}
			// do not exit while in partially written state but do allow processing
			// events and pausing while block processing is in progress
			pauseCb()
			batch = r.f.db.NewBatch()
		}
	}

	r.f.setRange(batch, r.f.indexedView, tempRange)
	// add or update filter rows
	for rowIndex := uint32(0); rowIndex < r.f.mapHeight; rowIndex++ {
		var (
			mapIndices []uint32
			rows       []FilterRow
		)
		for mapIndex := r.firstFinished; mapIndex < r.afterLastFinished; mapIndex++ {
			row := r.finishedMaps[mapIndex].filterMap[rowIndex]
			if fm, _ := r.f.filterMapCache.Get(mapIndex); fm != nil && row.Equal(fm[rowIndex]) {
				continue
			}
			mapIndices = append(mapIndices, mapIndex)
			rows = append(rows, row)
		}
		if newRange.afterLastRenderedMap == r.afterLastFinished { // head updated; remove future entries
			for mapIndex := r.afterLastFinished; mapIndex < oldRange.afterLastRenderedMap; mapIndex++ {
				if fm, _ := r.f.filterMapCache.Get(mapIndex); fm != nil && len(fm[rowIndex]) == 0 {
					continue
				}
				mapIndices = append(mapIndices, mapIndex)
				rows = append(rows, nil)
			}
		}
		if err := r.f.storeFilterMapRows(batch, mapIndices, rowIndex, rows); err != nil {
			return fmt.Errorf("failed to store filter maps %v row %d: %v", mapIndices, rowIndex, err)
		}
		checkWriteCnt()
	}
	// update filter map cache
	if newRange.afterLastRenderedMap == r.afterLastFinished {
		// head updated; cache new head maps and remove future entries
		for mapIndex := r.firstFinished; mapIndex < r.afterLastFinished; mapIndex++ {
			r.f.filterMapCache.Add(mapIndex, r.finishedMaps[mapIndex].filterMap)
		}
		for mapIndex := r.afterLastFinished; mapIndex < oldRange.afterLastRenderedMap; mapIndex++ {
			r.f.filterMapCache.Remove(mapIndex)
		}
	} else {
		// head not updated; do not cache maps during tail rendering because we
		// need head maps to be available in the cache
		for mapIndex := r.firstFinished; mapIndex < r.afterLastFinished; mapIndex++ {
			r.f.filterMapCache.Remove(mapIndex)
		}
	}
	// add or update block pointers
	blockNumber := r.finishedMaps[r.firstFinished].firstBlock()
	for mapIndex := r.firstFinished; mapIndex < r.afterLastFinished; mapIndex++ {
		renderedMap := r.finishedMaps[mapIndex]
		r.f.storeLastBlockOfMap(batch, mapIndex, renderedMap.lastBlock, renderedMap.lastBlockId)
		checkWriteCnt()
		if blockNumber != renderedMap.firstBlock() {
			panic("non-continuous block numbers")
		}
		for _, lvPtr := range renderedMap.blockLvPtrs {
			r.f.storeBlockLvPointer(batch, blockNumber, lvPtr)
			checkWriteCnt()
			blockNumber++
		}
	}
	if newRange.afterLastRenderedMap == r.afterLastFinished { // head updated; remove future entries
		for mapIndex := r.afterLastFinished; mapIndex < oldRange.afterLastRenderedMap; mapIndex++ {
			r.f.deleteLastBlockOfMap(batch, mapIndex)
			checkWriteCnt()
		}
		for ; blockNumber < oldRange.afterLastIndexedBlock; blockNumber++ {
			r.f.deleteBlockLvPointer(batch, blockNumber)
			checkWriteCnt()
		}
	}
	r.finishedMaps = make(map[uint32]*renderedMap)
	r.firstFinished = r.afterLastFinished
	r.f.setRange(batch, renderedView, newRange)
	if err := batch.Write(); err != nil {
		log.Crit("Error writing log index update batch", "error", err)
	}
	return nil
}

// getTempRange returns a temporary filterMapsRange that is committed to the
// database while the newly rendered maps are partially written. Writing all
// processed maps in a single database batch would be a serious hit on db
// performance so instead safety is ensured by first reverting the valid map
// range to the unchanged region until all new map data is committed.
func (r *mapRenderer) getTempRange() (filterMapsRange, error) {
	tempRange := r.f.indexedRange
	if err := tempRange.addRenderedRange(r.firstFinished, r.firstFinished, r.afterLastMap, r.f.mapsPerEpoch); err != nil {
		return filterMapsRange{}, fmt.Errorf("failed to update temporary rendered range: %v", err)
	}
	if tempRange.firstRenderedMap != r.f.indexedRange.firstRenderedMap {
		// first rendered map changed; update first indexed block
		if tempRange.firstRenderedMap > 0 {
			lastBlock, _, err := r.f.getLastBlockOfMap(tempRange.firstRenderedMap - 1)
			if err != nil {
				return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d before temporary range: %v", tempRange.firstRenderedMap-1, err)
			}
			tempRange.firstIndexedBlock = lastBlock + 1
		} else {
			tempRange.firstIndexedBlock = 0
		}
	}
	if tempRange.afterLastRenderedMap != r.f.indexedRange.afterLastRenderedMap {
		// first rendered map changed; update first indexed block
		if tempRange.afterLastRenderedMap > 0 {
			lastBlock, _, err := r.f.getLastBlockOfMap(tempRange.afterLastRenderedMap - 1)
			if err != nil {
				return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d at the end of temporary range: %v", tempRange.afterLastRenderedMap-1, err)
			}
			tempRange.afterLastIndexedBlock = lastBlock
		} else {
			tempRange.afterLastIndexedBlock = 0
		}
		tempRange.headBlockDelimiter = 0
	}
	return tempRange, nil
}

// getUpdatedRange returns the updated filterMapsRange after writing the newly
// rendered maps.
func (r *mapRenderer) getUpdatedRange() (filterMapsRange, error) {
	// update filterMapsRange
	newRange := r.f.indexedRange
	if err := newRange.addRenderedRange(r.firstFinished, r.afterLastFinished, r.afterLastMap, r.f.mapsPerEpoch); err != nil {
		return filterMapsRange{}, fmt.Errorf("failed to update rendered range: %v", err)
	}
	if newRange.firstRenderedMap != r.f.indexedRange.firstRenderedMap {
		// first rendered map changed; update first indexed block
		if newRange.firstRenderedMap > 0 {
			lastBlock, _, err := r.f.getLastBlockOfMap(newRange.firstRenderedMap - 1)
			if err != nil {
				return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d before rendered range: %v", newRange.firstRenderedMap-1, err)
			}
			newRange.firstIndexedBlock = lastBlock + 1
		} else {
			newRange.firstIndexedBlock = 0
		}
	}
	if newRange.afterLastRenderedMap == r.afterLastFinished {
		// last rendered map changed; update last indexed block and head pointers
		lm := r.finishedMaps[r.afterLastFinished-1]
		newRange.headBlockIndexed = lm.finished
		if lm.finished {
			newRange.afterLastIndexedBlock = r.f.targetView.headNumber + 1
			if lm.lastBlock != r.f.targetView.headNumber {
				panic("map rendering finished but last block != head block")
			}
			newRange.headBlockDelimiter = lm.headDelimiter
		} else {
			newRange.afterLastIndexedBlock = lm.lastBlock
			newRange.headBlockDelimiter = 0
		}
	} else {
		// last rendered map not replaced; ensure that target chain view matches
		// indexed chain view on the rendered section
		if lastBlock := r.finishedMaps[r.afterLastFinished-1].lastBlock; !matchViews(r.f.indexedView, r.f.targetView, lastBlock) {
			return filterMapsRange{}, errChainUpdate
		}
	}
	return newRange, nil
}

// addRenderedRange adds the range [firstRendered, afterLastRendered) and
// removes [afterLastRendered, afterLastRemoved) from the set of rendered maps.
func (fmr *filterMapsRange) addRenderedRange(firstRendered, afterLastRendered, afterLastRemoved, mapsPerEpoch uint32) error {
	if !fmr.initialized {
		return errors.New("log index not initialized")
	}

	// Here we create a slice of endpoints for the rendered sections. There are two endpoints
	// for each section: the index of the first map, and the index after the last map in the
	// section. We then iterate the endpoints -- adding d values -- to determine whether the
	// sections are contiguous or whether they have a gap.
	type endpoint struct {
		m uint32
		d int
	}
	endpoints := []endpoint{{fmr.firstRenderedMap, 1}, {fmr.afterLastRenderedMap, -1}, {firstRendered, 1}, {afterLastRendered, -101}, {afterLastRemoved, 100}}
	if fmr.tailPartialEpoch > 0 {
		endpoints = append(endpoints, []endpoint{{fmr.firstRenderedMap - mapsPerEpoch, 1}, {fmr.firstRenderedMap - mapsPerEpoch + fmr.tailPartialEpoch, -1}}...)
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].m < endpoints[j].m })
	var (
		sum    int
		merged []uint32
		last   bool
	)
	for i, e := range endpoints {
		sum += e.d
		if i < len(endpoints)-1 && endpoints[i+1].m == e.m {
			continue
		}
		if (sum > 0) != last {
			merged = append(merged, e.m)
			last = !last
		}
	}

	switch len(merged) {
	case 0:
		// Initialized database, but no finished maps yet.
		fmr.tailPartialEpoch = 0
		fmr.firstRenderedMap = firstRendered
		fmr.afterLastRenderedMap = firstRendered

	case 2:
		// One rendered section (no partial tail epoch).
		fmr.tailPartialEpoch = 0
		fmr.firstRenderedMap = merged[0]
		fmr.afterLastRenderedMap = merged[1]

	case 4:
		// Two rendered sections (with a gap).
		// First section (merged[0]-merged[1]) is for the partial tail epoch,
		// and it has to start exactly one epoch before the main section.
		if merged[2] != merged[0]+mapsPerEpoch {
			return fmt.Errorf("invalid tail partial epoch: %v", merged)
		}
		fmr.tailPartialEpoch = merged[1] - merged[0]
		fmr.firstRenderedMap = merged[2]
		fmr.afterLastRenderedMap = merged[3]

	default:
		return fmt.Errorf("invalid number of rendered sections: %v", merged)
	}
	return nil
}

// logIterator iterates on the linear log value index range.
type logIterator struct {
	chainView                       *ChainView
	blockNumber                     uint64
	receipts                        types.Receipts
	blockStart, delimiter, finished bool
	txIndex, logIndex, topicIndex   int
	lvIndex                         uint64
}

var errUnindexedRange = errors.New("unindexed range")

// newLogIteratorFromBlockDelimiter creates a logIterator starting at the
// given block's first log value entry (the block delimiter), according to the
// current targetView.
func (f *FilterMaps) newLogIteratorFromBlockDelimiter(blockNumber uint64) (*logIterator, error) {
	if blockNumber > f.targetView.headNumber {
		return nil, fmt.Errorf("iterator entry point %d after target chain head block %d", blockNumber, f.targetView.headNumber)
	}
	if blockNumber < f.indexedRange.firstIndexedBlock || blockNumber >= f.indexedRange.afterLastIndexedBlock {
		return nil, errUnindexedRange
	}
	var lvIndex uint64
	if f.indexedRange.headBlockIndexed && blockNumber+1 == f.indexedRange.afterLastIndexedBlock {
		lvIndex = f.indexedRange.headBlockDelimiter
	} else {
		var err error
		lvIndex, err = f.getBlockLvPointer(blockNumber + 1)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve log value pointer of block %d after delimiter: %v", blockNumber+1, err)
		}
		lvIndex--
	}
	finished := blockNumber == f.targetView.headNumber
	return &logIterator{
		chainView:   f.targetView,
		blockNumber: blockNumber,
		finished:    finished,
		delimiter:   !finished,
		lvIndex:     lvIndex,
	}, nil
}

// newLogIteratorFromMapBoundary creates a logIterator starting at the given
// map boundary, according to the current targetView.
func (f *FilterMaps) newLogIteratorFromMapBoundary(mapIndex uint32, startBlock, startLvPtr uint64) (*logIterator, error) {
	if startBlock > f.targetView.headNumber {
		return nil, fmt.Errorf("iterator entry point %d after target chain head block %d", startBlock, f.targetView.headNumber)
	}
	// get block receipts
	receipts := f.targetView.getReceipts(startBlock)
	if receipts == nil {
		return nil, fmt.Errorf("receipts not found for start block %d", startBlock)
	}
	// initialize iterator at block start
	l := &logIterator{
		chainView:   f.targetView,
		blockNumber: startBlock,
		receipts:    receipts,
		blockStart:  true,
		lvIndex:     startLvPtr,
	}
	l.nextValid()
	targetIndex := uint64(mapIndex) << f.logValuesPerMap
	if l.lvIndex > targetIndex {
		return nil, fmt.Errorf("log value pointer %d of last block of map is after map boundary %d", l.lvIndex, targetIndex)
	}
	// iterate to map boundary
	for l.lvIndex < targetIndex {
		if l.finished {
			return nil, fmt.Errorf("iterator already finished at %d before map boundary target %d", l.lvIndex, targetIndex)
		}
		if err := l.next(); err != nil {
			return nil, fmt.Errorf("failed to advance log iterator at %d before map boundary target %d: %v", l.lvIndex, targetIndex, err)
		}
	}
	return l, nil
}

// updateChainView updates the iterator's chain view if it still matches the
// previous view at the current position. Returns true if successful.
func (l *logIterator) updateChainView(cv *ChainView) bool {
	if !matchViews(cv, l.chainView, l.blockNumber) {
		return false
	}
	l.chainView = cv
	return true
}

// getValueHash returns the log value hash at the current position.
func (l *logIterator) getValueHash() common.Hash {
	if l.delimiter || l.finished {
		return common.Hash{}
	}
	log := l.receipts[l.txIndex].Logs[l.logIndex]
	if l.topicIndex == 0 {
		return addressValue(log.Address)
	}
	return topicValue(log.Topics[l.topicIndex-1])
}

// next moves the iterator to the next log value index.
func (l *logIterator) next() error {
	if l.finished {
		return nil
	}
	if l.delimiter {
		l.delimiter = false
		l.blockNumber++
		l.receipts = l.chainView.getReceipts(l.blockNumber)
		if l.receipts == nil {
			return fmt.Errorf("receipts not found for block %d", l.blockNumber)
		}
		l.txIndex, l.logIndex, l.topicIndex, l.blockStart = 0, 0, 0, true
	} else {
		l.topicIndex++
		l.blockStart = false
	}
	l.lvIndex++
	l.nextValid()
	return nil
}

// nextValid updates the internal transaction, log and topic index pointers
// to the next existing log value of the given block if necessary.
// Note that nextValid does not advance the log value index pointer.
func (l *logIterator) nextValid() {
	for ; l.txIndex < len(l.receipts); l.txIndex++ {
		receipt := l.receipts[l.txIndex]
		for ; l.logIndex < len(receipt.Logs); l.logIndex++ {
			log := receipt.Logs[l.logIndex]
			if l.topicIndex <= len(log.Topics) {
				return
			}
			l.topicIndex = 0
		}
		l.logIndex = 0
	}
	if l.blockNumber == l.chainView.headNumber {
		l.finished = true
	} else {
		l.delimiter = true
	}
}
