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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const testMinProxBinSize = 2
const MaxTimeout = 600

var (
	pof = pot.DefaultPof(256)

	conf     *synctestConfig
	ids      []discover.NodeID
	datadirs map[discover.NodeID]string
	ppmap    map[string]*network.PeerPot

	live    bool
	history bool

	longrunning = flag.Bool("longrunning", false, "do run long-running tests")
)

type synctestConfig struct {
	addrs            [][]byte
	hashes           []storage.Address
	idToChunksMap    map[discover.NodeID][]int
	chunksToNodesMap map[string][]int
	addrToIdMap      map[string]discover.NodeID
}

func init() {
	rand.Seed(time.Now().Unix())
}

//common_test needs to initialize the test in a init() func
//in order for adapters to register the NewStreamerService;
//this service is dependent on some global variables
//we thus need to initialize first as init() as well.
func initSyncTest() {
	//assign the toAddr func so NewStreamerService can build the addr
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		return addr
	}
	//global func to create local store
	if *useMockStore {
		createStoreFunc = createMockStore
	} else {
		createStoreFunc = createTestLocalStorageForId
	}
	//local stores
	stores = make(map[discover.NodeID]storage.ChunkStore)
	//data directories for each node and store
	datadirs = make(map[discover.NodeID]string)
	//deliveries for each node
	deliveries = make(map[discover.NodeID]*Delivery)
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
	if *useMockStore {
		createGlobalStore()
	}
}

//This test is a syncing test for nodes.
//One node is randomly selected to be the pivot node.
//A configurable number of chunks and nodes can be
//provided to the test, the number of chunks is uploaded
//to the pivot node, and we check that nodes get the chunks
//they are expected to store based on the syncing protocol.
//Number of chunks and nodes can be provided via commandline too.
func TestSyncing(t *testing.T) {
	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		log.Info(fmt.Sprintf("Running test with %d chunks and %d nodes...", *chunks, *nodes))
		testSyncing(t, *chunks, *nodes)
	} else {
		var nodeCnt []int
		var chnkCnt []int
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			chnkCnt = []int{1, 8, 32, 256, 1024}
			nodeCnt = []int{16, 32, 64, 128, 256}
		} else {
			//default test
			chnkCnt = []int{4, 32}
			nodeCnt = []int{32, 16}
		}
		for _, chnk := range chnkCnt {
			for _, n := range nodeCnt {
				log.Info(fmt.Sprintf("Long running test with %d chunks and %d nodes...", chnk, n))
				testSyncing(t, chnk, n)
			}
		}
	}
}

//Do run the tests
//Every test runs 3 times, a live, a history, and a live AND history
func testSyncing(t *testing.T, chunkCount int, nodeCount int) {
	//test live and NO history
	log.Info("Testing live and no history")
	live = true
	history = false
	err := runSyncTest(chunkCount, nodeCount, live, history)
	if err != nil {
		t.Fatal(err)
	}
	//test history only
	log.Info("Testing history only")
	live = false
	history = true
	err = runSyncTest(chunkCount, nodeCount, live, history)
	if err != nil {
		t.Fatal(err)
	}
	//finally test live and history
	log.Info("Testing live and history")
	live = true
	err = runSyncTest(chunkCount, nodeCount, live, history)
	if err != nil {
		t.Fatal(err)
	}
}

/*
The test generates the given number of chunks

The upload is done by dependency to the global
`live` and `history` variables;

If `live` is set, first stream subscriptions are established, then
upload to a random node.

If `history` is enabled, first upload then build up subscriptions.

For every chunk generated, the nearest node addresses
are identified, we verify that the nodes closer to the
chunk addresses actually do have the chunks in their local stores.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. The snapshot should have 'streamer' in its service list.

For every test run, a series of three tests will be executed:
- a LIVE test first, where first subscriptions are established,
  then a file (random chunks) is uploaded
- a HISTORY test, where the file is uploaded first, and then
  the subscriptions are established
- a crude LIVE AND HISTORY test last, where (different) chunks
  are uploaded twice, once before and once after subscriptions
*/
func runSyncTest(chunkCount int, nodeCount int, live bool, history bool) error {
	initSyncTest()
	//the ids of the snapshot nodes, initiate only now as we need nodeCount
	ids = make([]discover.NodeID, nodeCount)
	//initialize the test struct
	conf = &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[discover.NodeID][]int)
	//map of overlay address to discover ID
	conf.addrToIdMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)
	//channel to trigger node checks in the simulation
	trigger := make(chan discover.NodeID)
	//channel to check for disconnection errors
	disconnectC := make(chan error)
	//channel to close disconnection watcher routine
	quitC := make(chan struct{})

	//load nodes from the snapshot file
	net, err := initNetWithSnapshot(nodeCount)
	if err != nil {
		return err
	}
	var rpcSubscriptionsWg sync.WaitGroup
	//do cleanup after test is terminated
	defer func() {
		// close quitC channel to signall all goroutines to clanup
		// before calling simulation network shutdown.
		close(quitC)
		//wait for all rpc subscriptions to unsubscribe
		rpcSubscriptionsWg.Wait()
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
	node := nodes[idx]

	log.Info("Initializing test config")
	//iterate over all nodes...
	for c := 0; c < len(nodes); c++ {
		//create an array of discovery node IDs
		ids[c] = nodes[c].ID()
		//get the kademlia overlay address from this ID
		a := network.ToOverlayAddr(ids[c].Bytes())
		//append it to the array of all overlay addresses
		conf.addrs = append(conf.addrs, a)
		//the proximity calculation is on overlay addr,
		//the p2p/simulations check func triggers on discover.NodeID,
		//so we need to know which overlay addr maps to which nodeID
		conf.addrToIdMap[string(a)] = ids[c]
	}
	log.Info("Test config successfully initialized")

	//only needed for healthy call when debugging
	ppmap = network.NewPeerPotMap(testMinProxBinSize, conf.addrs)

	//define the action to be performed before the test checks: start syncing
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
			chunks, err := uploadFileToSingleNodeStore(node.ID(), chunkCount)
			if err != nil {
				return err
			}
			conf.hashes = append(conf.hashes, chunks...)
			//finally map chunks to the closest addresses
			mapKeysToNodes(conf)
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

		//second iteration: start syncing
		for j, id := range ids {
			log.Trace(fmt.Sprintf("Start syncing subscriptions: %d", j))
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
		}

		//now wait until the number of expected subscriptions has been finished
		//`watchSubscriptionEvents` will write with a `nil` value to errc
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

		log.Info("Stream subscriptions successfully requested")
		if live {
			//now upload the chunks to the selected random single node
			hashes, err := uploadFileToSingleNodeStore(node.ID(), chunkCount)
			if err != nil {
				return err
			}
			conf.hashes = append(conf.hashes, hashes...)
			//finally map chunks to the closest addresses
			log.Debug(fmt.Sprintf("Uploaded chunks for live syncing: %v", conf.hashes))
			mapKeysToNodes(conf)
			log.Info(fmt.Sprintf("Uploaded %d chunks to random single node", chunkCount))
		}

		log.Info("Action terminated")

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
		//select the local store for the given node
		//if there are more than one chunk, test only succeeds if all expected chunks are found
		allSuccess := true

		//all the chunk indexes which are supposed to be found for this node
		localChunks := conf.idToChunksMap[id]
		//for each expected chunk, check if it is in the local store
		for _, ch := range localChunks {
			//get the real chunk by the index in the index array
			chunk := conf.hashes[ch]
			log.Trace(fmt.Sprintf("node has chunk: %s:", chunk))
			//check if the expected chunk is indeed in the localstore
			var err error
			if *useMockStore {
				if globalStore == nil {
					return false, fmt.Errorf("Something went wrong; using mockStore enabled but globalStore is nil")
				}
				//use the globalStore if the mockStore should be used; in that case,
				//the complete localStore stack is bypassed for getting the chunk
				_, err = globalStore.Get(common.BytesToAddress(id.Bytes()), chunk)
			} else {
				//use the actual localstore
				lstore := stores[id]
				_, err = lstore.Get(chunk)
			}
			if err != nil {
				log.Warn(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
				allSuccess = false
			} else {
				log.Debug(fmt.Sprintf("Chunk %s IS FOUND for id %s", chunk, id))
			}
		}

		return allSuccess, nil
	}

	//for each tick, run the checks on all nodes
	timingTicker := time.NewTicker(time.Second * 1)
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
	log.Info("Simulation terminated")
	return nil
}

//the server func to start syncing
//issues `RequestSubscriptionMsg` to peers, based on po, by iterating over
//the kademlia's `EachBin` function.
//returns the number of subscriptions requested
func (r *TestRegistry) StartSyncing(ctx context.Context) (int, error) {
	var err error

	if log.Lvl(*loglevel) == log.LvlDebug {
		//PeerPot for this node
		addr := common.Bytes2Hex(r.addr.OAddr)
		pp := ppmap[addr]
		//call Healthy RPC
		h := r.delivery.overlay.Healthy(pp)
		//print info
		log.Debug(r.delivery.overlay.String())
		log.Debug(fmt.Sprintf("IS HEALTHY: %t", h.GotNN && h.KnowNN && h.Full))
	}

	kad, ok := r.delivery.overlay.(*network.Kademlia)
	if !ok {
		return 0, fmt.Errorf("Not a Kademlia!")
	}

	subCnt := 0
	//iterate over each bin and solicit needed subscription to bins
	kad.EachBin(r.addr.Over(), pof, 0, func(conn network.OverlayConn, po int) bool {
		//identify begin and start index of the bin(s) we want to subscribe to
		log.Debug(fmt.Sprintf("Requesting subscription by: registry %s from peer %s for bin: %d", r.addr.ID(), conf.addrToIdMap[string(conn.Address())], po))
		var histRange *Range
		if history {
			histRange = &Range{}
		}

		subCnt++
		err = r.RequestSubscription(conf.addrToIdMap[string(conn.Address())], NewStream("SYNC", FormatSyncBinKey(uint8(po)), live), histRange, Top)
		if err != nil {
			log.Error(fmt.Sprintf("Error in RequestSubsciption! %v", err))
			return false
		}
		return true

	})
	return subCnt, nil
}

//map chunk keys to addresses which are responsible
func mapKeysToNodes(conf *synctestConfig) {
	kmap := make(map[string][]int)
	nodemap := make(map[string][]int)
	//build a pot for chunk hashes
	np := pot.NewPot(nil, 0)
	indexmap := make(map[string]int)
	for i, a := range conf.addrs {
		indexmap[string(a)] = i
		np, _, _ = pot.Add(np, a, pof)
	}
	//for each address, run EachNeighbour on the chunk hashes pot to identify closest nodes
	log.Trace(fmt.Sprintf("Generated hash chunk(s): %v", conf.hashes))
	for i := 0; i < len(conf.hashes); i++ {
		pl := 256 //highest possible proximity
		var nns []int
		np.EachNeighbour([]byte(conf.hashes[i]), pof, func(val pot.Val, po int) bool {
			a := val.([]byte)
			if pl < 256 && pl != po {
				return false
			}
			if pl == 256 || pl == po {
				log.Trace(fmt.Sprintf("appending %s", conf.addrToIdMap[string(a)]))
				nns = append(nns, indexmap[string(a)])
				nodemap[string(a)] = append(nodemap[string(a)], i)
			}
			if pl == 256 && len(nns) >= testMinProxBinSize {
				//maxProxBinSize has been reached at this po, so save it
				//we will add all other nodes at the same po
				pl = po
			}
			return true
		})
		kmap[string(conf.hashes[i])] = nns
	}
	for addr, chunks := range nodemap {
		//this selects which chunks are expected to be found with the given node
		conf.idToChunksMap[conf.addrToIdMap[addr]] = chunks
	}
	log.Debug(fmt.Sprintf("Map of expected chunks by ID: %v", conf.idToChunksMap))
	conf.chunksToNodesMap = kmap
}

//upload a file(chunks) to a single local node store
func uploadFileToSingleNodeStore(id discover.NodeID, chunkCount int) ([]storage.Address, error) {
	log.Debug(fmt.Sprintf("Uploading to node id: %s", id))
	lstore := stores[id]
	size := chunkSize
	fileStore := storage.NewFileStore(lstore, storage.NewFileStoreParams())
	var rootAddrs []storage.Address
	for i := 0; i < chunkCount; i++ {
		ctx := context.TODO()
		rk, wait, err := fileStore.Store(ctx, io.LimitReader(crand.Reader, int64(size)), int64(size), false)
		if err != nil {
			return nil, err
		}
		err = wait(ctx)
		if err != nil {
			return nil, err
		}
		rootAddrs = append(rootAddrs, (rk))
	}

	return rootAddrs, nil
}

//initialize a network from a snapshot
func initNetWithSnapshot(nodeCount int) (*simulations.Network, error) {

	var a adapters.NodeAdapter
	//add the streamer service to the node adapter

	if *adapter == "exec" {
		dirname, err := ioutil.TempDir(".", "")
		if err != nil {
			return nil, err
		}
		a = adapters.NewExecAdapter(dirname)
	} else if *adapter == "tcp" {
		a = adapters.NewTCPAdapter(services)
	} else if *adapter == "sim" {
		a = adapters.NewSimAdapter(services)
	}

	log.Info("Setting up Snapshot network")

	net := simulations.NewNetwork(a, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "streamer",
	})

	f, err := os.Open(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	jsonbyte, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var snap simulations.Snapshot
	err = json.Unmarshal(jsonbyte, &snap)
	if err != nil {
		return nil, err
	}

	//the snapshot probably has the property EnableMsgEvents not set
	//just in case, set it to true!
	//(we need this to wait for messages before uploading)
	for _, n := range snap.Nodes {
		n.Node.Config.EnableMsgEvents = true
	}

	log.Info("Waiting for p2p connections to be established...")

	//now we can load the snapshot
	err = net.Load(&snap)
	if err != nil {
		return nil, err
	}
	log.Info("Snapshot loaded")
	return net, nil
}

//we want to wait for subscriptions to be established before uploading to test
//that live syncing is working correctly
func watchSubscriptionEvents(ctx context.Context, id discover.NodeID, client *rpc.Client, errc chan error, quitC chan struct{}) (doneC <-chan struct{}) {
	events := make(chan *p2p.PeerEvent)
	sub, err := client.Subscribe(context.Background(), "admin", events, "peerEvents")
	if err != nil {
		log.Error(err.Error())
		errc <- fmt.Errorf("error getting peer events for node %v: %s", id, err)
		return
	}
	c := make(chan struct{})

	go func() {
		defer func() {
			log.Trace("watch subscription events: unsubscribe", "id", id)
			sub.Unsubscribe()
			close(c)
		}()

		for {
			select {
			case <-quitC:
				return
			case <-ctx.Done():
				select {
				case errc <- ctx.Err():
				case <-quitC:
				}
				return
			case e := <-events:
				//just catch SubscribeMsg
				if e.Type == p2p.PeerEventTypeMsgRecv && e.Protocol == "stream" && e.MsgCode != nil && *e.MsgCode == 4 {
					errc <- nil
				}
			case err := <-sub.Err():
				if err != nil {
					select {
					case errc <- fmt.Errorf("error getting peer events for node %v: %v", id, err):
					case <-quitC:
					}
					return
				}
			}
		}
	}()
	return c
}

//create a local store for the given node
func createTestLocalStorageForId(id discover.NodeID, addr *network.BzzAddr) (storage.ChunkStore, error) {
	var datadir string
	var err error
	datadir, err = ioutil.TempDir("", fmt.Sprintf("syncer-test-%s", id.TerminalString()))
	if err != nil {
		return nil, err
	}
	datadirs[id] = datadir
	var store storage.ChunkStore
	params := storage.NewDefaultLocalStoreParams()
	params.ChunkDbPath = datadir
	params.BaseKey = addr.Over()
	store, err = storage.NewTestLocalStoreForAddr(params)
	if err != nil {
		return nil, err
	}
	return store, nil
}
