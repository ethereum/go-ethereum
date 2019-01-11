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
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	"golang.org/x/crypto/sha3"
)

func TestStreamerSubscribe(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)
	err = streamer.Subscribe(tester.Nodes[0].ID(), stream, NewRange(0, 0), Top)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

func TestStreamerRequestSubscription(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", false)
	err = streamer.RequestSubscription(tester.Nodes[0].ID(), stream, &Range{}, Top)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

var (
	hash0         = sha3.Sum256([]byte{0})
	hash1         = sha3.Sum256([]byte{1})
	hash2         = sha3.Sum256([]byte{2})
	hashesTmp     = append(hash0[:], hash1[:]...)
	hashes        = append(hashesTmp, hash2[:]...)
	corruptHashes = append(hashes[:40])
)

type testClient struct {
	t              string
	wait0          chan bool
	wait2          chan bool
	batchDone      chan bool
	receivedHashes map[string][]byte
}

func newTestClient(t string) *testClient {
	return &testClient{
		t:              t,
		wait0:          make(chan bool),
		wait2:          make(chan bool),
		batchDone:      make(chan bool),
		receivedHashes: make(map[string][]byte),
	}
}

func (self *testClient) NeedData(ctx context.Context, hash []byte) func(context.Context) error {
	self.receivedHashes[string(hash)] = hash
	if bytes.Equal(hash, hash0[:]) {
		return func(context.Context) error {
			<-self.wait0
			return nil
		}
	} else if bytes.Equal(hash, hash2[:]) {
		return func(context.Context) error {
			<-self.wait2
			return nil
		}
	}
	return nil
}

func (self *testClient) BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	close(self.batchDone)
	return nil
}

func (self *testClient) Close() {}

type testServer struct {
	t            string
	sessionIndex uint64
}

func newTestServer(t string, sessionIndex uint64) *testServer {
	return &testServer{
		t:            t,
		sessionIndex: sessionIndex,
	}
}

func (s *testServer) SessionIndex() (uint64, error) {
	return s.sessionIndex, nil
}

func (self *testServer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	return make([]byte, HashSize), from + 1, to + 1, nil, nil
}

func (self *testServer) GetData(context.Context, []byte) ([]byte, error) {
	return nil, nil
}

func (self *testServer) Close() {
}

func TestStreamerDownstreamSubscribeUnsubscribeMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterClientFunc("foo", func(p *Peer, t string, live bool) (Client, error) {
		return newTestClient(t), nil
	})

	node := tester.Nodes[0]

	stream := NewStream("foo", "", true)
	err = streamer.Subscribe(node.ID(), stream, NewRange(5, 8), Top)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(
		p2ptest.Exchange{
			Label: "Subscribe message",
			Expects: []p2ptest.Expect{
				{
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
		// trigger OfferedHashesMsg to actually create the client
		p2ptest.Exchange{
			Label: "OfferedHashes message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: hashes,
						From:   5,
						To:     8,
						Stream: stream,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 2,
					Msg: &WantedHashesMsg{
						Stream: stream,
						Want:   []byte{5},
						From:   9,
						To:     0,
					},
					Peer: node.ID(),
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Unsubscribe(node.ID(), stream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Unsubscribe message",
		Expects: []p2ptest.Expect{
			{
				Code: 0,
				Msg: &UnsubscribeMsg{
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeUnsubscribeMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", false)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 10), nil
	})

	node := tester.Nodes[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  NewRange(5, 8),
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: stream,
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					Hashes: make([]byte, HashSize),
					From:   6,
					To:     9,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "unsubscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 0,
				Msg: &UnsubscribeMsg{
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeUnsubscribeMsgExchangeLive(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 0), nil
	})

	node := tester.Nodes[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: stream,
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					Hashes: make([]byte, HashSize),
					From:   1,
					To:     0,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "unsubscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 0,
				Msg: &UnsubscribeMsg{
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeErrorMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 0), nil
	})

	stream := NewStream("bar", "", true)

	node := tester.Nodes[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  NewRange(5, 8),
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 7,
				Msg: &SubscribeErrorMsg{
					Error: "stream bar not registered",
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeLiveAndHistory(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 10), nil
	})

	node := tester.Nodes[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					History:  NewRange(5, 8),
					Priority: Top,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: NewStream("foo", "", false),
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					Hashes: make([]byte, HashSize),
					From:   6,
					To:     9,
				},
				Peer: node.ID(),
			},
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: stream,
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					From:   11,
					To:     0,
					Hashes: make([]byte, HashSize),
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerDownstreamCorruptHashesMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	var tc *testClient

	streamer.RegisterClientFunc("foo", func(p *Peer, t string, live bool) (Client, error) {
		tc = newTestClient(t)
		return tc, nil
	})

	node := tester.Nodes[0]

	err = streamer.Subscribe(node.ID(), stream, NewRange(5, 8), Top)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Expects: []p2ptest.Expect{
			{
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
			Label: "Corrupt offered hash message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: corruptHashes,
						From:   5,
						To:     8,
						Stream: stream,
					},
					Peer: node.ID(),
				},
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	expectedError := errors.New("Message handler error: (msg code 1): error invalid hashes length (len: 40)")
	if err := tester.TestDisconnected(&p2ptest.Disconnect{Peer: node.ID(), Error: expectedError}); err != nil {
		t.Fatal(err)
	}
}

func TestStreamerDownstreamOfferedHashesMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	var tc *testClient

	streamer.RegisterClientFunc("foo", func(p *Peer, t string, live bool) (Client, error) {
		tc = newTestClient(t)
		return tc, nil
	})

	node := tester.Nodes[0]

	err = streamer.Subscribe(node.ID(), stream, NewRange(5, 8), Top)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Expects: []p2ptest.Expect{
			{
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
			Label: "WantedHashes message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: hashes,
						From:   5,
						To:     8,
						Stream: stream,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 2,
					Msg: &WantedHashesMsg{
						Stream: stream,
						Want:   []byte{5},
						From:   9,
						To:     0,
					},
					Peer: node.ID(),
				},
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	if len(tc.receivedHashes) != 3 {
		t.Fatalf("Expected number of received hashes %v, got %v", 3, len(tc.receivedHashes))
	}

	close(tc.wait0)

	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()

	select {
	case <-tc.batchDone:
		t.Fatal("batch done early")
	case <-timeout.C:
	}

	close(tc.wait2)

	timeout2 := time.NewTimer(10000 * time.Millisecond)
	defer timeout2.Stop()

	select {
	case <-tc.batchDone:
	case <-timeout2.C:
		t.Fatal("timeout waiting batchdone call")
	}

}

func TestStreamerRequestSubscriptionQuitMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t, nil)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 10), nil
	})

	node := tester.Nodes[0]

	stream := NewStream("foo", "", true)
	err = streamer.RequestSubscription(node.ID(), stream, NewRange(5, 8), Top)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(
		p2ptest.Exchange{
			Label: "RequestSubscription message",
			Expects: []p2ptest.Expect{
				{
					Code: 8,
					Msg: &RequestSubscriptionMsg{
						Stream:   stream,
						History:  NewRange(5, 8),
						Priority: Top,
					},
					Peer: node.ID(),
				},
			},
		},
		p2ptest.Exchange{
			Label: "Subscribe message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 4,
					Msg: &SubscribeMsg{
						Stream:   stream,
						History:  NewRange(5, 8),
						Priority: Top,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						Stream: NewStream("foo", "", false),
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: make([]byte, HashSize),
						From:   6,
						To:     9,
					},
					Peer: node.ID(),
				},
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						Stream: stream,
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						From:   11,
						To:     0,
						Hashes: make([]byte, HashSize),
					},
					Peer: node.ID(),
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Quit(node.ID(), stream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Quit message",
		Expects: []p2ptest.Expect{
			{
				Code: 9,
				Msg: &QuitMsg{
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	historyStream := getHistoryStream(stream)

	err = streamer.Quit(node.ID(), historyStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Quit message",
		Expects: []p2ptest.Expect{
			{
				Code: 9,
				Msg: &QuitMsg{
					Stream: historyStream,
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

// TestMaxPeerServersWithUnsubscribe creates a registry with a limited
// number of stream servers, and performs a test with subscriptions and
// unsubscriptions, checking if unsubscriptions will remove streams,
// leaving place for new streams.
func TestMaxPeerServersWithUnsubscribe(t *testing.T) {
	var maxPeerServers = 6
	tester, streamer, _, teardown, err := newStreamerTester(t, &RegistryOptions{
		Retrieval:      RetrievalDisabled,
		Syncing:        SyncingDisabled,
		MaxPeerServers: maxPeerServers,
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 0), nil
	})

	node := tester.Nodes[0]

	for i := 0; i < maxPeerServers+10; i++ {
		stream := NewStream("foo", strconv.Itoa(i), true)

		err = tester.TestExchanges(p2ptest.Exchange{
			Label: "Subscribe message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 4,
					Msg: &SubscribeMsg{
						Stream:   stream,
						Priority: Top,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						Stream: stream,
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: make([]byte, HashSize),
						From:   1,
						To:     0,
					},
					Peer: node.ID(),
				},
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		err = tester.TestExchanges(p2ptest.Exchange{
			Label: "unsubscribe message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 0,
					Msg: &UnsubscribeMsg{
						Stream: stream,
					},
					Peer: node.ID(),
				},
			},
		})

		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestMaxPeerServersWithoutUnsubscribe creates a registry with a limited
// number of stream servers, and performs subscriptions to detect subscriptions
// error message exchange.
func TestMaxPeerServersWithoutUnsubscribe(t *testing.T) {
	var maxPeerServers = 6
	tester, streamer, _, teardown, err := newStreamerTester(t, &RegistryOptions{
		MaxPeerServers: maxPeerServers,
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t, 0), nil
	})

	node := tester.Nodes[0]

	for i := 0; i < maxPeerServers+10; i++ {
		stream := NewStream("foo", strconv.Itoa(i), true)

		if i >= maxPeerServers {
			err = tester.TestExchanges(p2ptest.Exchange{
				Label: "Subscribe message",
				Triggers: []p2ptest.Trigger{
					{
						Code: 4,
						Msg: &SubscribeMsg{
							Stream:   stream,
							Priority: Top,
						},
						Peer: node.ID(),
					},
				},
				Expects: []p2ptest.Expect{
					{
						Code: 7,
						Msg: &SubscribeErrorMsg{
							Error: ErrMaxPeerServers.Error(),
						},
						Peer: node.ID(),
					},
				},
			})

			if err != nil {
				t.Fatal(err)
			}
			continue
		}

		err = tester.TestExchanges(p2ptest.Exchange{
			Label: "Subscribe message",
			Triggers: []p2ptest.Trigger{
				{
					Code: 4,
					Msg: &SubscribeMsg{
						Stream:   stream,
						Priority: Top,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						Stream: stream,
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: make([]byte, HashSize),
						From:   1,
						To:     0,
					},
					Peer: node.ID(),
				},
			},
		})

		if err != nil {
			t.Fatal(err)
		}
	}
}

//TestHasPriceImplementation is to check that the Registry has a
//`Price` interface implementation
func TestHasPriceImplementation(t *testing.T) {
	_, r, _, teardown, err := newStreamerTester(t, &RegistryOptions{
		Retrieval: RetrievalDisabled,
		Syncing:   SyncingDisabled,
	})
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	if r.prices == nil {
		t.Fatal("No prices implementation available for the stream protocol")
	}

	pricesInstance, ok := r.prices.(*StreamerPrices)
	if !ok {
		t.Fatal("`Registry` does not have the expected Prices instance")
	}
	price := pricesInstance.Price(&ChunkDeliveryMsgRetrieval{})
	if price == nil || price.Value == 0 || price.Value != pricesInstance.getChunkDeliveryMsgRetrievalPrice() {
		t.Fatal("No prices set for chunk delivery msg")
	}

	price = pricesInstance.Price(&RetrieveRequestMsg{})
	if price == nil || price.Value == 0 || price.Value != pricesInstance.getRetrieveRequestMsgPrice() {
		t.Fatal("No prices set for chunk delivery msg")
	}
}

/*
TestRequestPeerSubscriptions is a unit test for stream's pull sync subscriptions.

The test does:
	* assign each connected peer to a bin map
  * build up a known kademlia in advance
	* run the EachConn function, which returns supposed subscription bins
	* store all supposed bins per peer in a map
	* check that all peers have the expected subscriptions

This kad table and its peers are copied from network.TestKademliaCase1,
it represents an edge case but for the purpose of testing the
syncing subscriptions it is just fine.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

The resulting kademlia looks like this:
=========================================================================
Fri Dec 21 20:02:39 UTC 2018 KΛÐΞMLIΛ hive: queen's address: 7efef1
population: 12 (12), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 8196 835f                    |  2 8196 (0) 835f (0)
001  2 2690 28f0                    |  2 2690 (0) 28f0 (0)
002  2 4d72 4a45                    |  2 4d72 (0) 4a45 (0)
003  1 646e                         |  1 646e (0)
004  3 769c 76d1 7656               |  3 769c (0) 76d1 (0) 7656 (0)
============ DEPTH: 5 ==========================================
005  1 7a48                         |  1 7a48 (0)
006  1 7cbd                         |  1 7cbd (0)
007  0                              |  0
008  0                              |  0
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestRequestPeerSubscriptions(t *testing.T) {
	// the pivot address; this is the actual kademlia node
	pivotAddr := "7efef1c41d77f843ad167be95f6660567eb8a4a59f39240000cce2e0d65baf8e"

	// a map of bin number to addresses from the given kademlia
	binMap := make(map[int][]string)
	binMap[0] = []string{
		"835fbbf1d16ba7347b6e2fc552d6e982148d29c624ea20383850df3c810fa8fc",
		"81968a2d8fb39114342ee1da85254ec51e0608d7f0f6997c2a8354c260a71009",
	}
	binMap[1] = []string{
		"28f0bc1b44658548d6e05dd16d4c2fe77f1da5d48b6774bc4263b045725d0c19",
		"2690a910c33ee37b91eb6c4e0731d1d345e2dc3b46d308503a6e85bbc242c69e",
	}
	binMap[2] = []string{
		"4a45f1fc63e1a9cb9dfa44c98da2f3d20c2923e5d75ff60b2db9d1bdb0c54d51",
		"4d72a04ddeb851a68cd197ef9a92a3e2ff01fbbff638e64929dd1a9c2e150112",
	}
	binMap[3] = []string{
		"646e9540c84f6a2f9cf6585d45a4c219573b4fd1b64a3c9a1386fc5cf98c0d4d",
	}
	binMap[4] = []string{
		"7656caccdc79cd8d7ce66d415cc96a718e8271c62fb35746bfc2b49faf3eebf3",
		"76d1e83c71ca246d042e37ff1db181f2776265fbcfdc890ce230bfa617c9c2f0",
		"769ce86aa90b518b7ed382f9fdacfbed93574e18dc98fe6c342e4f9f409c2d5a",
	}
	binMap[5] = []string{
		"7a48f75f8ca60487ae42d6f92b785581b40b91f2da551ae73d5eae46640e02e8",
	}
	binMap[6] = []string{
		"7cbd42350bde8e18ae5b955b5450f8e2cef3419f92fbf5598160c60fd78619f0",
	}

	// create the pivot's kademlia
	addr := common.FromHex(pivotAddr)
	k := network.NewKademlia(addr, network.NewKadParams())

	// construct the peers and the kademlia
	for _, binaddrs := range binMap {
		for _, a := range binaddrs {
			addr := common.FromHex(a)
			k.On(network.NewPeer(&network.BzzPeer{BzzAddr: &network.BzzAddr{OAddr: addr}}, k))
		}
	}

	// TODO: check kad table is same
	// currently k.String() prints date so it will never be the same :)
	// --> implement JSON representation of kad table
	log.Debug(k.String())

	// simulate that we would do subscriptions: just store the bin numbers
	fakeSubscriptions := make(map[string][]int)
	//after the test, we need to reset the subscriptionFunc to the default
	defer func() { subscriptionFunc = doRequestSubscription }()
	// define the function which should run for each connection
	// instead of doing real subscriptions, we just store the bin numbers
	subscriptionFunc = func(r *Registry, p *network.Peer, bin uint8, subs map[enode.ID]map[Stream]struct{}) bool {
		// get the peer ID
		peerstr := fmt.Sprintf("%x", p.Over())
		// create the array of bins per peer
		if _, ok := fakeSubscriptions[peerstr]; !ok {
			fakeSubscriptions[peerstr] = make([]int, 0)
		}
		// store the (fake) bin subscription
		log.Debug(fmt.Sprintf("Adding fake subscription for peer %s with bin %d", peerstr, bin))
		fakeSubscriptions[peerstr] = append(fakeSubscriptions[peerstr], int(bin))
		return true
	}
	// create just a simple Registry object in order to be able to call...
	r := &Registry{}
	r.requestPeerSubscriptions(k, nil)
	// calculate the kademlia depth
	kdepth := k.NeighbourhoodDepth()

	// now, check that all peers have the expected (fake) subscriptions
	// iterate the bin map
	for bin, peers := range binMap {
		// for every peer...
		for _, peer := range peers {
			// ...get its (fake) subscriptions
			fakeSubsForPeer := fakeSubscriptions[peer]
			// if the peer's bin is shallower than the kademlia depth...
			if bin < kdepth {
				// (iterate all (fake) subscriptions)
				for _, subbin := range fakeSubsForPeer {
					// ...only the peer's bin should be "subscribed"
					// (and thus have only one subscription)
					if subbin != bin || len(fakeSubsForPeer) != 1 {
						t.Fatalf("Did not get expected subscription for bin < depth; bin of peer %s: %d, subscription: %d", peer, bin, subbin)
					}
				}
			} else { //if the peer's bin is equal or higher than the kademlia depth...
				// (iterate all (fake) subscriptions)
				for i, subbin := range fakeSubsForPeer {
					// ...each bin from the peer's bin number up to k.MaxProxDisplay should be "subscribed"
					// as we start from depth we can use the iteration index to check
					if subbin != i+kdepth {
						t.Fatalf("Did not get expected subscription for bin > depth; bin of peer %s: %d, subscription: %d", peer, bin, subbin)
					}
					// the last "subscription" should be k.MaxProxDisplay
					if i == len(fakeSubsForPeer)-1 && subbin != k.MaxProxDisplay {
						t.Fatalf("Expected last subscription to be: %d, but is: %d", k.MaxProxDisplay, subbin)
					}
				}
			}
		}
	}

	// print some output
	for p, subs := range fakeSubscriptions {
		log.Debug(fmt.Sprintf("Peer %s has the following fake subscriptions: ", p))
		for _, bin := range subs {
			log.Debug(fmt.Sprintf("%d,", bin))
		}
	}
}
