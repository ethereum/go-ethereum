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
	"time"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
)

const softRequestTimeout = time.Second

// Module represents an update mechanism which is typically responsible for a
// passive data structure or a certain aspect of it. When registered to a Scheduler,
// it can be triggered either by server events, other modules or itself.
type Module interface {
	// SetupModuleTriggers allows modules to set up module triggers while getting
	// registered to a Scheduler. Module trigger signals are typically emitted
	// when the sender has changed the data structure it's responsible for in a
	// way that might allow subscriber modules to start new requests or do more
	// data processing.
	SetupModuleTriggers(trigger func(id string, subscribe bool) *ModuleTrigger)
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
	Process(env *Environment)
}

// RequestServer is a general server interface that can be extended by modules
// with specific request types.
type RequestServer interface {
	SubscribeHeads(newHead func(uint64, common.Hash), newSignedHead func(types.SignedHead))
	UnsubscribeHeads()
	Delay() time.Duration // if non-zero then no requests should be sent for the given duration
	Fail(string)          // report server failure
}

// ModuleTrigger allows modules to trigger themselves or each other when changes
// in their underlying data structures could have made further operations possible.
type ModuleTrigger struct {
	s        *Scheduler
	triggers map[Module]struct{}
}

// Trigger ensures that subscribed modules will be processed in the next processing
// round which is started either immediately or after the current round has been finished.
func (t *ModuleTrigger) Trigger() {
	if t.triggers == nil {
		return
	}
	t.s.triggerLock.Lock()
	defer t.s.triggerLock.Unlock()

	for m := range t.triggers {
		t.s.triggerModule(m)
	}
}

// Scheduler is a modular network data retrieval framework that coordinates multiple
// servers and retrieval mechanisms (modules). It implements a trigger mechanism
// that calls the Process function of registered modules whenever either the state
// of existing data structures or connected servers could allow new operations.
type Scheduler struct {
	headTracker *HeadTracker

	lock        sync.Mutex
	modules     []Module // first has highest priority
	servers     []*Server
	triggers    map[string]*ModuleTrigger
	triggeredBy map[Module][]*ModuleTrigger
	stopCh      chan chan struct{}

	triggerCh             chan struct{}
	triggerLock           sync.Mutex
	processing, triggered bool
	trModules             map[Module]struct{}
	trServers             map[*Server]struct{}
}

// NewScheduler creates a new Scheduler.
func NewScheduler(headTracker *HeadTracker) *Scheduler {
	s := &Scheduler{
		headTracker: headTracker,
		stopCh:      make(chan chan struct{}),
		triggerCh:   make(chan struct{}, 1),
		triggers:    make(map[string]*ModuleTrigger),
		triggeredBy: make(map[Module][]*ModuleTrigger),
	}
	headTracker.setupModuleTriggers(s.GetModuleTrigger)
	return s
}

// RegisterModule registers a module. Should be called before starting the scheduler.
// In each processing round the order of module processing depends on the order of
// registration.
func (s *Scheduler) RegisterModule(m Module) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
	m.SetupModuleTriggers(func(id string, subscribe bool) *ModuleTrigger {
		t := s.getModuleTrigger(id)
		if subscribe {
			s.triggeredBy[m] = append(s.triggeredBy[m], t)
			t.triggers[m] = struct{}{}
		}
		return t
	})
}

// GetModuleTrigger returns the ModuleTrigger with the given id or creates a new one.
func (s *Scheduler) GetModuleTrigger(id string) *ModuleTrigger {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.getModuleTrigger(id)
}

// getModuleTrigger returns the ModuleTrigger with the given id or creates a new one.
func (s *Scheduler) getModuleTrigger(id string) *ModuleTrigger {
	t, ok := s.triggers[id]
	if !ok {
		t = &ModuleTrigger{
			s:        s,
			triggers: make(map[Module]struct{}),
		}
		s.triggers[id] = t
	}
	return t
}

// RegisterServer registers a new server.
func (s *Scheduler) RegisterServer(requestServer RequestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server := s.newServer(requestServer)
	s.servers = append(s.servers, server)
	s.headTracker.registerServer(server)
}

// UnregisterServer removes a registered server.
func (s *Scheduler) UnregisterServer(RequestServer RequestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, server := range s.servers {
		if server.RequestServer == RequestServer {
			s.servers[i] = s.servers[len(s.servers)-1]
			s.servers = s.servers[:len(s.servers)-1]
			server.stop()
			s.headTracker.unregisterServer(server)
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
	for _, server := range s.servers {
		server.stop()
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
	s.lock.Lock()
	s.triggerLock.Lock()
	s.processing = true
	for {
		trModules, trServers := s.trModules, s.trServers
		s.trModules, s.trServers = nil, nil
		if trModules != nil || trServers != nil {
			s.triggerLock.Unlock()
			s.processModules(trModules, trServers)
			s.triggerLock.Lock()
		} else {
			s.processing = false
			s.triggerLock.Unlock()
			s.lock.Unlock()
			select {
			case stop := <-s.stopCh:
				close(stop)
				return
			case <-s.triggerCh:
			}
			s.lock.Lock()
			s.triggerLock.Lock()
			s.triggered = false
			s.processing = true
		}
	}
}

// processModules runs an entire processing round, calling processable modules
// with the appropriate Environment.
func (s *Scheduler) processModules(trModules map[Module]struct{}, trServers map[*Server]struct{}) {
	mtEnv := Environment{ // enables all servers for triggered modules
		HeadTracker:   s.headTracker,
		scheduler:     s,
		allServers:    s.servers,
		canRequestNow: make(map[*Server]struct{}),
	}
	stEnv := Environment{ // enables triggered servers only for other modules
		HeadTracker:   s.headTracker,
		scheduler:     s,
		allServers:    s.servers,
		canRequestNow: make(map[*Server]struct{}),
	}
	for _, server := range s.servers {
		if canRequest, _ := server.canRequestNow(); !canRequest {
			continue
		}
		mtEnv.canRequestNow[server] = struct{}{}
		if _, ok := trServers[server]; ok {
			stEnv.canRequestNow[server] = struct{}{}
		}
	}

	for _, module := range s.modules {
		if _, ok := trModules[module]; ok {
			module.Process(&mtEnv)
		} else if len(stEnv.canRequestNow) > 0 {
			module.Process(&stEnv)
		}
	}
}

// triggerServer ensures that a next processing round is initiated as soon as
// possible and every module will be called with the given server enabled in its
// Environment. Should be called when the given server has (again) become available
// for requests or when its range of servable requests has been expanded (typically
// when it announces a new head).
func (s *Scheduler) triggerServer(server *Server) {
	s.triggerLock.Lock()
	if s.trServers == nil {
		s.trServers = make(map[*Server]struct{})
	}
	s.trServers[server] = struct{}{}
	if !s.processing && !s.triggered {
		s.triggerCh <- struct{}{}
		s.triggered = true
	}
	s.triggerLock.Unlock()
}

// triggerModule ensures that a next processing round is initiated as soon as possible
// and the given module will be called with all servers enabled in its Environment.
// Called by ModuleTrigger.Trigger when the range of possible requests or processable
// data might have been expanded.
func (s *Scheduler) triggerModule(module Module) {
	if s.trModules == nil {
		s.trModules = make(map[Module]struct{})
	}
	s.trModules[module] = struct{}{}
	if !s.processing && !s.triggered {
		s.triggerCh <- struct{}{}
		s.triggered = true
	}
}
