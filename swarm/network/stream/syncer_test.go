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
	"io/ioutil"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	mockmem "github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

const dataChunkCount = 200

func TestSyncerSimulation(t *testing.T) {
	testSyncBetweenNodes(t, 2, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 4, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 8, dataChunkCount, true, 1)
	testSyncBetweenNodes(t, 16, dataChunkCount, true, 1)
}

func createMockStore(globalStore mock.GlobalStorer, id enode.ID, addr *network.BzzAddr) (lstore storage.ChunkStore, datadir string, err error) {
	address := common.BytesToAddress(id.Bytes())
	mockStore := globalStore.NewNodeStore(address)
	params := storage.NewDefaultLocalStoreParams()

	datadir, err = ioutil.TempDir("", "localMockStore-"+id.TerminalString())
	if err != nil {
		return nil, "", err
	}
	params.Init(datadir)
	params.BaseKey = addr.Over()
	lstore, err = storage.NewLocalStore(params, mockStore)
	if err != nil {
		return nil, "", err
	}
	return lstore, datadir, nil
}

func testSyncBetweenNodes(t *testing.T, nodes, chunkCount int, skipCheck bool, po uint8) {

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			var store storage.ChunkStore
			var datadir string

			node := ctx.Config.Node()
			addr := network.NewAddr(node)
			//hack to put addresses in same space
			addr.OAddr[0] = byte(0)

			if *useMockStore {
				store, datadir, err = createMockStore(mockmem.NewGlobalStore(), node.ID(), addr)
			} else {
				store, datadir, err = createTestLocalStorageForID(node.ID(), addr)
			}
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				store.Close()
				os.RemoveAll(datadir)
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyDB, netStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

			bucket.Store(bucketKeyDelivery, delivery)

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval: RetrievalDisabled,
				Syncing:   SyncingAutoSubscribe,
				SkipCheck: skipCheck,
			}, nil)

			fileStore := storage.NewFileStore(netStore, storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	// create context for simulation run
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel should come before defer simulation teardown
	defer cancel()

	_, err := sim.AddNodesAndConnectChain(nodes)
	if err != nil {
		t.Fatal(err)
	}
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) (err error) {
		nodeIDs := sim.UpNodeIDs()

		nodeIndex := make(map[enode.ID]int)
		for i, id := range nodeIDs {
			nodeIndex[id] = i
		}

		disconnections := sim.PeerEvents(
			context.Background(),
			sim.NodeIDs(),
			simulation.NewPeerEventsFilter().Drop(),
		)

		var disconnected atomic.Value
		go func() {
			for d := range disconnections {
				if d.Error != nil {
					log.Error("peer drop", "node", d.NodeID, "peer", d.PeerID)
					disconnected.Store(true)
				}
			}
		}()
		defer func() {
			if err != nil {
				if yes, ok := disconnected.Load().(bool); ok && yes {
					err = errors.New("disconnect events received")
				}
			}
		}()

		// each node Subscribes to each other's swarmChunkServerStreamName
		for j := 0; j < nodes-1; j++ {
			id := nodeIDs[j]
			client, err := sim.Net.GetNode(id).Client()
			if err != nil {
				t.Fatal(err)
			}
			sid := nodeIDs[j+1]
			client.CallContext(ctx, nil, "stream_subscribeStream", sid, NewStream("SYNC", FormatSyncBinKey(1), false), NewRange(0, 0), Top)
			if err != nil {
				return err
			}
			if j > 0 || nodes == 2 {
				item, ok := sim.NodeItem(nodeIDs[j], bucketKeyFileStore)
				if !ok {
					return fmt.Errorf("No filestore")
				}
				fileStore := item.(*storage.FileStore)
				size := chunkCount * chunkSize
				_, wait, err := fileStore.Store(ctx, testutil.RandomReader(j, size), int64(size), false)
				if err != nil {
					t.Fatal(err.Error())
				}
				wait(ctx)
			}
		}
		// here we distribute chunks of a random file into stores 1...nodes
		if _, err := sim.WaitTillHealthy(ctx); err != nil {
			return err
		}

		// collect hashes in po 1 bin for each node
		hashes := make([][]storage.Address, nodes)
		totalHashes := 0
		hashCounts := make([]int, nodes)
		for i := nodes - 1; i >= 0; i-- {
			if i < nodes-1 {
				hashCounts[i] = hashCounts[i+1]
			}
			item, ok := sim.NodeItem(nodeIDs[i], bucketKeyDB)
			if !ok {
				return fmt.Errorf("No DB")
			}
			netStore := item.(*storage.NetStore)
			netStore.Iterator(0, math.MaxUint64, po, func(addr storage.Address, index uint64) bool {
				hashes[i] = append(hashes[i], addr)
				totalHashes++
				hashCounts[i]++
				return true
			})
		}
		var total, found int
		for _, node := range nodeIDs {
			i := nodeIndex[node]

			for j := i; j < nodes; j++ {
				total += len(hashes[j])
				for _, key := range hashes[j] {
					item, ok := sim.NodeItem(nodeIDs[j], bucketKeyDB)
					if !ok {
						return fmt.Errorf("No DB")
					}
					db := item.(*storage.NetStore)
					_, err := db.Get(ctx, key)
					if err == nil {
						found++
					}
				}
			}
			log.Debug("sync check", "node", node, "index", i, "bin", po, "found", found, "total", total)
		}
		if total == found && total > 0 {
			return nil
		}
		return fmt.Errorf("Total not equallying found: total is %d", total)
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

//TestSameVersionID just checks that if the version is not changed,
//then streamer peers see each other
func TestSameVersionID(t *testing.T) {
	//test version ID
	v := uint(1)
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			var store storage.ChunkStore
			var datadir string

			node := ctx.Config.Node()
			addr := network.NewAddr(node)

			store, datadir, err = createTestLocalStorageForID(node.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				store.Close()
				os.RemoveAll(datadir)
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyDB, netStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

			bucket.Store(bucketKeyDelivery, delivery)

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval: RetrievalDisabled,
				Syncing:   SyncingAutoSubscribe,
			}, nil)
			//assign to each node the same version ID
			r.spec.Version = v

			bucket.Store(bucketKeyRegistry, r)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	//connect just two nodes
	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectChain(2)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")
	ctx := context.Background()
	//make sure they have time to connect
	time.Sleep(200 * time.Millisecond)
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//get the pivot node's filestore
		nodes := sim.UpNodeIDs()

		item, ok := sim.NodeItem(nodes[0], bucketKeyRegistry)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		registry := item.(*Registry)

		//the peers should connect, thus getting the peer should not return nil
		if registry.getPeer(nodes[1]) == nil {
			t.Fatal("Expected the peer to not be nil, but it is")
		}
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")
}

//TestDifferentVersionID proves that if the streamer protocol version doesn't match,
//then the peers are not connected at streamer level
func TestDifferentVersionID(t *testing.T) {
	//create a variable to hold the version ID
	v := uint(0)
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			var store storage.ChunkStore
			var datadir string

			node := ctx.Config.Node()
			addr := network.NewAddr(node)

			store, datadir, err = createTestLocalStorageForID(node.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				store.Close()
				os.RemoveAll(datadir)
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyDB, netStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

			bucket.Store(bucketKeyDelivery, delivery)

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval: RetrievalDisabled,
				Syncing:   SyncingAutoSubscribe,
			}, nil)

			//increase the version ID for each node
			v++
			r.spec.Version = v

			bucket.Store(bucketKeyRegistry, r)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	//connect the nodes
	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectChain(2)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")
	ctx := context.Background()
	//make sure they have time to connect
	time.Sleep(200 * time.Millisecond)
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//get the pivot node's filestore
		nodes := sim.UpNodeIDs()

		item, ok := sim.NodeItem(nodes[0], bucketKeyRegistry)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		registry := item.(*Registry)

		//getting the other peer should fail due to the different version numbers
		if registry.getPeer(nodes[1]) != nil {
			t.Fatal("Expected the peer to be nil, but it is not")
		}
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")

}
