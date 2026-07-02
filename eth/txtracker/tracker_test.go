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
// by GetCanonicalHash in the finalization-credit path.
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

func (c *mockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.blocksByHash[hash]
}

func (c *mockChain) GetCanonicalHash(number uint64) common.Hash {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.canonicalByNum[number]
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
// block for that height (looked up by GetCanonicalHash).
func (c *mockChain) addBlockAtHeight(num, salt uint64, txs []*types.Transaction, canonical bool) *types.Block {
	return c.addBlockAtHeightWithTime(num, salt, txs, canonical, uint64(time.Now().Unix()+3600))
}

// addBlockAtHeightWithTime is like addBlockAtHeight but takes an explicit
// block time. Used by the pre-slot gate test, which needs a block whose
// slot start is BEFORE the moment NotifyAccepted recorded its tx.
func (c *mockChain) addBlockAtHeightWithTime(num, salt uint64, txs []*types.Transaction, canonical bool, blockTime uint64) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Mix salt into Extra so siblings at the same height get distinct hashes.
	header := &types.Header{
		Number: new(big.Int).SetUint64(num),
		Extra:  big.NewInt(int64(salt)).Bytes(),
		Time:   blockTime,
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
		if got := tr.txs[h]; got.Deliverer != "peerA" {
			t.Fatalf("tx %d: expected deliverer=peerA, got %q", i, got.Deliverer)
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
// Regression check: handleChainHead uses GetBlock(hash, number) so a head
// event announcing sibling B fetches B, not the canonical block A.
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

// TestPreSlotGate verifies that a tx delivered at or after the slot of its
// inclusion block earns no credit. This blocks the simple
// post-block-propagation re-broadcast attribution attack: a peer that
// learns a tx from the just-mined block and re-broadcasts it to our pool
// should not gain credit when that block is processed. The finalization
// path applies the same gate (ti.AddedAt >= blockTime) and is exercised
// by the existing TestFinalization with the new clock semantics.
func TestPreSlotGate(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	// Pin the tracker's clock so NotifyAccepted records a known addedAt.
	const delivery = uint64(1_000_000)
	tr.now = func() uint64 { return delivery }

	// peerA delivers two txs at the same instant.
	preTx := makeTx(1)
	postTx := makeTx(2)
	tr.NotifyAccepted("peerA", []common.Hash{preTx.Hash(), postTx.Hash()})

	// Block 1: slot strictly AFTER delivery — pre-slot, credit allowed.
	chain.addBlockAtHeightWithTime(1, 1, []*types.Transaction{preTx}, true, delivery+100)
	chain.sendHead(1)
	waitStep(t, tr)

	preEMA := tr.GetAllPeerStats()["peerA"].RecentIncluded
	if preEMA <= 0 {
		t.Fatalf("expected RecentIncluded>0 after pre-slot delivery, got %f", preEMA)
	}

	// Block 2: slot strictly BEFORE delivery — post-slot, must NOT credit.
	chain.addBlockAtHeightWithTime(2, 2, []*types.Transaction{postTx}, true, delivery-1)
	chain.sendHead(2)
	waitStep(t, tr)

	// With the gate, only EMA decay occurs (no contribution this block).
	// Without the gate, RecentIncluded would have ticked up again.
	after := tr.GetAllPeerStats()["peerA"].RecentIncluded
	if after >= preEMA {
		t.Fatalf("expected EMA to decay (no credit for post-slot tx), got %f >= preEMA %f", after, preEMA)
	}
}
