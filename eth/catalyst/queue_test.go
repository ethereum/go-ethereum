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

package catalyst

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
)

type queueTestBackend struct {
	chain *core.BlockChain
	pool  *txpool.TxPool
}

func (b *queueTestBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *queueTestBackend) TxPool() *txpool.TxPool       { return b.pool }

type queueTestEngine struct {
	consensus.Engine
	calls   int
	entered chan struct{}
	release chan struct{}
}

func (e *queueTestEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	e.calls++
	if e.calls == 2 {
		close(e.entered)
		<-e.release
	}
	return e.Engine.Prepare(chain, header)
}

func TestPayloadQueuePutWhileResolving(t *testing.T) {
	for _, test := range []struct {
		name string
		full bool
	}{{"latest", false}, {"full", true}} {
		t.Run(test.name, func(t *testing.T) {
			fake := ethash.NewFaker()
			chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), &core.Genesis{Config: params.TestChainConfig}, fake, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer chain.Stop()
			pool, err := txpool.New(0, chain, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			synctest.Test(t, func(t *testing.T) {
				gate := &queueTestEngine{Engine: fake, entered: make(chan struct{}), release: make(chan struct{})}
				worker := miner.New(&queueTestBackend{chain, pool}, miner.DefaultConfig, gate)
				args := &miner.BuildPayloadArgs{Parent: chain.CurrentBlock().Hash(), Timestamp: 1, InclusionList: [][]byte{{1}}}
				payload, err := worker.BuildPayload(context.Background(), args, false)
				if err != nil {
					t.Fatal(err)
				}
				var release sync.Once
				defer func() {
					release.Do(func() { close(gate.release) })
					payload.Resolve()
				}()
				<-gate.entered
				queue := newPayloadQueue()
				queue.put(args.Id(), payload)
				resolved := make(chan *engine.ExecutionPayloadEnvelope, 1)
				go func() { resolved <- queue.get(args.Id(), test.full) }()
				synctest.Wait()
				if len(resolved) != 0 {
					t.Fatal("getter did not wait for the initial inclusion list build")
				}
				// The getter is waiting for the miner. A new FCU must still be able to
				// acquire the write lock and publish its payload ID.
				if !queue.lock.TryLock() {
					t.Fatal("payload resolution retains the queue lock, blocking new payload IDs")
				}
				queue.lock.Unlock()
				next := engine.PayloadID{1}
				queue.put(next, payload)
				if !queue.has(next) {
					t.Fatal("new payload ID was not inserted")
				}
				release.Do(func() { close(gate.release) })
				if <-resolved == nil {
					t.Fatal("queued payload was lost during insertion")
				}
			})
		})
	}
}
