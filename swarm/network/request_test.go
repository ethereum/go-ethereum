// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestStreamerRetrieveRequest(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	peerID := tester.IDs[0]

	streamer.delivery.RequestFromPeers(hash0[:], true)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key:       hash0[:],
					SkipCheck: true,
				},
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestStreamerUpstreamRetrieveRequestMsgExchangeWithoutStore(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	peerID := tester.IDs[0]

	chunk := storage.NewChunk(storage.Key(hash0[:]), nil)

	peer := streamer.getPeer(peerID)

	peer.handleSubscribeMsg(&SubscribeMsg{
		Stream:   retrieveRequestStream,
		Key:      nil,
		From:     0,
		To:       0,
		Priority: Top,
	})

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key: chunk.Key[:],
				},
				Peer: peerID,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: nil,
					Hashes:        nil,
					From:          0,
					To:            0,
				},
				Peer: peerID,
			},
		},
	})

	expectedError := "exchange 0: 'RetrieveRequestMsg' timed out"
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, err)
	}
}

// upstream request server receives a retrieve Request and responds with
// offered hashes or delivery if skipHash is set to true
func TestStreamerUpstreamRetrieveRequestMsgExchange(t *testing.T) {
	tester, streamer, localStore, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	peerID := tester.IDs[0]
	peer := streamer.getPeer(peerID)

	peer.handleSubscribeMsg(&SubscribeMsg{
		Stream:   retrieveRequestStream,
		Key:      nil,
		From:     0,
		To:       0,
		Priority: Top,
	})

	hash := storage.Key(hash0[:])
	chunk := storage.NewChunk(hash, nil)
	chunk.SData = hash
	localStore.Put(chunk)
	chunk.WaitToStore()

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key: hash,
				},
				Peer: peerID,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					Hashes: hash,
					From:   0,
					// TODO: why is this 32???
					To:     32,
					Key:    []byte{},
					Stream: retrieveRequestStream,
				},
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	hash = storage.Key(hash1[:])
	chunk = storage.NewChunk(hash, nil)
	chunk.SData = hash1[:]
	localStore.Put(chunk)
	chunk.WaitToStore()

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key:       hash,
					SkipCheck: true,
				},
				Peer: peerID,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 6,
				Msg: &ChunkDeliveryMsg{
					Key:   hash,
					SData: hash,
				},
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerDownstreamChunkDeliveryMsgExchange(t *testing.T) {
	tester, streamer, localStore, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterIncomingStreamer("foo", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return &testIncomingStreamer{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

	err = streamer.Subscribe(peerID, "foo", []byte{}, 5, 8, Top, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	chunkKey := hash0[:]
	chunkData := hash1[:]
	chunk, created := localStore.GetOrCreateRequest(chunkKey)

	if !created {
		t.Fatal("chunk already exists")
	}
	select {
	case <-chunk.ReqC:
		t.Fatal("chunk is already received")
	default:
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   "foo",
					Key:      []byte{},
					From:     5,
					To:       8,
					Priority: Top,
				},
				Peer: peerID,
			},
		},
	},
		p2ptest.Exchange{
			Label: "ChunkDeliveryRequest message",
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 6,
					Msg: &ChunkDeliveryMsg{
						Key:   chunkKey,
						SData: chunkData,
					},
					Peer: peerID,
				},
			},
		})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	timeout := time.NewTimer(1 * time.Second)

	select {
	case <-timeout.C:
		t.Fatal("timeout receiving chunk")
	case <-chunk.ReqC:
	}

	storedChunk, err := localStore.Get(chunkKey)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !bytes.Equal(storedChunk.SData, chunkData) {
		t.Fatal("Retrieved chunk has different data than original")
	}

}

func TestDeliveryFromNodes(t *testing.T) {
	testSimulation(t, testDeliveryFromNodes(2, 1, 8100, true))
	testSimulation(t, testDeliveryFromNodes(2, 1, 8100, false))
	testSimulation(t, testDeliveryFromNodes(3, 1, 8100, true))
	testSimulation(t, testDeliveryFromNodes(3, 1, 8100, false))
}

func testDeliveryFromNodes(nodes, conns, size int, skipCheck bool) func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
	return func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
		trigger := func(net *simulations.Network) chan discover.NodeID {
			triggerC := make(chan discover.NodeID)
			ticker := time.NewTicker(500 * time.Millisecond)
			go func() {
				defer ticker.Stop()
				// we are only testing the pivot node (net.Nodes[0])
				for range ticker.C {
					triggerC <- net.Nodes[0].ID()
				}
			}()
			return triggerC
		}

		action := func(net *simulations.Network) func(context.Context) error {
			// here we distribute chunks of a random file into localstores of nodes 1 to nodes
			rrdpa := storage.NewDPA(newRoundRobinStore(localStores[1:]...), storage.NewChunkerParams())
			rrdpa.Start()
			// create a retriever dpa for the pivot node
			dpacs := storage.NewNetStore(localStores[0].(*storage.LocalStore), func(chunk *storage.Chunk) error { return delivery.RequestFromPeers(chunk.Key[:], skipCheck) })
			dpa := storage.NewDPA(dpacs, storage.NewChunkerParams())
			dpa.Start()
			return func(context.Context) error {
				defer rrdpa.Stop()
				// upload an actual random file of size size
				hash, wait, err := rrdpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
				if err != nil {
					return err
				}
				// wait until all chunks stored
				// TODO: is wait() necessary?
				wait()
				// assign the fileHash to a global so that it is available for the check function
				fileHash = hash
				go func() {
					defer dpa.Stop()
					log.Debug(fmt.Sprintf("retrieve %v", fileHash))
					// start the retrieval on the pivot node - this will spawn retrieve requests for missing chunks
					// we must wait for the peer connections to have started before requesting
					time.Sleep(2 * time.Second)
					n, err := mustReadAll(dpa, fileHash)
					log.Debug(fmt.Sprintf("retrieved %v", fileHash), "read", n, "err", err)
				}()
				return nil
			}
		}

		check := func(net *simulations.Network, dpa *storage.DPA) func(ctx context.Context, id discover.NodeID) (bool, error) {
			return func(ctx context.Context, id discover.NodeID) (bool, error) {
				if id != net.Nodes[0].ID() {
					return true, nil
				}
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				default:
				}
				// try to locally retrieve the file to check if retrieve requests have been successful
				total, err := mustReadAll(dpa, fileHash)
				log.Debug(fmt.Sprintf("check if %08x is available locally: number of bytes read %v/%v (error: %v)", fileHash, total, size, err))
				if err != nil || total != size {
					return false, nil
				}
				return true, nil
				// node := net.GetNode(id)
				// if node == nil {
				// 	return false, fmt.Errorf("unknown node: %s", id)
				// }
				// client, err := node.Client()
				// if err != nil {
				// 	return false, fmt.Errorf("error getting node client: %s", err)
				// }
				// var response int
				// if err := client.Call(&response, "test_haslocal", hash); err != nil {
				// 	return false, fmt.Errorf("error getting bzz_has response: %s", err)
				// }
				// log.Debug(fmt.Sprintf("node has: %v\n%v", id, response))
				// return response == 0, nil
			}
		}

		result, err := runSimulation(nodes, conns, "delivery", NewAddrFromNodeID, action, trigger, check, adapter)
		if err != nil {
			return nil, fmt.Errorf("Setting up simulation failed: %v", err)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("Simulation failed: %s", result.Error)
		}
		return result, err
	}
}

// newDeliveryService
func newDeliveryService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	addr := NewAddrFromNodeID(id)
	kad := NewKademlia(addr.Over(), NewKadParams())
	localStore := localStores[nodeCount]
	dbAccess := NewDbAccess(localStore.(*storage.LocalStore))
	streamer := NewStreamer(NewDelivery(kad, dbAccess))
	if nodeCount == 0 {
		// the delivery service for the pivot node is assigned globally
		// so that the simulation action call can use it for the
		// swarm enabled dpa
		delivery = streamer.delivery
	}
	self := &testStreamerService{
		addr:     addr,
		streamer: streamer,
	}
	self.run = self.runDelivery
	nodeCount++
	return self, nil
}

func (b *testStreamerService) runDelivery(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	bzzPeer := &bzzPeer{
		Peer:      protocols.NewPeer(p, rw, StreamerSpec),
		localAddr: b.addr,
		BzzAddr:   NewAddrFromNodeID(p.ID()),
	}
	b.streamer.delivery.overlay.On(bzzPeer)
	defer b.streamer.delivery.overlay.Off(bzzPeer)
	go func() {
		// each node Subscribes to each other's retrieveRequestStream
		// need to wait till an aynchronous process registers the peers in streamer.peers
		// that is used by Subscribe
		time.Sleep(1 * time.Second)
		err := b.streamer.Subscribe(p.ID(), retrieveRequestStream, nil, 0, 0, Top, true)
		if err != nil {
			log.Warn("error in subscribe", "err", err)
		}
	}()
	return b.streamer.Run(bzzPeer)
}
