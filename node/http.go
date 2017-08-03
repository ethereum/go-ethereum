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
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/rs/cors"
)

type httpEndpoint struct {
	fd  net.Listener
	srv http.Server

	// Multiple handlers can be attached to a single endpoint.
	mu   sync.Mutex
	http http.Handler
	ws   http.Handler
}

// listenHTTP starts a new HTTP server or returns an existing one.
// n.lock must be held by the caller.
func (n *Node) listenHTTP(endpoint string) (*httpEndpoint, error) {
	_, portstring, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q", endpoint)
	}
	port, err := strconv.Atoi(portstring)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q", endpoint)
	}
	if port != 0 && n.httpServers[port] != nil {
		// Use existing server.
		return n.httpServers[port], nil
	}
	fd, err := net.Listen("tcp", endpoint)
	if err != nil {
		return nil, err
	}
	h := &httpEndpoint{fd: fd}
	h.srv.Handler = h
	n.httpServers[fd.Addr().(*net.TCPAddr).Port] = h
	go h.srv.Serve(fd)
	return h, nil
}

func (n *Node) stopAllHTTP() {
	for _, ep := range n.httpServers {
		ep.stop()
	}
}

func (ep *httpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ep.handler(req).ServeHTTP(resp, req)
}

func (ep *httpEndpoint) setWS(h http.Handler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.ws = h
}

func (ep *httpEndpoint) setHTTP(h http.Handler) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.http = h
}

func (ep *httpEndpoint) stop() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	err := ep.fd.Close()
	// TODO close rpc server too
	log.Info(fmt.Sprintf("HTTP server closed: http://%s", ep.fd.Addr()))
	return err
}

func (ep *httpEndpoint) handler(req *http.Request) http.Handler {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	switch {
	case ep.ws != nil && req.Method == "GET" && strings.ToLower(req.Header.Get("Upgrade")) == "websocket":
		return ep.ws
	case ep.http != nil:
		return ep.http
	default:
		return http.HandlerFunc(defaultHandler)
	}
}

func defaultHandler(resp http.ResponseWriter, req *http.Request) {
	http.Error(resp, "no handler configured on this endpoint, please try again later", http.StatusServiceUnavailable)
}

func newCorsHandler(corsOrigins []string, h http.Handler) http.Handler {
	if len(corsOrigins) == 0 {
		// disable CORS support if user has not specified a custom CORS configuration
		return h
	}
	c := cors.New(cors.Options{
		AllowedOrigins: corsOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(h)
}
