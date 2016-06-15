// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

// NewRemoteRPCClient returns a RPC client which connects to a running geth instance.
// Depending on the given context this can either be a IPC or a HTTP client.
func NewRemoteRPCClient(ctx *cli.Context) (rpc.Client, error) {
	if ctx.Args().Present() {
		endpoint := ctx.Args().First()
		return NewRemoteRPCClientFromString(endpoint)
	}
	// use IPC by default
	return rpc.NewIPCClient(node.DefaultIPCEndpoint())
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
