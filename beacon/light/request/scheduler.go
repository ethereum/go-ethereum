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

	"github.com/ethereum/go-ethereum/log"
)

// Module represents a mechanism which is typically responsible for downloading
// and updating a passive data structure. It does not directly interact with the
// servers. It can start requests using the Requester interface, maintain its
// internal state by receiving and processing Events and update its target data
// structure based on the obtained data.
// It is the Scheduler's responsibility to feed events to the modules, call
// Process as long as there might be something to process and then generate request
// candidates using MakeRequest and start the best possible requests.
// Modules are called by Scheduler whenever a global trigger is fired. All events
// fire the trigger. Changing a target data structure also triggers a next
// processing round as it could make further actions possible either by the same
// or another Module.
type Module interface {
	// Process is a non-blocking function responsible for starting requests,
	// processing events and updating the target data structures(s) and the
	// internal state of the module. Module state typically consists of information
	// about pending requests and registered servers.
	// Process is always called after an event is received or after a target data
	// structure has been changed.
	//
	// Note: Process functions of different modules are never called concurrently;
	// they are called by Scheduler in the same order of priority as they were
	// registered in.
	Process(Requester, []Event)
}

// Requester allows Modules to obtain the list of momentarily available servers,
// start new requests and report server failure when a response has been proven
// to be invalid in the processing phase.
// Note that all Requester functions should be safe to call from Module.Process.
type Requester interface {
	CanSendTo() []Server
	Send(Server, Request) ID
	Fail(Server, string)
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or events coming from registered servers could
// allow new operations.
type Scheduler struct {
	lock    sync.Mutex
	modules []Module // first has the highest priority
	names   map[Module]string
	servers map[server]struct{}
	targets map[targetData]uint64

	requesterLock sync.RWMutex
	serverOrder   []server
	pending       map[ServerAndID]pendingRequest

	// eventLock guards access to the events list. Note that eventLock can be
	// locked either while lock is locked or unlocked but lock cannot be locked
	// while eventLock is locked.
	eventLock sync.Mutex
	events    []Event
	stopCh    chan chan struct{}

	triggerCh chan struct{} // restarts waiting sync loop
	// if trigger has already been fired then send to testWaitCh blocks until
	// the triggered processing round is finished
	testWaitCh chan struct{}
}

type (
	// Server identifies a server without allowing any direct interaction.
	// Note: server interface is used by Scheduler and Tracker but not used by
	// the modules that do not interact with them directly.
	// In order to make module testing easier, Server interface is used in
	// events and modules.
	Server interface {
		Name() string
	}
	Request     any
	Response    any
	ID          uint64
	ServerAndID struct {
		Server Server
		ID     ID
	}
)

// targetData represents a registered target data structure that increases its
// ChangeCounter whenever it has been changed.
type targetData interface {
	ChangeCounter() uint64
}

// pendingRequest keeps track of sent and not yet finalized requests and their
// sender modules.
type pendingRequest struct {
	request Request
	module  Module
}

// NewScheduler creates a new Scheduler.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		servers: make(map[server]struct{}),
		names:   make(map[Module]string),
		pending: make(map[ServerAndID]pendingRequest),
		targets: make(map[targetData]uint64),
		stopCh:  make(chan chan struct{}),
		// Note: testWaitCh should not have capacity in order to ensure
		// that after a trigger happens testWaitCh will block until the resulting
		// processing round has been finished
		triggerCh:  make(chan struct{}, 1),
		testWaitCh: make(chan struct{}),
	}
	return s
}

// RegisterTarget registers a target data structure, ensuring that any changes
// made to it trigger a new round of Module.Process calls, giving a chance to
// modules to react to the changes.
func (s *Scheduler) RegisterTarget(t targetData) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.targets[t] = 0
}

// RegisterModule registers a module. Should be called before starting the scheduler.
// In each processing round the order of module processing depends on the order of
// registration.
func (s *Scheduler) RegisterModule(m Module, name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
	s.names[m] = name
}

// RegisterServer registers a new server.
func (s *Scheduler) RegisterServer(server server) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.addEvent(Event{Type: EvRegistered, Server: server})
	server.subscribe(func(event Event) {
		event.Server = server
		s.addEvent(event)
	})
}

// UnregisterServer removes a registered server.
func (s *Scheduler) UnregisterServer(server server) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server.unsubscribe()
	s.addEvent(Event{Type: EvUnregistered, Server: server})
}

// Start starts the scheduler. It should be called after registering all modules
// and before registering any servers.
func (s *Scheduler) Start() {
	go s.syncLoop()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	stop := make(chan struct{})
	s.stopCh <- stop
	<-stop
	s.lock.Lock()
	for server := range s.servers {
		server.unsubscribe()
	}
	s.servers = nil
	s.lock.Unlock()
}

// syncLoop is the main event loop responsible for event/data processing and
// sending new requests.
// A round of processing starts whenever the global trigger is fired. Triggers
// fired during a processing round ensure that there is going to be a next round.
func (s *Scheduler) syncLoop() {
	for {
		s.lock.Lock()
		s.processRound()
		s.lock.Unlock()
	loop:
		for {
			select {
			case stop := <-s.stopCh:
				close(stop)
				return
			case <-s.triggerCh:
				break loop
			case <-s.testWaitCh:
			}
		}
	}
}

// targetChanged returns true if a registered target data structure has been
// changed since the last call to this function.
func (s *Scheduler) targetChanged() (changed bool) {
	for target, counter := range s.targets {
		if newCounter := target.ChangeCounter(); newCounter != counter {
			s.targets[target] = newCounter
			changed = true
		}
	}
	return
}

// processRound runs an entire processing round. It calls the Process functions
// of all modules, passing all relevant events and repeating Process calls as
// long as any changes have been made to the registered target data structures.
// Once all events have been processed and a stable state has been achieved,
// requests are generated and sent if necessary and possible.
func (s *Scheduler) processRound() {
	for {
		log.Trace("Processing modules")
		filteredEvents := s.filterEvents()
		for _, module := range s.modules {
			log.Trace("Processing module", "name", s.names[module], "events", len(filteredEvents[module]))
			module.Process(requester{s, module}, filteredEvents[module])
		}
		if !s.targetChanged() {
			break
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

// addEvent adds an event to be processed in the next round. Note that it can be
// called regardless of the state of the lock mutex, making it safe for use in
// the server event callback.
func (s *Scheduler) addEvent(event Event) {
	s.eventLock.Lock()
	s.events = append(s.events, event)
	s.eventLock.Unlock()
	s.Trigger()
}

// filterEvent sorts each Event either as a request event or a server event,
// depending on its type. Request events are also sorted in a map based on the
// module that originally initiated the request. It also ensures that no events
// related to a server are returned before EvRegistered or after EvUnregistered.
// In case of an EvUnregistered server event it also closes all pending requests
// to the given server by adding a failed request event (EvFail), ensuring that
// all requests get finalized and thereby allowing the module logic to be safe
// and simple.
func (s *Scheduler) filterEvents() map[Module][]Event {
	s.eventLock.Lock()
	events := s.events
	s.events = nil
	s.eventLock.Unlock()

	s.requesterLock.Lock()
	defer s.requesterLock.Unlock()

	filteredEvents := make(map[Module][]Event)
	for _, event := range events {
		server := event.Server.(server)
		if _, ok := s.servers[server]; !ok && event.Type != EvRegistered {
			continue // before EvRegister or after EvUnregister, discard
		}

		if event.IsRequestEvent() {
			sid, _, _ := event.RequestInfo()
			pending, ok := s.pending[sid]
			if !ok {
				continue // request already closed, ignore further events
			}
			if event.Type == EvResponse || event.Type == EvFail {
				delete(s.pending, sid) // final event, close pending request
			}
			filteredEvents[pending.module] = append(filteredEvents[pending.module], event)
		} else {
			switch event.Type {
			case EvRegistered:
				s.servers[server] = struct{}{}
				s.serverOrder = append(s.serverOrder, nil)
				copy(s.serverOrder[1:], s.serverOrder[:len(s.serverOrder)-1])
				s.serverOrder[0] = server
			case EvUnregistered:
				s.closePending(event.Server, filteredEvents)
				delete(s.servers, server)
				for i, srv := range s.serverOrder {
					if srv == server {
						copy(s.serverOrder[i:len(s.serverOrder)-1], s.serverOrder[i+1:])
						s.serverOrder = s.serverOrder[:len(s.serverOrder)-1]
						break
					}
				}
			}
			for _, module := range s.modules {
				filteredEvents[module] = append(filteredEvents[module], event)
			}
		}
	}
	return filteredEvents
}

// closePending closes all pending requests to the given server and adds an EvFail
// event to properly finalize them
func (s *Scheduler) closePending(server Server, filteredEvents map[Module][]Event) {
	for sid, pending := range s.pending {
		if sid.Server == server {
			filteredEvents[pending.module] = append(filteredEvents[pending.module], Event{
				Type:   EvFail,
				Server: server,
				Data: RequestResponse{
					ID:      sid.ID,
					Request: pending.request,
				},
			})
			delete(s.pending, sid)
		}
	}
}

// requester implements Requester. Note that while requester basically wraps
// Scheduler (with the added information of the currently processed Module), all
// functions are safe to call from Module.Process which is running while
// the Scheduler.lock mutex is held.
type requester struct {
	*Scheduler
	module Module
}

// CanSendTo returns the list of currently available servers. It also returns
// them in an order of least to most recently used, ensuring a round-robin usage
// of suitable servers if the module always chooses the first suitable one.
func (s requester) CanSendTo() []Server {
	s.requesterLock.RLock()
	defer s.requesterLock.RUnlock()

	list := make([]Server, 0, len(s.serverOrder))
	for _, server := range s.serverOrder {
		if server.canRequestNow() {
			list = append(list, server)
		}
	}
	return list
}

// Send sends a request and adds an entry to Scheduler.pending map, ensuring that
// related request events will be delivered to the sender Module.
func (s requester) Send(srv Server, req Request) ID {
	s.requesterLock.Lock()
	defer s.requesterLock.Unlock()

	server := srv.(server)
	id := server.sendRequest(req)
	sid := ServerAndID{Server: srv, ID: id}
	s.pending[sid] = pendingRequest{request: req, module: s.module}
	for i, ss := range s.serverOrder {
		if ss == server {
			copy(s.serverOrder[i:len(s.serverOrder)-1], s.serverOrder[i+1:])
			s.serverOrder[len(s.serverOrder)-1] = server
			return id
		}
	}
	log.Error("Target server not found in ordered list of registered servers")
	return id
}

// Fail should be called when a server delivers invalid or useless information.
// Calling Fail disables the given server for a period that is initially short
// but is exponentially growing if it happens frequently. This results in a
// somewhat fault tolerant operation that avoids hammering servers with requests
// that they cannot serve but still gives them a chance periodically.
func (s requester) Fail(srv Server, desc string) {
	srv.(server).fail(desc)
}
