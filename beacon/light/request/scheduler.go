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

type Module interface {
	SetupTriggers(trigger func(id string, subscribe bool) *ModuleTrigger)
	Process(env *Environment)
}

type RequestServer interface {
	SubscribeHeads(newHead func(uint64, common.Hash), newSignedHead func(types.SignedHead))
	UnsubscribeHeads()
	Delay() time.Duration
	Fail(string)
}

// initialized when first trigger is added
type ModuleTrigger struct { // Scheduler lock
	s        *Scheduler
	triggers map[Module]struct{}
}

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

func NewScheduler(headTracker *HeadTracker) *Scheduler {
	return &Scheduler{
		headTracker: headTracker,
		stopCh:      make(chan chan struct{}),
		triggerCh:   make(chan struct{}, 1),
		triggers:    make(map[string]*ModuleTrigger),
		triggeredBy: make(map[Module][]*ModuleTrigger),
	}
}

// call before starting the scheduler
func (s *Scheduler) RegisterModule(m Module) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
	m.SetupTriggers(func(id string, subscribe bool) *ModuleTrigger { return s.addTrigger(m, id, subscribe) })
}

func (s *Scheduler) addTrigger(m Module, id string, subscribe bool) *ModuleTrigger {
	t, ok := s.triggers[id]
	if !ok {
		t = new(ModuleTrigger)
		s.triggers[id] = t
	}
	if !subscribe {
		return t
	}
	s.triggeredBy[m] = append(s.triggeredBy[m], t)
	if t.triggers == nil {
		t.s = s
		t.triggers = make(map[Module]struct{})
	}
	t.triggers[m] = struct{}{}
	return t
}

// GetModuleTrigger returns the ModuleTrigger with the given id or creates a new one.
func (s *Scheduler) GetModuleTrigger(id string) *ModuleTrigger {
	s.lock.Lock()
	defer s.lock.Unlock()

	t, ok := s.triggers[id]
	if !ok {
		t = new(ModuleTrigger)
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

// Start starts the scheduler. It should be called after registering all modules and before registering any servers.
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

// syncLoop calls all processable modules in the order of their registration. A round of processing starts whenever there is at least one processable module. Triggers triggered during a processing round do not affect the current round but ensure that there is going to be a next round.
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

// processModules runs an entire processing round, calling processable modules with the appropriate Environment.
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
		if canRequest, _ := server.CanRequestNow(); !canRequest {
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

// triggerServer ensures that a next processing round is initiated as soon as possible and every module will be called with the given server enabled in its Environment. Should be called when the given server has become available (again) or when its range of servable requests has been expanded.
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

// triggerModule ensures that a next processing round is initiated as soon as possible and the given module will be called with all servers enabled in its Environment. Called by ModuleTrigger.Trigger when the range of possible requests or processable data might have been expanded.
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
