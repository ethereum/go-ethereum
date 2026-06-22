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

// Package txtracker provides minimal per-peer transaction inclusion tracking.
//
// It records which peer delivered each accepted transaction (via NotifyAccepted)
// and monitors the chain for inclusion and finalization events. When a
// delivered transaction is finalized on chain, the delivering peer is
// credited. A per-block exponential moving average (EMA) of inclusions
// tracks recent peer productivity.
//
// The primary consumer is the peer dropper (eth/dropper.go), which uses
// these stats to protect high-value peers from random disconnection.
package txtracker

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// Maximum number of tx→deliverer mappings to retain.
	maxTracked = 262144
	// EMA smoothing factor for per-block inclusion rate.
	emaAlpha = 0.05
	// EMA smoothing factor for per-block finalization rate. Very slow on
	// purpose: finalization is permanent, and the score should reflect
	// sustained contribution over long windows, not recent bursts.
	// Half-life ≈ 6930 chain heads (~23 hours on 12s blocks).
	finalizedEMAAlpha = 0.0001
)

// PeerStats holds the per-peer inclusion data.
type PeerStats struct {
	RecentFinalized float64 // EMA of per-block finalization credits (slow)
	RecentIncluded  float64 // EMA of per-block inclusions (fast)
}

// Chain is the blockchain interface needed by the tracker.
type Chain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	GetBlockByNumber(number uint64) *types.Block
	GetBlock(hash common.Hash, number uint64) *types.Block
	CurrentFinalBlock() *types.Header
}

type peerStats struct {
	recentFinalized float64
	recentIncluded  float64
}

// txEntry tracks the deliverer of a transaction and when delivery was
// recorded. addedAt is unix seconds; it is compared against block.Time()
// to suppress credit for txs delivered at or after the slot of their
// inclusion block (e.g., re-broadcasts of just-mined txs).
type txEntry struct {
	peer    string
	addedAt uint64
}

// Tracker records which peer delivered each transaction and credits peers
// when their transactions appear on chain.
type Tracker struct {
	mu    sync.Mutex
	txs   map[common.Hash]txEntry // hash → deliverer + arrival time
	peers map[string]*peerStats
	order []common.Hash // insertion order for LRU eviction

	chain        Chain
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
		txs:   make(map[common.Hash]txEntry),
		peers: make(map[string]*peerStats),
		quit:  make(chan struct{}),
		step:  make(chan struct{}, 1),
		now:   func() uint64 { return uint64(time.Now().Unix()) },
	}
}

// Start begins listening for chain head events.
func (t *Tracker) Start(chain Chain) {
	t.chain = chain
	// Seed lastFinalNum so checkFinalization doesn't backfill from genesis.
	if fh := chain.CurrentFinalBlock(); fh != nil {
		t.lastFinalNum = fh.Number.Uint64()
	}
	t.headCh = make(chan core.ChainHeadEvent, 128)
	t.sub = chain.SubscribeChainHeadEvent(t.headCh)
	t.wg.Add(1)
	go t.loop()
}

// NotifyPeerDrop removes a disconnected peer's stats to prevent unbounded
// growth. Safe to call from any goroutine.
func (t *Tracker) NotifyPeerDrop(peer string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, peer)
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
		if _, ok := t.txs[hash]; ok {
			continue // already tracked, keep first deliverer
		}
		t.txs[hash] = txEntry{peer: peer, addedAt: addedAt}
		t.order = append(t.order, hash)
	}
	// Ensure the delivering peer has a stats entry.
	if len(hashes) > 0 && t.peers[peer] == nil {
		t.peers[peer] = &peerStats{}
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

// GetAllPeerStats returns a snapshot of per-peer inclusion statistics.
// Safe to call from any goroutine.
func (t *Tracker) GetAllPeerStats() map[string]PeerStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make(map[string]PeerStats, len(t.peers))
	for id, ps := range t.peers {
		result[id] = PeerStats{
			RecentFinalized: ps.recentFinalized,
			RecentIncluded:  ps.recentIncluded,
		}
	}
	return result
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

func (t *Tracker) handleChainHead(ev core.ChainHeadEvent) {
	// Fetch the head block by hash (not just number) to avoid using a
	// reorged block if the tracker goroutine lags behind the chain.
	block := t.chain.GetBlock(ev.Header.Hash(), ev.Header.Number.Uint64())
	if block == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	// Count per-peer inclusions in this block for the inclusion EMA.
	// Skip txs whose delivery arrived at or after this block's slot — those
	// are likely post-slot re-broadcasts of an already-mined tx, not genuine
	// relay work.
	blockTime := block.Time()
	blockIncl := make(map[string]int)
	for _, tx := range block.Transactions() {
		entry, ok := t.txs[tx.Hash()]
		if !ok || entry.addedAt >= blockTime {
			continue
		}
		blockIncl[entry.peer]++
	}
	// Accumulate per-peer finalization credits over the newly-finalized
	// range (possibly zero blocks). Only counts peers still tracked.
	blockFinal := t.collectFinalizationCredits()

	// Update both EMAs for all tracked peers (decays inactive ones).
	// Don't create entries for unknown peers — they may have been
	// removed by NotifyPeerDrop and should not be resurrected.
	for peer, ps := range t.peers {
		ps.recentIncluded = (1-emaAlpha)*ps.recentIncluded + emaAlpha*float64(blockIncl[peer])
		ps.recentFinalized = (1-finalizedEMAAlpha)*ps.recentFinalized + finalizedEMAAlpha*float64(blockFinal[peer])
	}
}

// collectFinalizationCredits accumulates per-peer finalization credits for
// blocks newly finalized since lastFinalNum. Returns a (possibly empty) map
// keyed by peer ID; advances lastFinalNum. Must be called with t.mu held.
// Peers that have already been removed by NotifyPeerDrop are skipped so
// dropped peers are not resurrected by old on-chain data.
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
	for num := t.lastFinalNum + 1; num <= finalNum; num++ {
		block := t.chain.GetBlockByNumber(num)
		if block == nil {
			continue
		}
		blockTime := block.Time()
		for _, tx := range block.Transactions() {
			entry, ok := t.txs[tx.Hash()]
			if !ok || entry.addedAt >= blockTime {
				continue // unknown, or post-slot re-broadcast
			}
			if _, ok := t.peers[entry.peer]; !ok {
				continue // peer disconnected, skip credit
			}
			credits[entry.peer]++
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
