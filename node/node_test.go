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
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
)

func testNodeConfig() *Config {
	return &Config{
		PrivateKey: testNodeKey,
		Name:       "test node",
	}
}

// Tests that an empty protocol stack can be started, restarted and stopped.
func TestNodeLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig())
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

// Tests whether services can be registered and duplicates caught.
func TestServiceRegistry(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of unique services and ensure they start successfully
	services := []ServiceConstructor{NewNoopServiceA, NewNoopServiceB, NewNoopServiceC}
	for i, constructor := range services {
		if err := stack.Register(constructor); err != nil {
			t.Fatalf("service #%d: registration failed: %v", i, err)
		}
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start original service stack: %v", err)
	}
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop original service stack: %v", err)
	}
	// Duplicate one of the services and retry starting the node
	if err := stack.Register(NewNoopServiceB); err != nil {
		t.Fatalf("duplicate registration failed: %v", err)
	}
	if err := stack.Start(); err == nil {
		t.Fatalf("duplicate service started")
	} else {
		if _, ok := err.(*DuplicateServiceError); !ok {
			t.Fatalf("duplicate error mismatch: have %v, want %v", err, DuplicateServiceError{})
		}
	}
}

// Tests that registered services get started and stopped correctly.
func TestServiceLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of life-cycle instrumented services
	services := map[string]InstrumentingWrapper{
		"A": InstrumentedServiceMakerA,
		"B": InstrumentedServiceMakerB,
		"C": InstrumentedServiceMakerC,
	}
	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for id, maker := range services {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func(*p2p.Server) { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(maker(constructor)); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Start the node and check that all services are running
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	for id := range services {
		if !started[id] {
			t.Fatalf("service %s: freshly started service not running", id)
		}
		if stopped[id] {
			t.Fatalf("service %s: freshly started service already stopped", id)
		}
	}
	// Stop the node and check that all services have been stopped
	if err := stack.Stop(); err != nil {
		t.Fatalf("failed to stop protocol stack: %v", err)
	}
	for id := range services {
		if !stopped[id] {
			t.Fatalf("service %s: freshly terminated service still running", id)
		}
	}
}

// Tests that services are restarted cleanly as new instances.
func TestServiceRestarts(t *testing.T) {
	stack, err := New(testNodeConfig())
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
			startHook: func(*p2p.Server) {
				if running {
					panic("already running")
				}
				running = true
				started++
			},
		}, nil
	}
	// Register the service and start the protocol stack
	if err := stack.Register(constructor); err != nil {
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
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Define a batch of good services
	services := map[string]InstrumentingWrapper{
		"A": InstrumentedServiceMakerA,
		"B": InstrumentedServiceMakerB,
		"C": InstrumentedServiceMakerC,
	}
	started := make(map[string]bool)
	for id, maker := range services {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func(*p2p.Server) { started[id] = true },
			}, nil
		}
		if err := stack.Register(maker(constructor)); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Register a service that fails to construct itself
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return nil, failure
	}
	if err := stack.Register(failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack and ensure none of the services get started
	for i := 0; i < 100; i++ {
		if err := stack.Start(); err != failure {
			t.Fatalf("iter %d: stack startup failure mismatch: have %v, want %v", i, err, failure)
		}
		for id := range services {
			if started[id] {
				t.Fatalf("service %s: started should not have", id)
			}
			delete(started, id)
		}
	}
}

// Tests that if a service fails to start, all others started before it will be
// shut down.
func TestServiceStartupAbortion(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of good services
	services := map[string]InstrumentingWrapper{
		"A": InstrumentedServiceMakerA,
		"B": InstrumentedServiceMakerB,
		"C": InstrumentedServiceMakerC,
	}
	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for id, maker := range services {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func(*p2p.Server) { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(maker(constructor)); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Register a service that fails to start
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return &InstrumentedService{
			start: failure,
		}, nil
	}
	if err := stack.Register(failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack and ensure all started services stop
	for i := 0; i < 100; i++ {
		if err := stack.Start(); err != failure {
			t.Fatalf("iter %d: stack startup failure mismatch: have %v, want %v", i, err, failure)
		}
		for id := range services {
			if started[id] && !stopped[id] {
				t.Fatalf("service %s: started but not stopped", id)
			}
			delete(started, id)
			delete(stopped, id)
		}
	}
}

// Tests that even if a registered service fails to shut down cleanly, it does
// not influece the rest of the shutdown invocations.
func TestServiceTerminationGuarantee(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of good services
	services := map[string]InstrumentingWrapper{
		"A": InstrumentedServiceMakerA,
		"B": InstrumentedServiceMakerB,
		"C": InstrumentedServiceMakerC,
	}
	started := make(map[string]bool)
	stopped := make(map[string]bool)

	for id, maker := range services {
		id := id // Closure for the constructor
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				startHook: func(*p2p.Server) { started[id] = true },
				stopHook:  func() { stopped[id] = true },
			}, nil
		}
		if err := stack.Register(maker(constructor)); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Register a service that fails to shot down cleanly
	failure := errors.New("fail")
	failer := func(*ServiceContext) (Service, error) {
		return &InstrumentedService{
			stop: failure,
		}, nil
	}
	if err := stack.Register(failer); err != nil {
		t.Fatalf("failer registration failed: %v", err)
	}
	// Start the protocol stack, and ensure that a failing shut down terminates all
	for i := 0; i < 100; i++ {
		// Start the stack and make sure all is online
		if err := stack.Start(); err != nil {
			t.Fatalf("iter %d: failed to start protocol stack: %v", i, err)
		}
		for id := range services {
			if !started[id] {
				t.Fatalf("iter %d, service %s: service not running", i, id)
			}
			if stopped[id] {
				t.Fatalf("iter %d, service %s: service already stopped", i, id)
			}
		}
		// Stop the stack, verify failure and check all terminations
		err := stack.Stop()
		if err, ok := err.(*StopError); !ok {
			t.Fatalf("iter %d: termination failure mismatch: have %v, want StopError", i, err)
		} else {
			failer := reflect.TypeOf(&InstrumentedService{})
			if err.Services[failer] != failure {
				t.Fatalf("iter %d: failer termination failure mismatch: have %v, want %v", i, err.Services[failer], failure)
			}
			if len(err.Services) != 1 {
				t.Fatalf("iter %d: failure count mismatch: have %d, want %d", i, len(err.Services), 1)
			}
		}
		for id := range services {
			if !stopped[id] {
				t.Fatalf("iter %d, service %s: service not terminated", i, id)
			}
			delete(started, id)
			delete(stopped, id)
		}
	}
}

// TestServiceRetrieval tests that individual services can be retrieved.
func TestServiceRetrieval(t *testing.T) {
	// Create a simple stack and register two service types
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	if err := stack.Register(NewNoopService); err != nil {
		t.Fatalf("noop service registration failed: %v", err)
	}
	if err := stack.Register(NewInstrumentedService); err != nil {
		t.Fatalf("instrumented service registration failed: %v", err)
	}
	// Make sure none of the services can be retrieved until started
	var noopServ *NoopService
	if err := stack.Service(&noopServ); err != ErrNodeStopped {
		t.Fatalf("noop service retrieval mismatch: have %v, want %v", err, ErrNodeStopped)
	}
	var instServ *InstrumentedService
	if err := stack.Service(&instServ); err != ErrNodeStopped {
		t.Fatalf("instrumented service retrieval mismatch: have %v, want %v", err, ErrNodeStopped)
	}
	// Start the stack and ensure everything is retrievable now
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start stack: %v", err)
	}
	defer stack.Stop()

	if err := stack.Service(&noopServ); err != nil {
		t.Fatalf("noop service retrieval mismatch: have %v, want %v", err, nil)
	}
	if err := stack.Service(&instServ); err != nil {
		t.Fatalf("instrumented service retrieval mismatch: have %v, want %v", err, nil)
	}
}

// Tests that all protocols defined by individual services get launched.
func TestProtocolGather(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of services with some configured number of protocols
	services := map[string]struct {
		Count int
		Maker InstrumentingWrapper
	}{
		"Zero Protocols":  {0, InstrumentedServiceMakerA},
		"Single Protocol": {1, InstrumentedServiceMakerB},
		"Many Protocols":  {25, InstrumentedServiceMakerC},
	}
	for id, config := range services {
		protocols := make([]p2p.Protocol, config.Count)
		for i := 0; i < len(protocols); i++ {
			protocols[i].Name = id
			protocols[i].Version = uint(i)
		}
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{
				protocols: protocols,
			}, nil
		}
		if err := stack.Register(config.Maker(constructor)); err != nil {
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
	for id, config := range services {
		for ver := 0; ver < config.Count; ver++ {
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

// Tests that all APIs defined by individual services get exposed.
func TestAPIGather(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Register a batch of services with some configured APIs
	calls := make(chan string, 1)
	makeAPI := func(result string) *OneMethodApi {
		return &OneMethodApi{fun: func() { calls <- result }}
	}
	services := map[string]struct {
		APIs  []rpc.API
		Maker InstrumentingWrapper
	}{
		"Zero APIs": {
			[]rpc.API{}, InstrumentedServiceMakerA},
		"Single API": {
			[]rpc.API{
				{Namespace: "single", Version: "1", Service: makeAPI("single.v1"), Public: true},
			}, InstrumentedServiceMakerB},
		"Many APIs": {
			[]rpc.API{
				{Namespace: "multi", Version: "1", Service: makeAPI("multi.v1"), Public: true},
				{Namespace: "multi.v2", Version: "2", Service: makeAPI("multi.v2"), Public: true},
				{Namespace: "multi.v2.nested", Version: "2", Service: makeAPI("multi.v2.nested"), Public: true},
			}, InstrumentedServiceMakerC},
	}

	for id, config := range services {
		config := config
		constructor := func(*ServiceContext) (Service, error) {
			return &InstrumentedService{apis: config.APIs}, nil
		}
		if err := stack.Register(config.Maker(constructor)); err != nil {
			t.Fatalf("service %s: registration failed: %v", id, err)
		}
	}
	// Start the services and ensure all API start successfully
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	defer stack.Stop()

	// Connect to the RPC server and verify the various registered endpoints
	client, err := stack.Attach()
	if err != nil {
		t.Fatalf("failed to connect to the inproc API server: %v", err)
	}
	defer client.Close()

	tests := []struct {
		Method string
		Result string
	}{
		{"single_theOneMethod", "single.v1"},
		{"multi_theOneMethod", "multi.v1"},
		{"multi.v2_theOneMethod", "multi.v2"},
		{"multi.v2.nested_theOneMethod", "multi.v2.nested"},
	}
	for i, test := range tests {
		if err := client.Call(nil, test.Method); err != nil {
			t.Errorf("test %d: API request failed: %v", i, err)
		}
		select {
		case result := <-calls:
			if result != test.Result {
				t.Errorf("test %d: result mismatch: have %s, want %s", i, result, test.Result)
			}
		case <-time.After(time.Second):
			t.Fatalf("test %d: rpc execution timeout", i)
		}
	}
}
