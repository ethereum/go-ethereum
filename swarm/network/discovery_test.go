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
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
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
	params := NewHiveParams()
	s, hive, err := newHiveTester(t, params, 7, nil)
	if err != nil {
		t.Fatal(err)
	}

	base := "11111111"
	hive.Kademlia = NewKademlia(pot.NewAddressFromString(base), NewKadParams())
	fmt.Println(hive)

	registerBzzAddr(s.Nodes[0], "00000011", hive, true)  // bin 0
	registerBzzAddr(s.Nodes[1], "10000000", hive, true)  // bin 1
	registerBzzAddr(s.Nodes[2], "11100000", hive, true)  // bin 3
	registerBzzAddr(s.Nodes[3], "11110000", hive, true)  // bin 4
	registerBzzAddr(s.Nodes[4], "10000001", hive, false) // add a known but not connected
	registerBzzAddr(s.Nodes[5], "11000001", hive, false) // add a known but not connected

	pivot := s.Nodes[6]
	registerBzzAddr(pivot, "11000000", hive, true) // bin 2

	// start the hive and wait for the connection
	hive.Start(s.Server)
	defer hive.Stop()

	err = s.TestExchanges(p2ptest.Exchange{
		Label: "incoming subPeersMsg",
		Triggers: []p2ptest.Trigger{
			{
				Code:    1,
				Msg:     &subPeersMsg{Depth: 2},
				Peer:    pivot.ID(),
				Timeout: 1 * time.Second,
			},
		},
	})

	fmt.Println(hive.Kademlia)
	if err != nil {
		t.Fatal(err)
	}
	return

}

func registerBzzAddr(node *enode.Node, kad string, hive *Hive, on bool) {
	a := pot.NewAddressFromString(kad)
	bzzAddr := &BzzAddr{OAddr: a, UAddr: []byte(node.String())}
	if on {
		peer := NewPeer(&BzzPeer{BzzAddr: bzzAddr, LightNode: false}, hive.Kademlia)
		hive.On(peer)
	} else {
		hive.Register(bzzAddr)
	}
}
