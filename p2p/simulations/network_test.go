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

package simulations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// Tests that a created snapshot with a minimal service only contains the expected connections
// and that a network when loaded with this snapshot only contains those same connections
func TestSnapshot(t *testing.T) {

	// PART I
	// create snapshot from ring network

	// this is a minimal service, whose protocol will take exactly one message OR close of connection before quitting
	adapter := adapters.NewSimAdapter(adapters.Services{
		"noopwoop": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return NewNoopService(nil), nil
		},
	})

	// create network
	network := NewNetwork(adapter, &NetworkConfig{
		DefaultService: "noopwoop",
	})
	// \todo consider making a member of network, set to true threadsafe when shutdown
	runningOne := true
	defer func() {
		if runningOne {
			network.Shutdown()
		}
	}()

	// create and start nodes
	nodeCount := 20
	ids := make([]enode.ID, nodeCount)
	for i := 0; i < nodeCount; i++ {
		conf := adapters.RandomNodeConfig()
		node, err := network.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		if err := network.Start(node.ID()); err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		ids[i] = node.ID()
	}

	// subscribe to peer events
	evC := make(chan *Event)
	sub := network.Events().Subscribe(evC)
	defer sub.Unsubscribe()

	// connect nodes in a ring
	// spawn separate thread to avoid deadlock in the event listeners
	go func() {
		for i, id := range ids {
			peerID := ids[(i+1)%len(ids)]
			if err := network.Connect(id, peerID); err != nil {
				t.Fatal(err)
			}
		}
	}()

	// collect connection events up to expected number
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	checkIds := make(map[enode.ID][]enode.ID)
	connEventCount := nodeCount
OUTER:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case ev := <-evC:
			if ev.Type == EventTypeConn && !ev.Control {

				// fail on any disconnect
				if !ev.Conn.Up {
					t.Fatalf("unexpected disconnect: %v -> %v", ev.Conn.One, ev.Conn.Other)
				}
				checkIds[ev.Conn.One] = append(checkIds[ev.Conn.One], ev.Conn.Other)
				checkIds[ev.Conn.Other] = append(checkIds[ev.Conn.Other], ev.Conn.One)
				connEventCount--
				log.Debug("ev", "count", connEventCount)
				if connEventCount == 0 {
					break OUTER
				}
			}
		}
	}

	// create snapshot of current network
	snap, err := network.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	j, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("snapshot taken", "nodes", len(snap.Nodes), "conns", len(snap.Conns), "json", string(j))

	// verify that the snap element numbers check out
	if len(checkIds) != len(snap.Conns) || len(checkIds) != len(snap.Nodes) {
		t.Fatalf("snapshot wrong node,conn counts %d,%d != %d", len(snap.Nodes), len(snap.Conns), len(checkIds))
	}

	// shut down sim network
	runningOne = false
	sub.Unsubscribe()
	network.Shutdown()

	// check that we have all the expected connections in the snapshot
	for nodid, nodConns := range checkIds {
		for _, nodConn := range nodConns {
			var match bool
			for _, snapConn := range snap.Conns {
				if snapConn.One == nodid && snapConn.Other == nodConn {
					match = true
					break
				} else if snapConn.Other == nodid && snapConn.One == nodConn {
					match = true
					break
				}
			}
			if !match {
				t.Fatalf("snapshot missing conn %v -> %v", nodid, nodConn)
			}
		}
	}
	log.Info("snapshot checked")

	// PART II
	// load snapshot and verify that exactly same connections are formed

	adapter = adapters.NewSimAdapter(adapters.Services{
		"noopwoop": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return NewNoopService(nil), nil
		},
	})
	network = NewNetwork(adapter, &NetworkConfig{
		DefaultService: "noopwoop",
	})
	defer func() {
		network.Shutdown()
	}()

	// subscribe to peer events
	// every node up and conn up event will generate one additional control event
	// therefore multiply the count by two
	evC = make(chan *Event, (len(snap.Conns)*2)+(len(snap.Nodes)*2))
	sub = network.Events().Subscribe(evC)
	defer sub.Unsubscribe()

	// load the snapshot
	// spawn separate thread to avoid deadlock in the event listeners
	err = network.Load(snap)
	if err != nil {
		t.Fatal(err)
	}

	// collect connection events up to expected number
	ctx, cancel = context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	connEventCount = nodeCount

OuterTwo:
	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case ev := <-evC:
			if ev.Type == EventTypeConn && !ev.Control {

				// fail on any disconnect
				if !ev.Conn.Up {
					t.Fatalf("unexpected disconnect: %v -> %v", ev.Conn.One, ev.Conn.Other)
				}
				log.Debug("conn", "on", ev.Conn.One, "other", ev.Conn.Other)
				checkIds[ev.Conn.One] = append(checkIds[ev.Conn.One], ev.Conn.Other)
				checkIds[ev.Conn.Other] = append(checkIds[ev.Conn.Other], ev.Conn.One)
				connEventCount--
				log.Debug("ev", "count", connEventCount)
				if connEventCount == 0 {
					break OuterTwo
				}
			}
		}
	}

	// check that we have all expected connections in the network
	for _, snapConn := range snap.Conns {
		var match bool
		for nodid, nodConns := range checkIds {
			for _, nodConn := range nodConns {
				if snapConn.One == nodid && snapConn.Other == nodConn {
					match = true
					break
				} else if snapConn.Other == nodid && snapConn.One == nodConn {
					match = true
					break
				}
			}
		}
		if !match {
			t.Fatalf("network missing conn %v -> %v", snapConn.One, snapConn.Other)
		}
	}

	// verify that network didn't generate any other additional connection events after the ones we have collected within a reasonable period of time
	ctx, cancel = context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
	case ev := <-evC:
		if ev.Type == EventTypeConn {
			t.Fatalf("Superfluous conn found %v -> %v", ev.Conn.One, ev.Conn.Other)
		}
	}

	// This test validates if all connections from the snapshot
	// are created in the network.
	t.Run("conns after load", func(t *testing.T) {
		// Create new network.
		n := NewNetwork(
			adapters.NewSimAdapter(adapters.Services{
				"noopwoop": func(ctx *adapters.ServiceContext) (node.Service, error) {
					return NewNoopService(nil), nil
				},
			}),
			&NetworkConfig{
				DefaultService: "noopwoop",
			},
		)
		defer n.Shutdown()

		// Load the same snapshot.
		err := n.Load(snap)
		if err != nil {
			t.Fatal(err)
		}

		// Check every connection from the snapshot
		// if it is in the network, too.
		for _, c := range snap.Conns {
			if n.GetConn(c.One, c.Other) == nil {
				t.Errorf("missing connection: %s -> %s", c.One, c.Other)
			}
		}
	})
}

// TestNetworkSimulation creates a multi-node simulation network with each node
// connected in a ring topology, checks that all nodes successfully handshake
// with each other and that a snapshot fully represents the desired topology
func TestNetworkSimulation(t *testing.T) {
	// create simulation network with 20 testService nodes
	adapter := adapters.NewSimAdapter(adapters.Services{
		"test": newTestService,
	})
	network := NewNetwork(adapter, &NetworkConfig{
		DefaultService: "test",
	})
	defer network.Shutdown()
	nodeCount := 20
	ids := make([]enode.ID, nodeCount)
	for i := 0; i < nodeCount; i++ {
		conf := adapters.RandomNodeConfig()
		node, err := network.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		if err := network.Start(node.ID()); err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		ids[i] = node.ID()
	}

	// perform a check which connects the nodes in a ring (so each node is
	// connected to exactly two peers) and then checks that all nodes
	// performed two handshakes by checking their peerCount
	action := func(_ context.Context) error {
		for i, id := range ids {
			peerID := ids[(i+1)%len(ids)]
			if err := network.Connect(id, peerID); err != nil {
				return err
			}
		}
		return nil
	}
	check := func(ctx context.Context, id enode.ID) (bool, error) {
		// check we haven't run out of time
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		// get the node
		node := network.GetNode(id)
		if node == nil {
			return false, fmt.Errorf("unknown node: %s", id)
		}

		// check it has exactly two peers
		client, err := node.Client()
		if err != nil {
			return false, err
		}
		var peerCount int64
		if err := client.CallContext(ctx, &peerCount, "test_peerCount"); err != nil {
			return false, err
		}
		switch {
		case peerCount < 2:
			return false, nil
		case peerCount == 2:
			return true, nil
		default:
			return false, fmt.Errorf("unexpected peerCount: %d", peerCount)
		}
	}

	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// trigger a check every 100ms
	trigger := make(chan enode.ID)
	go triggerChecks(ctx, ids, trigger, 100*time.Millisecond)

	result := NewSimulation(network).Run(ctx, &Step{
		Action:  action,
		Trigger: trigger,
		Expect: &Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}

	// take a network snapshot and check it contains the correct topology
	snap, err := network.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Nodes) != nodeCount {
		t.Fatalf("expected snapshot to contain %d nodes, got %d", nodeCount, len(snap.Nodes))
	}
	if len(snap.Conns) != nodeCount {
		t.Fatalf("expected snapshot to contain %d connections, got %d", nodeCount, len(snap.Conns))
	}
	for i, id := range ids {
		conn := snap.Conns[i]
		if conn.One != id {
			t.Fatalf("expected conn[%d].One to be %s, got %s", i, id, conn.One)
		}
		peerID := ids[(i+1)%len(ids)]
		if conn.Other != peerID {
			t.Fatalf("expected conn[%d].Other to be %s, got %s", i, peerID, conn.Other)
		}
	}
}

// TestMultiNodeRetrieval creates a multi-node simulation network.
// Full nodes, bootnodes and lightnodes are created.
// Functions for retrieving specific subgroups of nodes are then tested for correctness.
func TestMultiNodeRetrieval(t *testing.T) {
	adapter := adapters.NewSimAdapter(adapters.Services{
		"test": newTestService,
	})
	network := NewNetwork(adapter, &NetworkConfig{
		DefaultService: "test",
	})
	defer network.Shutdown()

	// Create a bootnode
	bootNodeConf := adapters.RandomNodeConfig()
	bootNodeConf.BootNode = true
	bootNode, err := network.NewNodeWithConfig(bootNodeConf)
	if err != nil {
		t.Fatalf("error creating bootnode: %s", err)
	}
	if err := network.Start(bootNode.ID()); err != nil {
		t.Fatalf("error starting bootnode: %s", err)
	}

	// Create 20 light nodes
	lightNodeCount := 20
	lightNodes := make(map[enode.ID]*Node, lightNodeCount)
	for i := 0; i < lightNodeCount; i++ {
		conf := adapters.RandomNodeConfig()
		conf.LightNode = true
		node, err := network.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error creating light node: %s", err)
		}
		if err := network.Start(node.ID()); err != nil {
			t.Fatalf("error starting light node: %s", err)
		}
		lightNodes[node.ID()] = node
	}

	// Create 20 full nodes
	fullNodeCount := 20
	fullNodes := make(map[enode.ID]*Node, fullNodeCount)
	for i := 0; i < fullNodeCount; i++ {
		conf := adapters.RandomNodeConfig()
		node, err := network.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		if err := network.Start(node.ID()); err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		fullNodes[node.ID()] = node
	}

	// Check that network.GetBootNodes returns the boot node we created and only that bootnode
	bootNodes := network.GetBootNodes()
	if len(bootNodes) == 0 {
		t.Fatal("GetBootNodes returned empty when size of one was expected")
	}

	for _, bn := range bootNodes {
		if !bytes.Equal(bn.ID().Bytes(), bootNode.ID().Bytes()) {
			t.Fatalf("Found an unexpected node in GetBootNodes: %s", bn.String())
		}
	}

	// Check that the boot node's ID is the only one returned by GetBoodNodeIDs()
	// If a non-matching ID is found, the test fails
	bootNodeIDs := network.GetBootNodeIDs()
	if len(bootNodeIDs) == 0 {
		t.Fatal("GetBootNodeIDs returned empty when one ID was expected")
	}

	for _, id := range bootNodeIDs {
		if !bytes.Equal(id.Bytes(), bootNode.ID().Bytes()) {
			t.Fatalf("Found an unexpected enode.ID in GetBootNodeIDs: %s", id.String())
		}
	}

	// Check that each of lightNodes (the light nodes we just created) are available from the GetLightNodes method.
	// If a light node isn't found in GetLightNodes, the test fails.
	for _, ln1 := range lightNodes {
		match := false
		lightNode1IDBytes := ln1.ID().Bytes()

		for _, ln2 := range network.GetLightNodes() {
			lightNode2IDBytes := ln2.ID().Bytes()
			if bytes.Equal(lightNode1IDBytes, lightNode2IDBytes) {
				match = true
				break
			}
		}

		if !match {
			t.Fatalf("A created light node was not returned by GetLightNodes(), ID: %s", ln1.ID().String())
		}
	}

	// Check that the IDs of each of fullNodes are returned by GetFullNodeIDs()
	// If a full not isn't found in GetFullNodeIDs(), the test fails
	lightNodeIDs := network.GetLightNodeIDs()
	for id1 := range lightNodes {
		match := false
		for _, id2 := range lightNodeIDs {
			if bytes.Equal(id1.Bytes(), id2.Bytes()) {
				match = true
				break
			}
		}

		if !match {
			t.Fatalf("Not all light nodes were returned by GetLightNodeIDs(), ID: %s", id1.String())
		}
	}

	// Check that each of fullNodes (the full nodes we just created) are available from the GetFullNodes method.
	// If a full node isn't found in GetFullNodes, the test fails.
	for _, fn1 := range fullNodes {
		match := false
		fullNode1IDBytes := fn1.ID().Bytes()

		for _, fn2 := range network.GetFullNodes() {
			fullNode2IDBytes := fn2.ID().Bytes()
			if bytes.Equal(fullNode1IDBytes, fullNode2IDBytes) {
				match = true
				break
			}
		}

		if !match {
			t.Fatalf("A created full node was not returned by GetFullNodes(), ID: %s", fn1.ID().String())
		}
	}

	// Check that the IDs of each of fullNodes are returned by GetFullNodeIDs()
	// If a full not isn't found in GetFullNodeIDs(), the test fails
	fullNodeIDs := network.GetFullNodeIDs()
	for id1 := range fullNodes {
		match := false
		for _, id2 := range fullNodeIDs {
			if bytes.Equal(id1.Bytes(), id2.Bytes()) {
				match = true
				break
			}
		}

		if !match {
			t.Fatalf("Not all full nodes were returned by GetFullNodeIDs(), ID: %s", id1.String())
		}
	}

	// Get all nodes, excluding the bootnode by passing the bootnode ID.
	// Checks that the bootnode is excluded as expected and fails the test if not.
	nodesExclBootNode := network.GetNodes(bootNode.ID())
	for _, node := range nodesExclBootNode {
		if bytes.Equal(node.ID().Bytes(), bootNode.ID().Bytes()) {
			t.Fatalf("Bootnode still found in GetNodes when it has been explicitly excluded.")
		}
	}

	// Get all node IDs and call GetNodesByID using them.
	// The test then confirms that the nodes returned from GetNodes() match those returned from GetNodesByID(allIDs)
	var nodeIDs []enode.ID
	for _, node := range network.GetNodes() {
		nodeIDs = append(nodeIDs, node.ID())
	}

	nodesByID := network.GetNodesByID(nodeIDs)
	for _, node1 := range network.GetNodes() {
		match := false
		node1IDBytes := node1.ID().Bytes()
		for _, node2 := range nodesByID {
			node2IDBytes := node2.ID().Bytes()
			if bytes.Equal(node1IDBytes, node2IDBytes) {
				match = true
				break
			}
		}

		if !match {
			t.Fatalf("A node was found in GetNodes() that was not returned by GetNodesByID() for all node IDs")
		}
	}
}

func triggerChecks(ctx context.Context, ids []enode.ID, trigger chan enode.ID, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			for _, id := range ids {
				select {
				case trigger <- id:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// \todo: refactor to implement shapshots
// and connect configuration methods once these are moved from
// swarm/network/simulations/connect.go
func BenchmarkMinimalService(b *testing.B) {
	b.Run("ring/32", benchmarkMinimalServiceTmp)
}

func benchmarkMinimalServiceTmp(b *testing.B) {

	// stop timer to discard setup time pollution
	args := strings.Split(b.Name(), "/")
	nodeCount, err := strconv.ParseInt(args[2], 10, 16)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		// this is a minimal service, whose protocol will close a channel upon run of protocol
		// making it possible to bench the time it takes for the service to start and protocol actually to be run
		protoCMap := make(map[enode.ID]map[enode.ID]chan struct{})
		adapter := adapters.NewSimAdapter(adapters.Services{
			"noopwoop": func(ctx *adapters.ServiceContext) (node.Service, error) {
				protoCMap[ctx.Config.ID] = make(map[enode.ID]chan struct{})
				svc := NewNoopService(protoCMap[ctx.Config.ID])
				return svc, nil
			},
		})

		// create network
		network := NewNetwork(adapter, &NetworkConfig{
			DefaultService: "noopwoop",
		})
		defer network.Shutdown()

		// create and start nodes
		ids := make([]enode.ID, nodeCount)
		for i := 0; i < int(nodeCount); i++ {
			conf := adapters.RandomNodeConfig()
			node, err := network.NewNodeWithConfig(conf)
			if err != nil {
				b.Fatalf("error creating node: %s", err)
			}
			if err := network.Start(node.ID()); err != nil {
				b.Fatalf("error starting node: %s", err)
			}
			ids[i] = node.ID()
		}

		// ready, set, go
		b.ResetTimer()

		// connect nodes in a ring
		for i, id := range ids {
			peerID := ids[(i+1)%len(ids)]
			if err := network.Connect(id, peerID); err != nil {
				b.Fatal(err)
			}
		}

		// wait for all protocols to signal to close down
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()
		for nodid, peers := range protoCMap {
			for peerid, peerC := range peers {
				log.Debug("getting ", "node", nodid, "peer", peerid)
				select {
				case <-ctx.Done():
					b.Fatal(ctx.Err())
				case <-peerC:
				}
			}
		}
	}
}

func TestNode_UnmarshalJSON(t *testing.T) {
	t.Run(
		"test unmarshal of Node up field",
		func(t *testing.T) {
			runNodeUnmarshalJSON(t, casesNodeUnmarshalJSONUpField())
		},
	)
	t.Run(
		"test unmarshal of Node Config field",
		func(t *testing.T) {
			runNodeUnmarshalJSON(t, casesNodeUnmarshalJSONConfigField())
		},
	)
}

func runNodeUnmarshalJSON(t *testing.T, tests []nodeUnmarshalTestCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Node
			if err := got.UnmarshalJSON([]byte(tt.marshaled)); err != nil {
				expectErrorMessageToContain(t, err, tt.wantErr)
			}
			expectNodeEquality(t, got, tt.want)
		})
	}
}

type nodeUnmarshalTestCase struct {
	name      string
	marshaled string
	want      Node
	wantErr   string
}

func expectErrorMessageToContain(t *testing.T, got error, want string) {
	t.Helper()
	if got == nil && want == "" {
		return
	}

	if got == nil && want != "" {
		t.Errorf("error was expected, got: nil, want: %v", want)
		return
	}

	if !strings.Contains(got.Error(), want) {
		t.Errorf(
			"unexpected error message, got  %v, want: %v",
			want,
			got,
		)
	}
}

func expectNodeEquality(t *testing.T, got Node, want Node) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Node.UnmarshalJSON() = %v, want %v", got, want)
	}
}

func casesNodeUnmarshalJSONUpField() []nodeUnmarshalTestCase {
	return []nodeUnmarshalTestCase{
		{
			name:      "empty json",
			marshaled: "{}",
			want: Node{
				up: false,
			},
		},
		{
			name:      "a stopped node",
			marshaled: "{\"up\": false}",
			want: Node{
				up: false,
			},
		},
		{
			name:      "a running node",
			marshaled: "{\"up\": true}",
			want: Node{
				up: true,
			},
		},
		{
			name:      "invalid JSON value on valid key",
			marshaled: "{\"up\": foo}",
			wantErr:   "invalid character",
		},
		{
			name:      "invalid JSON key and value",
			marshaled: "{foo: bar}",
			wantErr:   "invalid character",
		},
		{
			name:      "bool value expected but got something else (string)",
			marshaled: "{\"up\": \"true\"}",
			wantErr:   "cannot unmarshal string into Go struct",
		},
	}
}

func casesNodeUnmarshalJSONConfigField() []nodeUnmarshalTestCase {
	// Don't do a big fuss around testing, as adapters.NodeConfig should
	// handle it's own serialization. Just do a sanity check.
	return []nodeUnmarshalTestCase{
		{
			name:      "Config field is omitted",
			marshaled: "{}",
			want: Node{
				Config: nil,
			},
		},
		{
			name:      "Config field is nil",
			marshaled: "{\"config\": nil}",
			want: Node{
				Config: nil,
			},
		},
		{
			name:      "a non default Config field",
			marshaled: "{\"config\":{\"name\":\"node_ecdd0\",\"port\":44665}}",
			want: Node{
				Config: &adapters.NodeConfig{
					Name: "node_ecdd0",
					Port: 44665,
				},
			},
		},
	}
}
