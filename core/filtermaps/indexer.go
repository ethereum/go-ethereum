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
	"os"
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

var (
	mapCountGauge    = metrics.NewRegisteredGauge("filtermaps/maps/count", nil)      // actual number of rendered maps
	mapLogValueMeter = metrics.NewRegisteredMeter("filtermaps/maps/logvalues", nil)  // number of log values processed
	mapBlockMeter    = metrics.NewRegisteredMeter("filtermaps/maps/blocks", nil)     // number of block delimiters processed
	mapRenderTimer   = metrics.NewRegisteredTimer("filtermaps/maps/rendertime", nil) // time elapsed while rendering a single map
	mapWriteTimer    = metrics.NewRegisteredTimer("filtermaps/maps/writetime", nil)  // time elapsed while writing a batch of finished maps to db
)

// Indexer maintains a search data structure based on a parent blockchain that
// is intended to make log event search more efficient. Once indexed up to the
// chain head, it provides IndexView objects for recent chain heads.
// Indexer implements core.Indexer.
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
	lastFinalEpoch                                  uint32
}

// Config contains the configuration options for Indexer.
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

// NewIndexer creates a new Indexer.
func NewIndexer(db ethdb.KeyValueStore, params Params, config Config) *Indexer {
	params.sanitize()
	mapDb := newMapDatabase(&params, db, config.HashScheme)
	ix := &Indexer{
		config:         config,
		storage:        newMapStorage(&params, mapDb, nil),
		checkpoints:    checkpoints,
		snapshots:      make(map[common.Hash]*IndexView),
		headMapsCache:  lru.NewCache[uint32, *finishedMap](maxIndexViewMaps),
		lastFinalEpoch: math.MaxUint32,
	}
	if config.Disabled {
		ix.storage.deleteMaps(common.NewRange[uint32](0, math.MaxUint32))
		return ix
	}
	ix.headRenderer = ix.initMapBoundary(ix.storage.lastBoundaryBefore(math.MaxUint32), math.MaxUint32)
	ix.updateTailEpoch()
	ix.updateActiveViewTailEpoch()
	ix.updateTailState()
	return ix
}

// GetIndexView returns an immutable IndexView corresponding to the given head
// block hash if available. Note that each returned IndexView has to be released
// after use with the Release funcion in order to avoid memory leakage.
func (ix *Indexer) GetIndexView(headBlockHash common.Hash) *IndexView {
	if ix.config.Disabled {
		return nil
	}
	ix.snapshotsLock.RLock()
	iv := ix.snapshots[headBlockHash]
	ix.snapshotsLock.RUnlock()
	if iv == nil || iv.checkReleased() {
		return nil
	}
	iv.addRefCount(1)
	return iv
}

// Status returns the current indexer status. The ready flag indicates whether
// the indexer is ready to process new block data. The needBlocks range, if not
// empty, indicates that the indexer requests past blocks in order to complete
// the index. These blocks should be delivered in strictly ascending order.
// Note that if ready is false then needBlocks might still be non-empty, in
// which case the blocks are not expected to be delivered yet but the index
// server might already start pre-fetching them.
// Status implements core.Indexer.
func (ix *Indexer) Status() (bool, common.Range[uint64]) {
	if ix.config.Disabled {
		return false, common.Range[uint64]{}
	}
	return ix.storage.isReady(), ix.needBlocks()
}

// AddBlockData delivers block data for new heads and requested historical range.
// It returns the indexer status. Unwanted data is silently ignored. If the
// indexer is not ready to process then the received data is also ignored and
// then requested through the needBlocks response either in the current response
// or later.
// Note that this function also resumes the storage layer background process if
// it was previously suspended.
// AddBlockData implements core.Indexer.
func (ix *Indexer) AddBlockData(header *types.Header, receipts types.Receipts) (ready bool, needBlocks common.Range[uint64]) {
	if ix.config.Disabled {
		return false, common.Range[uint64]{}
	}
	ix.storage.suspendOrResume(false)
	if !ix.storage.isReady() {
		return false, ix.needBlocks()
	}
	ix.headNumber = max(ix.headNumber, header.Number.Uint64())
	number, hash := header.Number.Uint64(), header.Hash()
	if number > ix.headRenderer.nextBlock {
		ix.tryCheckpointInit(number, hash)
	}
	if number == ix.headRenderer.nextBlock {
		if ix.headRenderer.checkNextHash(hash) {
			ix.headRenderer.addReceipts(receipts)
			firstMapIndex, finishedMaps := ix.headRenderer.addHeader(header)
			ix.storeFinishedMaps(firstMapIndex, finishedMaps, true, true)
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
			ix.tailRenderer.addReceipts(receipts)
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
			ix.tailRenderer = nil
		}
	}
	return ix.storage.isReady(), ix.needBlocks()
}

// Revert resets the index head to the given block number. Note that the indexer
// might have to discard more data if a snapshot is not available for the given
// block number. In this case it will request previously delivered but discarded
// block data through the needBlocks status response.
// Note that Revert works even if the indexer is in a "not ready" status, thereby
// guaranteeing that all index data is always consistent with the canonical chain.
// Revert implements core.Indexer.
func (ix *Indexer) Revert(blockNumber uint64) {
	if ix.config.Disabled {
		return
	}
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
		mapIndex = ix.storage.lastBoundaryBefore(mapIndex - 1)
		if mapIndex == 0 {
			break
		}
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
	ix.updateTailEpoch()
}

// SetFinalized notifies the indexer about the latest finalized block number.
// SetFinalized implements core.Indexer.
func (ix *Indexer) SetFinalized(blockNumber uint64) {
	if ix.finalized == blockNumber {
		return
	}
	ix.finalized = blockNumber
	ix.exportCheckpoints()
}

// SetHistoryCutoff notifies the indexer about the latest historical cutoff point.
// The indexer will not request block data earlier than this point.
// SetHistoryCutoff implements core.Indexer.
func (ix *Indexer) SetHistoryCutoff(blockNumber uint64) {
	ix.historyCutoff = blockNumber
}

// Suspended suspends the asynchronous storage layer background process during
// block processing. The next AddBlockData call will resume this process.
// Suspended implements core.Indexer.
func (ix *Indexer) Suspended() {
	if ix.config.Disabled {
		return
	}
	ix.storage.suspendOrResume(true)
}

// initMapBoundary initializes a new map renderer at the last suitable map
// boundary before startMap. If this boundary is not right before startMap then
// startMap is lowered to right after the boundary. The returned renderState
// will render maps in the startMap..limitMap-1 range.
// Note that the first requested block typically still starts in the previous
// map and in case of tail renderers with an upper map limit, the last requested
// block typically ends after the upper limit. In this case the maps outside the
// rendered range are not modified, the log values outside the range are ignored.
func (ix *Indexer) initMapBoundary(startMap, limitMap uint32) *renderState {
	rs := &renderState{
		params: ix.storage.params,
	}
	for {
		startMap = ix.storage.lastBoundaryBefore(startMap)
		if startMap == 0 {
			break
		}
		lastNumber, lastHash, err := ix.storage.getLastBlockOfMap(startMap - 1)
		if err != nil {
			log.Error("Last block of map not found, reverting database", "mapIndex", startMap-1)
			startMap = ix.storage.lastBoundaryBefore(startMap - 1)
			ix.revertMaps(startMap)
			continue
		}
		lvPointer, err := ix.storage.getBlockLvPointer(lastNumber)
		if err != nil {
			log.Error("Block pointer of last block of map not found, reverting database", "mapIndex", startMap-1, "blockNumber", lastNumber)
			startMap = ix.storage.lastBoundaryBefore(startMap - 1)
			ix.revertMaps(startMap)
			continue
		}
		rs.lvPointer = lvPointer
		rs.mapIndex = uint32(lvPointer >> ix.storage.params.logValuesPerMap)
		rs.nextBlock = lastNumber
		rs.partialBlock = true
		rs.partialBlockHash = lastHash
		break
	}
	rs.renderRange = common.NewRange[uint32](startMap, limitMap-startMap)
	if rs.renderRange.Includes(rs.mapIndex) {
		rs.currentMap = rs.params.newMemoryMap()
	}
	return rs
}

// initSnapshot initializes a new map renderer based on a snapshot. Since this
// method is only used to initialize head renderers, a snapshot initialized
// renderState always has an upper render limit of MaxUint32-1.
func (ix *Indexer) initSnapshot(snapshot *IndexView) *renderState {
	mapIndex := ix.storage.lastBoundaryBefore(snapshot.firstMemoryMap)
	ix.revertMaps(mapIndex)
	if snapshot.checkInvalid() {
		log.Error("Failed to revert to invalidated snapshot", "blockNumber", snapshot.blockRange.Last())
		return nil
	}

	return &renderState{
		params:      ix.storage.params,
		renderRange: common.NewRange[uint32](snapshot.headMapIndex, math.MaxUint32-snapshot.headMapIndex),
		currentMap:  snapshot.headMap.clone(),
		mapIndex:    snapshot.headMapIndex,
		lvPointer:   snapshot.headLvPointer,
	}
}

// revertMaps removes all rendered maps starting from mapIndex. It also removes
// active renderers and snapshots invalidated by the revert and purges the map
// cache.
// Note that while headRenderer generally always exists, revertMaps might be
// called while headRenderer is nil and might set headRenderer to nil.
func (ix *Indexer) revertMaps(mapIndex uint32) {
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

// updateTailEpoch recalculates the current tailEpoch and the targetTailEpoch.
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
		if nextTailNumber+ix.config.History > ix.headRenderer.nextBlock {
			break
		}
		ix.targetTailEpoch++
	}
	for ix.targetTailEpoch > 0 {
		prevTailEpoch := ix.storage.params.mapEpoch(ix.storage.lastBoundaryBefore(ix.storage.params.firstEpochMap(ix.targetTailEpoch - 1)))
		prevTailNumber, err := ix.storage.tailNumberOfEpoch(prevTailEpoch)
		if err != nil {
			log.Error("Could not get tail block number of epoch", "epoch", prevTailEpoch, "error", err)
			return
		}
		if prevTailNumber+ix.config.History <= ix.headRenderer.nextBlock {
			break
		}
		ix.targetTailEpoch = prevTailEpoch
	}
}

// updateActiveViewTailEpoch recalculates activeViewTailEpoch which is the earliest
// tail epoch required by an active IndexView. Tail unindexing is only allowed
// if min(targetTailEpoch, activeViewTailEpoch) > tailEpoch.
func (ix *Indexer) updateActiveViewTailEpoch() {
	ix.snapshotsLock.RLock()
	defer ix.snapshotsLock.RUnlock()

	ix.activeViewTailEpoch = math.MaxUint32
	for _, iv := range ix.snapshots {
		ix.activeViewTailEpoch = min(ix.activeViewTailEpoch, iv.tailEpoch)
	}
}

// updateTailState performs tail unindexing or initializes a new tailRenderer to
// render a new tail epoch if necessary.
func (ix *Indexer) updateTailState() {
	epoch := min(ix.targetTailEpoch, ix.activeViewTailEpoch)
	if epoch >= ix.tailEpoch && ix.tailRenderer != nil {
		ix.tailRenderer = nil
		ix.storage.deleteMaps(common.NewRange[uint32](ix.storage.params.firstEpochMap(ix.tailEpoch-1), ix.storage.params.mapsPerEpoch))
	}
	if epoch > ix.tailEpoch {
		ix.storage.deleteMaps(common.NewRange[uint32](ix.storage.params.firstEpochMap(ix.tailEpoch), ix.storage.params.mapsPerEpoch*(epoch-ix.tailEpoch)))
		if epoch == ix.tailEpoch+1 {
			log.Info("Unindexed tail epoch #%d", ix.tailEpoch)
		} else {
			log.Info("Unindexed tail epochs #%d to #%d", ix.tailEpoch, epoch-1)
		}
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
}

// epochsUntilBlock returns the numer of epochs in the checkpoint list whose
// last block number is less than or equal to the specified number.
func (cpList checkpointList) epochsUntilBlock(number uint64) uint32 {
	first, last := uint32(0), uint32(len(cpList))
	for first < last {
		mid := (first + last) / 2
		if cpList[mid].BlockNumber > number {
			last = mid
		} else {
			first = mid + 1
		}
	}
	return first
}

func (ix *Indexer) tryCheckpointInit(number uint64, hash common.Hash) {
	var ci int
	for ci < len(ix.checkpoints) {
		cpList := ix.checkpoints[ci]
		epochs := cpList.epochsUntilBlock(number)
		if epochs == 0 || cpList[epochs-1].BlockNumber != number {
			// block number does not match, skip list (a relevant block might match later)
			ci++
			continue
		}
		if cpList[epochs-1].BlockHash == hash {
			// apply matching checkpoint, discard other lists
			if err := ix.storage.addKnownEpochs(cpList[:epochs]); err == nil {
				ix.checkpoints = []checkpointList{cpList}
				ix.headRenderer = ix.initMapBoundary(ix.storage.params.firstEpochMap(epochs), math.MaxUint32)
				return
			} else {
				log.Error("Error initializing epoch boundaries", "error", err)
			}
		}
		// checkpoint does not match, discard list
		ix.checkpoints[ci] = ix.checkpoints[len(ix.checkpoints)-1]
		ix.checkpoints = ix.checkpoints[:len(ix.checkpoints)-1]
	}
}

func (ix *Indexer) needBlocks() common.Range[uint64] {
	if ix.finalized > ix.headRenderer.nextBlock {
		// request potential checkpoint in this range if available
		for _, cpList := range ix.checkpoints {
			if epochs := cpList.epochsUntilBlock(ix.headNumber); epochs > 0 {
				blockNumber := cpList[epochs-1].BlockNumber
				if ix.storage.lastBoundaryBefore(math.MaxUint32) >= ix.storage.params.firstEpochMap(epochs) ||
					blockNumber <= ix.headRenderer.nextBlock || blockNumber < ix.historyCutoff {
					continue
				}
				return common.NewRange[uint64](blockNumber, 1)
			}
		}
	}
	if ix.headRenderer.nextBlock <= ix.headNumber && ix.headRenderer.nextBlock >= ix.historyCutoff {
		return common.NewRange[uint64](ix.headRenderer.nextBlock, ix.headNumber+1-ix.headRenderer.nextBlock)
	}
	if ix.tailRenderer != nil &&
		ix.tailRenderer.nextBlock <= ix.tailRenderLast && ix.tailRenderer.nextBlock >= ix.historyCutoff {
		return common.NewRange[uint64](ix.tailRenderer.nextBlock, ix.tailRenderLast+1-ix.tailRenderer.nextBlock)
	}
	return common.Range[uint64]{}
}

func (ix *Indexer) Stop() {
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
	if ix.headRenderer.currentMap == nil {
		return
	}
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
	tailEpoch := max(ix.tailEpoch, ix.targetTailEpoch)
	tailNumber, err := ix.storage.tailNumberOfEpoch(tailEpoch)
	if err != nil {
		log.Error("Could not get tail block number of epoch", "epoch", tailEpoch, "error", err)
		return
	}
	ix.snapshotsLock.Lock()
	ix.snapshots[hash] = &IndexView{
		refCount:         2,
		storage:          ix.storage,
		tailEpoch:        tailEpoch,
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

// exportCheckpoints exports epoch checkpoints in the format used by checkpoints.go.
func (ix *Indexer) exportCheckpoints() {
	finalLvPtr, err := ix.storage.getBlockLvPointer(ix.finalized + 1)
	if err != nil {
		if err != ErrOutOfRange {
			log.Error("Error fetching log value pointer of finalized block", "block", ix.finalized, "error", err)
		}
		return
	}
	epochCount := ix.storage.params.mapEpoch(uint32(finalLvPtr >> ix.storage.params.logValuesPerMap))
	if epochCount == ix.lastFinalEpoch {
		return
	}
	w, err := os.Create(ix.config.ExportFileName)
	if err != nil {
		log.Error("Error creating checkpoint export file", "name", ix.config.ExportFileName, "error", err)
		return
	}
	defer w.Close()

	log.Info("Exporting log index checkpoints", "epochs", epochCount, "file", ix.config.ExportFileName)
	w.WriteString("[\n")
	comma := ","
	for epoch := uint32(0); epoch < epochCount; epoch++ {
		lastBlock, lastBlockId, err := ix.storage.getLastBlockOfMap(ix.storage.params.lastEpochMap(epoch))
		if err != nil {
			log.Error("Error fetching last block of epoch", "epoch", epoch, "error", err)
			return
		}
		lvPtr, err := ix.storage.getBlockLvPointer(lastBlock)
		if err != nil {
			log.Error("Error fetching log value pointer of last block", "block", lastBlock, "error", err)
			return
		}
		if epoch == epochCount-1 {
			comma = ""
		}
		w.WriteString(fmt.Sprintf("{\"blockNumber\": %d, \"blockId\": \"0x%064x\", \"firstIndex\": %d}%s\n", lastBlock, lastBlockId, lvPtr, comma))
	}
	w.WriteString("]\n")
	ix.lastFinalEpoch = epochCount
}
