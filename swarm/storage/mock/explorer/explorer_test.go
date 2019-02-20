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

package explorer

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/db"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
)

// TestHandler_memGlobalStore runs a set of tests
// to validate handler with mem global store.
func TestHandler_memGlobalStore(t *testing.T) {
	t.Parallel()

	globalStore := mem.NewGlobalStore()

	testHandler(t, globalStore)
}

// TestHandler_dbGlobalStore runs a set of tests
// to validate handler with database global store.
func TestHandler_dbGlobalStore(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "swarm-mock-explorer-db-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	globalStore, err := db.NewGlobalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer globalStore.Close()

	testHandler(t, globalStore)
}

// testHandler stores data distributed by node addresses
// and validates if this data is correctly retrievable
// by using the http.Handler returned by NewHandler function.
// This test covers all HTTP routes and various get parameters
// on them to check paginated results.
func testHandler(t *testing.T, globalStore mock.GlobalStorer) {
	const (
		nodeCount       = 350
		keyCount        = 250
		keysOnNodeCount = 150
	)

	// keys for every node
	nodeKeys := make(map[string][]string)

	// a node address that is not present in global store
	invalidAddr := "0x7b8b72938c254cf002c4e1e714d27e022be88d93"

	// a key that is not present in global store
	invalidKey := "f9824192fb515cfb"

	for i := 1; i <= nodeCount; i++ {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(i))
		addr := common.BytesToAddress(b).Hex()
		nodeKeys[addr] = make([]string, 0)
	}

	for i := 1; i <= keyCount; i++ {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(i))

		key := common.Bytes2Hex(b)

		var c int
		for addr := range nodeKeys {
			nodeKeys[addr] = append(nodeKeys[addr], key)
			c++
			if c >= keysOnNodeCount {
				break
			}
		}
	}

	// sort keys for every node as they are expected to be
	// sorted in HTTP responses
	for _, keys := range nodeKeys {
		sort.Strings(keys)
	}

	// nodes for every key
	keyNodes := make(map[string][]string)

	// construct a reverse mapping of nodes for every key
	for addr, keys := range nodeKeys {
		for _, key := range keys {
			keyNodes[key] = append(keyNodes[key], addr)
		}
	}

	// sort node addresses with case insensitive sort,
	// as hex letters in node addresses are in mixed caps
	for _, addrs := range keyNodes {
		sortCaseInsensitive(addrs)
	}

	// find a key that is not stored at the address
	var (
		unmatchedAddr string
		unmatchedKey  string
	)
	for addr, keys := range nodeKeys {
		for key := range keyNodes {
			var found bool
			for _, k := range keys {
				if k == key {
					found = true
					break
				}
			}
			if !found {
				unmatchedAddr = addr
				unmatchedKey = key
			}
			break
		}
		if unmatchedAddr != "" {
			break
		}
	}
	// check if unmatched key/address pair is found
	if unmatchedAddr == "" || unmatchedKey == "" {
		t.Fatalf("could not find a key that is not associated with a node")
	}

	// store the data
	for addr, keys := range nodeKeys {
		for _, key := range keys {
			err := globalStore.Put(common.HexToAddress(addr), common.Hex2Bytes(key), []byte("data"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	handler := NewHandler(globalStore, nil)

	// this subtest confirms that it has uploaded key and that it does not have invalid keys
	t.Run("has key", func(t *testing.T) {
		for addr, keys := range nodeKeys {
			for _, key := range keys {
				testStatusResponse(t, handler, "/api/has-key/"+addr+"/"+key, http.StatusOK)
				testStatusResponse(t, handler, "/api/has-key/"+invalidAddr+"/"+key, http.StatusNotFound)
			}
			testStatusResponse(t, handler, "/api/has-key/"+addr+"/"+invalidKey, http.StatusNotFound)
		}
		testStatusResponse(t, handler, "/api/has-key/"+invalidAddr+"/"+invalidKey, http.StatusNotFound)
		testStatusResponse(t, handler, "/api/has-key/"+unmatchedAddr+"/"+unmatchedKey, http.StatusNotFound)
	})

	// this subtest confirms that all keys are are listed in correct order with expected pagination
	t.Run("keys", func(t *testing.T) {
		var allKeys []string
		for key := range keyNodes {
			allKeys = append(allKeys, key)
		}
		sort.Strings(allKeys)

		t.Run("limit 0", testKeys(handler, allKeys, 0, ""))
		t.Run("limit default", testKeys(handler, allKeys, mock.DefaultLimit, ""))
		t.Run("limit 2x default", testKeys(handler, allKeys, 2*mock.DefaultLimit, ""))
		t.Run("limit 0.5x default", testKeys(handler, allKeys, mock.DefaultLimit/2, ""))
		t.Run("limit max", testKeys(handler, allKeys, mock.MaxLimit, ""))
		t.Run("limit 2x max", testKeys(handler, allKeys, 2*mock.MaxLimit, ""))
		t.Run("limit negative", testKeys(handler, allKeys, -10, ""))
	})

	// this subtest confirms that all keys are are listed for every node in correct order
	// and that for one node different pagination options are correct
	t.Run("node keys", func(t *testing.T) {
		var limitCheckAddr string

		for addr, keys := range nodeKeys {
			testKeys(handler, keys, 0, addr)(t)
			if limitCheckAddr == "" {
				limitCheckAddr = addr
			}
		}
		testKeys(handler, nil, 0, invalidAddr)(t)

		limitCheckKeys := nodeKeys[limitCheckAddr]
		t.Run("limit 0", testKeys(handler, limitCheckKeys, 0, limitCheckAddr))
		t.Run("limit default", testKeys(handler, limitCheckKeys, mock.DefaultLimit, limitCheckAddr))
		t.Run("limit 2x default", testKeys(handler, limitCheckKeys, 2*mock.DefaultLimit, limitCheckAddr))
		t.Run("limit 0.5x default", testKeys(handler, limitCheckKeys, mock.DefaultLimit/2, limitCheckAddr))
		t.Run("limit max", testKeys(handler, limitCheckKeys, mock.MaxLimit, limitCheckAddr))
		t.Run("limit 2x max", testKeys(handler, limitCheckKeys, 2*mock.MaxLimit, limitCheckAddr))
		t.Run("limit negative", testKeys(handler, limitCheckKeys, -10, limitCheckAddr))
	})

	// this subtest confirms that all nodes are are listed in correct order with expected pagination
	t.Run("nodes", func(t *testing.T) {
		var allNodes []string
		for addr := range nodeKeys {
			allNodes = append(allNodes, addr)
		}
		sortCaseInsensitive(allNodes)

		t.Run("limit 0", testNodes(handler, allNodes, 0, ""))
		t.Run("limit default", testNodes(handler, allNodes, mock.DefaultLimit, ""))
		t.Run("limit 2x default", testNodes(handler, allNodes, 2*mock.DefaultLimit, ""))
		t.Run("limit 0.5x default", testNodes(handler, allNodes, mock.DefaultLimit/2, ""))
		t.Run("limit max", testNodes(handler, allNodes, mock.MaxLimit, ""))
		t.Run("limit 2x max", testNodes(handler, allNodes, 2*mock.MaxLimit, ""))
		t.Run("limit negative", testNodes(handler, allNodes, -10, ""))
	})

	// this subtest confirms that all nodes are are listed that contain a a particular key in correct order
	// and that for one key different node pagination options are correct
	t.Run("key nodes", func(t *testing.T) {
		var limitCheckKey string

		for key, addrs := range keyNodes {
			testNodes(handler, addrs, 0, key)(t)
			if limitCheckKey == "" {
				limitCheckKey = key
			}
		}
		testNodes(handler, nil, 0, invalidKey)(t)

		limitCheckKeys := keyNodes[limitCheckKey]
		t.Run("limit 0", testNodes(handler, limitCheckKeys, 0, limitCheckKey))
		t.Run("limit default", testNodes(handler, limitCheckKeys, mock.DefaultLimit, limitCheckKey))
		t.Run("limit 2x default", testNodes(handler, limitCheckKeys, 2*mock.DefaultLimit, limitCheckKey))
		t.Run("limit 0.5x default", testNodes(handler, limitCheckKeys, mock.DefaultLimit/2, limitCheckKey))
		t.Run("limit max", testNodes(handler, limitCheckKeys, mock.MaxLimit, limitCheckKey))
		t.Run("limit 2x max", testNodes(handler, limitCheckKeys, 2*mock.MaxLimit, limitCheckKey))
		t.Run("limit negative", testNodes(handler, limitCheckKeys, -10, limitCheckKey))
	})
}

// testsKeys returns a test function that validates wantKeys against a series of /api/keys
// HTTP responses with provided limit and node options.
func testKeys(handler http.Handler, wantKeys []string, limit int, node string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()

		wantLimit := limit
		if wantLimit <= 0 {
			wantLimit = mock.DefaultLimit
		}
		if wantLimit > mock.MaxLimit {
			wantLimit = mock.MaxLimit
		}
		wantKeysLen := len(wantKeys)
		var i int
		var startKey string
		for {
			var wantNext string
			start := i * wantLimit
			end := (i + 1) * wantLimit
			if end < wantKeysLen {
				wantNext = wantKeys[end]
			} else {
				end = wantKeysLen
			}
			testKeysResponse(t, handler, node, startKey, limit, KeysResponse{
				Keys: wantKeys[start:end],
				Next: wantNext,
			})
			if wantNext == "" {
				break
			}
			startKey = wantNext
			i++
		}
	}
}

// testNodes returns a test function that validates wantAddrs against a series of /api/nodes
// HTTP responses with provided limit and key options.
func testNodes(handler http.Handler, wantAddrs []string, limit int, key string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()

		wantLimit := limit
		if wantLimit <= 0 {
			wantLimit = mock.DefaultLimit
		}
		if wantLimit > mock.MaxLimit {
			wantLimit = mock.MaxLimit
		}
		wantAddrsLen := len(wantAddrs)
		var i int
		var startKey string
		for {
			var wantNext string
			start := i * wantLimit
			end := (i + 1) * wantLimit
			if end < wantAddrsLen {
				wantNext = wantAddrs[end]
			} else {
				end = wantAddrsLen
			}
			testNodesResponse(t, handler, key, startKey, limit, NodesResponse{
				Nodes: wantAddrs[start:end],
				Next:  wantNext,
			})
			if wantNext == "" {
				break
			}
			startKey = wantNext
			i++
		}
	}
}

// testStatusResponse validates a response made on url if it matches
// the expected StatusResponse.
func testStatusResponse(t *testing.T, handler http.Handler, url string, code int) {
	t.Helper()

	resp := httpGet(t, handler, url)

	if resp.StatusCode != code {
		t.Errorf("got status code %v, want %v", resp.StatusCode, code)
	}
	if got := resp.Header.Get("Content-Type"); got != jsonContentType {
		t.Errorf("got Content-Type header %q, want %q", got, jsonContentType)
	}
	var r StatusResponse
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

// testKeysResponse validates response returned from handler on /api/keys
// with node, start and limit options against KeysResponse.
func testKeysResponse(t *testing.T, handler http.Handler, node, start string, limit int, want KeysResponse) {
	t.Helper()

	u, err := url.Parse("/api/keys")
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if node != "" {
		q.Set("node", node)
	}
	if start != "" {
		q.Set("start", start)
	}
	if limit != 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	resp := httpGet(t, handler, u.String())

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status code %v, want %v", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != jsonContentType {
		t.Errorf("got Content-Type header %q, want %q", got, jsonContentType)
	}
	var r KeysResponse
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

// testNodesResponse validates response returned from handler on /api/nodes
// with key, start and limit options against NodesResponse.
func testNodesResponse(t *testing.T, handler http.Handler, key, start string, limit int, want NodesResponse) {
	t.Helper()

	u, err := url.Parse("/api/nodes")
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if key != "" {
		q.Set("key", key)
	}
	if start != "" {
		q.Set("start", start)
	}
	if limit != 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	resp := httpGet(t, handler, u.String())

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status code %v, want %v", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); got != jsonContentType {
		t.Errorf("got Content-Type header %q, want %q", got, jsonContentType)
	}
	var r NodesResponse
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

// httpGet uses httptest recorder to provide a response on handler's url.
func httpGet(t *testing.T, handler http.Handler, url string) (r *http.Response) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

// sortCaseInsensitive performs a case insensitive sort on a string slice.
func sortCaseInsensitive(s []string) {
	sort.Slice(s, func(i, j int) bool {
		return strings.ToLower(s[i]) < strings.ToLower(s[j])
	})
}
