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
	Status() (ready bool, needBlocks common.Range[uint64])
	// MissingBlocks signals to the indexer that certain historic blocks are not
	// available. Eventually all requested historic blocks will either be delivered
	// or reported as missing.
	MissingBlocks(missing common.Range[uint64])
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

func (f *indexServers) register(indexer Indexer) {
	f.lock.Lock()
	defer f.lock.Unlock()

	server := &indexServer{
		parent:    f,
		indexer:   indexer,
		sendTimer: time.NewTimer(0),
		lastHead:  f.chain.CurrentBlock(),
	}
	server.historyCutoff, _ = f.chain.HistoryPruningCutoff()
	f.servers = append(f.servers, server)
	f.closeWg.Add(1)
	indexer.MissingBlocks(common.NewRange[uint64](0, server.historyCutoff))
	blockReceipts := f.chain.GetReceipts(server.lastHead.Hash(), server.lastHead.Number.Uint64())
	if blockReceipts != nil {
		indexer.AddBlockData([]*types.Header{server.lastHead}, []types.Receipts{blockReceipts})
	} else {
		log.Error("Receipts belonging to init head are missing", "number", server.lastHead.Number, "hash", server.lastHead.Hash())
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

type indexServer struct {
	lock          sync.Mutex
	parent        *indexServers
	indexer       Indexer // always call under mutex lock; never call after stopped
	historyCutoff uint64
	stopped       bool

	lastHead            *types.Header
	ready               bool
	sendTimer           *time.Timer
	needBlocks          common.Range[uint64]
	historicRefBlock    uint64
	lastHistoryErrorLog time.Time
}

func (s *indexServer) eventLoop() {
	for {
		select {
		case <-s.sendTimer.C:
			s.lock.Lock()
			if s.ready {
				first := max(s.needBlocks.First(), s.historyCutoff)
				afterLast := min(first+maxBatchLength, s.needBlocks.AfterLast(), s.lastHead.Number.Uint64()+1)
				s.lock.Unlock()
				if first < afterLast {
					headers, receipts := s.historicBlockData(first, afterLast)
					if headers != nil {
						s.sendHistoricBlockData(headers, receipts)
					}
				}
			} else {
				s.ready, s.needBlocks = s.indexer.Status()
				s.setTimer()
				s.lock.Unlock()
			}
		case <-s.parent.closeCh:
			s.lock.Lock()
			s.indexer.Stop()
			s.stopped = true
			s.lock.Unlock()
			s.parent.closeWg.Done()
			return
		}
	}
}

func (s *indexServer) revert(header *types.Header) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.stopped {
		if header.Hash() == s.lastHead.Hash() {
			return
		}
		if header.Number.Uint64() >= s.lastHead.Number.Uint64() {
			panic("invalid indexer revert")
		}
		s.indexer.Revert(header.Number.Uint64())
		s.lastHead = header
		s.historicRefBlock = min(s.historicRefBlock, header.Number.Uint64())
	}
}

func (s *indexServer) sendHeadBlockData(headers []*types.Header, receipts []types.Receipts) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.stopped {
		if len(headers) > 0 && headers[0].Hash() == s.lastHead.Hash() {
			headers = headers[1:]
			receipts = receipts[1:]
		}
		if len(headers) == 0 {
			return
		}
		lastHash := s.lastHead.Hash()
		for _, header := range headers {
			if header.ParentHash != lastHash {
				panic("non-continuous head header chain sent to indexer")
			}
			lastHash = header.Hash()
		}
		s.ready, s.needBlocks = s.indexer.AddBlockData(headers, receipts)
		s.lastHead = headers[len(headers)-1]
		s.setTimer()
	}
}

func (s *indexServer) setTimer() {
	if s.needBlocks.IsEmpty() {
		s.sendTimer.Stop()
	} else {
		if s.ready {
			s.sendTimer.Reset(0)
		} else {
			s.sendTimer.Reset(busyDelay)
		}
	}
}

func (s *indexServer) sendHistoricBlockData(headers []*types.Header, receipts []types.Receipts) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.stopped {
		for len(headers) > 0 && headers[len(headers)-1].Number.Uint64() > s.historicRefBlock {
			headers = headers[:len(headers)-1]
			receipts = receipts[:len(receipts)-1]
		}
		if len(headers) != 0 {
			s.ready, s.needBlocks = s.indexer.AddBlockData(headers, receipts)
		} else {
			s.ready, s.needBlocks = s.indexer.Status()
		}
		s.setTimer()
	}
}

func (s *indexServer) historicBlockData(first, afterLast uint64) (headers []*types.Header, receipts []types.Receipts) {
	s.lock.Lock()
	head := s.lastHead
	s.historicRefBlock = head.Number.Uint64()
	s.lock.Unlock()
	numbers := make([]uint64, afterLast+1-first)
	for number := first; number < afterLast; number++ {
		numbers[number-first] = number
	}
	numbers[afterLast-first] = head.Number.Uint64()
	hashes, err := s.parent.chain.GetCanonicalHashes(numbers)
	if err != nil || uint64(len(hashes)) != afterLast+1-first || hashes[afterLast-first] != head.Hash() {
		return
	}
	headers = make([]*types.Header, 0, afterLast-first)
	receipts = make([]types.Receipts, 0, afterLast-first)
	for number := first; number < afterLast; number++ {
		hash := hashes[number-first]
		header := s.parent.chain.GetHeader(hash, number)
		if header == nil {
			if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
				s.lastHistoryErrorLog = time.Now()
				log.Error("Historical header missing", "number", number, "hash", hash)
			}
			s.historyCutoff = number + 1
			continue
		}
		blockReceipts := s.parent.chain.GetReceipts(hash, number)
		if blockReceipts == nil {
			if time.Since(s.lastHistoryErrorLog) >= time.Second*10 {
				s.lastHistoryErrorLog = time.Now()
				log.Error("Historical receipts are missing", "number", number, "hash", hash)
			}
			s.historyCutoff = number + 1
			continue
		}
		headers = append(headers, header)
		receipts = append(receipts, blockReceipts)
	}
	return
}
