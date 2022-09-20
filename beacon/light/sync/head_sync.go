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
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type HeadSync struct {
	headTracker     *light.HeadTracker
	chain           *light.CommitteeChain
	nextSyncPeriod  uint64
	chainInit       bool
	queuedHeads     map[request.Server][]types.SignedHeader
	serverHeads     map[request.Server]types.HeadInfo
	headServerCount map[types.HeadInfo]headServerCount
	headCounter     uint64
	prefetchHead    types.HeadInfo
}

type headServerCount struct {
	serverCount int
	headCounter uint64
}

func NewHeadSync(headTracker *light.HeadTracker, chain *light.CommitteeChain) *HeadSync {
	s := &HeadSync{
		headTracker:     headTracker,
		chain:           chain,
		nextSyncPeriod:  math.MaxUint64,
		queuedHeads:     make(map[request.Server][]types.SignedHeader),
		serverHeads:     make(map[request.Server]types.HeadInfo),
		headServerCount: make(map[types.HeadInfo]headServerCount),
	}
	return s
}

// Process implements request.Module
func (s *HeadSync) Process(tracker *request.RequestTracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) (trigger bool) {
	nextPeriod, chainInit := s.chain.NextSyncPeriod()
	if nextPeriod != s.nextSyncPeriod || chainInit != s.chainInit {
		s.nextSyncPeriod, s.chainInit = nextPeriod, chainInit
		s.processQueuedHeads()
	}
	for _, event := range serverEvents {
		switch event.Type {
		case EvNewHead:
			trigger = trigger || s.setServerHead(event.Server, event.Data.(types.HeadInfo))
		case EvNewSignedHead:
			s.newSignedHead(event.Server, event.Data.(types.SignedHeader))
		case request.EvUnregistered:
			trigger = trigger || s.setServerHead(event.Server, types.HeadInfo{})
			delete(s.serverHeads, event.Server)
			delete(s.queuedHeads, event.Server)
		}
	}
	return
}

func (s *HeadSync) newSignedHead(server request.Server, signedHead types.SignedHeader) {
	if signedHead.Header.SyncPeriod() > s.nextSyncPeriod {
		s.queuedHeads[server] = append(s.queuedHeads[server], signedHead) //TODO protect against future period spam
		return
	}
	if _, err := s.headTracker.Validate(signedHead); err != nil {
		server.Fail(fmt.Sprintf("Invalid signed head: %v", err))
	}
}

func (s *HeadSync) processQueuedHeads() {
	for server, queued := range s.queuedHeads {
		j := len(queued)
		for i := len(queued) - 1; i >= 0; i-- {
			if signedHead := queued[i]; signedHead.Header.SyncPeriod() <= s.nextSyncPeriod {
				if _, err := s.headTracker.Validate(signedHead); err != nil {
					server.Fail(fmt.Sprintf("Invalid queued head: %v", err))
				}
			} else {
				j--
				if j != i {
					queued[j] = queued[i]
				}
			}
		}
		if j != 0 {
			s.queuedHeads[server] = queued[j:]
		}
	}
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
