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
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	mockmem "github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

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

	if testutil.RaceEnabled {
		t.Skip("Segfaults on Travis with -race")
	}

	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		log.Info(fmt.Sprintf("Running test with %d chunks and %d nodes...", *chunks, *nodes))
		testSyncingViaGlobalSync(t, *chunks, *nodes)
	} else {
		chunkCounts := []int{4, 32}
		nodeCounts := []int{32, 16}

		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			chunkCounts = []int{64, 128}
			nodeCounts = []int{32, 64}
		}

		for _, chunkCount := range chunkCounts {
			for _, n := range nodeCounts {
				log.Info(fmt.Sprintf("Long running test with %d chunks and %d nodes...", chunkCount, n))
				testSyncingViaGlobalSync(t, chunkCount, n)
			}
		}
	}
}

var simServiceMap = map[string]simulation.ServiceFunc{
	"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
		addr, netStore, delivery, clean, err := newNetStoreAndDeliveryWithRequestFunc(ctx, bucket, dummyRequestFromPeers)
		if err != nil {
			return nil, nil, err
		}

		store := state.NewInmemoryStore()

		r := NewRegistry(addr.ID(), delivery, netStore, store, &RegistryOptions{
			Retrieval:       RetrievalDisabled,
			Syncing:         SyncingAutoSubscribe,
			SyncUpdateDelay: 3 * time.Second,
		}, nil)

		bucket.Store(bucketKeyRegistry, r)

		cleanup = func() {
			r.Close()
			clean()
		}

		return r, cleanup, nil
	},
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

	if _, err := sim.WaitTillHealthy(ctx); err != nil {
		t.Fatal(err)
	}

	result := runSim(conf, ctx, sim, chunkCount)

	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")
}

func runSim(conf *synctestConfig, ctx context.Context, sim *simulation.Simulation, chunkCount int) simulation.Result {

	return sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) (err error) {
		disconnected := watchDisconnections(ctx, sim)
		defer func() {
			if err != nil && disconnected.bool() {
				err = errors.New("disconnect events received")
			}
		}()

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
		node := sim.Net.GetRandomUpNode()
		item, ok := sim.NodeItem(node.ID(), bucketKeyStore)
		if !ok {
			return fmt.Errorf("No localstore")
		}
		lstore := item.(*storage.LocalStore)
		hashes, err := uploadFileToSingleNodeStore(node.ID(), chunkCount, lstore)
		if err != nil {
			return err
		}
		for _, h := range hashes {
			evt := &simulations.Event{
				Type: EventTypeChunkCreated,
				Node: sim.Net.GetNode(node.ID()),
				Data: h.String(),
			}
			sim.Net.Events().Send(evt)
		}
		conf.hashes = append(conf.hashes, hashes...)
		mapKeysToNodes(conf)

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
		var globalStore mock.GlobalStorer
		if *useMockStore {
			globalStore = mockmem.NewGlobalStore()
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

	ppmap := network.NewPeerPotMap(network.NewKadParams().NeighbourhoodSize, conf.addrs)

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
		rk, wait, err := fileStore.Store(context.TODO(), testutil.RandomReader(i, size), int64(size), false)
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
