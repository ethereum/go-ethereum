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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	ErrDatadirUsed    = errors.New("datadir already used")
	ErrNodeStopped    = errors.New("node not started")
	ErrNodeRunning    = errors.New("node already running")
	ErrServiceUnknown = errors.New("unknown service")

	datadirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
)

// Node represents a P2P node into which arbitrary (uniquely typed) services might
// be registered.
type Node struct {
	datadir  string         // Path to the currently used data directory
	eventmux *event.TypeMux // Event multiplexer used between the services of a stack

	serverConfig *p2p.Server // Configuration of the underlying P2P networking layer
	server       *p2p.Server // Currently running P2P networking layer

	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	rpcAPIs     []rpc.API    // List of APIs currently provided by the node
	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpCors      string       // HTTP RPC Cross-Origin Resource Sharing header
	httpListener  net.Listener // HTTP RPC listener socket to server API requests
	httpHandler   *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint  string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsWhitelist []string     // Websocket RPC modules to allow through this endpoint
	wsCors      string       // Websocket RPC Cross-Origin Resource Sharing header
	wsListener  net.Listener // Websocket RPC listener socket to server API requests
	wsHandler   *rpc.Server  // Websocket RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications
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
		serverConfig: &p2p.Server{
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
		serviceFuncs:  []ServiceConstructor{},
		ipcEndpoint:   conf.IpcEndpoint(),
		httpEndpoint:  conf.HttpEndpoint(),
		httpWhitelist: conf.HttpModules,
		httpCors:      conf.HttpCors,
		wsEndpoint:    conf.WsEndpoint(),
		wsWhitelist:   conf.WsModules,
		wsCors:        conf.WsCors,
		eventmux:      new(event.TypeMux),
	}, nil
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's already running
	if n.server != nil {
		return ErrNodeRunning
	}
	// Otherwise copy and specialize the P2P configuration
	running := new(p2p.Server)
	*running = *n.serverConfig

	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			datadir:  n.datadir,
			services: make(map[reflect.Type]Service),
			EventMux: n.eventmux,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
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
	started := []reflect.Type{}
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(running); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			running.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}
	// Lastly start the configured RPC interfaces
	if err := n.startRPC(services); err != nil {
		for _, service := range services {
			service.Stop()
		}
		running.Stop()
		return err
	}
	// Finish initializing the startup
	n.services = services
	n.server = running
	n.stop = make(chan struct{})

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services map[reflect.Type]Service) error {
	// Gather all the possible APIs to surface
	apis := n.apis()
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}
	// Start the various API endpoints, terminating all in case of errors
	if err := n.startIPC(apis); err != nil {
		return err
	}
	if err := n.startHTTP(n.httpEndpoint, apis, n.httpWhitelist, n.httpCors); err != nil {
		n.stopIPC()
		return err
	}
	if err := n.startWS(n.wsEndpoint, apis, n.wsWhitelist, n.wsCors); err != nil {
		n.stopHTTP()
		n.stopIPC()
		return err
	}
	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC(apis []rpc.API) error {
	// Short circuit if the IPC endpoint isn't being exposed
	if n.ipcEndpoint == "" {
		return nil
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		glog.V(logger.Debug).Infof("IPC registered %T under '%s'", api.Service, api.Namespace)
	}
	// All APIs registered, start the IPC listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = rpc.CreateIPCListener(n.ipcEndpoint); err != nil {
		return err
	}
	go func() {
		glog.V(logger.Info).Infof("IPC endpoint opened: %s", n.ipcEndpoint)

		for {
			conn, err := listener.Accept()
			if err != nil {
				// Terminate if the listener was closed
				n.lock.RLock()
				closed := n.ipcListener == nil
				n.lock.RUnlock()
				if closed {
					return
				}
				// Not closed, just some error; report and continue
				glog.V(logger.Error).Infof("IPC accept failed: %v", err)
				continue
			}
			go handler.ServeCodec(rpc.NewJSONCodec(conn))
		}
	}()
	// All listeners booted successfully
	n.ipcListener = listener
	n.ipcHandler = handler

	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil

		glog.V(logger.Info).Infof("IPC endpoint closed: %s", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, apis []rpc.API, modules []string, cors string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			glog.V(logger.Debug).Infof("HTTP registered %T under '%s'", api.Service, api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, handler).Serve(listener)
	glog.V(logger.Info).Infof("HTTP endpoint opened: http://%s", endpoint)

	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler
	n.httpCors = cors

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		glog.V(logger.Info).Infof("HTTP endpoint closed: http://%s", n.httpEndpoint)
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, apis []rpc.API, modules []string, cors string) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			glog.V(logger.Debug).Infof("WebSocket registered %T under '%s'", api.Service, api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewWSServer(cors, handler).Serve(listener)
	glog.V(logger.Info).Infof("WebSocket endpoint opened: ws://%s", endpoint)

	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler
	n.wsCors = cors

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		glog.V(logger.Info).Infof("WebSocket endpoint closed: ws://%s", n.wsEndpoint)
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}
	// Otherwise terminate the API, all services and the P2P server too
	n.stopWS()
	n.stopHTTP()
	n.stopIPC()
	n.rpcAPIs = nil

	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.server.Stop()

	n.services = nil
	n.server = nil
	close(n.stop)

	if len(failure.Services) > 0 {
		return failure
	}
	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		return
	}
	stop := n.stop
	n.lock.RUnlock()

	<-stop
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

	return n.server
}

// Service retrieves a currently running service registered of a specific type.
func (n *Node) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// DataDir retrieves the current datadir used by the protocol stack.
func (n *Node) DataDir() string {
	return n.datadir
}

// IpcEndpoint retrieves the current IPC endpoint used by the protocol stack.
func (n *Node) IpcEndpoint() string {
	return n.ipcEndpoint
}

// EventMux retrieves the event multiplexer used by all the network services in
// the current protocol stack.
func (n *Node) EventMux() *event.TypeMux {
	return n.eventmux
}

// apis returns the collection of RPC descriptors this node offers.
func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(n),
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPublicAdminAPI(n),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   debug.Handler,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(n),
			Public:    true,
		}, {
			Namespace: "web3",
			Version:   "1.0",
			Service:   NewPublicWeb3API(n),
			Public:    true,
		},
	}
}
