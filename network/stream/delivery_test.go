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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/network"
	pq "github.com/ethersphere/swarm/network/priorityqueue"
	"github.com/ethersphere/swarm/p2p/protocols"
	"github.com/ethersphere/swarm/storage"
)

//Test requesting a chunk from a peer then issuing a "empty" OfferedHashesMsg (no hashes available yet)
//Should time out as the peer does not have the chunk (no syncing happened previously)
func TestStreamerUpstreamRetrieveRequestMsgExchangeWithoutStore(t *testing.T) {
	tester, _, _, teardown, err := newStreamerTester(&RegistryOptions{
		Syncing: SyncingDisabled, //do no syncing
	})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	chunk := storage.NewChunk(storage.Address(hash0[:]), nil)

	//test the exchange
	err = tester.TestExchanges(p2ptest.Exchange{
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
					Hashes: nil,
					From:   0,
					To:     0,
				},
				Peer: node.ID(),
			},
		},
	})

	//should fail with a timeout as the peer we are requesting
	//the chunk from does not have the chunk
	expectedError := `exchange #0 "RetrieveRequestMsg": timed out`
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, err)
	}
}

// upstream request server receives a retrieve Request and responds with
// offered hashes or delivery if skipHash is set to true
func TestStreamerUpstreamRetrieveRequestMsgExchange(t *testing.T) {
	tester, _, localStore, teardown, err := newStreamerTester(&RegistryOptions{
		Syncing: SyncingDisabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	hash := storage.Address(hash1[:])
	ch := storage.NewChunk(hash, hash1[:])
	_, err = localStore.Put(context.TODO(), chunk.ModePutUpload, ch)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
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
				Code: 6,
				Msg: &ChunkDeliveryMsg{
					Addr:  ch.Address(),
					SData: ch.Data(),
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
	r := NewRegistry(addr.ID(), delivery, nil, nil, nil, nil)

	// an empty priorityQueue has to be created to prevent a goroutine being called after the test has finished
	sp := &Peer{
		BzzPeer:  &network.BzzPeer{Peer: protocolsPeer, BzzAddr: addr},
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: r,
	}
	r.setPeer(sp)
	req := storage.NewRequest(storage.Address(hash0[:]))
	id, err := delivery.FindPeer(context.TODO(), req)
	if err != nil {
		t.Fatal(err)
	}
	if id.ID() != dummyPeerID {
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
	r := NewRegistry(addr.ID(), delivery, nil, nil, nil, nil)
	// an empty priorityQueue has to be created to prevent a goroutine being called after the test has finished
	sp := &Peer{
		BzzPeer:  &network.BzzPeer{Peer: protocolsPeer, BzzAddr: addr},
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: r,
	}
	r.setPeer(sp)

	req := storage.NewRequest(storage.Address(hash0[:]))

	// making a request which should return with "no peer found"
	_, err := delivery.FindPeer(context.TODO(), req)

	expectedError := "no peer found"
	if err.Error() != expectedError {
		t.Fatalf("expected '%v', got %v", expectedError, err)
	}
}

func TestStreamerDownstreamChunkDeliveryMsgExchange(t *testing.T) {
	tester, streamer, localStore, teardown, err := newStreamerTester(&RegistryOptions{
		Syncing: SyncingDisabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

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
	storedChunk, err := localStore.Get(ctx, chunk.ModeGetRequest, chunkKey)
	for err != nil {
		select {
		case <-ctx.Done():
			t.Fatalf("Chunk is not in localstore after timeout, err: %v", err)
		default:
		}
		storedChunk, err = localStore.Get(ctx, chunk.ModeGetRequest, chunkKey)
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !bytes.Equal(storedChunk.Data(), chunkData) {
		t.Fatal("Retrieved chunk has different data than original")
	}

}
