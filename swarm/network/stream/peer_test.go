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

package stream

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
)

// TestSyncSubscriptionsDiff validates the output of syncSubscriptionsDiff
// function for various arguments.
func TestSyncSubscriptionsDiff(t *testing.T) {
	max := network.NewKadParams().MaxProxDisplay
	for _, tc := range []struct {
		po, prevDepth, newDepth int
		subBins, quitBins       []int
	}{
		{
			po: 0, prevDepth: -1, newDepth: 0,
			subBins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 1, prevDepth: -1, newDepth: 0,
			subBins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 2, prevDepth: -1, newDepth: 0,
			subBins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 0, prevDepth: -1, newDepth: 1,
			subBins: []int{0},
		},
		{
			po: 1, prevDepth: -1, newDepth: 1,
			subBins: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 2, prevDepth: -1, newDepth: 2,
			subBins: []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 3, prevDepth: -1, newDepth: 2,
			subBins: []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 1, prevDepth: -1, newDepth: 2,
			subBins: []int{1},
		},
		{
			po: 0, prevDepth: 0, newDepth: 0, // 0-16 -> 0-16
		},
		{
			po: 1, prevDepth: 0, newDepth: 0, // 0-16 -> 0-16
		},
		{
			po: 0, prevDepth: 0, newDepth: 1, // 0-16 -> 0
			quitBins: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 0, prevDepth: 0, newDepth: 2, // 0-16 -> 0
			quitBins: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 1, prevDepth: 0, newDepth: 1, // 0-16 -> 1-16
			quitBins: []int{0},
		},
		{
			po: 1, prevDepth: 1, newDepth: 0, // 1-16 -> 0-16
			subBins: []int{0},
		},
		{
			po: 4, prevDepth: 0, newDepth: 1, // 0-16 -> 1-16
			quitBins: []int{0},
		},
		{
			po: 4, prevDepth: 0, newDepth: 4, // 0-16 -> 4-16
			quitBins: []int{0, 1, 2, 3},
		},
		{
			po: 4, prevDepth: 0, newDepth: 5, // 0-16 -> 4
			quitBins: []int{0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 4, prevDepth: 5, newDepth: 0, // 4 -> 0-16
			subBins: []int{0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			po: 4, prevDepth: 5, newDepth: 6, // 4 -> 4
		},
	} {
		subBins, quitBins := syncSubscriptionsDiff(tc.po, tc.prevDepth, tc.newDepth, max)
		if fmt.Sprint(subBins) != fmt.Sprint(tc.subBins) {
			t.Errorf("po: %v, prevDepth: %v, newDepth: %v: got subBins %v, want %v", tc.po, tc.prevDepth, tc.newDepth, subBins, tc.subBins)
		}
		if fmt.Sprint(quitBins) != fmt.Sprint(tc.quitBins) {
			t.Errorf("po: %v, prevDepth: %v, newDepth: %v: got quitBins %v, want %v", tc.po, tc.prevDepth, tc.newDepth, quitBins, tc.quitBins)
		}
	}
}

// TestUpdateSyncingSubscriptions validates that syncing subscriptions are correctly
// made on initial node connections and that subscriptions are correctly changed
// when kademlia neighbourhood depth is changed by connecting more nodes.
func TestUpdateSyncingSubscriptions(t *testing.T) {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr, netStore, delivery, clean, err := newNetStoreAndDelivery(ctx, bucket)
			if err != nil {
				return nil, nil, err
			}
			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				SyncUpdateDelay: 100 * time.Millisecond,
				Syncing:         SyncingAutoSubscribe,
			}, nil)
			cleanup = func() {
				r.Close()
				clean()
			}
			bucket.Store("bzz-address", addr)
			return r, cleanup, nil
		},
	})
	defer sim.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) (err error) {
		// initial nodes, first one as pivot center of the start
		ids, err := sim.AddNodesAndConnectStar(10)
		if err != nil {
			return err
		}

		// pivot values
		pivotRegistryID := ids[0]
		pivotRegistry := sim.Service("streamer", pivotRegistryID).(*Registry)
		pivotKademlia := pivotRegistry.delivery.kad
		// nodes proximities from the pivot node
		nodeProximities := make(map[string]int)
		for _, id := range ids[1:] {
			bzzAddr, ok := sim.NodeItem(id, "bzz-address")
			if !ok {
				t.Fatal("no bzz address for node")
			}
			nodeProximities[id.String()] = chunk.Proximity(pivotKademlia.BaseAddr(), bzzAddr.(*network.BzzAddr).Over())
		}
		// wait until sync subscriptions are done for all nodes
		waitForSubscriptions(t, pivotRegistry, ids[1:]...)

		// check initial sync streams
		err = checkSyncStreamsWithRetry(pivotRegistry, nodeProximities)
		if err != nil {
			return err
		}

		// add more nodes until the depth is changed
		prevDepth := pivotKademlia.NeighbourhoodDepth()
		var noDepthChangeChecked bool // true it there was a check when no depth is changed
		for {
			ids, err := sim.AddNodes(5)
			if err != nil {
				return err
			}
			// add new nodes to sync subscriptions check
			for _, id := range ids {
				bzzAddr, ok := sim.NodeItem(id, "bzz-address")
				if !ok {
					t.Fatal("no bzz address for node")
				}
				nodeProximities[id.String()] = chunk.Proximity(pivotKademlia.BaseAddr(), bzzAddr.(*network.BzzAddr).Over())
			}
			err = sim.Net.ConnectNodesStar(ids, pivotRegistryID)
			if err != nil {
				return err
			}
			waitForSubscriptions(t, pivotRegistry, ids...)

			newDepth := pivotKademlia.NeighbourhoodDepth()
			// depth is not changed, check if streams are still correct
			if newDepth == prevDepth {
				err = checkSyncStreamsWithRetry(pivotRegistry, nodeProximities)
				if err != nil {
					return err
				}
				noDepthChangeChecked = true
			}
			// do the final check when depth is changed and
			// there has been at least one check
			// for the case when depth is not changed
			if newDepth != prevDepth && noDepthChangeChecked {
				// check sync streams for changed depth
				return checkSyncStreamsWithRetry(pivotRegistry, nodeProximities)
			}
			prevDepth = newDepth
		}
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

// waitForSubscriptions is a test helper function that blocks until
// stream server subscriptions are established on the provided registry
// to the nodes with provided IDs.
func waitForSubscriptions(t *testing.T, r *Registry, ids ...enode.ID) {
	t.Helper()

	for retries := 0; retries < 100; retries++ {
		subs := r.api.GetPeerServerSubscriptions()
		if allSubscribed(subs, ids) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("missing subscriptions")
}

// allSubscribed returns true if nodes with ids have subscriptions
// in provided subs map.
func allSubscribed(subs map[string][]string, ids []enode.ID) bool {
	for _, id := range ids {
		if s, ok := subs[id.String()]; !ok || len(s) == 0 {
			return false
		}
	}
	return true
}

// checkSyncStreamsWithRetry is calling checkSyncStreams with retries.
func checkSyncStreamsWithRetry(r *Registry, nodeProximities map[string]int) (err error) {
	for retries := 0; retries < 5; retries++ {
		err = checkSyncStreams(r, nodeProximities)
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return err
}

// checkSyncStreams validates that registry contains expected sync
// subscriptions to nodes with proximities in a map nodeProximities.
func checkSyncStreams(r *Registry, nodeProximities map[string]int) error {
	depth := r.delivery.kad.NeighbourhoodDepth()
	maxPO := r.delivery.kad.MaxProxDisplay
	for id, po := range nodeProximities {
		wantStreams := syncStreams(po, depth, maxPO)
		gotStreams := nodeStreams(r, id)

		if r.getPeer(enode.HexID(id)) == nil {
			// ignore removed peer
			continue
		}

		if !reflect.DeepEqual(gotStreams, wantStreams) {
			return fmt.Errorf("node %s got streams %v, want %v", id, gotStreams, wantStreams)
		}
	}
	return nil
}

// syncStreams returns expected sync streams that need to be
// established between a node with kademlia neighbourhood depth
// and a node with proximity order po.
func syncStreams(po, depth, maxPO int) (streams []string) {
	start, end := syncBins(po, depth, maxPO)
	for bin := start; bin < end; bin++ {
		streams = append(streams, NewStream("SYNC", FormatSyncBinKey(uint8(bin)), false).String())
		streams = append(streams, NewStream("SYNC", FormatSyncBinKey(uint8(bin)), true).String())
	}
	return streams
}

// nodeStreams returns stream server subscriptions on a registry
// to the peer with provided id.
func nodeStreams(r *Registry, id string) []string {
	streams := r.api.GetPeerServerSubscriptions()[id]
	sort.Strings(streams)
	return streams
}
