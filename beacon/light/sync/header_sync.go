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

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
)

type beaconHeaderServer interface {
	request.RequestServer
	RequestBeaconHeader(blockRoot common.Hash, response func(*types.Header))
}

type HeaderSync struct {
	lock                      sync.Mutex
	reqLock                   request.MultiLock
	chain                     *light.LightChain
	prefetch                  bool
	targetHead, syncPtr       types.Header
	targetTailSlot            uint64
	selfTrigger, chainTrigger *request.ModuleTrigger
}

func NewHeaderSync(chain *light.LightChain, prefetch bool) *HeaderSync {
	return &HeaderSync{
		chain:          chain,
		prefetch:       prefetch,
		targetTailSlot: math.MaxUint64,
	}
}

func (s *HeaderSync) SetupTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	s.selfTrigger = trigger("headerSync", true)
	s.reqLock.Trigger = s.selfTrigger
	trigger("validatedHead", true)
	s.chainTrigger = trigger("headerChain", false)
}

func (s *HeaderSync) SetTailTarget(targetTailSlot uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if targetTailSlot < s.targetTailSlot {
		s.selfTrigger.Trigger()
	}
	s.targetTailSlot = targetTailSlot
}

func (s *HeaderSync) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	validatedHead := env.ValidatedHead()
	if validatedHead != s.targetHead {
		s.targetHead = validatedHead
		s.syncPtr = validatedHead
		s.chain.AddHeader(validatedHead)
	}
	if s.targetHead == (types.Header{}) {
		return
	}
	chainHead, chainTail, chainInit := s.chain.HeaderRange()
	if !chainInit {
		s.chain.AddHeader(s.targetHead)
		s.chain.SetChainHead(s.targetHead)
		s.selfTrigger.Trigger()
		s.chainTrigger.Trigger()
	}
	if s.prefetch {
		if prefetchHead := env.PrefetchHead(); !s.chain.HasHeader(prefetchHead) {
			s.tryPrefetchHead(env, prefetchHead)
		}
	}
	if chainHead != s.targetHead && !s.trySyncHead(env, chainTail.Slot) {
		// always prioritize syncing to the latest head, do not start tail sync until done
		return
	}
	if s.targetTailSlot < chainTail.Slot {
		s.trySyncTail(env, chainTail)
	}
}

// returns true if targetHead has been reached
func (s *HeaderSync) trySyncHead(env *request.Environment, chainTailSlot uint64) bool {
	for {
		if s.syncPtr.Slot <= chainTailSlot || s.chain.IsCanonical(s.syncPtr) {
			s.chain.SetChainHead(s.targetHead)
			s.chainTrigger.Trigger()
			return true
		}
		if parent, err := s.chain.GetHeaderByHash(s.syncPtr.ParentRoot); err == nil {
			s.syncPtr = parent
		} else {
			s.tryRequestHeader(env, s.syncPtr.ParentRoot, false)
			return false
		}
	}
}

func (s *HeaderSync) trySyncTail(env *request.Environment, syncTail types.Header) {
	for {
		if parent, err := s.chain.GetHeaderByHash(syncTail.ParentRoot); err == nil {
			syncTail = parent
		} else {
			s.tryRequestHeader(env, syncTail.ParentRoot, false)
			return
		}
	}
}

func (s *HeaderSync) tryPrefetchHead(env *request.Environment, head common.Hash) {
	if head != (common.Hash{}) && !s.chain.HasHeader(head) {
		s.tryRequestHeader(env, head, true)
	}
}

func (s *HeaderSync) tryRequestHeader(env *request.Environment, blockRoot common.Hash, prefetch bool) {
	if s.reqLock.CanRequest(blockRoot) {
		env.TryRequest(headerRequest{
			HeaderSync: s,
			blockRoot:  blockRoot,
			prefetch:   prefetch,
		})
	}
}

type headerRequest struct {
	*HeaderSync
	blockRoot common.Hash
	prefetch  bool
}

func (r headerRequest) CanSendTo(server *request.Server) (canSend bool, priority uint64) {
	if _, ok := server.RequestServer.(beaconHeaderServer); !ok {
		return false, 0
	}
	if !r.prefetch {
		return true, 0
	}
	_, headRoot := server.LatestHead()
	return r.blockRoot == headRoot, 0
}

func (r headerRequest) SendTo(server *request.Server) {
	reqId := r.reqLock.Send(server, r.blockRoot)
	server.RequestServer.(beaconHeaderServer).RequestBeaconHeader(r.blockRoot, func(header *types.Header) {
		r.lock.Lock()
		defer r.lock.Unlock()

		r.reqLock.Returned(server, reqId, r.blockRoot)
		if header == nil {
			server.Fail("error retrieving beacon header")
			return
		}
		_, oldChainTail, _ := r.chain.HeaderRange()
		r.chain.AddHeader(*header)
		_, chainTail, _ := r.chain.HeaderRange()
		if chainTail != oldChainTail {
			r.chainTrigger.Trigger() //TODO do this in a nicer way?
		}
	})
}
