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
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/p2p/discover"
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
	pm := func(ps []*Peer) map[discover.NodeID]*Peer {
		m := make(map[discover.NodeID]*Peer)
		for _, p := range ps {
			m[p.rw.id] = p
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
			t.Errorf("round %d: new tasks mismatch:\ngot %v\nwant %v\nstate: %v\nrunning: %v\n",
				i, spew.Sdump(new), spew.Sdump(round.new), spew.Sdump(test.init), spew.Sdump(running))
		}

		// Time advances by 16 seconds on every round.
		vtime = vtime.Add(16 * time.Second)
		running += len(new)
	}
}

type fakeTable []*discover.Node

func (t fakeTable) Self() *discover.Node                     { return new(discover.Node) }
func (t fakeTable) Close()                                   {}
func (t fakeTable) Lookup(discover.NodeID) []*discover.Node  { return nil }
func (t fakeTable) Resolve(discover.NodeID) *discover.Node   { return nil }
func (t fakeTable) ReadRandomNodes(buf []*discover.Node) int { return copy(buf, t) }

// This test checks that dynamic dials are launched from discovery results.
func TestDialStateDynDial(t *testing.T) {
	runDialTest(t, dialtest{
		init: newDialState(nil, nil, fakeTable{}, 5, nil),
		rounds: []round{
			// A discovery query is launched.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{&discoverTask{}},
			},
			// Dynamic dials are launched when it completes.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&discoverTask{results: []*discover.Node{
						{ID: uintID(2)}, // this one is already connected and not dialed.
						{ID: uintID(3)},
						{ID: uintID(4)},
						{ID: uintID(5)},
						{ID: uintID(6)}, // these are not tried because max dyn dials is 5
						{ID: uintID(7)}, // ...
					}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// Some of the dials complete but no new ones are launched yet because
			// the sum of active dial count and dynamic peer count is == maxDynDials.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(3)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(4)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
				},
			},
			// No new dial tasks are launched in the this round because
			// maxDynDials has been reached.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(3)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(4)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// In this round, the peer with id 2 drops off. The query
			// results from last discovery lookup are reused.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(3)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(4)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(6)}},
				},
			},
			// More peers (3,4) drop off and dial for ID 6 completes.
			// The last query result from the discovery lookup is reused
			// and a new one is spawned because more candidates are needed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(6)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(7)}},
					&discoverTask{},
				},
			},
			// Peer 7 is connected, but there still aren't enough dynamic peers
			// (4 out of 5). However, a discovery is already running, so ensure
			// no new is started.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(5)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(7)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(7)}},
				},
			},
			// Finish the running node discovery with an empty set. A new lookup
			// should be immediately requested.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(0)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(5)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(7)}},
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
	bootnodes := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
	}
	table := fakeTable{
		{ID: uintID(4)},
		{ID: uintID(5)},
		{ID: uintID(6)},
		{ID: uintID(7)},
		{ID: uintID(8)},
	}
	runDialTest(t, dialtest{
		init: newDialState(nil, bootnodes, table, 5, nil),
		rounds: []round{
			// 2 dynamic dials attempted, bootnodes pending fallback interval
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
					&discoverTask{},
				},
			},
			// No dials succeed, bootnodes still pending fallback interval
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// No dials succeed, bootnodes still pending fallback interval
			{},
			// No dials succeed, 2 dynamic dials attempted and 1 bootnode too as fallback interval was reached
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// No dials succeed, 2nd bootnode is attempted
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(2)}},
				},
			},
			// No dials succeed, 3rd bootnode is attempted
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(2)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
			// No dials succeed, 1st bootnode is attempted again, expired random nodes retried
			{
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// Random dial succeeds, no more bootnodes are attempted
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(4)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
		},
	})
}

func TestDialStateDynDialFromTable(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	table := fakeTable{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
		{ID: uintID(4)},
		{ID: uintID(5)},
		{ID: uintID(6)},
		{ID: uintID(7)},
		{ID: uintID(8)},
	}

	runDialTest(t, dialtest{
		init: newDialState(nil, nil, table, 10, nil),
		rounds: []round{
			// 5 out of 8 of the nodes returned by ReadRandomNodes are dialed.
			{
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
					&discoverTask{},
				},
			},
			// Dialing nodes 1,2 succeeds. Dials from the lookup are launched.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&discoverTask{results: []*discover.Node{
						{ID: uintID(10)},
						{ID: uintID(11)},
						{ID: uintID(12)},
					}},
				},
				new: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(10)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(11)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(12)}},
					&discoverTask{},
				},
			},
			// Dialing nodes 3,4,5 fails. The dials from the lookup succeed.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(10)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(11)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
				done: []task{
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(5)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(10)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(11)}},
					&dialTask{flags: dynDialedConn, dest: &discover.Node{ID: uintID(12)}},
				},
			},
			// Waiting for expiry. No waitExpireTask is launched because the
			// discovery query is still running.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(10)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(11)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
			},
			// Nodes 3,4 are not tried again because only the first two
			// returned random nodes (nodes 1,2) are tried and they're
			// already connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(10)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(11)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
			},
		},
	})
}

// This test checks that candidates that do not match the netrestrict list are not dialed.
func TestDialStateNetRestrict(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	table := fakeTable{
		{ID: uintID(1), IP: net.ParseIP("127.0.0.1")},
		{ID: uintID(2), IP: net.ParseIP("127.0.0.2")},
		{ID: uintID(3), IP: net.ParseIP("127.0.0.3")},
		{ID: uintID(4), IP: net.ParseIP("127.0.0.4")},
		{ID: uintID(5), IP: net.ParseIP("127.0.2.5")},
		{ID: uintID(6), IP: net.ParseIP("127.0.2.6")},
		{ID: uintID(7), IP: net.ParseIP("127.0.2.7")},
		{ID: uintID(8), IP: net.ParseIP("127.0.2.8")},
	}
	restrict := new(netutil.Netlist)
	restrict.Add("127.0.2.0/24")

	runDialTest(t, dialtest{
		init: newDialState(nil, nil, table, 10, restrict),
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
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
		{ID: uintID(4)},
		{ID: uintID(5)},
	}

	runDialTest(t, dialtest{
		init: newDialState(wantStatic, nil, fakeTable{}, 0, nil),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
			// No new dial tasks are launched because all static
			// nodes are now connected.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(4)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(5)}},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Wait a round for dial history to expire, no new tasks should spawn.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(4)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
			},
			// If a static node is dropped, it should be immediately redialed,
			// irrespective whether it was originally static or dynamic.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(3)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(4)}},
				},
			},
		},
	})
}

// This test checks that past dials are not retried for some time.
func TestDialStateCache(t *testing.T) {
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
	}

	runDialTest(t, dialtest{
		init: newDialState(wantStatic, nil, fakeTable{}, 0, nil),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: nil,
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{rw: &conn{flags: staticDialedConn, id: uintID(1)}},
					{rw: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(1)}},
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(2)}},
				},
			},
			// A salvage task is launched to wait for node 3's history
			// entry to expire.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Still waiting for node 3's entry to expire in the cache.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
			},
			// The cache entry for node 3 has expired and is retried.
			{
				peers: []*Peer{
					{rw: &conn{flags: dynDialedConn, id: uintID(1)}},
					{rw: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{flags: staticDialedConn, dest: &discover.Node{ID: uintID(3)}},
				},
			},
		},
	})
}

func TestDialResolve(t *testing.T) {
	resolved := discover.NewNode(uintID(1), net.IP{127, 0, 55, 234}, 3333, 4444)
	table := &resolveMock{answer: resolved}
	state := newDialState(nil, nil, table, 0, nil)

	// Check that the task is generated with an incomplete ID.
	dest := discover.NewNode(uintID(1), nil, 0, 0)
	state.addStatic(dest)
	tasks := state.newTasks(0, nil, time.Time{})
	if !reflect.DeepEqual(tasks, []task{&dialTask{flags: staticDialedConn, dest: dest}}) {
		t.Fatalf("expected dial task, got %#v", tasks)
	}

	// Now run the task, it should resolve the ID once.
	config := Config{Dialer: TCPDialer{&net.Dialer{Deadline: time.Now().Add(-5 * time.Minute)}}}
	srv := &Server{ntab: table, Config: config}
	tasks[0].Do(srv)
	if !reflect.DeepEqual(table.resolveCalls, []discover.NodeID{dest.ID}) {
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

func uintID(i uint32) discover.NodeID {
	var id discover.NodeID
	binary.BigEndian.PutUint32(id[:], i)
	return id
}

// implements discoverTable for TestDialResolve
type resolveMock struct {
	resolveCalls []discover.NodeID
	answer       *discover.Node
}

func (t *resolveMock) Resolve(id discover.NodeID) *discover.Node {
	t.resolveCalls = append(t.resolveCalls, id)
	return t.answer
}

func (t *resolveMock) Self() *discover.Node                     { return new(discover.Node) }
func (t *resolveMock) Close()                                   {}
func (t *resolveMock) Bootstrap([]*discover.Node)               {}
func (t *resolveMock) Lookup(discover.NodeID) []*discover.Node  { return nil }
func (t *resolveMock) ReadRandomNodes(buf []*discover.Node) int { return 0 }
