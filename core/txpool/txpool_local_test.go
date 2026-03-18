package txpool

import (
	"errors"
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
	t.mu.Lock()
	defer t.mu.Unlock()
	*t.events = append(*t.events, "track")
	t.tracked = append(t.tracked, tx.Hash())
}

func (t *testLocalTracker) IsRetryableReject(err error) bool {
	return errors.Is(err, ErrUnderpriced)
}

type testSubPool struct {
	events *[]string

	lastAdd  []*types.Transaction
	lastSync bool
	addErrs  []error
}

func (s *testSubPool) Filter(tx *types.Transaction) bool { return true }

func (s *testSubPool) Init(gasTip uint64, head *types.Header, reserver *Reserver) error { return nil }

func (s *testSubPool) Close() error { return nil }

func (s *testSubPool) Reset(oldHead, newHead *types.Header) {}

func (s *testSubPool) SetGasTip(tip *big.Int) error { return nil }

func (s *testSubPool) Has(hash common.Hash) bool { return false }

func (s *testSubPool) Get(hash common.Hash) *types.Transaction { return nil }

func (s *testSubPool) ValidateTxBasics(tx *types.Transaction) error { return nil }

func (s *testSubPool) Add(txs []*types.Transaction, sync bool) []error {
	*s.events = append(*s.events, "add")
	s.lastAdd = txs
	s.lastSync = sync
	if len(s.addErrs) > 0 {
		errs := make([]error, len(txs))
		copy(errs, s.addErrs)
		return errs
	}
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

func TestAddLocalTracksAfterAdd(t *testing.T) {
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
	if !reflect.DeepEqual(events, []string{"add", "track"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalMultipleTracksAfterAdd(t *testing.T) {
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
	if err := pool.AddLocal(tx0, true); err != nil {
		t.Fatalf("AddLocal tx0 failed: %v", err)
	}
	if err := pool.AddLocal(tx1, true); err != nil {
		t.Fatalf("AddLocal tx1 failed: %v", err)
	}

	hashes := []common.Hash{tx0.Hash(), tx1.Hash()}
	if len(tracker.tracked) != len(hashes) {
		t.Fatalf("tracker tx count mismatch: have %d, want %d", len(tracker.tracked), len(hashes))
	}
	if !reflect.DeepEqual(tracker.tracked, hashes) {
		t.Fatalf("tracker hashes mismatch: have %v, want %v", tracker.tracked, hashes)
	}

	if len(subpool.lastAdd) != 1 || subpool.lastAdd[0].Hash() != tx1.Hash() {
		t.Fatalf("subpool Add did not receive second local tx")
	}
	if !subpool.lastSync {
		t.Fatalf("sync flag not propagated to subpool Add")
	}
	if !reflect.DeepEqual(events, []string{"add", "track", "add", "track"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalMultipleTracksOnlyAcceptedTransactions(t *testing.T) {
	events := []string{}
	tracker := &testLocalTracker{events: &events}
	subpool := &testSubPool{
		events:  &events,
		addErrs: []error{ErrInvalidSender},
	}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	pool.SetLocalTracker(tracker)

	tx0 := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	tx1 := types.NewTransaction(1, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	if err := pool.AddLocal(tx0, true); !errors.Is(err, ErrInvalidSender) {
		t.Fatalf("unexpected first error: have %v, want %v", err, ErrInvalidSender)
	}
	subpool.addErrs = nil
	if err := pool.AddLocal(tx1, true); err != nil {
		t.Fatalf("unexpected second error: %v", err)
	}

	hashes := []common.Hash{tx1.Hash()}
	if !reflect.DeepEqual(tracker.tracked, hashes) {
		t.Fatalf("tracker hashes mismatch: have %v, want %v", tracker.tracked, hashes)
	}
	if !reflect.DeepEqual(events, []string{"add", "add", "track"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalTracksOnlyAcceptedTransaction(t *testing.T) {
	events := []string{}
	tracker := &testLocalTracker{events: &events}
	subpool := &testSubPool{
		events:  &events,
		addErrs: []error{ErrInvalidSender},
	}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	pool.SetLocalTracker(tracker)

	tx := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	err = pool.AddLocal(tx, true)
	if !errors.Is(err, ErrInvalidSender) {
		t.Fatalf("unexpected error: have %v, want %v", err, ErrInvalidSender)
	}

	if len(tracker.tracked) != 0 {
		t.Fatalf("tracker should not receive failed local tx, have %d tracked", len(tracker.tracked))
	}
	if !reflect.DeepEqual(events, []string{"add"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalTracksTemporaryRejectedTransaction(t *testing.T) {
	events := []string{}
	tracker := &testLocalTracker{events: &events}
	subpool := &testSubPool{
		events:  &events,
		addErrs: []error{ErrUnderpriced},
	}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	pool.SetLocalTracker(tracker)

	tx := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	err = pool.AddLocal(tx, true)
	if !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("unexpected error: have %v, want %v", err, ErrUnderpriced)
	}

	if !reflect.DeepEqual(tracker.tracked, []common.Hash{tx.Hash()}) {
		t.Fatalf("tracker should receive temporary rejected local tx")
	}
	if !reflect.DeepEqual(events, []string{"add", "track"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}

func TestAddLocalTemporaryRejectWithoutTrackerReturnsError(t *testing.T) {
	events := []string{}
	subpool := &testSubPool{
		events:  &events,
		addErrs: []error{ErrUnderpriced},
	}

	pool, err := New(0, testChain{}, []SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	tx := types.NewTransaction(0, common.Address{0x1}, big.NewInt(1), 21000, big.NewInt(1), nil)
	err = pool.AddLocal(tx, true)
	if !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("unexpected error: have %v, want %v", err, ErrUnderpriced)
	}
	if !reflect.DeepEqual(events, []string{"add"}) {
		t.Fatalf("unexpected call order: have %v", events)
	}
}
