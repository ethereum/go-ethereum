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
)

type headTracker interface {
	ValidateHead(head types.SignedHeader) (bool, error)
	ValidateFinality(head types.FinalityUpdate) (bool, error)
	SetPrefetchHead(head types.HeadInfo)
}

// HeadSync implements request.Module; it updates the validated and prefetch
// heads of HeadTracker based on the EvHead and EvSignedHead events coming from
// registered servers.
// It can also postpone the validation of the latest announced signed head
// until the committee chain is synced up to at least the required period.
type HeadSync struct {
	headTracker          headTracker
	chain                committeeChain
	nextSyncPeriod       uint64
	chainInit            bool
	unvalidatedHeads     map[request.Server]types.SignedHeader
	unvalidatedFinality  map[request.Server]types.FinalityUpdate
	serverHeads          map[request.Server]types.HeadInfo
	serverValidatedEpoch map[request.Server]uint64
	headServerCount      map[types.HeadInfo]headServerCount
	headCounter          uint64
	prefetchHead         types.HeadInfo

	locked                                bool
	lockEpoch                             uint64
	validatedEpoch, finalityAttestedEpoch uint64
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
		headTracker:          headTracker,
		chain:                chain,
		unvalidatedHeads:     make(map[request.Server]types.SignedHeader),
		unvalidatedFinality:  make(map[request.Server]types.FinalityUpdate),
		serverHeads:          make(map[request.Server]types.HeadInfo),
		headServerCount:      make(map[types.HeadInfo]headServerCount),
		serverValidatedEpoch: make(map[request.Server]uint64),
	}
	return s
}

// Process implements request.Module.
func (s *HeadSync) Process(requester request.Requester, events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			s.processReqEvent(event)
		case EvNewHead:
			s.setServerHead(event.Server, event.Data.(types.HeadInfo))
		case EvNewSignedHead:
			s.newSignedHead(event.Server, event.Data.(types.SignedHeader))
		case EvNewFinalityUpdate:
			s.newFinalityUpdate(event.Server, event.Data.(types.FinalityUpdate))
		case request.EvUnregistered:
			s.setServerHead(event.Server, types.HeadInfo{})
			delete(s.servers, event.Server)
		}
	}

	nextPeriod, chainInit := s.chain.NextSyncPeriod()
	if nextPeriod != s.nextSyncPeriod || chainInit != s.chainInit {
		s.nextSyncPeriod, s.chainInit = nextPeriod, chainInit
		s.processUnvalidated()
	}

	if s.validatedEpoch > s.finalityAttestedEpoch && (!s.locked || s.validatedEpoch > s.lockEpoch) {
		s.tryRequestFinalityUpdate(requester, s.validatedEpoch)
	}
}

// newSignedHead handles received signed head; either validates it if the chain
// is properly synced or stores it for further validation.
func (s *HeadSync) newSignedHead(server request.Server, signedHead types.SignedHeader) {
	if !s.chainInit || types.SyncPeriod(signedHead.SignatureSlot) > s.nextSyncPeriod {
		s.unvalidatedHeads[server] = signedHead
		return
	}
	s.validateHead(server, signedHead)
}

func (s *HeadSync) validateHead(server request.Server, signedHead types.SignedHeader) bool {
	replaced, err := s.headTracker.ValidateHead(signedHead)
	if err != nil {
		return false
	}
	epoch := signedHead.Header.Epoch()
	s.servers[server].validatedEpoch = epoch
	if replaced {
		s.validatedEpoch = epoch
	}
	return true
}

// newSignedHead handles received signed head; either validates it if the chain
// is properly synced or stores it for further validation.
func (s *HeadSync) newFinalityUpdate(server request.Server, finalityUpdate types.FinalityUpdate) {
	if !s.chainInit || types.SyncPeriod(finalityUpdate.SignatureSlot) > s.nextSyncPeriod {
		s.unvalidatedFinality[server] = finalityUpdate
		return
	}
	s.validateFinality(finalityUpdate)
}

func (s *HeadSync) validateFinality(finalityUpdate types.FinalityUpdate) bool {
	replaced, err := s.headTracker.ValidateFinality(finalityUpdate)
	if err != nil {
		return false
	}
	if replaced {
		s.finalityAttestedEpoch = finalityUpdate.AttestedHeader.Header.Epoch()
	}
	return true
}

// processUnvalidatedHeads iterates the list of unvalidated heads and validates
// those which can be validated.
func (s *HeadSync) processUnvalidated() {
	if !s.chainInit {
		return
	}
	for server, signedHead := range s.unvalidatedHeads {
		if types.SyncPeriod(signedHead.SignatureSlot) <= s.nextSyncPeriod {
			s.validateHead(server, signedHead)
			delete(s.unvalidatedHeads, server)
		}
	}
	for server, finalityUpdate := range s.unvalidatedFinality {
		if types.SyncPeriod(finalityUpdate.SignatureSlot) <= s.nextSyncPeriod {
			s.validateFinality(finalityUpdate)
			delete(s.unvalidatedFinality, server)
		}
	}
}

func (s *HeadSync) processReqEvent(event request.Event) {
	sid, req, resp := event.RequestInfo()
	reqEpoch := uint64(req.(sync.ReqFinality))
	if s.locked && s.lockEpoch == reqEpoch {
		s.locked = false
	}
	if resp == nil {
		return
	}
	finalityUpdate := resp.(types.FinalityUpdate)
	if finalityUpdate.AttestedHeader.Header.Epoch() != reqEpoch {
		requester.Fail(sid.Server, "finality update for a wrong epoch")
		return
	}
	if !s.validateFinality(finalityUpdate) {
		requester.Fail(sid.Server, "invalid finality update")
	}
}

func (s *HeadSync) tryRequestFinalityUpdate(requester request.Requester, reqEpoch uint64) {
	for _, server := range requester.CanSendTo() {
		if s.serverValidatedEpoch[server] < reqEpoch {
			continue
		}
		id := requester.Send(server, sync.ReqFinality(reqEpoch))
		s.locked[blockRoot] = request.ServerAndID{Server: server, ID: id}
		return
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
