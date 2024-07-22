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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// request events
	EvResponse = &EventType{Name: "response", requestEvent: true} // data: RequestResponse; sent by requestServer
	EvFail     = &EventType{Name: "fail", requestEvent: true}     // data: RequestResponse; sent by requestServer
	EvTimeout  = &EventType{Name: "timeout", requestEvent: true}  // data: RequestResponse; sent by serverWithTimeout
	// server events
	EvRegistered      = &EventType{Name: "registered"}      // data: nil; sent by Scheduler
	EvUnregistered    = &EventType{Name: "unregistered"}    // data: nil; sent by Scheduler
	EvCanRequestAgain = &EventType{Name: "canRequestAgain"} // data: nil; sent by serverWithLimits
)

const (
	softRequestTimeout = time.Second      // allow resending request to a different server but do not cancel yet
	hardRequestTimeout = time.Second * 10 // cancel request
)

const (
	// serverWithLimits parameters
	parallelAdjustUp     = 0.1                    // adjust parallelLimit up in case of success under full load
	parallelAdjustDown   = 1                      // adjust parallelLimit down in case of timeout/failure
	minParallelLimit     = 1                      // parallelLimit lower bound
	defaultParallelLimit = 3                      // parallelLimit initial value
	minFailureDelay      = time.Millisecond * 100 // minimum disable time in case of request failure
	maxFailureDelay      = time.Minute            // maximum disable time in case of request failure
	maxServerEventBuffer = 5                      // server event allowance buffer limit
	maxServerEventRate   = time.Second            // server event allowance buffer recharge rate
)

// requestServer can send requests in a non-blocking way and feed back events
// through the event callback. After each request it should send back either
// EvResponse or EvFail. Additionally, it may also send application-defined
// events that the Modules can interpret.
type requestServer interface {
	Name() string
	Subscribe(eventCallback func(Event))
	SendRequest(ID, Request)
	Unsubscribe()
}

// server is implemented by a requestServer wrapped into serverWithTimeout and
// serverWithLimits and is used by Scheduler.
// In addition to requestServer functionality, server can also handle timeouts,
// limit the number of parallel in-flight requests and temporarily disable
// new requests based on timeouts and response failures.
type server interface {
	Server
	subscribe(eventCallback func(Event))
	canRequestNow() bool
	sendRequest(Request) ID
	fail(string)
	unsubscribe()
}

// NewServer wraps a requestServer and returns a server
func NewServer(rs requestServer, clock mclock.Clock) server {
	s := &serverWithLimits{}
	s.parent = rs
	s.serverWithTimeout.init(clock)
	s.init()
	return s
}

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

// serverWithTimeout wraps a requestServer and introduces timeouts.
// The request's lifecycle is concluded if EvResponse or EvFail emitted by the
// parent requestServer. If this does not happen until softRequestTimeout then
// EvTimeout is emitted, after which the final EvResponse or EvFail is still
// guaranteed to follow.
// If the parent fails to send this final event for hardRequestTimeout then
// serverWithTimeout emits EvFail and discards any further events from the
// parent related to the given request.
type serverWithTimeout struct {
	parent       requestServer
	lock         sync.Mutex
	clock        mclock.Clock
	childEventCb func(event Event)
	timeouts     map[ID]mclock.Timer
	lastID       ID
}

// Name implements request.Server
func (s *serverWithTimeout) Name() string {
	return s.parent.Name()
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

// sendRequest generated a new request ID, emits EvRequest, sets up the timeout
// timer, then sends the request through the parent (requestServer).
func (s *serverWithTimeout) sendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	s.lastID++
	id := s.lastID
	s.startTimeout(RequestResponse{ID: id, Request: request})
	s.lock.Unlock()
	s.parent.SendRequest(id, request)
	return id
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
			timer.Stop()
			delete(s.timeouts, id)
			if s.childEventCb != nil {
				s.childEventCb(event)
			}
		}
	default:
		if s.childEventCb != nil {
			s.childEventCb(event)
		}
	}
}

// startTimeout starts a timeout timer for the given request.
func (s *serverWithTimeout) startTimeout(reqData RequestResponse) {
	id := reqData.ID
	s.timeouts[id] = s.clock.AfterFunc(softRequestTimeout, func() {
		s.lock.Lock()
		if _, ok := s.timeouts[id]; !ok {
			s.lock.Unlock()
			return
		}
		s.timeouts[id] = s.clock.AfterFunc(hardRequestTimeout-softRequestTimeout, func() {
			s.lock.Lock()
			if _, ok := s.timeouts[id]; !ok {
				s.lock.Unlock()
				return
			}
			delete(s.timeouts, id)
			childEventCb := s.childEventCb
			s.lock.Unlock()
			if childEventCb != nil {
				childEventCb(Event{Type: EvFail, Data: reqData})
			}
		})
		childEventCb := s.childEventCb
		s.lock.Unlock()
		if childEventCb != nil {
			childEventCb(Event{Type: EvTimeout, Data: reqData})
		}
	})
}

// unsubscribe stops all goroutines associated with the server.
func (s *serverWithTimeout) unsubscribe() {
	s.lock.Lock()
	for _, timer := range s.timeouts {
		if timer != nil {
			timer.Stop()
		}
	}
	s.lock.Unlock()
	s.parent.Unsubscribe()
}

// serverWithLimits wraps serverWithTimeout and implements server. It limits the
// number of parallel in-flight requests and prevents sending new requests when a
// pending one has already timed out. Server events are also rate limited.
// It also implements a failure delay mechanism that adds an exponentially growing
// delay each time a request fails (wrong answer or hard timeout). This makes the
// syncing mechanism less brittle as temporary failures of the server might happen
// sometimes, but still avoids hammering a non-functional server with requests.
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
	serverEventBuffer          int
	eventBufferUpdated         mclock.AbsTime
}

// init initializes serverWithLimits
func (s *serverWithLimits) init() {
	s.softTimeouts = make(map[ID]struct{})
	s.parallelLimit = defaultParallelLimit
	s.serverEventBuffer = maxServerEventBuffer
}

// subscribe subscribes to events which include parent (serverWithTimeout) events
// plus EvCanRequestAgain.
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
	passEvent := true
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
		if s.canRequest() {
			sendCanRequestAgain = s.sendEvent
			s.sendEvent = false
		}
		if event.Type == EvFail {
			s.failLocked("failed request")
		}
	default:
		// server event; check rate limit
		if s.serverEventBuffer < maxServerEventBuffer {
			now := s.clock.Now()
			sinceUpdate := time.Duration(now - s.eventBufferUpdated)
			if sinceUpdate >= maxServerEventRate*time.Duration(maxServerEventBuffer-s.serverEventBuffer) {
				s.serverEventBuffer = maxServerEventBuffer
				s.eventBufferUpdated = now
			} else {
				addBuffer := int(sinceUpdate / maxServerEventRate)
				s.serverEventBuffer += addBuffer
				s.eventBufferUpdated += mclock.AbsTime(maxServerEventRate * time.Duration(addBuffer))
			}
		}
		if s.serverEventBuffer > 0 {
			s.serverEventBuffer--
		} else {
			passEvent = false
		}
	}
	childEventCb := s.childEventCb
	s.lock.Unlock()
	if passEvent && childEventCb != nil {
		childEventCb(event)
	}
	if sendCanRequestAgain && childEventCb != nil {
		childEventCb(Event{Type: EvCanRequestAgain})
	}
}

// sendRequest sends a request through the parent (serverWithTimeout).
func (s *serverWithLimits) sendRequest(request Request) (reqId ID) {
	s.lock.Lock()
	s.pendingCount++
	s.lock.Unlock()
	return s.serverWithTimeout.sendRequest(request)
}

// unsubscribe stops all goroutines associated with the server.
func (s *serverWithLimits) unsubscribe() {
	s.lock.Lock()
	if s.delayTimer != nil {
		s.delayTimer.Stop()
		s.delayTimer = nil
	}
	s.childEventCb = nil
	s.lock.Unlock()
	s.serverWithTimeout.unsubscribe()
}

// canRequest checks whether a new request can be started.
func (s *serverWithLimits) canRequest() bool {
	if s.delayTimer != nil || s.pendingCount >= int(s.parallelLimit) || s.timeoutCount > 0 {
		return false
	}
	if s.parallelLimit < minParallelLimit {
		s.parallelLimit = minParallelLimit
	}
	return true
}

// canRequestNow checks whether a new request can be started, according to the
// current in-flight request count and parallelLimit, and also the failure delay
// timer.
// If it returns false then it is guaranteed that an EvCanRequestAgain will be
// sent whenever the server becomes available for requesting again.
func (s *serverWithLimits) canRequestNow() bool {
	var sendCanRequestAgain bool
	s.lock.Lock()
	canRequest := s.canRequest()
	if canRequest {
		sendCanRequestAgain = s.sendEvent
		s.sendEvent = false
	}
	childEventCb := s.childEventCb
	s.lock.Unlock()
	if sendCanRequestAgain && childEventCb != nil {
		childEventCb(Event{Type: EvCanRequestAgain})
	}
	return canRequest
}

// delay sets the delay timer to the given duration, disabling new requests for
// the given period.
func (s *serverWithLimits) delay(delay time.Duration) {
	if s.delayTimer != nil {
		// Note: if stopping the timer is unsuccessful then the resulting AfterFunc
		// call will just do nothing
		s.delayTimer.Stop()
		s.delayTimer = nil
	}

	s.delayCounter++
	delayCounter := s.delayCounter
	log.Debug("Server delay started", "length", delay)
	s.delayTimer = s.clock.AfterFunc(delay, func() {
		log.Debug("Server delay ended", "length", delay)
		var sendCanRequestAgain bool
		s.lock.Lock()
		if s.delayTimer != nil && s.delayCounter == delayCounter { // do nothing if there is a new timer now
			s.delayTimer = nil
			if s.canRequest() {
				sendCanRequestAgain = s.sendEvent
				s.sendEvent = false
			}
		}
		childEventCb := s.childEventCb
		s.lock.Unlock()
		if sendCanRequestAgain && childEventCb != nil {
			childEventCb(Event{Type: EvCanRequestAgain})
		}
	})
}

// fail reports that a response from the server was found invalid by the processing
// Module, disabling new requests for a dynamically adjusted time period.
func (s *serverWithLimits) fail(desc string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.failLocked(desc)
}

// failLocked calculates the dynamic failure delay and applies it.
func (s *serverWithLimits) failLocked(desc string) {
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
