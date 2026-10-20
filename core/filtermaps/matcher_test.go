// Copyright 2024 The go-ethereum Authors
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

package filtermaps

import (
	"context"
	crand "crypto/rand"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestMatcher(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.chain.addBlocks(100, 10, 10, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	for i := 0; i < 2000; i++ {
		bhash := ts.chain.canonical[rand.Intn(len(ts.chain.canonical))]
		receipts := ts.chain.receipts[bhash]
		if len(receipts) == 0 {
			continue
		}
		receipt := receipts[rand.Intn(len(receipts))]
		if len(receipt.Logs) == 0 {
			continue
		}
		log := receipt.Logs[rand.Intn(len(receipt.Logs))]
		var ok bool
		addresses := make([]common.Address, rand.Intn(3))
		for i := range addresses {
			crand.Read(addresses[i][:])
		}
		if len(addresses) > 0 {
			addresses[rand.Intn(len(addresses))] = log.Address
			ok = true
		}
		topics := make([][]common.Hash, rand.Intn(len(log.Topics)+1))
		for j := range topics {
			topics[j] = make([]common.Hash, rand.Intn(3))
			for i := range topics[j] {
				crand.Read(topics[j][i][:])
			}
			if len(topics[j]) > 0 {
				topics[j][rand.Intn(len(topics[j]))] = log.Topics[j]
				ok = true
			}
		}
		if !ok {
			continue // cannot search for match-all pattern
		}
		mb := ts.fm.NewMatcherBackend()
		logs, err := GetPotentialMatches(context.Background(), mb, 0, 1000, addresses, topics)
		mb.Close()
		if err != nil {
			t.Fatalf("Log search error: %v", err)
		}
		var found bool
		for _, l := range logs {
			if l == log {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Log search did not return expected log (addresses: %v, topics: %v, expected log: %v)", addresses, topics, *log)
		}
	}
}

// hookBackend forwards every request to the wrapped backend, optionally calling
// a hook first. Once armed, it counts the requests that were served without an
// error.
type hookBackend struct {
	MatcherBackend
	beforeGetFilterMapRows func()
	beforeGetLogByLvIndex  func()

	armed                  atomic.Bool
	servedRows, servedLogs atomic.Int32 // requests started after arming and served successfully
}

func (b *hookBackend) GetFilterMapRows(ctx context.Context, mapIndices []uint32, rowIndex uint32, baseLayerOnly bool) ([]FilterRow, error) {
	if b.beforeGetFilterMapRows != nil {
		b.beforeGetFilterMapRows()
	}
	armed := b.armed.Load()
	rows, err := b.MatcherBackend.GetFilterMapRows(ctx, mapIndices, rowIndex, baseLayerOnly)
	if armed && err == nil {
		b.servedRows.Add(1)
	}
	return rows, err
}

func (b *hookBackend) GetLogByLvIndex(ctx context.Context, lvIndex uint64) (*types.Log, error) {
	if b.beforeGetLogByLvIndex != nil {
		b.beforeGetLogByLvIndex()
	}
	armed := b.armed.Load()
	log, err := b.MatcherBackend.GetLogByLvIndex(ctx, lvIndex)
	if armed && err == nil {
		b.servedLogs.Add(1)
	}
	return log, err
}

// newCancelTestSetup creates an indexed test chain and returns a matcher backend
// along with an address filter matching at least one of its logs.
func newCancelTestSetup(t *testing.T) (*testSetup, *FilterMapsMatcherBackend, []common.Address) {
	ts := newTestSetup(t)
	ts.chain.addBlocks(100, 10, 10, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	for _, bhash := range ts.chain.canonical {
		for _, receipt := range ts.chain.receipts[bhash] {
			if len(receipt.Logs) > 0 {
				return ts, ts.fm.NewMatcherBackend(), []common.Address{receipt.Logs[0].Address}
			}
		}
	}
	ts.close()
	t.Fatal("Test chain has no logs")
	return nil, nil, nil
}

// waitSearch waits for a cancelled search to return its error.
func waitSearch(t *testing.T, errCh chan error) error {
	select {
	case err := <-errCh:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Cancelled log search did not return")
		return nil
	}
}

// TestMatcherCancel checks that a cancelled search returns with the context
// error and does not process any further requests.
func TestMatcherCancel(t *testing.T) {
	ts, mb, addresses := newCancelTestSetup(t)
	defer ts.close()
	defer mb.Close()

	// search with an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GetPotentialMatches(ctx, mb, 0, 1000, addresses, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Search with cancelled context returned error %v, want %v", err, context.Canceled)
	}

	// cancel the search when it first accesses the filter maps
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var (
		backend = &hookBackend{MatcherBackend: mb}
		once    sync.Once
		errCh   = make(chan error, 1)
	)
	backend.beforeGetFilterMapRows = func() {
		once.Do(func() {
			cancel()
			backend.armed.Store(true)
		})
	}
	go func() {
		_, err := GetPotentialMatches(ctx, backend, 0, 1000, addresses, nil)
		errCh <- err
	}()
	if err := waitSearch(t, errCh); !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancelled search returned error %v, want %v", err, context.Canceled)
	}
	if rows, logs := backend.servedRows.Load(), backend.servedLogs.Load(); rows != 0 || logs != 0 {
		t.Fatalf("Cancelled search kept running (rows served: %d, logs served: %d)", rows, logs)
	}
}

// TestMatcherCancelWhileWaitingForIndexLock checks that log lookups waiting for
// the indexer to release the index lock do not proceed once the lock is
// released if the search has been cancelled in the meantime.
func TestMatcherCancelWhileWaitingForIndexLock(t *testing.T) {
	ts, mb, addresses := newCancelTestSetup(t)
	defer ts.close()
	defer mb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		backend = &hookBackend{MatcherBackend: mb}
		once    sync.Once
		locked  = make(chan struct{})
		errCh   = make(chan error, 1)
	)
	backend.beforeGetLogByLvIndex = func() {
		once.Do(func() {
			// stand in for the indexer writing a batch of rendered maps
			ts.fm.indexLock.Lock()
			backend.armed.Store(true)
			close(locked)
		})
	}
	go func() {
		_, err := GetPotentialMatches(ctx, backend, 0, 1000, addresses, nil)
		errCh <- err
	}()
	select {
	case <-locked:
	case err := <-errCh:
		t.Fatalf("Search finished without a log lookup (error: %v)", err)
	}
	cancel()
	ts.fm.indexLock.Unlock()

	if err := waitSearch(t, errCh); !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancelled search returned error %v, want %v", err, context.Canceled)
	}
	if logs := backend.servedLogs.Load(); logs != 0 {
		t.Fatalf("Cancelled search looked up %d logs after the index lock was released", logs)
	}
}
