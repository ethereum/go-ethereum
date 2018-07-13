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
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

//constants for random file generation
const (
	minFileSize = 2
	maxFileSize = 40
)

func initRetrievalTest() {
	//global func to get overlay address from discover ID
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		return addr
	}
	//global func to create local store
	createStoreFunc = createTestLocalStorageForId
	//local stores
	stores = make(map[discover.NodeID]storage.ChunkStore)
	//data directories for each node and store
	datadirs = make(map[discover.NodeID]string)
	//deliveries for each node
	deliveries = make(map[discover.NodeID]*Delivery)
	//global retrieve func
	getRetrieveFunc = func(id discover.NodeID) func(chunk *storage.Chunk) error {
		return func(chunk *storage.Chunk) error {
			skipCheck := true
			return deliveries[id].RequestFromPeers(chunk.Addr[:], skipCheck)
		}
	}
	//registries, map of discover.NodeID to its streamer
	registries = make(map[discover.NodeID]*TestRegistry)
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

//This test is a retrieval test for nodes.
//A configurable number of nodes can be
//provided to the test.
//Files are uploaded to nodes, other nodes try to retrieve the file
//Number of nodes can be provided via commandline too.
func TestFileRetrieval(t *testing.T) {
	if *nodes != 0 {
		fileRetrievalTest(t, *nodes)
	} else {
		nodeCnt := []int{16}
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			nodeCnt = append(nodeCnt, 32, 64, 128)
		}
		for _, n := range nodeCnt {
			fileRetrievalTest(t, n)
		}
	}
}

//This test is a retrieval test for nodes.
//One node is randomly selected to be the pivot node.
//A configurable number of chunks and nodes can be
//provided to the test, the number of chunks is uploaded
//to the pivot node and other nodes try to retrieve the chunk(s).
//Number of chunks and nodes can be provided via commandline too.
func TestRetrieval(t *testing.T) {
	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		retrievalTest(t, *chunks, *nodes)
	} else {
		var nodeCnt []int
		var chnkCnt []int
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			nodeCnt = []int{16, 32, 128}
			chnkCnt = []int{4, 32, 256}
		} else {
			//default test
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

//Every test runs 3 times, a live, a history, and a live AND history
func fileRetrievalTest(t *testing.T, nodeCount int) {
	//test live and NO history
	log.Info("Testing live and no history", "nodeCount", nodeCount)
	live = true
	history = false
	err := runFileRetrievalTest(nodeCount)
	if err != nil {
		t.Fatal(err)
	}
	//test history only
	log.Info("Testing history only", "nodeCount", nodeCount)
	live = false
	history = true
	err = runFileRetrievalTest(nodeCount)
	if err != nil {
		t.Fatal(err)
	}
	//finally test live and history
	log.Info("Testing live and history", "nodeCount", nodeCount)
	live = true
	err = runFileRetrievalTest(nodeCount)
	if err != nil {
		t.Fatal(err)
	}
}

//Every test runs 3 times, a live, a history, and a live AND history
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

The upload is done by dependency to the global
`live` and `history` variables;

If `live` is set, first stream subscriptions are established,
then files are uploaded to nodes.

If `history` is enabled, first upload files, then build up subscriptions.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. Nevertheless a health check runs in the
simulation's `action` function.

The snapshot should have 'streamer' in its service list.
*/
func runFileRetrievalTest(nodeCount int) error {
	//for every run (live, history), int the variables
	initRetrievalTest()
	//the ids of the snapshot nodes, initiate only now as we need nodeCount
	ids = make([]discover.NodeID, nodeCount)
	//channel to check for disconnection errors
	disconnectC := make(chan error)
	//channel to close disconnection watcher routine
	quitC := make(chan struct{})
	//the test conf (using same as in `snapshot_sync_test`
	conf = &synctestConfig{}
	//map of overlay address to discover ID
	conf.addrToIdMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)
	//load nodes from the snapshot file
	net, err := initNetWithSnapshot(nodeCount)
	if err != nil {
		return err
	}
	var rpcSubscriptionsWg sync.WaitGroup
	//do cleanup after test is terminated
	defer func() {
		//shutdown the snapshot network
		net.Shutdown()
		//after the test, clean up local stores initialized with createLocalStoreForId
		localStoreCleanup()
		//finally clear all data directories
		datadirsCleanup()
	}()
	//get the nodes of the network
	nodes := net.GetNodes()
	//iterate over all nodes...
	for c := 0; c < len(nodes); c++ {
		//create an array of discovery nodeIDS
		ids[c] = nodes[c].ID()
		a := network.ToOverlayAddr(ids[c].Bytes())
		//append it to the array of all overlay addresses
		conf.addrs = append(conf.addrs, a)
		conf.addrToIdMap[string(a)] = ids[c]
	}

	//needed for healthy call
	ppmap = network.NewPeerPotMap(testMinProxBinSize, conf.addrs)

	//an array for the random files
	var randomFiles []string
	//channel to signal when the upload has finished
	uploadFinished := make(chan struct{})
	//channel to trigger new node checks
	trigger := make(chan discover.NodeID)
	//simulation action
	action := func(ctx context.Context) error {
		//first run the health check on all nodes,
		//wait until nodes are all healthy
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			healthy := true
			for _, id := range ids {
				r := registries[id]
				//PeerPot for this node
				addr := common.Bytes2Hex(r.addr.OAddr)
				pp := ppmap[addr]
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
			conf.hashes, randomFiles, err = uploadFilesToNodes(nodes)
			if err != nil {
				return err
			}
		}

		//variables needed to wait for all subscriptions established before uploading
		errc := make(chan error)

		//now setup and start event watching in order to know when we can upload
		ctx, watchCancel := context.WithTimeout(context.Background(), MaxTimeout*time.Second)
		defer watchCancel()

		log.Info("Setting up stream subscription")
		//We need two iterations, one to subscribe to the subscription events
		//(so we know when setup phase is finished), and one to
		//actually run the stream subscriptions. We can't do it in the same iteration,
		//because while the first nodes in the loop are setting up subscriptions,
		//the latter ones have not subscribed to listen to peer events yet,
		//and then we miss events.

		//first iteration: setup disconnection watcher and subscribe to peer events
		for j, id := range ids {
			log.Trace(fmt.Sprintf("Subscribe to subscription events: %d", j))
			client, err := net.GetNode(id).Client()
			if err != nil {
				return err
			}
			wsDoneC := watchSubscriptionEvents(ctx, id, client, errc, quitC)
			// doneC is nil, the error happened which is sent to errc channel, already
			if wsDoneC == nil {
				continue
			}
			rpcSubscriptionsWg.Add(1)
			go func() {
				<-wsDoneC
				rpcSubscriptionsWg.Done()
			}()

			//watch for peers disconnecting
			wdDoneC, err := streamTesting.WatchDisconnections(id, client, disconnectC, quitC)
			if err != nil {
				return err
			}
			rpcSubscriptionsWg.Add(1)
			go func() {
				<-wdDoneC
				rpcSubscriptionsWg.Done()
			}()
		}

		//second iteration: start syncing and setup stream subscriptions
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
			//increment the number of subscriptions we need to wait for
			//by the count returned from startSyncing (SYNC subscriptions)
			subscriptionCount += cnt
			//now also add the number of RETRIEVAL_REQUEST subscriptions
			for snid := range registries[id].peers {
				subscriptionCount++
				err = client.CallContext(ctx, nil, "stream_subscribeStream", snid, NewStream(swarmChunkServerStreamName, "", false), nil, Top)
				if err != nil {
					return err
				}
			}
		}

		//now wait until the number of expected subscriptions has been finished
		//`watchSubscriptionEvents` will write with a `nil` value to errc
		//every time a `SubscriptionMsg` has been received
		for err := range errc {
			if err != nil {
				return err
			}
			//`nil` received, decrement count
			subscriptionCount--
			//all subscriptions received
			if subscriptionCount == 0 {
				break
			}
		}

		log.Info("Stream subscriptions successfully requested, action terminated")

		if live {
			//upload generated files to nodes
			var hashes []storage.Address
			var rfiles []string
			hashes, rfiles, err = uploadFilesToNodes(nodes)
			if err != nil {
				return err
			}
			conf.hashes = append(conf.hashes, hashes...)
			randomFiles = append(randomFiles, rfiles...)
			//signal to the trigger loop that the upload has finished
			uploadFinished <- struct{}{}
		}

		return nil
	}

	//check defines what will be checked during the test
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {

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

		//check on the node's FileStore (netstore)
		fileStore := registries[id].fileStore
		//check all chunks
		for i, hash := range conf.hashes {
			reader, _ := fileStore.Retrieve(context.TODO(), hash)
			//check that we can read the file size and that it corresponds to the generated file size
			if s, err := reader.Size(nil); err != nil || s != int64(len(randomFiles[i])) {
				allSuccess = false
				log.Warn("Retrieve error", "err", err, "hash", hash, "nodeId", id)
			} else {
				log.Debug(fmt.Sprintf("File with root hash %x successfully retrieved", hash))
			}
		}

		return allSuccess, nil
	}

	//for each tick, run the checks on all nodes
	timingTicker := time.NewTicker(5 * time.Second)
	defer timingTicker.Stop()
	go func() {
		//for live upload, we should wait for uploads to have finished
		//before starting to trigger the checks, due to file size
		if live {
			<-uploadFinished
		}
		for range timingTicker.C {
			for i := 0; i < len(ids); i++ {
				log.Trace(fmt.Sprintf("triggering step %d, id %s", i, ids[i]))
				trigger <- ids[i]
			}
		}
	}()

	log.Info("Starting simulation run...")

	timeout := MaxTimeout * time.Second
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

	if result.Error != nil {
		return result.Error
	}

	return nil
}

/*
The test generates the given number of chunks.

The upload is done by dependency to the global
`live` and `history` variables;

If `live` is set, first stream subscriptions are established, then
upload to a random node.

If `history` is enabled, first upload then build up subscriptions.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. Nevertheless a health check runs in the
simulation's `action` function.

The snapshot should have 'streamer' in its service list.
*/
func runRetrievalTest(chunkCount int, nodeCount int) error {
	//for every run (live, history), int the variables
	initRetrievalTest()
	//the ids of the snapshot nodes, initiate only now as we need nodeCount
	ids = make([]discover.NodeID, nodeCount)
	//channel to check for disconnection errors
	disconnectC := make(chan error)
	//channel to close disconnection watcher routine
	quitC := make(chan struct{})
	//the test conf (using same as in `snapshot_sync_test`
	conf = &synctestConfig{}
	//map of overlay address to discover ID
	conf.addrToIdMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)
	//load nodes from the snapshot file
	net, err := initNetWithSnapshot(nodeCount)
	if err != nil {
		return err
	}
	var rpcSubscriptionsWg sync.WaitGroup
	//do cleanup after test is terminated
	defer func() {
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
		conf.addrToIdMap[string(a)] = ids[c]
	}

	//needed for healthy call
	ppmap = network.NewPeerPotMap(testMinProxBinSize, conf.addrs)

	trigger := make(chan discover.NodeID)
	//simulation action
	action := func(ctx context.Context) error {
		//first run the health check on all nodes,
		//wait until nodes are all healthy
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			healthy := true
			for _, id := range ids {
				r := registries[id]
				//PeerPot for this node
				addr := common.Bytes2Hex(network.ToOverlayAddr(id.Bytes()))
				pp := ppmap[addr]
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
			conf.hashes, err = uploadFileToSingleNodeStore(uploadNode.ID(), chunkCount)
			if err != nil {
				return err
			}
		}

		//variables needed to wait for all subscriptions established before uploading
		errc := make(chan error)

		//now setup and start event watching in order to know when we can upload
		ctx, watchCancel := context.WithTimeout(context.Background(), MaxTimeout*time.Second)
		defer watchCancel()

		log.Info("Setting up stream subscription")
		//We need two iterations, one to subscribe to the subscription events
		//(so we know when setup phase is finished), and one to
		//actually run the stream subscriptions. We can't do it in the same iteration,
		//because while the first nodes in the loop are setting up subscriptions,
		//the latter ones have not subscribed to listen to peer events yet,
		//and then we miss events.

		//first iteration: setup disconnection watcher and subscribe to peer events
		for j, id := range ids {
			log.Trace(fmt.Sprintf("Subscribe to subscription events: %d", j))
			client, err := net.GetNode(id).Client()
			if err != nil {
				return err
			}

			//check for `SubscribeMsg` events to know when setup phase is complete
			wsDoneC := watchSubscriptionEvents(ctx, id, client, errc, quitC)
			// doneC is nil, the error happened which is sent to errc channel, already
			if wsDoneC == nil {
				continue
			}
			rpcSubscriptionsWg.Add(1)
			go func() {
				<-wsDoneC
				rpcSubscriptionsWg.Done()
			}()

			//watch for peers disconnecting
			wdDoneC, err := streamTesting.WatchDisconnections(id, client, disconnectC, quitC)
			if err != nil {
				return err
			}
			rpcSubscriptionsWg.Add(1)
			go func() {
				<-wdDoneC
				rpcSubscriptionsWg.Done()
			}()
		}

		//second iteration: start syncing and setup stream subscriptions
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
			//increment the number of subscriptions we need to wait for
			//by the count returned from startSyncing (SYNC subscriptions)
			subscriptionCount += cnt
			//now also add the number of RETRIEVAL_REQUEST subscriptions
			for snid := range registries[id].peers {
				subscriptionCount++
				err = client.CallContext(ctx, nil, "stream_subscribeStream", snid, NewStream(swarmChunkServerStreamName, "", false), nil, Top)
				if err != nil {
					return err
				}
			}
		}

		//now wait until the number of expected subscriptions has been finished
		//`watchSubscriptionEvents` will write with a `nil` value to errc
		//every time a `SubscriptionMsg` has been received
		for err := range errc {
			if err != nil {
				return err
			}
			//`nil` received, decrement count
			subscriptionCount--
			//all subscriptions received
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
			conf.hashes = append(conf.hashes, chnks...)
		}

		return nil
	}

	chunkSize := storage.DefaultChunkSize

	//check defines what will be checked during the test
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {

		//don't check the uploader node
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

		//check on the node's FileStore (netstore)
		fileStore := registries[id].fileStore
		//check all chunks
		for _, chnk := range conf.hashes {
			reader, _ := fileStore.Retrieve(context.TODO(), chnk)
			//assuming that reading the Size of the chunk is enough to know we found it
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

	timeout := MaxTimeout * time.Second
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

	if result.Error != nil {
		return result.Error
	}

	return nil
}

//upload generated files to nodes
//every node gets one file uploaded
func uploadFilesToNodes(nodes []*simulations.Node) ([]storage.Address, []string, error) {
	nodeCnt := len(nodes)
	log.Debug(fmt.Sprintf("Uploading %d files to nodes", nodeCnt))
	//array holding generated files
	rfiles := make([]string, nodeCnt)
	//array holding the root hashes of the files
	rootAddrs := make([]storage.Address, nodeCnt)

	var err error
	//for every node, generate a file and upload
	for i, n := range nodes {
		id := n.ID()
		fileStore := registries[id].fileStore
		//generate a file
		rfiles[i], err = generateRandomFile()
		if err != nil {
			return nil, nil, err
		}
		//store it (upload it) on the FileStore
		ctx := context.TODO()
		rk, wait, err := fileStore.Store(ctx, strings.NewReader(rfiles[i]), int64(len(rfiles[i])), false)
		log.Debug("Uploaded random string file to node")
		if err != nil {
			return nil, nil, err
		}
		err = wait(ctx)
		if err != nil {
			return nil, nil, err
		}
		rootAddrs[i] = rk
	}
	return rootAddrs, rfiles, nil
}

//generate a random file (string)
func generateRandomFile() (string, error) {
	//generate a random file size between minFileSize and maxFileSize
	fileSize := rand.Intn(maxFileSize-minFileSize) + minFileSize
	log.Debug(fmt.Sprintf("Generated file with filesize %d kB", fileSize))
	b := make([]byte, fileSize*1024)
	_, err := crand.Read(b)
	if err != nil {
		log.Error("Error generating random file.", "err", err)
		return "", err
	}
	return string(b), nil
}
