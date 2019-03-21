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
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
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

	// construct ProtocolTester and hive
	params := NewHiveParams()
	// setup
	prvkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := PrivateKeyToBzzKey(prvkey)
	to := NewKademlia(addr, NewKadParams())
	hive := NewHive(params, to, nil) // hive

	numOfTestNodes := 12

	s, err := newBzzBaseTester(t, numOfTestNodes, prvkey, DiscoverySpec, hive.Run)
	if err != nil {
		t.Fatal(err)
	}

	// the control node is the only one from the ProtocolTester
	control := s.Nodes[numOfTestNodes-1]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// start the hive
	hive.Start(s.Server)
	defer hive.Stop()

	// we need to wait until the control node is actually connected to our hive
WAIT_PIVOT:
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timed out waiting for the control node to connect")
		case <-time.After(100 * time.Millisecond):
			if len(hive.peers) == len(s.BzzAddrs) {
				break WAIT_PIVOT
			}
		}
	}

	// get BzzAddr of the control
	controlBzz := s.BzzAddrs[control.ID()].Over()

	controlKad := NewKademlia(controlBzz, NewKadParams())
	for _, p := range s.BzzAddrs {
		if !bytes.Equal(p.Over(), controlBzz) {
			controlKad.On(NewPeer(&BzzPeer{nil, p, time.Now(), false}, controlKad))
		}
	}
	controlDepth := controlKad.NeighbourhoodDepth()

	// now we need to identify which peers are expected
	// iterate the hive's connection and only add peers below testDepth
	var expectedPeers []*BzzAddr
	hive.EachConn(controlBzz, 255, func(p *Peer, po int) bool {
		if po < controlDepth {
			return false
		}
		// don't add the control node itself to expectedPeers;
		// the control node was not added as
		if !bytes.Equal(p.BzzAddr.Over(), controlBzz) {
			expectedPeers = append(expectedPeers, p.BzzAddr)
		}
		return true
	})

	hiveDepth := hive.NeighbourhoodDepth()

	// the test exchange is as follows:
	// 1. Trigger a subPeersMsg from control to our hive
	// 2. Hive will respond with peersMsg with the set of expected peers
	if controlDepth == 0 {
		controlDepth = 1
	}

	err = s.TestExchanges(p2ptest.Exchange{
		Label: "incoming subPeersMsg",
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: uint8(hiveDepth)},
				Peer: control.ID(),
			},
		},
	},
		p2ptest.Exchange{
			Label: "trigger subPeers and receive peersMsg",
			Triggers: []p2ptest.Trigger{
				{
					Code:    1,
					Msg:     &subPeersMsg{Depth: uint8(controlDepth)},
					Peer:    control.ID(),
					Timeout: 3 * time.Second,
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code:    0,
					Msg:     &peersMsg{Peers: expectedPeers},
					Peer:    control.ID(),
					Timeout: 3 * time.Second,
				},
			},
		})

	// for some configurations, there will be no advertised peers due to the
	// distance of the control peer to the hive and the set of connected peers
	// in this case, no peersMsg will be sent out, and we would run into a time out
	// catch this edge case
	if len(expectedPeers) == 0 {
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatal("expected timeout but didn't")
		}
		return
	}

	if err != nil {
		t.Fatal(err)
	}
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
	p2pPeer := p2p.NewPeer(adapters.RandomNodeConfig().ID, name, nil)
	return NewPeer(&BzzPeer{
		Peer:      protocols.NewPeer(p2pPeer, &dummyMsgRW{}, DiscoverySpec),
		BzzAddr:   bzzAddr,
		LightNode: false},
		hive.Kademlia)
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
