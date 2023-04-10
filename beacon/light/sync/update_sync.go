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
	lock           sync.Mutex
	reqLock        request.SingleLock
	chain          *light.CommitteeChain
	cs             *light.CheckpointStore
	checkpointHash common.Hash
	initialized    bool

	initTrigger *request.ModuleTrigger
}

func NewCheckpointInit(chain *light.CommitteeChain, cs *light.CheckpointStore, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		cs:             cs,
		checkpointHash: checkpointHash,
	}
}

// SetupModuleTriggers implements request.Module
func (s *CheckpointInit) SetupModuleTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	s.reqLock.Trigger = trigger("checkpointInit", true)
	s.initTrigger = trigger("committeeChainInit", false)
}

// Process implements request.Module
func (s *CheckpointInit) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.initialized {
		return
	}
	if checkpoint := s.cs.Get(s.checkpointHash); checkpoint != nil {
		checkpoint.InitChain(s.chain)
		s.initialized = true
		s.initTrigger.Trigger()
		return
	}
	if !env.CanRequestNow() {
		return
	}
	if s.reqLock.CanRequest() {
		env.TryRequest(checkpointRequest{
			CheckpointInit: s,
			checkpointHash: s.checkpointHash,
		})
	}
}

type checkpointRequest struct {
	*CheckpointInit
	checkpointHash common.Hash
}

func (r checkpointRequest) CanSendTo(server *request.Server) (canSend bool, priority uint64) {
	if cs, ok := server.RequestServer.(checkpointInitServer); !ok || !cs.CanRequestBootstrap() {
		return false, 0
	}
	return true, 0
}

func (r checkpointRequest) SendTo(server *request.Server) {
	reqId := r.reqLock.Send(server)
	server.RequestServer.(checkpointInitServer).RequestBootstrap(r.checkpointHash, func(checkpoint *light.CheckpointData) {
		r.lock.Lock()
		defer r.lock.Unlock()

		r.reqLock.Returned(server, reqId)
		if checkpoint == nil || !checkpoint.Validate() {
			server.Fail("error retrieving checkpoint data")
			return
		}
		checkpoint.InitChain(r.chain)
		r.cs.Store(checkpoint)
		r.initialized = true
		r.initTrigger.Trigger()
	})
}

type updateServer interface {
	request.RequestServer
	UpdateRange() types.PeriodRange
	RequestUpdates(first, count uint64, response func([]*types.LightClientUpdate, []*types.SerializedCommittee))
}

type ForwardUpdateSync struct {
	lock    sync.Mutex
	reqLock request.SingleLock
	chain   *light.CommitteeChain

	newUpdateTrigger *request.ModuleTrigger
}

func NewForwardUpdateSync(chain *light.CommitteeChain) *ForwardUpdateSync {
	return &ForwardUpdateSync{chain: chain}
}

// SetupModuleTriggers implements request.Module
func (s *ForwardUpdateSync) SetupModuleTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	s.reqLock.Trigger = trigger("forwardUpdateSync", true)
	// committeeChainInit signals that the committee chain is initialized (has fixed committee roots) and the first update request can be constructed.
	trigger("committeeChainInit", true)
	// validatedHead ensures that the UpdateRange of each server is re-checked as new heads appear and new updates are synced as they become available.
	trigger("validatedHead", true)
	// newUpdate is triggered when a new update is successfully added to the committee chain
	s.newUpdateTrigger = trigger("newUpdate", true)
}

// Process implements request.Module
func (s *ForwardUpdateSync) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !env.CanRequestNow() {
		return
	}
	first, ok := s.chain.NextSyncPeriod()
	if !ok {
		return
	}
	env.TryRequest(updateRequest{
		ForwardUpdateSync: s,
		first:             first,
	})
}

type updateRequest struct {
	*ForwardUpdateSync
	first uint64
}

func (r updateRequest) CanSendTo(server *request.Server) (canSend bool, priority uint64) {
	if us, ok := server.RequestServer.(updateServer); ok {
		if updateRange := us.UpdateRange(); updateRange.Includes(r.first) {
			return true, updateRange.AfterLast
		}
	}
	return false, 0
}

func (r updateRequest) SendTo(server *request.Server) {
	us := server.RequestServer.(updateServer)
	updateRange := us.UpdateRange()
	count := updateRange.AfterLast - r.first
	if count > maxUpdateRequest {
		count = maxUpdateRequest
	}
	reqId := r.reqLock.Send(server)
	us.RequestUpdates(r.first, count, func(updates []*types.LightClientUpdate, committees []*types.SerializedCommittee) {
		r.lock.Lock()
		defer r.lock.Unlock()

		r.reqLock.Returned(server, reqId)
		if len(updates) != int(count) || len(committees) != int(count) {
			server.Fail("wrong number of updates received")
			return
		}
		for i, update := range updates {
			if update.Header.SyncPeriod() != r.first+uint64(i) {
				server.Fail("update with wrong sync period received")
				return
			}
			if err := r.chain.InsertUpdate(update, committees[i]); err != nil {
				if err == light.ErrInvalidUpdate || err == light.ErrWrongCommitteeRoot || err == light.ErrCannotReorg {
					server.Fail("invalid update received")
				} else {
					log.Error("Unexpected InsertUpdate error", "error", err)
				}
				if i != 0 { // some updates were added
					r.newUpdateTrigger.Trigger()
				}
				return
			}
		}
		r.newUpdateTrigger.Trigger()
	})
}
