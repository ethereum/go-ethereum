// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"crypto/rand"
	"math/big"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type testServerPeerSub struct {
	regCh   chan *serverPeer
	unregCh chan *serverPeer
}

func newTestServerPeerSub() *testServerPeerSub {
	return &testServerPeerSub{
		regCh:   make(chan *serverPeer, 1),
		unregCh: make(chan *serverPeer, 1),
	}
}

func (t *testServerPeerSub) registerPeer(p *serverPeer)   { t.regCh <- p }
func (t *testServerPeerSub) unregisterPeer(p *serverPeer) { t.unregCh <- p }

func TestPeerSubscription(t *testing.T) {
	peers := newServerPeerSet()
	defer peers.close()

	checkIds := func(expect []string) {
		given := peers.ids()
		if len(given) == 0 && len(expect) == 0 {
			return
		}
		sort.Strings(given)
		sort.Strings(expect)
		if !reflect.DeepEqual(given, expect) {
			t.Fatalf("all peer ids mismatch, want %v, given %v", expect, given)
		}
	}
	checkPeers := func(peerCh chan *serverPeer) {
		select {
		case <-peerCh:
		case <-time.NewTimer(100 * time.Millisecond).C:
			t.Fatalf("timeout, no event received")
		}
		select {
		case <-peerCh:
			t.Fatalf("unexpected event received")
		case <-time.NewTimer(10 * time.Millisecond).C:
		}
	}
	checkIds([]string{})

	sub := newTestServerPeerSub()
	peers.subscribe(sub)

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])
	peer := newServerPeer(2, NetworkId, false, p2p.NewPeer(id, "name", nil), nil)
	peers.register(peer)

	checkIds([]string{peer.id})
	checkPeers(sub.regCh)

	peers.unregister(peer.id)
	checkIds([]string{})
	checkPeers(sub.unregCh)
}

func TestHandshake(t *testing.T) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer1 := newClientPeer(2, NetworkId, p2p.NewPeer(id, "name", nil), net)
	peer2 := newServerPeer(2, NetworkId, true, p2p.NewPeer(id, "name", nil), app)

	var (
		errCh1 = make(chan error, 1)
		errCh2 = make(chan error, 1)

		td      = big.NewInt(100)
		head    = common.HexToHash("deadbeef")
		headNum = uint64(10)
		genesis = common.HexToHash("cafebabe")
	)
	go func() {
		errCh1 <- peer1.handshake(td, head, headNum, genesis, func(list *keyValueList) {
			var announceType uint64 = announceTypeSigned
			*list = (*list).add("announceType", announceType)
		}, nil)
	}()
	go func() {
		errCh2 <- peer2.handshake(td, head, headNum, genesis, nil, func(recv keyValueMap) error {
			var reqType uint64
			err := recv.get("announceType", &reqType)
			if err != nil {
				t.Fatal(err)
			}
			if reqType != announceTypeSigned {
				t.Fatal("Expected announceTypeSigned")
			}
			return nil
		})
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh1:
			if err != nil {
				t.Fatalf("handshake failed, %v", err)
			}
		case err := <-errCh2:
			if err != nil {
				t.Fatalf("handshake failed, %v", err)
			}
		case <-time.NewTimer(100 * time.Millisecond).C:
			t.Fatalf("timeout")
		}
	}
}
