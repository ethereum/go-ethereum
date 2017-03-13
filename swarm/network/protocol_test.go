package network

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
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

func newTestBzzProtocol(addr *peerAddr, ct *protocols.CodeMap, services func(Peer) error) func(adapters.NodeAdapter) adapters.ProtoCall {
	if ct == nil {
		ct = BzzCodeMap()
	}
	// ct.Register(p2ptest.FlushMsg)
	return func(na adapters.NodeAdapter) adapters.ProtoCall {
		srv := func(p Peer) error {
			if services != nil {
				err := services(p)
				if err != nil {
					return err
				}
			}
			// id := p.ID()
			// p.Register(p2ptest.FlushMsg, func(interface{}) error {
			// 	flushc := na.(p2ptest.TestNetAdapter).GetPeer(&adapters.NodeId{id}).Flushc
			// 	flushc <- true
			// 	return nil
			// })
			return nil
		}

		protocol := Bzz(addr.OverlayAddr(), pp, na, ct, srv)
		return protocol.Run
	}
}

type bzzTester struct {
	*p2ptest.ExchangeSession
	// flushCode int
	addr *peerAddr
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
	s.TestConnected(peers...)
	s.TestExchanges(bzzHandshakeExchange(lhs, rhs, id)...)
	s.TestDisconnected(disconnects...)
}

// func (s *bzzTester) flush(ids ...*adapters.NodeId) {
// 	s.Flush(s.flushCode, ids...)
// }

func (s *bzzTester) runHandshakes(ids ...*adapters.NodeId) {
	if len(ids) == 0 {
		ids = s.Ids
	}
	for _, id := range ids {
		s.testHandshake(correctBzzHandshake(s.addr), correctBzzHandshake(NewPeerAddrFromNodeId(id)))
	}

}

func correctBzzHandshake(addr *peerAddr) *bzzHandshake {
	return &bzzHandshake{0, 322, addr}
}

func newBzzTester(t *testing.T, addr *peerAddr, pp PeerPool, ct *protocols.CodeMap, services func(Peer) error) *bzzTester {

	extraservices := func(p Peer) error {
		pp.Add(p)
		p.Register(&protocols.Disconnect{}, func(e interface{}) error { pp.Remove(p) })
		return services(p)
	}
	s := p2ptest.NewProtocolTester(t, NodeId(addr), 1, newTestBzzProtocol(addr, pp, ct, extarservices))
	return &bzzTester{
		addr: addr,
		// flushCode:       4,
		ExchangeSession: s,
	}
}

func TestBzzHandshakeNetworkIdMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, addr, pp, nil, nil)
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
	s := newBzzTester(t, addr, pp, nil, nil)
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
	s := newBzzTester(t, addr, pp, nil, nil)
	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{0, 322, NewPeerAddrFromNodeId(id)},
	)
}

func TestBzzPeerPoolAdd(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, addr, pp, nil, nil)

	id := s.Ids[0]
	glog.V(6).Infof("handshake with %v", id)
	s.runHandshakes()
	// s.TestConnected()
	if !pp.Has(id) {
		t.Fatalf("peer '%v' not added: %v", id, pp)
	}
}

func TestBzzPeerPoolRemove(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, addr, pp, nil, nil)
	s.runHandshakes()

	id := s.Ids[0]
	pp.Get(id).Drop()
	s.TestDisconnected(&p2ptest.Disconnect{id, fmt.Errorf("p2p: read or write on closed message pipe")})
	if pp.Has(id) {
		t.Fatalf("peer '%v' not removed: %v", id, pp)
	}
}

func TestBzzPeerPoolBothAddRemove(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, addr, pp, nil, nil)
	s.runHandshakes()

	id := s.Ids[0]
	if !pp.Has(id) {
		t.Fatalf("peer '%v' not added: %v", id, pp)
	}

	pp.Get(id).Drop()
	s.TestDisconnected(&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("p2p: read or write on closed message pipe")})
	if pp.Has(id) {
		t.Fatalf("peer '%v' not removed: %v", id, pp)
	}
}

func TestBzzPeerPoolNotAdd(t *testing.T) {
	addr := RandomAddr()
	pp := p2ptest.NewTestPeerPool()
	s := newBzzTester(t, addr, pp, nil, nil)

	id := s.Ids[0]
	s.testHandshake(correctBzzHandshake(addr), &bzzHandshake{0, 321, NewPeerAddrFromNodeId(id)}, &p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("network id mismatch 321 (!= 322)")})
	if pp.Has(id) {
		t.Fatalf("peer %v incorrectly added: %v", id, pp)
	}
}
