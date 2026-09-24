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
// The tracker owns the tx-hash → deliverer mapping with FIFO eviction,
// a chain-head subscription goroutine, and the computation of per-block
// inclusion counts and finalization credits. It does NOT maintain
// per-peer aggregates — that is peerstats' job.
package txtracker

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
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
	GetBlock(hash common.Hash, number uint64) *types.Block
	GetCanonicalHash(number uint64) common.Hash
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
// Either map may be empty but the map itself is never nil when called.
// NotifyBlock must not call back into the tracker.
type StatsConsumer interface {
	NotifyBlock(inclusions, finalized map[string]int)
}

// TxInfo records the per-transaction state the tracker maintains.
//
// Deliverer is the peer that first handed us this tx via NotifyAccepted.
// AddedAt is the unix-seconds wall-clock at that moment; it is compared
// against block.Time() to suppress credit for txs delivered at or after
// the slot of their inclusion block (re-broadcasts of just-mined txs).
//
// BlockNum / BlockHash are populated when the tracker first sees the tx
// in a head block (BlockNum == 0 means not yet seen on chain). BlockHash
// is re-checked against canonical-at-height at finalization time so
// reorgs do not yield credit.
type TxInfo struct {
	Deliverer string
	AddedAt   uint64
	BlockNum  uint64
	BlockHash common.Hash
}

// Tracker records which peer delivered each transaction and emits
// per-block inclusion and finalization signals to a StatsConsumer.
type Tracker struct {
	mu  sync.Mutex
	txs lru.BasicLRU[common.Hash, *TxInfo] // tx hash -> tx info with lru eviction

	chain        Chain
	consumer     StatsConsumer
	lastFinalNum uint64 // last finalized block number processed
	headCh       chan core.ChainHeadEvent
	sub          event.Subscription

	quit     chan struct{}
	stopOnce sync.Once
	step     chan struct{} // test sync: sent after each event is processed
	now      func() uint64 // unix-seconds clock; overridable in tests
	wg       sync.WaitGroup
}

// New creates a new tracker.
func New() *Tracker {
	return &Tracker{
		txs:  lru.NewBasicLRU[common.Hash, *TxInfo](maxTracked),
		quit: make(chan struct{}),
		step: make(chan struct{}, 1),
		now:  func() uint64 { return uint64(time.Now().Unix()) },
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
	t.stopOnce.Do(func() {
		if t.sub != nil {
			t.sub.Unsubscribe()
		}
		close(t.quit)
	})
	t.wg.Wait()
}

// NotifyAccepted records that a peer delivered transactions that were accepted
// by the pool. Only accepted (not rejected/duplicate) txs should be recorded
// to prevent attribution poisoning from replayed or invalid txs.
// Safe to call from any goroutine.
func (t *Tracker) NotifyAccepted(peer string, hashes []common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	addedAt := t.now()
	for _, hash := range hashes {
		if t.txs.Contains(hash) {
			continue // already tracked, keep first deliverer
		}
		t.txs.Add(hash, &TxInfo{Deliverer: peer, AddedAt: addedAt})
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

	// Count per-peer inclusions in this block, and record (BlockNum,
	// BlockHash) on first inclusion so the iterate-t.txs finalization
	// scan can find the entry later without re-reading the block. Skip
	// txs whose delivery arrived at or after this block's slot — those
	// are likely post-slot re-broadcasts of an already-mined tx, not
	// genuine relay work.
	blockTime := block.Time()
	blockNum := block.Number().Uint64()
	blockHash := block.Hash()
	inclusions := make(map[string]int)
	for _, tx := range block.Transactions() {
		ti, ok := t.txs.Peek(tx.Hash())
		if !ok || ti.AddedAt >= blockTime {
			continue
		}
		inclusions[ti.Deliverer]++
		if ti.BlockNum == 0 {
			ti.BlockNum = blockNum
			ti.BlockHash = blockHash
		}
	}
	// Accumulate per-peer finalization credits over the newly-finalized
	// range (possibly zero blocks).
	finalized := t.collectFinalizationCredits()
	t.mu.Unlock()

	if t.consumer != nil {
		t.consumer.NotifyBlock(inclusions, finalized)
	}
}

// collectFinalizationCredits accumulates per-peer finalization credits for
// blocks newly finalized since lastFinalNum, and advances lastFinalNum.
// Returns a (possibly empty) credits map keyed by peer ID. Must be called
// with t.mu held.
//
// The pivot here is to iterate t.txs (which we already maintain by hash
// with BlockNum + BlockHash recorded at inclusion time) rather than
// walking each newly-finalized block from disk. The walk over chain
// blocks was the dominant cost during catch-up after a restart: every
// block called GetBlockByNumber (cold-disk RLP-decode) and then per-tx
// tx.Hash() and types.Sender() against fresh cache-cold *Transaction
// instances. By inverting, the only chain query is one cheap canonical-
// hash lookup per unique BlockNum that has tracked entries, used to
// confirm the recorded BlockHash is still on the canonical chain (and
// thus the tx really is finalized). No tx iteration, no hashing, no
// sender derivation against cold blocks.
func (t *Tracker) collectFinalizationCredits() map[string]int {
	credits := make(map[string]int)
	finalHeader := t.chain.CurrentFinalBlock()
	if finalHeader == nil {
		return credits
	}
	finalNum := finalHeader.Number.Uint64()
	if finalNum <= t.lastFinalNum {
		return credits
	}

	// Group entries by their recorded BlockNum so the canonical-hash
	// lookup happens once per height, not once per tx. The BlockNum range
	// check filters both "not yet seen on chain" (BlockNum == 0) and
	// "already credited in a prior pass" (BlockNum <= lastFinalNum); no
	// separate status bookkeeping is needed.
	buckets := make(map[uint64][]*TxInfo)
	for _, hash := range t.txs.Keys() {
		ti, ok := t.txs.Peek(hash)
		if !ok || ti.BlockNum <= t.lastFinalNum || ti.BlockNum > finalNum {
			continue
		}
		buckets[ti.BlockNum] = append(buckets[ti.BlockNum], ti)
	}

	total := 0
	for num, tis := range buckets {
		canonHash := t.chain.GetCanonicalHash(num)
		if canonHash == (common.Hash{}) {
			continue
		}
		for _, ti := range tis {
			// BlockHash was recorded when the entry was first seen
			// on chain. If it doesn't match the canonical hash now,
			// the entry's recorded inclusion is in an orphaned
			// block; skip rather than misreport finality.
			if ti.BlockHash != canonHash {
				continue
			}
			if ti.Deliverer != "" {
				credits[ti.Deliverer]++
				total++
			}
		}
	}

	if total > 0 {
		log.Trace("Accumulated finalization credits",
			"from", t.lastFinalNum+1, "to", finalNum, "txs", total)
	}
	t.lastFinalNum = finalNum
	return credits
}
