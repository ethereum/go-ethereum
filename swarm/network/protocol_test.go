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
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

const (
	TestProtocolVersion   = 6
	TestProtocolNetworkID = 3
)

var (
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

func init() {
	flag.Parse()
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
}

type testStore struct {
	sync.Mutex

	values map[string][]byte
}

func newTestStore() *testStore {
	return &testStore{values: make(map[string][]byte)}
}

func (t *testStore) Load(key string) ([]byte, error) {
	t.Lock()
	defer t.Unlock()
	v, ok := t.values[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

func (t *testStore) Save(key string, v []byte) error {
	t.Lock()
	defer t.Unlock()
	t.values[key] = v
	return nil
}

func HandshakeMsgExchange(lhs, rhs *HandshakeMsg, id discover.NodeID) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		{
			Expects: []p2ptest.Expect{
				{
					Code: 0,
					Msg:  lhs,
					Peer: id,
				},
			},
		},
		{
			Triggers: []p2ptest.Trigger{
				{
					Code: 0,
					Msg:  rhs,
					Peer: id,
				},
			},
		},
	}
}

func newBzzBaseTester(t *testing.T, n int, addr *BzzAddr, spec *protocols.Spec, run func(*BzzPeer) error) *bzzTester {
	cs := make(map[string]chan bool)

	srv := func(p *BzzPeer) error {
		defer func() {
			if cs[p.ID().String()] != nil {
				close(cs[p.ID().String()])
			}
		}()
		return run(p)
	}

	protocol := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		return srv(&BzzPeer{
			Peer:      protocols.NewPeer(p, rw, spec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		})
	}

	s := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), n, protocol)

	for _, id := range s.IDs {
		cs[id.String()] = make(chan bool)
	}

	return &bzzTester{
		addr:           addr,
		ProtocolTester: s,
		cs:             cs,
	}
}

type bzzTester struct {
	*p2ptest.ProtocolTester
	addr *BzzAddr
	cs   map[string]chan bool
	bzz  *Bzz
}

func newBzz(addr *BzzAddr, lightNode bool) *Bzz {
	config := &BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   NewHiveParams(),
		NetworkID:    DefaultNetworkID,
		LightNode:    lightNode,
	}
	kad := NewKademlia(addr.OAddr, NewKadParams())
	bzz := NewBzz(config, kad, nil, nil, nil)
	return bzz
}

func newBzzHandshakeTester(t *testing.T, n int, addr *BzzAddr, lightNode bool) *bzzTester {
	bzz := newBzz(addr, lightNode)
	pt := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), n, bzz.runBzz)

	return &bzzTester{
		addr:           addr,
		ProtocolTester: pt,
		bzz:            bzz,
	}
}

// should test handshakes in one exchange? parallelisation
func (s *bzzTester) testHandshake(lhs, rhs *HandshakeMsg, disconnects ...*p2ptest.Disconnect) error {
	var peers []discover.NodeID
	id := NewNodeIDFromAddr(rhs.Addr)
	if len(disconnects) > 0 {
		for _, d := range disconnects {
			peers = append(peers, d.Peer)
		}
	} else {
		peers = []discover.NodeID{id}
	}

	if err := s.TestExchanges(HandshakeMsgExchange(lhs, rhs, id)...); err != nil {
		return err
	}

	if len(disconnects) > 0 {
		return s.TestDisconnected(disconnects...)
	}

	// If we don't expect disconnect, ensure peers remain connected
	err := s.TestDisconnected(&p2ptest.Disconnect{
		Peer:  s.IDs[0],
		Error: nil,
	})

	if err == nil {
		return fmt.Errorf("Unexpected peer disconnect")
	}

	if err.Error() != "timed out waiting for peers to disconnect" {
		return err
	}

	return nil
}

func correctBzzHandshake(addr *BzzAddr, lightNode bool) *HandshakeMsg {
	return &HandshakeMsg{
		Version:   TestProtocolVersion,
		NetworkID: TestProtocolNetworkID,
		Addr:      addr,
		LightNode: lightNode,
	}
}

func TestBzzHandshakeNetworkIDMismatch(t *testing.T) {
	lightNode := false
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr, lightNode)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr, lightNode),
		&HandshakeMsg{Version: TestProtocolVersion, NetworkID: 321, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("Handshake error: Message handler error: (msg code 0): network id mismatch 321 (!= 3)")},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestBzzHandshakeVersionMismatch(t *testing.T) {
	lightNode := false
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr, lightNode)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr, lightNode),
		&HandshakeMsg{Version: 0, NetworkID: TestProtocolNetworkID, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("Handshake error: Message handler error: (msg code 0): version mismatch 0 (!= %d)", TestProtocolVersion)},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestBzzHandshakeSuccess(t *testing.T) {
	lightNode := false
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr, lightNode)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr, lightNode),
		&HandshakeMsg{Version: TestProtocolVersion, NetworkID: TestProtocolNetworkID, Addr: NewAddrFromNodeID(id)},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestBzzHandshakeLightNode(t *testing.T) {
	var lightNodeTests = []struct {
		name      string
		lightNode bool
	}{
		{"on", true},
		{"off", false},
	}

	for _, test := range lightNodeTests {
		t.Run(test.name, func(t *testing.T) {
			randomAddr := RandomAddr()
			pt := newBzzHandshakeTester(t, 1, randomAddr, false)
			id := pt.IDs[0]
			addr := NewAddrFromNodeID(id)

			err := pt.testHandshake(
				correctBzzHandshake(randomAddr, false),
				&HandshakeMsg{Version: TestProtocolVersion, NetworkID: TestProtocolNetworkID, Addr: addr, LightNode: test.lightNode},
			)

			if err != nil {
				t.Fatal(err)
			}

			if pt.bzz.handshakes[id].LightNode != test.lightNode {
				t.Fatalf("peer LightNode flag is %v, should be %v", pt.bzz.handshakes[id].LightNode, test.lightNode)
			}
		})
	}
}
