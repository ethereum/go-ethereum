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

package sync

import (
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const maxUpdateRequest = 8

type checkpointInitServer interface {
	request.RequestServer
	CanRequestBootstrap() bool
	RequestBootstrap(checkpointHash common.Hash, response func(*light.CheckpointData))
}

type CheckpointInit struct {
	request.SingleLock
	lock           sync.Mutex
	chain          *light.CommitteeChain
	cs             *light.CheckpointStore
	checkpointHash common.Hash
	initialized    bool

	InitTrigger request.ModuleTrigger
}

func NewCheckpointInit(chain *light.CommitteeChain, cs *light.CheckpointStore, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		cs:             cs,
		checkpointHash: checkpointHash,
	}
}

func (s *CheckpointInit) Process(servers []*request.Server) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.initialized {
		return false
	}
	if checkpoint := s.cs.Get(s.checkpointHash); checkpoint != nil {
		checkpoint.InitChain(s.chain)
		s.initialized = true
		s.InitTrigger.Trigger()
		return false
	}
	srv := request.SelectServer(servers, func(server *request.Server) uint64 {
		if cserver, ok := server.RequestServer.(checkpointInitServer); ok && cserver.CanRequestBootstrap() && s.CanSend(server) {
			return 1
		}
		return 0
	})
	if srv == nil {
		return true
	}
	reqId, ok := s.TrySend(srv)
	if !ok {
		return true
	}
	server := srv.RequestServer.(checkpointInitServer)
	server.RequestBootstrap(s.checkpointHash, func(checkpoint *light.CheckpointData) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.Returned(srv, reqId)
		if checkpoint == nil || !checkpoint.Validate() {
			server.Fail("error retrieving checkpoint data")
			return
		}
		checkpoint.InitChain(s.chain)
		s.cs.Store(checkpoint)
		s.initialized = true
		s.InitTrigger.Trigger()
	})
	return true
}

type forwardUpdateServer interface {
	request.RequestServer
	UpdateRange() types.PeriodRange
	RequestUpdates(first, count uint64, response func([]*types.LightClientUpdate, []*types.SerializedCommittee))
}

type ForwardUpdateSyncer struct {
	request.SingleLock
	lock  sync.Mutex
	chain *light.CommitteeChain

	NewUpdateTrigger request.ModuleTrigger
}

func NewForwardUpdateSyncer(chain *light.CommitteeChain) *ForwardUpdateSyncer {
	return &ForwardUpdateSyncer{chain: chain}
}

func (s *ForwardUpdateSyncer) Process(servers []*request.Server) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	first, ok := s.chain.NextSyncPeriod()
	if !ok {
		return true
	}
	srv := request.SelectServer(servers, func(server *request.Server) uint64 {
		if fserver, ok := server.RequestServer.(forwardUpdateServer); ok && s.CanSend(server) {
			updateRange := fserver.UpdateRange()
			if first < updateRange.First {
				return 0
			}
			return updateRange.AfterLast
		}
		return 0
	})
	if srv == nil {
		return true
	}
	server := srv.RequestServer.(forwardUpdateServer)
	updateRange := server.UpdateRange()
	if updateRange.AfterLast <= first {
		return true
	}
	reqId, ok := s.TrySend(srv)
	if !ok {
		return true
	}
	count := updateRange.AfterLast - first
	if count > maxUpdateRequest { //TODO const
		count = maxUpdateRequest
	}
	server.RequestUpdates(first, count, func(updates []*types.LightClientUpdate, committees []*types.SerializedCommittee) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.Returned(srv, reqId)
		if len(updates) != int(count) || len(committees) != int(count) {
			server.Fail("wrong number of updates received")
			return
		}
		for i, update := range updates {
			if update.Header.SyncPeriod() != first+uint64(i) {
				server.Fail("update with wrong sync period received")
				return
			}
			if err := s.chain.InsertUpdate(update, committees[i]); err != nil {
				if err == light.ErrInvalidUpdate || err == light.ErrWrongCommitteeRoot || err == light.ErrCannotReorg {
					server.Fail("invalid update received")
				} else {
					log.Error("Unexpected InsertUpdate error", "error", err)
				}
				if i != 0 {
					s.NewUpdateTrigger.Trigger()
				}
				return
			}
		}
		s.NewUpdateTrigger.Trigger()
	})
	return true
}
