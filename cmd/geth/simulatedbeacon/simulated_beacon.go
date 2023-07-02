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

package simulatedbeacon

import (
	"sync"
	"time"

	"fmt"
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

// length returns the length of the pending withdrawals.
func (w *withdrawals) length() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.pending)
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
		ticker             = time.NewTimer(time.Second * time.Duration(c.period))
		buildWaitTime      = time.Millisecond * 100
		header             = c.eth.BlockChain().CurrentHeader()
		lastBlockTime      = header.Time
		engineAPI          = catalyst.NewConsensusAPI(c.eth)
		curForkchoiceState = engine.ForkchoiceStateV1{
			HeadBlockHash:      header.Hash(),
			SafeBlockHash:      header.Hash(),
			FinalizedBlockHash: header.Hash(),
		}
	)
	// if genesis block, send forkchoiceUpdated to trigger transition to PoS
	if header.Number.Sign() == 0 {
		if _, err := engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, nil); err != nil {
			log.Crit("failed to initiate PoS transition for genesis via Forkchoiceupdated", "err", err)
		}
	}

	beginSealing := func() (engine.ForkChoiceResponse, error) {
		tstamp := uint64(time.Now().Unix())
		if tstamp <= lastBlockTime {
			tstamp = lastBlockTime + 1
		}
		return engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, &engine.PayloadAttributes{
			Timestamp:             tstamp,
			SuggestedFeeRecipient: c.getFeeRecipient(),
			Withdrawals:           c.withdrawals.pop(),
		})
	}
	finalizeSealing := func(id *engine.PayloadID, onDemand bool) error {
		payload, err := engineAPI.GetPayloadV1(*id)
		if err != nil {
			return fmt.Errorf("error retrieving payload: %v", err)
		}

		if onDemand && len(payload.Transactions) == 0 && len(payload.Withdrawals) == 0 {
			// If the payload is empty, despite there being pending transactions,
			// that indicates that we need to give it more time to build the block.
			if buildWaitTime < 10*time.Second {
				buildWaitTime += buildWaitTime
			}
			// TODO: don't lose withdrawals
			return nil
		}
		buildWaitTime = 100 * time.Millisecond // Reset it
		// mark the payload as canon
		if _, err = engineAPI.NewPayloadV2(*payload); err != nil {
			return fmt.Errorf("failed to mark payload as canonical: %v", err)
		}
		curForkchoiceState = engine.ForkchoiceStateV1{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      payload.BlockHash,
			FinalizedBlockHash: payload.BlockHash,
		}
		// mark the block containing the payload as canonical
		if _, err = engineAPI.ForkchoiceUpdatedV2(curForkchoiceState, nil); err != nil {
			return fmt.Errorf("failed to mark block as canonical: %v", err)
		}
		lastBlockTime = payload.Timestamp
		return nil
	}
	var fcId *engine.PayloadID
	if fc, err := beginSealing(); err != nil {
		log.Error("Error starting sealing-work", "err", err)
		return
	} else {
		fcId = fc.PayloadID
	}
	onDemand := (c.period == 0)
	for {
		select {
		case <-c.shutdownCh:
			return
		case <-ticker.C:
			if onDemand {
				// Do nothing as long as blocks are empty
				if pendingTxs, _ := c.eth.APIBackend.TxPool().Stats(); pendingTxs == 0 && c.withdrawals.length() == 0 {
					ticker.Reset(buildWaitTime)
					continue
				}
			}
			if err := finalizeSealing(fcId, onDemand); err != nil {
				log.Error("Error collecting sealing-work", "err", err)
				return
			}
			if fc, err := beginSealing(); err != nil {
				log.Error("Error starting sealing-work", "err", err)
				return
			} else {
				fcId = fc.PayloadID
			}
			if !onDemand {
				ticker.Reset(time.Second * time.Duration(c.period))
			} else {
				ticker.Reset(buildWaitTime)
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
