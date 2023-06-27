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

package catalyst

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// withdrawalQueue implements a FIFO queue which holds withdrawals that are pending inclusion
type withdrawalQueue struct {
	// pending holds withdrawals that will be included in a future block
	pending []*types.Withdrawal
	mu      sync.Mutex
}

// queued returns withdrawals which are pending inclusion in the next block
func (w *withdrawalQueue) queued() []*types.Withdrawal {
	w.mu.Lock()
	defer w.mu.Unlock()

	var queueCount int
	if len(w.pending) >= 10 {
		queueCount = 10
	} else {
		queueCount = len(w.pending)
	}

	p := make([]*types.Withdrawal, queueCount)
	copy(p, w.pending[0:queueCount])
	return p
}

// add queues a withdrawal for future inclusion
func (w *withdrawalQueue) add(withdrawal *types.Withdrawal) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pending = append(w.pending, withdrawal)
	return nil
}

// clearQueued shifts up to 10 withdrawals out of the queue
func (w *withdrawalQueue) clearQueued(count int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if count > 10 {
		count = 10
	}
	if len(w.pending) < count {
		count = len(w.pending)
	}

	w.pending = append([]*types.Withdrawal{}, w.pending[count:]...)
}

type SimulatedBeacon struct {
	shutdownCh  chan struct{}
	eth         *eth.Ethereum
	period      uint64
	withdrawals withdrawalQueue

	feeRecipient     common.Address
	feeRecipientLock sync.Mutex // lock gates concurrent access to the feeRecipient

	engineAPI          *ConsensusAPI
	curForkchoiceState engine.ForkchoiceStateV1
	buildWaitTime      time.Duration
	lastBlockTime      uint64
}

func NewSimulatedBeacon(eth *eth.Ethereum) (*SimulatedBeacon, error) {
	chainConfig := eth.APIBackend.ChainConfig()
	if chainConfig.Dev == nil {
		return nil, errors.New("incompatible pre-existing chain configuration")
	}
	header := eth.BlockChain().CurrentHeader()
	current := engine.ForkchoiceStateV1{
		HeadBlockHash:      header.Hash(),
		SafeBlockHash:      header.Hash(),
		FinalizedBlockHash: header.Hash(),
	}
	engineAPI := NewConsensusAPI(eth)

	// if genesis block, send forkchoiceUpdated to trigger transition to PoS
	if header.Number.Sign() == 0 {
		if _, err := engineAPI.ForkchoiceUpdatedV2(current, nil); err != nil {
			return nil, err
		}
	}
	return &SimulatedBeacon{
		eth:                eth,
		period:             chainConfig.Dev.Period,
		shutdownCh:         make(chan struct{}),
		buildWaitTime:      time.Millisecond * 100,
		engineAPI:          engineAPI,
		lastBlockTime:      header.Time,
		curForkchoiceState: current,
	}, nil
}

func (c *SimulatedBeacon) setFeeRecipient(feeRecipient common.Address) {
	c.feeRecipientLock.Lock()
	c.feeRecipient = feeRecipient
	c.feeRecipientLock.Unlock()
}

// Start invokes the SimulatedBeacon life-cycle function in a goroutine
func (c *SimulatedBeacon) Start() error {
	go c.loop()
	return nil
}

// Stop halts the SimulatedBeacon service
func (c *SimulatedBeacon) Stop() error {
	close(c.shutdownCh)
	return nil
}

// beginSealing instructs the client to begin building a payload.
func (c *SimulatedBeacon) beginSealing() (engine.ForkChoiceResponse, error) {
	tstamp := uint64(time.Now().Unix())
	if tstamp <= c.lastBlockTime {
		tstamp = c.lastBlockTime + 1
	}
	c.feeRecipientLock.Lock()
	feeRecipient := c.feeRecipient
	c.feeRecipientLock.Unlock()

	return c.engineAPI.ForkchoiceUpdatedV2(c.curForkchoiceState, &engine.PayloadAttributes{
		Timestamp:             tstamp,
		SuggestedFeeRecipient: feeRecipient,
		Withdrawals:           c.withdrawals.queued(),
	})
}

// finalizeSealing retrieves a completed payload and marks it as canonical if it contains transactions or withdrawals.
func (c *SimulatedBeacon) finalizeSealing(id *engine.PayloadID, onDemand bool) error {
	payload, err := c.engineAPI.GetPayloadV1(*id)
	if err != nil {
		return fmt.Errorf("error retrieving payload: %v", err)
	}

	if onDemand && len(payload.Transactions) == 0 && len(payload.Withdrawals) == 0 {
		// If the payload is empty, despite there being pending transactions,
		// that indicates that we need to give it more time to build the block.
		if c.buildWaitTime < 10*time.Second {
			c.buildWaitTime += c.buildWaitTime
		}
		return nil
	}
	c.buildWaitTime = 100 * time.Millisecond // Reset it
	// mark the payload as canon
	if _, err = c.engineAPI.NewPayloadV2(*payload); err != nil {
		return fmt.Errorf("failed to mark payload as canonical: %v", err)
	}
	c.curForkchoiceState = engine.ForkchoiceStateV1{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: payload.BlockHash,
	}
	// mark the block containing the payload as canonical
	if _, err = c.engineAPI.ForkchoiceUpdatedV2(c.curForkchoiceState, nil); err != nil {
		return fmt.Errorf("failed to mark block as canonical: %v", err)
	}
	c.withdrawals.clearQueued(len(payload.Withdrawals))
	c.lastBlockTime = payload.Timestamp
	return nil
}

// loop manages the lifecycle of the SimulatedBeacon.
// It drives block production, taking the role of a CL client and interacting with Geth via public engine/eth APIs.
func (c *SimulatedBeacon) loop() {
	ticker := time.NewTimer(time.Second * time.Duration(c.period))
	var fcId *engine.PayloadID
	if fc, err := c.beginSealing(); err != nil {
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
				if pendingTxs, _ := c.eth.APIBackend.TxPool().Stats(); pendingTxs == 0 && len(c.withdrawals.queued()) == 0 {
					ticker.Reset(c.buildWaitTime)
					continue
				}
			}
			if err := c.finalizeSealing(fcId, onDemand); err != nil {
				log.Error("Error collecting sealing-work", "err", err)
				return
			}
			if fc, err := c.beginSealing(); err != nil {
				log.Error("Error starting sealing-work", "err", err)
				return
			} else {
				fcId = fc.PayloadID
			}
			if !onDemand {
				ticker.Reset(time.Second * time.Duration(c.period))
			} else {
				ticker.Reset(c.buildWaitTime)
			}
		}
	}
}

func RegisterSimulatedBeaconAPIs(stack *node.Node, sim *SimulatedBeacon) {
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "dev",
			Service:   &api{sim},
			Version:   "1.0",
		},
	})
}
