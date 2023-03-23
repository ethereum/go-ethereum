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

func (s *Scheduler) newServer(server RequestServer) *Server {
	return &Server{
		RequestServer: server,
		scheduler:     s,
		sent:          make(map[uint64]chan struct{}),
		stopCh:        make(chan struct{}),
	}
}

func (s *Server) setHead(slot uint64, blockRoot common.Hash) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	s.latestHeadSlot, s.latestHeadHash = slot, blockRoot
}

func (s *Server) trigger() {
	s.scheduler.triggerServer(s)
}

func (s *Server) LatestHead() (uint64, common.Hash) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.latestHeadSlot, s.latestHeadHash
}

// guarantees a server trigger later if the result is false
func (s *Server) CanRequestNow() (bool, uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isDelayed() || s.timeoutCount != 0 {
		s.needTrigger = true
		return false, 0
	}
	return true, uint64(rand.Uint32() + 1) //TODO use priority based on in-flight requests
}

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

func (s *Server) hasTimedOut(reqId uint64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ch, ok := s.sent[reqId]
	return ok && ch == nil
}

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

func (s *Server) stop() {
	close(s.stopCh)
}

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
