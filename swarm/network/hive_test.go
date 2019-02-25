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

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/state"
)

func newHiveTester(t *testing.T, params *HiveParams, n int, store state.Store) (*bzzTester, *Hive) {
	// setup
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())
	pp := NewHive(params, to, store) // hive

	return newBzzBaseTester(t, n, addr, DiscoverySpec, pp.Run), pp
}

// TestRegisterAndConnect verifies that the protocol runs successfully
// and that the peer connection exists afterwards
func TestRegisterAndConnect(t *testing.T) {
	params := NewHiveParams()
	s, pp := newHiveTester(t, params, 1, nil)

	node := s.Nodes[0]
	raddr := NewAddr(node)
	pp.Register(raddr)

	// start the hive
	err := pp.Start(s.Server)
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
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := state.NewDBStore(dir) //start the hive with an empty dbstore
	if err != nil {
		t.Fatal(err)
	}

	params := NewHiveParams()
	s, pp := newHiveTester(t, params, 5, store)

	peers := make(map[string]bool)
	for _, node := range s.Nodes {
		raddr := NewAddr(node)
		pp.Register(raddr)
		peers[raddr.String()] = true
	}

	// start and stop the hive
	// the known peers should be saved upon stopping
	err = pp.Start(s.Server)
	if err != nil {
		t.Fatal(err)
	}
	pp.Stop()
	store.Close()

	// start the hive with an empty dbstore
	persistedStore, err := state.NewDBStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	s1, pp := newHiveTester(t, params, 0, persistedStore)

	// start the hive and check that we know of all expected peers
	pp.Start(s1.Server)
	i := 0
	pp.Kademlia.EachAddr(nil, 256, func(addr *BzzAddr, po int) bool {
		delete(peers, addr.String())
		i++
		return true
	})
	// TODO remove this line when verified that test passes
	time.Sleep(time.Second)
	if i != 5 {
		t.Fatalf("invalid number of entries: got %v, want %v", i, 5)
	}
	if len(peers) != 0 {
		t.Fatalf("%d peers left over: %v", len(peers), peers)
	}

}
