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
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
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

// TestSubpeersMsg tests that the correct set of peers is suggested
// to another peer with a given depth and kademlia
func TestSubpeersMsg(t *testing.T) {

	// This is the defined depth
	testDepth := 2

	// construct ProtocolTester and hive
	params := NewHiveParams()
	s, hive := newHiveTester(t, params, 1, nil)

	// register some addresses in specific bins (must coincide with testDepth)
	registerBzzAddr(0, hive, true)  // bin 0
	registerBzzAddr(0, hive, true)  // bin 0
	registerBzzAddr(1, hive, true)  // bin 1
	registerBzzAddr(1, hive, true)  // bin 1
	registerBzzAddr(3, hive, true)  // bin 3
	registerBzzAddr(4, hive, true)  // bin 4
	registerBzzAddr(3, hive, false) // add a known but not connected peer
	registerBzzAddr(1, hive, false) // add a known but not connected peer

	// start the hive
	hive.Start(s.Server)
	defer hive.Stop()

	// the pivot node is the only one from the ProtocolTester
	pivot := s.Nodes[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// we need to wait until the pivot node is actually connected to our hive
WAIT_PIVOT:
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for the pivot node to connect")
		case <-time.After(100 * time.Millisecond):
			if _, ok := hive.peers[pivot.ID()]; ok {
				break WAIT_PIVOT
			}
		}
	}

	// get BzzAddr of the pivot
	pivotAddress := hive.peers[pivot.ID()]
	pivotBzz := pivotAddress.BzzAddr.Over()

	// now we need to identify which peers are expected
	// iterate the hive's connection and only add peers below testDepth
	var expectedPeers []*BzzAddr
	hive.EachConn(hive.BaseAddr(), 256, func(p *Peer, po int) bool {
		if po < testDepth {
			// don't add the pivot node itself to expectedPeers;
			// the pivot node was not added as
			if !bytes.Equal(p.BzzAddr.Over(), b) {
				expectedPeers = append(expectedPeers, p.BzzAddr)
			}
		}
		return true
	})

	// the test exchange is as follows:
	// 1. Trigger a subPeersMsg from pivot to our hive
	// 2. Hive will respond with peersMsg with the set of expected peers
	err := s.TestExchanges(p2ptest.Exchange{
		Label: "incoming subPeersMsg",
		Triggers: []p2ptest.Trigger{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: uint8(testDepth)},
				Peer: pivot.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: uint8(testDepth)},
				Peer: pivot.ID(),
			},
			{
				Code: 0,
				Msg:  &peersMsg{Peers: expectedPeers},
				Peer: pivot.ID(),
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	return
}

// add the BzzAddr to the hive
func registerBzzAddr(po int, hive *Hive, on bool) {
	a := pot.RandomAddressAt(pot.NewAddressFromBytes(hive.BaseAddr()), po)
	bzzAddr := &BzzAddr{OAddr: a.Bytes(), UAddr: a.Bytes()}
	if on {
		// actually connect
		peer := newDiscPeer(bzzAddr, a.String(), hive)
		hive.On(peer)
	} else {
		// only add to address book
		hive.Register(bzzAddr)
	}
}

// as we are not creating a real node via the protocol,
// we need to create the discovery peer objects for the additional kademlia
// nodes manually
func newDiscPeer(bzzAddr *BzzAddr, name string, hive *Hive) *Peer {
	p2pPeer := p2p.NewPeer(adapters.RandomNodeConfig().Node().ID(), name, nil)
	peer := NewPeer(&BzzPeer{
		Peer:      protocols.NewPeer(p2pPeer, &dummyMsgRW{}, DiscoverySpec),
		BzzAddr:   bzzAddr,
		LightNode: false},
		hive.Kademlia)
	return peer
}

// we also need this dummy object otherwise at hive.Stop(),
// which will call `Drop` on all nodes, we will have null pointer errors,
// as the underlying `p2p.Peer` objects were not created
type dummyMsgRW struct{}

func (d *dummyMsgRW) ReadMsg() (p2p.Msg, error) {
	return p2p.Msg{}, nil
}
func (d *dummyMsgRW) WriteMsg(msg p2p.Msg) error {
	return nil
}
