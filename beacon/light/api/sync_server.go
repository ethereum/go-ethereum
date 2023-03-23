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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
)

const (
	maxHeadLength = 4
)

type SyncServer struct {
	api  *BeaconLightApi
	lock sync.RWMutex

	unsubscribe                  func()
	canRequestBootstrap          bool
	firstUpdate, afterLastUpdate uint64
	firstState                   uint64 //TODO ...
}

func NewSyncServer(api *BeaconLightApi) *SyncServer {
	return &SyncServer{
		api:                 api,
		canRequestBootstrap: true,
	}
}

func (s *SyncServer) SubscribeHeads(newHead func(uint64, common.Hash), newSignedHead func(signedHead types.SignedHead)) {
	s.lock.Lock()
	s.unsubscribe = s.api.StartHeadListener(newHead, func(signedHead types.SignedHead) {
		s.lock.Lock()
		s.afterLastUpdate = types.PeriodOfSlot(signedHead.Header.Slot + 256)
		s.lock.Unlock()
		newSignedHead(signedHead)
	}, func(err error) {
		log.Warn("Head event stream error", "err", err)
	})
	s.lock.Unlock()
}

func (s *SyncServer) UnsubscribeHeads() {
	s.lock.Lock()
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
	s.lock.Unlock()
}

func (s *SyncServer) Delay() time.Duration { return 0 } //TODO

func (s *SyncServer) Fail(desc string) {
	log.Warn("API endpoint failure", "URL", s.api.url, "error", desc)
}

func (s *SyncServer) CanRequestBootstrap() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.canRequestBootstrap
}

func (s *SyncServer) RequestBootstrap(checkpointHash common.Hash, response func(*light.CheckpointData)) {
	go func() {
		if checkpoint, err := s.api.GetCheckpointData(checkpointHash); err == nil {
			response(checkpoint)
		} else {
			s.lock.Lock()
			s.canRequestBootstrap = false
			s.lock.Unlock()
			response(nil)
		}
	}()
}

func (s *SyncServer) UpdateRange() types.PeriodRange {
	s.lock.RLock()
	defer s.lock.RUnlock()

	r := types.PeriodRange{First: s.firstUpdate, AfterLast: s.afterLastUpdate}
	if !r.IsEmpty() {
		return r
	}
	return types.PeriodRange{}
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

func (s *SyncServer) BeaconStateTail() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.firstState
}

func (s *SyncServer) RequestBeaconState(slot uint64, stateRoot common.Hash, format merkle.ProofFormat, response func(*merkle.MultiProof)) {
	go func() {
		if proof, err := s.api.GetStateProof(stateRoot, format); err == nil {
			response(&proof)
		} else {
			s.lock.Lock()
			if slot >= s.firstState {
				s.firstState = slot + 1
			}
			s.lock.Unlock()
			response(nil)
		}
	}()
}
