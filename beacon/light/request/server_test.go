// Copyright 2024 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common/mclock"
)

const (
	testRequest  = "Life, the Universe, and Everything"
	testResponse = 42
)

var testEventType = &EventType{Name: "testEvent"}

func TestServerEvents(t *testing.T) {
	rs := &testRequestServer{}
	clock := &mclock.Simulated{}
	srv := NewServer(rs, clock)
	var lastEventType *EventType
	srv.subscribe(func(event Event) { lastEventType = event.Type })
	evTypeName := func(evType *EventType) string {
		if evType == nil {
			return "none"
		}
		return evType.Name
	}
	expEvent := func(expType *EventType) {
		if lastEventType != expType {
			t.Errorf("Wrong event type (expected %s, got %s)", evTypeName(expType), evTypeName(lastEventType))
		}
		lastEventType = nil
	}
	// user events should simply be passed through
	rs.eventCb(Event{Type: testEventType})
	expEvent(testEventType)
	// send request, soft timeout, then valid response
	srv.sendRequest(testRequest)
	clock.WaitForTimers(1)
	clock.Run(softRequestTimeout)
	expEvent(EvTimeout)
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 1, Request: testRequest, Response: testResponse}})
	expEvent(EvResponse)
	// send request, hard timeout (response after hard timeout should be ignored)
	srv.sendRequest(testRequest)
	clock.WaitForTimers(1)
	clock.Run(softRequestTimeout)
	expEvent(EvTimeout)
	clock.WaitForTimers(1)
	clock.Run(hardRequestTimeout)
	expEvent(EvFail)
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 1, Request: testRequest, Response: testResponse}})
	expEvent(nil)
	srv.unsubscribe()
}

func TestServerParallel(t *testing.T) {
	rs := &testRequestServer{}
	srv := NewServer(rs, &mclock.Simulated{})
	srv.subscribe(func(event Event) {})

	expSend := func(expSent int) {
		var sent int
		for sent <= expSent {
			if !srv.canRequestNow() {
				break
			}
			sent++
			srv.sendRequest(testRequest)
		}
		if sent != expSent {
			t.Errorf("Wrong number of parallel requests accepted (expected %d, got %d)", expSent, sent)
		}
	}
	// max out parallel allowance
	expSend(defaultParallelLimit)
	// 1 answered, should accept 1 more
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 1, Request: testRequest, Response: testResponse}})
	expSend(1)
	// 2 answered, should accept 2 more
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 2, Request: testRequest, Response: testResponse}})
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 3, Request: testRequest, Response: testResponse}})
	expSend(2)
	// failed request, should decrease allowance and not accept more
	rs.eventCb(Event{Type: EvFail, Data: RequestResponse{ID: 4, Request: testRequest}})
	expSend(0)
	srv.unsubscribe()
}

func TestServerFail(t *testing.T) {
	rs := &testRequestServer{}
	clock := &mclock.Simulated{}
	srv := NewServer(rs, clock)
	srv.subscribe(func(event Event) {})
	expCanRequest := func(expCanRequest bool) {
		if canRequest := srv.canRequestNow(); canRequest != expCanRequest {
			t.Errorf("Wrong result for canRequestNow (expected %v, got %v)", expCanRequest, canRequest)
		}
	}
	// timed out request
	expCanRequest(true)
	srv.sendRequest(testRequest)
	clock.WaitForTimers(1)
	expCanRequest(true)
	clock.Run(softRequestTimeout)
	expCanRequest(false) // cannot request when there is a timed out request
	rs.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 1, Request: testRequest, Response: testResponse}})
	expCanRequest(true)
	// explicit server.Fail
	srv.fail("")
	clock.WaitForTimers(1)
	expCanRequest(false) // cannot request for a while after a failure
	clock.Run(minFailureDelay)
	expCanRequest(true)
	// request returned with EvFail
	srv.sendRequest(testRequest)
	rs.eventCb(Event{Type: EvFail, Data: RequestResponse{ID: 2, Request: testRequest}})
	clock.WaitForTimers(1)
	expCanRequest(false) // EvFail should also start failure delay
	clock.Run(minFailureDelay)
	expCanRequest(false) // second failure delay is longer, should still be disabled
	clock.Run(minFailureDelay)
	expCanRequest(true)
	srv.unsubscribe()
}

func TestServerEventRateLimit(t *testing.T) {
	rs := &testRequestServer{}
	clock := &mclock.Simulated{}
	srv := NewServer(rs, clock)
	var eventCount int
	srv.subscribe(func(event Event) {
		eventCount++
	})
	expEvents := func(send, expAllowed int) {
		eventCount = 0
		for sent := 0; sent < send; sent++ {
			rs.eventCb(Event{Type: testEventType})
		}
		if eventCount != expAllowed {
			t.Errorf("Wrong number of server events passing rate limitation (sent %d, expected %d, got %d)", send, expAllowed, eventCount)
		}
	}
	expEvents(maxServerEventBuffer+5, maxServerEventBuffer)
	clock.Run(maxServerEventRate)
	expEvents(5, 1)
	clock.Run(maxServerEventRate * maxServerEventBuffer * 2)
	expEvents(maxServerEventBuffer+5, maxServerEventBuffer)
	srv.unsubscribe()
}

func TestServerUnsubscribe(t *testing.T) {
	rs := &testRequestServer{}
	clock := &mclock.Simulated{}
	srv := NewServer(rs, clock)
	var eventCount int
	srv.subscribe(func(event Event) {
		eventCount++
	})
	eventCb := rs.eventCb
	eventCb(Event{Type: testEventType})
	if eventCount != 1 {
		t.Errorf("Server event callback not called before unsubscribe")
	}
	srv.unsubscribe()
	if rs.eventCb != nil {
		t.Errorf("Server event callback not removed after unsubscribe")
	}
	eventCb(Event{Type: testEventType})
	if eventCount != 1 {
		t.Errorf("Server event callback called after unsubscribe")
	}
}

type testRequestServer struct {
	eventCb func(Event)
}

func (rs *testRequestServer) Name() string                  { return "" }
func (rs *testRequestServer) Subscribe(eventCb func(Event)) { rs.eventCb = eventCb }
func (rs *testRequestServer) SendRequest(ID, Request)       {}
func (rs *testRequestServer) Unsubscribe()                  { rs.eventCb = nil }
