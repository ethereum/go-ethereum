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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
	"github.com/ethereum/go-ethereum/swarm/testutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	swarmhttp "github.com/ethereum/go-ethereum/swarm/api/http"
)

func TestCLIFeedUpdate(t *testing.T) {

	srv := testutil.NewTestSwarmServer(t, func(api *api.API) testutil.TestServer {
		return swarmhttp.NewServer(api, "")
	}, nil)
	log.Info("starting a test swarm server")
	defer srv.Close()

	// create a private key file for signing
	pkfile, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer pkfile.Close()
	defer os.Remove(pkfile.Name())

	privkeyHex := "0000000000000000000000000000000000000000000000000000000000001979"
	privKey, _ := crypto.HexToECDSA(privkeyHex)
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	// save the private key to a file
	_, err = io.WriteString(pkfile, privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	// compose a topic. We'll be doing quotes about Miguel de Cervantes
	var topic feed.Topic
	subject := []byte("Miguel de Cervantes")
	copy(topic[:], subject[:])
	name := "quotes"

	// prepare some data for the update
	data := []byte("En boca cerrada no entran moscas")
	hexData := hexutil.Encode(data)

	flags := []string{
		"--bzzapi", srv.URL,
		"--bzzaccount", pkfile.Name(),
		"feed", "update",
		"--topic", topic.Hex(),
		"--name", name,
		hexData}

	// create an update and expect an exit without errors
	log.Info(fmt.Sprintf("updating a feed with 'swarm feed update'"))
	cmd := runSwarm(t, flags...)
	cmd.ExpectExit()

	// now try to get the update using the client
	client := swarm.NewClient(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// build the same topic as before, this time
	// we use NewTopic to create a topic automatically.
	topic, err = feed.NewTopic(name, subject)
	if err != nil {
		t.Fatal(err)
	}

	// Feed configures whose updates we will be looking up.
	fd := feed.Feed{
		Topic: topic,
		User:  address,
	}

	// Build a query to get the latest update
	query := feed.NewQueryLatest(&fd, lookup.NoClue)

	// retrieve content!
	reader, err := client.QueryFeed(query, "")
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	// check we retrieved the sent information
	if !bytes.Equal(data, retrieved) {
		t.Fatalf("Received %s, expected %s", retrieved, data)
	}

	// Now retrieve info for the next update
	flags = []string{
		"--bzzapi", srv.URL,
		"feed", "info",
		"--topic", topic.Hex(),
		"--user", address.Hex(),
	}

	log.Info(fmt.Sprintf("getting feed info with 'swarm feed info'"))
	cmd = runSwarm(t, flags...)
	_, matches := cmd.ExpectRegexp(`.*`) // regex hack to extract stdout
	cmd.ExpectExit()

	// verify we can deserialize the result as a valid JSON
	var request feed.Request
	err = json.Unmarshal([]byte(matches[0]), &request)
	if err != nil {
		t.Fatal(err)
	}

	// make sure the retrieved feed is the same
	if request.Feed != fd {
		t.Fatalf("Expected feed to be: %s, got %s", fd, request.Feed)
	}

	// test publishing a manifest
	flags = []string{
		"--bzzapi", srv.URL,
		"--bzzaccount", pkfile.Name(),
		"feed", "create",
		"--topic", topic.Hex(),
	}

	log.Info(fmt.Sprintf("Publishing manifest with 'swarm feed create'"))
	cmd = runSwarm(t, flags...)
	_, matches = cmd.ExpectRegexp(`[a-f\d]{64}`) // regex hack to extract stdout
	cmd.ExpectExit()

	manifestAddress := matches[0] // read the received feed manifest

	// now attempt to lookup the latest update using a manifest instead
	reader, err = client.QueryFeed(nil, manifestAddress)
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err = ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, retrieved) {
		t.Fatalf("Received %s, expected %s", retrieved, data)
	}
}
