// Copyright 2022 The go-ethereum Authors
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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

// FullSyncTester is an auxiliary service that allows Geth to perform full sync
// alone without consensus-layer attached. Users must specify a valid block as
// the sync target. This tester can be applied to different networks, no matter
// it's pre-merge or post-merge, but only for full-sync.
type FullSyncTester struct {
	api    *ConsensusAPI
	block  *types.Block
	closed chan struct{}
	wg     sync.WaitGroup
}

// RegisterFullSyncTester registers the full-sync tester service into the node
// stack for launching and stopping the service controlled by node.
func RegisterFullSyncTester(stack *node.Node, backend *eth.Ethereum, block *types.Block) (*FullSyncTester, error) {
	cl := &FullSyncTester{
		api:    NewConsensusAPI(backend),
		block:  block,
		closed: make(chan struct{}),
	}
	stack.RegisterLifecycle(cl)
	return cl, nil
}

// Start launches the full-sync tester by spinning up a background thread
// for keeping firing NewPayload-UpdateForkChoice combos with the provided
// target block, it may or may not trigger the beacon sync depends on if
// there are protocol peers connected.
func (tester *FullSyncTester) Start() error {
	tester.wg.Add(1)
	go func() {
		defer tester.wg.Done()

		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Don't bother downloader in case it's already syncing.
				if tester.api.eth.Downloader().Synchronising() {
					continue
				}
				// Short circuit in case the target block is already stored
				// locally.
				if tester.api.eth.BlockChain().HasBlock(tester.block.Hash(), tester.block.NumberU64()) {
					log.Info("Full-sync target reached", "number", tester.block.NumberU64(), "hash", tester.block.Hash())
					return
				}
				// Shoot out consensus events in order to trigger syncing.
				data := beacon.BlockToExecutableData(tester.block)
				tester.api.NewPayloadV1(*data)
				tester.api.ForkchoiceUpdatedV1(beacon.ForkchoiceStateV1{
					HeadBlockHash:      tester.block.Hash(),
					SafeBlockHash:      tester.block.Hash(),
					FinalizedBlockHash: tester.block.Hash(),
				}, nil)
			case <-tester.closed:
				return
			}
		}
	}()
	return nil
}

// Stop stops the full-sync tester to stop all background activities.
// This function can only be called for one time.
func (tester *FullSyncTester) Stop() error {
	close(tester.closed)
	tester.wg.Wait()
	return nil
}
