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
	//	crand "crypto/rand"
	"fmt"
	//"io"
	"math/rand"
	//"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	//"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	//"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	//"github.com/ethereum/go-ethereum/pot"
	//"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func initRetrievalTest() {
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		return addr
	}
	createStoreFunc = createTestLocalStorageForId
	//local stores
	stores = make(map[discover.NodeID]storage.ChunkStore)
	//data directories for each node and store
	datadirs = make(map[discover.NodeID]string)
	//deliveries for each node
	deliveries = make(map[discover.NodeID]*Delivery)
	getRetrieveFunc = func(id discover.NodeID) func(chunk *storage.Chunk) error {
		return func(chunk *storage.Chunk) error {
			skipCheck := true
			//fmt.Println(fmt.Sprintf("-- %s", id))
			return deliveries[id].RequestFromPeers(chunk.Key[:], skipCheck)
		}
	}
	//registries, map of discover.NodeID to its streamer
	registries = make(map[discover.NodeID]*TestRegistry)
	//channel to wait for peers connected
	//not needed for this test but required from common_test for NewStreamService
	waitPeerErrC = make(chan error)
	//also not needed for this test but required for NewStreamService
	peerCount = func(id discover.NodeID) int {
		if ids[0] == id || ids[len(ids)-1] == id {
			return 1
		}
		return 2
	}
}

func TestRetrieval(t *testing.T) {

	if *nodes != 0 && *chunks != 0 {
		retrievalTest(t, *chunks, *nodes)
	} else {
		var nodeCnt []int
		var chnkCnt []int
		if *longrunning {
			nodeCnt = []int{16, 32, 128}
			chnkCnt = []int{4, 32, 256}
		} else {
			nodeCnt = []int{16}
			chnkCnt = []int{32}
		}
		for _, n := range nodeCnt {
			for _, c := range chnkCnt {
				retrievalTest(t, c, n)
			}
		}
	}
}

func retrievalTest(t *testing.T, chunkCount int, nodeCount int) {
	//test live and NO history
	log.Info("Testing live and no history", "chunkCount", chunkCount, "nodeCount", nodeCount)
	live = true
	history = false
	err := runRetrievalTest(chunkCount, nodeCount)
	if err != nil {
		t.Fatal(err)
	}
	//test history only
	log.Info("Testing history only", "chunkCount", chunkCount, "nodeCount", nodeCount)
	live = false
	history = true
	err = runRetrievalTest(chunkCount, nodeCount)
	if err != nil {
		t.Fatal(err)
	}
	//finally test live and history
	log.Info("Testing live and history", "chunkCount", chunkCount, "nodeCount", nodeCount)
	live = true
	err = runRetrievalTest(chunkCount, nodeCount)
	if err != nil {
		t.Fatal(err)
	}
}

/*
The test generates the given number of chunks,
then uploads these to a random node.
Afterwards for every chunk generated, the nearest node addresses
are identified, syncing is started, and finally we verify
that the nodes closer to the chunk addresses actually do have
the chunks in their local stores.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. The snapshot should have 'streamer' in its service list.
*/
func runRetrievalTest(chunkCount int, nodeCount int) error {
	initRetrievalTest()
	ids = make([]discover.NodeID, nodeCount)
	disconnectC := make(chan error)
	quitC := make(chan struct{})
	conf = &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[discover.NodeID][]int)
	//map of discover ID to kademlia overlay address
	conf.idToAddrMap = make(map[discover.NodeID][]byte)
	//map of overlay address to discover ID
	conf.addrToIdMap = make(map[string]discover.NodeID)
	conf.chunks = make([]storage.Key, 0)
	//load nodes from the snapshot file
	net, err := initNetWithSnapshot(nodeCount)
	if err != nil {
		return err
	}
	//do cleanup after test is terminated
	defer func() {
		doRetrieve = defaultDoRetrieve
		//shutdown the snapshot network
		net.Shutdown()
		//after the test, clean up local stores initialized with createLocalStoreForId
		localStoreCleanup()
		//finally clear all data directories
		datadirsCleanup()
	}()
	//get the nodes of the network
	nodes := net.GetNodes()
	//select one index at random...
	idx := rand.Intn(len(nodes))
	//...and get the the node at that index
	//this is the node selected for upload
	uploadNode := nodes[idx]
	//iterate over all nodes...
	for c := 0; c < len(nodes); c++ {
		//create an array of discovery nodeIDS
		ids[c] = nodes[c].ID()
		a := network.ToOverlayAddr(ids[c].Bytes())
		//append it to the array of all overlay addresses
		conf.addrs = append(conf.addrs, a)
		conf.idToAddrMap[ids[c]] = a
		conf.addrToIdMap[string(a)] = ids[c]
	}

	//needed for healthy call
	ppmap = network.NewPeerPot(testMinProxBinSize, ids, conf.addrs)

	// channel to signal simulation initialisation with action call complete
	// or node disconnections
	//disconnectC := make(chan error)
	//quitC := make(chan struct{})

	trigger := make(chan discover.NodeID)
	action := func(ctx context.Context) error {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			healthy := true
			for _, id := range ids {
				r := registries[id]
				//PeerPot for this node
				pp := ppmap[id]
				//call Healthy RPC
				h := r.delivery.overlay.Healthy(pp)
				//print info
				log.Debug(r.delivery.overlay.String())
				log.Debug(fmt.Sprintf("IS HEALTHY: %t", h.GotNN && h.KnowNN && h.Full))
				if !h.GotNN || !h.Full {
					healthy = false
					break
				}
			}
			if healthy {
				break
			}
		}

		if history {
			log.Info("Uploading for history")
			//If testing only history, we upload the chunk(s) first
			conf.chunks, err = uploadFileToSingleNodeStore(uploadNode.ID(), chunkCount)
			if err != nil {
				return err
			}
		}

		//variables needed to wait for all subscriptions established before uploading
		errc := make(chan error)

		//now setup and start event watching in order to know when we can upload
		ctx, watchCancel := context.WithTimeout(context.Background(), MAX_TIMEOUT*time.Second)
		defer watchCancel()

		log.Info("Setting up stream subscription")
		// each node Subscribes to each other's swarmChunkServerStreamName
		for j, id := range ids {
			log.Trace(fmt.Sprintf("Subscribe to subscription events: %d", j))
			client, err := net.GetNode(id).Client()
			if err != nil {
				return err
			}
			//watch for peers disconnecting
			err = streamTesting.WatchDisconnections(id, client, disconnectC, quitC)
			if err != nil {
				return err
			}

			watchSubscriptionEvents(ctx, id, client, errc)
		}

		for j, id := range ids {
			log.Trace(fmt.Sprintf("Start syncing and stream subscriptions: %d", j))
			client, err := net.GetNode(id).Client()
			if err != nil {
				return err
			}
			//start syncing!
			var cnt int
			err = client.CallContext(ctx, &cnt, "stream_startSyncing")
			if err != nil {
				return err
			}
			subscriptionCount += cnt
			for snid := range registries[id].peers {
				subscriptionCount++
				err = client.CallContext(ctx, nil, "stream_subscribeStream", snid, NewStream(swarmChunkServerStreamName, "", false), nil, Top)
				if err != nil {
					return err
				}
			}
		}

		//now wait until the number of expected subscriptions has been finished
		for err := range errc {
			if err != nil {
				return err
			}
			subscriptionCount--
			if subscriptionCount == 0 {
				break
			}
		}

		log.Info("Stream subscriptions successfully requested, action terminated")

		if live {
			//now upload the chunks to the selected random single node
			chnks, err := uploadFileToSingleNodeStore(uploadNode.ID(), chunkCount)
			if err != nil {
				return err
			}
			conf.chunks = append(conf.chunks, chnks...)
		}

		return nil
	}

	chunkSize := storage.DefaultChunkSize

	//check defines what will be checked during the test
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {

		if id == uploadNode.ID() {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case e := <-disconnectC:
			log.Error(e.Error())
			return false, fmt.Errorf("Disconnect event detected, network unhealthy")
		default:
		}
		log.Trace(fmt.Sprintf("Checking node: %s", id))
		//if there are more than one chunk, test only succeeds if all expected chunks are found
		allSuccess := true

		dpa := registries[id].dpa
		for _, chnk := range conf.chunks {
			reader := dpa.Retrieve(chnk)
			if s, err := reader.Size(nil); err != nil || s != chunkSize {
				allSuccess = false
				log.Warn("Retrieve error", "err", err, "chunk", chnk, "nodeId", id)
			} else {
				log.Debug(fmt.Sprintf("Chunk %x found", chnk))
			}
		}
		return allSuccess, nil
	}

	//for each tick, run the checks on all nodes
	timingTicker := time.NewTicker(5 * time.Second)
	defer timingTicker.Stop()
	go func() {
		for range timingTicker.C {
			for i := 0; i < len(ids); i++ {
				log.Trace(fmt.Sprintf("triggering step %d, id %s", i, ids[i]))
				trigger <- ids[i]
			}
		}
	}()

	log.Info("Starting simulation run...")

	timeout := MAX_TIMEOUT * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//run the simulation
	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	//close(quitC)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

//upload a file(chunks)
/*
func uploadRandomChunks(net *simulations.Network, chunkCount int) error {
	log.Debug(fmt.Sprintf("Uploading to node id: %s", id))
	lstore := stores[id]
	size := chunkCount * chunkSize
	dpa := storage.NewDPA(lstore, storage.NewChunkerParams())
	dpa.Start()
	rootHash, wait, err := dpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
	wait()
	if err != nil {
		return nil, err
	}

	defer dpa.Stop()

	return rootHash, nil
}
*/
