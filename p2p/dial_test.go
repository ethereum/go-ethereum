// Copyright 2015 The go-ethereum Authors
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

package p2p

import (
	"encoding/binary"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/netutil"
)

func init() {
	spew.Config.Indent = "\t"
}

type dialtest struct {
	init   *dialstate // state before and after the test.
	rounds []round
}

type round struct {
	peers []*Peer // current peer set
	done  []task  // tasks that got done this round
	new   []task  // the result must match this one
}

func runDialTest(t *testing.T, test dialtest) {
	var (
		vtime   time.Time
		running int
	)
	pm := func(ps []*Peer) map[enode.ID]*Peer {
		m := make(map[enode.ID]*Peer)
		for _, p := range ps {
			m[p.ID()] = p
		}
		return m
	}
	for i, round := range test.rounds {
		for _, task := range round.done {
			running--
			if running < 0 {
				panic("running task counter underflow")
			}
			test.init.taskDone(task, vtime)
		}

		new := test.init.newTasks(running, pm(round.peers), vtime)
		if !sametasks(new, round.new) {
			t.Errorf("ERROR round %d: got %v\nwant %v\nstate: %v\nrunning: %v",
				i, spew.Sdump(new), spew.Sdump(round.new), spew.Sdump(test.init), spew.Sdump(running))
		}
		t.Logf("round %d new tasks: %s", i, strings.TrimSpace(spew.Sdump(new)))

		// Time advances by 16 seconds on every round.
		vtime = vtime.Add(16 * time.Second)
		running += len(new)
	}
}

type fakeTable []*enode.Node

func (t fakeTable) Self() *enode.Node                     { return new(enode.Node) }
func (t fakeTable) Close()                                {}
func (t fakeTable) LookupRandom() []*enode.Node           { return nil }
func (t fakeTable) Resolve(*enode.Node) *enode.Node       { return nil }
func (t fakeTable) ReadRandomNodes(buf []*enode.Node) int { return copy(buf, t) }

// This test checks that dynamic dials are launched from discovery results.
func TestDialStateDynDial(t *testing.T) {
	config := &Config{Logger: testlog.Logger(t, log.LvlTrace)}
	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, fakeTable{}, 5, config),
		rounds: []round{
			// A discovery query is launched.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
				},
				new: []task{&discoverTask{}},
			},
			// Dynamic dials are launched when it completes.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
				},
				done: []task{
					&discoverTask{results: []*enode.Node{
						newNode(uintID(2), nil), // this one is already connected and not dialed.
						newNode(uintID(3), nil),
						newNode(uintID(4), nil),
						newNode(uintID(5), nil),
						newNode(uintID(6), nil), // these are not tried because max dyn dials is 5
						newNode(uintID(7), nil), // ...
					}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
				},
			},
			// Some of the dials complete but no new ones are launched yet because
			// the sum of active dial count and dynamic peer count is == maxDynDials.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(4), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
				},
			},
			// No new dial tasks are launched in the this round because
			// maxDynDials has been reached.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(4), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(5), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
				},
				new: []task{
					&waitExpireTask{Duration: 19 * time.Second},
				},
			},
			// In this round, the peer with id 2 drops off. The query
			// results from last discovery lookup are reused.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(4), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(5), nil)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(6), nil)},
				},
			},
			// More peers (3,4) drop off and dial for ID 6 completes.
			// The last query result from the discovery lookup is reused
			// and a new one is spawned because more candidates are needed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(5), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(6), nil)},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(7), nil)},
					&discoverTask{},
				},
			},
			// Peer 7 is connected, but there still aren't enough dynamic peers
			// (4 out of 5). However, a discovery is already running, so ensure
			// no new is started.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(5), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(7), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(7), nil)},
				},
			},
			// Finish the running node discovery with an empty set. A new lookup
			// should be immediately requested.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(0), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(5), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(7), nil)}},
				},
				done: []task{
					&discoverTask{},
				},
				new: []task{
					&discoverTask{},
				},
			},
		},
	})
}

// Tests that bootnodes are dialed if no peers are connectd, but not otherwise.
func TestDialStateDynDialBootnode(t *testing.T) {
	config := &Config{
		BootstrapNodes: []*enode.Node{
			newNode(uintID(1), nil),
			newNode(uintID(2), nil),
			newNode(uintID(3), nil),
		},
		Logger: testlog.Logger(t, log.LvlTrace),
	}
	table := fakeTable{
		newNode(uintID(4), nil),
		newNode(uintID(5), nil),
		newNode(uintID(6), nil),
		newNode(uintID(7), nil),
		newNode(uintID(8), nil),
	}
	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, table, 5, config),
		rounds: []round{
			// 2 dynamic dials attempted, bootnodes pending fallback interval
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
					&discoverTask{},
				},
			},
			// No dials succeed, bootnodes still pending fallback interval
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
				},
			},
			// No dials succeed, bootnodes still pending fallback interval
			{},
			// No dials succeed, 2 dynamic dials attempted and 1 bootnode too as fallback interval was reached
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(1), nil)},
				},
			},
			// No dials succeed, 2nd bootnode is attempted
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(1), nil)},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(2), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
				},
			},
			// No dials succeed, 3rd bootnode is attempted
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(2), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
				},
			},
			// No dials succeed, 1st bootnode is attempted again, expired random nodes retried
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
				},
				new: []task{},
			},
			// Random dial succeeds, no more bootnodes are attempted
			{
				new: []task{
					&waitExpireTask{3 * time.Second},
				},
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(4), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(1), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
				},
			},
		},
	})
}

func TestDialStateDynDialFromTable(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	table := fakeTable{
		newNode(uintID(1), nil),
		newNode(uintID(2), nil),
		newNode(uintID(3), nil),
		newNode(uintID(4), nil),
		newNode(uintID(5), nil),
		newNode(uintID(6), nil),
		newNode(uintID(7), nil),
		newNode(uintID(8), nil),
	}

	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, table, 10, &Config{Logger: testlog.Logger(t, log.LvlTrace)}),
		rounds: []round{
			// 5 out of 8 of the nodes returned by ReadRandomNodes are dialed.
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(1), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(2), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
					&discoverTask{},
				},
			},
			// Dialing nodes 1,2 succeeds. Dials from the lookup are launched.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(1), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(2), nil)},
					&discoverTask{results: []*enode.Node{
						newNode(uintID(10), nil),
						newNode(uintID(11), nil),
						newNode(uintID(12), nil),
					}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(10), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(11), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(12), nil)},
					&discoverTask{},
				},
			},
			// Dialing nodes 3,4,5 fails. The dials from the lookup succeed.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(10), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(11), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(12), nil)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(3), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(5), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(10), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(11), nil)},
					&dialTask{flags: dynDialedConn, dest: newNode(uintID(12), nil)},
				},
			},
			// Waiting for expiry. No waitExpireTask is launched because the
			// discovery query is still running.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(10), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(11), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(12), nil)}},
				},
			},
			// Nodes 3,4 are not tried again because only the first two
			// returned random nodes (nodes 1,2) are tried and they're
			// already connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(10), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(11), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(12), nil)}},
				},
			},
		},
	})
}

func newNode(id enode.ID, ip net.IP) *enode.Node {
	var r enr.Record
	if ip != nil {
		r.Set(enr.IP(ip))
	}
	return enode.SignNull(&r, id)
}

// This test checks that candidates that do not match the netrestrict list are not dialed.
func TestDialStateNetRestrict(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	table := fakeTable{
		newNode(uintID(1), net.ParseIP("127.0.0.1")),
		newNode(uintID(2), net.ParseIP("127.0.0.2")),
		newNode(uintID(3), net.ParseIP("127.0.0.3")),
		newNode(uintID(4), net.ParseIP("127.0.0.4")),
		newNode(uintID(5), net.ParseIP("127.0.2.5")),
		newNode(uintID(6), net.ParseIP("127.0.2.6")),
		newNode(uintID(7), net.ParseIP("127.0.2.7")),
		newNode(uintID(8), net.ParseIP("127.0.2.8")),
	}
	restrict := new(netutil.Netlist)
	restrict.Add("127.0.2.0/24")

	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, table, 10, &Config{NetRestrict: restrict}),
		rounds: []round{
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: table[4]},
					&discoverTask{},
				},
			},
		},
	})
}

// This test checks that static dials are launched.
func TestDialStateStaticDial(t *testing.T) {
	config := &Config{
		StaticNodes: []*enode.Node{
			newNode(uintID(1), nil),
			newNode(uintID(2), nil),
			newNode(uintID(3), nil),
			newNode(uintID(4), nil),
			newNode(uintID(5), nil),
		},
		Logger: testlog.Logger(t, log.LvlTrace),
	}
	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, fakeTable{}, 0, config),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(3), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(5), nil)},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(3), nil)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(3), nil)},
				},
			},
			// No new dial tasks are launched because all static
			// nodes are now connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(4), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(5), nil)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(4), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(5), nil)},
				},
				new: []task{
					&waitExpireTask{Duration: 19 * time.Second},
				},
			},
			// Wait a round for dial history to expire, no new tasks should spawn.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(2), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(4), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(5), nil)}},
				},
			},
			// If a static node is dropped, it should be immediately redialed,
			// irrespective whether it was originally static or dynamic.
			{
				done: []task{
					&waitExpireTask{Duration: 19 * time.Second},
				},
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(3), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(5), nil)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(2), nil)},
				},
			},
		},
	})
}

// This test checks that past dials are not retried for some time.
func TestDialStateCache(t *testing.T) {
	config := &Config{
		StaticNodes: []*enode.Node{
			newNode(uintID(1), nil),
			newNode(uintID(2), nil),
			newNode(uintID(3), nil),
		},
		Logger: testlog.Logger(t, log.LvlTrace),
	}
	runDialTest(t, dialtest{
		init: newDialState(enode.ID{}, fakeTable{}, 0, config),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: nil,
				new: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(1), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(2), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(3), nil)},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(2), nil)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(1), nil)},
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(2), nil)},
				},
			},
			// A salvage task is launched to wait for node 3's history
			// entry to expire.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(2), nil)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(3), nil)},
				},
				new: []task{
					&waitExpireTask{Duration: 19 * time.Second},
				},
			},
			// Still waiting for node 3's entry to expire in the cache.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(2), nil)}},
				},
			},
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(2), nil)}},
				},
			},
			// The cache entry for node 3 has expired and is retried.
			{
				done: []task{
					&waitExpireTask{Duration: 19 * time.Second},
				},
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(1), nil)}},
					{rw: &conn{flags: staticDialedConn, node: newNode(uintID(2), nil)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: newNode(uintID(3), nil)},
				},
			},
		},
	})
}

func TestDialResolve(t *testing.T) {
	config := &Config{
		Logger: testlog.Logger(t, log.LvlTrace),
		Dialer: TCPDialer{&net.Dialer{Deadline: time.Now().Add(-5 * time.Minute)}},
	}
	resolved := newNode(uintID(1), net.IP{127, 0, 55, 234})
	table := &resolveMock{answer: resolved}
	state := newDialState(enode.ID{}, table, 0, config)

	// Check that the task is generated with an incomplete ID.
	dest := newNode(uintID(1), nil)
	state.addStatic(dest)
	tasks := state.newTasks(0, nil, time.Time{})
	if !reflect.DeepEqual(tasks, []task{&dialTask{flags: staticDialedConn, dest: dest}}) {
		t.Fatalf("expected dial task, got %#v", tasks)
	}

	// Now run the task, it should resolve the ID once.
	srv := &Server{ntab: table, log: config.Logger, Config: *config}
	tasks[0].Do(srv)
	if !reflect.DeepEqual(table.resolveCalls, []*enode.Node{dest}) {
		t.Fatalf("wrong resolve calls, got %v", table.resolveCalls)
	}

	// Report it as done to the dialer, which should update the static node record.
	state.taskDone(tasks[0], time.Now())
	if state.static[uintID(1)].dest != resolved {
		t.Fatalf("state.dest not updated")
	}
}

// compares task lists but doesn't care about the order.
func sametasks(a, b []task) bool {
	if len(a) != len(b) {
		return false
	}
next:
	for _, ta := range a {
		for _, tb := range b {
			if reflect.DeepEqual(ta, tb) {
				continue next
			}
		}
		return false
	}
	return true
}

func uintID(i uint32) enode.ID {
	var id enode.ID
	binary.BigEndian.PutUint32(id[:], i)
	return id
}

// implements discoverTable for TestDialResolve
type resolveMock struct {
	resolveCalls []*enode.Node
	answer       *enode.Node
}

func (t *resolveMock) Resolve(n *enode.Node) *enode.Node {
	t.resolveCalls = append(t.resolveCalls, n)
	return t.answer
}

func (t *resolveMock) Self() *enode.Node                     { return new(enode.Node) }
func (t *resolveMock) Close()                                {}
func (t *resolveMock) LookupRandom() []*enode.Node           { return nil }
func (t *resolveMock) ReadRandomNodes(buf []*enode.Node) int { return 0 }
