// Copyright 2016 The go-ethereum Authors
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
	"io"
	"net"
)

// inProcClient is an in-process buffer stream attached to an RPC server.
type inProcClient struct {
	server *Server
	cl     io.Closer
	enc    *json.Encoder
	dec    *json.Decoder
}

// Close tears down the request channel of the in-proc client.
func (c *inProcClient) Close() {
	c.cl.Close()
}

// NewInProcRPCClient creates an in-process buffer stream attachment to a given
// RPC server.
func NewInProcRPCClient(handler *Server) Client {
	p1, p2 := net.Pipe()
	go handler.ServeCodec(NewJSONCodec(p1))
	return &inProcClient{handler, p2, json.NewEncoder(p2), json.NewDecoder(p2)}
}

// Send marshals a message into a json format and injects in into the client
// request channel.
func (c *inProcClient) Send(msg interface{}) error {
	return c.enc.Encode(msg)
}

// Recv reads a message from the response channel and tries to parse it into the
// given msg interface.
func (c *inProcClient) Recv(msg interface{}) error {
	return c.dec.Decode(msg)
}

// Returns the collection of modules the RPC server offers.
func (c *inProcClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(c)
}
