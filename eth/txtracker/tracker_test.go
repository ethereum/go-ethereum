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
type mockChain struct {
	mu       sync.Mutex
	headFeed event.Feed
	blocks   map[uint64]*types.Block
	finalNum uint64
}

func newMockChain() *mockChain {
	return &mockChain{blocks: make(map[uint64]*types.Block)}
}

func (c *mockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return c.headFeed.Subscribe(ch)
}

func (c *mockChain) GetBlockByNumber(number uint64) *types.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.blocks[number]
}

func (c *mockChain) CurrentFinalBlock() *types.Header {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finalNum == 0 {
		return nil
	}
	return &types.Header{Number: new(big.Int).SetUint64(c.finalNum)}
}

func (c *mockChain) addBlock(num uint64, txs []*types.Transaction) {
	c.mu.Lock()
	defer c.mu.Unlock()
	header := &types.Header{Number: new(big.Int).SetUint64(num)}
	c.blocks[num] = types.NewBlock(header, &types.Body{Transactions: txs}, nil, trie.NewListHasher())
}

func (c *mockChain) setFinalBlock(num uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.finalNum = num
}

func (c *mockChain) sendHead(num uint64) {
	c.headFeed.Send(core.ChainHeadEvent{
		Header: &types.Header{Number: new(big.Int).SetUint64(num)},
	})
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

// waitForHead gives the tracker time to process a chain head event.
func waitForHead() {
	time.Sleep(50 * time.Millisecond)
}

func TestNotifyReceived(t *testing.T) {
	tr := New()
	chain := newMockChain()
	tr.Start(chain)
	defer tr.Stop()

	txs := []*types.Transaction{makeTx(1), makeTx(2), makeTx(3)}
	tr.NotifyAccepted("peerA", hashTxs(txs))

	// No chain events yet — stats should be empty.
	stats := tr.GetAllPeerStats()
	if len(stats) != 0 {
		t.Fatalf("expected empty stats before any chain events, got %d peers", len(stats))
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
	waitForHead()

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentIncluded <= 0 {
		t.Fatalf("expected RecentIncluded > 0 after inclusion, got %f", stats["peerA"].RecentIncluded)
	}
	ema1 := stats["peerA"].RecentIncluded

	// Block 2 has no txs from peerA — EMA should decay.
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitForHead()

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
	waitForHead()

	// Not finalized yet.
	stats := tr.GetAllPeerStats()
	if stats["peerA"].Finalized != 0 {
		t.Fatalf("expected Finalized=0 before finalization, got %d", stats["peerA"].Finalized)
	}

	// Finalize block 1, then send head 2 to trigger checkFinalization.
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitForHead()

	stats = tr.GetAllPeerStats()
	if stats["peerA"].Finalized != 1 {
		t.Fatalf("expected Finalized=1 after finalization, got %d", stats["peerA"].Finalized)
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
	waitForHead()

	// Finalize.
	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitForHead()

	stats := tr.GetAllPeerStats()
	if stats["peerA"].Finalized != 1 {
		t.Fatalf("peerA: expected Finalized=1, got %d", stats["peerA"].Finalized)
	}
	if stats["peerB"].Finalized != 1 {
		t.Fatalf("peerB: expected Finalized=1, got %d", stats["peerB"].Finalized)
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
	waitForHead()

	chain.setFinalBlock(1)
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitForHead()

	stats := tr.GetAllPeerStats()
	if stats["peerA"].Finalized != 1 {
		t.Fatalf("peerA should be credited, got Finalized=%d", stats["peerA"].Finalized)
	}
	if stats["peerB"].Finalized != 0 {
		t.Fatalf("peerB should NOT be credited, got Finalized=%d", stats["peerB"].Finalized)
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
	waitForHead()

	// Send more heads without finalization.
	chain.addBlock(2, nil)
	chain.sendHead(2)
	waitForHead()

	stats := tr.GetAllPeerStats()
	if stats["peerA"].Finalized != 0 {
		t.Fatalf("expected Finalized=0 without finalization, got %d", stats["peerA"].Finalized)
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
	waitForHead()

	// Send 30 empty blocks — EMA should decay close to zero.
	for i := uint64(2); i <= 31; i++ {
		chain.addBlock(i, nil)
		chain.sendHead(i)
		waitForHead()
	}

	stats := tr.GetAllPeerStats()
	if stats["peerA"].RecentIncluded > 0.02 {
		t.Fatalf("expected RecentIncluded near zero after 30 empty blocks, got %f", stats["peerA"].RecentIncluded)
	}
}
