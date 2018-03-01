// Copyright 2017 The go-ethereum Authors
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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

// TestCLISwarmUp tests that running 'swarm up' makes the resulting file
// available from all nodes via the HTTP API
func TestCLISwarmUp(t *testing.T) {
	// start 3 node cluster
	t.Log("starting 3 node cluster")
	cluster := newTestCluster(t, 3)
	defer cluster.Shutdown()

	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	assertNil(t, err)
	defer tmp.Close()
	defer os.Remove(tmp.Name())
	_, err = io.WriteString(tmp, "data")
	assertNil(t, err)

	// upload the file with 'swarm up' and expect a hash
	t.Log("uploading file with 'swarm up'")
	up := runSwarm(t, "--bzzapi", cluster.Nodes[0].URL, "up", tmp.Name())
	_, matches := up.ExpectRegexp(`[a-f\d]{64}`)
	up.ExpectExit()
	hash := matches[0]
	t.Logf("file uploaded with hash %s", hash)

	// get the file from the HTTP API of each node
	for _, node := range cluster.Nodes {
		t.Logf("getting file from %s", node.Name)
		res, err := http.Get(node.URL + "/bzz:/" + hash)
		assertNil(t, err)
		assertHTTPResponse(t, res, http.StatusOK, "data")
	}
}

func assertNil(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func assertHTTPResponse(t *testing.T, res *http.Response, expectedStatus int, expectedBody string) {
	defer res.Body.Close()
	if res.StatusCode != expectedStatus {
		t.Fatalf("expected HTTP status %d, got %s", expectedStatus, res.Status)
	}
	data, err := ioutil.ReadAll(res.Body)
	assertNil(t, err)
	if string(data) != expectedBody {
		t.Fatalf("expected HTTP body %q, got %q", expectedBody, data)
	}
}
