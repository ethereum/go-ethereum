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
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net"
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

const (
	maxPO     = 10
	maxPeerPO = 8
)

// TestInitialPeersMsg tests if peersMsg response to incoming subPeersMsg is correct
func TestSubPeersMsg(t *testing.T) {
	for po := 0; po < maxPO; po++ {
		for depth := 0; depth < maxPO; depth++ {
			t.Run(fmt.Sprintf("PO=%d,advetised depth=%d", po, depth), func(t *testing.T) {
				testSubPeersMsg(t, po, depth)
			})
		}
	}
}

// testSubPeersMsg tests that the correct set of peer info is sent
// to another peer after receiving their subPeersMsg request
func testSubPeersMsg(t *testing.T, peerPO, peerDepth int) {
	// generate random pivot address
	prvkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	pivotAddr := pot.NewAddressFromBytes(PrivateKeyToBzzKey(prvkey))
	// generate control peers address at peerPO wrt pivot
	peerAddr := pot.RandomAddressAt(pivotAddr, peerPO)
	// construct kademlia and hive
	to := NewKademlia(pivotAddr[:], NewKadParams())
	hive := NewHive(NewHiveParams(), to, nil)

	// expected addrs in peersMsg response
	var expBzzAddrs []*BzzAddr
	addrAt := func(a pot.Address, po int) []byte {
		b := pot.RandomAddressAt(a, po)
		return b[:]
	}
	connect := func(base pot.Address, po int) *BzzAddr {
		on := addrAt(base, po)
		peer := newDiscPeer(on)
		hive.On(peer)
		return peer.BzzAddr
	}
	register := func(base pot.Address, po int) {
		hive.Register(&BzzAddr{OAddr: addrAt(base, po)})
	}

	for po := maxPeerPO; po >= 0; po-- {
		// create a fake connected peer at po from peerAddr
		on := connect(peerAddr, po)
		// create a fake registered address at po from peerAddr
		register(peerAddr, po)
		// we collect expected peer addresses only up till peerPO
		if po < peerDepth {
			continue
		}
		expBzzAddrs = append(expBzzAddrs, on)
	}

	// create a special bzzBaseTester in which we can associate `enode.ID` to the `bzzAddr` we created above
	s, _, err := newBzzBaseTesterWithAddrs(t, prvkey, [][]byte{peerAddr[:]}, DiscoverySpec, hive.Run)
	if err != nil {
		t.Fatal(err)
	}

	// peerID to use in the protocol tester testExchange expect/trigger
	peerID := s.Nodes[0].ID()

	// now we need to wait until the tester's control peer appears in the hive
	// so the protocol started
	ticker := time.NewTicker(10 * time.Millisecond)
	attempts := 100
	for range ticker.C {
		if _, found := hive.peers[peerID]; found {
			break
		}
		attempts--
		if attempts == 0 {
			t.Fatal("timeout waiting for control peer to be in kademlia")
		}
	}

	// pivotDepth is the advertised depth of the pivot node we expect in the outgoing subPeersMsg
	pivotDepth := hive.saturation()
	// the test exchange is as follows:
	// 1. pivot sends to the control peer a `subPeersMsg` advertising its depth (ignored)
	// 2. peer sends to pivot a `subPeersMsg` advertising its own depth (arbitrarily chosen)
	// 3. pivot responds with `peersMsg` with the set of expected peers
	err = s.TestExchanges(
		p2ptest.Exchange{
			Label: "outgoing subPeersMsg",
			Expects: []p2ptest.Expect{
				{
					Code: 1,
					Msg:  &subPeersMsg{Depth: uint8(pivotDepth)},
					Peer: peerID,
				},
			},
		},
		p2ptest.Exchange{
			Label: "trigger subPeersMsg and expect peersMsg",
			Triggers: []p2ptest.Trigger{
				{
					Code: 1,
					Msg:  &subPeersMsg{Depth: uint8(peerDepth)},
					Peer: peerID,
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code:    0,
					Msg:     &peersMsg{Peers: expBzzAddrs},
					Peer:    peerID,
					Timeout: 100 * time.Millisecond,
				},
			},
		})

	// for values MaxPeerPO < peerPO < MaxPO the pivot has no peers to offer to the control peer
	// in this case, no peersMsg will be sent out, and we would run into a time out
	if err != nil {
		if len(expBzzAddrs) > 0 {
			t.Fatal(err)
		} else if err.Error() != "exchange #1 \"trigger subPeersMsg and expect peersMsg\": timed out" {
			t.Fatalf("expected timeout, got %v", err)
		}
	} else {
		if len(expBzzAddrs) == 0 {
			t.Fatalf("expected timeout, got no error")
		}
	}
}

// as we are not creating a real node via the protocol,
// we need to create the discovery peer objects for the additional kademlia
// nodes manually
func newDiscPeer(addr []byte) *Peer {
	pKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err.Error())
	}
	pubKey := pKey.PublicKey
	nod := enode.NewV4(&pubKey, net.IPv4(127, 0, 0, 1), 0, 0)
	bzzAddr := &BzzAddr{OAddr: addr, UAddr: []byte(nod.String())}
	id := nod.ID()
	p2pPeer := p2p.NewPeer(id, id.String(), nil)
	return NewPeer(&BzzPeer{
		Peer:    protocols.NewPeer(p2pPeer, &dummyMsgRW{}, DiscoverySpec),
		BzzAddr: bzzAddr,
	}, nil)
}

type dummyMsgRW struct{}

func (d *dummyMsgRW) ReadMsg() (p2p.Msg, error) {
	return p2p.Msg{}, nil
}
func (d *dummyMsgRW) WriteMsg(msg p2p.Msg) error {
	return nil
}
