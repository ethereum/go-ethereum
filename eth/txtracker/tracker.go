// Copyright 2026 The go-ethereum Authors
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

// Package txtracker maps accepted transactions to their delivering peer
// and observes chain-head and finalization events to emit per-block
// per-peer signals to a StatsConsumer (typically eth/peerstats).
//
// The tracker owns the tx-hash → deliverer-peer map with FIFO eviction,
// a chain-head subscription goroutine, and the computation of per-block
// inclusion counts and finalization credits. It does NOT maintain
// per-peer aggregates — that is peerstats' job.
package txtracker

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// Maximum number of tx→deliverer mappings to retain.
	maxTracked = 262144
)

// Chain is the blockchain interface needed by the tracker.
type Chain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	GetBlockByNumber(number uint64) *types.Block
	GetBlock(hash common.Hash, number uint64) *types.Block
	CurrentFinalBlock() *types.Header
}

// StatsConsumer receives per-block signals about peer inclusion and
// finalization. The tracker invokes NotifyBlock exactly once per handled chain
// head, AFTER releasing its own lock, with:
//
//   - inclusions: per-peer count of transactions in the head block
//   - finalized:  per-peer count of transactions in blocks that became
//     finalized since the previous call (possibly zero-range)
//
// Either map may be empty but the slice/map itself is never nil when
// called. NotifyBlock must not call back into the tracker.
type StatsConsumer interface {
	NotifyBlock(inclusions, finalized map[string]int)
}

// Tracker records which peer delivered each transaction and emits
// per-block inclusion and finalization signals to a StatsConsumer.
type Tracker struct {
	mu    sync.Mutex
	txs   map[common.Hash]string // hash → deliverer peer ID
	order []common.Hash          // insertion order for FIFO eviction

	chain        Chain
	consumer     StatsConsumer
	lastFinalNum uint64 // last finalized block number processed
	headCh       chan core.ChainHeadEvent
	sub          event.Subscription

	quit chan struct{}
	step chan struct{} // test sync: sent after each event is processed
	wg   sync.WaitGroup
}

// New creates a new tracker.
func New() *Tracker {
	return &Tracker{
		txs:  make(map[common.Hash]string),
		quit: make(chan struct{}),
		step: make(chan struct{}, 1),
	}
}

// Start begins listening for chain head events. `consumer` receives
// per-block signals; if nil, signals are computed but discarded
// (useful in tests that exercise only the tx-lifecycle surface).
func (t *Tracker) Start(chain Chain, consumer StatsConsumer) {
	t.chain = chain
	t.consumer = consumer
	// Seed lastFinalNum so checkFinalization doesn't backfill from genesis.
	if fh := chain.CurrentFinalBlock(); fh != nil {
		t.lastFinalNum = fh.Number.Uint64()
	}
	t.headCh = make(chan core.ChainHeadEvent, 128)
	t.sub = chain.SubscribeChainHeadEvent(t.headCh)
	t.wg.Add(1)
	go t.loop()
}

// Stop shuts down the tracker.
func (t *Tracker) Stop() {
	t.sub.Unsubscribe()
	close(t.quit)
	t.wg.Wait()
}

// NotifyAccepted records that a peer delivered transactions that were accepted
// by the pool. Only accepted (not rejected/duplicate) txs should be recorded
// to prevent attribution poisoning from replayed or invalid txs.
// Safe to call from any goroutine.
func (t *Tracker) NotifyAccepted(peer string, hashes []common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, hash := range hashes {
		if _, ok := t.txs[hash]; ok {
			continue // already tracked, keep first deliverer
		}
		t.txs[hash] = peer
		t.order = append(t.order, hash)
	}
	// Evict oldest entries if over capacity.
	for len(t.txs) > maxTracked {
		oldest := t.order[0]
		t.order = t.order[1:]
		delete(t.txs, oldest)
	}
	// Compact the backing array when it grows too large. Reslicing
	// with order[1:] doesn't free earlier slots in the array.
	if cap(t.order) > 2*maxTracked {
		t.order = append([]common.Hash(nil), t.order...)
	}
}

func (t *Tracker) loop() {
	defer t.wg.Done()

	for {
		select {
		case ev := <-t.headCh:
			t.handleChainHead(ev)
			select {
			case t.step <- struct{}{}:
			default:
			}
		case <-t.sub.Err():
			return
		case <-t.quit:
			return
		}
	}
}

// handleChainHead computes per-peer deltas for the new head block and any
// newly-finalized blocks, then hands them to the StatsConsumer AFTER
// releasing t.mu. The lock-release-before-consumer pattern avoids any
// cross-package lock ordering.
func (t *Tracker) handleChainHead(ev core.ChainHeadEvent) {
	// Fetch the head block by hash (not just number) to avoid using a
	// reorged block if the tracker goroutine lags behind the chain.
	block := t.chain.GetBlock(ev.Header.Hash(), ev.Header.Number.Uint64())
	if block == nil {
		return
	}

	t.mu.Lock()
	// Count per-peer inclusions in the head block.
	inclusions := make(map[string]int)
	for _, tx := range block.Transactions() {
		if peer := t.txs[tx.Hash()]; peer != "" {
			inclusions[peer]++
		}
	}
	// Compute per-peer finalization credits since the last call.
	finalized := t.collectFinalization()
	t.mu.Unlock()

	if t.consumer != nil {
		t.consumer.NotifyBlock(inclusions, finalized)
	}
}

// collectFinalization accumulates per-peer finalization credits for
// blocks newly finalized since lastFinalNum. Returns a (possibly empty)
// map; advances lastFinalNum. Must be called with t.mu held.
func (t *Tracker) collectFinalization() map[string]int {
	credits := make(map[string]int)
	finalHeader := t.chain.CurrentFinalBlock()
	if finalHeader == nil {
		return credits
	}
	finalNum := finalHeader.Number.Uint64()
	if finalNum <= t.lastFinalNum {
		return credits
	}
	for num := t.lastFinalNum + 1; num <= finalNum; num++ {
		block := t.chain.GetBlockByNumber(num)
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions() {
			if peer := t.txs[tx.Hash()]; peer != "" {
				credits[peer]++
			}
		}
	}
	if total := sumCounts(credits); total > 0 {
		log.Trace("Accumulated finalization credits",
			"from", t.lastFinalNum+1, "to", finalNum, "txs", total)
	}
	t.lastFinalNum = finalNum
	return credits
}

func sumCounts(m map[string]int) int {
	var sum int
	for _, v := range m {
		sum += v
	}
	return sum
}
