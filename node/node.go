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
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/tsdb/fileutil"
)

// Node is a container on which services can be registered.
type Node struct {
	eventmux *event.TypeMux // Event multiplexer used between the services of a stack
	config   *Config
	accman   *accounts.Manager

	ephemeralKeystore string            // if non-empty, the key directory that will be removed by Stop
	instanceDirLock   fileutil.Releaser // prevents concurrent use of instance directory

	server *p2p.Server // Currently running P2P networking layer

	ServiceContext *ServiceContext

	lifecycles []Lifecycle // All registered backends, services, and auxiliary services that have a lifecycle

	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	httpServers []*HTTPServer // Stores information about all http servers (if any), including http, ws, and graphql

	ipc *HTTPServer // Stores information about the ipc http server

	stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex

	log log.Logger
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	// Copy config and resolve the datadir so future changes to the current
	// working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absdatadir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absdatadir
	}
	// Ensure that the instance name doesn't cause weird conflicts with
	// other files in the data directory.
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	if conf.Name == datadirDefaultKeyStore {
		return nil, errors.New(`Config.Name cannot be "` + datadirDefaultKeyStore + `"`)
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name cannot end in ".ipc"`)
	}
	// Ensure that the AccountManager method works before the node has started.
	// We rely on this in cmd/geth.
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	if conf.Logger == nil {
		conf.Logger = log.New()
	}
	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	node := &Node{
		accman:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		ServiceContext: &ServiceContext{
			Config: *conf,
		},
		httpServers: make([]*HTTPServer, 0),
		ipc: &HTTPServer{
			endpoint: conf.IPCEndpoint(),
		},
		eventmux: new(event.TypeMux),
		log:      conf.Logger,
	}

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	node.server = &p2p.Server{Config: conf.P2P} // TODO add init step for p2p server
	node.server.Config.PrivateKey = node.config.NodeKey()
	node.server.Config.Name = node.config.NodeName()
	node.server.Config.Logger = node.log
	if node.server.Config.StaticNodes == nil {
		node.server.Config.StaticNodes = node.config.StaticNodes()
	}
	if node.server.Config.TrustedNodes == nil {
		node.server.Config.TrustedNodes = node.config.TrustedNodes()
	}
	if node.server.Config.NodeDatabase == "" {
		node.server.Config.NodeDatabase = node.config.NodeDB()
	}
	// Configure service context
	node.ServiceContext.EventMux = node.eventmux
	node.ServiceContext.AccountManager = node.accman
	// Configure HTTP server(s)
	if conf.HTTPHost != "" {
		httpServ := &HTTPServer{
			CorsAllowedOrigins: conf.HTTPCors,
			Vhosts:             conf.HTTPVirtualHosts,
			Whitelist:          conf.HTTPModules,
			Timeouts:           conf.HTTPTimeouts,
			Srv:                rpc.NewServer(),
			endpoint:           conf.HTTPEndpoint(),
			host:               conf.HTTPHost,
			port:               conf.HTTPPort,
			RPCAllowed:         true,
		}
		// check if ws is enabled and if ws port is the same as http port
		if conf.WSHost != "" && conf.WSPort == conf.HTTPPort {
			httpServ.WSAllowed = true
			httpServ.WsOrigins = conf.WSOrigins
			httpServ.Whitelist = append(httpServ.Whitelist, conf.WSModules...)
			node.httpServers = append(node.httpServers, httpServ)
			return node, nil
		}
		node.httpServers = append(node.httpServers, httpServ)
	}
	if conf.WSHost != "" {
		node.httpServers = append(node.httpServers, &HTTPServer{
			WsOrigins: conf.WSOrigins,
			Whitelist: conf.WSModules,
			Srv:       rpc.NewServer(),
			endpoint:  conf.WSEndpoint(),
			host:      conf.WSHost,
			port:      conf.WSPort,
			WSAllowed: true,
		})
	}

	return node, nil
}

// Close stops the Node and releases resources acquired in
// Node constructor New.
func (n *Node) Close() error {
	var errs []error

	// Terminate all subsystems and collect any errors
	if err := n.Stop(); err != nil && err != ErrNodeStopped {
		errs = append(errs, err)
	}
	if err := n.accman.Close(); err != nil {
		errs = append(errs, err)
	}
	// Report any errors that might have occurred
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("%v", errs)
	}
}

// RegisterLifecycle registers the given Lifecycle on the node
func (n *Node) RegisterLifecycle(lifecycle Lifecycle) {
	for _, existing := range n.lifecycles {
		if existing == lifecycle {
			utils.Fatalf("Lifecycle cannot be registered more than once", lifecycle)
		}
	}
	n.lifecycles = append(n.lifecycles, lifecycle)
}

// RegisterProtocols adds backend's protocols to the node's p2p server
func (n *Node) RegisterProtocols(protocols []p2p.Protocol) error {
	n.server.Protocols = append(n.server.Protocols, protocols...)
	return nil
}

// RegisterAPIs registers the APIs a service provides on the node
func (n *Node) RegisterAPIs(apis []rpc.API) {
	n.rpcAPIs = append(n.rpcAPIs, apis...)
}

// RegisterHTTPServer registers the given HTTP server on the node
func (n *Node) RegisterHTTPServer(server *HTTPServer) {
	n.httpServers = append(n.httpServers, server)
}

// ExistingHTTPServer checks if an HTTP server is already configured on the given endpoint
func (n *Node) ExistingHTTPServer(endpoint string) *HTTPServer {
	for _, httpServer := range n.httpServers {
		if endpoint == httpServer.endpoint {
			return httpServer
		}
	}
	return nil
}

// CreateHTTPServer creates an http.Server and adds it to the given HTTPServer // TODO improve?
func (n *Node) CreateHTTPServer(h *HTTPServer, exposeAll bool) error {
	// register apis and create handler stack
	err := RegisterApisFromWhitelist(n.rpcAPIs, h.Whitelist, h.Srv, exposeAll)
	if err != nil {
		return err
	}

	// start the HTTP listener
	listener, err := net.Listen("tcp", h.endpoint)
	if err != nil {
		return err
	}
	// create the HTTP server
	httpSrv := &http.Server{Handler: h.handler}
	// check timeouts if they exist
	if h.Timeouts != (rpc.HTTPTimeouts{}) {
		CheckTimeouts(&h.Timeouts)
		httpSrv.ReadTimeout = h.Timeouts.ReadTimeout
		httpSrv.WriteTimeout = h.Timeouts.WriteTimeout
		httpSrv.IdleTimeout = h.Timeouts.IdleTimeout
	}

	// complete the HTTPServer
	h.Listener = listener
	h.ListenerAddr = listener.Addr()
	h.Server = httpSrv

	return nil
}

// running returns true if the node's p2p server is already running
func (n *Node) running() bool {
	return n.server.Listening()
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's already running
	if n.running() {
		return ErrNodeRunning
	}
	if err := n.openDataDir(); err != nil {
		return err
	}

	// Start the p2p node
	if err := n.server.Start(); err != nil {
		return convertFileLockError(err)
	}
	n.log.Info("Starting peer-to-peer node", "instance", n.server.Name)

	// TODO running p2p server needs to somehow be added to the backend

	// Start all registered lifecycles
	var started []Lifecycle
	for _, lifecycle := range n.lifecycles {
		if err := lifecycle.Start(); err != nil {
			n.stopLifecycles(started)
		}
		started = append(started, lifecycle)
	}

	// Lastly, start the configured RPC interfaces
	if err := n.configureRPC(); err != nil {
		n.stopLifecycles(n.lifecycles)
		n.server.Stop()
		return err
	}
	// Finish initializing the service context
	n.ServiceContext.AccountManager = n.accman
	n.ServiceContext.EventMux = n.eventmux

	// Finish initializing the startup
	n.stop = make(chan struct{})
	return nil
}

// stopLifecycles stops the node's running Lifecycles
func (n *Node) stopLifecycles(started []Lifecycle) {
	for _, lifecycle := range started {
		lifecycle.Stop()
	}
}

// Config returns the configuration of node.
func (n *Node) Config() *Config {
	return n.config
}

func (n *Node) openDataDir() error {
	if n.config.DataDir == "" {
		return nil // ephemeral
	}

	instdir := filepath.Join(n.config.DataDir, n.config.name())
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	// Lock the instance directory to prevent concurrent use by another instance as well as
	// accidental use of the instance directory as a database.
	release, _, err := fileutil.Flock(filepath.Join(instdir, "LOCK"))
	if err != nil {
		return convertFileLockError(err)
	}
	n.instanceDirLock = release
	return nil
}

// configureRPC is a helper method to configure all the various RPC endpoints during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) configureRPC() error {
	n.RegisterAPIs(n.apis())

	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(); err != nil {
		return err
	}
	if err := n.startIPC(); err != nil {
		n.stopInProc()
		return err
	}

	for _, server := range n.httpServers {
		// configure the handlers
		if server.RPCAllowed {
			server.handler = NewHTTPHandlerStack(server.Srv, server.CorsAllowedOrigins, server.Vhosts)
			// wrap ws handler just in case ws is enabled through the console after start-up
			wsHandler := server.Srv.WebsocketHandler(server.WsOrigins)
			server.handler = server.NewWebsocketUpgradeHandler(server.handler, wsHandler)

			n.log.Info("HTTP configured on endpoint ", "endpoint", server.endpoint)
		}
		if server.WSAllowed && server.handler == nil {
			server.handler = server.Srv.WebsocketHandler(server.WsOrigins)
			n.log.Info("Websocket configured on endpoint ", "endpoint", server.endpoint)
		}
		if server.GQLAllowed {
			if server.handler == nil {
				server.handler = server.GQLHandler
			} else {
				server.handler = NewGQLUpgradeHandler(server.handler, server.GQLHandler)
			}
			n.log.Info("GraphQL configured on endpoint ", "endpoint", server.endpoint)
		}
		// create the HTTP server
		if err := n.CreateHTTPServer(server, false); err != nil {
			return err
		}
		// start HTTP server
		server.Start()
		n.log.Info("HTTP endpoint successfully opened", "url", fmt.Sprintf("http://%v/", server.ListenerAddr))
	}
	// All API endpoints started successfully
	return nil
}

// startInProc initializes an in-process RPC endpoint.
func (n *Node) startInProc() error {
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range n.rpcAPIs {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		n.log.Debug("InProc registered", "namespace", api.Namespace)
	}
	n.inprocHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *Node) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC() error {
	if n.ipc.Endpoint() == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(n.ipc.Endpoint(), n.rpcAPIs)
	if err != nil {
		return err
	}
	n.ipc.Listener = listener
	n.ipc.handler = handler
	n.log.Info("IPC endpoint opened", "url", n.ipc.Endpoint())
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipc.Listener != nil {
		n.ipc.Listener.Close()
		n.ipc.Listener = nil

		n.log.Info("IPC endpoint closed", "url", n.ipc.Endpoint())
	}
	if n.ipc.Srv != nil {
		n.ipc.Srv.Stop()
		n.ipc.Srv = nil
	}
}

//// startHTTP initializes and starts the HTTP RPC endpoint.
//func (n *Node) startHTTP(endpoint string, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts, wsOrigins []string) error {
//	// Short circuit if the HTTP endpoint isn't being exposed
//	if endpoint == "" {
//		return nil
//	}
//	// register apis and create handler stack
//	srv := rpc.NewServer()
//	err := RegisterApisFromWhitelist(n.rpcAPIs, modules, srv, false)
//	if err != nil {
//		return err
//	}
//	handler := NewHTTPHandlerStack(srv, cors, vhosts)
//	// wrap handler in websocket handler only if websocket port is the same as http rpc
//	if n.http.Endpoint() == n.ws.Endpoint() {
//		handler = n.http.NewWebsocketUpgradeHandler(handler, srv.WebsocketHandler(wsOrigins))
//	}
//	httpServer, addr, err := StartHTTPEndpoint(endpoint, timeouts, handler)
//	if err != nil {
//		return err
//	}
//	n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", addr),
//		"cors", strings.Join(cors, ","),
//		"vhosts", strings.Join(vhosts, ","))
//	if n.http.Endpoint() == n.ws.Endpoint() {
//		n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%v", addr))
//	}
//	// All listeners booted successfully
//	n.http.endpoint = endpoint
//	n.http.Server = httpServer
//	n.http.ListenerAddr = addr
//	n.http.Srv = srv
//
//	return nil
//}

// stopServers terminates the given HTTP servers' endpoints
func (n *Node) stopServer(server *HTTPServer) {
	if server.Server != nil {
		url := fmt.Sprintf("http://%v/", server.ListenerAddr)
		// Don't bother imposing a timeout here.
		server.Server.Shutdown(context.Background())
		n.log.Info("HTTP Endpoint closed", "url", url)
	}
	if server.Srv != nil {
		server.Srv.Stop()
		server.Srv = nil
	}
}

//// stopHTTP terminates the HTTP RPC endpoint.
//func (n *Node) stopHTTP() {
//	for _, server := range n.httpServers {
//		if server.RPCAllowed {
//			if server.Server != nil {
//				url := fmt.Sprintf("http://%v/", server.ListenerAddr)
//				// Don't bother imposing a timeout here.
//				server.Server.Shutdown(context.Background())
//				n.log.Info("HTTP Endpoint closed", "url", url)
//			}
//			if server.Srv != nil {
//				server.Srv.Stop()
//				server.Srv = nil
//			}
//		}
//	}
//}

//// startWS initializes and starts the websocket RPC endpoint.
//func (n *Node) startWS(endpoint string, modules []string, wsOrigins []string, exposeAll bool) error {
//	// Short circuit if the WS endpoint isn't being exposed
//	if endpoint == "" {
//		return nil
//	}
//
//	srv := rpc.NewServer()
//	handler := srv.WebsocketHandler(wsOrigins)
//	err := RegisterApisFromWhitelist(n.rpcAPIs, modules, srv, exposeAll)
//	if err != nil {
//		return err
//	}
//	httpServer, addr, err := startWSEndpoint(endpoint, handler)
//	if err != nil {
//		return err
//	}
//	n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", addr))
//	// All listeners booted successfully
//	n.ws.endpoint = endpoint
//	n.ws.ListenerAddr = addr
//	n.ws.Server = httpServer
//	n.ws.Srv = srv
//
//	return nil
//}

//// stopWS terminates the websocket RPC endpoint.
//func (n *Node) stopWS() {
//	if n.ws.Server != nil {
//		url := fmt.Sprintf("http://%v/", n.ws.ListenerAddr)
//		// Don't bother imposing a timeout here.
//		n.ws.Server.Shutdown(context.Background())
//		n.log.Info("HTTP Endpoint closed", "url", url)
//	}
//	if n.ws.Srv != nil {
//		n.ws.Srv.Stop()
//		n.ws.Srv = nil
//	}
//}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}

	// Terminate the API, services and the p2p server.
	for _, httpServer := range n.httpServers {
		n.stopServer(httpServer)
	}
	n.stopIPC()
	n.rpcAPIs = nil
	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for _, lifecycle := range n.lifecycles {
		if err := lifecycle.Stop(); err != nil {
			failure.Services[reflect.TypeOf(lifecycle)] = err
		}
	}
	n.server.Stop()
	n.server = nil

	// Release instance directory lock.
	if n.instanceDirLock != nil {
		if err := n.instanceDirLock.Release(); err != nil {
			n.log.Error("Can't release datadir lock", "err", err)
		}
		n.instanceDirLock = nil
	}

	// unblock n.Wait
	close(n.stop)

	// Remove the keystore if it was created ephemerally.
	var keystoreErr error
	if n.ephemeralKeystore != "" {
		keystoreErr = os.RemoveAll(n.ephemeralKeystore)
	}

	if len(failure.Services) > 0 {
		return failure
	}
	if keystoreErr != nil {
		return keystoreErr
	}
	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		n.lock.RUnlock()
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

// Attach creates an RPC client attached to an in-process API handler.
func (n *Node) Attach() (*rpc.Client, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.server == nil {
		return nil, ErrNodeStopped
	}
	return rpc.DialInProc(n.inprocHandler), nil
}

// RPCHandler returns the in-process RPC request handler.
func (n *Node) RPCHandler() (*rpc.Server, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.inprocHandler == nil {
		return nil, ErrNodeStopped
	}
	return n.inprocHandler, nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.server
}

// DataDir retrieves the current datadir used by the protocol stack.
// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) DataDir() string {
	return n.config.DataDir
}

// InstanceDir retrieves the instance directory used by the protocol stack.
func (n *Node) InstanceDir() string {
	return n.config.instanceDir()
}

// AccountManager retrieves the account manager used by the protocol stack.
func (n *Node) AccountManager() *accounts.Manager {
	return n.accman
}

// IPCEndpoint retrieves the current IPC endpoint used by the protocol stack.
func (n *Node) IPCEndpoint() string {
	return n.ipc.Endpoint()
}

// HTTPEndpoint retrieves the current HTTP endpoint used by the protocol stack.
func (n *Node) HTTPEndpoint() string {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, httpServer := range n.httpServers {
		if httpServer.RPCAllowed {
			if httpServer.Listener != nil {
				return httpServer.ListenerAddr.String()
			}
			return httpServer.endpoint
		}
	}

	return "" // TODO should return an empty string if http server not configured?
}

// WSEndpoint retrieves the current WS endpoint
// used by the protocol stack.
func (n *Node) WSEndpoint() string {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, httpServer := range n.httpServers {
		if httpServer.WSAllowed {
			if httpServer.Listener != nil {
				return httpServer.ListenerAddr.String()
			}
			return httpServer.endpoint
		}
	}

	return "" // TODO should return an empty string if ws server not configured?
}

// EventMux retrieves the event multiplexer used by all the network services in
// the current protocol stack.
func (n *Node) EventMux() *event.TypeMux {
	return n.eventmux
}

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory. If the node is
// ephemeral, a memory database is returned.
func (n *Node) OpenDatabase(name string, cache, handles int, namespace string) (ethdb.Database, error) {
	if n.config.DataDir == "" {
		return rawdb.NewMemoryDatabase(), nil
	}
	return rawdb.NewLevelDBDatabase(n.config.ResolvePath(name), cache, handles, namespace)
}

// OpenDatabaseWithFreezer opens an existing database with the given name (or
// creates one if no previous can be found) from within the node's data directory,
// also attaching a chain freezer to it that moves ancient chain data from the
// database to immutable append-only files. If the node is an ephemeral one, a
// memory database is returned.
func (n *Node) OpenDatabaseWithFreezer(name string, cache, handles int, freezer, namespace string) (ethdb.Database, error) {
	if n.config.DataDir == "" {
		return rawdb.NewMemoryDatabase(), nil
	}
	root := n.config.ResolvePath(name)

	switch {
	case freezer == "":
		freezer = filepath.Join(root, "ancient")
	case !filepath.IsAbs(freezer):
		freezer = n.config.ResolvePath(freezer)
	}
	return rawdb.NewLevelDBDatabaseWithFreezer(root, cache, handles, freezer, namespace)
}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (n *Node) ResolvePath(x string) string {
	return n.config.ResolvePath(x)
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
			Namespace: "web3",
			Version:   "1.0",
			Service:   NewPublicWeb3API(n),
			Public:    true,
		},
	}
}

// RegisterApisFromWhitelist checks the given modules' availability, generates a whitelist based on the allowed modules,
// and then registers all of the APIs exposed by the services.
func RegisterApisFromWhitelist(apis []rpc.API, modules []string, srv *rpc.Server, exposeAll bool) error {
	if bad, available := checkModuleAvailability(modules, apis); len(bad) > 0 {
		log.Error("Unavailable modules in HTTP API list", "unavailable", bad, "available", available)
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
		}
	}
	return nil
}
