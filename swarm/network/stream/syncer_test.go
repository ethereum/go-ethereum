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

package stream

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestSyncerSimulation(t *testing.T) {
	testSyncBetweenNodes(t, 2, 1, 81000, true, 1)
	testSyncBetweenNodes(t, 3, 1, 81000, true, 1)
}

func testSyncBetweenNodes(t *testing.T, nodes, conns, size int, skipCheck bool, po uint8) {
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		addr.OAddr[0] = byte(0)
		return addr
	}
	conf := &streamTesting.RunConfig{
		Adapter:   *adapter,
		NodeCount: nodes,
		ConnLevel: conns,
		ToAddr:    toAddr,
		Services:  services,
	}

	sim, teardown, err := streamTesting.NewSimulation(conf)
	defer teardown()
	if err != nil {
		t.Fatal(err.Error())
	}
	stores = make(map[discover.NodeID]storage.ChunkStore)
	deliveries = make(map[discover.NodeID]*Delivery)
	log.Warn("Stores", "len", len(sim.Stores))
	for i, id := range sim.IDs {
		stores[id] = sim.Stores[i]
	}

	// here we distribute chunks of a random file into Stores of nodes 1 to nodes
	rrdpa := storage.NewDPA(newRoundRobinStore(sim.Stores[1:]...), storage.NewChunkerParams())
	rrdpa.Start()
	_, wait, err := rrdpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
	defer rrdpa.Stop()
	if err != nil {
		t.Fatal(err.Error())
	}
	// wait until all chunks stored
	// TODO: is wait() necessary?
	wait()
	// each node Subscribes to each other's swarmChunkServerStreamName
	// need to wait till an aynchronous process registers the peers in streamer.peers
	// that is used by Subscribe
	// time.Sleep(1 * time.Second)
	// err := streamer.Subscribe(p.ID(), swarmChunkServerStreamName, nil, 0, 0, Top, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitPeerErrC = make(chan error)
	// create a retriever dpa for the pivot node
	action := func(context.Context) error {

		i := 0
		for err := range waitPeerErrC {
			if err != nil {
				return fmt.Errorf("error waiting for peers: %s", err)
			}
			i++
			if i == nodes {
				break
			}
		}

		for i := 0; i < len(sim.IDs)-1; i++ {
			id := sim.IDs[i]
			// if err := streamer.Subscribe(p.ID(), "SYNC", []byte{uint8(1)}, 0, 0, Top, false); err != nil {
			// 	log.Warn("error in subscribe", "err", err)
			// }
			node := sim.Net.GetNode(id)
			if node == nil {
				return fmt.Errorf("unknown node: %s", id)
			}
			client, err := node.Client()
			if err != nil {
				return fmt.Errorf("error getting node client: %s", err)
			}
			sid := sim.IDs[i+1]
			if err := client.Call(nil, "stream_subscribeStream", sid, "SYNC", []byte{uint8(1)}, 0, 0, Top, false); err != nil {
				return fmt.Errorf("error subscribing: %s", err)
			}
		}
		return nil
	}

	dbs := make([]*storage.DBAPI, nodes)
	for i := 0; i < nodes; i++ {
		dbs[i] = storage.NewDBAPI(sim.Stores[i].(*storage.LocalStore))
	}

	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
		if id != sim.Net.Nodes[0].ID() {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		var found, total int
		for i := 1; i < nodes; i++ {

			dbs[i].Iterator(0, math.MaxUint64, po, func(key storage.Key, index uint64) bool {
				_, err := dbs[0].Get(key)
				if err == nil {
					found++
				}
				total++
				return true
			})
		}
		log.Debug("sync check", "bin", po, "found", found, "total", total)
		return found == total, nil
	}

	trigger := make(chan discover.NodeID)
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		// we are only testing the pivot node (net.Nodes[0])
		for range ticker.C {
			trigger <- sim.Net.Nodes[0].ID()
		}
	}()

	conf.Step = &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: sim.IDs[0:1],
			Check: check,
		},
	}
	startedAt := time.Now()
	result, err := sim.Run(conf)
	finishedAt := time.Now()
	if err != nil {
		t.Fatalf("Setting up simulation failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Simulation failed: %s", result.Error)
	}
	streamTesting.CheckResult(t, result, startedAt, finishedAt)
}
