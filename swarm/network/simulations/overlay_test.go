// Copyright 2018 The go-ethereum Authors
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
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/log"
)

var (
	nodeCount = 16
)

//This test is used to test the overlay simulation.
//As the simulation is executed via a main, it is easily missed on changes
//An automated test will prevent that
//The test just connects to the simulations, starts the network,
//starts the mocker, gets the number of nodes, and stops it again.
//It also provides a documentation on the steps needed by frontends
//to use the simulations
func TestOverlaySim(t *testing.T) {
	t.Skip("Test is flaky, see: https://github.com/ethersphere/go-ethereum/issues/592")
	//start the simulation
	log.Info("Start simulation backend")
	//get the simulation networ; needed to subscribe for up events
	net := newSimulationNetwork()
	//create the overlay simulation
	sim := newOverlaySim(net)
	//create a http test server with it
	srv := httptest.NewServer(sim)
	defer srv.Close()

	log.Debug("Http simulation server started. Start simulation network")
	//start the simulation network (initialization of simulation)
	resp, err := http.Post(srv.URL+"/start", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected Status Code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	log.Debug("Start mocker")
	//start the mocker, needs a node count and an ID
	resp, err = http.PostForm(srv.URL+"/mocker/start",
		url.Values{
			"node-count":  {fmt.Sprintf("%d", nodeCount)},
			"mocker-type": {simulations.GetMockerList()[0]},
		})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		reason, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("Expected Status Code %d, got %d, response body %s", http.StatusOK, resp.StatusCode, string(reason))
	}

	//variables needed to wait for nodes being up
	var upCount int
	trigger := make(chan enode.ID)

	//wait for all nodes to be up
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//start watching node up events...
	go watchSimEvents(net, ctx, trigger)

	//...and wait until all expected up events (nodeCount) have been received
LOOP:
	for {
		select {
		case <-trigger:
			//new node up event received, increase counter
			upCount++
			//all expected node up events received
			if upCount == nodeCount {
				break LOOP
			}
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for up events")
		}

	}

	//at this point we can query the server
	log.Info("Get number of nodes")
	//get the number of nodes
	resp, err = http.Get(srv.URL + "/nodes")
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	//unmarshal number of nodes from JSON response
	var nodesArr []simulations.Node
	err = json.Unmarshal(b, &nodesArr)
	if err != nil {
		t.Fatal(err)
	}

	//check if number of nodes received is same as sent
	if len(nodesArr) != nodeCount {
		t.Fatal(fmt.Errorf("Expected %d number of nodes, got %d", nodeCount, len(nodesArr)))
	}

	//need to let it run for a little while, otherwise stopping it immediately can crash due running nodes
	//wanting to connect to already stopped nodes
	time.Sleep(1 * time.Second)

	log.Info("Stop the network")
	//stop the network
	resp, err = http.Post(srv.URL+"/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}

	log.Info("Reset the network")
	//reset the network (removes all nodes and connections)
	resp, err = http.Post(srv.URL+"/reset", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("err %s", resp.Status)
	}
}

//watch for events so we know when all nodes are up
func watchSimEvents(net *simulations.Network, ctx context.Context, trigger chan enode.ID) {
	events := make(chan *simulations.Event)
	sub := net.Events().Subscribe(events)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-events:
			//only catch node up events
			if ev.Type == simulations.EventTypeNode {
				if ev.Node.Up {
					log.Debug("got node up event", "event", ev, "node", ev.Node.Config.ID)
					select {
					case trigger <- ev.Node.Config.ID:
					case <-ctx.Done():
						return
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
