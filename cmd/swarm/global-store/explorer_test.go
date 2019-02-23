// Copyright 2019 The go-ethereum Authors
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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/explorer"
	mockRPC "github.com/ethereum/go-ethereum/swarm/storage/mock/rpc"
)

// TestExplorer validates basic chunk explorer functionality by storing
// a small set of chunk and making http requests on exposed endpoint.
// Full chunk explorer validation is done in mock/explorer package.
func TestExplorer(t *testing.T) {
	addr := findFreeTCPAddress(t)
	explorerAddr := findFreeTCPAddress(t)
	testCmd := runGlobalStore(t, "ws", "--addr", addr, "--explorer-address", explorerAddr)
	defer testCmd.Kill()

	client := websocketClient(t, addr)

	store := mockRPC.NewGlobalStore(client)
	defer store.Close()

	nodeKeys := map[string][]string{
		"a1": {"b1", "b2", "b3"},
		"a2": {"b3", "b4", "b5"},
	}

	keyNodes := make(map[string][]string)

	for addr, keys := range nodeKeys {
		for _, key := range keys {
			keyNodes[key] = append(keyNodes[key], addr)
		}
	}

	invalidAddr := "c1"
	invalidKey := "d1"

	for addr, keys := range nodeKeys {
		for _, key := range keys {
			err := store.Put(common.HexToAddress(addr), common.Hex2Bytes(key), []byte("data"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	endpoint := "http://" + explorerAddr

	t.Run("has key", func(t *testing.T) {
		for addr, keys := range nodeKeys {
			for _, key := range keys {
				testStatusResponse(t, endpoint+"/api/has-key/"+addr+"/"+key, http.StatusOK)
				testStatusResponse(t, endpoint+"/api/has-key/"+invalidAddr+"/"+key, http.StatusNotFound)
			}
			testStatusResponse(t, endpoint+"/api/has-key/"+addr+"/"+invalidKey, http.StatusNotFound)
		}
		testStatusResponse(t, endpoint+"/api/has-key/"+invalidAddr+"/"+invalidKey, http.StatusNotFound)
	})

	t.Run("keys", func(t *testing.T) {
		var keys []string
		for key := range keyNodes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		testKeysResponse(t, endpoint+"/api/keys", explorer.KeysResponse{
			Keys: keys,
		})
	})

	t.Run("nodes", func(t *testing.T) {
		var nodes []string
		for addr := range nodeKeys {
			nodes = append(nodes, common.HexToAddress(addr).Hex())
		}
		sort.Strings(nodes)
		testNodesResponse(t, endpoint+"/api/nodes", explorer.NodesResponse{
			Nodes: nodes,
		})
	})

	t.Run("node keys", func(t *testing.T) {
		for addr, keys := range nodeKeys {
			testKeysResponse(t, endpoint+"/api/keys?node="+addr, explorer.KeysResponse{
				Keys: keys,
			})
		}
		testKeysResponse(t, endpoint+"/api/keys?node="+invalidAddr, explorer.KeysResponse{})
	})

	t.Run("key nodes", func(t *testing.T) {
		for key, addrs := range keyNodes {
			var nodes []string
			for _, addr := range addrs {
				nodes = append(nodes, common.HexToAddress(addr).Hex())
			}
			sort.Strings(nodes)
			testNodesResponse(t, endpoint+"/api/nodes?key="+key, explorer.NodesResponse{
				Nodes: nodes,
			})
		}
		testNodesResponse(t, endpoint+"/api/nodes?key="+invalidKey, explorer.NodesResponse{})
	})
}

// TestExplorer_CORSOrigin validates if chunk explorer returns
// correct CORS origin header in GET and OPTIONS requests.
func TestExplorer_CORSOrigin(t *testing.T) {
	origin := "http://localhost/"
	addr := findFreeTCPAddress(t)
	explorerAddr := findFreeTCPAddress(t)
	testCmd := runGlobalStore(t, "ws",
		"--addr", addr,
		"--explorer-address", explorerAddr,
		"--explorer-cors-origin", origin,
	)
	defer testCmd.Kill()

	// wait until the server is started
	waitHTTPEndpoint(t, explorerAddr)

	url := "http://" + explorerAddr + "/api/keys"

	t.Run("get", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		header := resp.Header.Get("Access-Control-Allow-Origin")
		if header != origin {
			t.Errorf("got Access-Control-Allow-Origin header %q, want %q", header, origin)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		header := resp.Header.Get("Access-Control-Allow-Origin")
		if header != origin {
			t.Errorf("got Access-Control-Allow-Origin header %q, want %q", header, origin)
		}
	})
}

// testStatusResponse makes an http request to provided url
// and validates if response is explorer.StatusResponse for
// the expected status code.
func testStatusResponse(t *testing.T, url string, code int) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != code {
		t.Errorf("got status code %v, want %v", resp.StatusCode, code)
	}
	var r explorer.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if r.Code != code {
		t.Errorf("got response code %v, want %v", r.Code, code)
	}
	if r.Message != http.StatusText(code) {
		t.Errorf("got response message %q, want %q", r.Message, http.StatusText(code))
	}
}

// testKeysResponse makes an http request to provided url
// and validates if response machhes expected explorer.KeysResponse.
func testKeysResponse(t *testing.T, url string, want explorer.KeysResponse) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status code %v, want %v", resp.StatusCode, http.StatusOK)
	}
	var r explorer.KeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(r.Keys) != fmt.Sprint(want.Keys) {
		t.Errorf("got keys %v, want %v", r.Keys, want.Keys)
	}
	if r.Next != want.Next {
		t.Errorf("got next %s, want %s", r.Next, want.Next)
	}
}

// testNodeResponse makes an http request to provided url
// and validates if response machhes expected explorer.NodeResponse.
func testNodesResponse(t *testing.T, url string, want explorer.NodesResponse) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status code %v, want %v", resp.StatusCode, http.StatusOK)
	}
	var r explorer.NodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(r.Nodes) != fmt.Sprint(want.Nodes) {
		t.Errorf("got nodes %v, want %v", r.Nodes, want.Nodes)
	}
	if r.Next != want.Next {
		t.Errorf("got next %s, want %s", r.Next, want.Next)
	}
}
