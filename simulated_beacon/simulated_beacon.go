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
	"context"
	"fmt"
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
	next    uint64
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

	if withdrawal.Index < w.next {
		return fmt.Errorf("withdrawal has index (%d) less than or equal to latest received withdrawal index (%d)", withdrawal.Index, w.next-1)
	}
	w.next = withdrawal.Index + 1
	w.pending = append(w.pending, withdrawal)
	return nil
}

type SimulatedBeacon struct {
	ctx          context.Context
	cancel       context.CancelFunc
	eth          *eth.Ethereum
	period       time.Duration
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
		period:       time.Duration(chainConfig.Dev.Period) * time.Second,
		withdrawals:  withdrawals{[]*types.Withdrawal{}, 0, sync.Mutex{}},
		feeRecipient: common.Address{},
	}
}

func (c *SimulatedBeacon) setFeeRecipient(feeRecipient *common.Address) {
	c.mu.Lock()
	c.feeRecipient = *feeRecipient
	c.mu.Unlock()
}

// Start invokes the SimulatedBeacon life-cycle function in a goroutine
func (c *SimulatedBeacon) Start() error {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	go c.loop()
	return nil
}

// Stop halts the SimulatedBeacon service
func (c *SimulatedBeacon) Stop() error {
	c.cancel()
	return nil
}

// loop manages the lifecycle of the SimulatedBeacon.
// it drives block production, taking the role of a CL client and interacting with Geth via public engine/eth APIs
func (c *SimulatedBeacon) loop() {
	var (
		ticker             = time.NewTicker(time.Millisecond * 100)
		lastBlockTime      = time.Now()
		engineAPI          = catalyst.NewConsensusAPI(c.eth)
		header             = c.eth.BlockChain().CurrentHeader()
		curForkchoiceState = engine.ForkchoiceStateV1{
			HeadBlockHash:      header.Hash(),
			SafeBlockHash:      header.Hash(),
			FinalizedBlockHash: header.Hash(),
		}
	)

	// if genesis block, send forkchoiceUpdated to trigger transition to PoS
	if header.Number.BitLen() == 0 {
		if _, err := engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, nil); err != nil {
			log.Crit("failed to initiate PoS transition for genesis via Forkchoiceupdated", "err", err)
		}
	}

	for {
		select {
		case <-c.ctx.Done():
			break
		case curTime := <-ticker.C:
			if curTime.Unix() > lastBlockTime.Add(c.period).Unix() {
				c.mu.Lock()
				feeRecipient := c.feeRecipient
				pendingWithdrawals := c.withdrawals.pop()
				c.mu.Unlock()

				payloadAttr := &engine.PayloadAttributes{
					Timestamp:             uint64(curTime.Unix()),
					Random:                common.Hash{}, // TODO: make this configurable?
					SuggestedFeeRecipient: feeRecipient,
					Withdrawals:           pendingWithdrawals,
				}

				// trigger block building
				fcState, err := engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, payloadAttr)
				if err != nil {
					log.Crit("failed to trigger block building via forkchoiceupdated", "err", err)
				}

				var payload *engine.ExecutableData

				var (
					restartPayloadBuilding bool
					// building a payload times out after SECONDS_PER_SLOT (12s on mainnet).
					// trigger building a new payload if this amount of time elapses w/o any transactions or withdrawals
					// having been received.
					payloadTimeout = time.NewTimer(12 * time.Second)
					// interval to poll the pending state to detect if transactions have arrived, and proceed if they have
					// (or if there are pending withdrawals to include)
					buildTicker = time.NewTicker(100 * time.Millisecond)
				)
				for {
					select {
					case <-buildTicker.C:
						pendingHeader, err := c.eth.APIBackend.HeaderByNumber(context.Background(), rpc.PendingBlockNumber)
						if err != nil {
							log.Crit("failed to get pending block header", "err", err)
						}
						// don't build a block if we don't have pending txs or withdrawals
						if pendingHeader.TxHash == types.EmptyTxsHash && len(payloadAttr.Withdrawals) == 0 {
							continue
						}
						payload, err = engineAPI.GetPayloadV1(*fcState.PayloadID)
						if err != nil {
							log.Crit("error retrieving payload", "err", err)
						}
						// Don't build a block if it doesn't contain transactions or withdrawals.
						// Somehow, txs can arrive, be detected by this routine, but be missed by the miner payload builder.
						// So this last clause prevents empty blocks from being built.
						if len(payload.Transactions) == 0 && len(payloadAttr.Withdrawals) == 0 {
							restartPayloadBuilding = true
						}
					case <-payloadTimeout.C:
						restartPayloadBuilding = true
					case <-c.ctx.Done():
						return
					}
					break
				}
				if restartPayloadBuilding {
					continue
				}
				// mark the payload as the one we have chosen
				if _, err = engineAPI.NewPayloadV2(*payload); err != nil {
					log.Crit("failed to mark payload as canonical", "err", err)
				}

				newForkchoiceState := &engine.ForkchoiceStateV1{
					HeadBlockHash:      payload.BlockHash,
					SafeBlockHash:      payload.BlockHash,
					FinalizedBlockHash: payload.BlockHash,
				}
				// mark the block containing the payload as canonical
				_, err = engineAPI.ForkchoiceUpdatedV2(*newForkchoiceState, nil)
				if err != nil {
					log.Crit("failed to mark block as canonical", "err", err)
				}
				lastBlockTime = time.Unix(int64(payload.Timestamp), 0)
				curForkchoiceState = *newForkchoiceState
			}
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
