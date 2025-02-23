// Copyright 2023 The go-ethereum Authors
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
package ethclient

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// subscribeNewHeadPolling creates a block filter and polls it.
// This is the fallback for SubscribeNewHead on HTTP connections.
func (ec *Client) subscribeNewHeadPolling(ctx context.Context, ch chan<- *types.Header) (event.Subscription, error) {
	var id string
	err := ec.c.CallContext(ctx, &id, "eth_newBlockFilter")
	if err != nil {
		return nil, err
	}
	sub := newFilterSub(ec.c, id, ch)
	return sub, nil
}

// subscribeFilterLogs subscribes to a log filter.
// This is the fallback for SubscribeFilterLogs on HTTP connections.
func (ec *Client) subscribeFilterLogsPolling(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	var id string
	err = ec.c.CallContext(ctx, &id, "eth_newFilter", arg)
	if err != nil {
		return nil, err
	}
	sub := newFilterSub(ec.c, id, ch)
	return sub, nil
}

// filterSub implements event.Subscription with the polling filter API.
type filterSub[Result any] struct {
	// The channel to send the results to
	resultc   chan<- Result
	errc      chan error
	closed    chan struct{}
	unsubOnce sync.Once

	id     string
	client *rpc.Client
}

func newFilterSub[Result any](client *rpc.Client, id string, resultc chan<- Result) *filterSub[Result] {
	sub := &filterSub[Result]{
		resultc: resultc,
		errc:    make(chan error, 1),
		closed:  make(chan struct{}),
		client:  client,
	}
	go sub.poll()
	return sub
}

// Unsubscribe cancels the event subscription.
func (s *filterSub[Result]) Unsubscribe() {
	s.closeWithError(nil)
}

// Err returns the subscription error channel.
func (s *filterSub[Result]) Err() <-chan error {
	return s.errc
}

func (s *filterSub[Result]) poll() {
	var timer = time.NewTicker(10 * time.Second)
	for {
		err := s.getChanges()
		if err != nil {
			s.closeWithError(err)
			return
		}

		// Wait for next time.
		select {
		case <-timer.C:
		case <-s.closed:
			return
		}
	}
}

func (s *filterSub[Result]) closeWithError(err error) {
	s.unsubOnce.Do(func() {
		close(s.closed)
		unsubErr := s.uninstallFilter()
		if unsubErr != nil && err == nil {
			err = unsubErr
		}
		if err != nil {
			select {
			case s.errc <- err:
			default:
			}
		}
		close(s.errc)
	})
}

// getChanges calls eth_getFilterChanges and delivers the results.
func (s *filterSub[Result]) getChanges() error {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	var results []Result
	err := s.client.CallContext(ctx, &results, "eth_getFilterChanges", s.id)
	if err != nil {
		return err
	}
	for _, result := range results {
		select {
		case s.resultc <- result:
		case <-s.closed:
			return nil
		}
	}
	return nil
}

// uninstallFilter removes the filter on the server side.
func (s *filterSub[Result]) uninstallFilter() error {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	return s.client.CallContext(ctx, nil, "eth_uninstallFilter", s.id)
}
