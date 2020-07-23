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
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/tsdb/fileutil"
)

// Node is a container on which services can be registered.
type Node struct {
	eventmux      *event.TypeMux
	config        *Config
	accman        *accounts.Manager
	log           log.Logger
	ephemKeystore string            // if non-empty, the key directory that will be removed by Stop
	dirLock       fileutil.Releaser // prevents concurrent use of instance directory
	stop          chan struct{}     // Channel to wait for termination notifications
	server        *p2p.Server       // Currently running P2P networking layer
	startStopLock sync.Mutex        // Start/Stop are protected by an additional lock
	state         int               // Tracks state of node lifecycle

	lock          sync.Mutex
	lifecycles    []Lifecycle // All registered backends, services, and auxiliary services that have a lifecycle
	httpServers   serverMap   // serverMap stores information about the node's rpc, ws, and graphQL http servers.
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests
	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	ipc           *httpServer // Stores information about the ipc http server

	databases map[*closeTrackingDB]struct{} // All open databases
}

const (
	initializingState = iota
	runningState
	closedState
)

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
	if conf.Logger == nil {
		conf.Logger = log.New()
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

	node := &Node{
		config:        conf,
		httpServers:   make(serverMap),
		ipc:           &httpServer{endpoint: conf.IPCEndpoint()},
		inprocHandler: rpc.NewServer(),
		eventmux:      new(event.TypeMux),
		log:           conf.Logger,
		stop:          make(chan struct{}),
		server:        &p2p.Server{Config: conf.P2P},
		databases:     make(map[*closeTrackingDB]struct{}),
	}

	// Register built-in APIs.
	node.rpcAPIs = append(node.rpcAPIs, node.apis()...)

	// Acquire the instance directory lock.
	if err := node.openDataDir(); err != nil {
		return nil, err
	}
	// Ensure that the AccountManager method works before the node has started. We rely on
	// this in cmd/geth.
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	node.accman = am
	node.ephemKeystore = ephemeralKeystore

	// Initialize the p2p server. This creates the node key and discovery databases.
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

	// Configure HTTP servers.
	if conf.HTTPHost != "" {
		httpServ := &httpServer{
			CorsAllowedOrigins: conf.HTTPCors,
			Vhosts:             conf.HTTPVirtualHosts,
			Whitelist:          conf.HTTPModules,
			Timeouts:           conf.HTTPTimeouts,
			Srv:                rpc.NewServer(),
			endpoint:           conf.HTTPEndpoint(),
			host:               conf.HTTPHost,
			port:               conf.HTTPPort,
			RPCAllowed:         1,
		}
		// Enable WebSocket on HTTP port if enabled.
		if conf.WSHost != "" && conf.WSPort == conf.HTTPPort {
			httpServ.WSAllowed = 1
			httpServ.WsOrigins = conf.WSOrigins
			httpServ.Whitelist = append(httpServ.Whitelist, conf.WSModules...)

			node.httpServers[conf.HTTPEndpoint()] = httpServ
			return node, nil
		}
		node.httpServers[conf.HTTPEndpoint()] = httpServ
	}
	if conf.WSHost != "" {
		node.httpServers[conf.WSEndpoint()] = &httpServer{
			WsOrigins: conf.WSOrigins,
			Whitelist: conf.WSModules,
			Srv:       rpc.NewServer(),
			endpoint:  conf.WSEndpoint(),
			host:      conf.WSHost,
			port:      conf.WSPort,
			WSAllowed: 1,
		}
	}

	return node, nil
}

// Start starts all registered lifecycles, RPC services and p2p networking.
// Node can only be started once.
func (n *Node) Start() error {
	n.startStopLock.Lock()
	defer n.startStopLock.Unlock()

	n.lock.Lock()
	switch n.state {
	case runningState:
		n.lock.Unlock()
		return ErrNodeRunning
	case closedState:
		n.lock.Unlock()
		return ErrNodeStopped
	}
	n.state = runningState
	err := n.startNetworking()
	lifecycles := make([]Lifecycle, len(n.lifecycles))
	copy(lifecycles, n.lifecycles)
	n.lock.Unlock()

	// Check if networking startup failed.
	if err != nil {
		n.doClose(nil)
		return err
	}
	// Start all registered lifecycles.
	var started []Lifecycle
	for _, lifecycle := range lifecycles {
		if err = lifecycle.Start(); err != nil {
			break
		}
		started = append(started, lifecycle)
	}
	// Check if any lifecycle failed to start.
	if err != nil {
		n.stopServices(started)
		n.doClose(nil)
	}
	return err
}

// Close stops the Node and releases resources acquired in
// Node constructor New.
func (n *Node) Close() error {
	n.startStopLock.Lock()
	defer n.startStopLock.Unlock()

	n.lock.Lock()
	state := n.state
	n.lock.Unlock()
	switch state {
	case initializingState:
		// The node was never started.
		return n.doClose(nil)
	case runningState:
		// The node was started, release resources acquired by Start().
		var errs []error
		if err := n.stopServices(n.lifecycles); err != nil {
			errs = append(errs, err)
		}
		return n.doClose(errs)
	case closedState:
		return ErrNodeStopped
	default:
		panic(fmt.Sprintf("node is in unknown state %d", state))
	}
}

// doClose releases resources acquired by New(), collecting errors.
func (n *Node) doClose(errs []error) error {
	// Close databases. This needs the lock because it needs to
	// synchronize with OpenDatabase*.
	n.lock.Lock()
	n.state = closedState
	errs = append(errs, n.closeDatabases()...)
	n.lock.Unlock()

	if err := n.accman.Close(); err != nil {
		errs = append(errs, err)
	}
	if n.ephemKeystore != "" {
		if err := os.RemoveAll(n.ephemKeystore); err != nil {
			errs = append(errs, err)
		}
	}

	// Release instance directory lock.
	n.closeDataDir()

	// Unblock n.Wait.
	close(n.stop)

	// Report any errors that might have occurred.
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("%v", errs)
	}
}

// registerHTTPServer registers the given HTTP server on the node.
func (n *Node) registerHTTPServer(endpoint string, server *httpServer) {
	n.httpServers[endpoint] = server
}

// existingHTTPServer checks if an HTTP server is already configured on the given endpoint.
func (n *Node) existingHTTPServer(endpoint string) *httpServer {
	if server, exists := n.httpServers[endpoint]; exists {
		return server
	}
	return nil
}

// createHTTPServer creates an http.Server and adds it to the given httpServer.
func (n *Node) createHTTPServer(h *httpServer, exposeAll bool) error {
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
	httpSrv := &http.Server{Handler: &h.srvMux}
	// check timeouts if they exist
	if h.Timeouts != (rpc.HTTPTimeouts{}) {
		CheckTimeouts(&h.Timeouts)
		httpSrv.ReadTimeout = h.Timeouts.ReadTimeout
		httpSrv.WriteTimeout = h.Timeouts.WriteTimeout
		httpSrv.IdleTimeout = h.Timeouts.IdleTimeout
	}
	// add listener and http.Server to httpServer
	h.Listener = listener
	h.Server = httpSrv

	return nil
}

// startNetworking starts all network endpoints.
func (n *Node) startNetworking() error {
	n.log.Info("Starting peer-to-peer node", "instance", n.server.Name)
	if err := n.server.Start(); err != nil {
		return convertFileLockError(err)
	}
	err := n.configureRPC()
	if err != nil {
		n.httpServers.Stop()
		n.server.Stop()
	}
	return err
}

// containsLifecycle checks if 'lfs' contains 'l'.
func containsLifecycle(lfs []Lifecycle, l Lifecycle) bool {
	for _, obj := range lfs {
		if obj == l {
			return true
		}
	}
	return false
}

// stopServices terminates running services, RPC and p2p networking.
// It is the inverse of Start.
func (n *Node) stopServices(running []Lifecycle) error {
	n.stopIPC()
	n.rpcAPIs = nil

	// Stop running lifecycles in reverse order.
	failure := &StopError{Services: make(map[reflect.Type]error)}
	for i := len(running) - 1; i >= 0; i-- {
		if err := running[i].Stop(); err != nil {
			failure.Services[reflect.TypeOf(running[i])] = err
		}
	}

	// Stop p2p networking.
	n.server.Stop()

	if len(failure.Services) > 0 {
		return failure
	}
	return nil
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
	n.dirLock = release
	return nil
}

func (n *Node) closeDataDir() {
	// Release instance directory lock.
	if n.dirLock != nil {
		if err := n.dirLock.Release(); err != nil {
			n.log.Error("Can't release datadir lock", "err", err)
		}
		n.dirLock = nil
	}
}

// configureRPC is a helper method to configure all the various RPC endpoints during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) configureRPC() error {
	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(); err != nil {
		return err
	}
	if err := n.startIPC(); err != nil {
		n.stopInProc()
		return err
	}

	// Configure HTTP servers.
	for _, server := range n.httpServers {
		handler := n.createHandler(server)
		if handler != nil {
			server.srvMux.Handle("/", handler)
		}
		if err := n.createHTTPServer(server, false); err != nil {
			return err
		}
	}

	// Register HTTP server as a lifecycle if it has not already been registered.
	if !containsLifecycle(n.lifecycles, &n.httpServers) {
		n.lifecycles = append(n.lifecycles, &n.httpServers)
	}
	return nil
}

// createHandler creates the http.Handler for the given httpServer.
func (n *Node) createHandler(server *httpServer) http.Handler {
	var handler http.Handler
	if atomic.LoadInt32(&server.RPCAllowed) == 1 {
		handler = NewHTTPHandlerStack(server.Srv, server.CorsAllowedOrigins, server.Vhosts)
		// wrap ws handler just in case ws is enabled through the console after start-up
		wsHandler := server.Srv.WebsocketHandler(server.WsOrigins)
		handler = server.NewWebsocketUpgradeHandler(handler, wsHandler)

		n.log.Info("HTTP configured on endpoint ", "endpoint", fmt.Sprintf("http://%s/", server.endpoint))
		if atomic.LoadInt32(&server.WSAllowed) == 1 {
			n.log.Info("Websocket configured on endpoint ", "endpoint", fmt.Sprintf("ws://%s/", server.endpoint))
		}
	}
	if (atomic.LoadInt32(&server.WSAllowed) == 1) && handler == nil {
		handler = server.Srv.WebsocketHandler(server.WsOrigins)
		n.log.Info("Websocket configured on endpoint ", "endpoint", fmt.Sprintf("ws://%s/", server.endpoint))
	}

	return handler
}

// startInProc registers all RPC APIs on the inproc server.
func (n *Node) startInProc() error {
	for _, api := range n.rpcAPIs {
		if err := n.inprocHandler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
	}
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *Node) stopInProc() {
	n.inprocHandler.Stop()
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC() error {
	if n.ipc.endpoint == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(n.ipc.endpoint, n.rpcAPIs)
	if err != nil {
		return err
	}
	n.ipc.Listener = listener
	n.ipc.handler = handler
	n.log.Info("IPC endpoint opened", "url", n.ipc.endpoint)
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipc.Listener != nil {
		n.ipc.Listener.Close()
		n.ipc.Listener = nil
		n.log.Info("IPC endpoint closed", "url", n.ipc.endpoint)
	}
	if n.ipc.Srv != nil {
		n.ipc.Srv.Stop()
		n.ipc.Srv = nil
	}
}

// stopServers terminates the given HTTP servers' endpoints
func (n *Node) stopServer(server *httpServer) {
	if server.Server != nil {
		url := fmt.Sprintf("http://%v/", server.Listener.Addr())
		// Don't bother imposing a timeout here.
		server.Server.Shutdown(context.Background())
		n.log.Info("HTTP Endpoint closed", "url", url)
	}
	if server.Srv != nil {
		server.Srv.Stop()
		server.Srv = nil
	}
	// remove stopped http server from node's http servers
	delete(n.httpServers, server.endpoint)
}

// Wait blocks until the node is closed.
func (n *Node) Wait() {
	<-n.stop
}

// RegisterLifecycle registers the given Lifecycle on the node.
func (n *Node) RegisterLifecycle(lifecycle Lifecycle) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.state != initializingState {
		panic("can't register lifecycle on running/stopped node")
	}
	if containsLifecycle(n.lifecycles, lifecycle) {
		panic(fmt.Sprintf("attempt to register lifecycle %T more than once", lifecycle))
	}
	n.lifecycles = append(n.lifecycles, lifecycle)
}

// RegisterProtocols adds backend's protocols to the node's p2p server.
func (n *Node) RegisterProtocols(protocols []p2p.Protocol) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.state != initializingState {
		panic("can't register protocols on running/stopped node")
	}
	n.server.Protocols = append(n.server.Protocols, protocols...)
}

// RegisterAPIs registers the APIs a service provides on the node.
func (n *Node) RegisterAPIs(apis []rpc.API) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.state != initializingState {
		panic("can't register APIs on running/stopped node")
	}
	n.rpcAPIs = append(n.rpcAPIs, apis...)
}

// RegisterPath mounts the given handler on the given path on the canonical HTTP server.
func (n *Node) RegisterPath(path string, handler http.Handler) string {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.state != initializingState {
		panic("can't register HTTP handler on running/stopped node")
	}
	for _, server := range n.httpServers {
		if atomic.LoadInt32(&server.RPCAllowed) == 1 {
			server.srvMux.Handle(path, handler)
			return server.endpoint
		}
	}
	n.log.Warn(fmt.Sprintf("HTTP server not configured on node, path %s cannot be enabled", path))
	return ""
}

// Attach creates an RPC client attached to an in-process API handler.
func (n *Node) Attach() (*rpc.Client, error) {
	return rpc.DialInProc(n.inprocHandler), nil
}

// RPCHandler returns the in-process RPC request handler.
func (n *Node) RPCHandler() (*rpc.Server, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.state == closedState {
		return nil, ErrNodeStopped
	}
	return n.inprocHandler, nil
}

// Config returns the configuration of node.
func (n *Node) Config() *Config {
	return n.config
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server. Callers should not
// start or stop the returned server.
func (n *Node) Server() *p2p.Server {
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
	return n.ipc.endpoint
}

// WSEndpoint retrieves the current WS endpoint used by the protocol stack.
func (n *Node) WSEndpoint() string {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, httpServer := range n.httpServers {
		if atomic.LoadInt32(&httpServer.WSAllowed) == 1 {
			if httpServer.Listener != nil {
				return httpServer.Listener.Addr().String()
			}
			return httpServer.endpoint
		}
	}

	return n.config.WSEndpoint()
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
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.state == closedState {
		return nil, ErrNodeStopped
	}

	var db ethdb.Database
	var err error
	if n.config.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		db, err = rawdb.NewLevelDBDatabase(n.ResolvePath(name), cache, handles, namespace)
	}

	if err == nil {
		db = n.wrapDatabase(db)
	}
	return db, err
}

// OpenDatabaseWithFreezer opens an existing database with the given name (or
// creates one if no previous can be found) from within the node's data directory,
// also attaching a chain freezer to it that moves ancient chain data from the
// database to immutable append-only files. If the node is an ephemeral one, a
// memory database is returned.
func (n *Node) OpenDatabaseWithFreezer(name string, cache, handles int, freezer, namespace string) (ethdb.Database, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.state == closedState {
		return nil, ErrNodeStopped
	}

	var db ethdb.Database
	var err error
	if n.config.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		root := n.ResolvePath(name)
		switch {
		case freezer == "":
			freezer = filepath.Join(root, "ancient")
		case !filepath.IsAbs(freezer):
			freezer = n.ResolvePath(freezer)
		}
		db, err = rawdb.NewLevelDBDatabaseWithFreezer(root, cache, handles, freezer, namespace)
	}

	if err == nil {
		db = n.wrapDatabase(db)
	}
	return db, err
}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (n *Node) ResolvePath(x string) string {
	return n.config.ResolvePath(x)
}

// closeTrackingDB wraps the Close method of a database. When the database is closed by the
// service, the wrapper removes it from the node's database map. This ensures that Node
// won't auto-close the database if it is closed by the service that opened it.
type closeTrackingDB struct {
	ethdb.Database
	n *Node
}

func (db *closeTrackingDB) Close() error {
	db.n.lock.Lock()
	delete(db.n.databases, db)
	db.n.lock.Unlock()
	return db.Database.Close()
}

// wrapDatabase ensures the database will be auto-closed when Node is closed.
func (n *Node) wrapDatabase(db ethdb.Database) ethdb.Database {
	wrapper := &closeTrackingDB{db, n}
	n.databases[wrapper] = struct{}{}
	return wrapper
}

// closeDatabases closes all open databases.
func (n *Node) closeDatabases() (errors []error) {
	for db := range n.databases {
		delete(n.databases, db)
		if err := db.Database.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
