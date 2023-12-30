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

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type headTracker interface {
	Validate(head types.SignedHeader) (bool, error)
	SetPrefetchHead(head types.HeadInfo)
}

type HeadSync struct {
	headTracker      headTracker
	chain            committeeChain
	nextSyncPeriod   uint64
	chainInit        bool
	unvalidatedHeads map[any]types.SignedHeader
	serverHeads      map[any]types.HeadInfo
	headServerCount  map[types.HeadInfo]headServerCount
	headCounter      uint64
	prefetchHead     types.HeadInfo
}

type headServerCount struct {
	serverCount int
	headCounter uint64
}

func NewHeadSync(headTracker headTracker, chain committeeChain) *HeadSync {
	s := &HeadSync{
		headTracker:      headTracker,
		chain:            chain,
		nextSyncPeriod:   math.MaxUint64,
		unvalidatedHeads: make(map[any]types.SignedHeader),
		serverHeads:      make(map[any]types.HeadInfo),
		headServerCount:  make(map[types.HeadInfo]headServerCount),
	}
	return s
}

// Process implements request.Module
func (s *HeadSync) Process(tracker request.Tracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) (trigger bool) {
	nextPeriod, chainInit := s.chain.NextSyncPeriod()
	if nextPeriod != s.nextSyncPeriod || chainInit != s.chainInit {
		s.nextSyncPeriod, s.chainInit = nextPeriod, chainInit
		s.processUnvalidatedHeadsHeads()
	}
	for _, event := range serverEvents {
		switch event.Type {
		case EvNewHead:
			if s.setServerHead(event.Server, event.Data.(types.HeadInfo)) {
				trigger = true
			}
		case EvNewSignedHead:
			s.newSignedHead(event.Server, event.Data.(types.SignedHeader))
		case request.EvUnregistered:
			if s.setServerHead(event.Server, types.HeadInfo{}) {
				trigger = true
			}
			delete(s.serverHeads, event.Server)
			delete(s.unvalidatedHeads, event.Server)
		}
	}
	return
}

func (s *HeadSync) newSignedHead(server any, signedHead types.SignedHeader) {
	if !s.chainInit || signedHead.Header.SyncPeriod() > s.nextSyncPeriod {
		s.unvalidatedHeads[server] = signedHead
		return
	}
	s.headTracker.Validate(signedHead)
}

func (s *HeadSync) processUnvalidatedHeadsHeads() {
	if !s.chainInit {
		return
	}
	for server, signedHead := range s.unvalidatedHeads {
		if types.SyncPeriod(signedHead.SignatureSlot) <= s.nextSyncPeriod {
			s.headTracker.Validate(signedHead)
			delete(s.unvalidatedHeads, server)
		}
	}
}

// setServerHead processes non-validated server head announcements and updates
// the prefetch head if necessary.
//TODO report server failure if a server announces many heads that do not become validated soon.
func (s *HeadSync) setServerHead(server any, head types.HeadInfo) bool {
	if oldHead, ok := s.serverHeads[server]; ok {
		if head == oldHead {
			return false
		}
		h := s.headServerCount[oldHead]
		if h.serverCount--; h.serverCount > 0 {
			s.headServerCount[oldHead] = h
		} else {
			delete(s.headServerCount, oldHead)
		}
	}
	if head != (types.HeadInfo{}) {
		h, ok := s.headServerCount[head]
		if !ok {
			s.headCounter++
			h.headCounter = s.headCounter
		}
		h.serverCount++
		s.headServerCount[head] = h
		s.serverHeads[server] = head
	} else {
		delete(s.serverHeads, server)
	}
	var (
		bestHead     types.HeadInfo
		bestHeadInfo headServerCount
	)
	for head, headServerCount := range s.headServerCount {
		if headServerCount.serverCount > bestHeadInfo.serverCount ||
			(headServerCount.serverCount == bestHeadInfo.serverCount && headServerCount.headCounter > bestHeadInfo.headCounter) {
			bestHead, bestHeadInfo = head, headServerCount
		}
	}
	if bestHead == s.prefetchHead {
		return false
	}
	s.prefetchHead = bestHead
	s.headTracker.SetPrefetchHead(bestHead)
	return true
}
