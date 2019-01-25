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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
)

func TestUpDownNodeIDs(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	gotIDs := sim.NodeIDs()

	if !equalNodeIDs(ids, gotIDs) {
		t.Error("returned nodes are not equal to added ones")
	}

	stoppedIDs, err := sim.StopRandomNodes(3)
	if err != nil {
		t.Fatal(err)
	}

	gotIDs = sim.UpNodeIDs()

	for _, id := range gotIDs {
		if !sim.Net.GetNode(id).Up {
			t.Errorf("node %s should not be down", id)
		}
	}

	if !equalNodeIDs(ids, append(gotIDs, stoppedIDs...)) {
		t.Error("returned nodes are not equal to added ones")
	}

	gotIDs = sim.DownNodeIDs()

	for _, id := range gotIDs {
		if sim.Net.GetNode(id).Up {
			t.Errorf("node %s should not be up", id)
		}
	}

	if !equalNodeIDs(stoppedIDs, gotIDs) {
		t.Error("returned nodes are not equal to the stopped ones")
	}
}

func equalNodeIDs(one, other []enode.ID) bool {
	if len(one) != len(other) {
		return false
	}
	var count int
	for _, a := range one {
		var found bool
		for _, b := range other {
			if a == b {
				found = true
				break
			}
		}
		if found {
			count++
		} else {
			return false
		}
	}
	return count == len(one)
}

func TestAddNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	id, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	n := sim.Net.GetNode(id)
	if n == nil {
		t.Fatal("node not found")
	}

	if !n.Up {
		t.Error("node not started")
	}
}

func TestAddNodeWithMsgEvents(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	id, err := sim.AddNode(AddNodeWithMsgEvents(true))
	if err != nil {
		t.Fatal(err)
	}

	if !sim.Net.GetNode(id).Config.EnableMsgEvents {
		t.Error("EnableMsgEvents is false")
	}

	id, err = sim.AddNode(AddNodeWithMsgEvents(false))
	if err != nil {
		t.Fatal(err)
	}

	if sim.Net.GetNode(id).Config.EnableMsgEvents {
		t.Error("EnableMsgEvents is true")
	}
}

func TestAddNodeWithService(t *testing.T) {
	sim := New(map[string]ServiceFunc{
		"noop1": noopServiceFunc,
		"noop2": noopServiceFunc,
	})
	defer sim.Close()

	id, err := sim.AddNode(AddNodeWithService("noop1"))
	if err != nil {
		t.Fatal(err)
	}

	n := sim.Net.GetNode(id).Node.(*adapters.SimNode)
	if n.Service("noop1") == nil {
		t.Error("service noop1 not found on node")
	}
	if n.Service("noop2") != nil {
		t.Error("service noop2 should not be found on node")
	}
}

func TestAddNodeMultipleServices(t *testing.T) {
	sim := New(map[string]ServiceFunc{
		"noop1": noopServiceFunc,
		"noop2": noopService2Func,
	})
	defer sim.Close()

	id, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	n := sim.Net.GetNode(id).Node.(*adapters.SimNode)
	if n.Service("noop1") == nil {
		t.Error("service noop1 not found on node")
	}
	if n.Service("noop2") == nil {
		t.Error("service noop2 not found on node")
	}
}

func TestAddNodeDuplicateServiceError(t *testing.T) {
	sim := New(map[string]ServiceFunc{
		"noop1": noopServiceFunc,
		"noop2": noopServiceFunc,
	})
	defer sim.Close()

	wantErr := "duplicate service: *simulation.noopService"
	_, err := sim.AddNode()
	if err.Error() != wantErr {
		t.Errorf("got error %q, want %q", err, wantErr)
	}
}

func TestAddNodes(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	nodesCount := 12

	ids, err := sim.AddNodes(nodesCount)
	if err != nil {
		t.Fatal(err)
	}

	count := len(ids)
	if count != nodesCount {
		t.Errorf("expected %v nodes, got %v", nodesCount, count)
	}

	count = len(sim.Net.GetNodes())
	if count != nodesCount {
		t.Errorf("expected %v nodes, got %v", nodesCount, count)
	}
}

func TestAddNodesAndConnectFull(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	n := 12

	ids, err := sim.AddNodesAndConnectFull(n)
	if err != nil {
		t.Fatal(err)
	}

	simulations.VerifyFull(t, sim.Net, ids)
}

func TestAddNodesAndConnectChain(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodesAndConnectChain(12)
	if err != nil {
		t.Fatal(err)
	}

	// add another set of nodes to test
	// if two chains are connected
	_, err = sim.AddNodesAndConnectChain(7)
	if err != nil {
		t.Fatal(err)
	}

	simulations.VerifyChain(t, sim.Net, sim.UpNodeIDs())
}

func TestAddNodesAndConnectRing(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodesAndConnectRing(12)
	if err != nil {
		t.Fatal(err)
	}

	simulations.VerifyRing(t, sim.Net, ids)
}

func TestAddNodesAndConnectStar(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodesAndConnectStar(12)
	if err != nil {
		t.Fatal(err)
	}

	simulations.VerifyStar(t, sim.Net, ids, 0)
}

//To test that uploading a snapshot works
func TestUploadSnapshot(t *testing.T) {
	log.Debug("Creating simulation")
	s := New(map[string]ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			addr := network.NewAddr(ctx.Config.Node())
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			return network.NewBzz(config, kad, nil, nil, nil), nil, nil
		},
	})
	defer s.Close()

	nodeCount := 16
	log.Debug("Uploading snapshot")
	err := s.UploadSnapshot(fmt.Sprintf("../stream/testing/snapshot_%d.json", nodeCount))
	if err != nil {
		t.Fatalf("Error uploading snapshot to simulation network: %v", err)
	}

	ctx := context.Background()
	log.Debug("Starting simulation...")
	s.Run(ctx, func(ctx context.Context, sim *Simulation) error {
		log.Debug("Checking")
		nodes := sim.UpNodeIDs()
		if len(nodes) != nodeCount {
			t.Fatal("Simulation network node number doesn't match snapshot node number")
		}
		return nil
	})
	log.Debug("Done.")
}

func TestStartStopNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	id, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	n := sim.Net.GetNode(id)
	if n == nil {
		t.Fatal("node not found")
	}
	if !n.Up {
		t.Error("node not started")
	}

	err = sim.StopNode(id)
	if err != nil {
		t.Fatal(err)
	}
	if n.Up {
		t.Error("node not stopped")
	}

	// Sleep here to ensure that Network.watchPeerEvents defer function
	// has set the `node.Up = false` before we start the node again.
	// p2p/simulations/network.go:215
	//
	// The same node is stopped and started again, and upon start
	// watchPeerEvents is started in a goroutine. If the node is stopped
	// and then very quickly started, that goroutine may be scheduled later
	// then start and force `node.Up = false` in its defer function.
	// This will make this test unreliable.
	time.Sleep(time.Second)

	err = sim.StartNode(id)
	if err != nil {
		t.Fatal(err)
	}
	if !n.Up {
		t.Error("node not started")
	}
}

func TestStartStopRandomNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(3)
	if err != nil {
		t.Fatal(err)
	}

	id, err := sim.StopRandomNode()
	if err != nil {
		t.Fatal(err)
	}

	n := sim.Net.GetNode(id)
	if n == nil {
		t.Fatal("node not found")
	}
	if n.Up {
		t.Error("node not stopped")
	}

	id2, err := sim.StopRandomNode()
	if err != nil {
		t.Fatal(err)
	}

	// Sleep here to ensure that Network.watchPeerEvents defer function
	// has set the `node.Up = false` before we start the node again.
	// p2p/simulations/network.go:215
	//
	// The same node is stopped and started again, and upon start
	// watchPeerEvents is started in a goroutine. If the node is stopped
	// and then very quickly started, that goroutine may be scheduled later
	// then start and force `node.Up = false` in its defer function.
	// This will make this test unreliable.
	time.Sleep(time.Second)

	idStarted, err := sim.StartRandomNode()
	if err != nil {
		t.Fatal(err)
	}

	if idStarted != id && idStarted != id2 {
		t.Error("unexpected started node ID")
	}
}

func TestStartStopRandomNodes(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := sim.StopRandomNodes(3)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range ids {
		n := sim.Net.GetNode(id)
		if n == nil {
			t.Fatal("node not found")
		}
		if n.Up {
			t.Error("node not stopped")
		}
	}

	// Sleep here to ensure that Network.watchPeerEvents defer function
	// has set the `node.Up = false` before we start the node again.
	// p2p/simulations/network.go:215
	//
	// The same node is stopped and started again, and upon start
	// watchPeerEvents is started in a goroutine. If the node is stopped
	// and then very quickly started, that goroutine may be scheduled later
	// then start and force `node.Up = false` in its defer function.
	// This will make this test unreliable.
	time.Sleep(time.Second)

	ids, err = sim.StartRandomNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range ids {
		n := sim.Net.GetNode(id)
		if n == nil {
			t.Fatal("node not found")
		}
		if !n.Up {
			t.Error("node not started")
		}
	}
}
