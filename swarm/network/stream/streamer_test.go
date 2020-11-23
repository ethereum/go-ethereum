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

	"github.com/ethereum/go-ethereum/crypto/sha3"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func TestStreamerSubscribe(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)
	err = streamer.Subscribe(tester.IDs[0], stream, NewRange(0, 0), Top)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

func TestStreamerRequestSubscription(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", false)
	err = streamer.RequestSubscription(tester.IDs[0], stream, &Range{}, Top)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

var (
	hash0     = sha3.Sum256([]byte{0})
	hash1     = sha3.Sum256([]byte{1})
	hash2     = sha3.Sum256([]byte{2})
	hashesTmp = append(hash0[:], hash1[:]...)
	hashes    = append(hashesTmp, hash2[:]...)
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

func (self *testClient) NeedData(ctx context.Context, hash []byte) func() {
	self.receivedHashes[string(hash)] = hash
	if bytes.Equal(hash, hash0[:]) {
		return func() {
			<-self.wait0
		}
	} else if bytes.Equal(hash, hash2[:]) {
		return func() {
			<-self.wait2
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
	t string
}

func newTestServer(t string) *testServer {
	return &testServer{
		t: t,
	}
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
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterClientFunc("foo", func(p *Peer, t string, live bool) (Client, error) {
		return newTestClient(t), nil
	})

	peerID := tester.IDs[0]

	stream := NewStream("foo", "", true)
	err = streamer.Subscribe(peerID, stream, NewRange(5, 8), Top)
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
					Peer: peerID,
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
					Peer: peerID,
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
					Peer: peerID,
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Unsubscribe(peerID, stream)
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
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeUnsubscribeMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", false)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t), nil
	})

	peerID := tester.IDs[0]

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
				Peer: peerID,
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
				Peer: peerID,
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
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeUnsubscribeMsgExchangeLive(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t), nil
	})

	peerID := tester.IDs[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   stream,
					Priority: Top,
				},
				Peer: peerID,
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
					To:     1,
				},
				Peer: peerID,
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
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeErrorMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t), nil
	})

	stream := NewStream("bar", "", true)

	peerID := tester.IDs[0]

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
				Peer: peerID,
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 7,
				Msg: &SubscribeErrorMsg{
					Error: "stream bar not registered",
				},
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeLiveAndHistory(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	stream := NewStream("foo", "", true)

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return &testServer{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

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
				Peer: peerID,
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
				Peer: peerID,
			},
			{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: stream,
					HandoverProof: &HandoverProof{
						Handover: &Handover{},
					},
					From:   1,
					To:     1,
					Hashes: make([]byte, HashSize),
				},
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerDownstreamOfferedHashesMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
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

	peerID := tester.IDs[0]

	err = streamer.Subscribe(peerID, stream, NewRange(5, 8), Top)
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
				Peer: peerID,
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
					Peer: peerID,
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
					Peer: peerID,
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
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterServerFunc("foo", func(p *Peer, t string, live bool) (Server, error) {
		return newTestServer(t), nil
	})

	peerID := tester.IDs[0]

	stream := NewStream("foo", "", true)
	err = streamer.RequestSubscription(peerID, stream, NewRange(5, 8), Top)
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
					Peer: peerID,
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
					Peer: peerID,
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
					Peer: peerID,
				},
				{
					Code: 1,
					Msg: &OfferedHashesMsg{
						Stream: stream,
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						From:   1,
						To:     1,
						Hashes: make([]byte, HashSize),
					},
					Peer: peerID,
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Quit(peerID, stream)
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
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	historyStream := getHistoryStream(stream)

	err = streamer.Quit(peerID, historyStream)
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
				Peer: peerID,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}
