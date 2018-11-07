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
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	pq "github.com/ethereum/go-ethereum/swarm/network/priorityqueue"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

//Tests initializing a retrieve request
func TestStreamerRetrieveRequest(t *testing.T) {
	regOpts := &RegistryOptions{
		Retrieval: RetrievalClientOnly,
		Syncing:   SyncingDisabled,
	}
	tester, streamer, _, teardown, err := newStreamerTester(t, regOpts)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	node := tester.Nodes[0]

	ctx := context.Background()
	req := network.NewRequest(
		storage.Address(hash0[:]),
		true,
		&sync.Map{},
	)
	streamer.delivery.RequestFromPeers(ctx, req)

	stream := NewStream(swarmChunkServerStreamName, "", true)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Expects: []p2ptest.Expect{
			{ //start expecting a subscription for RETRIEVE_REQUEST due to `RetrievalClientOnly`
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  nil,
					Priority: Top,
				},
				Peer: node.ID(),
			},
			{ //expect a retrieve request message for the given hash
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Addr:      hash0[:],
					SkipCheck: true,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

//Test requesting a chunk from a peer then issuing a "empty" OfferedHashesMsg (no hashes available yet)
//Should time out as the peer does not have the chunk (no syncing happened previously)
func TestStreamerUpstreamRetrieveRequestMsgExchangeWithoutStore(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, &RegistryOptions{
		Retrieval: RetrievalEnabled,
		Syncing:   SyncingDisabled, //do no syncing
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	node := tester.Nodes[0]

	chunk := storage.NewChunk(storage.Address(hash0[:]), nil)

	peer := streamer.getPeer(node.ID())

	stream := NewStream(swarmChunkServerStreamName, "", true)
	//simulate pre-subscription to RETRIEVE_REQUEST stream on peer
	peer.handleSubscribeMsg(context.TODO(), &SubscribeMsg{
		Stream:   stream,
		History:  nil,
		Priority: Top,
	})

	//test the exchange
	err = tester.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			{ //first expect a subscription to the RETRIEVE_REQUEST stream
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  nil,
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
	}, p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			{ //then the actual RETRIEVE_REQUEST....
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Addr: chunk.Address()[:],
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{ //to which the peer responds with offered hashes
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: nil,
					Hashes:        nil,
					From:          0,
					To:            0,
				},
				Peer: node.ID(),
			},
		},
	})

	//should fail with a timeout as the peer we are requesting
	//the chunk from does not have the chunk
	expectedError := `exchange #1 "RetrieveRequestMsg": timed out`
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, err)
	}
}

// upstream request server receives a retrieve Request and responds with
// offered hashes or delivery if skipHash is set to true
func TestStreamerUpstreamRetrieveRequestMsgExchange(t *testing.T) {
	tester, streamer, localStore, teardown, err := newStreamerTester(t, &RegistryOptions{
		Retrieval: RetrievalEnabled,
		Syncing:   SyncingDisabled,
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	node := tester.Nodes[0]

	peer := streamer.getPeer(node.ID())

	stream := NewStream(swarmChunkServerStreamName, "", true)

	peer.handleSubscribeMsg(context.TODO(), &SubscribeMsg{
		Stream:   stream,
		History:  nil,
		Priority: Top,
	})

	hash := storage.Address(hash0[:])
	chunk := storage.NewChunk(hash, hash)
	err = localStore.Put(context.TODO(), chunk)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  nil,
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
	}, p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Addr: hash,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					Hashes: hash,
					From:   0,
					// TODO: why is this 32???
					To:     32,
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	hash = storage.Address(hash1[:])
	chunk = storage.NewChunk(hash, hash1[:])
	err = localStore.Put(context.TODO(), chunk)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Addr:      hash,
					SkipCheck: true,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 6,
				Msg: &ChunkDeliveryMsg{
					Addr:  hash,
					SData: hash,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

// if there is one peer in the Kademlia, RequestFromPeers should return it
func TestRequestFromPeers(t *testing.T) {
	dummyPeerID := enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8")

	addr := network.RandomAddr()
	to := network.NewKademlia(addr.OAddr, network.NewKadParams())
	delivery := NewDelivery(to, nil)
	protocolsPeer := protocols.NewPeer(p2p.NewPeer(dummyPeerID, "dummy", nil), nil, nil)
	peer := network.NewPeer(&network.BzzPeer{
		BzzAddr:   network.RandomAddr(),
		LightNode: false,
		Peer:      protocolsPeer,
	}, to)
	to.On(peer)
	r := NewRegistry(addr.ID(), delivery, nil, nil, nil)

	// an empty priorityQueue has to be created to prevent a goroutine being called after the test has finished
	sp := &Peer{
		Peer:     protocolsPeer,
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: r,
	}
	r.setPeer(sp)
	req := network.NewRequest(
		storage.Address(hash0[:]),
		true,
		&sync.Map{},
	)
	ctx := context.Background()
	id, _, err := delivery.RequestFromPeers(ctx, req)

	if err != nil {
		t.Fatal(err)
	}
	if *id != dummyPeerID {
		t.Fatalf("Expected an id, got %v", id)
	}
}

// RequestFromPeers should not return light nodes
func TestRequestFromPeersWithLightNode(t *testing.T) {
	dummyPeerID := enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8")

	addr := network.RandomAddr()
	to := network.NewKademlia(addr.OAddr, network.NewKadParams())
	delivery := NewDelivery(to, nil)

	protocolsPeer := protocols.NewPeer(p2p.NewPeer(dummyPeerID, "dummy", nil), nil, nil)
	// setting up a lightnode
	peer := network.NewPeer(&network.BzzPeer{
		BzzAddr:   network.RandomAddr(),
		LightNode: true,
		Peer:      protocolsPeer,
	}, to)
	to.On(peer)
	r := NewRegistry(addr.ID(), delivery, nil, nil, nil)
	// an empty priorityQueue has to be created to prevent a goroutine being called after the test has finished
	sp := &Peer{
		Peer:     protocolsPeer,
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: r,
	}
	r.setPeer(sp)

	req := network.NewRequest(
		storage.Address(hash0[:]),
		true,
		&sync.Map{},
	)

	ctx := context.Background()
	// making a request which should return with "no peer found"
	_, _, err := delivery.RequestFromPeers(ctx, req)

	expectedError := "no peer found"
	if err.Error() != expectedError {
		t.Fatalf("expected '%v', got %v", expectedError, err)
	}
}

func TestStreamerDownstreamChunkDeliveryMsgExchange(t *testing.T) {
	tester, streamer, localStore, teardown, err := newStreamerTester(t, &RegistryOptions{
		Retrieval: RetrievalDisabled,
		Syncing:   SyncingDisabled,
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterClientFunc("foo", func(p *Peer, t string, live bool) (Client, error) {
		return &testClient{
			t: t,
		}, nil
	})

	node := tester.Nodes[0]

	//subscribe to custom stream
	stream := NewStream("foo", "", true)
	err = streamer.Subscribe(node.ID(), stream, NewRange(5, 8), Top)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	chunkKey := hash0[:]
	chunkData := hash1[:]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Expects: []p2ptest.Expect{
			{ //first expect subscription to the custom stream...
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  NewRange(5, 8),
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
	},
		p2ptest.Exchange{
			Label: "ChunkDelivery message",
			Triggers: []p2ptest.Trigger{
				{ //...then trigger a chunk delivery for the given chunk from peer in order for
					//local node to get the chunk delivered
					Code: 6,
					Msg: &ChunkDeliveryMsg{
						Addr:  chunkKey,
						SData: chunkData,
					},
					Peer: node.ID(),
				},
			},
		})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// wait for the chunk to get stored
	storedChunk, err := localStore.Get(ctx, chunkKey)
	for err != nil {
		select {
		case <-ctx.Done():
			t.Fatalf("Chunk is not in localstore after timeout, err: %v", err)
		default:
		}
		storedChunk, err = localStore.Get(ctx, chunkKey)
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !bytes.Equal(storedChunk.Data(), chunkData) {
		t.Fatal("Retrieved chunk has different data than original")
	}

}

func TestDeliveryFromNodes(t *testing.T) {
	testDeliveryFromNodes(t, 2, 1, dataChunkCount, true)
	testDeliveryFromNodes(t, 2, 1, dataChunkCount, false)
	testDeliveryFromNodes(t, 4, 1, dataChunkCount, true)
	testDeliveryFromNodes(t, 4, 1, dataChunkCount, false)
	testDeliveryFromNodes(t, 8, 1, dataChunkCount, true)
	testDeliveryFromNodes(t, 8, 1, dataChunkCount, false)
	testDeliveryFromNodes(t, 16, 1, dataChunkCount, true)
	testDeliveryFromNodes(t, 16, 1, dataChunkCount, false)
}

func testDeliveryFromNodes(t *testing.T, nodes, conns, chunkCount int, skipCheck bool) {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			node := ctx.Config.Node()
			addr := network.NewAddr(node)
			store, datadir, err := createTestLocalStorageForID(node.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				os.RemoveAll(datadir)
				store.Close()
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}

			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				SkipCheck: skipCheck,
				Syncing:   SyncingDisabled,
				Retrieval: RetrievalEnabled,
			})
			bucket.Store(bucketKeyRegistry, r)

			fileStore := storage.NewFileStore(netStore, storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectChain(nodes)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")
	ctx := context.Background()
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		//determine the pivot node to be the first node of the simulation
		sim.SetPivotNode(nodeIDs[0])
		//distribute chunks of a random file into Stores of nodes 1 to nodes
		//we will do this by creating a file store with an underlying round-robin store:
		//the file store will create a hash for the uploaded file, but every chunk will be
		//distributed to different nodes via round-robin scheduling
		log.Debug("Writing file to round-robin file store")
		//to do this, we create an array for chunkstores (length minus one, the pivot node)
		stores := make([]storage.ChunkStore, len(nodeIDs)-1)
		//we then need to get all stores from the sim....
		lStores := sim.NodesItems(bucketKeyStore)
		i := 0
		//...iterate the buckets...
		for id, bucketVal := range lStores {
			//...and remove the one which is the pivot node
			if id == *sim.PivotNodeID() {
				continue
			}
			//the other ones are added to the array...
			stores[i] = bucketVal.(storage.ChunkStore)
			i++
		}
		//...which then gets passed to the round-robin file store
		roundRobinFileStore := storage.NewFileStore(newRoundRobinStore(stores...), storage.NewFileStoreParams())
		//now we can actually upload a (random) file to the round-robin store
		size := chunkCount * chunkSize
		log.Debug("Storing data to file store")
		fileHash, wait, err := roundRobinFileStore.Store(ctx, io.LimitReader(crand.Reader, int64(size)), int64(size), false)
		// wait until all chunks stored
		if err != nil {
			return err
		}
		err = wait(ctx)
		if err != nil {
			return err
		}

		log.Debug("Waiting for kademlia")
		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			return err
		}

		//get the pivot node's filestore
		item, ok := sim.NodeItem(*sim.PivotNodeID(), bucketKeyFileStore)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		pivotFileStore := item.(*storage.FileStore)
		log.Debug("Starting retrieval routine")
		go func() {
			// start the retrieval on the pivot node - this will spawn retrieve requests for missing chunks
			// we must wait for the peer connections to have started before requesting
			n, err := readAll(pivotFileStore, fileHash)
			log.Info(fmt.Sprintf("retrieved %v", fileHash), "read", n, "err", err)
			if err != nil {
				t.Fatalf("requesting chunks action error: %v", err)
			}
		}()

		log.Debug("Watching for disconnections")
		disconnections := sim.PeerEvents(
			context.Background(),
			sim.NodeIDs(),
			simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeDrop),
		)

		go func() {
			for d := range disconnections {
				if d.Error != nil {
					log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
					t.Fatal(d.Error)
				}
			}
		}()

		//finally check that the pivot node gets all chunks via the root hash
		log.Debug("Check retrieval")
		success := true
		var total int64
		total, err = readAll(pivotFileStore, fileHash)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("check if %08x is available locally: number of bytes read %v/%v (error: %v)", fileHash, total, size, err))
		if err != nil || total != int64(size) {
			success = false
		}

		if !success {
			return fmt.Errorf("Test failed, chunks not available on all nodes")
		}
		log.Debug("Test terminated successfully")
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

func BenchmarkDeliveryFromNodesWithoutCheck(b *testing.B) {
	for chunks := 32; chunks <= 128; chunks *= 2 {
		for i := 2; i < 32; i *= 2 {
			b.Run(
				fmt.Sprintf("nodes=%v,chunks=%v", i, chunks),
				func(b *testing.B) {
					benchmarkDeliveryFromNodes(b, i, 1, chunks, true)
				},
			)
		}
	}
}

func BenchmarkDeliveryFromNodesWithCheck(b *testing.B) {
	for chunks := 32; chunks <= 128; chunks *= 2 {
		for i := 2; i < 32; i *= 2 {
			b.Run(
				fmt.Sprintf("nodes=%v,chunks=%v", i, chunks),
				func(b *testing.B) {
					benchmarkDeliveryFromNodes(b, i, 1, chunks, false)
				},
			)
		}
	}
}

func benchmarkDeliveryFromNodes(b *testing.B, nodes, conns, chunkCount int, skipCheck bool) {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			node := ctx.Config.Node()
			addr := network.NewAddr(node)
			store, datadir, err := createTestLocalStorageForID(node.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				os.RemoveAll(datadir)
				store.Close()
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				SkipCheck:       skipCheck,
				Syncing:         SyncingDisabled,
				Retrieval:       RetrievalDisabled,
				SyncUpdateDelay: 0,
			})

			fileStore := storage.NewFileStore(netStore, storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	log.Info("Initializing test config")
	_, err := sim.AddNodesAndConnectChain(nodes)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		node := nodeIDs[len(nodeIDs)-1]

		item, ok := sim.NodeItem(node, bucketKeyFileStore)
		if !ok {
			b.Fatal("No filestore")
		}
		remoteFileStore := item.(*storage.FileStore)

		pivotNode := nodeIDs[0]
		item, ok = sim.NodeItem(pivotNode, bucketKeyNetStore)
		if !ok {
			b.Fatal("No filestore")
		}
		netStore := item.(*storage.NetStore)

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
				if d.Error != nil {
					log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
					b.Fatal(d.Error)
				}
			}
		}()
		// benchmark loop
		b.ResetTimer()
		b.StopTimer()
	Loop:
		for i := 0; i < b.N; i++ {
			// uploading chunkCount random chunks to the last node
			hashes := make([]storage.Address, chunkCount)
			for i := 0; i < chunkCount; i++ {
				// create actual size real chunks
				ctx := context.TODO()
				hash, wait, err := remoteFileStore.Store(ctx, io.LimitReader(crand.Reader, int64(chunkSize)), int64(chunkSize), false)
				if err != nil {
					b.Fatalf("expected no error. got %v", err)
				}
				// wait until all chunks stored
				err = wait(ctx)
				if err != nil {
					b.Fatalf("expected no error. got %v", err)
				}
				// collect the hashes
				hashes[i] = hash
			}
			// now benchmark the actual retrieval
			// netstore.Get is called for each hash in a go routine and errors are collected
			b.StartTimer()
			errs := make(chan error)
			for _, hash := range hashes {
				go func(h storage.Address) {
					_, err := netStore.Get(ctx, h)
					log.Warn("test check netstore get", "hash", h, "err", err)
					errs <- err
				}(hash)
			}
			// count and report retrieval errors
			// if there are misses then chunk timeout is too low for the distance and volume (?)
			var total, misses int
			for err := range errs {
				if err != nil {
					log.Warn(err.Error())
					misses++
				}
				total++
				if total == chunkCount {
					break
				}
			}
			b.StopTimer()

			if misses > 0 {
				err = fmt.Errorf("%v chunk not found out of %v", misses, total)
				break Loop
			}
		}
		if err != nil {
			b.Fatal(err)
		}
		return nil
	})
	if result.Error != nil {
		b.Fatal(result.Error)
	}

}
