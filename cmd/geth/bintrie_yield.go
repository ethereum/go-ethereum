// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core"
)

// blockYield lets an import running inside the node stand aside while the
// node processes a block: the import's loops pause between batches for as
// long as one is in flight. A block takes a few percent of its slot, so the
// import loses little time, and the block gets the cores, the memory
// bandwidth and the disk to itself. A nil blockYield never pauses.
type blockYield struct {
	busy atomic.Bool
	mu   sync.Mutex
	idle chan struct{} // closed while no block is processed
}

func newBlockYield() *blockYield {
	y := &blockYield{idle: make(chan struct{})}
	close(y.idle)
	return y
}

// set records whether the node is processing a block.
func (y *blockYield) set(processing bool) {
	y.mu.Lock()
	defer y.mu.Unlock()
	if processing == y.busy.Load() {
		return
	}
	if processing {
		y.idle = make(chan struct{})
	} else {
		close(y.idle)
	}
	y.busy.Store(processing)
}

// pause waits while the node processes a block, then returns ctx's error, if
// any: the import's loops call it between batches, where they check for
// cancellation.
func (y *blockYield) pause(ctx context.Context) error {
	if y != nil && y.busy.Load() {
		y.mu.Lock()
		idle := y.idle
		y.mu.Unlock()
		select {
		case <-idle:
		case <-ctx.Done():
		}
	}
	return ctx.Err()
}

// followBlockProcessing feeds y from the chain's block processing events, as
// the log indexer follows them (filtermaps.SetBlockProcessing), until the
// returned stop is called.
func followBlockProcessing(chain *core.BlockChain, y *blockYield) (stop func()) {
	ch := make(chan bool, 16)
	sub := chain.SubscribeBlockProcessingEvent(ch)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case processing := <-ch:
				y.set(processing)
			case <-sub.Err():
				return
			}
		}
	}()
	return func() {
		sub.Unsubscribe()
		<-done
		y.set(false)
	}
}
