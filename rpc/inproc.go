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

import "encoding/json"

// NewInProcRPCClient creates an in-process buffer stream attachment to a given
// RPC server.
func NewInProcRPCClient(handler *Server) Client {
	buffer := &inprocBuffer{
		requests:  make(chan []byte, 16),
		responses: make(chan []byte, 16),
	}
	client := &inProcClient{
		server: handler,
		buffer: buffer,
	}
	go handler.ServeCodec(NewJSONCodec(client.buffer))
	return client
}

// inProcClient is an in-process buffer stream attached to an RPC server.
type inProcClient struct {
	server *Server
	buffer *inprocBuffer
}

// Close tears down the request channel of the in-proc client.
func (c *inProcClient) Close() {
	c.buffer.Close()
}

// Send marshals a message into a json format and injects in into the client
// request channel.
func (c *inProcClient) Send(msg interface{}) error {
	d, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.buffer.requests <- d
	return nil
}

// Recv reads a message from the response channel and tries to parse it into the
// given msg interface.
func (c *inProcClient) Recv(msg interface{}) error {
	data := <-c.buffer.responses
	return json.Unmarshal(data, &msg)
}

// Returns the collection of modules the RPC server offers.
func (c *inProcClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(c)
}

// inprocBuffer represents the connection between the RPC server and console
type inprocBuffer struct {
	readBuf   []byte      // store remaining request bytes after a partial read
	requests  chan []byte // list with raw serialized requests
	responses chan []byte // list with raw serialized responses
}

// Read will read the next request in json format.
func (b *inprocBuffer) Read(p []byte) (int, error) {
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
		// inprocBuffer too small, store remaining chunk for next read
		b.readBuf = req[n:]
	}
	return n, nil
}

// Write sends the given buffer to the backend.
func (b *inprocBuffer) Write(p []byte) (n int, err error) {
	b.responses <- p
	return len(p), nil
}

// Close cleans up obtained resources.
func (b *inprocBuffer) Close() error {
	close(b.requests)
	close(b.responses)

	return nil
}
