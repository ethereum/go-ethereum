// Copyright 2019 The go-ethereum Authors
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

package graphql

import (
	"fmt"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildSchema(t *testing.T) {
	// Make sure the schema can be parsed and matched up to the object model.
	if _, err := newHandler(nil); err != nil {
		t.Errorf("Could not construct GraphQL handler: %v", err)
	}
}

// Tests that a graphql handler can be added to an existing HTTPServer
func TestGQLAllowed(t *testing.T) {
	stack, err := node.New(&node.Config{
		HTTPHost:              node.DefaultHTTPHost,
		HTTPPort:              9393,
	})
	if err != nil {
		t.Fatalf("could not create node: %v", err)
	}
	defer stack.Close()
	// create backend
	ethBackend, err  := eth.New(stack, &eth.DefaultConfig)
	if err != nil {
		t.Fatalf("could not create eth backend: %v", err)
	}
	// set endpoint and create new gql service
	endpoint := fmt.Sprintf("%s:%d", node.DefaultHTTPHost, 9393)
	err = New(stack,ethBackend,nil, endpoint, []string{}, []string{}, rpc.DefaultHTTPTimeouts)
	if err != nil {
		t.Fatalf("could not create graphql service: %v", err)
	}
	// start node
	if err = stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// check that server was created
	server := stack.ExistingHTTPServer(endpoint)
	if server == nil {
		t.Errorf("server was not created on the given endpoint: %v", err)
	}
	// assert that server allows GQL requests
	assert.True(t, server.GQLAllowed)
}
