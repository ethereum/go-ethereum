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
// Modules are called by Scheduler whenever a global trigger is fired. All events
// fire the trigger. Modules themselves can also self-trigger, ensuring an
// immediate next processing round after the target data structure has been
// changed in a way that could make further actions possible either by the same
// or another Module.
type Module interface {
	// Process is a non-blocking function that is called on each Module whenever
	// a processing round is triggered. It can start new requests through the
	// received Tracker, process events and/or do other data processing tasks.
	// Note that request events are only passed to the module that made the given
	// request while server events are passed to every module.
	//
	// Note: Process functions of different modules are never called concurrently;
	// they are called by Scheduler in the same order of priority as they were
	// registered in.
	HandleEvent(Event)
	Process()
	MakeRequest(Server) (Request, float32)
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or events coming from registered servers could
// allow new operations.
type Scheduler struct {
	lock    sync.Mutex
	clock   mclock.Clock
	modules []Module // first has highest priority
	names   map[Module]string
	servers map[server]struct{}
	targets map[targetData]uint64

	pending   map[ServerAndID]pendingRequest
	eventLock sync.Mutex
	events    []Event
	stopCh    chan chan struct{}

	triggerCh chan struct{} // restarts waiting sync loop
	//	testWaitCh       chan struct{} // accepts sends when sync loop is waiting
	//	testTimerResults []bool        // true is appended when simulated timer is processed; false when stopped
}

type (
	// Server identifies a server without allowing any direct interaction.
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
	RequestWithID struct {
		ServerAndID
		Request Request
	}
)

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
func NewScheduler(clock mclock.Clock) *Scheduler {
	s := &Scheduler{
		clock:   clock,
		servers: make(map[server]struct{}),
		names:   make(map[Module]string),
		pending: make(map[ServerAndID]pendingRequest),
		targets: make(map[targetData]uint64),
		stopCh:  make(chan chan struct{}),
		// Note: testWaitCh should not have capacity in order to ensure
		// that after a trigger happens testWaitCh will block until the resulting
		// processing round has been finished
		triggerCh: make(chan struct{}, 1),
		//testWaitCh: make(chan struct{}),
	}
	return s
}

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
func (s *Scheduler) RegisterServer(rs requestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server := newServer(rs, s.clock)
	s.addEvent(Event{Type: EvRegistered, Server: server})
	server.subscribe(func(event Event) {
		s.lock.Lock()
		if _, ok := s.servers[server]; ok {
			event.Server = server
			s.addEvent(event)
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
			s.addEvent(Event{Type: EvUnregistered, Server: server})
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
	for server := range s.servers {
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
		s.lock.Lock()
		s.handleEvents()
		for s.targetChanged() {
			s.processModules()
		}
		s.sendRequests()
		s.lock.Unlock()
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

func (s *Scheduler) targetChanged() (changed bool) {
	for target, counter := range s.targets {
		if newCounter := target.ChangeCounter(); newCounter != counter {
			s.targets[target] = newCounter
			changed = true
		}
	}
	return
}

// processModules runs an entire processing round, calling the Process functions
// of all modules, passing all relevant events.
func (s *Scheduler) processModules() {
	for _, module := range s.modules {
		module.Process()
	}
}

func (s *Scheduler) sendRequests() {
	servers := make(serverSet)
	for server := range s.servers {
		if ok, _ := server.canRequestNow(); ok {
			servers[server] = struct{}{}
		}
	}
	log.Debug("Processing modules", "servers", len(servers))

	for _, module := range s.modules {
		for {
			if len(servers) == 0 {
				return
			}
			if req, sent := s.tryRequest(module, servers); sent {
				module.HandleEvent(Event{
					Type:   EvRequest,
					Server: req.Server,
					Data: RequestResponse{
						ID:      req.ID,
						Request: req.Request,
					},
				})
			} else {
				break
			}
		}
	}
}

func (s *Scheduler) tryRequest(module Module, servers serverSet) (RequestWithID, bool) {
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
		return RequestWithID{}, false
	}
	id := ServerAndID{Server: bestServer, ID: bestServer.sendRequest(bestRequest)}
	s.pending[id] = pendingRequest{request: bestRequest, module: module}
	return RequestWithID{ServerAndID: id, Request: bestRequest}, true
}

// Trigger starts a new processing round. If fired during processing, it ensures
// another full round of processing all modules.
func (s *Scheduler) Trigger() {
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

func (s *Scheduler) addEvent(event Event) {
	s.eventLock.Lock()
	s.events = append(s.events, event)
	s.Trigger()
	s.eventLock.Unlock()
}

func (s *Scheduler) handleEvents() {
	s.eventLock.Lock()
	events := s.events
	s.events = nil
	s.eventLock.Unlock()
	for _, event := range events {
		s.handleEvent(event)
	}
}

// handleEvent processes an Event and adds it either as a request event or a
// server event, depending on its type. In case of an EvUnregistered server event
// it also closes all pending requests to the given server by emitting a failed
// request event (EvFail), ensuring that all requests get finalized and thereby
// allowing the module logic to be safe and simple.
func (s *Scheduler) handleEvent(event Event) {
	if event.IsRequestEvent() {
		sid, _, _ := event.RequestInfo()
		if pr, ok := s.pending[sid]; ok {
			pr.module.HandleEvent(event)
			if event.Type != EvTimeout {
				delete(s.pending, sid)
			}
		}
		return
	}
	if event.Type == EvUnregistered {
		for id, pending := range s.pending {
			if id.Server != event.Server {
				continue
			}
			pending.module.HandleEvent(Event{
				Type:   EvFail,
				Server: event.Server,
				Data: RequestResponse{
					ID:      id.ID,
					Request: pending.request,
				},
			})
		}
	}
	for _, module := range s.modules {
		module.HandleEvent(event)
	}
}
