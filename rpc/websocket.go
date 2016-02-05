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

package rpc

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/websocket"
	"gopkg.in/fatih/set.v0"
)

// wsReaderWriterCloser reads and write payloads from and to a websocket  connection.
type wsReaderWriterCloser struct {
	c *websocket.Conn
}

// Read will read incoming payload data into p.
func (rw *wsReaderWriterCloser) Read(p []byte) (int, error) {
	return rw.c.Read(p)
}

// Write writes p to the websocket.
func (rw *wsReaderWriterCloser) Write(p []byte) (int, error) {
	return rw.c.Write(p)
}

// Close closes the websocket connection.
func (rw *wsReaderWriterCloser) Close() error {
	return rw.c.Close()
}

// wsHandshakeValidator returns a handler that verifies the origin during the
// websocket upgrade process. When a '*' is specified as an allowed origins all
// connections are accepted.
func wsHandshakeValidator(allowedOrigins []string) func(*websocket.Config, *http.Request) error {
	origins := set.New()
	allowAllOrigins := false

	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
		}
		if origin != "" {
			origins.Add(origin)
		}
	}

	// allow localhost if no allowedOrigins are specified
	if len(origins.List()) == 0 {
		origins.Add("http://localhost")
		if hostname, err := os.Hostname(); err == nil {
			origins.Add("http://" + hostname)
		}
	}

	glog.V(logger.Debug).Infof("Allowed origin(s) for WS RPC interface %v\n", origins.List())

	f := func(cfg *websocket.Config, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if allowAllOrigins || origins.Has(origin) {
			return nil
		}
		glog.V(logger.Debug).Infof("origin '%s' not allowed on WS-RPC interface\n", origin)
		return fmt.Errorf("origin %s not allowed", origin)
	}

	return f
}

// NewWSServer creates a new websocket RPC server around an API provider.
func NewWSServer(cors string, handler *Server) *http.Server {
	return &http.Server{
		Handler: websocket.Server{
			Handshake: wsHandshakeValidator(strings.Split(cors, ",")),
			Handler: func(conn *websocket.Conn) {
				handler.ServeCodec(NewJSONCodec(&wsReaderWriterCloser{conn}))
			},
		},
	}
}

// wsClient represents a RPC client that communicates over websockets with a
// RPC server.
type wsClient struct {
	endpoint string
	connMu   sync.Mutex
	conn     *websocket.Conn
}

// NewWSClientj creates a new RPC client that communicates with a RPC server
// that is listening on the given endpoint using JSON encoding.
func NewWSClient(endpoint string) (*wsClient, error) {
	return &wsClient{endpoint: endpoint}, nil
}

// connection will return a websocket connection to the RPC server. It will
// (re)connect when necessary.
func (client *wsClient) connection() (*websocket.Conn, error) {
	if client.conn != nil {
		return client.conn, nil
	}

	origin, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	origin = "http://" + origin
	client.conn, err = websocket.Dial(client.endpoint, "", origin)

	return client.conn, err
}

// SupportedModules is the collection of modules the RPC server offers.
func (client *wsClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(client)
}

// Send writes the JSON serialized msg to the websocket. It will create a new
// websocket connection to the server if the client is currently not connected.
func (client *wsClient) Send(msg interface{}) (err error) {
	client.connMu.Lock()
	defer client.connMu.Unlock()

	var conn *websocket.Conn
	if conn, err = client.connection(); err == nil {
		if err = websocket.JSON.Send(conn, msg); err != nil {
			client.conn.Close()
			client.conn = nil
		}
	}

	return err
}

// Recv reads a JSON message from the websocket and unmarshals it into msg.
func (client *wsClient) Recv(msg interface{}) (err error) {
	client.connMu.Lock()
	defer client.connMu.Unlock()

	var conn *websocket.Conn
	if conn, err = client.connection(); err == nil {
		if err = websocket.JSON.Receive(conn, msg); err != nil {
			client.conn.Close()
			client.conn = nil
		}
	}
	return
}

// Close closes the underlaying websocket connection.
func (client *wsClient) Close() {
	client.connMu.Lock()
	defer client.connMu.Unlock()

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}

}
