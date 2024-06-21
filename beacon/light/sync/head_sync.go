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
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/log"
)

type headTracker interface {
	ValidateOptimistic(update types.OptimisticUpdate) (bool, error)
	ValidateFinality(head types.FinalityUpdate) (bool, error)
	ValidatedFinality() (types.FinalityUpdate, bool)
	SetPrefetchHead(head types.HeadInfo)
}

// HeadSync implements request.Module; it updates the validated and prefetch
// heads of HeadTracker based on the EvHead and EvSignedHead events coming from
// registered servers.
// It can also postpone the validation of the latest announced signed head
// until the committee chain is synced up to at least the required period.
type HeadSync struct {
	headTracker           headTracker
	chain                 committeeChain
	nextSyncPeriod        uint64
	chainInit             bool
	unvalidatedOptimistic map[request.Server]types.OptimisticUpdate
	unvalidatedFinality   map[request.Server]types.FinalityUpdate
	serverHeads           map[request.Server]types.HeadInfo
	reqFinalityEpoch      map[request.Server]uint64 // next epoch to request finality update
	headServerCount       map[types.HeadInfo]headServerCount
	headCounter           uint64
	prefetchHead          types.HeadInfo
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
		headTracker:           headTracker,
		chain:                 chain,
		unvalidatedOptimistic: make(map[request.Server]types.OptimisticUpdate),
		unvalidatedFinality:   make(map[request.Server]types.FinalityUpdate),
		serverHeads:           make(map[request.Server]types.HeadInfo),
		headServerCount:       make(map[types.HeadInfo]headServerCount),
		reqFinalityEpoch:      make(map[request.Server]uint64),
	}
	return s
}

// Process implements request.Module.
func (s *HeadSync) Process(requester request.Requester, events []request.Event) {
	nextPeriod, chainInit := s.chain.NextSyncPeriod()
	if nextPeriod != s.nextSyncPeriod || chainInit != s.chainInit {
		s.nextSyncPeriod, s.chainInit = nextPeriod, chainInit
		s.processUnvalidatedUpdates()
	}

	for _, event := range events {
		switch event.Type {
		case EvNewHead:
			s.setServerHead(event.Server, event.Data.(types.HeadInfo))
		case EvNewOptimisticUpdate:
			update := event.Data.(types.OptimisticUpdate)
			s.newOptimisticUpdate(event.Server, update)
			epoch := update.Attested.Epoch()
			if epoch < s.reqFinalityEpoch[event.Server] {
				continue
			}
			if finality, ok := s.headTracker.ValidatedFinality(); ok && finality.Attested.Header.Epoch() >= epoch {
				continue
			}
			requester.Send(event.Server, ReqFinality{})
			s.reqFinalityEpoch[event.Server] = epoch + 1
		case EvNewFinalityUpdate:
			s.newFinalityUpdate(event.Server, event.Data.(types.FinalityUpdate))
		case request.EvResponse:
			_, _, resp := event.RequestInfo()
			s.newFinalityUpdate(event.Server, resp.(types.FinalityUpdate))
		case request.EvUnregistered:
			s.setServerHead(event.Server, types.HeadInfo{})
			delete(s.serverHeads, event.Server)
			delete(s.unvalidatedOptimistic, event.Server)
			delete(s.unvalidatedFinality, event.Server)
		}
	}
}

// newOptimisticUpdate handles received optimistic update; either validates it if
// the chain is properly synced or stores it for further validation.
func (s *HeadSync) newOptimisticUpdate(server request.Server, optimisticUpdate types.OptimisticUpdate) {
	if !s.chainInit || types.SyncPeriod(optimisticUpdate.SignatureSlot) > s.nextSyncPeriod {
		s.unvalidatedOptimistic[server] = optimisticUpdate
		return
	}
	if _, err := s.headTracker.ValidateOptimistic(optimisticUpdate); err != nil {
		log.Debug("Error validating optimistic update", "error", err)
	}
}

// newFinalityUpdate handles received finality update; either validates it if
// the chain is properly synced or stores it for further validation.
func (s *HeadSync) newFinalityUpdate(server request.Server, finalityUpdate types.FinalityUpdate) {
	if !s.chainInit || types.SyncPeriod(finalityUpdate.SignatureSlot) > s.nextSyncPeriod {
		s.unvalidatedFinality[server] = finalityUpdate
		return
	}
	if _, err := s.headTracker.ValidateFinality(finalityUpdate); err != nil {
		log.Debug("Error validating finality update", "error", err)
	}
}

// processUnvalidatedUpdates iterates the list of unvalidated updates and validates
// those which can be validated.
func (s *HeadSync) processUnvalidatedUpdates() {
	if !s.chainInit {
		return
	}
	for server, optimisticUpdate := range s.unvalidatedOptimistic {
		if types.SyncPeriod(optimisticUpdate.SignatureSlot) <= s.nextSyncPeriod {
			if _, err := s.headTracker.ValidateOptimistic(optimisticUpdate); err != nil {
				log.Debug("Error validating deferred optimistic update", "error", err)
			}
			delete(s.unvalidatedOptimistic, server)
		}
	}
	for server, finalityUpdate := range s.unvalidatedFinality {
		if types.SyncPeriod(finalityUpdate.SignatureSlot) <= s.nextSyncPeriod {
			if _, err := s.headTracker.ValidateFinality(finalityUpdate); err != nil {
				log.Debug("Error validating deferred finality update", "error", err)
			}
			delete(s.unvalidatedFinality, server)
		}
	}
}

// setServerHead processes non-validated server head announcements and updates
// the prefetch head if necessary.
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
