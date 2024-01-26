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

	"github.com/ethereum/go-ethereum/log"
)

// Module represents a mechanism which is typically responsible for downloading
// and updating a passive data structure. It does not directly interact with the
// servers (except for reporting server side failures). It receives and processes
// events, maintains its internal state and generates request candidates. It is
// the Scheduler's responsibility to feed events to the modules, call Process as
// long as there might be something to process and then generate request
// candidates using MakeRequest and start the best possible requests.
// Modules are called by Scheduler whenever a global trigger is fired. All events
// fire the trigger. Changing a target data structure also triggers a next
// processing round as it could make further actions possible either by the same
// or another Module.
type Module interface {
	// Process is a non-blocking function responsible for maintaining the target
	// data structures(s) and the internal state of the module. This state
	// typically consists of information about pending requests and registered
	// servers and it is updated based on the received events.
	// Process is always called after an event is received or after a target data
	// structure has been changed.
	//
	// Note: Process functions of different modules are never called concurrently;
	// they are called by Scheduler in the same order of priority as they were
	// registered in.
	Process([]Event)
	// MakeRequest generates a request candidate based on the state of the target
	// structure(s) and the internal state of the module. This candidate is
	// typically the next obtainable item (or range of items) of the target
	// structure that is assumed to be available at the given server and has not
	// been requested yet (or has been requested but already timed out and should
	// be resent).
	// MakeRequest is always called after Process. Note that it is the Scheduler's
	// job to select the best possible requests and actually send them. If a
	// request has been sent, the module is notified through an EvRequest event
	// which also immediately triggers a next processing round, allowing modules
	// to send more requests if possible and necessary.
	MakeRequest(Server) (Request, float32)
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or events coming from registered servers could
// allow new operations.
type Scheduler struct {
	lock    sync.Mutex
	modules []Module // first has highest priority
	names   map[Module]string
	servers map[server]struct{}
	targets map[targetData]uint64

	pending map[ServerAndID]pendingRequest
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
	// Server identifies a server without allowing any direct interaction except
	// for reporting a server side failure.
	// Note: server interface is used by Scheduler and Tracker but not used by
	// the modules that do not interact with them directly.
	// In order to make module testing easier, Server interface is used in
	// events and modules.
	Server interface {
		Fail(desc string)
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
		filteredEvents := s.filterEvents()
		log.Debug("Processing modules")
		for _, module := range s.modules {
			log.Debug("Processing module", "name", s.names[module], "events", len(filteredEvents[module]))
			module.Process(filteredEvents[module])
		}
		if !s.targetChanged() {
			break
		}
	}
	s.sendRequests()
}

// sendRequests lets each module generate a request if necessary and sends it to
// a suitable server if possible.
// Note that if a request is sent, an EvRequest event will immediately trigger a
// next processing round, thereby allowing modules to create any number of requests
// in any suitable moment as long as there is a server that can accept them.
func (s *Scheduler) sendRequests() {
	servers := make(map[server]struct{})
	for server := range s.servers {
		if ok, _ := server.canRequestNow(); ok {
			servers[server] = struct{}{}
		}
	}
	log.Debug("Generating request candidates", "servers", len(servers))

	for _, module := range s.modules {
		if len(servers) == 0 {
			return
		}
		if s.tryRequest(module, servers) {
			log.Debug("Sent request", "module", s.names[module])
		}
	}
}

// tryRequest tries to generate request candidates for a given module and a given
// set of servers, then selects the best candidate if there is one and sends the
// request to the server it was generated for.
// The candidates are primarily ranked based on "request priority", a number that
// Module.MakeRequest has returned along with the request candidate. This ranking
// may or may not be used depending on the type of the request, identical requests
// typically have the same priority while multiple item requests may have a
// priority based on the number of items requested.
// If there are multiple candidates with identical request priority then they are
// ranked based on "server priority" which is determined by the server. This value
// is typically higher is the server is expected to respond quicker or with a
// higher chance (typically a lower number of pending requests).
// Note that tryRequest can also remove items from the set of available servers
// if they are no longer able to accept requests in the current processing round.
func (s *Scheduler) tryRequest(module Module, servers map[server]struct{}) bool {
	var (
		maxServerPriority, maxRequestPriority float32
		bestServer                            server
		bestRequest                           Request
	)
	maxServerPriority, maxRequestPriority = -math.MaxFloat32, -math.MaxFloat32
	serverCount := len(servers)
	var removed, candidates int
	for server := range servers {
		canRequest, serverPriority := server.canRequestNow()
		if !canRequest {
			delete(servers, server)
			removed++
			continue
		}
		request, requestPriority := module.MakeRequest(server)
		if request != nil {
			candidates++
		}
		if request == nil || requestPriority < maxRequestPriority ||
			(requestPriority == maxRequestPriority && serverPriority <= maxServerPriority) {
			continue
		}
		maxServerPriority, maxRequestPriority = serverPriority, requestPriority
		bestServer, bestRequest = server, request
	}
	log.Debug("Request attempt", "serverCount", serverCount, "removedServers", removed, "requestCandidates", candidates)
	if bestServer == nil {
		return false
	}
	sid := ServerAndID{Server: bestServer, ID: bestServer.sendRequest(bestRequest)}
	s.pending[sid] = pendingRequest{request: bestRequest, module: module}
	return true
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
			case EvUnregistered:
				s.closePending(event.Server, filteredEvents)
				delete(s.servers, server)
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
