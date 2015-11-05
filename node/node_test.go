// Copyright 2015 The go-ethereum Authors
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

package node

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	testNodeKey, _ = crypto.GenerateKey()

	testNodeConfig = &Config{
		PrivateKey: testNodeKey,
		Name:       "test node",
	}
)

// Tests that an empty protocol stack can be started, restarted and stopped.
func TestNodeLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Ensure that a stopped node can be stopped again
	for i := 0; i < 3; i++ {
		if err := stack.Stop(); err != ErrNodeStopped {
			t.Fatalf("iter %d: stop failure mismatch: have %v, want %v", i, err, ErrNodeStopped)
		}
	}
	// Ensure that a node can be successfully started, but only once
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	if err := stack.Start(); err != ErrNodeRunning {
		t.Fatalf("start failure mismatch: have %v, want %v ", err, ErrNodeRunning)
	}
	// Ensure that a node can be restarted arbitrarily many times
	for i := 0; i < 3; i++ {
		if err := stack.Restart(); err != nil {
			t.Fatalf("iter %d: failed to restart node: %v", i, err)
		}
	}
	// Ensure that a node can be stopped, but only once
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop node: %v", err)
	}
	if err := stack.Stop(); err != ErrNodeStopped {
		t.Fatalf("stop failure mismatch: have %v, want %v ", err, ErrNodeStopped)
	}
}

// Tests that if the data dir is already in use, an appropriate error is returned.
func TestNodeUsedDataDir(t *testing.T) {
	// Create a temporary folder to use as the data directory
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary data directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create a new node based on the data directory
	original, err := New(&Config{DataDir: dir})
	if err != nil {
		t.Fatalf("failed to create original protocol stack: %v", err)
	}
	if err := original.Start(); err != nil {
		t.Fatalf("failed to start original protocol stack: %v", err)
	}
	defer original.Stop()

	// Create a second node based on the same data directory and ensure failure
	duplicate, err := New(&Config{DataDir: dir})
	if err != nil {
		t.Fatalf("failed to create duplicate protocol stack: %v", err)
	}
	if err := duplicate.Start(); err != ErrDatadirUsed {
		t.Fatalf("duplicate datadir failure mismatch: have %v, want %v", err, ErrDatadirUsed)
	}
}

// NoopService is a trivial implementation of the Service interface.
type NoopService struct{}

func (s *NoopService) Protocols() []p2p.Protocol { return nil }
func (s *NoopService) Start() error              { return nil }
func (s *NoopService) Stop() error               { return nil }

func NewNoopService(*ServiceContext) (Service, error) { return new(NoopService), nil }

// Tests whether services can be registered and unregistered.
func TestServiceRegistry(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Create a batch of dummy services and ensure they don't exist
	ids := []string{"A", "B", "C"}
	for i, id := range ids {
		if err := stack.Unregister(id); err != ErrServiceUnknown {
			t.Fatalf("service %d: pre-unregistration failure mismatch: have %v, want %v", i, err, ErrServiceUnknown)
		}
	}
	// Register the services, checking that the operation succeeds only once
	for i, id := range ids {
		if err := stack.Register(id, NewNoopService); err != nil {
			t.Fatalf("service %d: registration failed: %v", i, err)
		}
		if err := stack.Register(id, NewNoopService); err != ErrServiceRegistered {
			t.Fatalf("service %d: registration failure mismatch: have %v, want %v", i, err, ErrServiceRegistered)
		}
	}
	// Unregister the services, checking that the operation succeeds only once
	for i, id := range ids {
		if err := stack.Unregister(id); err != nil {
			t.Fatalf("service %d: unregistration failed: %v", i, err)
		}
		if err := stack.Unregister(id); err != ErrServiceUnknown {
			t.Fatalf("service %d: unregistration failure mismatch: have %v, want %v", i, err, ErrServiceUnknown)
		}
	}
}

// InstrumentedService is an implementation of Service for which all interface
// methods can be instrumented both return value as well as event hook wise.
type InstrumentedService struct {
	protocols []p2p.Protocol
	start     error
	stop      error

	protocolsHook func()
	startHook     func()
	stopHook      func()
}

func (s *InstrumentedService) Protocols() []p2p.Protocol {
	if s.protocolsHook != nil {
		s.protocolsHook()
	}
	return s.protocols
}

func (s *InstrumentedService) Start() error {
	if s.startHook != nil {
		s.startHook()
	}
	return s.start
}

func (s *InstrumentedService) Stop() error {
	if s.stopHook != nil {
		s.stopHook()
	}
	return s.stop
}

// Tests that registered services get started and stopped correctly.
func TestServiceLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of life-cycle instrumented services
	ids := []string{"A", "B", "C"}

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for i, id := range ids {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func() { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(id, constructor); err != nil {
			t.Fatalf("service %d: registration failed: %v", i, err)
		}
	}
	// Start the node and check that all services are running
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	for i, id := range ids {
		if !started[id] {
			t.Fatalf("service %d: freshly started service not running", i)
		}
		if stopped[id] {
			t.Fatalf("service %d: freshly started service already stopped", i)
		}
		if stack.Service(id) == nil {
			t.Fatalf("service %d: freshly started service unaccessible", i)
		}
	}
	// Stop the node and check that all services have been stopped
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop protocol stack: %v", err)
	}
	for i, id := range ids {
		if !stopped[id] {
			t.Fatalf("service %d: freshly terminated service still running", i)
		}
		if service := stack.Service(id); service != nil {
			t.Fatalf("service %d: freshly terminated service still accessible: %v", i, service)
		}
	}
}

// Tests that services are restarted cleanly as new instances.
func TestServiceRestarts(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Define a service that does not support restarts
	var (
		running bool
		started int
	)
	constructor := func(*ServiceContext) (Service, error) {
		running = false

		return &InstrumentedService{
			startHook: func() {
				if running {
					panic("already running")
				}
				running = true
				started++
			},
		}, nil
	}
	// Register the service and start the protocol stack
	if err := stack.Register("service", constructor); err != nil {
		t.Fatalf("failed to register the service: %v", err)
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	defer stack.Stop()

	if running != true || started != 1 {
		t.Fatalf("running/started mismatch: have %v/%d, want true/1", running, started)
	}
	// Restart the stack a few times and check successful service restarts
	for i := 0; i < 3; i++ {
		if err := stack.Restart(); err != nil {
			t.Fatalf("iter %d: failed to restart stack: %v", i, err)
		}
	}
	if running != true || started != 4 {
		t.Fatalf("running/started mismatch: have %v/%d, want true/4", running, started)
	}
}

// Tests that if a service fails to initialize itself, none of the other services
// will be allowed to even start.
func TestServiceConstructionAbortion(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Define a batch of good services
	ids := []string{"A", "B", "C", "D", "E", "F"}

	started := make(map[string]bool)
	for i, id := range ids {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func() { started[id] = true },
			}, nil
		}
		if err := stack.Register(id, constructor); err != nil {
			t.Fatalf("service %d: registration failed: %v", i, err)
		}
	}
	// Register a service that fails to construct itself
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return nil, failure
	}
	if err := stack.Register("failer", failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack and ensure none of the services get started
	for i := 0; i < 100; i++ {
		if err := stack.Start(); err != failure {
			t.Fatalf("iter %d: stack startup failure mismatch: have %v, want %v", i, err, failure)
		}
		for i, id := range ids {
			if started[id] {
				t.Fatalf("service %d: started should not have", i)
			}
			delete(started, id)
		}
	}
}

// Tests that if a service fails to start, all others started before it will be
// shut down.
func TestServiceStartupAbortion(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of good services
	ids := []string{"A", "B", "C", "D", "E", "F"}

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for i, id := range ids {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func() { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(id, constructor); err != nil {
			t.Fatalf("service %d: registration failed: %v", i, err)
		}
	}
	// Register a service that fails to start
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return &InstrumentedService{
			start: failure,
		}, nil
	}
	if err := stack.Register("failer", failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack and ensure all started services stop
	for i := 0; i < 100; i++ {
		if err := stack.Start(); err != failure {
			t.Fatalf("iter %d: stack startup failure mismatch: have %v, want %v", i, err, failure)
		}
		for i, id := range ids {
			if started[id] && !stopped[id] {
				t.Fatalf("service %d: started but not stopped", i)
			}
			delete(started, id)
			delete(stopped, id)
		}
	}
}

// Tests that even if a registered service fails to shut down cleanly, it does
// not influece the rest of the shutdown invocations.
func TestServiceTerminationGuarantee(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of good services
	ids := []string{"A", "B", "C", "D", "E", "F"}

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for i, id := range ids {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func() { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(id, constructor); err != nil {
			t.Fatalf("service %d: registration failed: %v", i, err)
		}
	}
	// Register a service that fails to shot down cleanly
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return &InstrumentedService{
			stop: failure,
		}, nil
	}
	if err := stack.Register("failer", failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack, and ensure that a failing shut down terminates all
	for i := 0; i < 100; i++ {
		// Start the stack and make sure all is online
		if err := stack.Start(); err != nil {
			t.Fatalf("iter %d: failed to start protocol stack: %v", i, err)
		}
		for j, id := range ids {
			if !started[id] {
				t.Fatalf("iter %d, service %d: service not running", i, j)
			}
			if stopped[id] {
				t.Fatalf("iter %d, service %d: service already stopped", i, j)
			}
		}
		// Stop the stack, verify failure and check all terminations
		err := stack.Stop()
		if err, ok := err.(*StopError); !ok {
			t.Fatalf("iter %d: termination failure mismatch: have %v, want StopError", i, err)
		} else {
			if err.Services["failer"] != failure {
				t.Fatalf("iter %d: failer termination failure mismatch: have %v, want %v", i, err.Services["failer"], failure)
			}
			if len(err.Services) != 1 {
				t.Fatalf("iter %d: failure count mismatch: have %d, want %d", i, len(err.Services), 1)
			}
		}
		for j, id := range ids {
			if !stopped[id] {
				t.Fatalf("iter %d, service %d: service not terminated", i, j)
			}
			delete(started, id)
			delete(stopped, id)
		}
	}
}

// Tests that all protocols defined by individual services get launched.
func TestProtocolGather(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of services with some configured number of protocols
	services := map[string]int{
		"Zero Protocols":  0,
		"Single Protocol": 1,
		"Many Protocols":  25,
	}
	for id, count := range services {
		protocols := make([]p2p.Protocol, count)
		for i := 0; i < len(protocols); i++ {
			protocols[i].Name = id
			protocols[i].Version = uint(i)
		}
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				protocols: protocols,
			}, nil
		}
		if err := stack.Register(id, constructor); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Start the services and ensure all protocols start successfully
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	defer stack.Stop()

	protocols := stack.Server().Protocols
	if len(protocols) != 26 {
		t.Fatalf("mismatching number of protocols launched: have %d, want %d", len(protocols), 26)
	}
	for id, count := range services {
		for ver := 0; ver < count; ver++ {
			launched := false
			for i := 0; i < len(protocols); i++ {
				if protocols[i].Name == id && protocols[i].Version == uint(ver) {
					launched = true
					break
				}
			}
			if !launched {
				t.Errorf("configured protocol not launched: %s v%d", id, ver)
			}
		}
	}
}
