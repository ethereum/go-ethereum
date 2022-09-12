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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
)

type testServerPeerSub struct {
	regCh   chan *peer
	unregCh chan *peer
}

func newTestServerPeerSub() *testServerPeerSub {
	return &testServerPeerSub{
		regCh:   make(chan *peer, 1),
		unregCh: make(chan *peer, 1),
	}
}

func (t *testServerPeerSub) registerPeer(p *peer)   { t.regCh <- p }
func (t *testServerPeerSub) unregisterPeer(p *peer) { t.unregCh <- p }

func TestPeerSubscription(t *testing.T) {
	peers := newPeerSet()
	defer peers.close()

	checkIds := func(expect []string) {
		givenIDs := peers.ids()
		given := make([]string, len(givenIDs))
		for i, id := range givenIDs {
			given[i] = id.String()
		}
		if len(given) == 0 && len(expect) == 0 {
			return
		}
		sort.Strings(given)
		sort.Strings(expect)
		if !reflect.DeepEqual(given, expect) {
			t.Fatalf("all peer ids mismatch, want %v, given %v", expect, given)
		}
	}
	checkPeers := func(peerCh chan *peer) {
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
	peer := newPeer(2, NetworkId, p2p.NewPeer(id, "name", nil), nil)
	peers.register(peer)

	checkIds([]string{peer.id})
	checkPeers(sub.regCh)

	peers.unregister(peer.ID())
	checkIds([]string{})
	checkPeers(sub.unregCh)
}

type fakeChain struct{}

func (f *fakeChain) Config() *params.ChainConfig { return params.MainnetChainConfig }
func (f *fakeChain) Genesis() *types.Block {
	return core.DefaultGenesisBlock().ToBlock()
}
func (f *fakeChain) CurrentHeader() *types.Header { return &types.Header{Number: big.NewInt(10000000)} }

type testHandshakeModule struct {
	genesis    common.Hash
	forkID     forkid.ID
	forkFilter forkid.Filter
}

func (m *testHandshakeModule) sendHandshake(p *peer, send *keyValueList) {
	sendGeneralInfo(p, send, m.genesis, m.forkID)
}

func (m *testHandshakeModule) receiveHandshake(p *peer, recv keyValueMap) error {
	return receiveGeneralInfo(p, recv, m.genesis, m.forkFilter)
}

func TestHandshake(t *testing.T) { //TODO make this test work
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer1 := newPeer(2, NetworkId, p2p.NewPeer(id, "name", nil), net)
	peer2 := newPeer(2, NetworkId, p2p.NewPeer(id, "name", nil), app)

	var (
		errCh1 = make(chan error, 1)
		errCh2 = make(chan error, 1)

		genesis = common.HexToHash("cafebabe")

		chain1, chain2 = &fakeChain{}, &fakeChain{}
	)
	module1 := &testHandshakeModule{
		genesis:    genesis,
		forkID:     forkid.NewID(chain1.Config(), chain1.Genesis().Hash(), chain1.CurrentHeader().Number.Uint64()),
		forkFilter: forkid.NewFilter(chain1),
	}
	module2 := &testHandshakeModule{
		genesis:    genesis,
		forkID:     forkid.NewID(chain2.Config(), chain2.Genesis().Hash(), chain2.CurrentHeader().Number.Uint64()),
		forkFilter: forkid.NewFilter(chain2),
	}

	go func() {
		errCh1 <- peer1.handshake([]handshakeModule{module1})
	}()
	go func() {
		errCh2 <- peer2.handshake([]handshakeModule{module2})
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
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}
