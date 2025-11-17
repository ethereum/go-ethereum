// Copyright 2022 The go-ethereum Authors
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

package rpc_test

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// This example configures a HTTP-based RPC client with two options - one setting the
// overall request timeout, the other adding a custom HTTP header to all requests.
func ExampleDialOptions() {
	tokenHeader := rpc.WithHeader("x-token", "foo")
	httpClient := rpc.WithHTTPClient(&http.Client{
		Timeout: 10 * time.Second,
	})

	ctx := context.Background()
	c, err := rpc.DialOptions(ctx, "http://rpc.example.com", httpClient, tokenHeader)
	if err != nil {
		panic(err)
	}
	c.Close()
}
