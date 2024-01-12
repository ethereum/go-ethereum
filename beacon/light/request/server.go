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
	EvRequest  = &EventType{Name: "request", requestEvent: true}  // data: RequestResponse; sent by Scheduler
	EvResponse = &EventType{Name: "response", requestEvent: true} // data: RequestResponse; sent by requestServer
	EvFail     = &EventType{Name: "fail", requestEvent: true}     // data: RequestResponse; sent by requestServer
	EvTimeout  = &EventType{Name: "timeout", requestEvent: true}  // data: RequestResponse; sent by serverWithTimeout
	// server events
	EvRegistered      = &EventType{Name: "registered"}      // data: nil; sent by Scheduler
	EvUnregistered    = &EventType{Name: "unregistered"}    // data: nil; sent by Scheduler
	EvCanRequestAgain = &EventType{Name: "canRequestAgain"} // data: nil; sent by serverWithLimits
)

const (
	softRequestTimeout = time.Second
	hardRequestTimeout = time.Second * 10
)

const (
	// serverWithLimits parameters
	parallelAdjustUp     = 0.1                    // adjust parallelLimit up in case of success under full load
	parallelAdjustDown   = 1                      // adjust parallelLimit down in case of timeout/failure
	minParallelLimit     = 1                      // parallelLimit lower bound
	defaultParallelLimit = 3                      // parallelLimit initial value
	minFailureDelay      = time.Millisecond * 100 // minimum disable time in case of request failure
	maxFailureDelay      = time.Minute            // maximum disable time in case of request failure
)

// requestServer can send requests in a non-blocking way and feed back events
// through the event callback. After each request, it should send back either
// EvResponse or EvFail. Additionally, it may also send application-defined
// events that the Modules can interpret.
type requestServer interface {
	Subscribe(eventCallback func(event Event))
	SendRequest(request Request) ID
	Unsubscribe()
}

// server is implemented by a requestServer wrapped into serverWithTimeout and
// serverWithLimits and is used by Scheduler.
// In addition to requestServer functionality, server can also handle timeouts,
// limit the number of parallel in-flight requests and temporarily disable
// new requests based on timeouts and response failures.
type server interface {
	subscribe(eventCallback func(event Event))
	canRequestNow() (bool, float32)
	sendRequest(request Request) ID
	Fail(desc string)
	unsubscribe()
}

// newServer wraps a requestServer and returns a server
func newServer(rs requestServer, clock mclock.Clock) server {
	s := &serverWithLimits{}
	s.parent = rs
	s.serverWithTimeout.init(clock)
	s.init()
	return s
}

type serverSet map[server]struct{}

// EventType identifies an event type, either related to a request or the server
// in general. Server events can also be externally defined.
type EventType struct {
	Name         string
	requestEvent bool // all request events are pre-defined in request package
}

// Event describes an event where the type of Data depends on Type.
// Server field is not required when sent through the event callback; it is filled
// out when processed by the Scheduler. Note that the Scheduler can also create
// and send events (EvRegistered, EvUnregistered) directly.
type Event struct {
	Type   *EventType
	Server Server // filled by Scheduler
	Data   any
}

// IsRequestEvent returns true if the event is a request event
func (e *Event) IsRequestEvent() bool {
	return e.Type.requestEvent
}

// RequestInfo assumes that the event is a request event and returns its contents
// in a convenient form.
func (e *Event) RequestInfo() (ServerAndID, Request, Response) {
	data := e.Data.(RequestResponse)
	return ServerAndID{Server: e.Server, ID: data.ID}, data.Request, data.Response
}

// RequestResponse is the Data type of request events.
type RequestResponse struct {
	ID       ID
	Request  Request
	Response Response
}

// serverWithTimeout wraps a requestServer and implements timeouts. After
// softRequestTimeout it sends an EvTimeout after which and EvResponse or an
// EvFail will still follow (EvTimeout cannot follow the latter two).
// After hardRequestTimeout it sends an EvFail and blocks any further events
// related to the given request coming from the parent requestServer.
type serverWithTimeout struct {
	parent       requestServer
	lock         sync.Mutex
	clock        mclock.Clock
	childEventCb func(event Event)
	timeouts     map[ID]mclock.Timer
}

// init initializes serverWithTimeout
func (s *serverWithTimeout) init(clock mclock.Clock) {
	s.clock = clock
	s.timeouts = make(map[ID]mclock.Timer)
}

// subscribe subscribes to events which include parent (requestServer) events
// plus EvTimeout.
func (s *serverWithTimeout) subscribe(eventCallback func(event Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.childEventCb = eventCallback
	s.parent.Subscribe(s.eventCallback)
}

// eventCallback is called by parent (requestServer) event subscription.
func (s *serverWithTimeout) eventCallback(event Event) {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch event.Type {
	case EvResponse, EvFail:
		id := event.Data.(RequestResponse).ID
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

// sendRequest sends a request through the parent (requestServer) and starts a
// timer for request timeout.
func (s *serverWithTimeout) sendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	reqId = s.parent.SendRequest(request)
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
			childEventCb(Event{Type: EvFail, Data: RequestResponse{ID: reqId, Request: request}})
		})
		childEventCb := s.childEventCb
		s.lock.Unlock()
		childEventCb(Event{Type: EvTimeout, Data: RequestResponse{ID: reqId, Request: request}})
	})
	return reqId
}

// stop stops all goroutines associated with the server.
func (s *serverWithTimeout) unsubscribe() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, timer := range s.timeouts {
		if timer != nil {
			s.stopTimer(timer)
		}
	}
	s.childEventCb = nil
	s.parent.Unsubscribe()
}

// stopTimer stops the given timer
func (s *serverWithTimeout) stopTimer(timer mclock.Timer) {
	timer.Stop()
	/*if timer.Stop() && s.scheduler.testTimerResults != nil {
		s.scheduler.testTimerResults = append(s.scheduler.testTimerResults, false) // simulated timer stopped
	}*/
}

// serverWithLimits wraps serverWithTimeout and implements server. It limits the
// number of parallel in-flight requests and prevents sending new requests when a
// pending one has already timed out. It also implements a failure delay mechanism
// that adds an exponentially growing delay each time a request fails (wrong answer
// or hard timeout). This makes the syncing mechanism less brittle as temporary
// failures of the server might happen sometimes, but still avoids hammering a
// non-functional server with requests.
//
// TODO protect against excessive server events
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

// init initializes serverWithLimits
func (s *serverWithLimits) init() {
	s.softTimeouts = make(map[ID]struct{})
	s.parallelLimit = defaultParallelLimit
}

// subscribe subscribes to events which include parent (serverWithTimeout) events
// plus EvCanRequstAgain.
func (s *serverWithLimits) subscribe(eventCallback func(event Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.childEventCb = eventCallback
	s.serverWithTimeout.subscribe(s.eventCallback)
}

// eventCallback is called by parent (serverWithTimeout) event subscription.
func (s *serverWithLimits) eventCallback(event Event) {
	s.lock.Lock()
	var sendCanRequestAgain bool
	switch event.Type {
	case EvTimeout:
		id := event.Data.(RequestResponse).ID
		s.softTimeouts[id] = struct{}{}
		s.timeoutCount++
		s.parallelLimit -= parallelAdjustDown
		if s.parallelLimit < minParallelLimit {
			s.parallelLimit = minParallelLimit
		}
		log.Debug("Server timeout", "count", s.timeoutCount, "parallelLimit", s.parallelLimit)
	case EvResponse, EvFail:
		id := event.Data.(RequestResponse).ID
		if _, ok := s.softTimeouts[id]; ok {
			delete(s.softTimeouts, id)
			s.timeoutCount--
			log.Debug("Server timeout finalized", "count", s.timeoutCount, "parallelLimit", s.parallelLimit)
		}
		if event.Type == EvResponse && s.pendingCount >= int(s.parallelLimit) {
			s.parallelLimit += parallelAdjustUp
		}
		s.pendingCount--
		if canRequest, _ := s.canRequest(); canRequest {
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

// sendRequest sends a request through the parent (serverWithTimeout).
func (s *serverWithLimits) sendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.pendingCount++
	return s.serverWithTimeout.sendRequest(request)
}

// stop stops all goroutines associated with the server.
func (s *serverWithLimits) unsubscribe() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.delayTimer != nil {
		s.stopTimer(s.delayTimer)
		s.delayTimer = nil
	}
	s.childEventCb = nil
	s.serverWithTimeout.unsubscribe()
}

// canRequest checks whether a new request can be started.
func (s *serverWithLimits) canRequest() (bool, float32) {
	if s.delayTimer != nil || s.pendingCount >= int(s.parallelLimit) {
		return false, 0
	}
	if s.parallelLimit < minParallelLimit {
		s.parallelLimit = minParallelLimit
	}
	return true, -(float32(s.pendingCount) + rand.Float32()) / s.parallelLimit
}

// canRequestNow checks whether a new request can be started, according to the
// current in-flight request count and parallelLimit, and also the failure delay
// timer.
// If a new request is allowed then it also returns a priority value that can be
// used to select the least overloaded server from an otherwise equally suitable
// set of servers.
// If it returns false then it is guaranteed that an EvCanRequestAgain will be
// sent whenever the server becomes available for requesting again.
func (s *serverWithLimits) canRequestNow() (bool, float32) {
	var sendCanRequestAgain bool
	s.lock.Lock()
	canRequest, priority := s.canRequest()
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

// delay sets the delay timer to the given duration, disabling new requests for
// the given period.
func (s *serverWithLimits) delay(delay time.Duration) {
	if s.delayTimer != nil {
		// Note: if stopping the timer is unsuccessful then the resulting AfterFunc
		// call will just do nothing
		s.stopTimer(s.delayTimer)
		s.delayTimer = nil
	}

	s.delayCounter++
	delayCounter := s.delayCounter
	log.Debug("Server delay started", "length", delay)
	s.delayTimer = s.clock.AfterFunc(delay, func() {
		log.Debug("Server delay ended", "length", delay)
		/*if s.scheduler.testTimerResults != nil {
			s.scheduler.testTimerResults = append(s.scheduler.testTimerResults, true) // simulated timer finished
		}*/
		var sendCanRequestAgain bool
		s.lock.Lock()
		if s.delayTimer != nil && s.delayCounter == delayCounter { // do nothing if there is a new timer now
			s.delayTimer = nil
			if canRequest, _ := s.canRequest(); canRequest {
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

// fail reports that a response from the server was found invalid by the processing
// Module, disabling new requests for a dynamically adjused time period.
func (s *serverWithLimits) Fail(desc string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.fail(desc)
}

// fail calculates the dynamic failure delay and applies it.
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
