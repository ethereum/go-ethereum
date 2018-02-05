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

	err = streamer.Subscribe(tester.IDs[0], "foo", nil, 0, 0, Top, true)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

var (
	hash0                            = sha3.Sum256([]byte{0})
	hash1                            = sha3.Sum256([]byte{1})
	hash2                            = sha3.Sum256([]byte{2})
	hashesTmp                        = append(hash0[:], hash1[:]...)
	hashes                           = append(hashesTmp, hash2[:]...)
	receivedHashes map[string][]byte = make(map[string][]byte)
	wait0                            = make(chan bool)
	wait2                            = make(chan bool)
	batchDone                        = make(chan bool)
)

type testClient struct {
	t []byte
}

type testServer struct {
	t []byte
}

func (self *testClient) NeedData(hash []byte) func() {
	receivedHashes[string(hash)] = hash
	if bytes.Equal(hash, hash0[:]) {
		return func() {
			<-wait0
		}
	} else if bytes.Equal(hash, hash2[:]) {
		return func() {
			<-wait2
		}
	}
	return nil
}

func (self *testClient) BatchDone(string, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	close(batchDone)
	return nil
}

func (self *testClient) Close() {}

func (self *testServer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	return make([]byte, HashSize), from + 1, to + 1, nil, nil
}

func (self *testServer) GetData([]byte) ([]byte, error) {
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

	streamer.RegisterClientFunc("foo", func(p *Peer, t []byte) (Client, error) {
		return &testClient{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

	err = streamer.Subscribe(peerID, "foo", []byte{}, 5, 8, Top, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
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
	})

	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Unsubscribe(peerID, "foo", []byte{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Unsubscribe message",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 0,
				Msg: &UnsubscribeMsg{
					Stream: "foo",
					Key:    []byte{},
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

	streamer.RegisterServerFunc("foo", func(p *Peer, t []byte) (Server, error) {
		return &testServer{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
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
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg: &OfferedHashesMsg{
					Stream: "foo",
					Key:    []byte{},
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
			p2ptest.Trigger{
				Code: 0,
				Msg: &UnsubscribeMsg{
					Stream: "foo",
					Key:    []byte{},
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

	streamer.RegisterServerFunc("foo", func(p *Peer, t []byte) (Server, error) {
		return &testServer{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "Subscribe message",
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   "bar",
					Key:      []byte{},
					From:     5,
					To:       8,
					Priority: Top,
				},
				Peer: peerID,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
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

func TestStreamerDownstreamOfferedHashesMsgExchange(t *testing.T) {
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterClientFunc("foo", func(p *Peer, t []byte) (Client, error) {
		return &testClient{
			t: t,
		}, nil
	})

	peerID := tester.IDs[0]

	err = streamer.Subscribe(peerID, "foo", []byte{}, 5, 8, Top, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
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
			Label: "WantedHashes message",
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 1,
					Msg: &OfferedHashesMsg{
						HandoverProof: &HandoverProof{
							Handover: &Handover{},
						},
						Hashes: hashes,
						From:   5,
						To:     8,
						Stream: "foo",
					},
					Peer: peerID,
				},
			},
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 2,
					Msg: &WantedHashesMsg{
						Stream: "foo",
						Want:   []byte{5},
						From:   8,
						To:     0,
					},
					Peer: peerID,
				},
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	if len(receivedHashes) != 3 {
		t.Fatalf("Expected number of received hashes %v, got %v", 3, len(receivedHashes))
	}

	close(wait0)

	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()

	select {
	case <-batchDone:
		t.Fatal("batch done early")
	case <-timeout.C:
	}

	close(wait2)

	timeout2 := time.NewTimer(10000 * time.Millisecond)
	defer timeout2.Stop()

	select {
	case <-batchDone:
	case <-timeout2.C:
		t.Fatal("timeout waiting batchdone call")
	}

}
