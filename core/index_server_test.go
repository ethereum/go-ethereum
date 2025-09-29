// Copyright 2025 The go-ethereum Authors
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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestIndexServer(t *testing.T) {
	ti := &testIndexer{
		t:        t,
		eventCh:  make(chan testIndexerEvent),
		statusCh: make(chan testIndexerStatus),
	}
	var blockchain *BlockChain
	doneCh := make(chan struct{})
	expDone := func() {
		select {
		case <-doneCh:
		case <-time.After(time.Second * 5):
			t.Fatalf("Expected chain operation done but not finished yet")
		}
	}
	testSuspendHookCh := make(chan struct{}, 1)
	run := func(fn func()) {
		go func() {
			fn()
			doneCh <- struct{}{}
		}()
	}
	insert := func(chain []*types.Block) {
		run(func() {
			if i, err := blockchain.InsertChain(chain); err != nil {
				t.Fatalf("failed to insert chain[%d]: %v", i, err)
			}
		})
	}
	waitSuspend := func() {
		select {
		case <-testSuspendHookCh:
		case <-time.After(time.Second * 5):
			t.Fatalf("Expected index server suspend but suspended state not reached")
		}
	}

	gspec := &Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	db := rawdb.NewMemoryDatabase()
	blockchain, _ = NewBlockChain(db, gspec, ethash.NewFaker(), DefaultConfig())
	chain := []*types.Block{gspec.ToBlock()}
	blocks, _ := GenerateChain(gspec.Config, chain[0], ethash.NewFaker(), db, 110, func(i int, gen *BlockGen) {})
	chain = append(chain, blocks...)

	run(func() {
		blockchain.RegisterIndexer(ti, "")
		blockchain.indexServers.servers[0].testSuspendHookCh = testSuspendHookCh
	})
	ti.expEvent(testIndexerEvent{ev: "SetHistoryCutoff", blockNumber: 0})
	ti.expEvent(testIndexerEvent{ev: "SetFinalized", blockNumber: 0})
	ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: 0, blockHash: chain[0].Hash()})
	ti.status(testIndexerStatus{ready: true})
	expDone()

	insert(chain[1:101])
	waitSuspend()
	ti.expEvent(testIndexerEvent{ev: "Suspended"})
	for i := uint64(1); i <= 100; i++ {
		ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: i, blockHash: chain[i].Hash()})
		ti.status(testIndexerStatus{ready: true})
	}
	expDone()

	run(blockchain.Stop)
	ti.expEvent(testIndexerEvent{ev: "Stop"})
	expDone()

	blockchain, _ = NewBlockChain(db, gspec, ethash.NewFaker(), DefaultConfig())
	run(func() {
		blockchain.RegisterIndexer(ti, "")
		blockchain.indexServers.servers[0].testSuspendHookCh = testSuspendHookCh
	})
	ti.expEvent(testIndexerEvent{ev: "SetHistoryCutoff", blockNumber: 0})
	ti.expEvent(testIndexerEvent{ev: "SetFinalized", blockNumber: 0})
	ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: 100, blockHash: chain[100].Hash()})
	ti.status(testIndexerStatus{ready: true, needBlocks: common.NewRange[uint64](0, 100)})
	expDone()
	// request entire chain as historical range, add a new block in the middle and check suspend mechanism
	for i := uint64(0); i <= 49; i++ {
		ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: i, blockHash: chain[i].Hash()})
		ti.status(testIndexerStatus{ready: true, needBlocks: common.NewRange[uint64](i+1, 99-i)})
	}
	ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: 50, blockHash: chain[50].Hash()})
	insert(chain[101:102])
	waitSuspend()
	ti.status(testIndexerStatus{ready: true, needBlocks: common.NewRange[uint64](51, 49)})
	ti.expEvent(testIndexerEvent{ev: "Suspended"})
	ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: 101, blockHash: chain[101].Hash()})
	ti.status(testIndexerStatus{ready: true, needBlocks: common.NewRange[uint64](51, 50)})
	expDone()
	for i := uint64(51); i <= 100; i++ {
		ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: i, blockHash: chain[i].Hash()})
		ti.status(testIndexerStatus{ready: true, needBlocks: common.NewRange[uint64](i+1, 100-i)})
	}

	run(func() {
		blockchain.SetHead(80)
	})
	ti.expEvent(testIndexerEvent{ev: "Revert", blockNumber: 80})
	expDone()
	chain = chain[:81]
	blocks, _ = GenerateChain(gspec.Config, chain[80], ethash.NewFaker(), db, 45, func(i int, gen *BlockGen) {})
	chain = append(chain, blocks...)
	insert(chain[81:121])
	waitSuspend()
	ti.expEvent(testIndexerEvent{ev: "Suspended"})
	for i := uint64(81); i <= 120; i++ {
		ti.expEvent(testIndexerEvent{ev: "AddBlockData", blockNumber: i, blockHash: chain[i].Hash()})
		ti.status(testIndexerStatus{ready: true})
	}
	expDone()

	run(blockchain.Stop)
	ti.expEvent(testIndexerEvent{ev: "Stop"})
	expDone()
}

type testIndexer struct {
	t        *testing.T
	eventCh  chan testIndexerEvent
	statusCh chan testIndexerStatus
}

type testIndexerEvent struct {
	ev          string
	blockNumber uint64
	blockHash   common.Hash
}

type testIndexerStatus struct {
	ready      bool
	needBlocks common.Range[uint64]
}

func (ti *testIndexer) expEvent(exp testIndexerEvent) {
	var got testIndexerEvent
	select {
	case got = <-ti.eventCh:
	case <-time.After(time.Second * 5):
	}
	if got != exp {
		ti.t.Fatalf("Wrong indexer event received (expected: %v, got: %v)", exp, got)
	}
}

func (ti *testIndexer) status(status testIndexerStatus) {
	ti.statusCh <- status
}

func (ti *testIndexer) AddBlockData(header *types.Header, receipts types.Receipts) (ready bool, needBlocks common.Range[uint64]) {
	ti.eventCh <- testIndexerEvent{ev: "AddBlockData", blockNumber: header.Number.Uint64(), blockHash: header.Hash()}
	status := <-ti.statusCh
	return status.ready, status.needBlocks
}

func (ti *testIndexer) Revert(blockNumber uint64) {
	ti.eventCh <- testIndexerEvent{ev: "Revert", blockNumber: blockNumber}
}

func (ti *testIndexer) Status() (ready bool, needBlocks common.Range[uint64]) {
	ti.eventCh <- testIndexerEvent{ev: "Status"}
	status := <-ti.statusCh
	return status.ready, status.needBlocks
}

func (ti *testIndexer) SetHistoryCutoff(blockNumber uint64) {
	ti.eventCh <- testIndexerEvent{ev: "SetHistoryCutoff", blockNumber: blockNumber}
}

func (ti *testIndexer) SetFinalized(blockNumber uint64) {
	ti.eventCh <- testIndexerEvent{ev: "SetFinalized", blockNumber: blockNumber}
}

func (ti *testIndexer) Suspended() {
	ti.eventCh <- testIndexerEvent{ev: "Suspended"}
}

func (ti *testIndexer) Stop() {
	ti.eventCh <- testIndexerEvent{ev: "Stop"}
}
