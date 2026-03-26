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

package txpool

import (
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

type nilHeadSubChain struct{}

func (nilHeadSubChain) Config() *params.ChainConfig { return params.TestChainConfig }

func (nilHeadSubChain) CurrentBlock() *types.Header { return &types.Header{Root: types.EmptyRootHash} }

func (nilHeadSubChain) SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (nilHeadSubChain) StateAt(common.Hash) (*state.StateDB, error) {
	return state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
}

type trackedHeadSubChain struct {
	nilHeadSubChain
	sub *subscriptionSpy
}

func (c *trackedHeadSubChain) SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription {
	c.sub = newSubscriptionSpy()
	return c.sub
}

type subscriptionSpy struct {
	err    chan error
	mu     sync.Mutex
	once   sync.Once
	closed bool
}

func newSubscriptionSpy() *subscriptionSpy {
	return &subscriptionSpy{err: make(chan error)}
}

func (s *subscriptionSpy) Unsubscribe() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.err)
	})
}

func (s *subscriptionSpy) Err() <-chan error {
	return s.err
}

func (s *subscriptionSpy) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

type failingSubPool struct{}

func (failingSubPool) Filter(*types.Transaction) bool { return false }

func (failingSubPool) FilterType(byte) bool { return false }

func (failingSubPool) Init(uint64, *types.Header, Reserver) error {
	return errors.New("boom")
}

func (failingSubPool) Close() error { return nil }

func (failingSubPool) Reset(*types.Header, *types.Header) {}

func (failingSubPool) SetGasTip(*big.Int) {}

func (failingSubPool) Has(common.Hash) bool { return false }

func (failingSubPool) Get(common.Hash) *types.Transaction { return nil }

func (failingSubPool) GetRLP(common.Hash) []byte { return nil }

func (failingSubPool) GetMetadata(common.Hash) *TxMetadata { return nil }

func (failingSubPool) ValidateTxBasics(*types.Transaction) error { return nil }

func (failingSubPool) Add([]*types.Transaction, bool) []error { return nil }

func (failingSubPool) Pending(PendingFilter) (map[common.Address][]*LazyTransaction, int) {
	return nil, 0
}

func (failingSubPool) SubscribeTransactions(chan<- core.NewTxsEvent, bool) event.Subscription {
	return nil
}

func (failingSubPool) Nonce(common.Address) uint64 { return 0 }

func (failingSubPool) Stats() (int, int) { return 0, 0 }

func (failingSubPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return nil, nil
}

func (failingSubPool) ContentFrom(common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}

func (failingSubPool) Status(common.Hash) TxStatus { return TxStatusUnknown }

func (failingSubPool) Clear() {}

func TestTxPoolCloseUnsubscribesHeadSubscription(t *testing.T) {
	t.Parallel()

	chain := &trackedHeadSubChain{}
	pool, err := New(0, chain, nil)
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	if chain.sub == nil {
		t.Fatal("expected head subscription")
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !chain.sub.isClosed() {
		t.Fatal("expected head subscription to be unsubscribed on close")
	}
}

func TestTxPoolNewUnsubscribesHeadSubscriptionOnInitFailure(t *testing.T) {
	t.Parallel()

	chain := &trackedHeadSubChain{}
	if _, err := New(0, chain, []SubPool{failingSubPool{}}); err == nil {
		t.Fatal("expected init failure")
	}
	if chain.sub == nil {
		t.Fatal("expected head subscription")
	}
	if !chain.sub.isClosed() {
		t.Fatal("expected head subscription to be unsubscribed on init failure")
	}
}

func TestTxPoolCloseNilHeadSubscription(t *testing.T) {
	t.Parallel()

	// TxPool.BlockChain exists to allow mocked chains in tests. A mock that
	// opts out of head notifications may return a nil subscription.
	pool, err := New(0, nilHeadSubChain{}, nil)
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	select {
	case <-pool.term:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for txpool loop termination")
	}
}
