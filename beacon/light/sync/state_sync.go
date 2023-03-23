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
	"math"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type beaconStateServer interface {
	request.RequestServer
	BeaconStateTail() uint64
	RequestBeaconState(slot uint64, stateRoot common.Hash, format merkle.ProofFormat, response func(*merkle.MultiProof))
}

type StateSync struct {
	lock                          sync.Mutex
	reqLock                       request.MultiLock
	chain                         *light.LightChain
	prefetch                      bool
	targetTailSlot                uint64
	headSyncPossible              uint32
	selfTrigger, headStateTrigger *request.ModuleTrigger
}

func NewStateSync(chain *light.LightChain, prefetch bool) *StateSync {
	return &StateSync{
		chain:          chain,
		prefetch:       prefetch,
		targetTailSlot: math.MaxUint64,
	}
}

func (s *StateSync) SetupTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	s.selfTrigger = trigger("stateSync", true)
	s.reqLock.Trigger = s.selfTrigger
	trigger("headerChain", true)
	trigger("prefetchHeader", true)
	s.headStateTrigger = trigger("headState", false)
}

func (s *StateSync) SetTailTarget(targetTailSlot uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if targetTailSlot < s.targetTailSlot {
		s.selfTrigger.Trigger()
	}
	s.targetTailSlot = targetTailSlot
}

func (s *StateSync) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	chainHead, chainTail, chainInit := s.chain.HeaderRange()
	if !chainInit {
		return
	}
	if s.prefetch {
		if header, err := s.chain.GetHeaderByHash(env.PrefetchHead()); err == nil && !s.chain.HasStateProof(header) {
			s.tryPrefetchHead(env, header)
		}
	}
	stateHead, stateTail, stateInit := s.chain.StateProofRange()
	if !stateInit {
		s.tryRequestState(env, chainHead, false)
		stateHead, stateTail = chainHead, chainHead
	} else if stateHead != chainHead {
		if !s.trySyncHead(env, stateHead) {
			return
		}
	}
	targetTailSlot := s.targetTailSlot
	if chainTail.Slot > targetTailSlot {
		targetTailSlot = chainTail.Slot
	}
	if targetTailSlot < stateTail.Slot {
		s.trySyncTail(env, stateTail, targetTailSlot)
	}
}

func (s *StateSync) trySyncHead(env *request.Environment, stateHead types.Header) bool {
	slot, lastBlockRoot := stateHead.Slot, stateHead.Hash()
	for {
		slot++
		header, err := s.chain.GetHeaderBySlot(slot)
		if err == light.ErrEmptySlot {
			continue
		}
		if err == light.ErrNotFound {
			// no more canonical headers; head sync success
			atomic.StoreUint32(&s.headSyncPossible, 1)
			return true
		}
		if err != nil {
			log.Error("Unexpected error during state head sync", "error", err)
			return false
		}
		if header.ParentRoot != lastBlockRoot {
			s.selfTrigger.Trigger() // reorg happened, stop and retry
			return false
		}
		lastBlockRoot = header.Hash()
		if !s.chain.HasStateProof(header) {
			if sentOrLocked, tryLater := s.tryRequestState(env, header, false); !sentOrLocked {
				if !tryLater {
					atomic.StoreUint32(&s.headSyncPossible, 0)
				}
				return false
			}
		}
	}
}

func (s *StateSync) HeadSyncPossible() bool {
	return atomic.LoadUint32(&s.headSyncPossible) == 1
}

func (s *StateSync) trySyncTail(env *request.Environment, stateTail types.Header, targetTailSlot uint64) {
	for stateTail.Slot > targetTailSlot {
		var err error
		stateTail, err = s.chain.GetParent(stateTail)
		if err != nil {
			log.Error("Unexpected error during state tail sync", "error", err)
			return
		}
		if !s.chain.HasStateProof(stateTail) {
			if sentOrLocked, _ := s.tryRequestState(env, stateTail, false); !sentOrLocked {
				return
			}
		}
	}
}

func (s *StateSync) tryPrefetchHead(env *request.Environment, head types.Header) {
	s.tryRequestState(env, head, true)
}

// tryRequestState starts a request for the partial beacon state belonging to the
// specified header if possible. It returns true if further requests should be
// attempted (either starting this one was successful or unnecessary because it
// is already locked by a recent attempt).
func (s *StateSync) tryRequestState(env *request.Environment, header types.Header, prefetch bool) (sentOrLocked, tryLater bool) {
	if !s.reqLock.CanRequest(header.StateRoot) {
		return true, false
	}
	req := stateRequest{
		StateSync: s,
		header:    header,
		prefetch:  prefetch,
	}
	sentOrLocked, _ = env.TryRequest(req)
	if !sentOrLocked {
		tryLater = env.CanRequestLater(req)
	}
	return
}

type stateRequest struct {
	*StateSync
	header   types.Header
	prefetch bool
}

func (r stateRequest) CanSendTo(server *request.Server) (canSend bool, priority uint64) {
	if rs, ok := server.RequestServer.(beaconStateServer); !ok || r.header.Slot < rs.BeaconStateTail() {
		return false, 0
	}
	if !r.prefetch {
		return true, 0
	}
	_, headRoot := server.LatestHead()
	return r.header.Hash() == headRoot, 0
}

func (r stateRequest) SendTo(server *request.Server) {
	reqId := r.reqLock.Send(server, r.header.StateRoot)
	server.RequestServer.(beaconStateServer).RequestBeaconState(r.header.Slot, r.header.StateRoot, r.chain.StateProofFormat(r.header), func(proof *merkle.MultiProof) {
		r.lock.Lock()
		defer r.lock.Unlock()

		r.reqLock.Returned(server, reqId, r.header.StateRoot)
		if proof == nil {
			//server.Fail("error retrieving beacon state proof")
			return
		}
		oldStateHead, _, _ := r.chain.StateProofRange()
		if err := r.chain.AddStateProof(r.header, *proof); err != nil {
			server.Fail("invalid beacon state proof: " + err.Error())
			return
		}
		chainHead, _, _ := r.chain.HeaderRange()
		stateHead, _, _ := r.chain.StateProofRange()
		if stateHead == chainHead && oldStateHead != chainHead {
			r.headStateTrigger.Trigger()
		}
	})
}
