// Copyright 2017 The go-ethereum Authors
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

// Package simulations simulates p2p networks.
// A NodeRunner simulates starting and stopping real nodes in a network.
package simulations

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/julienschmidt/httprouter"
)

//A NodeRunner only has a Run function
type NodeRunner interface {
	Run()
}

//used to stop the NodeRunner loop
var quit chan struct{}

//nodeRunner holds properties needed
//the idea is that we create specific noderunner instances
//with their own run loop, with each one implementing Run()
type nodeRunner struct {
	router    *httprouter.Router
	nodes     []discover.NodeID
	network   *Network
	nodeCount int
}

//startStopRunner: just starts and stops nodes periodically
type startStopRunner struct {
	nodeRunner
}

//probabilisticRunner: starts and stops nodes in a more probabilistic pattern
type probabilisticRunner struct {
	nodeRunner
}

//bootRunner: only boots up all nodes
type bootRunner struct {
	nodeRunner
}

//create a new runner
func NewNodeRunner(network *Network, runnerId string, nodeCount int) {
	var runner NodeRunner
	//init properties
	nr := nodeRunner{
		nodeCount: nodeCount,
		network:   network,
	}
	//create stop channel
	quit = make(chan struct{}, 1)
	//instantiate specific runner
	switch runnerId {
	case "startStop":
		runner = &startStopRunner{nr}
	case "probabilistic":
		runner = &probabilisticRunner{nr}
	case "boot":
		runner = &bootRunner{nr}
	default:
		runner = nil
	}
	if runner == nil {
		panic("No runner assigned")
	}
	//setup HHTP Routes
	nr.setupRoutes(runner)
	//start HTTP endpoint
	http.ListenAndServe(":8889", nil)
}

//setup HTTP endpoints for a runner
//available routes are:
//* `runSim`: run a node simulation
//* `stopSim`: stop the node simulation
func (r *nodeRunner) setupRoutes(runner NodeRunner) {
	//runSim
	http.HandleFunc("/runSim", func(w http.ResponseWriter, req *http.Request) {
		log.Info("Starting node simulation...")
		go runner.Run()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
	})
	//stopSim
	http.HandleFunc("/stopSim", func(w http.ResponseWriter, req *http.Request) {
		log.Info("Stopping node simulation...")
		r.stopSim()
		r.network.StopAll()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
	})
}

//stopping the node simulation is just sending an empty signal to the channel
func (r nodeRunner) stopSim() {
	quit <- struct{}{}
}

//the bootRunner only starts up all nodes and doesn't do anything else
func (r *bootRunner) Run() {
	r.nodeRunner.connectNodesInRing()
}

//the startStopRunner first boots all nodes in a ring,
//then starts and stops a (randomly selected) node in a periodic interval
func (r *startStopRunner) Run() {
	nodes := r.nodeRunner.connectNodesInRing()
	net := r.nodeRunner.network
	for range time.Tick(10 * time.Second) {
		select {
		case <-quit:
			log.Info("Terminating simulation loop")
			return
		default:
		}
		id := nodes[rand.Intn(len(nodes))]
		go func() {
			log.Info("stopping node", "id", id)
			if err := net.Stop(id); err != nil {
				log.Error("error stopping node", "id", id, "err", err)
				return
			}

			time.Sleep(3 * time.Second)

			log.Debug("starting node", "id", id)
			if err := net.Start(id); err != nil {
				log.Error("error starting node", "id", id, "err", err)
				return
			}
		}()
	}
}

//the probabilisticRunner has a more probabilistic pattern (can be improved of course):
//nodes are connected in a ring, then selects a varying number of random nodes
//stops and starts them in random intervals, and continues the loop
func (r *probabilisticRunner) Run() {
	nodes := r.nodeRunner.connectNodesInRing()
	net := r.nodeRunner.network
	for {
		select {
		case <-quit:
			log.Info("Terminating simulation loop")
			return
		default:
		}
		var lowid, highid int
		var wg sync.WaitGroup
		randWait := rand.Intn(5000) + 1000
		rand1 := rand.Intn(9)
		rand2 := rand.Intn(9)
		if rand1 < rand2 {
			lowid = rand1
			highid = rand2
		} else if rand1 > rand2 {
			highid = rand1
			lowid = rand2
		} else {
			if rand1 == 0 {
				rand2 = 9
			} else if rand1 == 9 {
				rand1 = 0
			}
			lowid = rand1
			highid = rand2
		}
		var steps = highid - lowid
		wg.Add(steps)
		for i := lowid; i < highid; i++ {
			select {
			case <-quit:
				log.Info("Terminating simulation loop")
				return
			default:
			}
			log.Debug(fmt.Sprintf("node %v shutting down", nodes[i]))
			net.Stop(nodes[i])
			go func(id discover.NodeID) {
				time.Sleep(time.Duration(randWait) * time.Millisecond)
				net.Start(id)
				wg.Done()
			}(nodes[i])
			time.Sleep(time.Duration(randWait) * time.Millisecond)
		}
		wg.Wait()
	}

}

//connect nodeCount number of  nodes in a ring
func (r *nodeRunner) connectNodesInRing() []discover.NodeID {
	ids := make([]discover.NodeID, r.nodeCount)
	net := r.network
	for i := 0; i < r.nodeCount; i++ {
		node, err := net.NewNode()
		if err != nil {
			panic(err.Error())
		}
		ids[i] = node.ID()
	}

	for _, id := range ids {
		if err := net.Start(id); err != nil {
			panic(err.Error())
		}
		log.Debug(fmt.Sprintf("node %v starting up", id))
	}
	for i, id := range ids {
		var peerID discover.NodeID
		//first node connects with last one
		if i == 0 {
			peerID = ids[len(ids)-1]
		} else {
			//every other one connects with previous
			peerID = ids[i-1]
		}
		if err := net.Connect(id, peerID); err != nil {
			panic(err.Error())
		}
	}

	return ids
}
