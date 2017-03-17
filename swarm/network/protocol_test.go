package network

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func bzzHandshakeExchange(lhs, rhs *bzzHandshake, id *adapters.NodeId) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 0,
					Msg:  lhs,
					Peer: id,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 0,
					Msg:  rhs,
					Peer: id,
				},
			},
		},
	}
}

func newBzzBaseTester(t *testing.T, n int, addr *peerAddr, ct *protocols.CodeMap, services func(Peer) error) *bzzTester {
	if ct == nil {
		ct = BzzCodeMap()
	}

	cs := make(map[string]chan bool)

	srv := func(p Peer) error {
		defer close(cs[p.ID().String()])
		return services(p)
	}

	protocall := func(na adapters.NodeAdapter) adapters.ProtoCall {
		protocol := Bzz(addr.OverlayAddr(), na, ct, srv, nil, nil)
		return protocol.Run
	}

	s := p2ptest.NewProtocolTester(t, NodeId(addr), n, protocall)

	for _, id := range s.Ids {
		cs[id.NodeID.String()] = make(chan bool)
	}

	return &bzzTester{
		addr:           addr,
		ProtocolTester: s,
		cs:             cs,
	}
}

type bzzTester struct {
	*p2ptest.ProtocolTester
	addr *peerAddr
	cs   map[string]chan bool
}

func newBzzTester(t *testing.T, n int, addr *peerAddr, pp *p2ptest.TestPeerPool, ct *protocols.CodeMap, services func(Peer) error) *bzzTester {

	extraservices := func(p Peer) error {
		pp.Add(p)
		p.DisconnectHook(func(err error) {
			pp.Remove(p)
		})
		if services != nil {
			err := services(p)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return newBzzBaseTester(t, n, addr, ct, extraservices)
}

// should test handshakes in one exchange? parallelisation
func (s *bzzTester) testHandshake(lhs, rhs *bzzHandshake, disconnects ...*p2ptest.Disconnect) {
	var peers []*adapters.NodeId
	id := NodeId(rhs.Addr)
	if len(disconnects) > 0 {
		for _, d := range disconnects {
			peers = append(peers, d.Peer)
		}
	} else {
		peers = []*adapters.NodeId{id}
	}
	<-s.GetPeer(id).Connc

	s.TestExchanges(bzzHandshakeExchange(lhs, rhs, id)...)
	s.TestDisconnected(disconnects...)
}

func (s *bzzTester) runHandshakes(ids ...*adapters.NodeId) {
	if len(ids) == 0 {
		ids = s.Ids
	}
	for _, id := range ids {
		s.testHandshake(correctBzzHandshake(s.addr), correctBzzHandshake(NewPeerAddrFromNodeId(id)))
		<-s.cs[id.NodeID.String()]
	}

}

func correctBzzHandshake(addr *peerAddr) *bzzHandshake {
	return &bzzHandshake{0, 322, addr}
}

func TestBzzHandshakeNetworkIdMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{0, 321, NewPeerAddrFromNodeId(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("network id mismatch 321 (!= 322)")},
	)
}

func TestBzzHandshakeVersionMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{1, 322, NewPeerAddrFromNodeId(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("version mismatch 1 (!= 0)")},
	)
}

func TestBzzHandshakeSuccess(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{0, 322, NewPeerAddrFromNodeId(id)},
	)
}

func TestBzzPeerPoolAdd(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)

	id := s.Ids[0]
	glog.V(logger.Detail).Infof("handshake with %v", id)
	s.runHandshakes()

	if !pp.Has(id) {
		t.Fatalf("peer '%v' not added: %v", id, pp)
	}
}

func TestBzzPeerPoolRemove(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	s.runHandshakes()

	id := s.Ids[0]
	pp.Get(id).Drop(fmt.Errorf("p2p: read or write on closed message pipe"))
	s.TestDisconnected(&p2ptest.Disconnect{id, fmt.Errorf("p2p: read or write on closed message pipe")})
	if pp.Has(id) {
		t.Fatalf("peer '%v' not removed: %v", id, pp)
	}
}

func TestBzzPeerPoolBothAddRemove(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	s.runHandshakes()

	id := s.Ids[0]
	if !pp.Has(id) {
		t.Fatalf("peer '%v' not added: %v", id, pp)
	}

	pp.Get(id).Drop(fmt.Errorf("p2p: read or write on closed message pipe"))
	s.TestDisconnected(&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("p2p: read or write on closed message pipe")})
	if pp.Has(id) {
		t.Fatalf("peer '%v' not removed: %v", id, pp)
	}
}

func TestBzzPeerPoolNotAdd(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, 1, addr, pp, nil, nil)

	id := s.Ids[0]
	s.testHandshake(correctBzzHandshake(addr), &bzzHandshake{0, 321, NewPeerAddrFromNodeId(id)}, &p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("network id mismatch 321 (!= 322)")})
	if pp.Has(id) {
		t.Fatalf("peer %v incorrectly added: %v", id, pp)
	}
}
