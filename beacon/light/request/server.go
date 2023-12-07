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

	"github.com/ethereum/go-ethereum/common/mclock"
)

type RequestServer interface {
	Subscribe(eventCallback func(event Event))
	CanSendRequest(request interface{}) (bool, float32)
	SendRequest(request interface{}) (reqId interface{})
	Unsubscribe()
}

type Server interface {
	RequestServer
	CanRequestNow() (bool, float32)
}

type Event struct {
	Type        int
	ReqId, Data interface{}
}

const (
	EvValidResponse   = iota // data: response struct
	EvInvalidResponse        // data: nil
	EvSoftTimeout            // data: nil
	EvHardTimeout            // data: nil
	EvCanRequestAgain        // reqId: nil  data: nil
)

const (
	softRequestTimeout = time.Second
	hardRequestTimeout = time.Second * 10
)

type serverWithTimeout struct {
	RequestServer
	lock         sync.Mutex
	clock        mclock.Clock
	childEventCb func(event Event)
	timeouts     map[interface{}]mclock.Timer
}

func (s *serverWithTimeout) Subscribe(eventCallback func(event Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.childEventCb = eventCallback
	s.RequestServer.Subscribe(s.eventCallback)
}

func (s *serverWithTimeout) eventCallback(event Event) {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch event.Type {
	case EvValidResponse, EvInvalidResponse:
		if timer, ok := s.timeouts[event.ReqId]; ok {
			// Note: if stopping the timer is unsuccessful then the resulting AfterFunc
			// call will just do nothing
			s.stopTimer(timer)
			delete(s.timeouts, event.ReqId)
			s.childEventCb(event)
		}
	default:
		s.childEventCb(event)
	}
}

func (s *serverWithTimeout) SendRequest(request interface{}) (reqId interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	reqId = s.RequestServer.SendRequest(request)
	s.timeouts[reqId] = s.clock.AfterFunc(softRequestTimeout, func() {
		/*if s.testTimerResults != nil {
			s.testTimerResults = append(s.testTimerResults, true) // simulated timer finished
		}*/
		s.lock.Lock()
		defer s.lock.Unlock()

		if _, ok := s.timeouts[reqId]; !ok {
			return
		}
		s.timeouts[reqId] = s.clock.AfterFunc(hardRequestTimeout-softRequestTimeout, func() {
			/*if s.testTimerResults != nil {
				s.testTimerResults = append(s.testTimerResults, true) // simulated timer finished
			}*/
			s.lock.Lock()
			defer s.lock.Unlock()

			if _, ok := s.timeouts[reqId]; !ok {
				return
			}
			delete(s.timeouts, reqId)
			s.childEventCb(Event{Type: EvHardTimeout, ReqId: reqId})
		})
		s.childEventCb(Event{Type: EvSoftTimeout, ReqId: reqId})
	})
	return reqId
}

// stop stops all goroutines associated with the server.
func (s *serverWithTimeout) Unsubscribe() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, timer := range s.timeouts {
		if timer != nil {
			s.stopTimer(timer)
		}
	}
	s.childEventCb = nil
	s.RequestServer.Unsubscribe()
}

func (s *serverWithTimeout) stopTimer(timer mclock.Timer) {
	timer.Stop()
	/*if timer.Stop() && s.scheduler.testTimerResults != nil {
		s.scheduler.testTimerResults = append(s.scheduler.testTimerResults, false) // simulated timer stopped
	}*/
}

type serverWithDelay struct {
	serverWithTimeout
	lock                       sync.Mutex
	childEventCb               func(event Event)
	softTimeouts               map[interface{}]struct{}
	pendingCount, timeoutCount int
	parallelLimit              float32
	sendEvent                  bool
	delayTimer                 mclock.Timer
	delayCounter               int

	parallelAdjustUp, parallelAdjustDown, minParallelLimit float32
}

func (s *serverWithDelay) Subscribe(eventCallback func(event Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.childEventCb = eventCallback
	s.serverWithTimeout.Subscribe(s.eventCallback)
}

func (s *serverWithDelay) eventCallback(event Event) {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch event.Type {
	case EvSoftTimeout:
		s.softTimeouts[event.ReqId] = struct{}{}
		s.timeoutCount++
		s.parallelLimit -= s.parallelAdjustDown
		if s.parallelLimit < s.minParallelLimit {
			s.parallelLimit = s.minParallelLimit
		}
	case EvValidResponse, EvInvalidResponse, EvHardTimeout:
		if _, ok := s.softTimeouts[event.ReqId]; ok {
			delete(s.softTimeouts, event.ReqId)
			s.timeoutCount--
		}
		if event.Type == EvValidResponse && s.pendingCount >= int(s.parallelLimit) {
			s.parallelLimit -= s.parallelAdjustUp
		}
		s.pendingCount--
		s.canRequestNow() // send event if needed
	}
	s.childEventCb(event)
}

func (s *serverWithDelay) SendRequest(request interface{}) (reqId interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.pendingCount++
	return s.serverWithTimeout.SendRequest(request)
}

// stop stops all goroutines associated with the server.
func (s *serverWithDelay) Unsubscribe() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.delayTimer != nil {
		s.stopTimer(s.delayTimer)
		s.delayTimer = nil
	}
	s.childEventCb = nil
	s.serverWithTimeout.Unsubscribe()
}

func (s *serverWithDelay) canRequestNow() (bool, float32) {
	if s.delayTimer != nil || s.pendingCount >= int(s.parallelLimit) {
		return false, 0
	}
	if s.sendEvent {
		s.childEventCb(Event{Type: EvCanRequestAgain})
		s.sendEvent = false
	}
	if s.parallelLimit < s.minParallelLimit {
		s.parallelLimit = s.minParallelLimit
	}
	return true, -(float32(s.pendingCount) + rand.Float32()) / s.parallelLimit
}

// EvCanRequestAgain guaranteed if it returns false
func (s *serverWithDelay) CanRequestNow() (bool, float32) {
	s.lock.Lock()
	defer s.lock.Unlock()

	canSend, priority := s.canRequestNow()
	if !canSend {
		s.sendEvent = true
	}
	return canSend, priority
}

func (s *serverWithDelay) Delay(delay time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.delayTimer != nil {
		// Note: if stopping the timer is unsuccessful then the resulting AfterFunc
		// call will just do nothing
		s.stopTimer(s.delayTimer)
		s.delayTimer = nil
	}

	s.delayCounter++
	delayCounter := s.delayCounter
	s.delayTimer = s.clock.AfterFunc(delay, func() {
		/*if s.scheduler.testTimerResults != nil {
			s.scheduler.testTimerResults = append(s.scheduler.testTimerResults, true) // simulated timer finished
		}*/
		s.lock.Lock()
		if s.delayTimer != nil && s.delayCounter == delayCounter { // do nothing if there is a new timer now
			s.delayTimer = nil
			s.canRequestNow() // send event if necessary
		}
		s.lock.Unlock()
	})
}

//func (s *serverWithDelay) Fail()

type ServerSet map[Server]struct{}

// TryRequest tries to send the given request and returns true in case of success.
func (s *ServerSet) TryRequest(request interface{}) (Server, interface{}) {
	var (
		maxServerPriority, maxRequestPriority float32
		bestServer                            Server
	)
	for server, _ := range *s {
		canRequest, serverPriority := server.CanRequestNow()
		if !canRequest {
			delete(*s, server)
			continue
		}
		canSend, requestPriority := server.CanSendRequest(request)
		if !canSend || requestPriority < maxRequestPriority ||
			(requestPriority == maxRequestPriority && serverPriority <= maxServerPriority) {
			continue
		}
		maxServerPriority, maxRequestPriority = serverPriority, requestPriority
		bestServer = server
	}
	if bestServer != nil {
		return bestServer, bestServer.SendRequest(request)
	}
	return nil, nil
}
