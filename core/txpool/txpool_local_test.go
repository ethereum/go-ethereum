package txpool

import (
	"math/big"
	"reflect"
	"sync"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/event"
)

type testChain struct{}

func (testChain) CurrentBlock() *types.Header { return &types.Header{Number: big.NewInt(0)} }

func (testChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

type testLocalTracker struct {
	mu      sync.Mutex
	events  *[]string
	tracked []common.Hash
}

func (t *testLocalTracker) Track(tx *types.Transaction) {
	t.TrackAll([]*types.Transaction{tx})
}

func (t *testLocalTracker) TrackAll(txs []*types.Transaction) {
	t.mu.Lock()
	defer t.mu.Unlock()
	*t.events = append(*t.events, "track")
	for _, tx := range txs {
		t.tracked = append(t.tracked, tx.Hash())
	}
}

type testSubPool struct {
	events *[]string

	lastAdd  []*types.Transaction
	lastSync bool
}

func (s *testSubPool) Filter(tx *types.Transaction) bool { return true }

func (s *testSubPool) Init(gasTip uint64, head *types.Header, reserver *Reserver) error { return nil }

func (s *testSubPool) Close() error { return nil }

func (s *testSubPool) Reset(oldHead, newHead *types.Header) {}

func (s *testSubPool) SetGasTip(tip *big.Int) error { return nil }

func (s *testSubPool) Has(hash common.Hash) bool { return false }

func (s *testSubPool) Get(hash common.Hash) *types.Transaction { return nil }

func (s *testSubPool) Add(txs []*types.Transaction, sync bool) []error {
	*s.events = append(*s.events, "add")
	s.lastAdd = txs
	s.lastSync = sync
	return make([]error, len(txs))
}

func (s *testSubPool) Pending(filter PendingFilter) map[common.Address][]*LazyTransaction {
	return map[common.Address][]*LazyTransaction{}
}

func (s *testSubPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

func (s *testSubPool) Nonce(addr common.Address) uint64 { return 0 }

func (s *testSubPool) Stats() (int, int) { return 0, 0 }

func (s *testSubPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return map[common.Address][]*types.Transaction{}, map[common.Address][]*types.Transaction{}
}

func (s *testSubPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}

func (s *testSubPool) Status(hash common.Hash) TxStatus { return TxStatusUnknown }

func (s *testSubPool) SetSigner(f func(address common.Address) bool) {}

func (s *testSubPool) IsSigner(addr common.Address) bool { return false }

func TestAddLocalTracksBeforeAdd(t *testing.T) {
	events := []string{}
	tracker := &testLocalTracker{events: &events}
	subpool := &testSubPool{events: &events}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	pool.SetLocalTracker(tracker)

	tx := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	if err := pool.AddLocal(tx, true); err != nil {
		t.Fatalf("AddLocal failed: %v", err)
	}

	if len(tracker.tracked) != 1 || tracker.tracked[0] != tx.Hash() {
		t.Fatalf("tracker did not receive local tx hash")
	}
	if len(subpool.lastAdd) != 1 || subpool.lastAdd[0].Hash() != tx.Hash() {
		t.Fatalf("subpool Add did not receive local tx")
	}
	if !subpool.lastSync {
		t.Fatalf("sync flag not propagated to subpool Add")
	}
	if !reflect.DeepEqual(events, []string{"track", "add"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalsTracksBeforeAdd(t *testing.T) {
	events := []string{}
	tracker := &testLocalTracker{events: &events}
	subpool := &testSubPool{events: &events}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	pool.SetLocalTracker(tracker)

	tx0 := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	tx1 := types.NewTransaction(1, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	txs := []*types.Transaction{tx0, tx1}

	errs := pool.AddLocals(txs, true)
	if len(errs) != len(txs) {
		t.Fatalf("unexpected error result length: have %d, want %d", len(errs), len(txs))
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("AddLocals error at index %d: %v", i, err)
		}
	}

	hashes := []common.Hash{tx0.Hash(), tx1.Hash()}
	if len(tracker.tracked) != len(hashes) {
		t.Fatalf("tracker tx count mismatch: have %d, want %d", len(tracker.tracked), len(hashes))
	}
	if !reflect.DeepEqual(tracker.tracked, hashes) {
		t.Fatalf("tracker hashes mismatch: have %v, want %v", tracker.tracked, hashes)
	}

	if len(subpool.lastAdd) != len(hashes) {
		t.Fatalf("subpool Add tx count mismatch: have %d, want %d", len(subpool.lastAdd), len(hashes))
	}
	for i, tx := range subpool.lastAdd {
		if tx.Hash() != hashes[i] {
			t.Fatalf("subpool Add hash mismatch at index %d", i)
		}
	}
	if !subpool.lastSync {
		t.Fatalf("sync flag not propagated to subpool Add")
	}
	if !reflect.DeepEqual(events, []string{"track", "add"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}
