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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	busyDelay            = time.Second // indexer status polling frequency when not ready
	maxHistoricPrefetch  = 16          // size of block data pre-fetch queue
	rawReceiptsCacheSize = 8
	logFrequency         = time.Second * 20 // log info frequency during long indexing/unindexing process
	headLogDelay         = time.Second      // head indexing log info delay (do not log if finished faster)
)

type Indexer interface {
	// AddBlockData delivers a header and receipts belonging to a block that is
	// either a direct descendant of the latest delivered head or the first one
	// in the last requested range.
	// The current ready/busy status and the requested historic range are returned.
	// Note that the indexer should never block even if it is busy processing.
	// It is allowed to re-request the delivered blocks later if the indexer could
	// not process them when first delivered.
	AddBlockData(header *types.Header, body *types.Body, receipts types.Receipts) (ready bool, needBlocks common.Range[uint64])
	// Revert rewinds the index to the given head block number. Subsequent
	// AddBlockData calls will deliver blocks starting from this point.
	Revert(blockNumber uint64)
	// Status returns the current ready/busy status and the requested historic range.
	// Only the new head blocks are delivered if the indexer reports busy status.
	Status() (ready bool, needBlocks common.Range[uint64])
	// SetHistoryCutoff signals the historical cutoff point to the indexer.
	// Note that any block number that is consistently being requested in the
	// needBlocks response that is not older than the cutoff point is guaranteed
	// to be delivered eventually. If the required data belonging to certain
	// block numbers is missing then the cutoff point is moved after the missing
	// section in order to maintain this guarantee.
	SetHistoryCutoff(blockNumber uint64)
	// SetFinalized signals the latest finalized block number to the indexer.
	SetFinalized(blockNumber uint64)
	// Suspended signals to the indexer that block processing has started and
	// any non-essential asynchronous tasks of the indexer should be suspended.
	// The next AddBlockData call signals the end of the suspended state.
	// Note that if multiple blocks are inserted then the indexer is only
	// suspended once, before the first block processing begins, so according
	// to the rule above it will not be suspended while processing the rest of
	// the batch. This behavior should be fine because indexing can happen in
	// parallel with forward syncing, the purpose of the suspend mechanism is
	// to handle historical index backfilling with a lower priority so that it
	// does not increase block latency.
	Suspended()
	// Stop initiates indexer shutdown. No subsequent calls are made through this
	// interface after Stop.
	Stop()
}

// indexServers operates as a part of BlockChain and can serve multiple chain
// indexers that implement the Indexer interface.
type indexServers struct {
	lock             sync.Mutex
	servers          []*indexServer
	chain            *BlockChain
	rawReceiptsCache *lru.Cache[common.Hash, []*types.Receipt]

	lastHead                  *types.Header
	lastHeadBody              *types.Body
	lastHeadReceipts          types.Receipts
	finalBlock, historyCutoff uint64

	closeCh chan struct{}
	closeWg sync.WaitGroup
}

// init initializes indexServers.
func (f *indexServers) init(chain *BlockChain) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.chain = chain
	f.lastHead = chain.CurrentBlock()
	if f.lastHead != nil {
		f.lastHeadBody = chain.GetBody(f.lastHead.Hash())
		f.lastHeadReceipts = chain.GetRawReceipts(f.lastHead.Hash(), f.lastHead.Number.Uint64())
	}
	f.closeCh = make(chan struct{})
	f.rawReceiptsCache = lru.NewCache[common.Hash, []*types.Receipt](rawReceiptsCacheSize)
}

// stop shuts down all registered Indexers and their serving goroutines.
func (f *indexServers) stop() {
	f.lock.Lock()
	defer f.lock.Unlock()

	close(f.closeCh)
	f.closeWg.Wait()
	f.servers = nil
}

// register adds a new Indexer to the chain.
func (f *indexServers) register(indexer Indexer, name string, needBodies, needReceipts bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	server := &indexServer{
		parent:       f,
		indexer:      indexer,
		sendTimer:    time.NewTimer(0),
		name:         name,
		statusCh:     make(chan indexerStatus, 1),
		blockDataCh:  make(chan blockData, maxHistoricPrefetch),
		suspendCh:    make(chan bool, 1),
		needBodies:   needBodies,
		needReceipts: needReceipts,
	}
	f.servers = append(f.servers, server)
	f.closeWg.Add(2)
	indexer.SetHistoryCutoff(f.historyCutoff)
	indexer.SetFinalized(f.finalBlock)
	if f.lastHead != nil && f.lastHeadBody != nil && f.lastHeadReceipts != nil {
		server.sendHeadBlockData(f.lastHead, f.lastHeadBody, f.lastHeadReceipts)
	}
	go server.historicReadLoop()
	go server.historicSendLoop()
}

// cacheRawReceipts caches a set of raw receipts during block processing in order
// to avoid having to read it back from the database during broadcast.
func (f *indexServers) cacheRawReceipts(blockHash common.Hash, blockReceipts types.Receipts) {
	f.rawReceiptsCache.Add(blockHash, blockReceipts)
}

// broadcast sends a new head block to all registered Indexer instances.
func (f *indexServers) broadcast(block *types.Block) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Note that individual Indexer servers might ignore block bodies and
	// receipts. We still always fetch receipts for simplicity because in the
	// typical case it is cached during block processing and costs nothing.
	blockHash := block.Hash()
	blockReceipts, _ := f.rawReceiptsCache.Get(blockHash)
	if blockReceipts == nil {
		blockReceipts = f.chain.GetRawReceipts(blockHash, block.NumberU64())
		if blockReceipts == nil {
			log.Error("Receipts belonging to new head are missing", "number", block.NumberU64(), "hash", block.Hash())
			return
		}
		f.rawReceiptsCache.Add(blockHash, blockReceipts)
	}
	f.lastHead, f.lastHeadBody, f.lastHeadReceipts = block.Header(), block.Body(), blockReceipts
	for _, server := range f.servers {
		server.sendHeadBlockData(block.Header(), block.Body(), blockReceipts)
	}
}

// revert notifies all registered Indexer instances about the chain being rolled
// back to the given head or last common ancestor.
func (f *indexServers) revert(header *types.Header) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, server := range f.servers {
		server.revert(header)
	}
}

// setFinalBlock notifies all Indexer instances about the latest finalized block.
func (f *indexServers) setFinalBlock(blockNumber uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.finalBlock == blockNumber {
		return
	}
	f.finalBlock = blockNumber
	for _, server := range f.servers {
		server.setFinalBlock(blockNumber)
	}
}

// setHistoryCutoff notifies all Indexer instances about the history cutoff point.
// The indexers cannot expect any data being delivered if needBlocks.First() is
// before this point.
func (f *indexServers) setHistoryCutoff(blockNumber uint64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.historyCutoff == blockNumber {
		return
	}
	f.historyCutoff = blockNumber
	for _, server := range f.servers {
		server.setHistoryCutoff(blockNumber)
	}
}

// setBlockProcessing suspends serving historical blocks requested by the indexers
// while a chain segment is being processed and added to the chain.
func (f *indexServers) setBlockProcessing(processing bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, server := range f.servers {
		server.setBlockProcessing(processing)
	}
}

// indexServer sends updates to a single Indexer instance. It sends all new heads
// and reorg events, and also sends historical block data upon request.
// It guarantees that Indexer functions are never called concurrently and also
// they always present a consistent view of the chain to the indexer.
type indexServer struct {
	lock    sync.Mutex
	parent  *indexServers
	indexer Indexer // always call under mutex lock; never call after stopped
	stopped bool

	lastHead                          *types.Header
	sendStatus                        indexerStatus
	statusCh                          chan indexerStatus
	blockDataCh                       chan blockData
	suspendCh                         chan bool
	testSuspendHookCh                 chan struct{} // initialized by test, capacity = 1
	sendTimer                         *time.Timer
	historyCutoff, missingBlockCutoff uint64
	needBodies, needReceipts          bool

	readStatus  indexerStatus
	readPointer uint64 // next block to be queued

	name                    string
	processed               uint64
	logged                  bool
	startedAt, lastLoggedAt time.Time
	lastHistoryErrorLog     time.Time
}

// indexerStatus is maintained by the historicSendLoop goroutine and all changes
// are sent to the historicReadLoop goroutine through statusCh.
type indexerStatus struct {
	ready                        bool                 // last feedback received from the indexer
	needBlocks                   common.Range[uint64] // last feedback received from the indexer
	suspended                    bool                 // suspend historic block delivery during block processing
	resetQueueCount              uint64               // total number of queue resets
	revertCount, lastRevertBlock uint64               // detect entries potentially expired by a revert/reorg
}

// isNextExpected returns true if the received blockData (potentially based on a
// previously sent indexerStatus) is still guaranteed to be valid and the one
// expected by the indexer according to the latest indexerStatus.
func (s *indexerStatus) isNextExpected(b blockData) bool {
	if s.needBlocks.IsEmpty() || s.needBlocks.First() != b.blockNumber {
		return false // not the expected next block number or no historical blocks expected at all
	}
	// block number is the expected one; check if a reorg might have invalidated it
	switch s.revertCount {
	case b.revertCount:
		return true // no reorgs happened since the collection of block data
	case b.revertCount + 1:
		// one reorg happened to s.lastRevertBlock; b is valid if not newer than this
		return b.blockNumber <= s.lastRevertBlock
	default:
		// multiple reorgs happened; previous revert blocks are not remembered
		// so we don't know if b is still valid and therefore we have to discard it.
		return false
	}
}

// blockData represents the indexable data of a single block being sent from the
// reader to the sender goroutine and optionally queued in blockDataCh between.
// It also includes the latest revertCount known before reading the block data,
// which allows the sender to guarantee that all sent block data is always
// consistent with the indexer's canonical chain view while the reading of block
// data can still happen asynchronously.
type blockData struct {
	blockNumber, revertCount uint64
	valid                    bool
	header                   *types.Header
	body                     *types.Body
	receipts                 types.Receipts
}

// sendHeadBlockData immediately sends the latest head block data to the indexer
// and updates the status of the historical block data serving mechanism
// accordingly.
func (s *indexServer) sendHeadBlockData(header *types.Header, body *types.Body, receipts types.Receipts) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	if header.Hash() == s.lastHead.Hash() {
		return
	}
	s.lastHead = header
	if !s.needBodies {
		body = nil
	}
	if !s.needReceipts {
		receipts = nil
	}
	ready, needBlocks := s.indexer.AddBlockData(header, body, receipts)
	s.updateIndexerStatus(ready, needBlocks, 0)
	s.updateSendStatus()
}

// updateIndexerStatus updates the ready / needBlocks fields in the send loop
// status. The number of historic blocks added since the last update is
// specified in addedBlocks. During continuous historical block range delivery
// the starting point of the new needBlocks range is expected to advance with
// each new block added. If the new range does not match the expectation then
// a blockDataCh queue reset is requested.
func (s *indexServer) updateIndexerStatus(ready bool, needBlocks common.Range[uint64], addedBlocks uint64) {
	if needBlocks.First() != s.sendStatus.needBlocks.First()+addedBlocks {
		s.sendStatus.resetQueueCount++ // request queue reset
	}
	s.sendStatus.ready, s.sendStatus.needBlocks = ready, needBlocks
}

// revert immediately reverts the indexer to the given block and updates the
// status of the historical block data serving mechanism accordingly.
func (s *indexServer) revert(header *types.Header) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped || s.lastHead == nil {
		return
	}
	if header.Hash() == s.lastHead.Hash() {
		return
	}
	blockNumber := header.Number.Uint64()
	if blockNumber >= s.lastHead.Number.Uint64() {
		panic("invalid indexer revert")
	}
	s.lastHead = header
	s.sendStatus.revertCount++
	s.sendStatus.lastRevertBlock = blockNumber
	s.updateSendStatus()
	s.indexer.Revert(blockNumber)
}

// suspendOrStop blocks the send loop until it is unsuspended or the parent
// chain is stopped. It also notifies the indexer by calling Suspend and
// suspends the read loop through updateStatus.
func (s *indexServer) suspendOrStop(suspended bool) bool {
	if !suspended {
		panic("unexpected 'false' signal on suspendCh")
	}
	s.lock.Lock()
	s.sendStatus.suspended = true
	s.updateSendStatus()
	s.indexer.Suspended()
	s.lock.Unlock()
	select {
	case <-s.parent.closeCh:
		return true
	case suspended = <-s.suspendCh:
	}
	if suspended {
		panic("unexpected 'true' signal on suspendCh")
	}
	s.lock.Lock()
	s.sendStatus.suspended = false
	s.updateSendStatus()
	s.lock.Unlock()
	return false
}

// historicSendLoop is the main event loop that interacts with the indexer in
// case when historical block data is requested. It sends status updates to
// the reader goroutine through statusCh and feeds the fetched data coming from
// blockDataCh into the indexer.
func (s *indexServer) historicSendLoop() {
	defer func() {
		s.lock.Lock()
		s.indexer.Stop()
		s.stopped = true
		s.lock.Unlock()
		s.parent.closeWg.Done()
	}()

	for {
		select {
		// do a separate non-blocking select to ensure that a suspend attempt
		// during the previous historical AddBlockData will be catched in the
		// next round.
		case suspend := <-s.suspendCh:
			if s.suspendOrStop(suspend) {
				return
			}
		default:
		}
		select {
		case <-s.parent.closeCh:
			return
		case suspend := <-s.suspendCh:
			if s.suspendOrStop(suspend) {
				return
			}
		case nextBlockData := <-s.blockDataCh:
			s.addHistoricBlockData(nextBlockData)
		case <-s.sendTimer.C:
			s.handleHistoricLoopTimer()
		}
	}
}

// handleHistoricLoopTimer queries the indexer status again if the last known
// status was "not ready". By calling updateSendStatus it also restarts the
// timer if the indexer is still not ready.
func (s *indexServer) handleHistoricLoopTimer() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.sendStatus.ready {
		ready, needBlocks := s.indexer.Status()
		s.updateIndexerStatus(ready, needBlocks, 0)
		s.updateSendStatus()
	}
}

// addHistoricBlockData checks if the next blockData fetched by the asynchronous
// historicReadLoop is still the one to be delivered next to the indexer and
// delivers it if possible. If the requested block range has changed since or
// a reorg might have made the fetched data invalid then it triggers a queue
// reset by increasing resetQueueCount. This ensures that the read loop discards
// queued blockData and starts sending newly fetched data according to the new
// needBlocks range.
func (s *indexServer) addHistoricBlockData(nextBlockData blockData) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// check if received block data is indeed from the next expected
	// block and is still guaranteed to be canonical; ignore and request
	// a queue reset otherwise.
	if s.sendStatus.isNextExpected(nextBlockData) {
		// check if the has actually been found in the database
		if nextBlockData.valid {
			ready, needBlocks := s.indexer.AddBlockData(nextBlockData.header, nextBlockData.body, nextBlockData.receipts)
			s.updateIndexerStatus(ready, needBlocks, 1)
			if s.sendStatus.needBlocks.IsEmpty() {
				s.logDelivered(nextBlockData.blockNumber)
				s.logFinished()
			} else if s.sendStatus.needBlocks.First() == nextBlockData.blockNumber+1 {
				s.logDelivered(nextBlockData.blockNumber)
			}
		} else {
			// report error and update missingBlockCutoff in order to
			// avoid spinning forever on the same error.
			if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
				s.lastHistoryErrorLog = time.Now()
				if nextBlockData.header == nil {
					log.Error("Historical header is missing", "number", nextBlockData.blockNumber)
				} else if s.needBodies && nextBlockData.body == nil {
					log.Error("Historical block body is missing", "number", nextBlockData.blockNumber, "hash", nextBlockData.header.Hash())
				} else if s.needReceipts && nextBlockData.receipts == nil {
					log.Error("Historical receipts are missing", "number", nextBlockData.blockNumber, "hash", nextBlockData.header.Hash())
				}
			}
			s.missingBlockCutoff = max(s.missingBlockCutoff, nextBlockData.blockNumber+1)
			s.indexer.SetHistoryCutoff(max(s.historyCutoff, s.missingBlockCutoff))
			ready, needBlocks := s.indexer.Status()
			s.updateIndexerStatus(ready, needBlocks, 0)
		}
	} else {
		// trigger resetting the queue and sending blockData from needBlocks.First()
		s.sendStatus.resetQueueCount++
	}
	s.updateSendStatus()
}

// updateStatus updates the asynchronous reader goroutine's status based on the
// latest indexer status. If necessary then it trims the needBlocks range based
// on the locally available block range. If there is already an unread status
// update waiting on statusCh then it is replaced by the new one.
func (s *indexServer) updateSendStatus() {
	if s.sendStatus.ready || s.sendStatus.suspended {
		s.sendTimer.Stop()
	} else {
		s.sendTimer.Reset(busyDelay)
	}
	var headNumber uint64
	if s.lastHead != nil {
		headNumber = s.lastHead.Number.Uint64()
	}
	if headNumber+1 < s.sendStatus.needBlocks.AfterLast() {
		s.sendStatus.needBlocks.SetLast(headNumber)
	}
	if s.sendStatus.needBlocks.IsEmpty() || max(s.historyCutoff, s.missingBlockCutoff) > s.sendStatus.needBlocks.First() {
		s.sendStatus.needBlocks = common.Range[uint64]{}
	}
	select {
	case <-s.statusCh:
	default:
	}
	s.statusCh <- s.sendStatus
}

// setBlockProcessing suspends serving historical blocks requested by the indexer
// while a chain segment is being processed and added to the chain.
func (s *indexServer) setBlockProcessing(suspended bool) {
	select {
	case old := <-s.suspendCh:
		if old == suspended {
			panic("unexpected value pulled back from suspendCh")
		}
	default:
		// only send new suspended flag if previous (opposite) value has been
		// read by the send loop already
		s.suspendCh <- suspended
	}
	if suspended && s.testSuspendHookCh != nil {
		select {
		case s.testSuspendHookCh <- struct{}{}:
		default:
		}
	}
}

// clearBlockQueue removes all entries from blockDataCh.
func (s *indexServer) clearBlockQueue() {
	for {
		select {
		case <-s.blockDataCh:
		default:
			return
		}
	}
}

// updateReadStatus updates readStatus and checks whether a queue reset has been
// requested by the send loop. In that case it empties the queue and resets the
// readPointer to the first block of the needBlocks range.
// Note that the blocks betweeen needBlocks.First() and readPointer-1 are assumed
// to already be queued in blockDataCh. If needBlocks.First() does not advance
// with each delivered block or an expired blockData is received by the send
// loop then a queue reset is requested.
func (s *indexServer) updateReadStatus(newStatus indexerStatus) {
	if newStatus.resetQueueCount != s.readStatus.resetQueueCount {
		s.clearBlockQueue()
		s.readPointer = newStatus.needBlocks.First()
	}
	s.readStatus = newStatus
}

// canQueueNextBlock returns true if there are more blocks to read in the
// needBlocks range and we have not yet reached the capacity of blockDataCh yet.
// Note that the latter check assumes that blocks between needBlocks.First() and
// readPointer-1 are queued.
func (s *indexServer) canQueueNextBlock() bool {
	return s.readStatus.needBlocks.Includes(s.readPointer) &&
		s.readPointer < s.readStatus.needBlocks.First()+maxHistoricPrefetch
}

// historicReadLoop reads requested historical block data asynchronously.
// It receives indexer status updates on statusCh and sends block data to
// blockDataCh. If the latest status indicates that there the server is not
// suspended then it is guaranteed that eventually a corresponding block data
// response will be sent unless a new status update is received before this
// happens.
// Note that blockDataCh can queue multiple block data pre-fetched by
// historicReadLoop. If the requested range is changed while there is still
// queued data in the channel that corresponds to the previous requested range
// then the receiver sends a new status update with increased resetQueueCount.
// In this case historicReadLoop removes all remaining entries from the queue
// and starts sending block data from the beginning of the new range.
func (s *indexServer) historicReadLoop() {
	defer s.parent.closeWg.Done()

	s.readStatus.resetQueueCount = math.MaxUint64
	for {
		if !s.readStatus.suspended && s.canQueueNextBlock() {
			// Send next item to the queue.
			bd := blockData{blockNumber: s.readPointer, revertCount: s.readStatus.revertCount, valid: true}
			if bd.header = s.parent.chain.GetHeaderByNumber(bd.blockNumber); bd.header != nil {
				blockHash := bd.header.Hash()
				if s.needBodies {
					bd.body = s.parent.chain.GetBody(blockHash)
					if bd.body == nil {
						bd.valid = false
					}
				}
				if s.needReceipts {
					bd.receipts, _ = s.parent.rawReceiptsCache.Get(blockHash)
					if bd.receipts == nil {
						// Note: we do not cache historical receipts because typically
						// each indexer requests them at different times.
						bd.receipts = s.parent.chain.GetRawReceipts(blockHash, bd.blockNumber)
						if bd.receipts == nil {
							bd.valid = false
						}
					}
				}
			}
			// Note that a response with missing block data is still sent in case of
			// a read error, signaling to the sender logic that something is missing.
			// This might be either due to a database error or a reorg.
			select {
			case s.blockDataCh <- bd:
				s.readPointer++
			default:
				// Note: updateIndexerStatus in the send loop and canQueueNextBlock
				// in the read loop should ensure that no send is attempted at
				// blockDataCh when it is filled to full capacity. If it happens
				// anyway then we print an error and set the suspended flag to
				// true until the next status update.
				if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
					s.lastHistoryErrorLog = time.Now()
					log.Error("Historical block queue is full")
				}
				s.readStatus.suspended = true
			}
			// Keep checking status updates without blocking as long as there is
			// something to do.
			select {
			case <-s.parent.closeCh:
				return
			case status := <-s.statusCh:
				s.updateReadStatus(status)
			default:
			}
		} else {
			// There was nothing to do; wait for a next status update.
			select {
			case <-s.parent.closeCh:
				return
			case status := <-s.statusCh:
				s.updateReadStatus(status)
			}
		}
	}
}

// logDelivered periodically prints log messages that report the current state
// of the indexing process. If should be called after processing each new block.
func (s *indexServer) logDelivered(position uint64) {
	if s.processed == 0 {
		s.startedAt = time.Now()
	}
	s.processed++
	if s.logged {
		if time.Since(s.lastLoggedAt) < logFrequency {
			return
		}
	} else {
		if time.Since(s.startedAt) < headLogDelay {
			return
		}
		s.logged = true
	}
	s.lastLoggedAt = time.Now()
	log.Info("Generating "+s.name, "block", position, "processed", s.processed, "elapsed", time.Since(s.startedAt))
}

// logFinished prints a log message that report the end of the indexing process.
// Note that any log message is only printed if the process took longer than
// headLogDelay.
func (s *indexServer) logFinished() {
	if s.logged {
		log.Info("Finished "+s.name, "processed", s.processed, "elapsed", time.Since(s.startedAt))
		s.logged = false
	}
	s.processed = 0
}

// setFinalBlock notifies the Indexer instance about the latest finalized block.
func (s *indexServer) setFinalBlock(blockNumber uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	s.indexer.SetFinalized(blockNumber)
}

// setHistoryCutoff notifies the Indexer instance about the history cutoff point.
// The indexer cannot expect any data being delivered if needBlocks.First() is
// before this point.
// Note that if some historical block data could not be loaded from the database
// then the historical cutoff point reported to the indexer might be modified by
// missingBlockCutoff. This workaround ensures that the indexing process does not
// get stuck permanently in case of missing data.
func (s *indexServer) setHistoryCutoff(blockNumber uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	s.historyCutoff = blockNumber
	s.indexer.SetHistoryCutoff(max(s.historyCutoff, s.missingBlockCutoff))
	ready, needBlocks := s.indexer.Status()
	s.updateIndexerStatus(ready, needBlocks, 0)
	s.updateSendStatus()
}
