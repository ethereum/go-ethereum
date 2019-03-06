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

package simulation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
)

/*
	TestWaitTillHealthy tests that we indeed get a healthy network after we wait for it.
	For this to be tested, a bit of a snake tail bite needs to happen:
		* First we create a first simulation
		* Run it as nodes connected in a ring
		* Wait until the network is healthy
		* Then we create a snapshot
		* With this snapshot we create a new simulation
		* This simulation is expected to have a healthy configuration, as it uses the snapshot
		* Thus we just iterate all nodes and check that their kademlias are healthy
		* If all kademlias are healthy, the test succeeded, otherwise it failed
*/
func TestWaitTillHealthy(t *testing.T) {

	testNodesNum := 10

	// create the first simulation
	sim := New(createSimServiceMap(true))

	// connect and...
	nodeIDs, err := sim.AddNodesAndConnectRing(testNodesNum)
	if err != nil {
		t.Fatal(err)
	}

	// array of all overlay addresses
	var addrs [][]byte
	// iterate once to be able to build the peer map
	for _, node := range nodeIDs {
		//get the kademlia overlay address from this ID
		a := node.Bytes()
		//append it to the array of all overlay addresses
		addrs = append(addrs, a)
	}
	// build a PeerPot only once
	pp := network.NewPeerPotMap(network.NewKadParams().NeighbourhoodSize, addrs)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// ...wait until healthy
	ill, err := sim.WaitTillHealthy(ctx)
	if err != nil {
		for id, kad := range ill {
			t.Log("Node", id)
			t.Log(kad.String())
		}
		t.Fatal(err)
	}

	// now create a snapshot of this network
	snap, err := sim.Net.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	// close the initial simulation
	sim.Close()
	// create a control simulation
	controlSim := New(createSimServiceMap(false))
	defer controlSim.Close()

	// load the snapshot into this control simulation
	err = controlSim.Net.Load(snap)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controlSim.WaitTillHealthy(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, node := range nodeIDs {
		// ...get its kademlia
		item, ok := controlSim.NodeItem(node, BucketKeyKademlia)
		if !ok {
			t.Fatal("No kademlia bucket item")
		}
		kad := item.(*network.Kademlia)
		// get its base address
		kid := common.Bytes2Hex(kad.BaseAddr())

		//get the health info
		info := kad.GetHealthInfo(pp[kid])
		log.Trace("Health info", "info", info)
		// check that it is healthy
		healthy := info.Healthy()
		if !healthy {
			t.Fatalf("Expected node %v of control simulation to be healthy, but it is not, unhealthy kademlias: %v", node, kad.String())
		}
	}
}

// createSimServiceMap returns the services map
// this function will create the sim services with or without discovery enabled
// based on the flag passed
func createSimServiceMap(discovery bool) map[string]ServiceFunc {
	return map[string]ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			addr := network.NewAddr(ctx.Config.Node())
			hp := network.NewHiveParams()
			hp.Discovery = discovery
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			// store kademlia in node's bucket under BucketKeyKademlia
			// so that it can be found by WaitTillHealthy method.
			b.Store(BucketKeyKademlia, kad)
			return network.NewBzz(config, kad, nil, nil, nil), nil, nil
		},
	}
}
