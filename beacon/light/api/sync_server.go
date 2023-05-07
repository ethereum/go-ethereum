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
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
)

const (
	maxHeadLength = 4
)

type SyncServer struct {
	api         *BeaconLightApi
	unsubscribe func()
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

func (s *SyncServer) DelayUntil() mclock.AbsTime { return 0 } //TODO

func (s *SyncServer) Fail(desc string) {
	log.Warn("API endpoint failure", "URL", s.api.url, "error", desc)
}

func (s *SyncServer) RequestBootstrap(checkpointHash common.Hash, response func(*light.CheckpointData)) {
	go func() {
		if checkpoint, err := s.api.GetCheckpointData(checkpointHash); err == nil {
			response(checkpoint)
		} else {
			response(nil)
		}
	}()
}

func (s *SyncServer) RequestUpdates(first, count uint64, response func([]*types.LightClientUpdate, []*types.SerializedCommittee)) {
	go func() {
		if updates, committees, err := s.api.GetBestUpdatesAndCommittees(first, count); err == nil {
			response(updates, committees)
		} else {
			response(nil, nil)
		}
	}()
}

func (s *SyncServer) RequestBeaconBlock(blockRoot common.Hash, response func(*capella.BeaconBlock)) {
	go func() {
		if block, err := s.api.GetBeaconBlock(blockRoot); err == nil {
			response(block)
		} else {
			response(nil)
		}
	}()
}

func (s *SyncServer) RequestBeaconHeader(blockRoot common.Hash, response func(*types.Header)) {
	go func() {
		if header, err := s.api.GetHeader(blockRoot); err == nil {
			response(&header)
		} else {
			response(nil)
		}
	}()
}

func (s *SyncServer) RequestBeaconState(stateRoot common.Hash, format merkle.CompactProofFormat, response func(*merkle.MultiProof)) {
	go func() {
		if proof, err := s.api.GetStateProof(stateRoot, format); err == nil {
			response(&proof)
		} else {
			response(nil)
		}
	}()
}
