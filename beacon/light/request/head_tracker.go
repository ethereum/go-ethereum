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

package request

import (
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
)

// HeadTracker subscribes to the head events of each connected server and keeps
// track of validated and prefetch heads.
// Note that it does not do signed head validation; it simply passes signed heads
// received from server event feeds to the callback received in the constructor.
// The actual validated head is set externally. In a light client setup its source
// is the signed head validation mechanism but in a different setup it can be
// something else. In a local full beacon node setup the head can be validated by
// the trusted engine API while head BLS signatures are validated later as they
// appear, in order to be passed on to other clients.
type HeadTracker struct {
	newSignedHead func(server *Server, signedHead types.SignedHeader)

	validatedLock        sync.RWMutex
	validatedHead        types.Header
	validatedHeadTrigger *ModuleTrigger

	prefetchLock        sync.RWMutex
	serverHeads         map[*Server]common.Hash
	headInfo            map[common.Hash]serverHeadInfo
	headCounter         uint64
	prefetchHead        common.Hash
	prefetchHeadTrigger *ModuleTrigger
}

type serverHeadInfo struct {
	serverCount int
	headCounter uint64
}

// NewHeadTracker creates a new HeadTracker. The newSignedHead head callback is
// called whenever a signed head is received from any of the connected servers.
func NewHeadTracker(newSignedHead func(server *Server, signedHead types.SignedHeader)) *HeadTracker {
	return &HeadTracker{
		serverHeads:   make(map[*Server]common.Hash),
		headInfo:      make(map[common.Hash]serverHeadInfo),
		newSignedHead: newSignedHead,
	}
}

// setupModuleTriggers sets up triggers for new validated and prefetch heads when
// HeadTracker is added to a Scheduler.
func (s *HeadTracker) setupModuleTriggers(trigger func(id string) *ModuleTrigger) {
	s.validatedHeadTrigger = trigger("validatedHead")
	s.prefetchHeadTrigger = trigger("prefetchHead")
}

// SetValidatedHead is called by the external validated head source.
func (s *HeadTracker) SetValidatedHead(head types.Header) {
	s.validatedLock.Lock()
	defer s.validatedLock.Unlock()

	s.validatedHead = head
	s.validatedHeadTrigger.Trigger()
}

// ValidatedHead returns the latest validated head.
func (s *HeadTracker) ValidatedHead() types.Header {
	s.validatedLock.RLock()
	defer s.validatedLock.RUnlock()

	return s.validatedHead
}

// registerServer registers a server and subscribes to its head events.
func (s *HeadTracker) registerServer(server *Server) {
	s.prefetchLock.Lock()
	defer s.prefetchLock.Unlock()

	server.SubscribeHeads(func(slot uint64, blockRoot common.Hash) {
		s.prefetchLock.Lock()
		defer s.prefetchLock.Unlock()

		if server.unregistered {
			return
		}
		server.setHead(slot, blockRoot)
		s.setServerHead(server, blockRoot)
		server.scheduler.triggerServer(server)
	}, func(signedHead types.SignedHeader) {
		s.newSignedHead(server, signedHead)
	})
}

// unregisterServer removes a server and unsubscribes from its events.
func (s *HeadTracker) unregisterServer(server *Server) {
	s.prefetchLock.Lock()
	defer s.prefetchLock.Unlock()

	server.UnsubscribeHeads()
	server.unregistered = true
	s.setServerHead(server, common.Hash{})
}

// setServerHead processes non-validated server head announcements and updates
// the prefetch head if necessary.
//TODO report server failure if a server announces many heads that do not become validated soon.
func (s *HeadTracker) setServerHead(server *Server, head common.Hash) {
	if oldHead, ok := s.serverHeads[server]; ok {
		if head == oldHead {
			return
		}
		h := s.headInfo[oldHead]
		if h.serverCount--; h.serverCount > 0 {
			s.headInfo[oldHead] = h
		} else {
			delete(s.headInfo, oldHead)
		}
	}
	if head != (common.Hash{}) {
		h, ok := s.headInfo[head]
		if !ok {
			s.headCounter++
			h.headCounter = s.headCounter
		}
		h.serverCount++
		s.headInfo[head] = h
	}
	var (
		bestHead     common.Hash
		bestHeadInfo serverHeadInfo
	)
	for head, headInfo := range s.headInfo {
		if headInfo.serverCount > bestHeadInfo.serverCount ||
			(headInfo.serverCount == bestHeadInfo.serverCount && headInfo.headCounter > bestHeadInfo.headCounter) {
			bestHead, bestHeadInfo = head, headInfo
		}
	}
	if bestHead != s.prefetchHead {
		s.prefetchHead = bestHead
		s.prefetchHeadTrigger.Trigger()
	} else if head == s.prefetchHead {
		server.scheduler.triggerServer(server)
	}
}

// PrefetchHead returns the current prefetch head block root. The prefetch head
// is defined as the one that the most currently connected servers have as their
// latest announced head. If multiple heads are announced by the same number of
// servers then the newest one is selected.
// Prefetch heads should not be trusted but can be used to start fetching
// block-related data before it becomes validated.
func (s *HeadTracker) PrefetchHead() common.Hash {
	s.prefetchLock.RLock()
	defer s.prefetchLock.RUnlock()

	return s.prefetchHead
}
