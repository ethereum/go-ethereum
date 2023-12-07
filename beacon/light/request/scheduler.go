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
)

// Module represents an update mechanism which is typically responsible for a
// passive data structure or a certain aspect of it. When registered to a Scheduler,
// it can be triggered either by server events, other modules or itself.
type Module interface {
	// Process is a non-blocking function that is called whenever the module is
	// triggered. It can start network requests through the received Environment
	// and/or do other data processing tasks. If triggers are set up correctly,
	// Process is eventually called whenever it might have something new to do
	// either because the data structures have been changed or because new servers
	// became available or new requests became available at existing ones.
	//
	// Note: Process functions of different modules are never called concurrently;
	// they are called by Scheduler in the same order of priority as they were
	// registered in.
	Process(ServerSet, []ServerEvent) bool
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or connected servers could allow new operations.
type Scheduler struct {
	lock    sync.Mutex
	clock   mclock.Clock
	modules []Module // first has highest priority
	servers map[Server]struct{}
	stopCh  chan chan struct{}

	triggerCh chan struct{} // restarts waiting sync loop
	//	testWaitCh       chan struct{} // accepts sends when sync loop is waiting
	//	testTimerResults []bool        // true is appended when simulated timer is processed; false when stopped
	events []ServerEvent
}

type ServerEvent struct {
	Server Server
	Event  Event
}

// NewScheduler creates a new Scheduler.
func NewScheduler(clock mclock.Clock) *Scheduler {
	s := &Scheduler{
		clock:   clock,
		servers: make(map[Server]struct{}),
		stopCh:  make(chan chan struct{}),
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
func (s *Scheduler) RegisterModule(m Module) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
}

// RegisterServer registers a new server.
func (s *Scheduler) RegisterServer(server Server) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server.Subscribe(func(event Event) {
		s.handleEvent(server, event)
	})
	s.servers[server] = struct{}{}
	s.Trigger()
}

// UnregisterServer removes a registered server.
func (s *Scheduler) UnregisterServer(server Server) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server.Unsubscribe()
	delete(s.servers, server)
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
		server.Unsubscribe()
	}
	s.servers = nil
	s.lock.Unlock()
	stop := make(chan struct{})
	s.stopCh <- stop
	<-stop
}

// syncLoop calls all processable modules in the order of their registration.
// A round of processing starts whenever there is at least one processable module.
// Triggers triggered during a processing round do not affect the current round
// but ensure that there is going to be a next round.
func (s *Scheduler) syncLoop() {
	for {
		s.lock.Lock()
		s.processModules()
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

// processModules runs an entire processing round, calling processable modules
// with the appropriate Environment.
func (s *Scheduler) processModules() {
	servers := make(ServerSet)
	for server, _ := range s.servers {
		if ok, _ := server.CanRequestNow(); ok {
			servers[server] = struct{}{}
		}
	}
	for _, module := range s.modules {
		if module.Process(servers, s.events) {
			s.Trigger()
		}
	}
	s.events = nil
}

func (s *Scheduler) Trigger() {
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

func (s *Scheduler) handleEvent(server Server, event Event) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.events = append(s.events, ServerEvent{Server: server, Event: event})
	s.Trigger()
}
