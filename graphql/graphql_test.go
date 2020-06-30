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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/assert"
)

func TestBuildSchema(t *testing.T) {
	// Make sure the schema can be parsed and matched up to the object model.
	if _, err := newHandler(nil); err != nil {
		t.Errorf("Could not construct GraphQL handler: %v", err)
	}
}

// Tests that a graphql handler can be added to an existing HTTPServer
func TestGQLAllowed(t *testing.T) {
	stack := createNode(t, true)
	defer stack.Close()
	// start node
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// check that server was created
	server := stack.ExistingHTTPServer("127.0.0.1:9393")
	if server == nil {
		t.Errorf("server was not created on the given endpoint")
	}
	// assert that server allows GQL requests
	assert.True(t, server.GQLAllowed)
}

// Tests to make sure an HTTPServer is created that handles for http, ws, and graphQL
func TestMultiplexedServer(t *testing.T) {
	stack := createNode(t, true)
	defer stack.Close()
	// start the node
	if err := stack.Start(); err != nil {
		t.Error("could not start http service on node ", err)
	}
	server := stack.ExistingHTTPServer("127.0.0.1:9393")
	if server == nil {
		t.Fatalf("server was not configured on the given endpoint")
	}
	assert.True(t, server.RPCAllowed)
	assert.True(t, server.WSAllowed)
	assert.True(t, server.GQLAllowed)
}

// Tests that a graphQL request is successfully handled when graphql is enabled on the specified endpoint
func TestGraphQLHTTPOnSamePort_GQLRequest_Successful(t *testing.T) {
	stack := createNode(t, true)
	defer stack.Close()
	// start node
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// create http request
	body := strings.NewReader("{\"query\": \"{block{number}}\",\"variables\": null}")
	gqlReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/graphql", "127.0.0.1:9393"), body)
	if err != nil {
		t.Error("could not issue new http request ", err)
	}
	gqlReq.Header.Set("Content-Type", "application/json")
	// read from response
	resp := doHTTPRequest(t, gqlReq)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read from response body: %v", err)
	}
	expected := "{\"data\":{\"block\":{\"number\":\"0x0\"}}}"
	assert.Equal(t, expected, string(bodyBytes))
}

// Tests that a graphQL request is not handled successfully when graphql is not enabled on the specified endpoint
func TestGraphQLHTTPOnSamePort_GQLRequest_Unsuccessful(t *testing.T) {
	stack := createNode(t, false)
	defer stack.Close()
	// start node
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// make sure GQL is not enabled
	server := stack.ExistingHTTPServer("127.0.0.1:9797")
	if server == nil {
		t.Fatalf("server was not created on the given endpoint")
	}
	assert.False(t, server.GQLAllowed)
	// create http request
	body := strings.NewReader("{\"query\": \"{block{number}}\",\"variables\": null}")
	gqlReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/graphql", "127.0.0.1:9797"), body)
	if err != nil {
		t.Error("could not issue new http request ", err)
	}
	gqlReq.Header.Set("Content-Type", "application/json")
	// read from response
	resp := doHTTPRequest(t, gqlReq)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read from response body: %v", err)
	}
	// make sure the request is not handled successfully
	expected := "{\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32600,\"message\":\"invalid request\"}}\n"
	assert.Equal(t, string(bodyBytes), expected)
}

// Tests that graphql can be successfully enabled on a separate port than rpc and ws.
func TestGraphqlOnSeparatePort(t *testing.T) {
	stack := createNode(t, false)
	defer stack.Close()

	separateTestEndpoint := "127.0.0.1:7474"

	createGQLService(t, stack, separateTestEndpoint)
	// start node
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// create http request
	body := strings.NewReader("{\"query\": \"{block{number}}\",\"variables\": null}")
	gqlReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/graphql", separateTestEndpoint), body)
	if err != nil {
		t.Error("could not issue new http request ", err)
	}
	gqlReq.Header.Set("Content-Type", "application/json")
	// read from response
	resp := doHTTPRequest(t, gqlReq)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read from response body: %v", err)
	}
	expected := "{\"data\":{\"block\":{\"number\":\"0x0\"}}}"
	assert.Equal(t, expected, string(bodyBytes))

}

func createNode(t *testing.T, gqlEnabled bool) *node.Node {
	stack, err := node.New(&node.Config{
		HTTPHost: "127.0.0.1",
		HTTPPort: 9393,
		WSHost:   "127.0.0.1",
		WSPort:   9393,
	})
	if err != nil {
		t.Fatalf("could not create node: %v", err)
	}
	if !gqlEnabled {
		return stack
	}

	createGQLService(t, stack, "127.0.0.1:9393")

	return stack
}

func createGQLService(t *testing.T, stack *node.Node, endpoint string) {
	// create backend
	ethBackend, err := eth.New(stack, &eth.DefaultConfig)
	if err != nil {
		t.Fatalf("could not create eth backend: %v", err)
	}

	// create gql service
	err = New(stack, ethBackend.APIBackend, endpoint, []string{}, []string{}, rpc.DefaultHTTPTimeouts)
	if err != nil {
		t.Fatalf("could not create graphql service: %v", err)
	}
}

func doHTTPRequest(t *testing.T, req *http.Request) *http.Response {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("could not issue a GET request to the given endpoint", err)

	}
	return resp
}
