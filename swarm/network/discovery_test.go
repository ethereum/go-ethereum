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
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
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

// TestSubpeersMsg runs testSubpeersMsg multiple times
func TestSubpeersMsg(t *testing.T) {
	repetitions := 10
	for r := 0; r < repetitions; r++ {
		t.Run(fmt.Sprintf("Test Run number %d", r), testSubpeersMsg)
	}
}

// testSubpeersMsg tests that the correct set of peers is suggested
// to another peer with a given depth and kademlia
func testSubpeersMsg(t *testing.T) {

	// construct ProtocolTester and hive
	params := NewHiveParams()
	// setup the hive
	prvkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := PrivateKeyToBzzKey(prvkey)
	to := NewKademlia(addr, NewKadParams())
	// create the hive
	hive := NewHive(params, to, nil)

	// we will use a set of preconstructed addresses
	var bzzAddrs []*BzzAddr
	// define a number of nodes for the test
	nodeCount := 12
	// for every of these nodes, add to the hive's connections...
	for i := 0; i < nodeCount; i++ {
		// ...create a BzzAddr and connect it in the hive (`hive.On(p)`)
		a, err := registerBzzAddr(hive)
		if err != nil {
			t.Fatal(err)
		}
		// also store the actual bzzAddr into the slice
		bzzAddrs = append(bzzAddrs, a)
	}

	// create a channel. We will wait later...
	waitC := make(chan struct{})

	// create a special bzzBaseTester in which we can associate `enode.ID` to the `bzzAddr` we created above
	s, err := newPreconnectedBzzBaseTester(t, waitC, bzzAddrs, nodeCount, prvkey, DiscoverySpec, hive.Run)
	if err != nil {
		t.Fatal(err)
	}

	// start the hive
	hive.Start(s.Server)
	defer hive.Stop()

	// ...so we wait here until all connections have been established
	<-waitC

	// choose a control node
	control := s.Nodes[0]

	// get BzzAddr of the control
	controlBzz := hive.peers[control.ID()].Over()
	// build a control kademlia for the control node from the address pool
	// we use this so we can identify the actual `controlDepth` of the control node
	controlKad := NewKademlia(controlBzz, NewKadParams())
	for _, p := range bzzAddrs {
		if !bytes.Equal(p.Over(), controlBzz) {
			controlKad.On(NewPeer(&BzzPeer{nil, p, time.Now(), false}, controlKad))
		}
	}
	// to be fully correct, even the pivot's hive should be added
	controlKad.On(NewPeer(&BzzPeer{nil, &BzzAddr{OAddr: hive.BaseAddr(), UAddr: hive.BaseAddr()}, time.Now(), false}, controlKad))
	// now we can evaluate the depth of the control node, which we need...
	controlDepth := controlKad.NeighbourhoodDepth()

	// ...to identify which peers are expected to be advertized
	// iterate the hive's connection and only add peers below controlDepth
	var expectedPeers []*BzzAddr
	hive.EachConn(controlBzz, 255, func(p *Peer, po int) bool {
		if po < controlDepth {
			return false
		}
		expectedPeers = append(expectedPeers, p.BzzAddr)
		return true
	})

	// this is the hive's depth, which will be sent first to the control node initiating the test exchanges
	hiveDepth := hive.NeighbourhoodDepth()

	// if the controlDepth is 0, nothing will happen, so in this case artificially set it to 2
	if controlDepth == 0 {
		controlDepth = 2
	}

	// the test exchange is as follows:
	// 1. Wait for a `subPeersMsg` advertizing the hive's depth from our hive to our control node
	// 2. Trigger a `suPeersMsg` from the control node advertizing its own depth
	// 3. Hive will respond with peersMsg with the set of expected peers
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

	close(waitC)
}

// add the BzzAddr to the hive
func registerBzzAddr(hive *Hive) (*BzzAddr, error) {
	addr := pot.RandomAddress()
	pKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pubKey := pKey.PublicKey
	nod := enode.NewV4(&pubKey, net.IPv4(127, 0, 0, 1), 0, 0)
	bzzAddr := &BzzAddr{OAddr: addr.Bytes(), UAddr: []byte(nod.String())}
	// actually connect
	peer := newDiscPeer(bzzAddr, nod.ID(), hive)
	hive.On(peer)
	return bzzAddr, nil
}

// as we are not creating a real node via the protocol,
// we need to create the discovery peer objects for the additional kademlia
// nodes manually
func newDiscPeer(bzzAddr *BzzAddr, id enode.ID, hive *Hive) *Peer {
	p2pPeer := p2p.NewPeer(id, id.String(), nil)
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
