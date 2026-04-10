// Package txtracker provides minimal per-peer transaction inclusion tracking.
// It records which peer delivered each transaction and credits peers when
// their delivered transactions are included on chain.
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
	Included       int64   // Cumulative on-chain inclusions attributed to this peer
	RecentIncluded float64 // EMA of per-block inclusions
}

// Chain is the blockchain interface needed by the tracker.
type Chain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	GetBlockByNumber(number uint64) *types.Block
}

type peerStats struct {
	included       int64
	recentIncluded float64
}

// Tracker records which peer delivered each transaction and credits peers
// when their transactions appear on chain.
type Tracker struct {
	mu    sync.Mutex
	txs   map[common.Hash]string // hash → deliverer peer ID
	peers map[string]*peerStats
	order []common.Hash // insertion order for LRU eviction

	chain  Chain
	headCh chan core.ChainHeadEvent
	sub    event.Subscription

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
			Included:       ps.included,
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
	block := t.chain.GetBlockByNumber(ev.Header.Number.Uint64())
	if block == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	// Credit delivering peers for each included transaction.
	blockIncl := make(map[string]int)
	for _, tx := range block.Transactions() {
		hash := tx.Hash()
		peer, ok := t.txs[hash]
		if !ok || peer == "" {
			continue
		}
		ps := t.peers[peer]
		if ps == nil {
			ps = &peerStats{}
			t.peers[peer] = ps
		}
		ps.included++
		blockIncl[peer]++
	}
	// Update per-peer recent-inclusion EMA for all tracked peers.
	for peer, ps := range t.peers {
		ps.recentIncluded = (1-emaAlpha)*ps.recentIncluded + emaAlpha*float64(blockIncl[peer])
	}
	if len(blockIncl) > 0 {
		log.Trace("Credited peers for block inclusions", "block", ev.Header.Number, "peers", len(blockIncl))
	}
}
