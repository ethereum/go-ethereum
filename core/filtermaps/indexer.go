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
		close(f.disabledCh)
		return
	}
	log.Info("Started log indexer")

	for !f.stop {
		// Note: acquiring the indexLock read lock is unnecessary here,
		// as the `indexedRange` is accessed within the indexerLoop.
		if !f.indexedRange.initialized {
			if f.targetView.HeadNumber() == 0 {
				// initialize when chain head is available
				f.processSingleEvent(true)
				continue
			}
			if err := f.init(); err != nil {
				f.disableForError("initialization", err)
				f.reset() // remove broken index from DB
				return
			}
		}
		if !f.targetHeadIndexed() {
			if err := f.tryIndexHead(); err != nil && err != errChainUpdate {
				f.disableForError("head rendering", err)
				return
			}
		} else {
			if f.finalBlock != f.lastFinal {
				if f.exportFileName != "" {
					f.exportCheckpoints()
				}
				f.lastFinal = f.finalBlock
			}
			// always attempt unindexing before indexing the tail in order to
			// ensure that a potentially dirty previously unindexed epoch is
			// always cleaned up before any new maps are rendered.
			if done, err := f.tryUnindexTail(); err != nil {
				f.disableForError("tail unindexing", err)
				return
			} else if !done {
				continue
			}
			if done, err := f.tryIndexTail(); err != nil {
				f.disableForError("tail rendering", err)
				return
			} else if !done {
				continue
			}
			// tail indexing/unindexing is done; if head is also indexed then
			// wait here until there is a new head
			f.waitForNewHead()
		}
	}
}

// disableForError is called when the indexer encounters a database error, for example a
// missing receipt. We can't continue operating when the database is broken, so the
// indexer goes into disabled state.
// Note that the partial index is left in disk; maybe a client update can fix the
// issue without reindexing.
func (f *FilterMaps) disableForError(op string, err error) {
	log.Error("Log index "+op+" failed, reverting to unindexed mode", "error", err)
	f.disabled = true
	close(f.disabledCh)
}

type targetUpdate struct {
	targetView                *ChainView
	historyCutoff, finalBlock uint64
}

// SetTarget sets a new target chain view for the indexer to render.
// Note that SetTargetView never blocks.
func (f *FilterMaps) SetTarget(targetView *ChainView, historyCutoff, finalBlock uint64) {
	if targetView == nil {
		panic("nil targetView")
	}
	for {
		select {
		case <-f.targetCh:
		case f.targetCh <- targetUpdate{
			targetView:    targetView,
			historyCutoff: historyCutoff,
			finalBlock:    finalBlock,
		}:
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
	for {
		ch := make(chan bool)
		select {
		case f.waitIdleCh <- ch:
			if <-ch {
				return
			}
		case <-f.disabledCh:
			f.closeWg.Wait()
			return
		}
	}
}

// waitForNewHead blocks until there is a new target head to index and block
// processing has been finished.
func (f *FilterMaps) waitForNewHead() {
	for !f.stop && (f.blockProcessing || f.targetHeadIndexed()) {
		f.processSingleEvent(true)
	}
}

// processEvents processes all events, blocking only if a block processing is
// happening and indexing should be suspended.
func (f *FilterMaps) processEvents() {
	if f.testProcessEventsHook != nil {
		f.testProcessEventsHook()
	}
	for f.processSingleEvent(f.blockProcessing) {
	}
}

// processSingleEvent processes a single event either in a blocking or
// non-blocking manner. It returns true if it did process an event.
func (f *FilterMaps) processSingleEvent(blocking bool) bool {
	if f.stop {
		return false
	}
	// Note: acquiring the indexLock read lock is unnecessary here,
	// as this function is always called within the indexLoop.
	if !f.hasTempRange {
		for _, mb := range f.matcherSyncRequests {
			mb.synced()
		}
		f.matcherSyncRequests = nil
	}
	if blocking {
		select {
		case target := <-f.targetCh:
			f.setTarget(target)
		case mb := <-f.matcherSyncCh:
			f.matcherSyncRequests = append(f.matcherSyncRequests, mb)
		case f.blockProcessing = <-f.blockProcessingCh:
		case <-f.closeCh:
			f.stop = true
		case ch := <-f.waitIdleCh:
			select {
			case target := <-f.targetCh:
				f.setTarget(target)
			default:
			}
			ch <- !f.blockProcessing && f.targetHeadIndexed()
		}
	} else {
		select {
		case target := <-f.targetCh:
			f.setTarget(target)
		case mb := <-f.matcherSyncCh:
			f.matcherSyncRequests = append(f.matcherSyncRequests, mb)
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
func (f *FilterMaps) setTarget(target targetUpdate) {
	f.targetView = target.targetView
	f.historyCutoff = target.historyCutoff
	f.finalBlock = target.finalBlock
}

// tryIndexHead tries to render head maps according to the current targetView.
// Should be called when targetHeadIndexed returns false. If this function
// returns no error then either stop is true or head indexing is finished.
func (f *FilterMaps) tryIndexHead() error {
	headRenderer, err := f.renderMapsBefore(math.MaxUint32)
	if err != nil {
		return err
	}
	if headRenderer == nil {
		return errors.New("head indexer has nothing to do") // tryIndexHead should be called when head is not indexed
	}
	if !f.startedHeadIndex {
		f.lastLogHeadIndex = time.Now()
		f.startedHeadIndexAt = f.lastLogHeadIndex
		f.startedHeadIndex = true
		f.ptrHeadIndex = f.indexedRange.blocks.AfterLast()
	}
	if _, err := headRenderer.run(func() bool {
		f.processEvents()
		return f.stop
	}, func() {
		f.tryUnindexTail()
		if f.indexedRange.hasIndexedBlocks() && f.indexedRange.blocks.AfterLast() >= f.ptrHeadIndex &&
			((!f.loggedHeadIndex && time.Since(f.startedHeadIndexAt) > headLogDelay) ||
				time.Since(f.lastLogHeadIndex) > logFrequency) {
			log.Info("Log index head rendering in progress",
				"firstblock", f.indexedRange.blocks.First(), "lastblock", f.indexedRange.blocks.Last(),
				"processed", f.indexedRange.blocks.AfterLast()-f.ptrHeadIndex,
				"remaining", f.indexedView.HeadNumber()-f.indexedRange.blocks.Last(),
				"elapsed", common.PrettyDuration(time.Since(f.startedHeadIndexAt)))
			f.loggedHeadIndex = true
			f.lastLogHeadIndex = time.Now()
		}
	}); err != nil {
		return err
	}
	if f.loggedHeadIndex && f.indexedRange.hasIndexedBlocks() {
		log.Info("Log index head rendering finished",
			"firstblock", f.indexedRange.blocks.First(), "lastblock", f.indexedRange.blocks.Last(),
			"processed", f.indexedRange.blocks.AfterLast()-f.ptrHeadIndex,
			"elapsed", common.PrettyDuration(time.Since(f.startedHeadIndexAt)))
	}
	f.loggedHeadIndex, f.startedHeadIndex = false, false
	return nil
}

// tryIndexTail tries to render tail epochs until the tail target block is
// indexed and returns true if successful.
// Note that tail indexing is only started if the log index head is fully
// rendered according to targetView and is suspended as soon as the targetView
// is changed.
func (f *FilterMaps) tryIndexTail() (bool, error) {
	for {
		firstEpoch := f.indexedRange.maps.First() >> f.logMapsPerEpoch
		if firstEpoch == 0 || !f.needTailEpoch(firstEpoch-1) {
			break
		}
		f.processEvents()
		if f.stop || !f.targetHeadIndexed() {
			return false, nil
		}
		// resume process if tail rendering was interrupted because of head rendering
		tailRenderer := f.tailRenderer
		f.tailRenderer = nil
		if tailRenderer != nil && tailRenderer.renderBefore != f.indexedRange.maps.First() {
			tailRenderer = nil
		}
		if tailRenderer == nil {
			var err error
			tailRenderer, err = f.renderMapsBefore(f.indexedRange.maps.First())
			if err != nil {
				return false, err
			}
		}
		if tailRenderer == nil {
			break
		}
		if !f.startedTailIndex {
			f.lastLogTailIndex = time.Now()
			f.startedTailIndexAt = f.lastLogTailIndex
			f.startedTailIndex = true
			f.ptrTailIndex = f.indexedRange.blocks.First() - f.tailPartialBlocks()
		}
		done, err := tailRenderer.run(func() bool {
			f.processEvents()
			return f.stop || !f.targetHeadIndexed()
		}, func() {
			tpb, ttb := f.tailPartialBlocks(), f.tailTargetBlock()
			remaining := uint64(1)
			if f.indexedRange.blocks.First() > ttb+tpb {
				remaining = f.indexedRange.blocks.First() - ttb - tpb
			}
			if f.indexedRange.hasIndexedBlocks() && f.ptrTailIndex >= f.indexedRange.blocks.First() &&
				(!f.loggedTailIndex || time.Since(f.lastLogTailIndex) > logFrequency) {
				log.Info("Log index tail rendering in progress",
					"firstblock", f.indexedRange.blocks.First(), "last block", f.indexedRange.blocks.Last(),
					"processed", f.ptrTailIndex-f.indexedRange.blocks.First()+tpb,
					"remaining", remaining,
					"next tail epoch percentage", f.indexedRange.tailPartialEpoch*100/f.mapsPerEpoch,
					"elapsed", common.PrettyDuration(time.Since(f.startedTailIndexAt)))
				f.loggedTailIndex = true
				f.lastLogTailIndex = time.Now()
			}
		})
		if err != nil && !f.needTailEpoch(firstEpoch-1) {
			// stop silently if cutoff point has move beyond epoch boundary while rendering
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if !done {
			f.tailRenderer = tailRenderer // only keep tail renderer if interrupted by stopCb
			return false, nil
		}
	}
	if f.loggedTailIndex && f.indexedRange.hasIndexedBlocks() {
		log.Info("Log index tail rendering finished",
			"firstblock", f.indexedRange.blocks.First(), "lastblock", f.indexedRange.blocks.Last(),
			"processed", f.ptrTailIndex-f.indexedRange.blocks.First(),
			"elapsed", common.PrettyDuration(time.Since(f.startedTailIndexAt)))
		f.loggedTailIndex = false
	}
	return true, nil
}

// tryUnindexTail removes entire epochs of log index data as long as the first
// fully indexed block is at least as old as the tail target.
// Note that unindexing is very quick as it only removes continuous ranges of
// data from the database and is also called while running head indexing.
func (f *FilterMaps) tryUnindexTail() (bool, error) {
	firstEpoch := f.indexedRange.maps.First() >> f.logMapsPerEpoch
	if f.indexedRange.tailPartialEpoch > 0 && firstEpoch > 0 {
		firstEpoch--
	}
	for epoch := min(firstEpoch, f.cleanedEpochsBefore); !f.needTailEpoch(epoch); epoch++ {
		if !f.startedTailUnindex {
			f.startedTailUnindexAt = time.Now()
			f.startedTailUnindex = true
			f.ptrTailUnindexMap = f.indexedRange.maps.First() - f.indexedRange.tailPartialEpoch
			f.ptrTailUnindexBlock = f.indexedRange.blocks.First() - f.tailPartialBlocks()
		}
		if done, err := f.deleteTailEpoch(epoch); !done {
			return false, err
		}
		f.processEvents()
		if f.stop || !f.targetHeadIndexed() {
			return false, nil
		}
	}
	if f.startedTailUnindex && f.indexedRange.hasIndexedBlocks() {
		log.Info("Log index tail unindexing finished",
			"first block", f.indexedRange.blocks.First(), "last block", f.indexedRange.blocks.Last(),
			"removed maps", f.indexedRange.maps.First()-f.ptrTailUnindexMap,
			"removed blocks", f.indexedRange.blocks.First()-f.tailPartialBlocks()-f.ptrTailUnindexBlock,
			"elapsed", common.PrettyDuration(time.Since(f.startedTailUnindexAt)))
		f.startedTailUnindex = false
	}
	return true, nil
}

// needTailEpoch returns true if the given tail epoch needs to be kept
// according to the current tail target, false if it can be removed.
func (f *FilterMaps) needTailEpoch(epoch uint32) bool {
	firstEpoch := f.indexedRange.maps.First() >> f.logMapsPerEpoch
	if epoch > firstEpoch {
		return true
	}
	if (epoch+1)<<f.logMapsPerEpoch >= f.indexedRange.maps.AfterLast() {
		return true
	}
	if epoch+1 < firstEpoch {
		return false
	}
	var lastBlockOfPrevEpoch uint64
	if epoch > 0 {
		var err error
		lastBlockOfPrevEpoch, _, err = f.getLastBlockOfMap(epoch<<f.logMapsPerEpoch - 1)
		if err != nil {
			log.Error("Could not get last block of previous epoch", "epoch", epoch-1, "error", err)
			return epoch >= firstEpoch
		}
	}
	if f.historyCutoff > lastBlockOfPrevEpoch {
		return false
	}
	lastBlockOfEpoch, _, err := f.getLastBlockOfMap((epoch+1)<<f.logMapsPerEpoch - 1)
	if err != nil {
		log.Error("Could not get last block of epoch", "epoch", epoch, "error", err)
		return epoch >= firstEpoch
	}
	return f.tailTargetBlock() <= lastBlockOfEpoch
}

// tailTargetBlock returns the target value for the tail block number according
// to the log history parameter and the current index head.
func (f *FilterMaps) tailTargetBlock() uint64 {
	if f.history == 0 || f.indexedView.HeadNumber() < f.history {
		return 0
	}
	return f.indexedView.HeadNumber() + 1 - f.history
}

// tailPartialBlocks returns the number of rendered blocks in the partially
// rendered next tail epoch.
func (f *FilterMaps) tailPartialBlocks() uint64 {
	if f.indexedRange.tailPartialEpoch == 0 {
		return 0
	}
	end, _, err := f.getLastBlockOfMap(f.indexedRange.maps.First() - f.mapsPerEpoch + f.indexedRange.tailPartialEpoch - 1)
	if err != nil {
		log.Error("Error fetching last block of map", "mapIndex", f.indexedRange.maps.First()-f.mapsPerEpoch+f.indexedRange.tailPartialEpoch-1, "error", err)
	}
	var start uint64
	if f.indexedRange.maps.First()-f.mapsPerEpoch > 0 {
		start, _, err = f.getLastBlockOfMap(f.indexedRange.maps.First() - f.mapsPerEpoch - 1)
		if err != nil {
			log.Error("Error fetching last block of map", "mapIndex", f.indexedRange.maps.First()-f.mapsPerEpoch-1, "error", err)
		}
	}
	return end - start
}

// targetHeadIndexed returns true if the current log index is consistent with
// targetView with its head block fully rendered.
func (f *FilterMaps) targetHeadIndexed() bool {
	return equalViews(f.targetView, f.indexedView) && f.indexedRange.headIndexed
}
