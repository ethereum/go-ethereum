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

package txtracker

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/trie"
)

// mockChain implements the Chain interface for testing.
//
// Blocks are stored by hash to exercise the reorg-safe lookup path in
// tracker.handleChainHead (which calls GetBlock(hash, number)). A separate
// canonicalByNum index maps each height to its canonical block hash, used
// by GetBlockByNumber (the finalization path).
type mockChain struct {
	mu             sync.Mutex
	headFeed       event.Feed
	blocksByHash   map[common.Hash]*types.Block
	canonicalByNum map[uint64]common.Hash
	finalNum       uint64
}

func newMockChain() *mockChain {
	return &mockChain{
		blocksByHash:   make(map[common.Hash]*types.Block),
		canonicalByNum: make(map[uint64]common.Hash),
	}
}

func (c *mockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return c.headFeed.Subscribe(ch)
}

func (c *mockChain) GetBlockByNumber(number uint64) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	hash, ok := c.canonicalByNum[number]
	if !ok {
		return nil
	}
	return c.blocksByHash[hash]
}

func (c *mockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.blocksByHash[hash]
}

func (c *mockChain) CurrentFinalBlock() *types.Header {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finalNum == 0 {
		return nil
	}
	return &types.Header{Number: new(big.Int).SetUint64(c.finalNum)}
}

// addBlock adds a canonical block at the given height. Overwrites any
// prior canonical block at that height.
func (c *mockChain) addBlock(num uint64, txs []*types.Transaction) *types.Block {
	return c.addBlockAtHeight(num, num, txs, true)
}

// addBlockAtHeight adds a block at the given height. The salt parameter
// ensures distinct block hashes for two blocks at the same height (used
// for reorg tests). If canonical is true, the block becomes the canonical
// block for that height (looked up by GetBlockByNumber).
func (c *mockChain) addBlockAtHeight(num, salt uint64, txs []*types.Transaction, canonical bool) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Mix salt into Extra so siblings at the same height get distinct hashes.
	header := &types.Header{
		Number: new(big.Int).SetUint64(num),
		Extra:  big.NewInt(int64(salt)).Bytes(),
	}
	block := types.NewBlock(header, &types.Body{Transactions: txs}, nil, trie.NewListHasher())
	c.blocksByHash[block.Hash()] = block
	if canonical {
		c.canonicalByNum[num] = block.Hash()
	}
	return block
}

func (c *mockChain) setFinalBlock(num uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.finalNum = num
}

// sendHead emits a chain head event for the canonical block at the given
// height. The emitted header carries the real block's hash so the
// tracker's GetBlock(hash, number) lookup resolves correctly.
func (c *mockChain) sendHead(num uint64) {
	c.mu.Lock()
	hash := c.canonicalByNum[num]
	block := c.blocksByHash[hash]
	c.mu.Unlock()
	if block == nil {
		panic("sendHead: no canonical block at height")
	}
	c.headFeed.Send(core.ChainHeadEvent{Header: block.Header()})
}

// sendHeadBlock emits a chain head event for the given block (may be
// non-canonical). Used for reorg tests.
func (c *mockChain) sendHeadBlock(block *types.Block) {
	c.headFeed.Send(core.ChainHeadEvent{Header: block.Header()})
}

func hashTxs(txs []*types.Transaction) []common.Hash {
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	return hashes
}

func makeTx(nonce uint64) *types.Transaction {
	return types.NewTx(&types.LegacyTx{Nonce: nonce, GasPrice: big.NewInt(1), Gas: 21000})
}

// waitStep blocks until the tracker has processed one event.
func waitStep(t *testing.T, tr *Tracker) {
	t.Helper()
	select {
	case <-tr.step:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for tracker step")
	}
}

func TestNotifyReceived(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	txs := []*types.Transaction{makeTx(1), makeTx(2), makeTx(3)}
	hashes := hashTxs(txs)
	tr.NotifyAccepted("peerA", hashes)

	// Public surface: peer entry was created with zero stats before any
	// chain events. Map lookups would return a zero value for a missing
	// key, so assert presence explicitly.
	stats := tr.GetAllPeerStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 peer entry, got %d", len(stats))
	}
	ps, ok := stats["peerA"]
	if !ok {
		t.Fatal("expected peerA entry, not found")
	}
	if ps.RecentFinalized != 0 || ps.RecentIncluded != 0 {
		t.Fatalf("expected zero stats before chain events, got %+v", ps)
	}

	// Internal state: all tx→deliverer mappings recorded, insertion order
	// preserved in the FIFO slice.
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.txs) != 3 {
		t.Fatalf("expected 3 tracked txs, got %d", len(tr.txs))
	}
	if len(tr.order) != 3 {
		t.Fatalf("expected order length 3, got %d", len(tr.order))
	}
	for i, h := range hashes {
		if got := tr.txs[h]; got != "peerA" {
			t.Fatalf("tx %d: expected deliverer=peerA, got %q", i, got)
		}
		if tr.order[i] != h {
			t.Fatalf("order[%d] mismatch", i)
		}
	}
}

func TestInclusionEMA(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Block 1 includes peerA's tx.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentIncluded <= 0 {
		t.Fatalf("expected RecentIncluded > 0 after inclusion, got %f", stats["peerA"].RecentIncluded)
	}
	ema1 := stats["peerA"].RecentIncluded

	// Block 2 has no txs from peerA — EMA should decay.
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	stats = tr.GetAllPeerStats()
	if stats["peerA"].RecentIncluded >= ema1 {
		t.Fatalf("expected EMA to decay, got %f >= %f", stats["peerA"].RecentIncluded, ema1)
	}
}

func TestFinalization(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Include in block 1.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	// Not finalized yet.
	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentFinalized != 0 {
		t.Fatalf("expected RecentFinalized=0 before finalization, got %f", stats["peerA"].RecentFinalized)
	}

	// Finalize block 1, then send head 2 to trigger the finalization EMA update.
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	stats = tr.GetAllPeerStats()
	if stats["peerA"].RecentFinalized <= 0 {
		t.Fatalf("expected RecentFinalized>0 after finalization, got %f", stats["peerA"].RecentFinalized)
	}
}

func TestMultiplePeers(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx1 := makeTx(1)
	tx2 := makeTx(2)
	tr.NotifyAccepted("peerA", []common.Hash{tx1.Hash()})
	tr.NotifyAccepted("peerB", []common.Hash{tx2.Hash()})

	// Both included in block 1.
	chain.addBlock(1, []*types.Transaction{tx1, tx2})
	chain.sendHead(1)
	waitStep(t, tr)

	// Finalize.
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentFinalized <= 0 {
		t.Fatalf("peerA: expected RecentFinalized>0, got %f", stats["peerA"].RecentFinalized)
	}
	if stats["peerB"].RecentFinalized <= 0 {
		t.Fatalf("peerB: expected RecentFinalized>0, got %f", stats["peerB"].RecentFinalized)
	}
}

func TestFirstDelivererWins(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})
	tr.NotifyAccepted("peerB", []common.Hash{tx.Hash()}) // duplicate, should be ignored

	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentFinalized <= 0 {
		t.Fatalf("peerA should be credited, got RecentFinalized=%f", stats["peerA"].RecentFinalized)
	}
	if stats["peerB"].RecentFinalized != 0 {
		t.Fatalf("peerB should NOT be credited, got RecentFinalized=%f", stats["peerB"].RecentFinalized)
	}
}

func TestNoFinalizationCredit(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Include but don't finalize.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	// Send more heads without finalization.
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentFinalized != 0 {
		t.Fatalf("expected RecentFinalized=0 without finalization, got %f", stats["peerA"].RecentFinalized)
	}
}

func TestEMADecay(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Include in block 1.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	// Send 30 empty blocks — EMA should decay close to zero.
	for i := uint64(2); i <= 31; i++ {
		chain.addBlock(i, nil)
		chain.sendHead(i)
		waitStep(t, tr)
	}

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentIncluded > 0.02 {
		t.Fatalf("expected RecentIncluded near zero after 30 empty blocks, got %f", stats["peerA"].RecentIncluded)
	}
}

// TestReorgSafety verifies that handleChainHead resolves the head block by
// HASH (not just by number), so a head event announcing a sibling block at
// the same height does not credit transactions from the canonical block.
//
// Regression check: if the tracker were changed to use GetBlockByNumber,
// it would always fetch the canonical block A and credit peerA even when
// the head points to sibling B.
func TestReorgSafety(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Two blocks at height 1: canonical A contains tx; sibling B does not.
	blockA := chain.addBlockAtHeight(1, 1, []*types.Transaction{tx}, true)
	blockB := chain.addBlockAtHeight(1, 2, nil, false)
	if blockA.Hash() == blockB.Hash() {
		t.Fatal("sibling blocks ended up with the same hash")
	}

	// Head announces sibling B. A hash-aware tracker fetches B, sees no
	// peerA txs, and leaves the EMA at zero. A number-only tracker would
	// instead fetch A and credit peerA.
	chain.sendHeadBlock(blockB)
	waitStep(t, tr)

	if got := tr.GetAllPeerStats()["peerA"].RecentIncluded; got != 0 {
		t.Fatalf("expected RecentIncluded=0 after sibling-B head event, got %f (tracker followed the wrong block)", got)
	}

	// Now announce canonical A; peerA should be credited.
	chain.sendHeadBlock(blockA)
	waitStep(t, tr)

	if got := tr.GetAllPeerStats()["peerA"].RecentIncluded; got <= 0 {
		t.Fatalf("expected RecentIncluded>0 after canonical-A head event, got %f", got)
	}
}

// TestRecentFinalizedDecays verifies that the finalization EMA decays
// for a peer that earned credits in the past but has no new
// finalization activity. The decay is slow (α=0.0001), so we
// just assert monotonic decrease, not convergence to zero.
func TestRecentFinalizedDecays(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Include and finalize in block 1.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	peak := tr.GetAllPeerStats()["peerA"].RecentFinalized
	if peak <= 0 {
		t.Fatalf("expected RecentFinalized>0 after finalization, got %f", peak)
	}

	// Send many empty heads — peer contributes zero each block,
	// EMA should decay monotonically.
	for i := uint64(3); i <= 50; i++ {
		chain.addBlock(i, nil)
		chain.sendHead(i)
		waitStep(t, tr)
	}

	after := tr.GetAllPeerStats()["peerA"].RecentFinalized
	if after >= peak {
		t.Fatalf("expected RecentFinalized to decay, got %f >= peak %f", after, peak)
	}
}

// TestRequestLatencyFirstSampleBootstrap asserts that the first latency
// sample seeds the EMA directly (no slow ramp-up from zero), and that the
// sample counter starts at 1.
func TestRequestLatencyFirstSampleBootstrap(t *testing.T) {
	tr := New()
	tr.NotifyRequestLatency("peerA", 200*time.Millisecond)

	stats := tr.GetAllPeerStats()
	ps := stats["peerA"]
	if ps.RequestLatencyEMA != 200*time.Millisecond {
		t.Fatalf("expected first sample to seed EMA at 200ms, got %v", ps.RequestLatencyEMA)
	}
	if ps.RequestSamples != 1 {
		t.Fatalf("expected RequestSamples=1, got %d", ps.RequestSamples)
	}
}

// TestRequestLatencyEMAUpdate verifies the EMA formula (1-α)·old + α·new.
func TestRequestLatencyEMAUpdate(t *testing.T) {
	tr := New()
	tr.NotifyRequestLatency("peerA", 100*time.Millisecond)
	tr.NotifyRequestLatency("peerA", 1000*time.Millisecond)

	// Expected: 0.99*100ms + 0.01*1000ms = 109ms
	got := tr.GetAllPeerStats()["peerA"].RequestLatencyEMA
	want := 109 * time.Millisecond
	delta := got - want
	if delta < 0 {
		delta = -delta
	}
	if delta > 1*time.Microsecond {
		t.Fatalf("EMA mismatch: got %v, want %v (delta %v)", got, want, delta)
	}
	if samples := tr.GetAllPeerStats()["peerA"].RequestSamples; samples != 2 {
		t.Fatalf("expected RequestSamples=2, got %d", samples)
	}
}

// TestRequestLatencySlowEMAConvergence verifies that the slow alpha
// requires many samples to noticeably shift the EMA. Starting at 100ms
// and feeding 5s (timeout) samples, the EMA should still be well below
// 1s after 50 samples.
func TestRequestLatencySlowEMAConvergence(t *testing.T) {
	tr := New()
	tr.NotifyRequestLatency("peerA", 100*time.Millisecond)
	for i := 0; i < 50; i++ {
		tr.NotifyRequestLatency("peerA", 5*time.Second)
	}
	got := tr.GetAllPeerStats()["peerA"].RequestLatencyEMA
	if got < 1*time.Second {
		// Expected ≈ (0.99)^50 * 100ms + (1-(0.99)^50) * 5s ≈ 1.99s
		// The lower bound proves a meaningful shift; the upper bound (below)
		// proves the slow alpha damped the convergence.
		t.Fatalf("EMA did not move enough under sustained timeouts, got %v", got)
	}
	if got > 3*time.Second {
		t.Fatalf("EMA converged too fast for slow alpha=0.01, got %v", got)
	}
}

// TestRequestLatencyMultiplePeersIsolated verifies per-peer isolation: a
// sample for peerA does not affect peerB's stats.
func TestRequestLatencyMultiplePeersIsolated(t *testing.T) {
	tr := New()
	tr.NotifyRequestLatency("peerA", 100*time.Millisecond)
	tr.NotifyRequestLatency("peerB", 5*time.Second)

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RequestLatencyEMA != 100*time.Millisecond {
		t.Errorf("peerA EMA: got %v, want 100ms", stats["peerA"].RequestLatencyEMA)
	}
	if stats["peerB"].RequestLatencyEMA != 5*time.Second {
		t.Errorf("peerB EMA: got %v, want 5s", stats["peerB"].RequestLatencyEMA)
	}
	if stats["peerA"].RequestSamples != 1 || stats["peerB"].RequestSamples != 1 {
		t.Errorf("expected RequestSamples=1 for each peer, got A=%d B=%d",
			stats["peerA"].RequestSamples, stats["peerB"].RequestSamples)
	}
}

// TestRequestLatencyPeerDropResetsStats verifies that NotifyPeerDrop
// removes the peer's latency history along with its other stats.
func TestRequestLatencyPeerDropResetsStats(t *testing.T) {
	tr := New()
	tr.NotifyRequestLatency("peerA", 200*time.Millisecond)
	tr.NotifyPeerDrop("peerA")

	if _, ok := tr.GetAllPeerStats()["peerA"]; ok {
		t.Fatal("peerA stats should be removed after NotifyPeerDrop")
	}

	// A subsequent latency sample re-creates the entry as a fresh peer.
	tr.NotifyRequestLatency("peerA", 50*time.Millisecond)
	ps := tr.GetAllPeerStats()["peerA"]
	if ps.RequestSamples != 1 {
		t.Fatalf("expected RequestSamples=1 after re-add, got %d", ps.RequestSamples)
	}
	if ps.RequestLatencyEMA != 50*time.Millisecond {
		t.Fatalf("expected fresh EMA bootstrap, got %v", ps.RequestLatencyEMA)
	}
}
