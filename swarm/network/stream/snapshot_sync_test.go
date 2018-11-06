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
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	mockdb "github.com/ethereum/go-ethereum/swarm/storage/mock/db"
)

const MaxTimeout = 600

type synctestConfig struct {
	addrs         [][]byte
	hashes        []storage.Address
	idToChunksMap map[enode.ID][]int
	//chunksToNodesMap map[string][]int
	addrToIDMap map[string]enode.ID
}

const (
	// EventTypeNode is the type of event emitted when a node is either
	// created, started or stopped
	EventTypeChunkCreated   simulations.EventType = "chunkCreated"
	EventTypeChunkOffered   simulations.EventType = "chunkOffered"
	EventTypeChunkWanted    simulations.EventType = "chunkWanted"
	EventTypeChunkDelivered simulations.EventType = "chunkDelivered"
	EventTypeChunkArrived   simulations.EventType = "chunkArrived"
	EventTypeSimTerminated  simulations.EventType = "simTerminated"
)

// Tests in this file should not request chunks from peers.
// This function will panic indicating that there is a problem if request has been made.
func dummyRequestFromPeers(_ context.Context, req *network.Request) (*enode.ID, chan struct{}, error) {
	panic(fmt.Sprintf("unexpected request: address %s, source %s", req.Addr.String(), req.Source.String()))
}

//This test is a syncing test for nodes.
//One node is randomly selected to be the pivot node.
//A configurable number of chunks and nodes can be
//provided to the test, the number of chunks is uploaded
//to the pivot node, and we check that nodes get the chunks
//they are expected to store based on the syncing protocol.
//Number of chunks and nodes can be provided via commandline too.
func TestSyncingViaGlobalSync(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("TRAVIS") == "true" {
		t.Skip("Flaky on mac on travis")
	}
	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		log.Info(fmt.Sprintf("Running test with %d chunks and %d nodes...", *chunks, *nodes))
		testSyncingViaGlobalSync(t, *chunks, *nodes)
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
				testSyncingViaGlobalSync(t, chnk, n)
			}
		}
	}
}

func TestSyncingViaDirectSubscribe(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("TRAVIS") == "true" {
		t.Skip("Flaky on mac on travis")
	}
	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		log.Info(fmt.Sprintf("Running test with %d chunks and %d nodes...", *chunks, *nodes))
		err := testSyncingViaDirectSubscribe(t, *chunks, *nodes)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		var nodeCnt []int
		var chnkCnt []int
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			chnkCnt = []int{1, 8, 32, 256, 1024}
			nodeCnt = []int{32, 16}
		} else {
			//default test
			chnkCnt = []int{4, 32}
			nodeCnt = []int{32, 16}
		}
		for _, chnk := range chnkCnt {
			for _, n := range nodeCnt {
				log.Info(fmt.Sprintf("Long running test with %d chunks and %d nodes...", chnk, n))
				err := testSyncingViaDirectSubscribe(t, chnk, n)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

var simServiceMap = map[string]simulation.ServiceFunc{
	"streamer": streamerFunc,
}

func streamerFunc(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
	n := ctx.Config.Node()
	addr := network.NewAddr(n)
	store, datadir, err := createTestLocalStorageForID(n.ID(), addr)
	if err != nil {
		return nil, nil, err
	}
	bucket.Store(bucketKeyStore, store)
	localStore := store.(*storage.LocalStore)
	netStore, err := storage.NewNetStore(localStore, nil)
	if err != nil {
		return nil, nil, err
	}
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	delivery := NewDelivery(kad, netStore)
	netStore.NewNetFetcherFunc = network.NewFetcherFactory(dummyRequestFromPeers, true).New

	r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
		Retrieval:       RetrievalDisabled,
		Syncing:         SyncingAutoSubscribe,
		SyncUpdateDelay: 3 * time.Second,
	})

	bucket.Store(bucketKeyRegistry, r)

	cleanup = func() {
		os.RemoveAll(datadir)
		netStore.Close()
		r.Close()
	}

	return r, cleanup, nil

}

func testSyncingViaGlobalSync(t *testing.T, chunkCount int, nodeCount int) {
	sim := simulation.New(simServiceMap)
	defer sim.Close()

	log.Info("Initializing test config")

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelSimRun()

	if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
		t.Fatal(err)
	}

	disconnections := sim.PeerEvents(
		context.Background(),
		sim.NodeIDs(),
		simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeDrop),
	)

	go func() {
		for d := range disconnections {
			log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
			t.Fatal("unexpected disconnect")
			cancelSimRun()
		}
	}()

	result := runSim(conf, ctx, sim, chunkCount)

	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")
}

func runSim(conf *synctestConfig, ctx context.Context, sim *simulation.Simulation, chunkCount int) simulation.Result {

	return sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := n.Bytes()
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on enode.ID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		//get the node at that index
		//this is the node selected for upload
		node := sim.RandomUpNode()
		item, ok := sim.NodeItem(node.ID, bucketKeyStore)
		if !ok {
			return fmt.Errorf("No localstore")
		}
		lstore := item.(*storage.LocalStore)
		hashes, err := uploadFileToSingleNodeStore(node.ID, chunkCount, lstore)
		if err != nil {
			return err
		}
		for _, h := range hashes {
			evt := &simulations.Event{
				Type: EventTypeChunkCreated,
				Node: sim.Net.GetNode(node.ID),
				Data: h.String(),
			}
			sim.Net.Events().Send(evt)
		}
		conf.hashes = append(conf.hashes, hashes...)
		mapKeysToNodes(conf)

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
		var gDir string
		var globalStore *mockdb.GlobalStore
		if *useMockStore {
			gDir, globalStore, err = createGlobalStore()
			if err != nil {
				return fmt.Errorf("Something went wrong; using mockStore enabled but globalStore is nil")
			}
			defer func() {
				os.RemoveAll(gDir)
				err := globalStore.Close()
				if err != nil {
					log.Error("Error closing global store! %v", "err", err)
				}
			}()
		}
	REPEAT:
		for {
			for _, id := range nodeIDs {
				//for each expected chunk, check if it is in the local store
				localChunks := conf.idToChunksMap[id]
				for _, ch := range localChunks {
					//get the real chunk by the index in the index array
					chunk := conf.hashes[ch]
					log.Trace(fmt.Sprintf("node has chunk: %s:", chunk))
					//check if the expected chunk is indeed in the localstore
					var err error
					if *useMockStore {
						//use the globalStore if the mockStore should be used; in that case,
						//the complete localStore stack is bypassed for getting the chunk
						_, err = globalStore.Get(common.BytesToAddress(id.Bytes()), chunk)
					} else {
						//use the actual localstore
						item, ok := sim.NodeItem(id, bucketKeyStore)
						if !ok {
							return fmt.Errorf("Error accessing localstore")
						}
						lstore := item.(*storage.LocalStore)
						_, err = lstore.Get(ctx, chunk)
					}
					if err != nil {
						log.Debug(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
						// Do not get crazy with logging the warn message
						time.Sleep(500 * time.Millisecond)
						continue REPEAT
					}
					evt := &simulations.Event{
						Type: EventTypeChunkArrived,
						Node: sim.Net.GetNode(id),
						Data: chunk.String(),
					}
					sim.Net.Events().Send(evt)
					log.Debug(fmt.Sprintf("Chunk %s IS FOUND for id %s", chunk, id))
				}
			}
			return nil
		}
	})
}

/*
The test generates the given number of chunks

For every chunk generated, the nearest node addresses
are identified, we verify that the nodes closer to the
chunk addresses actually do have the chunks in their local stores.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. The snapshot should have 'streamer' in its service list.
*/
func testSyncingViaDirectSubscribe(t *testing.T, chunkCount int, nodeCount int) error {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			n := ctx.Config.Node()
			addr := network.NewAddr(n)
			store, datadir, err := createTestLocalStorageForID(n.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(dummyRequestFromPeers, true).New

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval: RetrievalDisabled,
				Syncing:   SyncingRegisterOnly,
			})
			bucket.Store(bucketKeyRegistry, r)

			fileStore := storage.NewFileStore(netStore, storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			cleanup = func() {
				os.RemoveAll(datadir)
				netStore.Close()
				r.Close()
			}

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelSimRun()

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		return err
	}

	if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
		return err
	}

	disconnections := sim.PeerEvents(
		context.Background(),
		sim.NodeIDs(),
		simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeDrop),
	)

	go func() {
		for d := range disconnections {
			log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
			t.Fatal("unexpected disconnect")
			cancelSimRun()
		}
	}()

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := n.Bytes()
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on enode.ID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		var subscriptionCount int

		filter := simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeMsgRecv).Protocol("stream").MsgCode(4)
		eventC := sim.PeerEvents(ctx, nodeIDs, filter)

		for j, node := range nodeIDs {
			log.Trace(fmt.Sprintf("Start syncing subscriptions: %d", j))
			//start syncing!
			item, ok := sim.NodeItem(node, bucketKeyRegistry)
			if !ok {
				return fmt.Errorf("No registry")
			}
			registry := item.(*Registry)

			var cnt int
			cnt, err = startSyncing(registry, conf)
			if err != nil {
				return err
			}
			//increment the number of subscriptions we need to wait for
			//by the count returned from startSyncing (SYNC subscriptions)
			subscriptionCount += cnt
		}

		for e := range eventC {
			if e.Error != nil {
				return e.Error
			}
			subscriptionCount--
			if subscriptionCount == 0 {
				break
			}
		}
		//select a random node for upload
		node := sim.RandomUpNode()
		item, ok := sim.NodeItem(node.ID, bucketKeyStore)
		if !ok {
			return fmt.Errorf("No localstore")
		}
		lstore := item.(*storage.LocalStore)
		hashes, err := uploadFileToSingleNodeStore(node.ID, chunkCount, lstore)
		if err != nil {
			return err
		}
		conf.hashes = append(conf.hashes, hashes...)
		mapKeysToNodes(conf)

		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			return err
		}

		var gDir string
		var globalStore *mockdb.GlobalStore
		if *useMockStore {
			gDir, globalStore, err = createGlobalStore()
			if err != nil {
				return fmt.Errorf("Something went wrong; using mockStore enabled but globalStore is nil")
			}
			defer os.RemoveAll(gDir)
		}
		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
	REPEAT:
		for {
			for _, id := range nodeIDs {
				//for each expected chunk, check if it is in the local store
				localChunks := conf.idToChunksMap[id]
				for _, ch := range localChunks {
					//get the real chunk by the index in the index array
					chunk := conf.hashes[ch]
					log.Trace(fmt.Sprintf("node has chunk: %s:", chunk))
					//check if the expected chunk is indeed in the localstore
					var err error
					if *useMockStore {
						//use the globalStore if the mockStore should be used; in that case,
						//the complete localStore stack is bypassed for getting the chunk
						_, err = globalStore.Get(common.BytesToAddress(id.Bytes()), chunk)
					} else {
						//use the actual localstore
						item, ok := sim.NodeItem(id, bucketKeyStore)
						if !ok {
							return fmt.Errorf("Error accessing localstore")
						}
						lstore := item.(*storage.LocalStore)
						_, err = lstore.Get(ctx, chunk)
					}
					if err != nil {
						log.Debug(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
						// Do not get crazy with logging the warn message
						time.Sleep(500 * time.Millisecond)
						continue REPEAT
					}
					log.Debug(fmt.Sprintf("Chunk %s IS FOUND for id %s", chunk, id))
				}
			}
			return nil
		}
	})

	if result.Error != nil {
		return result.Error
	}

	log.Info("Simulation ended")
	return nil
}

//the server func to start syncing
//issues `RequestSubscriptionMsg` to peers, based on po, by iterating over
//the kademlia's `EachBin` function.
//returns the number of subscriptions requested
func startSyncing(r *Registry, conf *synctestConfig) (int, error) {
	var err error
	kad := r.delivery.kad
	subCnt := 0
	//iterate over each bin and solicit needed subscription to bins
	kad.EachBin(r.addr[:], pof, 0, func(conn *network.Peer, po int) bool {
		//identify begin and start index of the bin(s) we want to subscribe to
		subCnt++
		err = r.RequestSubscription(conf.addrToIDMap[string(conn.Address())], NewStream("SYNC", FormatSyncBinKey(uint8(po)), true), NewRange(0, 0), High)
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
	nodemap := make(map[string][]int)
	//build a pot for chunk hashes
	np := pot.NewPot(nil, 0)
	indexmap := make(map[string]int)
	for i, a := range conf.addrs {
		indexmap[string(a)] = i
		np, _, _ = pot.Add(np, a, pof)
	}

	var kadMinProxSize = 2

	ppmap := network.NewPeerPotMap(kadMinProxSize, conf.addrs)

	//for each address, run EachNeighbour on the chunk hashes pot to identify closest nodes
	log.Trace(fmt.Sprintf("Generated hash chunk(s): %v", conf.hashes))
	for i := 0; i < len(conf.hashes); i++ {
		var a []byte
		np.EachNeighbour([]byte(conf.hashes[i]), pof, func(val pot.Val, po int) bool {
			// take the first address
			a = val.([]byte)
			return false
		})

		nns := ppmap[common.Bytes2Hex(a)].NNSet
		nns = append(nns, a)

		for _, p := range nns {
			nodemap[string(p)] = append(nodemap[string(p)], i)
		}
	}
	for addr, chunks := range nodemap {
		//this selects which chunks are expected to be found with the given node
		conf.idToChunksMap[conf.addrToIDMap[addr]] = chunks
	}
	log.Debug(fmt.Sprintf("Map of expected chunks by ID: %v", conf.idToChunksMap))
}

//upload a file(chunks) to a single local node store
func uploadFileToSingleNodeStore(id enode.ID, chunkCount int, lstore *storage.LocalStore) ([]storage.Address, error) {
	log.Debug(fmt.Sprintf("Uploading to node id: %s", id))
	fileStore := storage.NewFileStore(lstore, storage.NewFileStoreParams())
	size := chunkSize
	var rootAddrs []storage.Address
	for i := 0; i < chunkCount; i++ {
		rk, wait, err := fileStore.Store(context.TODO(), io.LimitReader(crand.Reader, int64(size)), int64(size), false)
		if err != nil {
			return nil, err
		}
		err = wait(context.TODO())
		if err != nil {
			return nil, err
		}
		rootAddrs = append(rootAddrs, (rk))
	}

	return rootAddrs, nil
}
