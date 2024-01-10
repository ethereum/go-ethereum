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

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
)

// Module represents a mechanism which is typically responsible for downloading
// and updating a passive data structure.
// Modules can start network requests through Tracker and receive request events
// related to the sent requests that can signal a response, a failure or a timeout.
// They also receive server-related events. Note that they do not directly interact
// with servers but may keep track of certain parameters of registered servers,
// based on the received server events. These server parameters may affect the
// possible range of requests to be sent to a given server.
// Modules are called by Scheduler whenever a global trigger is fired. All request
// and server events fire the trigger. Modules themselves can also self-trigger,
// ensuring an immediate next processing round after the target data structure has
// been changed in a way that could make further actions possible either by the
// same or another Module.
type Module interface {
	// Process is a non-blocking function that is called on each Module whenever
	// a processing round is triggered. It can start new requests through the
	// received Tracker, process events related to servers and previosly sent
	// requests and/or do other data processing tasks. Note that request events
	// are only passed to the module that made the given request while server
	// events are passed to every module. Process can also trigger a next
	// processing round by returning true.
	//
	// Note: Process functions of different modules are never called concurrently;
	// they are called by Scheduler in the same order of priority as they were
	// registered in.
	Process(Tracker, []Event) bool
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or connected servers could allow new operations.
type Scheduler struct {
	lock         sync.Mutex
	clock        mclock.Clock
	modules      []Module // first has highest priority
	names        map[Module]string
	trackers     map[Module]*tracker
	servers      map[server]struct{}
	pending      map[ServerAndID]pendingRequest
	serverEvents []Event
	stopCh       chan chan struct{}

	triggerCh chan struct{} // restarts waiting sync loop
	//	testWaitCh       chan struct{} // accepts sends when sync loop is waiting
	//	testTimerResults []bool        // true is appended when simulated timer is processed; false when stopped
}

// pendingRequest keeps track of sent and not finalized requests and their sender
// modules and whether a soft timeout has already happened.
type pendingRequest struct {
	request Request
	module  Module
}

// NewScheduler creates a new Scheduler.
func NewScheduler(clock mclock.Clock) *Scheduler {
	s := &Scheduler{
		clock:    clock,
		servers:  make(map[server]struct{}),
		names:    make(map[Module]string),
		trackers: make(map[Module]*tracker),
		pending:  make(map[ServerAndID]pendingRequest),
		stopCh:   make(chan chan struct{}),
		// Note: testWaitCh should not have capacity in order to ensure
		// that after a trigger happens testWaitCh will block until the resulting
		// processing round has been finished
		triggerCh: make(chan struct{}, 1),
		//testWaitCh: make(chan struct{}),
	}
	return s
}

// RegisterModule registers a module. Should be called before starting the scheduler.
// In each processing round the order of module processing depends on the order of
// registration.
func (s *Scheduler) RegisterModule(m Module, name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
	s.trackers[m] = &tracker{
		scheduler: s,
		module:    m,
	}
	s.names[m] = name
}

// RegisterServer registers a new server.
func (s *Scheduler) RegisterServer(rs requestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server := newServer(rs, s.clock)
	s.handleEvent(Event{Type: EvRegistered, Server: server})
	server.subscribe(func(event Event) {
		s.lock.Lock()
		if _, ok := s.servers[server]; ok {
			event.Server = server
			s.handleEvent(event)
		} else {
			log.Error("Event received from unsubscribed server")
		}
		s.lock.Unlock()
	})
	s.servers[server] = struct{}{}
}

// UnregisterServer removes a registered server.
func (s *Scheduler) UnregisterServer(rs requestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for server := range s.servers {
		if sl, ok := server.(*serverWithLimits); ok && sl.parent == rs {
			server.unsubscribe()
			delete(s.servers, server)
			s.handleEvent(Event{Type: EvUnregistered, Server: server})
			return
		}
	}
}

// Start starts the scheduler. It should be called after registering all modules
// and before registering any servers.
func (s *Scheduler) Start() {
	go s.syncLoop()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.lock.Lock()
	for server, _ := range s.servers {
		server.unsubscribe()
	}
	s.servers = nil
	s.lock.Unlock()
	stop := make(chan struct{})
	s.stopCh <- stop
	<-stop
}

// syncLoop calls all modules in the order of their registration.
// A round of processing starts whenever the global trigger is fired. Triggers
// fired during a processing round ensure that there is going to be a next round.
func (s *Scheduler) syncLoop() {
	for {
		s.processModules()
	loop:
		for {
			select {
			case stop := <-s.stopCh:
				close(stop)
				return
			case <-s.triggerCh:
				break loop
				//case <-s.testWaitCh:
			}
		}
	}
}

// processModules runs an entire processing round, calling the process functions
// of all modules, passing all relevant events.
func (s *Scheduler) processModules() {
	s.lock.Lock()
	servers := make(serverSet)
	for server, _ := range s.servers {
		if ok, _ := server.canRequestNow(); ok {
			servers[server] = struct{}{}
		}
	}
	serverEvents := s.serverEvents
	s.serverEvents = nil
	s.lock.Unlock()

	eventTypes := make([]string, len(serverEvents))
	for i, ev := range serverEvents {
		eventTypes[i] = ev.Type.Name
	}
	log.Debug("Processing modules", "servers", len(servers), "server events", eventTypes)

	for _, module := range s.modules {
		s.lock.Lock()
		tracker := s.trackers[module]
		tracker.servers = servers
		requestEvents := tracker.requestEvents
		tracker.requestEvents = nil
		s.lock.Unlock()

		var respCount, failCount, timeoutCount int
		for _, ev := range requestEvents {
			switch ev.Type {
			case EvResponse:
				respCount++
			case EvFail:
				failCount++
			case EvTimeout:
				timeoutCount++
			}
		}
		log.Debug("Processing module", "name", s.names[module], "responses", respCount, "fails", failCount, "timeouts", timeoutCount)

		if module.Process(tracker, append(serverEvents, requestEvents...)) {
			s.Trigger()
		}
	}
}

// Trigger starts a new processing round. If fired during processing, it ensures
// another full round of processing all modules.
func (s *Scheduler) Trigger() {
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

// addRequestEvent adds a request event to the sender module's Tracker, ensuring
// that the module receives it in the next processing round.
func (s *Scheduler) addRequestEvent(event Event) {
	sid, _, _ := event.RequestInfo()
	if pr, ok := s.pending[sid]; ok {
		tracker := s.trackers[pr.module]
		tracker.requestEvents = append(tracker.requestEvents, event)
		if event.Type != EvTimeout {
			delete(s.pending, sid)
		}
	}
}

// addServerEvent adds a server event to the global server event list, ensuring
// that all modules receive it in the next processing round.
func (s *Scheduler) addServerEvent(event Event) {
	s.serverEvents = append(s.serverEvents, event)
}

// handleEvent processes an Event and adds it either as a request event or a
// server event, depending on its type. In case of an EvUnregistered server event
// it also closes all pending requests to the given server by emitting a failed
// request event (Finalized without Response), ensuring that all requests get
// finalized and thereby allowing the module logic to be safe and simple.
func (s *Scheduler) handleEvent(event Event) {
	s.Trigger()
	if event.IsRequestEvent() {
		s.addRequestEvent(event)
		return
	}
	if event.Type == EvUnregistered {
		for id, pending := range s.pending {
			if id.Server != event.Server {
				continue
			}
			s.addRequestEvent(Event{
				Type:   EvFail,
				Server: event.Server,
				Data: RequestResponse{
					ID:      id.ID,
					Request: pending.request,
				},
			})
		}
	}
	s.addServerEvent(event)
}
