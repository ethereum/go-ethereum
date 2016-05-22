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
	"encoding/json"
	"net"
)

// CreateIPCListener creates an listener, on Unix platforms this is a unix socket, on Windows this is a named pipe
func CreateIPCListener(endpoint string) (net.Listener, error) {
	return ipcListen(endpoint)
}

// ipcClient represent an IPC RPC client. It will connect to a given endpoint and tries to communicate with a node using
// JSON serialization.
type ipcClient struct {
	endpoint string
	conn     net.Conn
	out      *json.Encoder
	in       *json.Decoder
}

// NewIPCClient create a new IPC client that will connect on the given endpoint. Messages are JSON encoded and encoded.
// On Unix it assumes the endpoint is the full path to a unix socket, and Windows the endpoint is an identifier for a
// named pipe.
func NewIPCClient(endpoint string) (Client, error) {
	conn, err := newIPCConnection(endpoint)
	if err != nil {
		return nil, err
	}
	return &ipcClient{endpoint: endpoint, conn: conn, in: json.NewDecoder(conn), out: json.NewEncoder(conn)}, nil
}

// Send will serialize the given message and send it to the server.
// When sending the message fails it will try to reconnect once and send the message again.
func (client *ipcClient) Send(msg interface{}) error {
	if err := client.out.Encode(msg); err == nil {
		return nil
	}

	// retry once
	client.conn.Close()

	conn, err := newIPCConnection(client.endpoint)
	if err != nil {
		return err
	}

	client.conn = conn
	client.in = json.NewDecoder(conn)
	client.out = json.NewEncoder(conn)

	return client.out.Encode(msg)
}

// Recv will read a message from the connection and tries to parse it. It assumes the received message is JSON encoded.
func (client *ipcClient) Recv(msg interface{}) error {
	return client.in.Decode(&msg)
}

// Close will close the underlying IPC connection
func (client *ipcClient) Close() {
	client.conn.Close()
}

// SupportedModules will return the collection of offered RPC modules.
func (client *ipcClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(client)
}
