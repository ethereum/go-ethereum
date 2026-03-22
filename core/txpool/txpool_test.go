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
type trackedHeadSubChain struct{ nilHeadSubChain }

func (nilHeadSubChain) Config() *params.ChainConfig { return params.TestChainConfig }

func (nilHeadSubChain) CurrentBlock() *types.Header { return &types.Header{Root: types.EmptyRootHash} }

func (nilHeadSubChain) SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (nilHeadSubChain) StateAt(common.Hash) (*state.StateDB, error) {
	return state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
}

func (trackedHeadSubChain) SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription {
	return event.NewSubscription(func(<-chan struct{}) error { return nil })
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

func TestTxPoolNewTracksHeadSubscription(t *testing.T) {
	t.Parallel()

	pool, err := New(0, trackedHeadSubChain{}, nil)
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	if count := pool.subs.Count(); count != 1 {
		t.Fatalf("unexpected subscription count: have %d want %d", count, 1)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}
