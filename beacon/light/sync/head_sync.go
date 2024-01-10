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

// HeadSync implements request.Module; it updates the validated and prefetch
// heads of HeadTracker based on the EvHead and EvSignedHead events coming from
// registered servers.
// It can also postpone the validation of the latest announced signed head
// until the committee chain is synced up to at least the required period.
type HeadSync struct {
	headTracker      headTracker
	chain            committeeChain
	nextSyncPeriod   uint64
	chainInit        bool
	unvalidatedHeads map[request.Server]types.SignedHeader
	serverHeads      map[request.Server]types.HeadInfo
	headServerCount  map[types.HeadInfo]headServerCount
	headCounter      uint64
	prefetchHead     types.HeadInfo
}

// headServerCount is associated with most recently seen head infos; it counts
// the number of servers currently having the given head info as their announced
// head and a counter signaling how recent that head is.
// This data is used for selecting the prefetch head.
type headServerCount struct {
	serverCount int
	headCounter uint64
}

// NewHeadSync creates a new HeadSync.
func NewHeadSync(headTracker headTracker, chain committeeChain) *HeadSync {
	s := &HeadSync{
		headTracker:      headTracker,
		chain:            chain,
		nextSyncPeriod:   math.MaxUint64,
		unvalidatedHeads: make(map[request.Server]types.SignedHeader),
		serverHeads:      make(map[request.Server]types.HeadInfo),
		headServerCount:  make(map[types.HeadInfo]headServerCount),
	}
	return s
}

// Process implements request.Module
func (s *HeadSync) Process(tracker request.Tracker, events []request.Event) (trigger bool) {
	nextPeriod, chainInit := s.chain.NextSyncPeriod()
	if nextPeriod != s.nextSyncPeriod || chainInit != s.chainInit {
		s.nextSyncPeriod, s.chainInit = nextPeriod, chainInit
		trigger = s.processUnvalidatedHeads()
	}
	for _, event := range events {
		switch event.Type {
		case EvNewHead:
			if s.setServerHead(event.Server, event.Data.(types.HeadInfo)) {
				trigger = true
			}
		case EvNewSignedHead:
			if s.newSignedHead(event.Server, event.Data.(types.SignedHeader)) {
				trigger = true
			}
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

// newSignedHead handles received signed head; either validates it if the chain
// is properly synced or stores it for further validation.
func (s *HeadSync) newSignedHead(server request.Server, signedHead types.SignedHeader) (trigger bool) {
	if !s.chainInit || types.SyncPeriod(signedHead.SignatureSlot) > s.nextSyncPeriod {
		s.unvalidatedHeads[server] = signedHead
		return false
	}
	updated, _ := s.headTracker.Validate(signedHead)
	return updated
}

// processUnvalidatedHeads iterates the list of unvalidated heads and validates
// those which can be validated.
func (s *HeadSync) processUnvalidatedHeads() (trigger bool) {
	if !s.chainInit {
		return false
	}
	for server, signedHead := range s.unvalidatedHeads {
		if types.SyncPeriod(signedHead.SignatureSlot) <= s.nextSyncPeriod {
			if updated, _ := s.headTracker.Validate(signedHead); updated {
				trigger = true
			}
			delete(s.unvalidatedHeads, server)
		}
	}
	return
}

// setServerHead processes non-validated server head announcements and updates
// the prefetch head if necessary.
//TODO report server failure if a server announces many heads that do not become validated soon.
func (s *HeadSync) setServerHead(server request.Server, head types.HeadInfo) bool {
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
