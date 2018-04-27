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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

// TestCLISwarmUp tests that running 'swarm up' makes the resulting file
// available from all nodes via the HTTP API
func TestCLISwarmUp(t *testing.T) {
	testCLISwarmUp(false, t)
}

// TestCLISwarmUpEncrypted tests that running 'swarm encrypted-up' makes the resulting file
// available from all nodes via the HTTP API
func TestCLISwarmUpEncrypted(t *testing.T) {
	testCLISwarmUp(true, t)
}

func testCLISwarmUp(toEncrypt bool, t *testing.T) {
	log.Info("starting 3 node cluster")
	cluster := newTestCluster(t, 3)
	defer cluster.Shutdown()

	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	// write data to file
	data := "randomdata"
	_, err = io.WriteString(tmp, data)
	if err != nil {
		t.Fatal(err)
	}

	hashRegexp := `[a-f\d]{64}`
	flags := []string{
		"--bzzapi", cluster.Nodes[0].URL,
		"up",
		tmp.Name()}
	if toEncrypt {
		hashRegexp = `[a-f\d]{128}`
		flags = []string{
			"--bzzapi", cluster.Nodes[0].URL,
			"up",
			"--encrypted",
			tmp.Name()}
	}
	// upload the file with 'swarm up' and expect a hash
	log.Info(fmt.Sprintf("uploading file with 'swarm up'"))
	up := runSwarm(t, flags...)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()
	hash := matches[0]
	log.Info("file uploaded", "hash", hash)

	// get the file from the HTTP API of each node
	for _, node := range cluster.Nodes {
		log.Info("getting file from node", "node", node.Name)

		res, err := http.Get(node.URL + "/bzz:/" + hash)
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != 200 {
			t.Fatalf("expected HTTP status 200, got %s", res.Status)
		}

		reply, err := ioutil.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(reply) != data {
			t.Fatalf("expected HTTP body %q, got %q", data, reply)
		}
	}

	// get an non-existent hash from each node
	for _, node := range cluster.Nodes {
		res, err := http.Get(node.URL + "/bzz:/1023e8bae0f70be7d7b5f74343088ba408a218254391490c85ae16278e230340")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 404 {
			t.Fatalf("expected HTTP status 404, got %s", res.Status)
		}
	}
}
