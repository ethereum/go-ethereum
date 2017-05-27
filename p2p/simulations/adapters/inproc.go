// Copyright 2016 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
)

// SimAdapter is a NodeAdapter which creates in-memory nodes and connects them
// using an in-memory p2p.MsgReadWriter pipe
type SimAdapter struct {
	mtx      sync.RWMutex
	nodes    map[discover.NodeID]*SimNode
	services map[string]ServiceFunc
}

// NewSimAdapter creates a SimAdapter which is capable of running in-memory
// nodes running any of the given services (the service to run on a particular
// node is passed to the NewNode function in the NodeConfig)
func NewSimAdapter(services map[string]ServiceFunc) *SimAdapter {
	return &SimAdapter{
		nodes:    make(map[discover.NodeID]*SimNode),
		services: services,
	}
}

// Name returns the name of the adapter for logging purpoeses
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

	node := &SimNode{
		ID:      id,
		config:  config,
		adapter: s,
	}
	s.nodes[id] = node
	return node, nil
}

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

// GetNode returns the node with the given ID if it exists
func (s *SimAdapter) GetNode(id discover.NodeID) (*SimNode, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	node, ok := s.nodes[id]
	return node, ok
}

// SimNode is an in-memory node which connects to other SimNodes using an
// in-memory p2p.MsgReadWriter pipe, running an underlying service protocol
// directly over that pipe.
//
// It implements the p2p.Server interface so it can be used transparently
// by the underlying service.
type SimNode struct {
	lock    sync.RWMutex
	ID      discover.NodeID
	config  *NodeConfig
	adapter *SimAdapter
	node    *node.Node
	running []node.Service
	client  *rpc.Client
	rpcMux  *rpcMux
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
// underlying service (it is set once the node has started)
func (self *SimNode) Client() (*rpc.Client, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.client == nil {
		return nil, errors.New("RPC not started")
	}
	return self.client, nil
}

// ServeRPC serves RPC requests over the given connection using the node's
// RPC multiplexer
func (self *SimNode) ServeRPC(conn net.Conn) error {
	self.lock.Lock()
	mux := self.rpcMux
	self.lock.Unlock()
	if mux == nil {
		return errors.New("RPC not started")
	}
	mux.Serve(conn)
	return nil
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (self *SimNode) Snapshots() (map[string][]byte, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.client == nil {
		return nil, errors.New("RPC not started")
	}
	var snapshots map[string][]byte
	return snapshots, self.client.Call(&snapshots, "simulation_snapshot")
}

// Start starts the RPC handler and the underlying service
func (self *SimNode) Start(snapshots map[string][]byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.node != nil {
		return errors.New("node already started")
	}

	newService := func(name string) func(ctx *node.ServiceContext) (node.Service, error) {
		return func(ctx *node.ServiceContext) (node.Service, error) {
			var snapshot []byte
			if snapshots != nil {
				snapshot = snapshots[name]
			}
			serviceFunc := self.adapter.services[name]
			service := serviceFunc(self.ID, snapshot)
			self.running = append(self.running, service)
			return service, nil
		}
	}

	node, err := node.New(&node.Config{
		P2P: p2p.Config{
			PrivateKey:      self.config.PrivateKey,
			MaxPeers:        math.MaxInt32,
			NoDiscovery:     true,
			Dialer:          self.adapter,
			EnableMsgEvents: false,
		},
		NoUSB: true,
	})
	if err != nil {
		return err
	}

	for _, name := range self.config.Services {
		if err := node.Register(newService(name)); err != nil {
			return err
		}
	}

	if err := node.Start(); err != nil {
		return err
	}

	handler, err := node.RPCHandler()
	if err != nil {
		return err
	}

	// create an in-process RPC multiplexer
	pipe1, pipe2 := net.Pipe()
	go handler.ServeCodec(rpc.NewJSONCodec(pipe1), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
	self.rpcMux = newRPCMux(pipe2)

	// create an in-process RPC client
	self.client = self.rpcMux.Client()

	self.node = node

	return nil
}

func (self *SimNode) Stop() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.node == nil {
		return nil
	}
	if err := self.node.Stop(); err != nil {
		return err
	}
	self.node = nil
	return nil
}

// Services returns the underlying services
func (self *SimNode) Services() []node.Service {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.running
}

func (self *SimNode) Server() *p2p.Server {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.node == nil {
		return nil
	}
	return self.node.Server()
}

func (self *SimNode) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	srv := self.Server()
	if srv == nil {
		panic("node not running")
	}
	return srv.SubscribeEvents(ch)
}

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
