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

package catalyst

import (
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// Blsync tracks the head of the beacon chain through the beacon light client
// and drives the local node via ConsensusAPI.
type Blsync struct {
	eth     *eth.Ethereum
	engine  *ConsensusAPI
	client  *light.LightClient
	headCh  chan light.ChainHeadEvent
	headSub event.Subscription

	quitCh chan struct{}
}

// NewBlsync creates a new beacon light syncer.
func NewBlsync(eth *eth.Ethereum) *Blsync {
	engine := newConsensusAPIWithoutHeartbeat(eth)
	return &Blsync{
		eth:    eth,
		engine: engine,
		client: eth.Beacon(),
		headCh: make(chan light.ChainHeadEvent, 128),
		quitCh: make(chan struct{}),
	}
}

// Start starts underlying beacon light client and the sync logic for driving
// the local node.
func (b *Blsync) Start() error {
	log.Info("Blsync started")
	b.headSub = b.client.SubscribeChainHeadEvent(b.headCh)
	go b.client.Start()

	for {
		select {
		case <-b.quitCh:
			return nil
		case head := <-b.headCh:
			if _, err := b.engine.NewPayloadV2(*head.Data); err != nil {
				log.Error("failed to send new payload", "err", err)
				continue
			}
			update := engine.ForkchoiceStateV1{
				HeadBlockHash:      head.Data.BlockHash,
				SafeBlockHash:      common.Hash{},
				FinalizedBlockHash: common.Hash{},
			}
			if _, err := b.engine.ForkchoiceUpdatedV1(update, nil); err != nil {
				log.Error("failed to send forkchoice updated", "err", err)
				continue
			}
		}
	}
}

// Stop signals to the light client and syncer to exit.
func (b *Blsync) Stop() error {
	b.client.Stop()
	close(b.quitCh)
	return nil
}
