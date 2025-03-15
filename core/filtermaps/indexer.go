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
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	logFrequency = time.Second * 20 // log info frequency during long indexing/unindexing process
	headLogDelay = time.Second      // head indexing log info delay (do not log if finished faster)
)

// updateLoop initializes and updates the log index structure according to the
// current targetView.
func (f *FilterMaps) indexerLoop() {
	defer f.closeWg.Done()

	if f.disabled {
		f.reset()
		return
	}
	log.Info("Started log indexer")

	for !f.stop {
		if !f.indexedRange.initialized {
			if err := f.init(); err != nil {
				log.Error("Error initializing log index", "error", err)
				f.waitForEvent()
				continue
			}
		}
		if !f.targetHeadIndexed() {
			if !f.tryIndexHead() {
				f.waitForEvent()
			}
		} else {
			if f.finalBlock != f.lastFinal {
				if f.exportFileName != "" {
					f.exportCheckpoints()
				}
				f.lastFinal = f.finalBlock
			}
			if f.tryIndexTail() && f.tryUnindexTail() {
				f.waitForEvent()
			}
		}
	}
}

// SetTargetView sets a new target chain view for the indexer to render.
// Note that SetTargetView never blocks.
func (f *FilterMaps) SetTargetView(targetView *ChainView) {
	if targetView == nil {
		panic("nil targetView")
	}
	for {
		select {
		case <-f.targetViewCh:
		case f.targetViewCh <- targetView:
			return
		}
	}
}

// SetFinalBlock sets the finalized block number used for exporting checkpoints.
// Note that SetFinalBlock never blocks.
func (f *FilterMaps) SetFinalBlock(finalBlock uint64) {
	for {
		select {
		case <-f.finalBlockCh:
		case f.finalBlockCh <- finalBlock:
			return
		}
	}
}

// SetBlockProcessing sets the block processing flag that temporarily suspends
// log index rendering.
// Note that SetBlockProcessing never blocks.
func (f *FilterMaps) SetBlockProcessing(blockProcessing bool) {
	for {
		select {
		case <-f.blockProcessingCh:
		case f.blockProcessingCh <- blockProcessing:
			return
		}
	}
}

// WaitIdle blocks until the indexer is in an idle state while synced up to the
// latest targetView.
func (f *FilterMaps) WaitIdle() {
	if f.disabled {
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

// waitForEvent blocks until an event happens that the indexer might react to.
func (f *FilterMaps) waitForEvent() {
	for !f.stop && (f.blockProcessing || f.targetHeadIndexed()) {
		f.processSingleEvent(true)
	}
}

// processEvents processes all events, blocking only if a block processing is
// happening and indexing should be suspended.
func (f *FilterMaps) processEvents() {
	for !f.stop && f.processSingleEvent(f.blockProcessing) {
	}
}

// processSingleEvent processes a single event either in a blocking or
// non-blocking manner.
func (f *FilterMaps) processSingleEvent(blocking bool) bool {
	if f.matcherSyncRequest != nil && f.targetHeadIndexed() {
		f.matcherSyncRequest.synced()
		f.matcherSyncRequest = nil
	}
	if blocking {
		select {
		case targetView := <-f.targetViewCh:
			f.setTargetView(targetView)
		case f.finalBlock = <-f.finalBlockCh:
		case f.matcherSyncRequest = <-f.matcherSyncCh:
		case f.blockProcessing = <-f.blockProcessingCh:
		case <-f.closeCh:
			f.stop = true
		case ch := <-f.waitIdleCh:
			select {
			case targetView := <-f.targetViewCh:
				f.setTargetView(targetView)
			default:
			}
			ch <- !f.blockProcessing && f.targetHeadIndexed()
		}
	} else {
		select {
		case targetView := <-f.targetViewCh:
			f.setTargetView(targetView)
		case f.finalBlock = <-f.finalBlockCh:
		case f.matcherSyncRequest = <-f.matcherSyncCh:
		case f.blockProcessing = <-f.blockProcessingCh:
		case <-f.closeCh:
			f.stop = true
		default:
			return false
		}
	}
	return true
}

// setTargetView updates the target chain view of the iterator.
func (f *FilterMaps) setTargetView(targetView *ChainView) {
	f.targetView = targetView
}

// tryIndexHead tries to render head maps according to the current targetView
// and returns true if successful.
func (f *FilterMaps) tryIndexHead() bool {
	headRenderer, err := f.renderMapsBefore(math.MaxUint32)
	if err != nil {
		log.Error("Error creating log index head renderer", "error", err)
		return false
	}
	if headRenderer == nil {
		return true
	}
	if !f.startedHeadIndex {
		f.lastLogHeadIndex = time.Now()
		f.startedHeadIndexAt = f.lastLogHeadIndex
		f.startedHeadIndex = true
		f.ptrHeadIndex = f.indexedRange.afterLastIndexedBlock
	}
	if _, err := headRenderer.run(func() bool {
		f.processEvents()
		return f.stop
	}, func() {
		f.tryUnindexTail()
		if f.indexedRange.hasIndexedBlocks() && f.indexedRange.afterLastIndexedBlock >= f.ptrHeadIndex &&
			((!f.loggedHeadIndex && time.Since(f.startedHeadIndexAt) > headLogDelay) ||
				time.Since(f.lastLogHeadIndex) > logFrequency) {
			log.Info("Log index head rendering in progress",
				"first block", f.indexedRange.firstIndexedBlock, "last block", f.indexedRange.afterLastIndexedBlock-1,
				"processed", f.indexedRange.afterLastIndexedBlock-f.ptrHeadIndex,
				"remaining", f.indexedView.headNumber+1-f.indexedRange.afterLastIndexedBlock,
				"elapsed", common.PrettyDuration(time.Since(f.startedHeadIndexAt)))
			f.loggedHeadIndex = true
			f.lastLogHeadIndex = time.Now()
		}
	}); err != nil {
		log.Error("Log index head rendering failed", "error", err)
		return false
	}
	if f.loggedHeadIndex {
		log.Info("Log index head rendering finished",
			"first block", f.indexedRange.firstIndexedBlock, "last block", f.indexedRange.afterLastIndexedBlock-1,
			"processed", f.indexedRange.afterLastIndexedBlock-f.ptrHeadIndex,
			"elapsed", common.PrettyDuration(time.Since(f.startedHeadIndexAt)))
	}
	f.loggedHeadIndex, f.startedHeadIndex = false, false
	return true
}

// tryIndexTail tries to render tail epochs until the tail target block is
// indexed and returns true if successful.
// Note that tail indexing is only started if the log index head is fully
// rendered according to targetView and is suspended as soon as the targetView
// is changed.
func (f *FilterMaps) tryIndexTail() bool {
	for firstEpoch := f.indexedRange.firstRenderedMap >> f.logMapsPerEpoch; firstEpoch > 0 && f.needTailEpoch(firstEpoch-1); {
		f.processEvents()
		if f.stop || !f.targetHeadIndexed() {
			return false
		}
		// resume process if tail rendering was interrupted because of head rendering
		tailRenderer := f.tailRenderer
		f.tailRenderer = nil
		if tailRenderer != nil && tailRenderer.afterLastMap != f.indexedRange.firstRenderedMap {
			tailRenderer = nil
		}
		if tailRenderer == nil {
			var err error
			tailRenderer, err = f.renderMapsBefore(f.indexedRange.firstRenderedMap)
			if err != nil {
				log.Error("Error creating log index tail renderer", "error", err)
				return false
			}
		}
		if tailRenderer == nil {
			return true
		}
		if !f.startedTailIndex {
			f.lastLogTailIndex = time.Now()
			f.startedTailIndexAt = f.lastLogTailIndex
			f.startedTailIndex = true
			f.ptrTailIndex = f.indexedRange.firstIndexedBlock - f.tailPartialBlocks()
		}
		done, err := tailRenderer.run(func() bool {
			f.processEvents()
			return f.stop || !f.targetHeadIndexed()
		}, func() {
			tpb, ttb := f.tailPartialBlocks(), f.tailTargetBlock()
			remaining := uint64(1)
			if f.indexedRange.firstIndexedBlock > ttb+tpb {
				remaining = f.indexedRange.firstIndexedBlock - ttb - tpb
			}
			if f.indexedRange.hasIndexedBlocks() && f.ptrTailIndex >= f.indexedRange.firstIndexedBlock &&
				(!f.loggedTailIndex || time.Since(f.lastLogTailIndex) > logFrequency) {
				log.Info("Log index tail rendering in progress",
					"first block", f.indexedRange.firstIndexedBlock, "last block", f.indexedRange.afterLastIndexedBlock-1,
					"processed", f.ptrTailIndex-f.indexedRange.firstIndexedBlock+tpb,
					"remaining", remaining,
					"next tail epoch percentage", f.indexedRange.tailPartialEpoch*100/f.mapsPerEpoch,
					"elapsed", common.PrettyDuration(time.Since(f.startedTailIndexAt)))
				f.loggedTailIndex = true
				f.lastLogTailIndex = time.Now()
			}
		})
		if err != nil {
			log.Error("Log index tail rendering failed", "error", err)
		}
		if !done {
			f.tailRenderer = tailRenderer // only keep tail renderer if interrupted by stopCb
			return false
		}
	}
	if f.loggedTailIndex {
		log.Info("Log index tail rendering finished",
			"first block", f.indexedRange.firstIndexedBlock, "last block", f.indexedRange.afterLastIndexedBlock-1,
			"processed", f.ptrTailIndex-f.indexedRange.firstIndexedBlock,
			"elapsed", common.PrettyDuration(time.Since(f.startedTailIndexAt)))
		f.loggedTailIndex = false
	}
	return true
}

// tryUnindexTail removes entire epochs of log index data as long as the first
// fully indexed block is at least as old as the tail target.
// Note that unindexing is very quick as it only removes continuous ranges of
// data from the database and is also called while running head indexing.
func (f *FilterMaps) tryUnindexTail() bool {
	for {
		firstEpoch := (f.indexedRange.firstRenderedMap - f.indexedRange.tailPartialEpoch) >> f.logMapsPerEpoch
		if f.needTailEpoch(firstEpoch) {
			break
		}
		f.processEvents()
		if f.stop {
			return false
		}
		if !f.startedTailUnindex {
			f.startedTailUnindexAt = time.Now()
			f.startedTailUnindex = true
			f.ptrTailUnindexMap = f.indexedRange.firstRenderedMap - f.indexedRange.tailPartialEpoch
			f.ptrTailUnindexBlock = f.indexedRange.firstIndexedBlock - f.tailPartialBlocks()
		}
		if err := f.deleteTailEpoch(firstEpoch); err != nil {
			log.Error("Log index tail epoch unindexing failed", "error", err)
			return false
		}
	}
	if f.startedTailUnindex {
		log.Info("Log index tail unindexing finished",
			"first block", f.indexedRange.firstIndexedBlock, "last block", f.indexedRange.afterLastIndexedBlock-1,
			"removed maps", f.indexedRange.firstRenderedMap-f.ptrTailUnindexMap,
			"removed blocks", f.indexedRange.firstIndexedBlock-f.tailPartialBlocks()-f.ptrTailUnindexBlock,
			"elapsed", common.PrettyDuration(time.Since(f.startedTailUnindexAt)))
		f.startedTailUnindex = false
	}
	return true
}

// needTailEpoch returns true if the given tail epoch needs to be kept
// according to the current tail target, false if it can be removed.
func (f *FilterMaps) needTailEpoch(epoch uint32) bool {
	firstEpoch := f.indexedRange.firstRenderedMap >> f.logMapsPerEpoch
	if epoch > firstEpoch {
		return true
	}
	if epoch+1 < firstEpoch {
		return false
	}
	tailTarget := f.tailTargetBlock()
	if tailTarget < f.indexedRange.firstIndexedBlock {
		return true
	}
	tailLvIndex, err := f.getBlockLvPointer(tailTarget)
	if err != nil {
		log.Error("Could not get log value index of tail block", "error", err)
		return true
	}
	return uint64(epoch+1)<<(f.logValuesPerMap+f.logMapsPerEpoch) >= tailLvIndex
}

// tailTargetBlock returns the target value for the tail block number according
// to the log history parameter and the current index head.
func (f *FilterMaps) tailTargetBlock() uint64 {
	if f.history == 0 || f.indexedView.headNumber < f.history {
		return 0
	}
	return f.indexedView.headNumber + 1 - f.history
}

// tailPartialBlocks returns the number of rendered blocks in the partially
// rendered next tail epoch.
func (f *FilterMaps) tailPartialBlocks() uint64 {
	if f.indexedRange.tailPartialEpoch == 0 {
		return 0
	}
	end, _, err := f.getLastBlockOfMap(f.indexedRange.firstRenderedMap - f.mapsPerEpoch + f.indexedRange.tailPartialEpoch - 1)
	if err != nil {
		log.Error("Error fetching last block of map", "mapIndex", f.indexedRange.firstRenderedMap-f.mapsPerEpoch+f.indexedRange.tailPartialEpoch-1, "error", err)
	}
	var start uint64
	if f.indexedRange.firstRenderedMap-f.mapsPerEpoch > 0 {
		start, _, err = f.getLastBlockOfMap(f.indexedRange.firstRenderedMap - f.mapsPerEpoch - 1)
		if err != nil {
			log.Error("Error fetching last block of map", "mapIndex", f.indexedRange.firstRenderedMap-f.mapsPerEpoch-1, "error", err)
		}
	}
	return end - start
}

// targetHeadIndexed returns true if the current log index is consistent with
// targetView with its head block fully rendered.
func (f *FilterMaps) targetHeadIndexed() bool {
	return equalViews(f.targetView, f.indexedView) && f.indexedRange.headBlockIndexed
}
