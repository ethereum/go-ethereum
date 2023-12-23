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
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// request events
	EvResponse = "response" // data: IdAndResponse; sent by RequestServer
	EvFail     = "fail"     // data: ID; sent by RequestServer
	EvTimeout  = "timeout"  // data: ID; sent by serverWithTimeout
	// server events
	EvRegistered      = "registered"      // data: nil; sent by Scheduler
	EvUnregistered    = "unregistered"    // data: nil; sent by Scheduler
	EvCanRequestAgain = "canRequestAgain" // data: nil; sent by serverWithLimits
)

const (
	softRequestTimeout = time.Second
	hardRequestTimeout = time.Second * 10
)

const (
	parallelAdjustUp     = 0.1
	parallelAdjustDown   = 1
	minParallelLimit     = 1
	defaultParallelLimit = 3
	minFailureDelay      = time.Millisecond * 100
	maxFailureDelay      = time.Minute
)

// RequestServer can send a set of requests pre-defined by the application and
// signal events through the event callback. After each request, it should send
// back either EvResponse or EvFail. Additionally, it may also send application-
// defined events that the Modules can interpret.
type RequestServer interface {
	Subscribe(eventCallback func(event Event))
	SendRequest(request Request) ID
	Unsubscribe()
}

type Server interface {
	RequestServer
	CanRequestNow() (bool, float32)
	Fail(desc string)
}

func NewServer(rs RequestServer, clock mclock.Clock) Server {
	s := &serverWithLimits{}
	s.serverWithTimeout.RequestServer = rs
	s.serverWithTimeout.init(clock)
	s.init()
	return s
}

type serverSet map[Server]struct{}

type Event struct {
	Type string
	Data any
}

type IdAndResponse struct {
	ID       ID
	Response Response
}

// serverWithTimeout wraps a RequestServer and implements timeouts. After
// softRequestTimeout it sends an EvTimeout after which and EvResponse or an
// EvFail will still follow (EvTimeout cannot follow the latter two).
// After hardRequestTimeout it sends an EvFail and blocks any further events
// related to the given request coming from the parent RequestServer.
type serverWithTimeout struct {
	RequestServer
	lock         sync.Mutex
	clock        mclock.Clock
	childEventCb func(event Event)
	timeouts     map[ID]mclock.Timer
}

func (s *serverWithTimeout) init(clock mclock.Clock) {
	s.clock = clock
	s.timeouts = make(map[ID]mclock.Timer)
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
	case EvResponse, EvFail:
		var id ID
		if event.Type == EvResponse {
			id = event.Data.(IdAndResponse).ID
		} else {
			id = event.Data.(ID)
		}
		if timer, ok := s.timeouts[id]; ok {
			// Note: if stopping the timer is unsuccessful then the resulting AfterFunc
			// call will just do nothing
			s.stopTimer(timer)
			delete(s.timeouts, id)
			s.childEventCb(event)
		}
	default:
		s.childEventCb(event)
	}
}

func (s *serverWithTimeout) SendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	reqId = s.RequestServer.SendRequest(request)
	s.timeouts[reqId] = s.clock.AfterFunc(softRequestTimeout, func() {
		/*if s.testTimerResults != nil {
			s.testTimerResults = append(s.testTimerResults, true) // simulated timer finished
		}*/
		s.lock.Lock()
		if _, ok := s.timeouts[reqId]; !ok {
			s.lock.Unlock()
			return
		}
		s.timeouts[reqId] = s.clock.AfterFunc(hardRequestTimeout-softRequestTimeout, func() {
			/*if s.testTimerResults != nil {
				s.testTimerResults = append(s.testTimerResults, true) // simulated timer finished
			}*/
			s.lock.Lock()
			if _, ok := s.timeouts[reqId]; !ok {
				s.lock.Unlock()
				return
			}
			delete(s.timeouts, reqId)
			childEventCb := s.childEventCb
			s.lock.Unlock()
			childEventCb(Event{Type: EvFail, Data: reqId})
		})
		childEventCb := s.childEventCb
		s.lock.Unlock()
		childEventCb(Event{Type: EvTimeout, Data: reqId})
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

// serverWithLimits wraps serverWithTimeout and implements Server. It limits the
// number of parallel in-flight requests and prevents sending new requests when a
// pending one has already timed out. It also implements a failure delay mechanism
// that adds an exponentially growing delay each time a request fails (wrong answer
// or hard timeout). This makes the syncing mechanism less brittle as temporary
// failures of the server might happen sometimes, but still avoids hammering a
// non-functional server with requests.
type serverWithLimits struct {
	serverWithTimeout
	lock                       sync.Mutex
	childEventCb               func(event Event)
	softTimeouts               map[ID]struct{}
	pendingCount, timeoutCount int
	parallelLimit              float32
	sendEvent                  bool
	delayTimer                 mclock.Timer
	delayCounter               int
	failureDelayEnd            mclock.AbsTime
	failureDelay               float64
}

func (s *serverWithLimits) init() {
	s.softTimeouts = make(map[ID]struct{})
	s.parallelLimit = defaultParallelLimit
}

func (s *serverWithLimits) Subscribe(eventCallback func(event Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.childEventCb = eventCallback
	s.serverWithTimeout.Subscribe(s.eventCallback)
}

func (s *serverWithLimits) eventCallback(event Event) {
	s.lock.Lock()
	var sendCanRequestAgain bool
	switch event.Type {
	case EvTimeout:
		s.softTimeouts[event.Data.(ID)] = struct{}{}
		s.timeoutCount++
		s.parallelLimit -= parallelAdjustDown
		if s.parallelLimit < minParallelLimit {
			s.parallelLimit = minParallelLimit
		}
	case EvResponse, EvFail:
		var id ID
		if event.Type == EvResponse {
			id = event.Data.(IdAndResponse).ID
		} else {
			id = event.Data.(ID)
		}
		if _, ok := s.softTimeouts[id]; ok {
			delete(s.softTimeouts, id)
			s.timeoutCount--
		}
		if event.Type == EvResponse && s.pendingCount >= int(s.parallelLimit) {
			s.parallelLimit -= parallelAdjustUp
		}
		s.pendingCount--
		if canRequest, _ := s.canRequestNow(); canRequest {
			sendCanRequestAgain = s.sendEvent
			s.sendEvent = false
		}
		if event.Type == EvFail {
			s.fail("failed request")
		}
	}
	childEventCb := s.childEventCb
	s.lock.Unlock()
	childEventCb(event)
	if sendCanRequestAgain {
		childEventCb(Event{Type: EvCanRequestAgain})
	}
}

func (s *serverWithLimits) SendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.pendingCount++
	id := s.serverWithTimeout.SendRequest(request)
	return id
}

// stop stops all goroutines associated with the server.
func (s *serverWithLimits) Unsubscribe() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.delayTimer != nil {
		s.stopTimer(s.delayTimer)
		s.delayTimer = nil
	}
	s.childEventCb = nil
	s.serverWithTimeout.Unsubscribe()
}

func (s *serverWithLimits) canRequestNow() (bool, float32) {
	if s.delayTimer != nil || s.pendingCount >= int(s.parallelLimit) {
		return false, 0
	}
	if s.parallelLimit < minParallelLimit {
		s.parallelLimit = minParallelLimit
	}
	return true, -(float32(s.pendingCount) + rand.Float32()) / s.parallelLimit
}

// EvCanRequestAgain guaranteed if it returns false
func (s *serverWithLimits) CanRequestNow() (bool, float32) {
	var sendCanRequestAgain bool
	s.lock.Lock()
	canRequest, priority := s.canRequestNow()
	if canRequest {
		sendCanRequestAgain = s.sendEvent
		s.sendEvent = false
	}
	childEventCb := s.childEventCb
	s.lock.Unlock()
	if sendCanRequestAgain {
		childEventCb(Event{Type: EvCanRequestAgain})
	}
	return canRequest, priority
}

func (s *serverWithLimits) delay(delay time.Duration) {
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
		var sendCanRequestAgain bool
		s.lock.Lock()
		if s.delayTimer != nil && s.delayCounter == delayCounter { // do nothing if there is a new timer now
			s.delayTimer = nil
			if canRequest, _ := s.canRequestNow(); canRequest {
				sendCanRequestAgain = s.sendEvent
				s.sendEvent = false
			}
		}
		childEventCb := s.childEventCb
		s.lock.Unlock()
		if sendCanRequestAgain {
			childEventCb(Event{Type: EvCanRequestAgain})
		}
	})
}

func (s *serverWithLimits) Fail(desc string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.fail(desc)
}

func (s *serverWithLimits) fail(desc string) {
	log.Debug("Server error", "description", desc)
	s.failureDelay *= 2
	now := s.clock.Now()
	if now > s.failureDelayEnd {
		s.failureDelay *= math.Pow(2, -float64(now-s.failureDelayEnd)/float64(maxFailureDelay))
	}
	if s.failureDelay < float64(minFailureDelay) {
		s.failureDelay = float64(minFailureDelay)
	}
	s.failureDelayEnd = now + mclock.AbsTime(s.failureDelay)
	s.delay(time.Duration(s.failureDelay))
}
