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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/state"
)

func newHiveTester(params *HiveParams, n int, store state.Store) (*bzzTester, *Hive, error) {
	// setup
	prvkey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	addr := PrivateKeyToBzzKey(prvkey)
	to := NewKademlia(addr, NewKadParams())
	pp := NewHive(params, to, store) // hive

	bt, err := newBzzBaseTester(n, prvkey, DiscoverySpec, pp.Run)
	if err != nil {
		return nil, nil, err
	}
	return bt, pp, nil
}

// TestRegisterAndConnect verifies that the protocol runs successfully
// and that the peer connection exists afterwards
func TestRegisterAndConnect(t *testing.T) {
	params := NewHiveParams()
	s, pp, err := newHiveTester(params, 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	node := s.Nodes[0]
	raddr := NewAddr(node)
	pp.Register(raddr)

	// start the hive
	err = pp.Start(s.Server)
	if err != nil {
		t.Fatal(err)
	}
	defer pp.Stop()

	// both hive connect and disconect check have time delays
	// therefore we need to verify that peer is connected
	// so that we are sure that the disconnect timeout doesn't complete
	// before the hive connect method is run at least once
	timeout := time.After(time.Second)
	for {
		select {
		case <-timeout:
			t.Fatalf("expected connection")
		default:
		}
		i := 0
		pp.Kademlia.EachConn(nil, 256, func(addr *Peer, po int) bool {
			i++
			return true
		})
		if i > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	// check that the connection actually exists
	// the timeout error means no disconnection events
	// were received within the a certain timeout
	err = s.TestDisconnected(&p2ptest.Disconnect{
		Peer:  s.Nodes[0].ID(),
		Error: nil,
	})

	if err == nil || err.Error() != "timed out waiting for peers to disconnect" {
		t.Fatalf("expected no disconnection event")
	}
}

// TestHiveStatePersistance creates a protocol simulation with n peers for a node
// After protocols complete, the node is shut down and the state is stored.
// Another simulation is created, where 0 nodes are created, but where the stored state is passed
// The test succeeds if all the peers from the stored state are known after the protocols of the
// second simulation have completed
//
// Actual connectivity is not in scope for this test, as the peers loaded from state are not known to
// the simulation; the test only verifies that the peers are known to the node
func TestHiveStatePersistance(t *testing.T) {
	dir, err := ioutil.TempDir("", "hive_test_store")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	const peersCount = 5

	startHive := func(t *testing.T, dir string) (h *Hive) {
		store, err := state.NewDBStore(dir)
		if err != nil {
			t.Fatal(err)
		}

		params := NewHiveParams()
		params.Discovery = false

		prvkey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		h = NewHive(params, NewKademlia(PrivateKeyToBzzKey(prvkey), NewKadParams()), store)
		s := p2ptest.NewProtocolTester(prvkey, 0, func(p *p2p.Peer, rw p2p.MsgReadWriter) error { return nil })

		if err := h.Start(s.Server); err != nil {
			t.Fatal(err)
		}
		return h
	}

	h1 := startHive(t, dir)
	peers := make(map[string]bool)
	for i := 0; i < peersCount; i++ {
		raddr := RandomAddr()
		h1.Register(raddr)
		peers[raddr.String()] = true
	}
	if err = h1.Stop(); err != nil {
		t.Fatal(err)
	}

	// start the hive and check that we know of all expected peers
	h2 := startHive(t, dir)
	defer func() {
		if err = h2.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	i := 0
	h2.Kademlia.EachAddr(nil, 256, func(addr *BzzAddr, po int) bool {
		delete(peers, addr.String())
		i++
		return true
	})
	if i != peersCount {
		t.Fatalf("invalid number of entries: got %v, want %v", i, peersCount)
	}
	if len(peers) != 0 {
		t.Fatalf("%d peers left over: %v", len(peers), peers)
	}
}
