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

var (
	adapter  = flag.String("adapter", "sim", "type of simulation: sim|socket|exec|docker")
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

	protocall := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		return srv(&BzzPeer{
			Peer:      protocols.NewPeer(p, rw, spec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		})
	}

	s := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), n, protocall)

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
}

func newBzzTester(t *testing.T, n int, addr *BzzAddr, pp *p2ptest.TestPeerPool, spec *protocols.Spec, services func(Peer) error) *bzzTester {

	extraservices := func(p *BzzPeer) error {
		pp.Add(p)
		defer pp.Remove(p)
		if services == nil {
			return nil
		}
		return services(p)
	}
	return newBzzBaseTester(t, n, addr, spec, extraservices)
}

// should test handshakes in one exchange? parallelisation
func (s *bzzTester) testHandshake(lhs, rhs *HandshakeMsg, disconnects ...*p2ptest.Disconnect) {
	var peers []discover.NodeID
	id := NewNodeIDFromAddr(rhs.Addr)
	if len(disconnects) > 0 {
		for _, d := range disconnects {
			peers = append(peers, d.Peer)
		}
	} else {
		peers = []discover.NodeID{id}
	}

	s.TestExchanges(HandshakeMsgExchange(lhs, rhs, id)...)
	s.TestDisconnected(disconnects...)
}

func (s *bzzTester) runHandshakes(ids ...discover.NodeID) {
	if len(ids) == 0 {
		ids = s.IDs
	}
	for _, id := range ids {
		s.testHandshake(correctBzzHandshake(s.addr), correctBzzHandshake(NewAddrFromNodeID(id)))
		<-s.cs[id.String()]
	}

}

func correctBzzHandshake(addr *BzzAddr) *HandshakeMsg {
	return &HandshakeMsg{
		Version:   0,
		NetworkID: 322,
		Addr:      addr,
	}
}

func TestBzzHandshakeNetworkIDMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.IDs[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 0, NetworkID: 321, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("network id mismatch 321 (!= 322)")},
	)
}

func TestBzzHandshakeVersionMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.IDs[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 1, NetworkID: 322, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("version mismatch 1 (!= 0)")},
	)
}

func TestBzzHandshakeSuccess(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.IDs[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 0, NetworkID: 322, Addr: NewAddrFromNodeID(id)},
	)
}
