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

// +build !go1.5

package rpc

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/context"
)

// In older versions of Go (below 1.5), dials cannot be canceled
// via a channel or context. The context deadline can still applied.

// contextDialer returns a dialer that applies the deadline value from the given context.
func contextDialer(ctx context.Context) *net.Dialer {
	dialer := &net.Dialer{KeepAlive: tcpKeepAliveInterval}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	} else {
		dialer.Deadline = time.Now().Add(defaultDialTimeout)
	}
	return dialer
}

// dialContext connects to the given address, aborting the dial if ctx is canceled.
func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return contextDialer(ctx).Dial(network, addr)
}

// requestWithContext copies req, adding the cancelation channel and deadline from ctx.
func requestWithContext(c *http.Client, req *http.Request, ctx context.Context) (*http.Client, *http.Request) {
	// Set Timeout on the client if the context has a deadline.
	// Note that there is no default timeout (unlike in contextDialer) because
	// the timeout applies to the entire request, including reads from body.
	if deadline, ok := ctx.Deadline(); ok {
		c2 := *c
		c2.Timeout = deadline.Sub(time.Now())
		c = &c2
	}
	req2 := *req
	return c, &req2
}
