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
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
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

/*
TestGetSubscriptionsRPC sets up a simulation network of 16 nodes,
starts the simulation, waits for SyncUpdateDelay in order to kick off
stream registration, then tests that there are subscriptions.
If provided with the `-printstats = true` option, it will print
the information of who is subscribed to who to STDOUT
*/
func TestGetSubscriptionsRPC(t *testing.T) {
	//arbitrarily set to 16
	nodeCount := 16
	//set the syncUpdateDelay for sync registrations to start
	syncUpdateDelay := 500 * time.Millisecond
	//we will later need the kad table for each node
	bucketKeyKad := simulation.BucketKey("kademlia")
	//create a standard sim
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			n := ctx.Config.Node()
			addr := network.NewAddr(n)
			store, datadir, err := createTestLocalStorageForID(n.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			//store the kad table
			bucket.Store(bucketKeyKad, kad)
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(dummyRequestFromPeers, true).New
			//configure so that sync registrations actually happen
			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval:       RetrievalEnabled,
				Syncing:         SyncingAutoSubscribe, //enable sync registrations
				SyncUpdateDelay: syncUpdateDelay,
			}, nil)

			bucket.Store(bucketKeyRegistry, r)
			cleanup = func() {
				os.RemoveAll(datadir)
				netStore.Close()
				r.Close()
			}

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelSimRun()

	//upload a snapshot
	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		t.Fatal(err)
	}

	//run the simulation
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		log.Info("Simulation running")
		nodes := sim.Net.Nodes
		//setup the filter for SubscribeMsg
		msgs := sim.PeerEvents(
			context.Background(),
			sim.NodeIDs(),
			simulation.NewPeerEventsFilter().ReceivedMessages().Protocol("stream").MsgCode(4), //4 is SubscribeMsg
		)

		//setup the vars we need
		msgCount := 0
		expectedMsgCount := 0
		allSubscriptionsDone := make(chan struct{})

		//in a loop, catch all SubscribeMsg and just add up
		go func() {
			for m := range msgs {
				if m.Error != nil {
					log.Error("stream message", "err", m.Error)
					continue
				}
				log.Trace("stream message", "node", m.NodeID, "peer", m.PeerID)
				//add one
				msgCount += 1
				if msgCount == expectedMsgCount {
					//the expected amount is reached
					allSubscriptionsDone <- struct{}{}
					return
				}
			}
		}()

		//first iterate all nodes to get the expected number of subscriptions from the kad table
		for _, node := range nodes {
			item, ok := sim.NodeItem(node.ID(), bucketKeyKad)
			if !ok {
				return fmt.Errorf("No kademlia")
			}
			kad := item.(*network.Kademlia)
			//define the function which should run for each connection - just count subscriptions
			//this is not actually subscribing but iterating the same way as the subscriptions do,
			//as we need just the number
			eachBinFunc := func(p *network.Peer, bin int) bool {
				expectedMsgCount += 1
				return true
			}
			//call the actual kademlia for the count
			kad.EachBin(kad.BaseAddr(), pot.DefaultPof(kad.MaxProxDisplay), 0, eachBinFunc)
		}
		log.Debug("Expected message count: ", "expectedMsgCount", expectedMsgCount)
		//wait until all subscriptions are done
		<-allSubscriptionsDone
		log.Info("All subscriptions received")
		//now iterate again, this time we call each node via RPC to get its subscriptions
		for _, node := range nodes {
			//create rpc client
			client, err := node.Client()
			if err != nil {
				t.Fatalf("create node 1 rpc client fail: %v", err)
			}

			item, ok := sim.NodeItem(node.ID(), bucketKeyRegistry)
			if !ok {
				return fmt.Errorf("No registry")
			}
			registry := item.(*Registry)
			//ask it for subscriptions
			pstreams := make(map[string][]string)
			err = client.Call(&pstreams, "stream_getPeerSubscriptions")
			if err != nil {
				t.Fatal(err)
			}
			//length of the subscriptions can not be smaller than number of peers
			if len(pstreams) < len(registry.peers) {
				t.Fatal("Subscription count is smaller than expected")
			}
			log.Debug(fmt.Sprintf("node %s subscriptions:", node.String()))
			for p, ps := range pstreams {
				log.Debug(fmt.Sprintf("...with node %s: ", p))
				for _, s := range ps {
					log.Debug(fmt.Sprintf("......%s", s))
				}
			}
		}
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
}
