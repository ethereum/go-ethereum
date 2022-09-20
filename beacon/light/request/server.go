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
)

func SelectServer(servers []*Server, priority func(server *Server) uint64) *Server {
	var (
		maxPriority uint64
		mpCount     int
		bestServer  *Server
	)
	for _, server := range servers {
		pri := priority(server)
		if pri == 0 || pri < maxPriority { // 0 means it cannot serve the request at all
			continue
		}
		if pri > maxPriority {
			maxPriority = pri
			mpCount = 1
			bestServer = server
		} else {
			mpCount++
			if rand.Intn(mpCount) == 0 {
				bestServer = server
			}
		}
	}
	return bestServer
}

type Server struct { //TODO name?
	RequestServer
	scheduler    *Scheduler
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

// guarantees a server trigger later if the result is false
func (s *Server) CanSend() bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isDelayed() || s.timeoutCount != 0 {
		s.needTrigger = true
		return false
	}
	return true
}

// guarantees a server trigger later if the result is false
func (s *Server) TrySend() (uint64, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isDelayed() || s.timeoutCount != 0 {
		s.needTrigger = true
		return 0, false
	}
	s.lastReqId++
	returnCh := make(chan struct{})
	s.sent[s.lastReqId] = returnCh
	s.delayChecked = false
	go func() {
		timer := time.NewTimer(softRequestTimeout)
		select {
		case <-timer.C:
			s.lock.Lock()
			if _, ok := s.sent[s.lastReqId]; ok {
				s.sent[s.lastReqId] = nil
				s.timeoutCount++
			}
			s.lock.Unlock()
		case <-returnCh:
			timer.Stop()
		case <-s.stopCh:
			timer.Stop()
		}
	}()
	return s.lastReqId, true
}

func (s *Server) Timeout(reqId uint64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ch, ok := s.sent[reqId]
	return ok && ch == nil
}

func (s *Server) Returned(reqId uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if ch, ok := s.sent[reqId]; ok {
		if ch != nil {
			close(ch)
		} else {
			s.timeoutCount--
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
				trigger := s.needTrigger && s.timeoutCount == 0
				s.lock.Unlock()
				if trigger {
					s.scheduler.serverTrigger(s)
				}
			case <-s.stopCh:
				timer.Stop()
			}
		}()
	}
	return s.delayed
}
