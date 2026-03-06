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

func (nilHeadSubChain) CurrentBlock() *types.Header { return &types.Header{} }

func (nilHeadSubChain) SubscribeChainHeadEvent(chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (nilHeadSubChain) StateAt(common.Hash) (*state.StateDB, error) {
	return nil, errors.New("not implemented")
}

func TestTxPoolLoopNilHeadSubscription(t *testing.T) {
	t.Parallel()

	pool := &TxPool{
		chain: nilHeadSubChain{},
		quit:  make(chan chan error),
		term:  make(chan struct{}),
		sync:  make(chan chan error),
	}
	go pool.loop(nil)

	errc := make(chan error, 1)
	select {
	case pool.quit <- errc:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for txpool loop to accept quit signal")
	}
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("unexpected close error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for txpool loop to stop")
	}
	select {
	case <-pool.term:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for txpool loop termination")
	}
}
