// Copyright 2017 The go-ethereum Authors
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

package adapters

import (
	"errors"
	"fmt"
	"math"
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
)

// SimAdapter is a NodeAdapter which creates in-memory simulation nodes and
// connects them using in-memory net.Pipe connections
type SimAdapter struct {
	mtx      sync.RWMutex
	nodes    map[discover.NodeID]*SimNode
	services map[string]ServiceFunc
}

// NewSimAdapter creates a SimAdapter which is capable of running in-memory
// simulation nodes running any of the given services (the services to run on a
// particular node are passed to the NewNode function in the NodeConfig)
func NewSimAdapter(services map[string]ServiceFunc) *SimAdapter {
	return &SimAdapter{
		nodes:    make(map[discover.NodeID]*SimNode),
		services: services,
	}
}

// Name returns the name of the adapter for logging purposes
func (s *SimAdapter) Name() string {
	return "sim-adapter"
}

// NewNode returns a new SimNode using the given config
func (s *SimAdapter) NewNode(config *NodeConfig) (Node, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// check a node with the ID doesn't already exist
	id := config.ID
	if _, exists := s.nodes[id]; exists {
		return nil, fmt.Errorf("node already exists: %s", id)
	}

	// check the services are valid
	if len(config.Services) == 0 {
		return nil, errors.New("node must have at least one service")
	}
	for _, service := range config.Services {
		if _, exists := s.services[service]; !exists {
			return nil, fmt.Errorf("unknown node service %q", service)
		}
	}

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			PrivateKey:      config.PrivateKey,
			MaxPeers:        math.MaxInt32,
			NoDiscovery:     true,
			Dialer:          s,
			EnableMsgEvents: true,
		},
		NoUSB:  true,
		Logger: log.New("node.id", id.String()),
	})
	if err != nil {
		return nil, err
	}

	simNode := &SimNode{
		ID:      id,
		config:  config,
		node:    n,
		adapter: s,
		running: make(map[string]node.Service),
	}
	s.nodes[id] = simNode
	return simNode, nil
}

// Dial implements the p2p.NodeDialer interface by connecting to the node using
// an in-memory net.Pipe connection
func (s *SimAdapter) Dial(dest *discover.Node) (conn net.Conn, err error) {
	node, ok := s.GetNode(dest.ID)
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", dest.ID)
	}
	srv := node.Server()
	if srv == nil {
		return nil, fmt.Errorf("node not running: %s", dest.ID)
	}
	pipe1, pipe2 := net.Pipe()
	go srv.SetupConn(pipe1, 0, nil)
	return pipe2, nil
}

// DialRPC implements the RPCDialer interface by creating an in-memory RPC
// client of the given node
func (s *SimAdapter) DialRPC(id discover.NodeID) (*rpc.Client, error) {
	node, ok := s.GetNode(id)
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", id)
	}
	handler, err := node.node.RPCHandler()
	if err != nil {
		return nil, err
	}
	return rpc.DialInProc(handler), nil
}

// GetNode returns the node with the given ID if it exists
func (s *SimAdapter) GetNode(id discover.NodeID) (*SimNode, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	node, ok := s.nodes[id]
	return node, ok
}

// SimNode is an in-memory simulation node which connects to other nodes using
// an in-memory net.Pipe connection (see SimAdapter.Dial), running devp2p
// protocols directly over that pipe
type SimNode struct {
	lock         sync.RWMutex
	ID           discover.NodeID
	config       *NodeConfig
	adapter      *SimAdapter
	node         *node.Node
	running      map[string]node.Service
	client       *rpc.Client
	registerOnce sync.Once
}

// Addr returns the node's discovery address
func (self *SimNode) Addr() []byte {
	return []byte(self.Node().String())
}

// Node returns a discover.Node representing the SimNode
func (self *SimNode) Node() *discover.Node {
	return discover.NewNode(self.ID, net.IP{127, 0, 0, 1}, 30303, 30303)
}

// Client returns an rpc.Client which can be used to communicate with the
// underlying services (it is set once the node has started)
func (self *SimNode) Client() (*rpc.Client, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.client == nil {
		return nil, errors.New("node not started")
	}
	return self.client, nil
}

// ServeRPC serves RPC requests over the given connection by creating an
// in-memory client to the node's RPC server
func (self *SimNode) ServeRPC(conn net.Conn) error {
	handler, err := self.node.RPCHandler()
	if err != nil {
		return err
	}
	handler.ServeCodec(rpc.NewJSONCodec(conn), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
	return nil
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (self *SimNode) Snapshots() (map[string][]byte, error) {
	self.lock.RLock()
	services := make(map[string]node.Service, len(self.running))
	for name, service := range self.running {
		services[name] = service
	}
	self.lock.RUnlock()
	if len(services) == 0 {
		return nil, errors.New("no running services")
	}
	snapshots := make(map[string][]byte)
	for name, service := range services {
		if s, ok := service.(interface {
			Snapshot() ([]byte, error)
		}); ok {
			snap, err := s.Snapshot()
			if err != nil {
				return nil, err
			}
			snapshots[name] = snap
		}
	}
	return snapshots, nil
}

// Start registers the services and starts the underlying devp2p node
func (self *SimNode) Start(snapshots map[string][]byte) error {
	newService := func(name string) func(ctx *node.ServiceContext) (node.Service, error) {
		return func(nodeCtx *node.ServiceContext) (node.Service, error) {
			ctx := &ServiceContext{
				RPCDialer:   self.adapter,
				NodeContext: nodeCtx,
				Config:      self.config,
			}
			if snapshots != nil {
				ctx.Snapshot = snapshots[name]
			}
			serviceFunc := self.adapter.services[name]
			service, err := serviceFunc(ctx)
			if err != nil {
				return nil, err
			}
			self.running[name] = service
			return service, nil
		}
	}

	// ensure we only register the services once in the case of the node
	// being stopped and then started again
	var regErr error
	self.registerOnce.Do(func() {
		for _, name := range self.config.Services {
			if err := self.node.Register(newService(name)); err != nil {
				regErr = err
				return
			}
		}
	})
	if regErr != nil {
		return regErr
	}

	if err := self.node.Start(); err != nil {
		return err
	}

	// create an in-process RPC client
	handler, err := self.node.RPCHandler()
	if err != nil {
		return err
	}

	self.lock.Lock()
	self.client = rpc.DialInProc(handler)
	self.lock.Unlock()

	return nil
}

// Stop closes the RPC client and stops the underlying devp2p node
func (self *SimNode) Stop() error {
	self.lock.Lock()
	if self.client != nil {
		self.client.Close()
		self.client = nil
	}
	self.lock.Unlock()
	return self.node.Stop()
}

// Services returns a copy of the underlying services
func (self *SimNode) Services() []node.Service {
	self.lock.RLock()
	defer self.lock.RUnlock()
	services := make([]node.Service, 0, len(self.running))
	for _, service := range self.running {
		services = append(services, service)
	}
	return services
}

// Server returns the underlying p2p.Server
func (self *SimNode) Server() *p2p.Server {
	return self.node.Server()
}

// SubscribeEvents subscribes the given channel to peer events from the
// underlying p2p.Server
func (self *SimNode) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	srv := self.Server()
	if srv == nil {
		panic("node not running")
	}
	return srv.SubscribeEvents(ch)
}

// NodeInfo returns information about the node
func (self *SimNode) NodeInfo() *p2p.NodeInfo {
	server := self.Server()
	if server == nil {
		return &p2p.NodeInfo{
			ID:    self.ID.String(),
			Enode: self.Node().String(),
		}
	}
	return server.NodeInfo()
}
