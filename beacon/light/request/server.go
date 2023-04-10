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
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Server is a wrapper around RequestServer that handles request timeouts, delays
// and keeps track of the server's latest reported (not necessarily validated) head.
type Server struct {
	RequestServer
	scheduler *Scheduler

	headLock       sync.RWMutex
	latestHeadSlot uint64
	latestHeadHash common.Hash
	unregistered   bool // accessed under HeadTracker.prefetchLock

	lock         sync.Mutex
	sent         map[uint64]chan struct{} // closed when returned; nil when timed out
	timeoutCount int
	delayed      bool
	delayChecked bool
	needTrigger  bool
	lastReqId    uint64
	stopCh       chan struct{}
}

// newServer creates a new Server.
func (s *Scheduler) newServer(server RequestServer) *Server {
	return &Server{
		RequestServer: server,
		scheduler:     s,
		sent:          make(map[uint64]chan struct{}),
		stopCh:        make(chan struct{}),
	}
}

// setHead is called by the head event subscription.
func (s *Server) setHead(slot uint64, blockRoot common.Hash) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	s.latestHeadSlot, s.latestHeadHash = slot, blockRoot
}

// LatestHead returns the server's latest reported head (slot and block root).
// Note: the reason we can't return the full header here is that the standard
// beacon API head event only contains the slot and block root.
func (s *Server) LatestHead() (uint64, common.Hash) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.latestHeadSlot, s.latestHeadHash
}

// canRequestNow returns true if a request can be sent to the server immediately
// (has no timed out requests and underlying RequestServer does not require new
// requests to be delayed). It also returns a priority value that is taken into
// account when otherwise equally good servers are available.
// Note: if canRequestNow ever returns false then it is guaranteed that a server
// trigger will be emitted as soon as it becomes true again.
func (s *Server) canRequestNow() (bool, uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isDelayed() || s.timeoutCount != 0 {
		s.needTrigger = true
		return false, 0
	}
	//TODO use priority based on in-flight requests (less is better)
	return true, uint64(rand.Uint32() + 1)
}

// isDelayed returns true if the underlying RequestServer requires requests to be
// delayed. In this case it also starts a timer to ensure that a server trigger
// can be emitted when the server becomes available again.
func (s *Server) isDelayed() bool {
	if s.delayChecked {
		return s.delayed
	}
	s.delayChecked = true
	delay := s.RequestServer.Delay()
	if s.delayed = delay > 0; s.delayed {
		go func() {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				s.lock.Lock()
				s.delayed = false
				if s.needTrigger && s.timeoutCount == 0 {
					s.needTrigger = false
					s.scheduler.triggerServer(s)
				}
				s.lock.Unlock()
			case <-s.stopCh:
				timer.Stop()
			}
		}()
	}
	return s.delayed
}

// sendRequest generates a request ID and starts a timeout timer. If the timeout
// is reached then the trigger is triggered (if specified).
func (s *Server) sendRequest(timeoutTrigger *ModuleTrigger) uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lastReqId++
	reqId := s.lastReqId
	returnCh := make(chan struct{})
	s.sent[reqId] = returnCh
	s.delayChecked = false
	go func() {
		timer := time.NewTimer(softRequestTimeout)
		select {
		case <-timer.C:
			s.lock.Lock()
			if _, ok := s.sent[reqId]; ok {
				s.sent[reqId] = nil
				s.timeoutCount++
			}
			s.lock.Unlock()
			if timeoutTrigger != nil {
				timeoutTrigger.Trigger()
			}
		case <-returnCh:
			timer.Stop()
		case <-s.stopCh:
			timer.Stop()
		}
	}()
	return reqId
}

// hasTimedOut returns true if the given request has timed out.
func (s *Server) hasTimedOut(reqId uint64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ch, ok := s.sent[reqId]
	return ok && ch == nil
}

// returned stops the timeout timer and removes the entry associated with the
// request ID. It should always be called for every sent request (even if the
// response is an error or useless) unless the server is dropped.
func (s *Server) returned(reqId uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if ch, ok := s.sent[reqId]; ok {
		if ch != nil {
			close(ch)
		} else {
			s.timeoutCount--
			if s.needTrigger && s.timeoutCount == 0 && !s.isDelayed() {
				s.needTrigger = false
				s.scheduler.triggerServer(s)
			}
		}
		delete(s.sent, reqId)
	}
}

// stop stops all goroutines associated with the server.
func (s *Server) stop() {
	close(s.stopCh)
}
