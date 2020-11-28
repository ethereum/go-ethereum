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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
)

func TestBuildSchema(t *testing.T) {
	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		t.Fatalf("could not create new node: %v", err)
	}
	// Make sure the schema can be parsed and matched up to the object model.
	if err := newHandler(stack, nil, []string{}, []string{}); err != nil {
		t.Errorf("Could not construct GraphQL handler: %v", err)
	}
}

// Tests that a graphQL request is successfully handled when graphql is enabled on the specified endpoint
func TestGraphQLBlockSerialization(t *testing.T) {
	stack := createNode(t, true)
	defer stack.Close()
	// start node
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}

	for i, tt := range []struct {
		name string
		body string
		want string
		code int
	}{
		{
			name: "get_number",
			body: `{"query": "{block{number}}","variables": null}`,
			want: `{"data":{"block":{"number":0}}}`,
			code: 200,
		},
		{
			name: "get_numeric_fields",
			body: `{"query": "{block{number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":{"number":0,"gasUsed":0,"gasLimit":11500000}}}`,
			code: 200,
		},
		{
			name: "number_0",
			body: `{"query": "{block(number:0){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":{"number":0,"gasUsed":0,"gasLimit":11500000}}}`,
			code: 200,
		},
		{
			name: "number_-1",
			body: `{"query": "{block(number:-1){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":null}}`,
			code: 200,
		},
		{
			name: "number_-500",
			body: `{"query": "{block(number:-500){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":null}}`,
			code: 200,
		},
		{
			name: "string_0",
			body: `{"query": "{block(number:\"0\"){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":{"number":0,"gasUsed":0,"gasLimit":11500000}}}`,
			code: 200,
		},
		{
			name: "string_-33",
			body: `{"query": "{block(number:\"-33\"){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":null}}`,
			code: 200,
		},
		{
			name: "string_1337",
			body: `{"query": "{block(number:\"1337\"){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"data":{"block":null}}`,
			code: 200,
		},
		// remove to allow hex string support
		{
			name: "string_1337",
			body: `{"query": "{block(number:\"0xbad\"){number,gasUsed,gasLimit}}","variables": null}`,
			want: `{"errors":[{"message":"strconv.ParseInt: parsing \"0xbad\": invalid syntax"}],"data":{}}`,
			code: 400,
		},
		// uncomment to test hex string support
		//{
		//	name: "string_0x0",
		//	body: `{"query": "{block(number:\"0x0\"){number,gasUsed,gasLimit}}","variables": null}`,
		//	want: `{"data":{"block":{"number":0,"gasUsed":0,"gasLimit":11500000}}}`,
		//	code: 200,
		//},
		//{
		//	name: "string_0xbad",
		//	body: `{"query": "{block(number:\"0xxxxbad\"){number,gasUsed,gasLimit}}","variables": null}`,
		//	want: `{"errors":[{"message":"invalid hex string"}],"data":{}}`,
		//	code: 400,
		//},
		//{
		//	name: "string_0xa",
		//	body: `{"query": "{block(number:\"0xa\"){number,gasUsed,gasLimit}}","variables": null}`,
		//	want: `{"data":{"block":null}}`,
		//	code: 200,
		//},
		{
			name: "bleh_query",
			body: `{"query": "{bleh{number}}","variables": null}"`,
			want: `{"errors":[{"message":"Cannot query field \"bleh\" on type \"Query\".","locations":[{"line":1,"column":2}]}]}`,
			code: 400,
		},
	} {
		resp, err := http.Post(fmt.Sprintf("http://%s/graphql", "127.0.0.1:9393"), "application/json", strings.NewReader(tt.body))
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("could not read from response body: %v", err)
		}
		if have := string(bodyBytes); have != tt.want {
			t.Errorf("testcase %s (%d), have:\n%v\nwant:\n%v", tt.name, i, have, tt.want)
		}
		if tt.code != resp.StatusCode {
			t.Errorf("testcase %s (%d), wrong statuscode, have:\n%v\nwant:%v", tt.name, i, resp.StatusCode, tt.code)
		}
	}
}

// Tests that a graphQL request is not handled successfully when graphql is not enabled on the specified endpoint
func TestGraphQLHTTPOnSamePort_GQLRequest_Unsuccessful(t *testing.T) {
	stack := createNode(t, false)
	defer stack.Close()
	if err := stack.Start(); err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	body := strings.NewReader(`{"query": "{block{number}}","variables": null}`)
	resp, err := http.Post(fmt.Sprintf("http://%s/graphql", "127.0.0.1:9393"), "application/json", body)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read from response body: %v", err)
	}
	// make sure the request is not handled successfully
	if want, have := "404 page not found\n", string(bodyBytes); have != want {
		t.Errorf("have:\n%v\nwant:\n%v", have, want)
	}
	if want, have := 404, resp.StatusCode; want != have {
		t.Errorf("wrong statuscode, have:\n%v\nwant:%v", have, want)
	}
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
	// create backend (use a config which is light on mem consumption)
	ethConf := &eth.Config{
		Genesis: core.DeveloperGenesisBlock(15, common.Address{}),
		Miner: miner.Config{
			Etherbase: common.HexToAddress("0xaabb"),
		},
		Ethash: ethash.Config{
			PowMode: ethash.ModeTest,
		},
		NetworkId:               1337,
		TrieCleanCache:          5,
		TrieCleanCacheJournal:   "triecache",
		TrieCleanCacheRejournal: 60 * time.Minute,
		TrieDirtyCache:          5,
		TrieTimeout:             60 * time.Minute,
		SnapshotCache:           5,
	}
	ethBackend, err := eth.New(stack, ethConf)
	if err != nil {
		t.Fatalf("could not create eth backend: %v", err)
	}

	// create gql service
	err = New(stack, ethBackend.APIBackend, []string{}, []string{})
	if err != nil {
		t.Fatalf("could not create graphql service: %v", err)
	}
}
