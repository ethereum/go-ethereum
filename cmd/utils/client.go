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

package utils

import (
	"encoding/json"
	"fmt"

	"strings"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewInProcRPCClient will start a new RPC server for the given node and returns a client to interact with it.
func NewInProcRPCClient(stack *node.Node) *inProcClient {
	server := rpc.NewServer()

	offered := stack.APIs()
	for _, api := range offered {
		server.RegisterName(api.Namespace, api.Service)
	}

	web3 := node.NewPublicWeb3API(stack)
	server.RegisterName("web3", web3)

	var ethereum *eth.Ethereum
	if err := stack.Service(&ethereum); err == nil {
		net := eth.NewPublicNetAPI(stack.Server(), ethereum.NetVersion())
		server.RegisterName("net", net)
	} else {
		glog.V(logger.Warn).Infof("%v\n", err)
	}

	buf := &buf{
		requests:  make(chan []byte),
		responses: make(chan []byte),
	}
	client := &inProcClient{
		server: server,
		buf:    buf,
	}

	go func() {
		server.ServeCodec(rpc.NewJSONCodec(client.buf))
	}()

	return client
}

// buf represents the connection between the RPC server and console
type buf struct {
	readBuf   []byte      // store remaining request bytes after a partial read
	requests  chan []byte // list with raw serialized requests
	responses chan []byte // list with raw serialized responses
}

// will read the next request in json format
func (b *buf) Read(p []byte) (int, error) {
	// last read didn't read entire request, return remaining bytes
	if len(b.readBuf) > 0 {
		n := copy(p, b.readBuf)
		if n < len(b.readBuf) {
			b.readBuf = b.readBuf[:n]
		} else {
			b.readBuf = b.readBuf[:0]
		}
		return n, nil
	}

	// read next request
	req := <-b.requests
	n := copy(p, req)
	if n < len(req) {
		// buf too small, store remaining chunk for next read
		b.readBuf = req[n:]
	}

	return n, nil
}

// Write send the given buffer to the backend
func (b *buf) Write(p []byte) (n int, err error) {
	b.responses <- p
	return len(p), nil
}

// Close cleans up obtained resources.
func (b *buf) Close() error {
	close(b.requests)
	close(b.responses)

	return nil
}

// inProcClient starts a RPC server and uses buf to communicate with it.
type inProcClient struct {
	server *rpc.Server
	buf    *buf
}

// Close will stop the RPC server
func (c *inProcClient) Close() {
	c.server.Stop()
}

// Send a msg to the endpoint
func (c *inProcClient) Send(msg interface{}) error {
	d, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.buf.requests <- d
	return nil
}

// Recv reads a message and tries to parse it into the given msg
func (c *inProcClient) Recv(msg interface{}) error {
	data := <-c.buf.responses
	return json.Unmarshal(data, &msg)
}

// Returns the collection of modules the RPC server offers.
func (c *inProcClient) SupportedModules() (map[string]string, error) {
	return rpc.SupportedModules(c)
}

// NewRemoteRPCClient returns a RPC client which connects to a running geth instance.
// Depending on the given context this can either be a IPC or a HTTP client.
func NewRemoteRPCClient(ctx *cli.Context) (rpc.Client, error) {
	if ctx.Args().Present() {
		endpoint := ctx.Args().First()
		return NewRemoteRPCClientFromString(endpoint)
	}

	// use IPC by default
	endpoint := IPCSocketPath(ctx)
	return rpc.NewIPCClient(endpoint)
}

// NewRemoteRPCClientFromString returns a RPC client which connects to the given
// endpoint. It must start with either `ipc:` or `rpc:` (HTTP).
func NewRemoteRPCClientFromString(endpoint string) (rpc.Client, error) {
	if strings.HasPrefix(endpoint, "ipc:") {
		return rpc.NewIPCClient(endpoint[4:])
	}
	if strings.HasPrefix(endpoint, "rpc:") {
		return rpc.NewHTTPClient(endpoint[4:])
	}
	if strings.HasPrefix(endpoint, "http://") {
		return rpc.NewHTTPClient(endpoint)
	}
	if strings.HasPrefix(endpoint, "ws:") {
		return rpc.NewWSClient(endpoint)
	}

	return nil, fmt.Errorf("invalid endpoint")
}
