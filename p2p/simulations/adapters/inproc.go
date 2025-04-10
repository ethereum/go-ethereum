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

	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/gorilla/websocket"
)

// SimAdapter is a NodeAdapter which creates in-memory simulation nodes and
// connects them using in-memory net.Pipe connections
type SimAdapter struct {
	mtx        sync.RWMutex
	nodes      map[discover.NodeID]*SimNode
	lifecycles LifecycleConstructors
}

// NewSimAdapter creates a SimAdapter which is capable of running in-memory
// simulation nodes running any of the given services (the services to run on a
// particular node are passed to the NewNode function in the NodeConfig)
func NewSimAdapter(services LifecycleConstructors) *SimAdapter {
	return &SimAdapter{
		// nodes:      make(map[discover.NodeID]*SimNode),
		// lifecycles: lifecycles,
		nodes:      make(map[discover.NodeID]*SimNode),
		lifecycles: services,
	}
}

// Name returns the name of the adapter for logging purposes
func (sa *SimAdapter) Name() string {
	return "sim-adapter"
}

// NewNode returns a new SimNode using the given config
func (sa *SimAdapter) NewNode(config *NodeConfig) (Node, error) {
	sa.mtx.Lock()
	defer sa.mtx.Unlock()

	// check a node with the ID doesn't already exist
	id := config.ID
	if _, exists := sa.nodes[id]; exists {
		return nil, fmt.Errorf("node already exists: %s", id)
	}

	// check the services are valid
	if len(config.Lifecycles) == 0 {
		return nil, errors.New("node must have at least one service")
	}
	for _, service := range config.Lifecycles {
		if _, exists := sa.lifecycles[service]; !exists {
			return nil, fmt.Errorf("unknown node service %q", service)
		}
	}

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			PrivateKey:      config.PrivateKey,
			MaxPeers:        math.MaxInt32,
			NoDiscovery:     true,
			Dialer:          sa,
			EnableMsgEvents: true,
		},
		Logger: log.New("node.id", id.String()),
	})
	if err != nil {
		return nil, err
	}

	simNode := &SimNode{
		ID:        id,
		config:    config,
		node:      n,
		adapter:   sa,
		running:   make(map[string]node.Lifecycle),
		connected: make(map[discover.NodeID]bool),
	}
	sa.nodes[id] = simNode
	return simNode, nil
}

// Dial implements the p2p.NodeDialer interface by connecting to the node using
// an in-memory net.Pipe connection
func (sa *SimAdapter) Dial(dest *discover.Node) (conn net.Conn, err error) {
	node, ok := sa.GetNode(dest.ID)
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", dest.ID)
	}
	if node.connected[dest.ID] {
		return nil, fmt.Errorf("dialed node: %s", dest.ID)
	}
	srv := node.Server()
	if srv == nil {
		return nil, fmt.Errorf("node not running: %s", dest.ID)
	}
	pipe1, pipe2 := net.Pipe()
	go srv.SetupConn(pipe1, 0, nil)
	node.connected[dest.ID] = true
	return pipe2, nil
}

// DialRPC implements the RPCDialer interface by creating an in-memory RPC
// client of the given node
func (sa *SimAdapter) DialRPC(id discover.NodeID) (*rpc.Client, error) {
	node, ok := sa.GetNode(id)
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", id)
	}
	return node.node.Attach()
}

// GetNode returns the node with the given ID if it exists
func (sa *SimAdapter) GetNode(id discover.NodeID) (*SimNode, bool) {
	sa.mtx.RLock()
	defer sa.mtx.RUnlock()
	node, ok := sa.nodes[id]
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
	running      map[string]node.Lifecycle
	client       *rpc.Client
	registerOnce sync.Once
	connected    map[discover.NodeID]bool
}

// Close closes the underlaying node.Node to release
// acquired resources.
func (sn *SimNode) Close() error {
	return sn.node.Close()
}

// Addr returns the node's discovery address
func (sn *SimNode) Addr() []byte {
	return []byte(sn.Node().String())
}

// Node returns a discover.Node representing the SimNode
func (sn *SimNode) Node() *discover.Node {
	return discover.NewNode(sn.ID, net.IP{127, 0, 0, 1}, 30303, 30303)
}

// Client returns an rpc.Client which can be used to communicate with the
// underlying services (it is set once the node has started)
func (sn *SimNode) Client() (*rpc.Client, error) {
	sn.lock.RLock()
	defer sn.lock.RUnlock()
	if sn.client == nil {
		return nil, errors.New("node not started")
	}
	return sn.client, nil
}

// ServeRPC serves RPC requests over the given connection by creating an
// in-memory client to the node's RPC server
func (sn *SimNode) ServeRPC(conn *websocket.Conn) error {
	handler, err := sn.node.RPCHandler()
	if err != nil {
		return err
	}
	codec := rpc.NewFuncCodec(conn, conn.WriteJSON, conn.ReadJSON)
	handler.ServeCodec(codec, 0)
	return nil
}

// Snapshots creates snapshots of the services by calling the
// simulation_snapshot RPC method
func (sn *SimNode) Snapshots() (map[string][]byte, error) {
	sn.lock.RLock()
	services := make(map[string]node.Lifecycle, len(sn.running))
	for name, service := range sn.running {
		services[name] = service
	}
	sn.lock.RUnlock()
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
func (sn *SimNode) Start(snapshots map[string][]byte) error {
	// ensure we only register the services once in the case of the node
	// being stopped and then started again
	var regErr error
	sn.registerOnce.Do(func() {
		for _, name := range sn.config.Lifecycles {
			ctx := &ServiceContext{
				RPCDialer: sn.adapter,
				Config:    sn.config,
			}
			if snapshots != nil {
				ctx.Snapshot = snapshots[name]
			}
			serviceFunc := sn.adapter.lifecycles[name]
			service, err := serviceFunc(ctx, sn.node)
			if err != nil {

				regErr = err
				return
			}
			// if the service has already been registered, don't register it again.
			if _, ok := sn.running[name]; ok {
				continue
			}
			sn.running[name] = service
			sn.node.RegisterLifecycle(service)
		}
	})
	if regErr != nil {
		return regErr
	}

	if err := sn.node.Start(); err != nil {
		return err
	}

	// create an in-process RPC client
	client, err := sn.node.Attach()
	if err != nil {
		return err
	}

	sn.lock.Lock()
	sn.client = client
	sn.lock.Unlock()

	return nil
}

// Stop closes the RPC client and stops the underlying devp2p node
func (sn *SimNode) Stop() error {
	sn.lock.Lock()
	if sn.client != nil {
		sn.client.Close()
		sn.client = nil
	}
	sn.lock.Unlock()
	return sn.node.Close()
}

// Services returns a copy of the underlying services
func (sn *SimNode) Services() []node.Lifecycle {
	sn.lock.RLock()
	defer sn.lock.RUnlock()
	services := make([]node.Lifecycle, 0, len(sn.running))
	for _, service := range sn.running {
		services = append(services, service)
	}
	return services
}

// Server returns the underlying p2p.Server
func (sn *SimNode) Server() *p2p.Server {
	return sn.node.Server()
}

// SubscribeEvents subscribes the given channel to peer events from the
// underlying p2p.Server
func (sn *SimNode) SubscribeEvents(ch chan *p2p.PeerEvent) event.Subscription {
	srv := sn.Server()
	if srv == nil {
		panic("node not running")
	}
	return srv.SubscribeEvents(ch)
}

// NodeInfo returns information about the node
func (sn *SimNode) NodeInfo() *p2p.NodeInfo {
	server := sn.Server()
	if server.Running == false {
		return &p2p.NodeInfo{
			ID:    sn.ID.String(),
			Enode: sn.Node().String(),
		}
	}
	return server.NodeInfo()
}
