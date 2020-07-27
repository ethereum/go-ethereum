// Copyright 2020 The go-ethereum Authors
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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/cors"
)

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

// httpConfig is the JSON-RPC/HTTP configuration.
type httpConfig struct {
	Modules            []string
	CorsAllowedOrigins []string
	Vhosts             []string
}

// wsConfig is the JSON-RPC/Websocket configuration
type wsConfig struct {
	Origins []string
	Modules []string
}

type httpServer struct {
	log      log.Logger
	timeouts rpc.HTTPTimeouts
	mux      http.ServeMux // registered handlers go here

	mu       sync.Mutex
	server   *http.Server
	listener net.Listener // non-nil when server is running

	httpConfig  httpConfig
	httpRPC     *rpc.Server
	httpHandler http.Handler

	wsConfig  wsConfig
	wsRPC     *rpc.Server
	wsHandler http.Handler

	endpoint string
	host     string
	port     int
	config   httpConfig

	handlerNames map[string]string

	// atomic flags for the handler
	rpcAllowedFlag int32
	wsAllowedFlag  int32
}

func newHTTPServer(log log.Logger, timeouts rpc.HTTPTimeouts) *httpServer {
	return &httpServer{log: log, timeouts: timeouts, handlerNames: make(map[string]string)}
}

// setListenAddr configures the listening address of the server.
// The address can only be set while the server isn't running.
func (h *httpServer) setListenAddr(host string, port int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil && (host != h.host || port != h.port) {
		return fmt.Errorf("HTTP server already running on %s", h.endpoint)
	}

	h.host, h.port = host, port
	h.endpoint = fmt.Sprintf("%s:%d", host, port)
	return nil
}

// listenAddr returns the listening address of the server.
func (h *httpServer) listenAddr() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil {
		return h.listener.Addr().String()
	}
	return h.endpoint
}

// enableRPC turns on JSON-RPC over HTTP on the server.
func (h *httpServer) enableRPC(apis []rpc.API, config httpConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.httpRPC != nil {
		return fmt.Errorf("JSON-RPC over HTTP is already enabled")
	}

	// Create RPC server.
	srv := rpc.NewServer()
	if err := RegisterApisFromWhitelist(apis, h.httpConfig.Modules, srv, false); err != nil {
		return err
	}
	// Create handler.
	h.httpRPC = srv
	h.httpHandler = NewHTTPHandlerStack(h.httpRPC, config.CorsAllowedOrigins, config.Vhosts)
	atomic.StoreInt32(&h.rpcAllowedFlag, 1)
	return nil
}

// enableWS turns on JSON-RPC over WebSocket on the server.
func (h *httpServer) enableWS(apis []rpc.API, config wsConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.wsRPC != nil {
		return fmt.Errorf("JSON-RPC over WebSocket is already enabled")
	}

	// Create RPC server.
	srv := rpc.NewServer()
	if err := RegisterApisFromWhitelist(apis, h.wsConfig.Modules, srv, false); err != nil {
		return err
	}
	// Create handler.
	h.wsRPC = rpc.NewServer()
	h.wsHandler = h.wsRPC.WebsocketHandler(config.Origins)
	atomic.StoreInt32(&h.wsAllowedFlag, 1)
	return nil
}

// disableWS disables JSON-RPC over WebSocket.
func (h *httpServer) disableWS() {
	h.mu.Lock()
	defer h.mu.Unlock()

	atomic.StoreInt32(&h.wsAllowedFlag, 0)
	h.wsRPC.Stop()
	h.wsRPC = nil
}

// start starts the HTTP server if it is enabled and not already running.
func (h *httpServer) start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.endpoint == "" || h.listener != nil {
		return nil // already running or not configured
	}

	// Initialize the server.
	h.server = &http.Server{Handler: h}
	if h.timeouts != (rpc.HTTPTimeouts{}) {
		CheckTimeouts(&h.timeouts)
		h.server.ReadTimeout = h.timeouts.ReadTimeout
		h.server.WriteTimeout = h.timeouts.WriteTimeout
		h.server.IdleTimeout = h.timeouts.IdleTimeout
	}

	// Start the server.
	listener, err := net.Listen("tcp", h.endpoint)
	if err != nil {
		return err
	}
	h.listener = listener
	go h.server.Serve(listener)

	// if server is websocket only, return after logging
	if h.wsAllowed() && !h.rpcAllowed() {
		h.log.Info("Websocket enabled", "url", fmt.Sprintf("ws://%v/", listener.Addr()))
		return nil
	}
	// log http endpoint
	h.log.Info("HTTP server started",
		"endpoint", listener.Addr(),
		"cors", strings.Join(h.config.CorsAllowedOrigins, ","),
		"vhosts", strings.Join(h.config.Vhosts, ","),
	)
	// log all handlers mounted on server
	for path, name := range h.handlerNames {
		log.Info(name + " enabled", "url", "http://" + listener.Addr().String() + path)
	}

	return nil
}

// stop shuts down the HTTP server.
func (h *httpServer) stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.listener == nil {
		return nil // not running
	}

	// Shut down the server.
	if h.httpRPC != nil {
		h.httpRPC.Stop()
	}
	if h.wsRPC != nil {
		h.wsRPC.Stop()
	}
	h.server.Shutdown(context.Background())
	h.listener.Close()
	h.log.Info("HTTP endpoint closed", "url", h.endpoint)

	// Clear out everything to allow re-configuring it later.
	h.host, h.port, h.endpoint = "", 0, ""
	h.server, h.listener = nil, nil
	h.httpRPC, h.httpHandler, h.wsRPC, h.wsHandler = nil, nil, nil, nil
	atomic.StoreInt32(&h.rpcAllowedFlag, 0)
	atomic.StoreInt32(&h.wsAllowedFlag, 0)
	return nil
}

// rpcAllowed returns true when JSON-RPC over HTTP is enabled.
func (h *httpServer) rpcAllowed() bool {
	return atomic.LoadInt32(&h.rpcAllowedFlag) == 1
}

// wsAllowed returns true when JSON-RPC over WebSocket is enabled.
func (h *httpServer) wsAllowed() bool {
	return atomic.LoadInt32(&h.wsAllowedFlag) == 1
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.wsAllowed() && isWebsocket(r) {
		h.wsHandler.ServeHTTP(w, r)
		return
	}
	if r.RequestURI == "/" {
		if h.rpcAllowed() {
			h.httpHandler.ServeHTTP(w, r)
		} else {
			w.WriteHeader(404)
		}
	} else {
		h.mux.ServeHTTP(w, r)
	}
}

// isWebsocket checks the header of an http request for a websocket upgrade request.
func isWebsocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.ToLower(r.Header.Get("Connection")) == "upgrade"
}

// NewHTTPHandlerStack returns wrapped http-related handlers
func NewHTTPHandlerStack(srv http.Handler, cors []string, vhosts []string) http.Handler {
	// Wrap the CORS-handler within a host-handler
	handler := newCorsHandler(srv, cors)
	handler = newVHostHandler(vhosts, handler)
	return newGzipHandler(handler)
}

func newCorsHandler(srv http.Handler, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(srv)
}

// virtualHostHandler is a handler which validates the Host-header of incoming requests.
// Using virtual hosts can help prevent DNS rebinding attacks, where a 'random' domain name points to
// the service ip address (but without CORS headers). By verifying the targeted virtual host, we can
// ensure that it's a destination that the node operator has defined.
type virtualHostHandler struct {
	vhosts map[string]struct{}
	next   http.Handler
}

func newVHostHandler(vhosts []string, next http.Handler) http.Handler {
	vhostMap := make(map[string]struct{})
	for _, allowedHost := range vhosts {
		vhostMap[strings.ToLower(allowedHost)] = struct{}{}
	}
	return &virtualHostHandler{vhostMap, next}
}

// ServeHTTP serves JSON-RPC requests over HTTP, implements http.Handler
func (h *virtualHostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if r.Host is not set, we can continue serving since a browser would set the Host header
	if r.Host == "" {
		h.next.ServeHTTP(w, r)
		return
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		// Either invalid (too many colons) or no port specified
		host = r.Host
	}
	if ipAddr := net.ParseIP(host); ipAddr != nil {
		// It's an IP address, we can serve that
		h.next.ServeHTTP(w, r)
		return

	}
	// Not an IP address, but a hostname. Need to validate
	if _, exist := h.vhosts["*"]; exist {
		h.next.ServeHTTP(w, r)
		return
	}
	if _, exist := h.vhosts[host]; exist {
		h.next.ServeHTTP(w, r)
		return
	}
	http.Error(w, "invalid host specified", http.StatusForbidden)
}

var gzPool = sync.Pool{
	New: func() interface{} {
		w := gzip.NewWriter(ioutil.Discard)
		return w
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func newGzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)

		gz.Reset(w)
		defer gz.Close()

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

type ipcServer struct {
	log      log.Logger
	endpoint string

	mu       sync.Mutex
	listener net.Listener
	srv      *rpc.Server
	apis     []rpc.API
}

func newIPCServer(log log.Logger, endpoint string) *ipcServer {
	return &ipcServer{log: log, endpoint: endpoint}
}

// Start starts the httpServer's http.Server
func (is *ipcServer) start() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.listener != nil {
		return nil // already running
	}
	listener, srv, err := rpc.StartIPCEndpoint(is.endpoint, is.apis)
	if err != nil {
		return err
	}
	is.log.Info("IPC endpoint opened", "url", is.endpoint)
	is.listener, is.srv = listener, srv
	return nil
}

func (is *ipcServer) stop() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.listener == nil {
		return nil // not running
	}
	err := is.listener.Close()
	is.srv.Stop()
	is.listener, is.srv = nil, nil
	is.log.Info("IPC endpoint closed", "url", is.endpoint)
	return err
}
