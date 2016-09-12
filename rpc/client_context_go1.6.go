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

// +build go1.6,!go1.7

package rpc

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/context"
)

// In Go 1.6, net.Dialer gained the ability to cancel via a channel.

// contextDialer returns a dialer that applies the deadline value from the given context.
func contextDialer(ctx context.Context) *net.Dialer {
	dialer := &net.Dialer{Cancel: ctx.Done(), KeepAlive: tcpKeepAliveInterval}
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
	// We set Timeout on the client for Go <= 1.5. There
	// is no need to do that here because the dial will be canceled
	// by package http.
	req2 := *req
	req2.Cancel = ctx.Done()
	return c, &req2
}
