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
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/pot"
)

/***
 *
 * - after connect, that outgoing subpeersmsg is sent
 *
 */
func TestDiscovery(t *testing.T) {
	params := NewHiveParams()
	s, pp, err := newHiveTester(t, params, 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	node := s.Nodes[0]
	raddr := NewAddr(node)
	pp.Register(raddr)

	// start the hive and wait for the connection
	pp.Start(s.Server)
	defer pp.Stop()

	// send subPeersMsg to the peer
	err = s.TestExchanges(p2ptest.Exchange{
		Label: "outgoing subPeersMsg",
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: 0},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestSubpeersMsg(t *testing.T) {

	testDepth := 2

	params := NewHiveParams()
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	node := enode.NewV4(&key.PublicKey, net.IP{127, 0, 0, 1}, 30303, 30303)
	addr := NewAddr(node)
	kad := NewKademlia(addr.OAddr, NewKadParams())
	hive := NewHive(params, kad, nil) // hive
	s := newBzzBaseTester(t, 1, addr, DiscoverySpec, hive.Run)
	pivot := s.Nodes[0]

	registerBzzAddr(0, hive, true)  // bin 0
	registerBzzAddr(1, hive, true)  // bin 1
	registerBzzAddr(3, hive, true)  // bin 3
	registerBzzAddr(4, hive, true)  // bin 4
	registerBzzAddr(3, hive, false) // add a known but not connected peer
	registerBzzAddr(1, hive, false) // add a known but not connected peer

	var expectedPeers []*BzzAddr
	hive.EachConn(hive.BaseAddr(), 256, func(p *Peer, po int) bool {
		if po < testDepth {
			expectedPeers = append(expectedPeers, p.BzzAddr)
		}
		return true
	})

	// start the hive and wait for the connection
	hive.Start(s.Server)
	defer hive.Stop()

	err = s.TestExchanges(p2ptest.Exchange{
		Label: "incoming subPeersMsg",
		Expects: []p2ptest.Expect{
			{
				Code: 0,
				Msg:  &peersMsg{Peers: expectedPeers},
				Peer: pivot.ID(),
			},
		},
		Triggers: []p2ptest.Trigger{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: uint8(testDepth)},
				Peer: pivot.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	return
}

func registerBzzAddr(po int, hive *Hive, on bool) {
	a := pot.RandomAddressAt(pot.NewAddressFromBytes(hive.BaseAddr()), po)
	bzzAddr := &BzzAddr{OAddr: a.Bytes(), UAddr: a.Bytes()}
	if on {
		peer := newDiscPeer(bzzAddr, a.String(), hive)
		hive.On(peer)
	} else {
		hive.Register(bzzAddr)
	}
}

func newDiscPeer(bzzAddr *BzzAddr, name string, hive *Hive) *Peer {
	p2pPeer := p2p.NewPeer(adapters.RandomNodeConfig().Node().ID(), name, nil)
	peer := NewPeer(&BzzPeer{
		Peer:      protocols.NewPeer(p2pPeer, &dummyMsgRW{}, DiscoverySpec),
		BzzAddr:   bzzAddr,
		LightNode: false},
		hive.Kademlia)
	return peer
}

type dummyMsgRW struct{}

func (d *dummyMsgRW) ReadMsg() (p2p.Msg, error) {
	return p2p.Msg{}, nil
}
func (d *dummyMsgRW) WriteMsg(msg p2p.Msg) error {
	return nil
}
