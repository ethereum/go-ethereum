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
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"

	"github.com/stretchr/testify/assert"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
)

func testNodeConfig() *Config {
	return &Config{
		Name: "test node",
		P2P:  p2p.Config{PrivateKey: testNodeKey},
	}
}

// Tests that an empty protocol stack can be started and stopped.
func TestNodeLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

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
	defer original.Close()

	if err := original.Start(); err != nil {
		t.Fatalf("failed to start original protocol stack: %v", err)
	}
	defer original.Stop()

	// Create a second node based on the same data directory and ensure failure
	duplicate, err := New(&Config{DataDir: dir})
	if err != nil {
		t.Fatalf("failed to create duplicate protocol stack: %v", err)
	}
	defer duplicate.Close()

	if err := duplicate.Start(); err != ErrDatadirUsed {
		t.Fatalf("duplicate datadir failure mismatch: have %v, want %v", err, ErrDatadirUsed)
	}
}

// Tests whether a Lifecycle can be registered.
func TestLifecycleRegistry(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

	noop := NewNoop()
	stack.RegisterLifecycle(noop)

	if _, exists := stack.lifecycles[reflect.TypeOf(noop)]; !exists {
		t.Fatalf("lifecycle was not properly registered on the node, %v", err)
	}
}

// Tests that registered Lifecycles get started and stopped correctly.
func TestLifecycleLifeCycle(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	// Create a batch of instrumented services
	lifecycles := map[string]Lifecycle{
		"A": &InstrumentedServiceA{
			InstrumentedService{
				startHook: func() { started["A"] = true },
				stopHook:  func() { stopped["A"] = true },
			},
		},
		"B": &InstrumentedServiceB{
			InstrumentedService{
				startHook: func() { started["B"] = true },
				stopHook:  func() { stopped["B"] = true },
			},
		},
		"C": &InstrumentedServiceC{
			InstrumentedService{
				startHook: func() { started["C"] = true },
				stopHook:  func() { stopped["C"] = true },
			},
		},
	}
	// register lifecycles on node
	for _, lifecycle := range lifecycles {
		stack.RegisterLifecycle(lifecycle)
	}
	// Start the node and check that all services are running
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	for id := range lifecycles {
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
	for id := range lifecycles {
		if !stopped[id] {
			t.Fatalf("service %s: freshly terminated service still running", id)
		}
	}
}

// Tests that if a Lifecycle fails to start, all others started before it will be
// shut down.
func TestLifecycleStartupAbortion(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	// Create a batch of instrumented services
	lifecycles := map[string]Lifecycle{
		"A": &InstrumentedServiceA{
			InstrumentedService{
				startHook: func() { started["A"] = true },
				stopHook:  func() { stopped["A"] = true },
			},
		},
		"B": &InstrumentedServiceB{
			InstrumentedService{
				startHook: func() { started["B"] = true },
				stopHook:  func() { stopped["B"] = true },
			},
		},
		"C": &InstrumentedServiceC{
			InstrumentedService{
				startHook: func() { started["C"] = true },
				stopHook:  func() { stopped["C"] = true },
			},
		},
	}
	// register lifecycles on node
	for _, lifecycle := range lifecycles {
		stack.RegisterLifecycle(lifecycle)
	}

	// Register a service that fails to construct itself
	failure := errors.New("fail")
	failer := &InstrumentedService{start: failure}
	stack.RegisterLifecycle(failer)

	// Start the protocol stack and ensure all started services stop
	for i := 0; i < 100; i++ {
		if err := stack.Start(); err != failure {
			t.Fatalf("iter %d: stack startup failure mismatch: have %v, want %v", i, err, failure)
		}
		for id := range lifecycles {
			if started[id] && !stopped[id] {
				t.Fatalf("service %s: started but not stopped", id)
			}
			delete(started, id)
			delete(stopped, id)
		}
	}
}

// Tests that even if a registered Lifecycle fails to shut down cleanly, it does
// not influence the rest of the shutdown invocations.
func TestLifecycleTerminationGuarantee(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

	started := make(map[string]bool)
	stopped := make(map[string]bool)

	// Create a batch of instrumented services
	lifecycles := map[string]Lifecycle{
		"A": &InstrumentedServiceA{
			InstrumentedService{
				startHook: func() { started["A"] = true },
				stopHook:  func() { stopped["A"] = true },
			},
		},
		"B": &InstrumentedServiceB{
			InstrumentedService{
				startHook: func() { started["B"] = true },
				stopHook:  func() { stopped["B"] = true },
			},
		},
		"C": &InstrumentedServiceC{
			InstrumentedService{
				startHook: func() { started["C"] = true },
				stopHook:  func() { stopped["C"] = true },
			},
		},
	}
	// register lifecycles on node
	for _, lifecycle := range lifecycles {
		stack.RegisterLifecycle(lifecycle)
	}

	// Register a service that fails to shot down cleanly
	failure := errors.New("fail")
	failer := &InstrumentedService{stop: failure}
	stack.RegisterLifecycle(failer)

	// Start the protocol stack, and ensure that a failing shut down terminates all // TODO, deleting loop because constructors no longer stored on node.
	// Start the stack and make sure all is online
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start protocol stack: %v", err)
	}
	for id := range lifecycles {
		if !started[id] {
			t.Fatalf("service %s: service not running", id)
		}
		if stopped[id] {
			t.Fatalf("service %s: service already stopped", id)
		}
	}
	// Stop the stack, verify failure and check all terminations
	err = stack.Stop()
	if err, ok := err.(*StopError); !ok {
		t.Fatalf("termination failure mismatch: have %v, want StopError", err)
	} else {
		failer := reflect.TypeOf(&InstrumentedService{})
		if err.Services[failer] != failure {
			t.Fatalf("failer termination failure mismatch: have %v, want %v", err.Services[failer], failure)
		}
		if len(err.Services) != 1 {
			t.Fatalf("failure count mismatch: have %d, want %d", len(err.Services), 1)
		}
	}
	for id := range lifecycles {
		if !stopped[id] {
			t.Fatalf("service %s: service not terminated", id)
		}
		delete(started, id)
		delete(stopped, id)
	}

	stack.server = &p2p.Server{}
	stack.server.PrivateKey = testNodeKey
}

// TestLifecycleRetrieval tests that individual services can be retrieved.
func TestLifecycleRetrieval(t *testing.T) {
	// Create a simple stack and register two service types
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()

	noop := NewNoop()
	stack.RegisterLifecycle(noop)

	is, err := NewInstrumentedService()
	if err != nil {
		t.Fatalf("instrumented service creation failed: %v", err)
	}
	stack.RegisterLifecycle(is)

	// Make sure none of the services can be retrieved until started
	var noopServ *Noop
	if err := stack.Lifecycle(&noopServ); err != ErrNodeStopped {
		t.Fatalf("noop service retrieval mismatch: have %v, want %v", err, ErrNodeStopped)
	}
	var instServ *InstrumentedService
	if err := stack.Lifecycle(&instServ); err != ErrNodeStopped {
		t.Fatalf("instrumented service retrieval mismatch: have %v, want %v", err, ErrNodeStopped)
	}
	// Start the stack and ensure everything is retrievable now
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start stack: %v", err)
	}
	defer stack.Stop()

	if err := stack.Lifecycle(&noopServ); err != nil {
		t.Fatalf("noop service retrieval mismatch: have %v, want %v", err, nil)
	}
	if err := stack.Lifecycle(&instServ); err != nil {
		t.Fatalf("instrumented service retrieval mismatch: have %v, want %v", err, nil)
	}
}

// Tests whether websocket requests can be handled on the same port as a regular http server
func TestWebsocketHTTPOnSamePort_WebsocketRequest(t *testing.T) {
	node := startHTTP(t)
	defer node.Close()

	wsReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:7453", nil)
	if err != nil {
		t.Error("could not issue new http request ", err)
	}
	wsReq.Header.Set("Connection", "upgrade")
	wsReq.Header.Set("Upgrade", "websocket")
	wsReq.Header.Set("Sec-WebSocket-Version", "13")
	wsReq.Header.Set("Sec-Websocket-Key", "SGVsbG8sIHdvcmxkIQ==")

	resp := doHTTPRequest(t, wsReq)
	assert.Equal(t, "websocket", resp.Header.Get("Upgrade"))
}

// Tests whether http requests can be handled successfully
func TestWebsocketHTTPOnSamePort_HTTPRequest(t *testing.T) {
	node := startHTTP(t)
	defer node.Close()

	httpReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:7453", nil)
	if err != nil {
		t.Error("could not issue new http request ", err)
	}
	httpReq.Header.Set("Accept-Encoding", "gzip")

	resp := doHTTPRequest(t, httpReq)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
}

func startHTTP(t *testing.T) *Node {
	conf := &Config{
		HTTPHost: "127.0.0.1",
		HTTPPort: 7453,
		WSHost:   "127.0.0.1",
		WSPort:   7453,
	}
	node, err := New(conf)
	if err != nil {
		t.Error("could not create a new node ", err)
	}
	err = node.Start()
	if err != nil {
		t.Error("could not start http service on node ", err)
	}

	return node
}

func doHTTPRequest(t *testing.T, req *http.Request) *http.Response {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("could not issue a GET request to the given endpoint", err)

	}
	return resp
}
