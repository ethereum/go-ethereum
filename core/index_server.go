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
	AddBlockData(header *types.Header, receipts types.Receipts) (ready bool, needBlocks common.Range[uint64])
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
func (f *indexServers) register(indexer Indexer, name string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	server := &indexServer{
		parent:      f,
		indexer:     indexer,
		sendTimer:   time.NewTimer(0),
		lastHead:    f.lastHead,
		name:        name,
		statusCh:    make(chan indexerStatus, 1),
		blockDataCh: make(chan blockData, maxHistoricPrefetch),
		suspendCh:   make(chan bool, 1),
	}
	f.servers = append(f.servers, server)
	f.closeWg.Add(2)
	indexer.SetHistoryCutoff(f.historyCutoff)
	indexer.SetFinalized(f.finalBlock)
	if f.lastHead != nil {
		server.status.ready, server.status.needBlocks = indexer.AddBlockData(f.lastHead, f.lastHeadReceipts)
		server.updateStatus()
	}
	go server.historicReadLoop()
	go server.historicSendLoop()
}

func (f *indexServers) cacheRawReceipts(blockHash common.Hash, blockReceipts types.Receipts) {
	f.rawReceiptsCache.Add(blockHash, blockReceipts)
}

// broadcast sends a new head block to all registered Indexer instances.
func (f *indexServers) broadcast(header *types.Header) {
	f.lock.Lock()
	defer f.lock.Unlock()

	blockHash := header.Hash()
	blockReceipts, _ := f.rawReceiptsCache.Get(blockHash)
	if blockReceipts == nil {
		blockReceipts = f.chain.GetRawReceipts(blockHash, header.Number.Uint64())
		if blockReceipts == nil {
			log.Error("Receipts belonging to new head are missing", "number", header.Number, "hash", header.Hash())
			return
		}
		f.rawReceiptsCache.Add(blockHash, blockReceipts)
	}
	f.lastHead, f.lastHeadReceipts = header, blockReceipts
	for _, server := range f.servers {
		server.sendHeadBlockData(header, blockReceipts)
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
	status                            indexerStatus
	statusCh                          chan indexerStatus
	blockDataCh                       chan blockData
	suspendCh                         chan bool
	testSuspendHookCh                 chan struct{} // initialized by test, capacity = 1
	lastRevertBlock                   uint64
	sendTimer                         *time.Timer
	historyCutoff, missingBlockCutoff uint64

	name                    string
	processed               uint64
	logged                  bool
	startedAt, lastLoggedAt time.Time
	lastHistoryErrorLog     time.Time
}

// indexerStatus represents the state of the indexer and also has fields that
// serve the coordination between historic reader and sender goroutines.
type indexerStatus struct {
	ready, suspended, resetQueue bool
	revertCount                  uint64
	needBlocks                   common.Range[uint64]
}

// blockData represents the indexable data of a single block being sent from the
// reader to the sender goroutine and optionally queued in blockDataCh between.
// It also includes the latest revertCount known before reading the block data,
// which allows the sender to guarantee that all sent block data is always
// consistent with the indexer's canonical chain view while the reading of block
// data can still happen asynchronously.
type blockData struct {
	blockNumber, revertCount uint64
	header                   *types.Header
	receipts                 types.Receipts
}

// sendHeadBlockData immediately sends the latest head block data to the indexer
// and updates the status of the historical block data serving mechanism
// accordingly.
func (s *indexServer) sendHeadBlockData(header *types.Header, receipts types.Receipts) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	if header.Hash() == s.lastHead.Hash() {
		return
	}
	s.lastHead = header
	s.status.ready, s.status.needBlocks = s.indexer.AddBlockData(header, receipts)
	s.updateStatus()
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
	s.status.revertCount++
	s.lastRevertBlock = blockNumber
	s.updateStatus()
	s.indexer.Revert(blockNumber)
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

	// suspendOrStop blocks the send loop until it is unsuspended or the parent
	// chain is stopped. It also notifies the indexer by calling Suspend and
	// suspends the read loop through updateStatus.
	suspendOrStop := func(suspended bool) bool {
		if !suspended {
			panic("unexpected 'false' signal on suspendCh")
		}
		s.lock.Lock()
		s.status.suspended = true
		s.updateStatus()
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
		s.status.suspended = false
		s.updateStatus()
		s.lock.Unlock()
		return false
	}

	for {
		select {
		// do a separate non-blocking select to ensure that a suspend attempt
		// during the previous historical AddBlockData will be catched in the
		// next round.
		case ch := <-s.suspendCh:
			if suspendOrStop(ch) {
				return
			}
		default:
		}
		select {
		case <-s.parent.closeCh:
			return
		case ch := <-s.suspendCh:
			if suspendOrStop(ch) {
				return
			}
		case nextBlockData := <-s.blockDataCh:
			s.lock.Lock()
			s.status.resetQueue = true
			// check if received block data is indeed from the next expected
			// block and is still guaranteed to be canonical; ignore and request
			// again otherwise.
			if !s.status.needBlocks.IsEmpty() && s.status.needBlocks.First() == nextBlockData.blockNumber &&
				(nextBlockData.revertCount == s.status.revertCount || (nextBlockData.revertCount+1 == s.status.revertCount && nextBlockData.blockNumber <= s.lastRevertBlock)) {
				s.status.ready, s.status.needBlocks = s.indexer.AddBlockData(nextBlockData.header, nextBlockData.receipts)
				// check if the has actually been found in the database
				if nextBlockData.header != nil && nextBlockData.receipts != nil {
					s.status.resetQueue = false
					if s.status.needBlocks.IsEmpty() {
						s.logDelivered(nextBlockData.blockNumber)
						s.logFinished()
					} else if s.status.needBlocks.First() == nextBlockData.blockNumber+1 {
						s.logDelivered(nextBlockData.blockNumber)
					}
				} else {
					// report error and update missingBlockCutoff in order to
					// avoid spinning forever on the same error.
					if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
						s.lastHistoryErrorLog = time.Now()
						if nextBlockData.header == nil {
							log.Error("Historical header is missing", "number", nextBlockData.blockNumber)
						} else {
							log.Error("Historical receipts are missing", "number", nextBlockData.blockNumber, "hash", nextBlockData.header.Hash())
						}
					}
					s.missingBlockCutoff = max(s.missingBlockCutoff, nextBlockData.blockNumber+1)
					s.indexer.SetHistoryCutoff(max(s.historyCutoff, s.missingBlockCutoff))
					s.status.ready, s.status.needBlocks = s.indexer.Status()
				}
			}
			s.updateStatus()
			s.lock.Unlock()
		case <-s.sendTimer.C:
			s.lock.Lock()
			if !s.status.ready {
				s.status.ready, s.status.needBlocks = s.indexer.Status()
				s.updateStatus()
			}
			s.lock.Unlock()
		}
	}
}

// updateStatus updates the asynchronous reader goroutine's status based on the
// latest indexer status. If necessary then it trims the needBlocks range based
// on the locally available block range. If there is already an unread status
// update waiting on statusCh then it is replaced by the new one.
func (s *indexServer) updateStatus() {
	if s.status.ready || s.status.suspended {
		s.sendTimer.Stop()
	} else {
		s.sendTimer.Reset(busyDelay)
	}
	var headNumber uint64
	if s.lastHead != nil {
		headNumber = s.lastHead.Number.Uint64()
	}
	if headNumber+1 < s.status.needBlocks.AfterLast() {
		s.status.needBlocks.SetLast(headNumber)
	}
	if s.status.needBlocks.IsEmpty() || max(s.historyCutoff, s.missingBlockCutoff) > s.status.needBlocks.First() {
		s.status.needBlocks = common.Range[uint64]{}
	}
	select {
	case <-s.statusCh:
	default:
	}
	s.statusCh <- s.status
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

// historicReadLoop reads requested historical block data asynchronously.
// It receives indexer status updates on statusCh and sends block data to
// blockDataCh. If the latest status indicates that there the server is not
// suspended then it is guaranteed that eventually a corresponding block data
// response will be sent unless a new status update is received before this
// happens.
// Note that blockDataCh can queue multiple block data pre-fetched by
// historicReadLoop. If the requested range is changed while there is still
// queued data in the channel that corresponds to the previous requested range
// then the receiver sends a new status update with the resetQueue flag set to
// true. In this case historicReadLoop removes all remaining entries from the
// queue and starts sending block data from the beginning of the new range.
func (s *indexServer) historicReadLoop() {
	defer s.parent.closeWg.Done()

	var (
		status    indexerStatus
		sendRange common.Range[uint64]
	)

	statusUpdated := func() {
		if status.resetQueue {
			// If the receiver found an item in the queue that is no longer
			// relevant then we remove all remaining items first.
		loop:
			for {
				select {
				case <-s.blockDataCh:
				default:
					break loop
				}
			}
		}
		if !status.resetQueue && !sendRange.IsEmpty() && status.needBlocks.Includes(sendRange.First()) {
			// Here we assume that the block data between needBlocks.First() and
			// sendRange.First()-1 is already in the queue.
			r := status.needBlocks
			r.SetFirst(sendRange.First())
			sendRange = r
		} else {
			sendRange = status.needBlocks
		}
		if sendRange.Count() > maxHistoricPrefetch {
			// Note: in a normal use case where needBlocks.First() is advanced
			// after reading the previous item from blockDataCh, this check will
			// prevent reading more data than what fits into the channel capacity.
			sendRange.SetAfterLast(sendRange.First() + maxHistoricPrefetch)
		}
	}

	for {
		if !sendRange.IsEmpty() && !status.suspended {
			// Send next item to the queue.
			bd := blockData{blockNumber: sendRange.First(), revertCount: status.revertCount}
			if bd.header = s.parent.chain.GetHeaderByNumber(bd.blockNumber); bd.header != nil {
				blockHash := bd.header.Hash()
				bd.receipts, _ = s.parent.rawReceiptsCache.Get(blockHash)
				if bd.receipts == nil {
					bd.receipts = s.parent.chain.GetRawReceipts(blockHash, bd.blockNumber)
					// Note: we do not cache historical receipts because typically
					// each indexer requests them at different times.
				}
			}
			// Note that a response with missing block data is still sent in case of
			// a read error, signaling to the sender logic that something is missing.
			// This might be either due to a database error or a reorg.
			select {
			case s.blockDataCh <- bd:
				sendRange.SetFirst(bd.blockNumber + 1)
			default:
				// Note: in extreme corner cases where sendRange.Count() check
				// does not prevent trying to overfill the channel, we simply
				// reset sendRange to empty preventing more wasted reads.
				// Sending the queued data will generate more status updates
				// which will reinitialize sendRange once the queue is again
				// below full capacity.
				sendRange = common.Range[uint64]{}
			}
			// Keep checking status updates without blocking as long as there is
			// something to do.
			select {
			case <-s.parent.closeCh:
				return
			case status = <-s.statusCh:
				statusUpdated()
			default:
			}
		} else {
			// There was nothing to do; wait for a next status update.
			select {
			case <-s.parent.closeCh:
				return
			case status = <-s.statusCh:
				statusUpdated()
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
	s.status.ready, s.status.needBlocks = s.indexer.Status()
	s.updateStatus()
}
