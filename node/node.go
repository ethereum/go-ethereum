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

// Package node represents the Ethereum protocol stack container.
package node

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	ErrDatadirUsed       = errors.New("datadir already used")
	ErrNodeStopped       = errors.New("node not started")
	ErrNodeRunning       = errors.New("node already running")
	ErrServiceUnknown    = errors.New("service not registered")
	ErrServiceRegistered = errors.New("service already registered")

	datadirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
)

// Node represents a P2P node into which arbitrary services might be registered.
type Node struct {
	datadir string                        // Path to the currently used data directory
	config  *p2p.Server                   // Configuration of the underlying P2P networking layer
	stack   map[string]ServiceConstructor // Protocol stack registered into this node
	emux    *event.TypeMux                // Event multiplexer used between the services of a stack

	running  *p2p.Server        // Currently running P2P networking layer
	services map[string]Service // Currently running services

	lock sync.RWMutex
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	// Ensure the data directory exists, failing if it cannot be created
	if conf.DataDir != "" {
		if err := os.MkdirAll(conf.DataDir, 0700); err != nil {
			return nil, err
		}
	}
	// Assemble the networking layer and the node itself
	nodeDbPath := ""
	if conf.DataDir != "" {
		nodeDbPath = filepath.Join(conf.DataDir, datadirNodeDatabase)
	}
	return &Node{
		datadir: conf.DataDir,
		config: &p2p.Server{
			PrivateKey:      conf.NodeKey(),
			Name:            conf.Name,
			Discovery:       !conf.NoDiscovery,
			BootstrapNodes:  conf.BootstrapNodes,
			StaticNodes:     conf.StaticNodes(),
			TrustedNodes:    conf.TrusterNodes(),
			NodeDatabase:    nodeDbPath,
			ListenAddr:      conf.ListenAddr,
			NAT:             conf.NAT,
			Dialer:          conf.Dialer,
			NoDial:          conf.NoDial,
			MaxPeers:        conf.MaxPeers,
			MaxPendingPeers: conf.MaxPendingPeers,
		},
		stack: make(map[string]ServiceConstructor),
		emux:  new(event.TypeMux),
	}, nil
}

// Register injects a new service into the node's stack.
func (n *Node) Register(id string, constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node is running or if the id is taken
	if n.running != nil {
		return ErrNodeRunning
	}
	if _, ok := n.stack[id]; ok {
		return ErrServiceRegistered
	}
	// Otherwise register the service and return
	n.stack[id] = constructor

	return nil
}

// Unregister removes a service from a node's stack. If the node is currently
// running, an error will be returned.
func (n *Node) Unregister(id string) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node is running, or if the service is unknown
	if n.running != nil {
		return ErrNodeRunning
	}
	if _, ok := n.stack[id]; !ok {
		return ErrServiceUnknown
	}
	// Otherwise drop the service and return
	delete(n.stack, id)
	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's already running
	if n.running != nil {
		return ErrNodeRunning
	}
	// Otherwise copy and specialize the P2P configuration
	running := new(p2p.Server)
	*running = *n.config

	ctx := &ServiceContext{
		dataDir:  n.datadir,
		EventMux: n.emux,
	}
	services := make(map[string]Service)
	for id, constructor := range n.stack {
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		services[id] = service
	}
	// Gather the protocols and start the freshly assembled P2P server
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	if err := running.Start(); err != nil {
		if errno, ok := err.(syscall.Errno); ok && datadirInUseErrnos[uint(errno)] {
			return ErrDatadirUsed
		}
		return err
	}
	// Start each of the services
	started := []string{}
	for id, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(); err != nil {
			for _, id := range started {
				services[id].Stop()
			}
			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, id)
	}
	// Finish initializing the startup
	n.services = services
	n.running = running

	return nil
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's not running
	if n.running == nil {
		return ErrNodeStopped
	}
	// Otherwise terminate all the services and the P2P server too
	failure := &StopError{
		Services: make(map[string]error),
	}
	for id, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[id] = err
		}
	}
	n.running.Stop()

	n.services = nil
	n.running = nil

	if len(failure.Services) > 0 {
		return failure
	}
	return nil
}

// Restart terminates a running node and boots up a new one in its place. If the
// node isn't running, an error is returned.
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.running
}

// Service retrieves a currently running services registered under a given id.
func (n *Node) Service(id string) Service {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.services == nil {
		return nil
	}
	return n.services[id]
}

// DataDir retrieves the current datadir used by the protocol stack.
func (n *Node) DataDir() string {
	return n.datadir
}

// EventMux retrieves the event multiplexer used by all the network services in
// the current protocol stack.
func (n *Node) EventMux() *event.TypeMux {
	return n.emux
}
