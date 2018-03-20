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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

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
	ids := make([]discover.NodeID, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := network.NewNode()
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
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
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
	trigger := make(chan discover.NodeID)
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

func triggerChecks(ctx context.Context, ids []discover.NodeID, trigger chan discover.NodeID, interval time.Duration) {
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
