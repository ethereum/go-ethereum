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

package simulated_beacon

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// withdrawals implements a FIFO queue which holds withdrawals that are pending inclusion
type withdrawals struct {
	pending []*types.Withdrawal
	mu      sync.Mutex
}

// pop removes up to 10 withdrawals from the queue
func (w *withdrawals) pop() []*types.Withdrawal {
	w.mu.Lock()
	defer w.mu.Unlock()

	var popCount int
	if len(w.pending) >= 10 {
		popCount = 10
	} else {
		popCount = len(w.pending)
	}

	popped := make([]*types.Withdrawal, popCount)
	copy(popped[:], w.pending[0:popCount])
	w.pending = append([]*types.Withdrawal{}, w.pending[popCount:]...)
	return popped
}

// add adds a withdrawal to the queue
func (w *withdrawals) add(withdrawal *types.Withdrawal) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pending = append(w.pending, withdrawal)
	return nil
}

type SimulatedBeacon struct {
	shutdownCh   chan struct{}
	eth          *eth.Ethereum
	period       uint64
	withdrawals  withdrawals
	feeRecipient common.Address
	// mu gates concurrent access to the feeRecipient
	mu sync.Mutex
}

func NewSimulatedBeacon(eth *eth.Ethereum) *SimulatedBeacon {
	chainConfig := eth.APIBackend.ChainConfig()
	if chainConfig.Dev == nil {
		log.Crit("incompatible pre-existing chain configuration")
	}

	return &SimulatedBeacon{
		eth:          eth,
		period:       chainConfig.Dev.Period,
		withdrawals:  withdrawals{[]*types.Withdrawal{}, sync.Mutex{}},
		feeRecipient: common.Address{},
	}
}

func (c *SimulatedBeacon) setFeeRecipient(feeRecipient *common.Address) {
	c.mu.Lock()
	c.feeRecipient = *feeRecipient
	c.mu.Unlock()
}

func (c *SimulatedBeacon) getFeeRecipient() common.Address {
	c.mu.Lock()
	feeRecipient := c.feeRecipient
	c.mu.Unlock()

	return feeRecipient
}

// Start invokes the SimulatedBeacon life-cycle function in a goroutine
func (c *SimulatedBeacon) Start() error {
	c.shutdownCh = make(chan struct{})
	go c.loop()
	return nil
}

// Stop halts the SimulatedBeacon service
func (c *SimulatedBeacon) Stop() error {
	close(c.shutdownCh)
	return nil
}

// loop manages the lifecycle of the SimulatedBeacon.
// it drives block production, taking the role of a CL client and interacting with Geth via public engine/eth APIs
func (c *SimulatedBeacon) loop() {
	var (
		ticker = time.NewTicker(time.Millisecond * 100)

		header             = c.eth.BlockChain().CurrentHeader()
		lastBlockTime      = header.Time
		engineAPI          = catalyst.NewConsensusAPI(c.eth)
		curForkchoiceState = engine.ForkchoiceStateV1{
			HeadBlockHash:      header.Hash(),
			SafeBlockHash:      header.Hash(),
			FinalizedBlockHash: header.Hash(),
		}
		buildWaitTime      = time.Millisecond * 100
		pendingWithdrawals []*types.Withdrawal
	)

	// if genesis block, send forkchoiceUpdated to trigger transition to PoS
	if header.Number.Sign() == 0 {
		if _, err := engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, nil); err != nil {
			log.Crit("failed to initiate PoS transition for genesis via Forkchoiceupdated", "err", err)
		}
	}

	for {
		select {
		case <-c.shutdownCh:
			break
		case curTime := <-ticker.C:
			if c.period != 0 && uint64(curTime.Unix()) <= lastBlockTime+c.period {
				// In period=N, mine every N seconds
				continue
			}
			pendingWithdrawals = c.withdrawals.pop()
			if c.period == 0 {
				// In period=0, mine whenever we have stuff to mine
				if pendingTxs, _ := c.eth.APIBackend.TxPool().Stats(); pendingTxs == 0 && len(pendingWithdrawals) == 0 {
					continue
				}
			}
			// Looks like it's time to build us a block!
			tstamp := uint64(curTime.Unix())
			if tstamp <= lastBlockTime {
				tstamp = lastBlockTime + 1
			}
			fcState, err := engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, &engine.PayloadAttributes{
				Timestamp:             tstamp,
				Random:                common.Hash{}, // TODO: make this configurable?
				SuggestedFeeRecipient: c.getFeeRecipient(),
				Withdrawals:           pendingWithdrawals,
			})
			if err != nil {
				log.Crit("failed to trigger block building via forkchoiceupdated", "err", err)
			}

			time.Sleep(buildWaitTime) // Give it some time to build
			var payload *engine.ExecutableData
			if payload, err = engineAPI.GetPayloadV1(*fcState.PayloadID); err != nil {
				log.Crit("error retrieving payload", "err", err)
			}
			// Don't accept empty blocks if perdiod == 0
			if len(payload.Transactions) == 0 && len(payload.Withdrawals) == 0 && c.period == 0 {
				// If the payload is empty, despite there being pending transactions,
				// that indicates that we need to give it more time to build the block.
				if buildWaitTime < 10*time.Second {
					buildWaitTime += buildWaitTime
				}
				// If we hit here, we will lose the pendingWithdrawals, We either
				// need to undo the 'pop', or we need to remember the popped withdrawals
				// locally.
				continue
			}
			buildWaitTime = 100 * time.Millisecond // Set back (might have been bumped)
			// mark the payload as canon
			if _, err = engineAPI.NewPayloadV2(*payload); err != nil {
				log.Crit("failed to mark payload as canonical", "err", err)
			}
			curForkchoiceState = engine.ForkchoiceStateV1{
				HeadBlockHash:      payload.BlockHash,
				SafeBlockHash:      payload.BlockHash,
				FinalizedBlockHash: payload.BlockHash,
			}
			// mark the block containing the payload as canonical
			if _, err = engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, nil); err != nil {
				log.Crit("failed to mark block as canonical", "err", err)
			}
			lastBlockTime = payload.Timestamp
		}
	}
}

func RegisterAPIs(stack *node.Node, c *SimulatedBeacon) {
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "dev",
			Service:   &API{c},
			Version:   "1.0",
		},
	})
}
