package network

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

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

func newBzzBaseTester(t *testing.T, n int, addr *bzzAddr, spec *protocols.Spec, run func(*bzzPeer) error) *bzzTester {
	cs := make(map[string]chan bool)

	srv := func(p *bzzPeer) error {
		defer close(cs[p.ID().String()])
		return run(p)
	}

	protocall := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		return srv(&bzzPeer{
			Peer:      protocols.NewPeer(p, rw, spec),
			localAddr: addr,
			bzzAddr:   NewAddrFromNodeId(&adapters.NodeId{NodeID: p.ID()}),
		})
	}

	s := p2ptest.NewProtocolTester(t, NewNodeIdFromAddr(addr), n, protocall)

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
	addr *bzzAddr
	cs   map[string]chan bool
}

func newBzzTester(t *testing.T, n int, addr *bzzAddr, pp *p2ptest.TestPeerPool, spec *protocols.Spec, services func(Peer) error) *bzzTester {

	extraservices := func(p *bzzPeer) error {
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
func (s *bzzTester) testHandshake(lhs, rhs *bzzHandshake, disconnects ...*p2ptest.Disconnect) {
	var peers []*adapters.NodeId
	id := NewNodeIdFromAddr(rhs.Addr)
	if len(disconnects) > 0 {
		for _, d := range disconnects {
			peers = append(peers, d.Peer)
		}
	} else {
		peers = []*adapters.NodeId{id}
	}

	s.TestExchanges(bzzHandshakeExchange(lhs, rhs, id)...)
	s.TestDisconnected(disconnects...)
}

func (s *bzzTester) runHandshakes(ids ...*adapters.NodeId) {
	if len(ids) == 0 {
		ids = s.Ids
	}
	for _, id := range ids {
		s.testHandshake(correctBzzHandshake(s.addr), correctBzzHandshake(NewAddrFromNodeId(id)))
		<-s.cs[id.NodeID.String()]
	}

}

func correctBzzHandshake(addr *bzzAddr) *bzzHandshake {
	return &bzzHandshake{
		Version:   0,
		NetworkId: 322,
		Addr:      addr,
	}
}

func TestBzzHandshakeNetworkIdMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{Version: 0, NetworkId: 321, Addr: NewAddrFromNodeId(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("network id mismatch 321 (!= 322)")},
	)
}

func TestBzzHandshakeVersionMismatch(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{Version: 1, NetworkId: 322, Addr: NewAddrFromNodeId(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("version mismatch 1 (!= 0)")},
	)
}

func TestBzzHandshakeSuccess(t *testing.T) {
	pp := p2ptest.NewTestPeerPool()
	addr := RandomAddr()
	s := newBzzTester(t, 1, addr, pp, nil, nil)
	defer s.Stop()

	id := s.Ids[0]
	s.testHandshake(
		correctBzzHandshake(addr),
		&bzzHandshake{Version: 0, NetworkId: 322, Addr: NewAddrFromNodeId(id)},
	)
}
