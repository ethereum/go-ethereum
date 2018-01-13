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
	"testing"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestStreamerRetrieveRequest(t *testing.T) {
	// TODO: we only need streamer
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
	// TODO: we only need streamer
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
	// TODO: we only need streamer
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
					HandoverProof: nil,
					Hashes:        hash,
					From:          0,
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
