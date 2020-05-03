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

	// TODO: removed p2pConfig b/c p2pServer already contains p2pConfig (is there a reason for it to be duplicated?
	server *p2p.Server // Currently running P2P networking layer

	ServiceContext *ServiceContext

	lifecycles []Lifecycle // All registered backends, services, and auxiliary services that have a lifecycle

	backend     Backend                           // The registered Backend of the node
	services    map[reflect.Type]Service          // Currently running services
	auxServices map[reflect.Type]AuxiliaryService // Currently running auxiliary services

	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	ipc  *HTTPServer // TODO
	http *HTTPServer // TODO
	ws   *HTTPServer // TODO


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
		services:    make(map[reflect.Type]Service),
		auxServices: make(map[reflect.Type]AuxiliaryService),
		ipc: &HTTPServer{
			endpoint: conf.IPCEndpoint(),
		},
		http: &HTTPServer{
			CorsAllowedOrigins: conf.HTTPCors,
			Vhosts: conf.HTTPVirtualHosts,
			Whitelist: conf.HTTPModules,
			Timeouts: conf.HTTPTimeouts,
			Srv: rpc.NewServer(),
			endpoint: conf.HTTPEndpoint(),
			host: conf.HTTPHost,
			port: conf.HTTPPort,
		},
		ws: &HTTPServer{
			CorsAllowedOrigins: conf.WSOrigins,
			Whitelist: conf.WSModules,
			Srv: rpc.NewServer(),
			endpoint: conf.WSEndpoint(),
			host: conf.WSHost,
			port: conf.WSPort,
		},
		eventmux: new(event.TypeMux),
		log:      conf.Logger,
	}
	node.ServiceContext.EventMux = node.eventmux
	node.ServiceContext.AccountManager = node.accman
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

// TODO document
func (n *Node) RegisterLifecycle(lifecycle Lifecycle) {
	n.lifecycles = append(n.lifecycles, lifecycle)
}

// TODO document
func (n *Node) RegisterBackend(backend Backend) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	// check if p2p node is already running
	if n.running() {
		return ErrNodeRunning
	}
	// check that there is not already a backend registered on the node
	if n.backend != nil {
		return errors.New("a backend has already been registered on the node") // TODO is this error okay?
	}
	n.backend = backend
	return nil
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) RegisterService(service Service) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.running() {
		return ErrNodeRunning
	}

	kind := reflect.TypeOf(service)
	if _, exists := n.services[kind]; exists {
		return &DuplicateServiceError{Kind: kind}
	}
	n.services[kind] = service

	return nil
}

// TODO document
func (n *Node) RegisterAuxService(auxService AuxiliaryService) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.running() {
		return ErrNodeRunning
	}
	// make sure auxiliary service is not duplicated
	kind := reflect.TypeOf(auxService)
	if _, exists := n.auxServices[kind]; exists {
		return &DuplicateServiceError{Kind: kind}
	}
	n.auxServices[kind] = auxService

	return nil
}

func (n *Node) RegisterProtocols(protocols []p2p.Protocol) error {
	if !n.running() {
		return ErrNodeStopped
	}
	// add backend's protocols to the o2o server
	n.server.Protocols = append(n.server.Protocols, protocols...)
	return nil
}

func (n *Node) RegisterRPC(apis []rpc.API) {
	// Gather all the possible APIs to surface
	apis = append(apis, n.backend.APIs()...)
	for _, service := range n.services {
		apis = append(apis, service.APIs()...)
	}
	n.rpcAPIs = apis
}

// TODO what is the purpose of this function? Define it and document it.
func (n *Node) RegisterHTTP(h *HTTPServer, exposeAll bool) error {
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
	return n.server != nil
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

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	n.server = &p2p.Server{Config: n.config.P2P}
	n.server.Config.PrivateKey = n.config.NodeKey()
	n.server.Config.Name = n.config.NodeName()
	n.server.Config.Logger = n.log
	if n.server.Config.StaticNodes == nil {
		n.server.Config.StaticNodes = n.config.StaticNodes()
	}
	if n.server.Config.TrustedNodes == nil {
		n.server.Config.TrustedNodes = n.config.TrustedNodes()
	}
	if n.server.Config.NodeDatabase == "" {
		n.server.Config.NodeDatabase = n.config.NodeDB()
	}

	// Start the p2p node
	if err := n.server.Start(); err != nil {
		return convertFileLockError(err)
	}
	n.log.Info("Starting peer-to-peer node", "instance", n.server.Name)

	// Register the running p2p server with the Backend
	if err := n.backend.P2PServer(n.server); err != nil {
		n.server.Stop()
		return err
	}
	// Register the Backend's protocols with the p2p server
	if err := n.RegisterProtocols(n.backend.Protocols()); err != nil {
		n.server.Stop()
		return err
	}

	// Start all registered lifecycles
	var started []Lifecycle
	for _, lifecycle := range n.lifecycles {
		if err := lifecycle.Start(); err != nil {
			n.stopLifecycles(started)
		}
		started = append(started, lifecycle)
	}

	// Lastly, start the configured RPC interfaces
	if err := n.startRPC(); err != nil {
		n.stopLifecycles(n.lifecycles)
		n.server.Stop()
		return err
	}
	// Finish initializing the service context
	n.ServiceContext.backend = n.backend
	n.ServiceContext.AccountManager = n.accman
	n.ServiceContext.EventMux = n.eventmux
	n.ServiceContext.services = n.services
	n.ServiceContext.auxServices = n.auxServices

	// Finish initializing the startup
	n.stop = make(chan struct{})
	return nil
}

// TODO document
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

// startRPC is a helper method to start all the various RPC endpoints during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC() error {
	n.RegisterRPC(n.apis())
	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(); err != nil {
		return err
	}
	if err := n.startIPC(); err != nil {
		n.stopInProc()
		return err
	}
	// create and start http server if the endpoint exists
	if n.http.endpoint != "" {
		// wrap handler in websocket handler only if websocket port is the same as http rpc
		n.http.handler = NewHTTPHandlerStack(n.http.Srv, n.http.CorsAllowedOrigins, n.http.Vhosts)
		if n.http.endpoint == n.ws.endpoint {
			n.http.handler = NewWebsocketUpgradeHandler(n.http.handler, n.http.Srv.WebsocketHandler(n.ws.CorsAllowedOrigins))
		}
		if err := n.RegisterHTTP(n.http, false); err != nil {
			n.stopIPC()
			n.stopInProc()
			return err
		}
		n.http.Start()
		n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", n.http.ListenerAddr),
			"cors", strings.Join(n.http.CorsAllowedOrigins, ","),
			"vhosts", strings.Join(n.http.Vhosts, ","))
		if n.http.Endpoint() == n.ws.Endpoint() {
			n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%v", n.http.ListenerAddr))
		}
	}
	//  create and start ws server if the endpoint exists
	if n.ws.endpoint != "" && n.http.endpoint != n.ws.endpoint {
		n.ws.handler = n.ws.Srv.WebsocketHandler(n.ws.CorsAllowedOrigins)
		if err := n.RegisterHTTP(n.ws, n.config.WSExposeAll); err != nil {
			n.stopIPC()
			n.stopInProc()
			return err
		}
		n.ws.Start()
		n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", n.ws.ListenerAddr))
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

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts, wsOrigins []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// register apis and create handler stack
	srv := rpc.NewServer()
	err := RegisterApisFromWhitelist(n.rpcAPIs, modules, srv, false)
	if err != nil {
		return err
	}
	handler := NewHTTPHandlerStack(srv, cors, vhosts)
	// wrap handler in websocket handler only if websocket port is the same as http rpc
	if n.http.Endpoint() == n.ws.Endpoint() {
		handler = NewWebsocketUpgradeHandler(handler, srv.WebsocketHandler(wsOrigins))
	}
	httpServer, addr, err := StartHTTPEndpoint(endpoint, timeouts, handler)
	if err != nil {
		return err
	}
	n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", addr),
		"cors", strings.Join(cors, ","),
		"vhosts", strings.Join(vhosts, ","))
	if n.http.Endpoint() == n.ws.Endpoint() {
		n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%v", addr))
	}
	// All listeners booted successfully
	n.http.endpoint = endpoint
	n.http.Server = httpServer
	n.http.ListenerAddr = addr
	n.http.Srv = srv

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.http.Server != nil {
		url := fmt.Sprintf("http://%v/", n.http.ListenerAddr)
		// Don't bother imposing a timeout here.
		n.http.Server.Shutdown(context.Background())
		n.log.Info("HTTP Endpoint closed", "url", url)
	}
	if n.http.Srv != nil {
		n.http.Srv.Stop()
		n.http.Srv = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, modules []string, wsOrigins []string, exposeAll bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}

	srv := rpc.NewServer()
	handler := srv.WebsocketHandler(wsOrigins)
	err := RegisterApisFromWhitelist(n.rpcAPIs, modules, srv, exposeAll)
	if err != nil {
		return err
	}
	httpServer, addr, err := startWSEndpoint(endpoint, handler)
	if err != nil {
		return err
	}
	n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", addr))
	// All listeners booted successfully
	n.ws.endpoint = endpoint
	n.ws.ListenerAddr = addr
	n.ws.Server = httpServer
	n.ws.Srv = srv

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.ws.Server != nil {
		url := fmt.Sprintf("http://%v/", n.ws.ListenerAddr)
		// Don't bother imposing a timeout here.
		n.ws.Server.Shutdown(context.Background())
		n.log.Info("HTTP Endpoint closed", "url", url)
	}
	if n.ws.Srv != nil {
		n.ws.Srv.Stop()
		n.ws.Srv = nil
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

	// Terminate the API, services and the p2p server.
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

func (n *Node) Backend() Backend {
	return n.backend
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

// Service retrieves a currently running service registered of a specific type.
func (n *Node) AuxService(auxService interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(auxService).Elem()
	if running, ok := n.auxServices[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
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

	if n.http.Server != nil {
		return n.http.ListenerAddr.String()
	}
	return n.http.Endpoint()
}

// WSEndpoint retrieves the current WS endpoint
// used by the protocol stack.
func (n *Node) WSEndpoint() string {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.ws.Server != nil {
		return n.ws.ListenerAddr.String()
	}
	return n.ws.Endpoint()
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
