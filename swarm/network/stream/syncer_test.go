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
	"io/ioutil"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const dataChunkCount = 200

func TestSyncerSimulation(t *testing.T) {
	testSyncBetweenNodes(t, 2, 1, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 4, 1, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 8, 1, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 16, 1, dataChunkCount, true, 1)
}

func createMockStore(id discover.NodeID, addr *network.BzzAddr) (storage.ChunkStore, error) {
	var err error
	address := common.BytesToAddress(id.Bytes())
	mockStore := globalStore.NewNodeStore(address)
	params := storage.NewDefaultLocalStoreParams()
	datadirs[id], err = ioutil.TempDir("", "localMockStore-"+id.TerminalString())
	if err != nil {
		return nil, err
	}
	params.Init(datadirs[id])
	params.BaseKey = addr.Over()
	lstore, err := storage.NewLocalStore(params, mockStore)
	return lstore, nil
}

func testSyncBetweenNodes(t *testing.T, nodes, conns, chunkCount int, skipCheck bool, po uint8) {
	defer setDefaultSkipCheck(defaultSkipCheck)
	defaultSkipCheck = skipCheck
	//data directories for each node and store
	datadirs = make(map[discover.NodeID]string)
	if *useMockStore {
		createStoreFunc = createMockStore
		createGlobalStore()
	} else {
		createStoreFunc = createTestLocalStorageFromSim
	}
	defer datadirsCleanup()

	registries = make(map[discover.NodeID]*TestRegistry)
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		//hack to put addresses in same space
		addr.OAddr[0] = byte(0)
		return addr
	}
	conf := &streamTesting.RunConfig{
		Adapter:         *adapter,
		NodeCount:       nodes,
		ConnLevel:       conns,
		ToAddr:          toAddr,
		Services:        services,
		EnableMsgEvents: false,
	}
	// HACK: these are global variables in the test so that they are available for
	// the service constructor function
	// TODO: will this work with exec/docker adapter?
	// localstore of nodes made available for action and check calls
	stores = make(map[discover.NodeID]storage.ChunkStore)
	deliveries = make(map[discover.NodeID]*Delivery)
	// create context for simulation run
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel should come before defer simulation teardown
	defer cancel()

	// create simulation network with the config
	sim, teardown, err := streamTesting.NewSimulation(conf)
	var rpcSubscriptionsWg sync.WaitGroup
	defer func() {
		rpcSubscriptionsWg.Wait()
		teardown()
	}()
	if err != nil {
		t.Fatal(err.Error())
	}

	nodeIndex := make(map[discover.NodeID]int)
	for i, id := range sim.IDs {
		nodeIndex[id] = i
		if !*useMockStore {
			stores[id] = sim.Stores[i]
			sim.Stores[i] = stores[id]
		}
	}
	// peerCount function gives the number of peer connections for a nodeID
	// this is needed for the service run function to wait until
	// each protocol  instance runs and the streamer peers are available
	peerCount = func(id discover.NodeID) int {
		if sim.IDs[0] == id || sim.IDs[nodes-1] == id {
			return 1
		}
		return 2
	}
	waitPeerErrC = make(chan error)

	// create DBAPI-s for all nodes
	dbs := make([]*storage.DBAPI, nodes)
	for i := 0; i < nodes; i++ {
		dbs[i] = storage.NewDBAPI(sim.Stores[i].(*storage.LocalStore))
	}

	// collect hashes in po 1 bin for each node
	hashes := make([][]storage.Address, nodes)
	totalHashes := 0
	hashCounts := make([]int, nodes)
	for i := nodes - 1; i >= 0; i-- {
		if i < nodes-1 {
			hashCounts[i] = hashCounts[i+1]
		}
		dbs[i].Iterator(0, math.MaxUint64, po, func(addr storage.Address, index uint64) bool {
			hashes[i] = append(hashes[i], addr)
			totalHashes++
			hashCounts[i]++
			return true
		})
	}

	// errc is error channel for simulation
	errc := make(chan error, 1)
	quitC := make(chan struct{})
	defer close(quitC)

	// action is subscribe
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
		for j := 0; j < nodes-1; j++ {
			id := sim.IDs[j]
			sim.Stores[j] = stores[id]
			err := sim.CallClient(id, func(client *rpc.Client) error {
				// report disconnect events to the error channel cos peers should not disconnect
				doneC, err := streamTesting.WatchDisconnections(id, client, errc, quitC)
				if err != nil {
					return err
				}
				rpcSubscriptionsWg.Add(1)
				go func() {
					<-doneC
					rpcSubscriptionsWg.Done()
				}()
				ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
				// start syncing, i.e., subscribe to upstream peers po 1 bin
				sid := sim.IDs[j+1]
				return client.CallContext(ctx, nil, "stream_subscribeStream", sid, NewStream("SYNC", FormatSyncBinKey(1), false), NewRange(0, 0), Top)
			})
			if err != nil {
				return err
			}
		}
		// here we distribute chunks of a random file into stores 1...nodes
		rrFileStore := storage.NewFileStore(newRoundRobinStore(sim.Stores[1:]...), storage.NewFileStoreParams())
		size := chunkCount * chunkSize
		_, wait, err := rrFileStore.Store(ctx, io.LimitReader(crand.Reader, int64(size)), int64(size), false)
		if err != nil {
			t.Fatal(err.Error())
		}
		// need to wait cos we then immediately collect the relevant bin content
		wait(ctx)
		if err != nil {
			t.Fatal(err.Error())
		}

		return nil
	}

	// this makes sure check is not called before the previous call finishes
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case err := <-errc:
			return false, err
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		i := nodeIndex[id]
		var total, found int

		for j := i; j < nodes; j++ {
			total += len(hashes[j])
			for _, key := range hashes[j] {
				chunk, err := dbs[i].Get(ctx, key)
				if err == storage.ErrFetching {
					<-chunk.ReqC
				} else if err != nil {
					continue
				}
				// needed for leveldb not to be closed?
				// chunk.WaitToStore()
				found++
			}
		}
		log.Debug("sync check", "node", id, "index", i, "bin", po, "found", found, "total", total)
		return total == found, nil
	}

	conf.Step = &simulations.Step{
		Action:  action,
		Trigger: streamTesting.Trigger(500*time.Millisecond, quitC, sim.IDs[0:nodes-1]...),
		Expect: &simulations.Expectation{
			Nodes: sim.IDs[0:1],
			Check: check,
		},
	}
	startedAt := time.Now()
	result, err := sim.Run(ctx, conf)
	finishedAt := time.Now()
	if err != nil {
		t.Fatalf("Setting up simulation failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Simulation failed: %s", result.Error)
	}
	streamTesting.CheckResult(t, result, startedAt, finishedAt)
}
