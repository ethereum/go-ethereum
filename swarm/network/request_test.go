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
	"time"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestStreamerUpstreamRetrieveRequestMsgExchangeWithoutStore(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	chunk := storage.NewChunk(storage.Key(hash0[:]), nil)

	peer := streamer.getPeer(peerId)

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
				Peer: peerId,
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
				Peer: peerId,
			},
		},
	})

	expectedError := "exchange 0: 'RetrieveRequestMsg' timed out"
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, err)
	}
}

func TestStreamerUpstreamRetrieveRequestMsgExchange(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, localStore, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	chunk := storage.NewChunk(storage.Key(hash0[:]), nil)

	peer := streamer.getPeer(peerId)

	peer.handleSubscribeMsg(&SubscribeMsg{
		Stream:   retrieveRequestStream,
		Key:      nil,
		From:     0,
		To:       0,
		Priority: Top,
	})

	chunk.SData = hash0[:]
	localStore.Put(chunk)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key: chunk.Key[:],
				},
				Peer: peerId,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: nil,
					Hashes:        chunk.Key[:],
					From:          0,
					// TODO: why is this 32???
					To:     32,
					Key:    []byte{},
					Stream: retrieveRequestStream,
				},
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}
