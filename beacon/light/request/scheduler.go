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
)

const softRequestTimeout = time.Second

type Module interface {
	Process(servers []*Server) bool // removed if return value is false
}

type RequestServer interface {
	SetTriggerCallback(func())
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
		t.s.moduleTrigger(m)
	}
}

type Scheduler struct {
	lock        sync.Mutex
	modules     []Module // first has highest priority
	servers     []*Server
	triggeredBy map[Module][]*ModuleTrigger
	stopCh      chan chan struct{}

	triggerCh             chan struct{}
	triggerLock           sync.Mutex
	processing, triggered bool
	trModules             map[Module]struct{}
	trServers             map[*Server]struct{}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		stopCh:      make(chan chan struct{}),
		triggerCh:   make(chan struct{}, 1),
		triggeredBy: make(map[Module][]*ModuleTrigger),
	}
}

// call before starting the scheduler
func (s *Scheduler) RegisterModule(m Module) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.modules = append(s.modules, m)
}

func (s *Scheduler) AddTriggers(m Module, triggeredBy []*ModuleTrigger) {
	s.triggeredBy[m] = append(s.triggeredBy[m], triggeredBy...)
	for _, t := range triggeredBy {
		if t.triggers == nil {
			t.s = s
			t.triggers = make(map[Module]struct{})
		}
		t.triggers[m] = struct{}{}
	}
}

func (s *Scheduler) unregisterModule(m Module, t []*ModuleTrigger) {
	for i, module := range s.modules {
		if module == m {
			copy(s.modules[i:len(s.modules)-1], s.modules[i+1:])
			s.modules = s.modules[:len(s.modules)-1]
			break
		}
	}
	triggeredBy := s.triggeredBy[m]
	delete(s.triggeredBy, m)
	for _, t := range triggeredBy {
		delete(t.triggers, m)
	}
}

func (s *Scheduler) RegisterServer(RequestServer RequestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	server := s.newServer(RequestServer)
	s.servers = append(s.servers, server)
	RequestServer.SetTriggerCallback(func() {
		s.ServerTrigger(server)
	})
	s.ServerTrigger(server)
}

func (s *Scheduler) UnregisterServer(RequestServer RequestServer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, server := range s.servers {
		if server.RequestServer == RequestServer {
			s.servers[i] = s.servers[len(s.servers)-1]
			s.servers = s.servers[:len(s.servers)-1]
			server.stop()
			return
		}
	}
}

// call before registering servers
func (s *Scheduler) Start() {
	go s.syncLoop()
}

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

func (s *Scheduler) processModules(trModules map[Module]struct{}, trServers map[*Server]struct{}) {
	trs := make([]*Server, 0, len(s.servers))
	for _, server := range s.servers {
		if _, ok := trServers[server]; ok {
			trs = append(trs, server)
		}
	}
	var i int
	for _, module := range s.modules {
		keep := true
		if _, ok := trModules[module]; ok {
			keep = module.Process(s.servers)
		} else if len(trs) > 0 {
			keep = module.Process(trs)
		}
		if keep {
			s.modules[i] = module
			i++
		}
	}
	s.modules = s.modules[:i]
}

func (s *Scheduler) ServerTrigger(server *Server) {
	s.triggerLock.Lock()
	s.serverTrigger(server)
	s.triggerLock.Unlock()
}

func (s *Scheduler) serverTrigger(server *Server) {
	if s.trServers == nil {
		s.trServers = make(map[*Server]struct{})
	}
	s.trServers[server] = struct{}{}
	if !s.processing && !s.triggered {
		s.triggerCh <- struct{}{}
		s.triggered = true
	}
}

func (s *Scheduler) ModuleTrigger(module Module) {
	s.triggerLock.Lock()
	s.moduleTrigger(module)
	s.triggerLock.Unlock()
}

func (s *Scheduler) moduleTrigger(module Module) {
	if s.trModules == nil {
		s.trModules = make(map[Module]struct{})
	}
	s.trModules[module] = struct{}{}
	if !s.processing && !s.triggered {
		s.triggerCh <- struct{}{}
		s.triggered = true
	}
}
