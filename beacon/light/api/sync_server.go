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

package api

import (
	"math"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
)

const (
	maxHeadLength   = 4
	minFailureDelay = time.Millisecond * 100
	maxFailureDelay = time.Minute
)

type SyncServer struct {
	api         *BeaconLightApi
	unsubscribe func()

	failureDelayUntil mclock.AbsTime
	failureDelay      float64
}

func NewSyncServer(api *BeaconLightApi) *SyncServer {
	return &SyncServer{api: api}
}

func (s *SyncServer) SubscribeHeads(newHead func(uint64, common.Hash), newSignedHead func(signedHead types.SignedHeader)) {
	s.unsubscribe = s.api.StartHeadListener(newHead, newSignedHead, func(err error) {
		log.Warn("Head event stream error", "err", err)
	})
}

// Note: UnsubscribeHeads should not be called concurrently with SubscribeHeads
func (s *SyncServer) UnsubscribeHeads() {
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
}

func (s *SyncServer) DelayUntil() mclock.AbsTime {
	return s.failureDelayUntil
}

func (s *SyncServer) Fail(desc string) {
	s.failureDelay *= 2
	now := mclock.Now()
	if now > s.failureDelayUntil {
		s.failureDelay *= math.Pow(2, -float64(now-s.failureDelayUntil)/float64(maxFailureDelay))
	}
	if s.failureDelay < float64(minFailureDelay) {
		s.failureDelay = float64(minFailureDelay)
	}
	s.failureDelayUntil = now + mclock.AbsTime(s.failureDelay)
	log.Warn("API endpoint failure", "URL", s.api.url, "error", desc)

}

func (s *SyncServer) RequestBootstrap(checkpointHash common.Hash, response func(*light.CheckpointData, error)) {
	go response(s.api.GetCheckpointData(checkpointHash))
}

func (s *SyncServer) RequestUpdates(first, count uint64, response func([]*types.LightClientUpdate, []*types.SerializedSyncCommittee, error)) {
	go response(s.api.GetBestUpdatesAndCommittees(first, count))
}

func (s *SyncServer) RequestBeaconBlock(blockRoot common.Hash, response func(*capella.BeaconBlock, error)) {
	go response(s.api.GetBeaconBlock(blockRoot))
}
