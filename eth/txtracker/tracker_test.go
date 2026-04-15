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

// addBlock adds a canonical block at the given height.
func (c *mockChain) addBlock(num uint64, txs []*types.Transaction) *types.Block {
	return c.addBlockAtHeight(num, num, txs, true)
}

// addBlockAtHeight adds a block at the given height. The salt parameter
// ensures distinct block hashes for two blocks at the same height. If
// canonical is true, the block becomes the canonical block for that height.
func (c *mockChain) addBlockAtHeight(num, salt uint64, txs []*types.Transaction, canonical bool) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
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

// sendHead emits a chain head event for the canonical block at the given height.
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

// mockConsumer captures NotifyBlock invocations so tests can assert on the
// signals the tracker emits.
type mockConsumer struct {
	mu      sync.Mutex
	signals []signal
}

type signal struct {
	inclusions, finalized map[string]int
}

func (c *mockConsumer) NotifyBlock(inclusions, finalized map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Deep-copy so tests inspecting older signals aren't tripped up by
	// later iterations mutating the same map (they don't today, but
	// this keeps the assertion model simple).
	in := make(map[string]int, len(inclusions))
	for k, v := range inclusions {
		in[k] = v
	}
	fn := make(map[string]int, len(finalized))
	for k, v := range finalized {
		fn[k] = v
	}
	c.signals = append(c.signals, signal{in, fn})
}

func (c *mockConsumer) last() signal {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.signals) == 0 {
		return signal{}
	}
	return c.signals[len(c.signals)-1]
}

func (c *mockConsumer) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.signals)
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

// TestNotifyAcceptedRecordsMapping verifies the tx-lifecycle surface:
// NotifyAccepted records tx→peer mappings in insertion order, with
// first-deliverer-wins semantics on duplicates.
func TestNotifyAcceptedRecordsMapping(t *testing.T) {
	tr := New()

	txs := []*types.Transaction{makeTx(1), makeTx(2), makeTx(3)}
	hashes := hashTxs(txs)
	tr.NotifyAccepted("peerA", hashes)

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

// TestNotifyAcceptedFirstDelivererWins verifies duplicate accepts
// preserve the original deliverer.
func TestNotifyAcceptedFirstDelivererWins(t *testing.T) {
	tr := New()
	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})
	tr.NotifyAccepted("peerB", []common.Hash{tx.Hash()})

	tr.mu.Lock()
	defer tr.mu.Unlock()
	if got := tr.txs[tx.Hash()]; got != "peerA" {
		t.Fatalf("expected first deliverer peerA to win, got %q", got)
	}
	if len(tr.order) != 1 {
		t.Fatalf("expected single order entry, got %d", len(tr.order))
	}
}

// TestHandleChainHeadEmitsInclusions verifies the tracker emits a
// correct per-peer inclusion map to its consumer when a head block
// contains tracked transactions.
func TestHandleChainHeadEmitsInclusions(t *testing.T) {
	tr := New()
	chain := newMockChain()
	consumer := &mockConsumer{}
	tr.Start(chain, consumer)
	defer tr.Stop()

	tx1, tx2 := makeTx(1), makeTx(2)
	tr.NotifyAccepted("peerA", []common.Hash{tx1.Hash()})
	tr.NotifyAccepted("peerB", []common.Hash{tx2.Hash()})

	chain.addBlock(1, []*types.Transaction{tx1, tx2})
	chain.sendHead(1)
	waitStep(t, tr)

	sig := consumer.last()
	if sig.inclusions["peerA"] != 1 {
		t.Errorf("peerA inclusions: got %d, want 1", sig.inclusions["peerA"])
	}
	if sig.inclusions["peerB"] != 1 {
		t.Errorf("peerB inclusions: got %d, want 1", sig.inclusions["peerB"])
	}
	if len(sig.finalized) != 0 {
		t.Errorf("expected empty finalized map, got %v", sig.finalized)
	}
}

// TestHandleChainHeadEmptyBlock verifies an empty head block emits an
// empty inclusion map (so peerstats can decay all known peers).
func TestHandleChainHeadEmptyBlock(t *testing.T) {
	tr := New()
	chain := newMockChain()
	consumer := &mockConsumer{}
	tr.Start(chain, consumer)
	defer tr.Stop()

	chain.addBlock(1, nil)
	chain.sendHead(1)
	waitStep(t, tr)

	sig := consumer.last()
	if len(sig.inclusions) != 0 {
		t.Errorf("expected empty inclusions, got %v", sig.inclusions)
	}
}

// TestHandleChainHeadEmitsFinalization verifies that when finalization
// advances, the consumer receives per-peer finalization credits
// accumulated over the newly-finalized range.
func TestHandleChainHeadEmitsFinalization(t *testing.T) {
	tr := New()
	chain := newMockChain()
	consumer := &mockConsumer{}
	tr.Start(chain, consumer)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Include in block 1, not yet finalized.
	chain.addBlock(1, []*types.Transaction{tx})
	chain.sendHead(1)
	waitStep(t, tr)

	if credits := consumer.last().finalized["peerA"]; credits != 0 {
		t.Fatalf("expected no finalization credits before finalization, got %d", credits)
	}

	// Finalize block 1; next head triggers the finalization scan.
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitStep(t, tr)

	if credits := consumer.last().finalized["peerA"]; credits != 1 {
		t.Fatalf("expected 1 finalization credit, got %d", credits)
	}
}

// TestReorgSafety verifies the tracker resolves the head block by HASH
// so a head event pointing at a sibling block does not emit inclusions
// from the canonical block at the same height.
func TestReorgSafety(t *testing.T) {
	tr := New()
	chain := newMockChain()
	consumer := &mockConsumer{}
	tr.Start(chain, consumer)
	defer tr.Stop()

	tx := makeTx(1)
	tr.NotifyAccepted("peerA", []common.Hash{tx.Hash()})

	// Two blocks at height 1: canonical A contains tx; sibling B does not.
	blockA := chain.addBlockAtHeight(1, 1, []*types.Transaction{tx}, true)
	blockB := chain.addBlockAtHeight(1, 2, nil, false)
	if blockA.Hash() == blockB.Hash() {
		t.Fatal("sibling blocks ended up with the same hash")
	}

	// Head announces sibling B — emit must contain no peerA inclusions.
	chain.sendHeadBlock(blockB)
	waitStep(t, tr)
	if incl := consumer.last().inclusions["peerA"]; incl != 0 {
		t.Fatalf("sibling-B head should emit 0 peerA inclusions, got %d", incl)
	}

	// Head announces canonical A — emit must contain 1 peerA inclusion.
	chain.sendHeadBlock(blockA)
	waitStep(t, tr)
	if incl := consumer.last().inclusions["peerA"]; incl != 1 {
		t.Fatalf("canonical-A head should emit 1 peerA inclusion, got %d", incl)
	}
}

// TestHandleChainHeadNilConsumer verifies the tracker tolerates a nil
// consumer (useful for tests that only exercise tx-lifecycle behavior).
func TestHandleChainHeadNilConsumer(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain, nil)
	defer tr.Stop()

	chain.addBlock(1, nil)
	chain.sendHead(1)
	waitStep(t, tr) // should not panic
}
