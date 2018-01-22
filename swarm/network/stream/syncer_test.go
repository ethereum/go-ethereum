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
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const dataChunkCount = 500

func TestSyncerSimulation(t *testing.T) {
	testSyncBetweenNodes(t, 2, 1, dataChunkCount, true, 1)
	// testSyncBetweenNodes(t, 2, 1, dataChunkCount, false, 1)
	testSyncBetweenNodes(t, 4, 1, dataChunkCount, true, 1)
	// testSyncBetweenNodes(t, 4, 1, dataChunkCount, false, 1)
	testSyncBetweenNodes(t, 8, 1, dataChunkCount, true, 1)
	// testSyncBetweenNodes(t, 8, 1, dataChunkCount, false, 1)
	testSyncBetweenNodes(t, 16, 1, dataChunkCount, true, 1)
	// testSyncBetweenNodes(t, 16, 1, dataChunkCount, false, 1)
}

func testSyncBetweenNodes(t *testing.T, nodes, conns, chunkCount int, skipCheck bool, po uint8) {
	defaultSkipCheck = skipCheck
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
	for i, id := range sim.IDs {
		stores[id] = sim.Stores[i]
	}
	peerCount = func(id discover.NodeID) int {
		if sim.IDs[0] == id || sim.IDs[nodes-1] == id {
			return 1
		}
		return 2
	}
	// here we distribute chunks of a random file into Stores of nodes 1 to nodes
	rrdpa := storage.NewDPA(newRoundRobinStore(sim.Stores[1:]...), storage.NewChunkerParams())
	rrdpa.Start()
	size := chunkCount * chunkSize
	_, wait, err := rrdpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
	// need to wait cos we then immediately collect the relevant bin content
	wait()
	defer rrdpa.Stop()
	if err != nil {
		t.Fatal(err.Error())
	}

	// collect hashes in po 1 from all nodes
	var hashes []storage.Key
	dbs := make([]*storage.DBAPI, nodes)
	for i := 0; i < nodes; i++ {
		dbs[i] = storage.NewDBAPI(sim.Stores[i].(*storage.LocalStore))
	}
	for i := 1; i < nodes; i++ {
		dbs[i].Iterator(0, math.MaxUint64, po, func(key storage.Key, index uint64) bool {
			hashes = append(hashes, key)
			return true
		})
	}

	waitPeerErrC = make(chan error)
	action := func(ctx context.Context) error {
		// need to wait till an aynchronous process registers the peers in streamer.peers
		// that is used by Subscribe
		// the global peerCount function tells how many connections each node has
		// TODO: this is to be reimplemented with peerEvent watcher without global var
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
		// each node Subscribes to each other's swarmChunkServerStreamName
		j := 0
		return sim.CallClient(func(client *rpc.Client) error {
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			j++
			return client.CallContext(ctx, nil, "stream_subscribeStream", sim.IDs[j], "SYNC", []byte{1}, 0, 0, Top, false)
		}, sim.IDs[0:nodes-1]...)
	}

	// this makes sure check is not called before the previous call finishes
	checkC := make(chan struct{})
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
		defer func() { checkC <- struct{}{} }()

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		var found int
		total := len(hashes)
		for _, key := range hashes {
			_, err := dbs[0].Get(key)
			if err == nil {
				found++
			}
		}
		log.Debug("sync check", "bin", po, "found", found, "total", total)
		return found == total, nil
	}

	conf.Step = &simulations.Step{
		Action:  action,
		Trigger: streamTesting.PivotTrigger(10*time.Millisecond, checkC, sim.IDs[0]),
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
