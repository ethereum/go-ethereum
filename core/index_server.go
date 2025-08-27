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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	maxBatchLength = 64
	busyDelay      = time.Second
)

type Indexer interface {
	// AddBlockData delivers a continuous range of headers and receipts that are
	// either direct descendants of the latest delivered head or part of the
	// requested historic range.
	// The current ready/busy status and the requested historic range are returned.
	// Note that the indexer should never block even if it is busy processing.
	// It is allowed to re-request the delivered blocks later if the indexer could
	// not process them when first delivered.
	AddBlockData(headers []*types.Header, receipts []types.Receipts) (ready bool, needBlocks common.Range[uint64])
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
	// Suspended signals to the indexer that historical block delivery has been
	// temporarily suspended due to block processing priority. If the indexer
	// is running non-essential asynchronous tasks then those should also be
	// suspended.
	// The next AddBlockData call signals the end of the suspended state.
	Suspended()
	// Stop initiates indexer shutdown. No subsequent calls are made through this
	// interface.
	Stop()
}

type indexServers struct {
	lock    sync.Mutex
	servers []*indexServer
	chain   *BlockChain

	headers  []*types.Header  // broadcast head header batch
	receipts []types.Receipts // broadcast head receipts batch

	lastHead                  *types.Header
	lastHeadReceipts          types.Receipts
	finalBlock, historyCutoff uint64

	closeCh chan struct{}
	closeWg sync.WaitGroup
}

func (f *indexServers) init(chain *BlockChain) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.chain = chain
	f.closeCh = make(chan struct{})
}

func (f *indexServers) stop() {
	f.lock.Lock()
	defer f.lock.Unlock()

	close(f.closeCh)
	f.closeWg.Wait()
	f.servers = nil
}

func (f *indexServers) register(indexer Indexer, name string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	server := &indexServer{
		parent:    f,
		indexer:   indexer,
		sendTimer: time.NewTimer(0),
		lastHead:  f.lastHead,
		name:      name,
	}
	f.servers = append(f.servers, server)
	f.closeWg.Add(1)
	indexer.SetHistoryCutoff(f.historyCutoff)
	indexer.SetFinalized(f.finalBlock)
	if f.lastHead != nil {
		server.ready, server.needBlocks = indexer.AddBlockData([]*types.Header{f.lastHead}, []types.Receipts{f.lastHeadReceipts})
	}
	go server.eventLoop()
}

func (f *indexServers) broadcast(header *types.Header, head bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	blockReceipts := f.chain.GetReceipts(header.Hash(), header.Number.Uint64())
	if blockReceipts == nil {
		log.Error("Receipts belonging to new head are missing", "number", header.Number, "hash", header.Hash())
		return
	}
	f.lastHead, f.lastHeadReceipts = header, blockReceipts
	f.headers = append(f.headers, header)
	f.receipts = append(f.receipts, blockReceipts)
	if head || len(f.headers) >= maxBatchLength {
		for _, server := range f.servers {
			server.sendHeadBlockData(f.headers, f.receipts)
		}
		f.headers = f.headers[:0]
		f.receipts = f.receipts[:0]
	}
}

func (f *indexServers) revert(header *types.Header) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, server := range f.servers {
		server.revert(header)
	}
}

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

func (f *indexServers) setSuspended(suspended bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, server := range f.servers {
		server.setSuspended(suspended)
	}
}

type indexServer struct {
	lock    sync.Mutex
	parent  *indexServers
	indexer Indexer // always call under mutex lock; never call after stopped
	stopped bool

	lastHead                          *types.Header
	ready                             bool
	suspendCh                         chan struct{}
	needBlocks, historicBatchRange    common.Range[uint64]
	sendTimer                         *time.Timer
	historyCutoff, missingBlockCutoff uint64

	name                    string
	processed               uint64
	logged                  bool
	startedAt, lastLoggedAt time.Time
	lastHistoryErrorLog     time.Time
}

func (s *indexServer) eventLoop() {
loop:
	for {
		select {
		case <-s.parent.closeCh:
			s.lock.Lock()
			s.indexer.Stop()
			s.stopped = true
			s.lock.Unlock()
			s.parent.closeWg.Done()
			return
		case <-s.sendTimer.C:
			var (
				headers  []*types.Header
				receipts []types.Receipts
			)
			s.lock.Lock()
			s.historicBatchRange = s.nextHistoricBatchRange()
			//fmt.Println("needBlocks", s.needBlocks, "historicBatchRange", s.historicBatchRange, "ready", s.ready)
			if !s.historicBatchRange.IsEmpty() {
				historicBatchRange := s.historicBatchRange
				s.lock.Unlock() // do not hold lock that can block BlockChain while reading a historic batch
				headers, receipts = s.historicBlockData(historicBatchRange)
				s.lock.Lock()
				// wait if historic block delivery has been suspended while collecting data
				if s.suspendCh != nil {
					ch := s.suspendCh
					s.lock.Unlock() // do not hold lock that can block BlockChain while reading a historic batch
					select {
					case <-ch:
					case <-s.parent.closeCh:
						continue loop
					}
					s.lock.Lock()
				}
				// ensure that the delivered data still matches the latest required range
				s.historicBatchRange = s.historicBatchRange.Intersection(s.nextHistoricBatchRange())
				// trim results if historicBatchRange has been shortened while collecting data
				for len(headers) > 0 && !s.historicBatchRange.Includes(headers[0].Number.Uint64()) {
					headers, receipts = headers[1:], receipts[1:]
				}
				if len(headers) > 0 && s.historicBatchRange.First() != headers[0].Number.Uint64() {
					headers, receipts = nil, nil
				}
				for len(headers) > 0 && !s.historicBatchRange.Includes(headers[len(headers)-1].Number.Uint64()) {
					headers, receipts = headers[:len(headers)-1], receipts[:len(receipts)-1]
				}
			}
			if len(headers) > 0 {
				s.ready, s.needBlocks = s.indexer.AddBlockData(headers, receipts)
				if s.needBlocks.IsEmpty() {
					s.logDelivered(headers[len(headers)-1].Number.Uint64(), uint64(len(headers)))
					s.logFinished()
				} else if s.needBlocks.First() > headers[0].Number.Uint64() && s.needBlocks.First() <= headers[len(headers)-1].Number.Uint64()+1 {
					s.logDelivered(s.needBlocks.First()-1, s.needBlocks.First()-headers[0].Number.Uint64())
				}
			} else {
				s.ready, s.needBlocks = s.indexer.Status()
			}
			s.setTimer()
			s.lock.Unlock()
		}
	}
}

func (s *indexServer) logDelivered(position, amount uint64) {
	if s.processed == 0 {
		s.startedAt = time.Now()
	}
	s.processed += amount
	if s.logged {
		if time.Since(s.lastLoggedAt) < time.Second*10 {
			return
		}
	} else {
		if time.Since(s.startedAt) < time.Second {
			return
		}
		s.logged = true
	}
	s.lastLoggedAt = time.Now()
	log.Info("Generating "+s.name, "block", position, "processed", s.processed, "elapsed", time.Since(s.startedAt))
}

func (s *indexServer) logFinished() {
	if s.logged {
		log.Info("Finished "+s.name, "processed", s.processed, "elapsed", time.Since(s.startedAt))
	}
	s.processed = 0
}

func (s *indexServer) nextHistoricBatchRange() common.Range[uint64] {
	if !s.ready || s.lastHead == nil {
		return common.Range[uint64]{}
	}
	first := max(s.needBlocks.First(), s.historyCutoff, s.missingBlockCutoff)
	afterLast := min(first+maxBatchLength, s.needBlocks.AfterLast(), s.lastHead.Number.Uint64()+1)
	if first < afterLast {
		return common.NewRange[uint64](first, afterLast-first)
	} else {
		return common.Range[uint64]{}
	}
}

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
	if blockNumber+1 < s.historicBatchRange.AfterLast() {
		s.historicBatchRange.SetLast(blockNumber)
	}
	s.indexer.Revert(blockNumber)
}

func (s *indexServer) setFinalBlock(blockNumber uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	s.indexer.SetFinalized(blockNumber)
}

func (s *indexServer) setHistoryCutoff(blockNumber uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	s.historyCutoff = blockNumber
	cutoff := max(s.historyCutoff, s.missingBlockCutoff)
	if cutoff > s.historicBatchRange.First() {
		s.historicBatchRange.SetFirst(cutoff)
	}
	s.indexer.SetHistoryCutoff(cutoff)
}

func (s *indexServer) setSuspended(suspended bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	if s.lastHead != nil && s.needBlocks.AfterLast() > s.lastHead.Number.Uint64() {
		suspended = false
	}
	if (s.suspendCh != nil) == suspended {
		return
	}
	//fmt.Println("setSuspended", suspended)
	if suspended {
		s.suspendCh = make(chan struct{})
		s.indexer.Suspended()
	} else {
		close(s.suspendCh)
		s.suspendCh = nil
	}
	s.setTimer()
}

func (s *indexServer) sendHeadBlockData(headers []*types.Header, receipts []types.Receipts) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}
	if len(headers) > 0 && headers[0].Hash() == s.lastHead.Hash() {
		headers = headers[1:]
		receipts = receipts[1:]
	}
	if len(headers) == 0 {
		return
	}
	/*lastHash := s.lastHead.Hash()
	for _, header := range headers {
		if header.ParentHash != lastHash {
			panic("non-continuous head header chain sent to indexer")
		}
		lastHash = header.Hash()
	}*/
	s.ready, s.needBlocks = s.indexer.AddBlockData(headers, receipts)
	if s.suspendCh != nil {
		//fmt.Println("setSuspended false (head)")
		close(s.suspendCh)
		s.suspendCh = nil
	}
	s.lastHead = headers[len(headers)-1]
	s.setTimer()
}

func (s *indexServer) setTimer() {
	if s.nextHistoricBatchRange().IsEmpty() || s.suspendCh != nil {
		//fmt.Println("setTimer stop")
		s.sendTimer.Stop()
	} else {
		if s.ready {
			//fmt.Println("setTimer 0")
			s.sendTimer.Reset(0)
		} else {
			//fmt.Println("setTimer busy")
			s.sendTimer.Reset(busyDelay)
		}
	}
}

func (s *indexServer) historicBlockData(batchRange common.Range[uint64]) (headers []*types.Header, receipts []types.Receipts) {
	headers = make([]*types.Header, 0, batchRange.Count())
	receipts = make([]types.Receipts, 0, batchRange.Count())

	for !batchRange.IsEmpty() {
		number := batchRange.First()
		header := s.parent.chain.GetHeaderByNumber(number)
		var blockReceipts types.Receipts
		if header != nil {
			blockReceipts = s.parent.chain.GetReceipts(header.Hash(), number)
			if blockReceipts != nil {
				headers = append(headers, header)
				receipts = append(receipts, blockReceipts)
				batchRange.SetFirst(number + 1)
				continue
			}
		}
		// something is missing, update batch range and check if a reorg/pruning has happened
		s.lock.Lock()
		batchRange = s.historicBatchRange
		s.lock.Unlock()
		if number >= batchRange.AfterLast() {
			return // end of requested section has been reorged; nothing to do here, deliver what we have
		}
		headers, receipts = nil, nil
		if number < batchRange.First() {
			continue // beginning of requested section has been pruned; resume with new range
		}
		// something is missing in the supposedly available range; print error log,
		// update missingBlockCutoff, send new cutoff limit to indexer and return
		// no results (main event loop will also update status based on new cutoff).
		if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
			s.lastHistoryErrorLog = time.Now()
			if header == nil {
				log.Error("Historical header missing", "number", number)
			} else {
				log.Error("Historical receipts are missing", "number", number, "hash", header.Hash())
			}
		}
		s.lock.Lock()
		s.missingBlockCutoff = number + 1
		if s.missingBlockCutoff > s.historyCutoff {
			s.indexer.SetHistoryCutoff(s.missingBlockCutoff)
		}
		s.lock.Unlock()
		return
	}
	return
}
