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
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

const (
	maxCanonicalSnapshots = 4
	maxRecentSnapshots    = 4
	maxIndexViewMaps      = 2
)

var ( //TODO
	mapCountGauge    = metrics.NewRegisteredGauge("filtermaps/maps/count", nil)      // actual number of rendered maps
	mapLogValueMeter = metrics.NewRegisteredMeter("filtermaps/maps/logvalues", nil)  // number of log values processed
	mapBlockMeter    = metrics.NewRegisteredMeter("filtermaps/maps/blocks", nil)     // number of block delimiters processed
	mapRenderTimer   = metrics.NewRegisteredTimer("filtermaps/maps/rendertime", nil) // time elapsed while rendering a single map
	mapWriteTimer    = metrics.NewRegisteredTimer("filtermaps/maps/writetime", nil)  // time elapsed while writing a batch of finished maps to db
)

type Indexer struct {
	config                                          Config
	storage                                         *mapStorage
	checkpoints                                     []checkpointList
	headRenderer, tailRenderer                      *renderState
	historyCutoff, finalized                        uint64
	tailRenderLast, headNumber                      uint64
	lastCanonical                                   uint64
	tailEpoch, targetTailEpoch, activeViewTailEpoch uint32
	canonicalHashes                                 []common.Hash // last one belongs to lastCanonical
	recentHashes                                    []common.Hash // last one is the most recently saved
	snapshotsLock                                   sync.RWMutex
	snapshots                                       map[common.Hash]*IndexView
	headMapsCache                                   *lru.Cache[uint32, *finishedMap]
}

// Config contains the configuration options for NewFilterMaps.
type Config struct {
	History  uint64 // number of historical blocks to index
	Disabled bool   // disables indexing completely

	// This option enables the checkpoint JSON file generator.
	// If set, the given file will be updated with checkpoint information.
	ExportFileName string

	// expect trie nodes of hash based state scheme in the filtermaps key range;
	// use safe iterator based implementation of DeleteRange that skips them
	HashScheme bool
}

// TODO blockId vs blockHash?
// TODO disable, export, history, finalized
func NewIndexer(db ethdb.KeyValueStore, params *Params, config Config) *Indexer {
	params.sanitize()
	mapDb := newMapDatabase(params, db, config.HashScheme)
	ix := &Indexer{
		config:        config,
		storage:       newMapStorage(&DefaultParams, mapDb),
		checkpoints:   checkpoints,
		snapshots:     make(map[common.Hash]*IndexView),
		headMapsCache: lru.NewCache[uint32, *finishedMap](maxIndexViewMaps),
	}
	ix.headRenderer = ix.initMapBoundary(ix.storage.lastBoundaryBefore(math.MaxUint32), math.MaxUint32)
	ix.updateTailEpoch()
	ix.updateActiveViewTailEpoch()
	ix.updateTailState()
	fmt.Println("init  tail epoch", ix.tailEpoch, "tail target", ix.targetTailEpoch, "head number", ix.headNumber)
	return ix
}

func (ix *Indexer) initMapBoundary(nextMap, limitMap uint32) *renderState {
	fmt.Println("initMapBoundary", nextMap, limitMap)
	rs := &renderState{
		params:      ix.storage.params,
		renderRange: common.NewRange[uint32](nextMap, limitMap-nextMap),
	}
	for {
		nextMap = ix.storage.lastBoundaryBefore(nextMap)
		fmt.Println(" lbb", nextMap)
		if nextMap == 0 {
			// initialize at genesis
			fmt.Println(" genesis")
			rs.currentMap = rs.params.newMemoryMap()
			return rs
		}
		lastNumber, lastHash, err := ix.storage.getLastBlockOfMap(nextMap - 1)
		if err != nil {
			log.Error("Last block of map not found, reverting database", "mapIndex", nextMap-1)
			nextMap = ix.storage.lastBoundaryBefore(nextMap - 1)
			ix.revertMaps(nextMap)
			continue
		}
		lvPointer, err := ix.storage.getBlockLvPointer(lastNumber)
		if err != nil {
			log.Error("Block pointer of last block of map not found, reverting database", "mapIndex", nextMap-1, "blockNumber", lastNumber)
			nextMap = ix.storage.lastBoundaryBefore(nextMap - 1)
			ix.revertMaps(nextMap)
			continue
		}
		rs.lvPointer = lvPointer
		rs.mapIndex = uint32(lvPointer >> ix.storage.params.logValuesPerMap)
		rs.nextBlock = lastNumber
		rs.partialBlock = true
		rs.partialBlockHash = lastHash
		fmt.Println(" nextBlock", rs.nextBlock, "mapIndex", rs.mapIndex)
		return rs
	}
}

func (ix *Indexer) initSnapshot(snapshot *IndexView) *renderState {
	mapIndex := ix.storage.lastBoundaryBefore(snapshot.firstMemoryMap)
	ix.revertMaps(mapIndex)
	if snapshot.checkInvalid() {
		log.Error("Failed to revert to invalidated snapshot", "blockNumber", snapshot.blockRange.Last())
		return nil
	}

	fmt.Println("initSnapshot", snapshot.headBlockHash)
	return &renderState{
		params:      ix.storage.params,
		renderRange: common.NewRange[uint32](snapshot.headMapIndex, math.MaxUint32-snapshot.headMapIndex),
		currentMap:  snapshot.headMap.clone(),
		mapIndex:    snapshot.headMapIndex,
		lvPointer:   snapshot.headLvPointer,
	}
}

// Note that revertMaps might be called while headRenderer is nil and might set
// headRenderer to nil.
func (ix *Indexer) revertMaps(mapIndex uint32) {
	fmt.Println("revertMaps", mapIndex)
	if mapIndex < ix.storage.lastBoundaryBefore(math.MaxUint32) {
		for hash, iv := range ix.snapshots {
			if iv.firstMemoryMap > mapIndex {
				iv.invalidate()
				ix.snapshotsLock.Lock()
				delete(ix.snapshots, hash)
				ix.snapshotsLock.Unlock()
			}
		}
		ix.storage.deleteMaps(common.NewRange[uint32](mapIndex, math.MaxUint32-mapIndex))
		ix.headMapsCache.Purge() // invalidate all maps cached by index
	}
	if ix.headRenderer != nil && mapIndex <= ix.headRenderer.mapIndex {
		ix.headRenderer = nil
	}
	if ix.tailRenderer != nil && mapIndex <= ix.tailRenderer.mapIndex {
		ix.tailRenderer = nil
	}
}

func (ix *Indexer) updateTailEpoch() {
	ix.tailEpoch = ix.storage.tailEpoch()
	if ix.config.History == 0 {
		ix.targetTailEpoch = 0
		return
	}
	headEpoch := ix.storage.params.mapEpoch(ix.headRenderer.mapIndex)
	for ix.targetTailEpoch < headEpoch {
		nextTailNumber, err := ix.storage.tailNumberOfEpoch(ix.targetTailEpoch + 1)
		if err != nil {
			log.Error("Could not get tail block number of epoch", "epoch", ix.targetTailEpoch+1, "error", err)
			return
		}
		if nextTailNumber+ix.config.History > ix.headNumber+1 {
			break
		}
		ix.targetTailEpoch++
	}
	for ix.targetTailEpoch > 0 {
		prevTailNumber, err := ix.storage.tailNumberOfEpoch(ix.targetTailEpoch - 1)
		if err != nil {
			log.Error("Could not get tail block number of epoch", "epoch", ix.targetTailEpoch-1, "error", err)
			return
		}
		if prevTailNumber+ix.config.History <= ix.headNumber+1 {
			break
		}
		ix.targetTailEpoch--
	}
}

func (ix *Indexer) updateActiveViewTailEpoch() {
	ix.snapshotsLock.RLock()
	defer ix.snapshotsLock.RUnlock()

	ix.activeViewTailEpoch = math.MaxUint32
	for _, iv := range ix.snapshots {
		ix.activeViewTailEpoch = min(ix.activeViewTailEpoch, iv.tailEpoch)
	}
}

func (ix *Indexer) updateTailState() {
	epoch := min(ix.targetTailEpoch, ix.activeViewTailEpoch)
	//fmt.Println("updateTailState", ix.tailEpoch, ix.targetTailEpoch, ix.activeViewTailEpoch, ix.tailRenderer != nil)
	if epoch >= ix.tailEpoch && ix.tailRenderer != nil {
		ix.tailRenderer = nil
		ix.storage.deleteMaps(common.NewRange[uint32](ix.storage.params.firstEpochMap(ix.tailEpoch-1), ix.storage.params.mapsPerEpoch))
	}
	if epoch > ix.tailEpoch {
		ix.storage.deleteMaps(common.NewRange[uint32](ix.storage.params.firstEpochMap(ix.tailEpoch), ix.storage.params.mapsPerEpoch*(epoch-ix.tailEpoch)))
		ix.tailEpoch = epoch
	}
	if epoch < ix.tailEpoch && ix.tailRenderer == nil {
		if lastBlock, _, err := ix.storage.getLastBlockOfMap(ix.storage.params.lastEpochMap(ix.tailEpoch - 1)); err == nil {
			ix.tailRenderer = ix.initMapBoundary(ix.storage.lastBoundaryBefore(ix.storage.params.lastEpochMap(ix.tailEpoch-1)), ix.storage.params.firstEpochMap(ix.tailEpoch))
			ix.tailRenderLast = lastBlock
		} else {
			log.Error("Could not get last block of new tail epoch", "epoch", ix.tailEpoch-1, "error", err)
		}
	}
	//fmt.Println(" after", ix.tailEpoch, ix.targetTailEpoch, ix.activeViewTailEpoch, ix.tailRenderer != nil)
}

func (ix *Indexer) AddBlockData(headers []*types.Header, receipts []types.Receipts) (ready bool, needBlocks common.Range[uint64]) {
	//fmt.Println("/AddBlockData")
	//defer fmt.Println("\\AddBlockData")

	if len(headers) == 0 {
		return ix.Status()
	}
	if !ix.storage.isReady() {
		return false, ix.needBlocks()
	}
	//fmt.Println(" a1")
	ix.headNumber = max(ix.headNumber, headers[len(headers)-1].Number.Uint64())
	for i, header := range headers {
		number, hash := header.Number.Uint64(), header.Hash()
		if number > ix.headRenderer.nextBlock {
			//fmt.Println(" a2")
			ix.tryCheckpointInit(number, hash)
			//fmt.Println(" a3")
		}
		if number == ix.headRenderer.nextBlock {
			if ix.headRenderer.checkNextHash(hash) {
				ix.headRenderer.addReceipts(receipts[i])
				firstMapIndex, finishedMaps := ix.headRenderer.addHeader(header)
				ix.storeFinishedMaps(firstMapIndex, finishedMaps, i == len(headers)-1, true)
				if number+maxCanonicalSnapshots > ix.headNumber {
					ix.storeHeadIndexView(number, hash)
				}
			} else {
				ix.headRenderer = ix.initMapBoundary(max(ix.headRenderer.renderRange.First(), 1)-1, math.MaxUint32)
			}
			ix.updateTailEpoch()
			ix.updateTailState()
		}
		if ix.tailRenderer != nil && number == ix.tailRenderer.nextBlock {
			if ix.tailRenderer.checkNextHash(hash) {
				ix.tailRenderer.addReceipts(receipts[i])
				firstMapIndex, finishedMaps := ix.tailRenderer.addHeader(header)
				ix.storeFinishedMaps(firstMapIndex, finishedMaps, false, false)
				if ix.tailRenderer.finished() {
					ix.tailEpoch--
					ix.tailRenderer = nil
					ix.updateTailState()
				}
			} else {
				// Note that if there is a canonical hash mismatch at the tail epoch then we need to revert the head renderer before this point.
				ix.headRenderer = ix.initMapBoundary(max(ix.tailRenderer.renderRange.First(), 1)-1, math.MaxUint32)
			}
		}
	}
	//fmt.Println(" a4")
	ix.storage.suspendOrResume(false)
	return ix.Status()
}

// epochsUntilBlock returns the numer of epochs in the checkpoint list whose
// last block number is less than or equal to the specified number.
func (cpList checkpointList) epochsUntilBlock(number uint64) uint32 {
	//fmt.Println("epochsUntilBlock", number)
	first, last := uint32(0), uint32(len(cpList))
	for first < last {
		//fmt.Println(" *", first, last)
		mid := (first + last) / 2
		if cpList[mid].BlockNumber > number {
			last = mid
		} else {
			first = mid + 1
		}
	}
	//fmt.Println(" **", first)
	return first
}

func (ix *Indexer) tryCheckpointInit(number uint64, id common.Hash) {
	//fmt.Println("tryCheckpointInit", number, id)
	var ci int
	for ci < len(ix.checkpoints) {
		//fmt.Println(" t1")
		cpList := ix.checkpoints[ci]
		epochs := cpList.epochsUntilBlock(number)
		//fmt.Println(" cpList", len(cpList), epochs)
		if epochs == 0 || cpList[epochs-1].BlockNumber != number {
			/*if epochs == 0 {
				fmt.Println(" skip *", number)
			} else {
				fmt.Println(" skip", cpList[epochs-1].BlockNumber, number)
			}*/
			// block number does not match, skip list (a relevant block might match later)
			ci++
			//fmt.Println(" t2")
			continue
		}
		if cpList[epochs-1].BlockId == id {
			//fmt.Println(" t4")
			// apply matching checkpoint, discard other lists
			if err := ix.storage.addKnownEpochs(cpList[:epochs]); err == nil {
				ix.checkpoints = []checkpointList{cpList}
				ix.headRenderer = ix.initMapBoundary(epochs*ix.storage.params.mapsPerEpoch, math.MaxUint32)
				//fmt.Println(" success")
				return
			} else {
				log.Error("Error initializing epoch boundaries", "error", err)
			}
		}
		//fmt.Println(" t3")
		// checkpoint does not match, discard list
		ix.checkpoints[ci] = ix.checkpoints[len(ix.checkpoints)-1]
		ix.checkpoints = ix.checkpoints[:len(ix.checkpoints)-1]
	}
	//fmt.Println(" no match")
}

func (ix *Indexer) SetFinalized(blockNumber uint64) {
	ix.finalized = blockNumber
}

func (ix *Indexer) SetHistoryCutoff(blockNumber uint64) {
	ix.historyCutoff = blockNumber
}

func (ix *Indexer) Suspended() {
	//fmt.Println("/Suspended")
	//defer fmt.Println("\\Suspended")

	ix.storage.suspendOrResume(true)
}

func (ix *Indexer) Revert(blockNumber uint64) {
	//fmt.Println("/Revert")
	//defer fmt.Println("\\Revert")

	firstCanonical := ix.lastCanonical + 1 - uint64(len(ix.canonicalHashes))
	if blockNumber >= firstCanonical && blockNumber <= ix.lastCanonical {
		blockHash := ix.canonicalHashes[blockNumber-firstCanonical]
		if snapshot, ok := ix.snapshots[blockHash]; ok {
			ix.headRenderer = ix.initSnapshot(snapshot)
			if ix.headRenderer != nil {
				return
			}
		}
	}
	mapIndex := uint32(math.MaxUint32)
	for mapIndex > 0 {
		mapIndex = ix.storage.lastBoundaryBefore(mapIndex)
		lastNumber, _, err := ix.storage.getLastBlockOfMap(mapIndex - 1)
		if err != nil {
			log.Error("Last block of map not found, reverting database", "mapIndex", mapIndex)
			mapIndex--
			continue
		}
		if lastNumber < blockNumber {
			break
		}
	}
	ix.revertMaps(mapIndex)
	ix.headRenderer = ix.initMapBoundary(mapIndex, math.MaxUint32)
	ix.headNumber = blockNumber
}

func (ix *Indexer) Status() (bool, common.Range[uint64]) {
	//fmt.Println("/Status")
	//defer fmt.Println("\\Status")
	//fmt.Println("isReady", ix.storage.isReady())
	return ix.storage.isReady(), ix.needBlocks()
}

func (ix *Indexer) needBlocks() common.Range[uint64] {
	//fmt.Println("needBlocks", ix.finalized, ix.headRenderer.nextBlock, ix.tailRenderer != nil)
	if ix.finalized > ix.headRenderer.nextBlock {
		// request potential checkpoint in this range if available
		for _, cpList := range ix.checkpoints {
			//fmt.Println("cpList", len(cpList))
			if epochs := cpList.epochsUntilBlock(ix.headNumber); epochs > 0 {
				blockNumber := cpList[epochs-1].BlockNumber
				//fmt.Println("epochs", epochs, "blockNumber", blockNumber)
				if ix.storage.lastBoundaryBefore(math.MaxUint32) >= epochs*ix.storage.params.mapsPerEpoch ||
					blockNumber <= ix.headRenderer.nextBlock || blockNumber < ix.historyCutoff {
					//fmt.Println(" cont", ix.storage.lastBoundaryBefore(math.MaxUint32), ix.historyCutoff, ix.storage.params.mapsPerEpoch)
					continue
				}
				//fmt.Println(" chk", blockNumber)
				return common.NewRange[uint64](blockNumber, 1)
			}
		}
	}
	//fmt.Println("nb head", ix.headRenderer.nextBlock, ix.headNumber)
	if ix.headRenderer.nextBlock <= ix.headNumber && ix.headRenderer.nextBlock >= ix.historyCutoff {
		return common.NewRange[uint64](ix.headRenderer.nextBlock, ix.headNumber+1-ix.headRenderer.nextBlock)
	}
	/*if ix.tailRenderer != nil {
		fmt.Println("nb tail", ix.tailRenderer.nextBlock, ix.tailRenderLast)
	}*/
	if ix.tailRenderer != nil &&
		ix.tailRenderer.nextBlock <= ix.tailRenderLast && ix.tailRenderer.nextBlock >= ix.historyCutoff {
		return common.NewRange[uint64](ix.tailRenderer.nextBlock, ix.tailRenderLast+1-ix.tailRenderer.nextBlock)
	}
	//fmt.Println("nb none")
	return common.Range[uint64]{}
}

func (ix *Indexer) Stop() {
	fmt.Println("/Stop")
	defer fmt.Println("\\Stop")

	ix.storage.stop()
}

func (ix *Indexer) releaseView(hash common.Hash) {
	iv := ix.snapshots[hash]
	if iv == nil {
		return
	}
	if iv.addRefCount(-1) {
		iv.invalidate()
		ix.snapshotsLock.Lock()
		delete(ix.snapshots, hash)
		ix.snapshotsLock.Unlock()
	}
}

func (ix *Indexer) GetIndexView(hash common.Hash) *IndexView {
	ix.snapshotsLock.RLock()
	iv := ix.snapshots[hash]
	ix.snapshotsLock.RUnlock()
	if iv == nil || iv.checkReleased() {
		return nil
	}
	iv.addRefCount(1)
	return iv
}

func (ix *Indexer) storeFinishedMaps(firstMapIndex uint32, maps []*finishedMap, forceCommit, cacheHeadMaps bool) {
	if len(maps) == 0 {
		return
	}
	for i, fm := range maps {
		ix.storage.addMap(firstMapIndex+uint32(i), fm, forceCommit && i == len(maps)-1)
		if cacheHeadMaps {
			ix.headMapsCache.Add(firstMapIndex+uint32(i), fm)
		}
	}
}

func (ix *Indexer) getFilterMap(mapIndex uint32) (*finishedMap, error) {
	if fm, ok := ix.headMapsCache.Get(mapIndex); ok {
		return fm, nil
	}
	fm, err := ix.storage.getFilterMap(mapIndex)
	if err != nil {
		return nil, err
	}
	ix.headMapsCache.Add(mapIndex, fm)
	return fm, nil
}

func (ix *Indexer) checkReleasedViews() {
	var deleted bool
	for hash, iv := range ix.snapshots {
		if iv.checkReleased() {
			iv.invalidate()
			ix.snapshotsLock.Lock()
			delete(ix.snapshots, hash)
			ix.snapshotsLock.Unlock()
			deleted = true
		}
	}
	if deleted {
		ix.updateActiveViewTailEpoch()
		ix.updateTailState()
	}
}

func (ix *Indexer) storeHeadIndexView(number uint64, hash common.Hash) {
	ix.checkReleasedViews()
	firstMemoryMap := max(ix.headRenderer.mapIndex, maxIndexViewMaps) - maxIndexViewMaps
	finishedMaps := make([]*finishedMap, 0, ix.headRenderer.mapIndex-firstMemoryMap)
	for mapIndex := firstMemoryMap; mapIndex < ix.headRenderer.mapIndex; mapIndex++ {
		fm, err := ix.getFilterMap(mapIndex)
		if err != nil {
			log.Error("Error loading recent filter map", "mapIndex", mapIndex, "error", err)
		}
		if fm != nil && err == nil {
			finishedMaps = append(finishedMaps, fm)
		} else {
			finishedMaps = finishedMaps[:0]
			firstMemoryMap = mapIndex + 1
		}
	}
	var firstMemoryBlock uint64
	if len(finishedMaps) > 0 {
		firstMemoryBlock = finishedMaps[0].firstBlock()
	} else {
		firstMemoryBlock = ix.headRenderer.currentMap.firstBlock()
	}
	tailNumber, err := ix.storage.tailNumberOfEpoch(ix.tailEpoch)
	if err != nil {
		log.Error("Could not get tail block number of epoch", "epoch", ix.tailEpoch, "error", err)
		return
	}
	ix.snapshotsLock.Lock()
	ix.snapshots[hash] = &IndexView{
		refCount:         2,
		storage:          ix.storage,
		tailEpoch:        ix.tailEpoch,
		blockRange:       common.NewRange(tailNumber, number+1-tailNumber),
		headBlockHash:    hash,
		headLvPointer:    ix.headRenderer.lvPointer,
		headMap:          ix.headRenderer.currentMap.clone(),
		headMapIndex:     ix.headRenderer.mapIndex,
		firstMemoryMap:   firstMemoryMap,
		firstMemoryBlock: firstMemoryBlock,
		finishedMaps:     finishedMaps,
	}
	ix.snapshotsLock.Unlock()
	if number == ix.lastCanonical+1 {
		if len(ix.canonicalHashes) == maxCanonicalSnapshots {
			ix.releaseView(ix.canonicalHashes[0])
			copy(ix.canonicalHashes[0:maxCanonicalSnapshots-1], ix.canonicalHashes[1:maxCanonicalSnapshots])
			ix.canonicalHashes[maxCanonicalSnapshots-1] = hash
		} else {
			ix.canonicalHashes = append(ix.canonicalHashes, hash)
		}
	} else {
		for _, oldHash := range ix.canonicalHashes {
			ix.releaseView(oldHash)
		}
		ix.canonicalHashes = []common.Hash{hash}
	}
	ix.lastCanonical = number
	if len(ix.recentHashes) == maxRecentSnapshots {
		ix.releaseView(ix.recentHashes[0])
		copy(ix.recentHashes[0:maxRecentSnapshots-1], ix.recentHashes[1:maxRecentSnapshots])
		ix.recentHashes[maxRecentSnapshots-1] = hash
	} else {
		ix.recentHashes = append(ix.recentHashes, hash)
	}
	ix.updateActiveViewTailEpoch()
	ix.updateTailState()
}
