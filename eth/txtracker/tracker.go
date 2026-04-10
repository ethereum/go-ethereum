// Package txtracker provides minimal per-peer transaction inclusion tracking.
//
// It records which peer delivered each transaction body (via NotifyReceived)
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
)

// PeerStats holds the per-peer inclusion data.
type PeerStats struct {
	Finalized      int64   // Cumulative finalized inclusions attributed to this peer
	RecentIncluded float64 // EMA of per-block inclusions (at chain head time)
}

// Chain is the blockchain interface needed by the tracker.
type Chain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	GetBlockByNumber(number uint64) *types.Block
	CurrentFinalBlock() *types.Header
}

type peerStats struct {
	finalized      int64
	recentIncluded float64
}

// Tracker records which peer delivered each transaction and credits peers
// when their transactions appear on chain.
type Tracker struct {
	mu    sync.Mutex
	txs   map[common.Hash]string // hash → deliverer peer ID
	peers map[string]*peerStats
	order []common.Hash // insertion order for LRU eviction

	chain        Chain
	lastFinalNum uint64 // last finalized block number processed
	headCh       chan core.ChainHeadEvent
	sub          event.Subscription

	quit chan struct{}
	wg   sync.WaitGroup
}

// New creates a new tracker.
func New() *Tracker {
	return &Tracker{
		txs:   make(map[common.Hash]string),
		peers: make(map[string]*peerStats),
		quit:  make(chan struct{}),
	}
}

// Start begins listening for chain head events.
func (t *Tracker) Start(chain Chain) {
	t.chain = chain
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

// NotifyReceived records that a peer delivered transaction bodies.
// Safe to call from any goroutine.
func (t *Tracker) NotifyReceived(peer string, txs []*types.Transaction) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, tx := range txs {
		hash := tx.Hash()
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
}

// GetAllPeerStats returns a snapshot of per-peer inclusion statistics.
// Safe to call from any goroutine.
func (t *Tracker) GetAllPeerStats() map[string]PeerStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make(map[string]PeerStats, len(t.peers))
	for id, ps := range t.peers {
		result[id] = PeerStats{
			Finalized:      ps.finalized,
			RecentIncluded: ps.recentIncluded,
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
		case <-t.sub.Err():
			return
		case <-t.quit:
			return
		}
	}
}

func (t *Tracker) handleChainHead(ev core.ChainHeadEvent) {
	// Update recent-inclusion EMA from the new head block.
	block := t.chain.GetBlockByNumber(ev.Header.Number.Uint64())
	if block == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	// Count per-peer inclusions in this block for the EMA.
	blockIncl := make(map[string]int)
	for _, tx := range block.Transactions() {
		if peer := t.txs[tx.Hash()]; peer != "" {
			blockIncl[peer]++
		}
	}
	// Ensure peers with inclusions in this block have entries.
	for peer := range blockIncl {
		if t.peers[peer] == nil {
			t.peers[peer] = &peerStats{}
		}
	}
	// Update EMA for all tracked peers (decay inactive ones).
	for peer, ps := range t.peers {
		ps.recentIncluded = (1-emaAlpha)*ps.recentIncluded + emaAlpha*float64(blockIncl[peer])
	}
	// Check if the finalized block has advanced.
	t.checkFinalization()
}

// checkFinalization credits peers for transactions in newly finalized blocks.
// Must be called with t.mu held.
func (t *Tracker) checkFinalization() {
	finalHeader := t.chain.CurrentFinalBlock()
	if finalHeader == nil {
		return
	}
	finalNum := finalHeader.Number.Uint64()
	if finalNum <= t.lastFinalNum {
		return
	}
	// Credit peers for all blocks from lastFinalNum+1 to finalNum.
	var credited int
	for num := t.lastFinalNum + 1; num <= finalNum; num++ {
		block := t.chain.GetBlockByNumber(num)
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions() {
			peer := t.txs[tx.Hash()]
			if peer == "" {
				continue
			}
			ps := t.peers[peer]
			if ps == nil {
				ps = &peerStats{}
				t.peers[peer] = ps
			}
			ps.finalized++
			credited++
		}
	}
	if credited > 0 {
		log.Trace("Credited peers for finalized inclusions",
			"from", t.lastFinalNum+1, "to", finalNum, "txs", credited)
	}
	t.lastFinalNum = finalNum
}
