// Copyright 2018 The go-ethereum Authors
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

package rpc

import (
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// CheckModuleAvailability checks that all names given in modules are actually
// available API services. It assumes that the MetadataApi module ("rpc") is always available;
// the registration of this "rpc" module happens in NewServer() and is thus common to all endpoints.
func CheckModuleAvailability(modules []string, apis []API) (bad, available []string) {
	availableSet := make(map[string]struct{})
	for _, api := range apis {
		if _, ok := availableSet[api.Namespace]; !ok {
			availableSet[api.Namespace] = struct{}{}
			available = append(available, api.Namespace)
		}
	}
	for _, name := range modules {
		if _, ok := availableSet[name]; !ok && name != MetadataApi {
			bad = append(bad, name)
		}
	}
	return bad, available
}

// StartHTTPEndpoint starts the HTTP RPC endpoint, configured with cors/vhosts/modules.
func StartHTTPEndpoint(endpoint string, apis []API, modules []string, timeouts HTTPTimeouts, handler http.Handler) (net.Listener, error) {
	// Start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, err
	}
	// Make sure timeout values are meaningful
	if timeouts.ReadTimeout < time.Second {
		log.Warn("Sanitizing invalid HTTP read timeout", "provided", timeouts.ReadTimeout, "updated", DefaultHTTPTimeouts.ReadTimeout)
		timeouts.ReadTimeout = DefaultHTTPTimeouts.ReadTimeout
	}
	if timeouts.WriteTimeout < time.Second {
		log.Warn("Sanitizing invalid HTTP write timeout", "provided", timeouts.WriteTimeout, "updated", DefaultHTTPTimeouts.WriteTimeout)
		timeouts.WriteTimeout = DefaultHTTPTimeouts.WriteTimeout
	}
	if timeouts.IdleTimeout < time.Second {
		log.Warn("Sanitizing invalid HTTP idle timeout", "provided", timeouts.IdleTimeout, "updated", DefaultHTTPTimeouts.IdleTimeout)
		timeouts.IdleTimeout = DefaultHTTPTimeouts.IdleTimeout
	}
	// Bundle and start the HTTP server
	httpSrv := &http.Server{
		Handler:      handler,
		ReadTimeout:  timeouts.ReadTimeout,
		WriteTimeout: timeouts.WriteTimeout,
		IdleTimeout:  timeouts.IdleTimeout,
	}
	go httpSrv.Serve(listener)
	return listener, err
}

// StartWSEndpoint starts a websocket endpoint.
func StartWSEndpoint(endpoint string, apis []API, modules []string, wsOrigins []string, exposeAll bool) (net.Listener, *Server, error) {
	if bad, available := CheckModuleAvailability(modules, apis); len(bad) > 0 {
		log.Error("Unavailable modules in WS API list", "unavailable", bad, "available", available)
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := NewServer()
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return nil, nil, err
			}
			log.Debug("WebSocket registered", "service", api.Service, "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}
	go NewWSServer(wsOrigins, handler).Serve(listener)
	return listener, handler, err

}

// StartIPCEndpoint starts an IPC endpoint.
func StartIPCEndpoint(ipcEndpoint string, apis []API) (net.Listener, *Server, error) {
	// Register all the APIs exposed by the services.
	handler := NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
		log.Debug("IPC registered", "namespace", api.Namespace)
	}
	// All APIs registered, start the IPC listener.
	listener, err := ipcListen(ipcEndpoint)
	if err != nil {
		return nil, nil, err
	}
	go handler.ServeListener(listener)
	return listener, handler, nil
}
