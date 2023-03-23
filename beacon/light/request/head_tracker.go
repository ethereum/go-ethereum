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

type HeadTracker struct {
	newSignedHead func(server *Server, signedHead types.SignedHead)

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

func NewHeadTracker(newSignedHead func(server *Server, signedHead types.SignedHead)) *HeadTracker {
	return &HeadTracker{
		serverHeads:   make(map[*Server]common.Hash),
		headInfo:      make(map[common.Hash]serverHeadInfo),
		newSignedHead: newSignedHead,
	}
}

func (s *HeadTracker) SetupTriggers(trigger func(id string) *ModuleTrigger) {
	s.validatedHeadTrigger = trigger("validatedHead")
	s.prefetchHeadTrigger = trigger("prefetchHead")
}

func (s *HeadTracker) SetValidatedHead(head types.Header) {
	s.validatedLock.Lock()
	defer s.validatedLock.Unlock()

	s.validatedHead = head
	s.validatedHeadTrigger.Trigger()
}

func (s *HeadTracker) ValidatedHead() types.Header {
	s.validatedLock.RLock()
	defer s.validatedLock.RUnlock()

	return s.validatedHead
}

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
		server.trigger()
	}, func(signedHead types.SignedHead) {
		s.newSignedHead(server, signedHead)
	})
}

func (s *HeadTracker) unregisterServer(server *Server) {
	s.prefetchLock.Lock()
	defer s.prefetchLock.Unlock()

	server.UnsubscribeHeads()
	server.unregistered = true
	s.setServerHead(server, common.Hash{})
}

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
		server.trigger()
	}
}

func (s *HeadTracker) PrefetchHead() common.Hash {
	s.prefetchLock.RLock()
	defer s.prefetchLock.RUnlock()

	return s.prefetchHead
}
