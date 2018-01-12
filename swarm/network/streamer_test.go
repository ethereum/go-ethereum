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
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	sha3 "github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func init() {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlWarn, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

// TODO: extract newStreamer
func newStreamerTester(t *testing.T) (*p2ptest.ProtocolTester, *Streamer, *storage.LocalStore, func(), error) {
	// setup
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())

	// temp datadir
	datadir, err := ioutil.TempDir("", "streamer")
	if err != nil {
		return nil, nil, nil, func() {}, err
	}
	teardown := func() {
		os.RemoveAll(datadir)
	}

	localStore, err := storage.NewTestLocalStore(datadir)
	if err != nil {
		return nil, nil, nil, teardown, err
	}

	dbAccess := NewDbAccess(localStore)
	streamer := NewStreamer(to, dbAccess)

	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		bzzPeer := &bzzPeer{
			Peer:      protocols.NewPeer(p, rw, StreamerSpec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		}
		to.On(bzzPeer)
		return streamer.Run(bzzPeer)
	}

	protocolTester := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), 1, run)
	return protocolTester, streamer, localStore, teardown, nil
}

// TODO
// func newStreamer() (*Streamer, error) {
//
// }

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

type testIncomingStreamer struct {
	t []byte
}

type testOutgoingStreamer struct {
	t []byte
}

func (self *testIncomingStreamer) NeedData(hash []byte) func() {
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

func (self *testIncomingStreamer) BatchDone(string, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	close(batchDone)
	return nil
}

func (self *testOutgoingStreamer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	return make([]byte, HashSize), from + 1, to + 1, nil, nil
}

func (self *testOutgoingStreamer) GetData([]byte) []byte {
	return nil
}

func TestStreamerDownstreamSubscribeMsgExchange(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterIncomingStreamer("foo", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return &testIncomingStreamer{
			t: t,
		}, nil
	})

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	err = streamer.Subscribe(peerId, "foo", []byte{}, 5, 8, Top, true)
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
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamerUpstreamSubscribeMsgExchange(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterOutgoingStreamer("foo", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		return &testOutgoingStreamer{
			t: t,
		}, nil
	})

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

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
				Peer: peerId,
			},
		},
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg: &OfferedHashesMsg{
					HandoverProof: nil,
					Hashes:        make([]byte, HashSize),
					From:          6,
					To:            9,
				},
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

}

func TestStreamerDownstreamOfferedHashesMsgExchange(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterIncomingStreamer("foo", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return &testIncomingStreamer{
			t: t,
		}, nil
	})

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	err = streamer.Subscribe(peerId, "foo", []byte{}, 5, 8, Top, true)
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
				Peer: peerId,
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
					Peer: peerId,
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
					Peer: peerId,
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

func TestRetrieveRequest(t *testing.T) {
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

	streamer.Retrieve(chunk)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RetrieveRequestMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 5,
				Msg: &RetrieveRequestMsg{
					Key: chunk.Key[:],
				},
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUpstreamRetrieveRequestMsgExchangeWithoutStore(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, _, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	// streamer.RegisterOutgoingStreamer("foo", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
	// 	return &testOutgoingStreamer{
	// 		t: t,
	// 	}, nil
	// })

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "SubscribeMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   retrieveRequestStream,
					Key:      nil,
					From:     0,
					To:       0,
					Priority: Top,
				},
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

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

func TestUpstreamRetrieveRequestMsgExchange(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, localStore, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	// streamer.RegisterOutgoingStreamer("foo", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
	// 	return &testOutgoingStreamer{
	// 		t: t,
	// 	}, nil
	// })

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	peerId := tester.IDs[0]

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "SubscribeMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream:   retrieveRequestStream,
					Key:      nil,
					From:     0,
					To:       0,
					Priority: Top,
				},
				Peer: peerId,
			},
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

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

func waitForPeers(streamer *Streamer, timeout time.Duration) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	timeoutTimer := time.NewTimer(timeout)
	for {
		select {
		case <-ticker.C:
			if len(streamer.peers) > 0 {
				return nil
			}
		case <-timeoutTimer.C:
			return errors.New("timeout")
		}
	}
}
