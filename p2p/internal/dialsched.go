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

package p2pint

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

/*

Dial Candidate Selection Algorithm

This algorithm is responsible for picking dynamic dial candidates from
various sources.

- Random nodes from the the discovery table. Nodes in the discovery
  table are usually long-lived, and querying the local table is fast.
- Results from random discovery lookups.
- Future: Results from discovery topic queries. Protocols may declare
  topic hashes that they wish to find peers for.
- Suggestions drawn from the (optional) PeerSuggestions channel for
  each protocol. While protocol suggestions take precedence over other
  sources, random connections are still required for bootstrapping and
  attack resistance.

The algorithm randomises output from available sources. There
are more potential candidates than available connection slots. Not all
sources can provide equally many candidates. Dial candidates should be
a uniform selection out of all potential candidates.

Avoid 'close nodes' bias. The XOR distance metric used by p2p/discover
provides a useful network structure for the DHT, but can lead to less
than ideal TCP connectivity if used naively. Since the discovery table
prefers close nodes, the candidate selection algorithm must balance
distances to overcome this bias.

Honour per-protocol connection limits. If enough nodes are connected
to satisfy the maximum number of peers for a given protocol, no new
nodes need to be found.
*/

const (
	// This is the amount of time spent waiting in between
	// redialing a certain node.
	dialHistoryExpiration = 30 * time.Second

	// Discovery lookups are throttled and can only run
	// once every few seconds.
	lookupInterval = 4 * time.Second
)

// DialState schedules dials, discovery lookups and collects
// peer sugggestions from protocols.
// It gets a chance to compute new tasks on every iteration
// of the main loop in Server.run.
type DialState struct {
	peers       *PeerSet
	ntab        DiscoverTable
	maxDynDials int // per protocol

	lookupRunning bool
	bootstrapped  bool
	lookupBuf     []*discover.Node // current discovery lookup results
	randomNodes   []*discover.Node // filled from Table

	dialing map[discover.NodeID]Flag
	static  map[discover.NodeID]*discover.Node
	hist    *dialHistory
}

// the dial history remembers recent dials.
type dialHistory []pastDial

// pastDial is an entry in the dial history.
type pastDial struct {
	id  discover.NodeID
	exp time.Time
}

type DiscoverTable interface {
	Self() *discover.Node
	Close()
	Bootstrap([]*discover.Node)
	Lookup(target discover.NodeID) []*discover.Node
	ReadRandomNodes([]*discover.Node) int
}

// A DialTask is generated for each node that is dialed.
type DialTask struct {
	Flags Flag
	Dest  *discover.Node
}

// DiscoverTask runs discovery table operations.
// Only one DiscoverTask is active at any time.
//
// If bootstrap is true, the task runs Table.Bootstrap,
// otherwise it performs a random lookup and leaves the
// results in the task.
type DiscoverTask struct {
	Bootstrap bool
	Results   []*discover.Node
}

// A WaitExpireTask is generated if there are no other tasks
// running and the dial history is non-empty. This ensures that
// the loop executing the tasks keeps ticking.
type WaitExpireTask struct {
	time.Duration
}

func NewDialState(ps *PeerSet, ntab DiscoverTable, maxdyn int) *DialState {
	s := &DialState{
		peers:       ps,
		maxDynDials: maxdyn,
		ntab:        ntab,
		static:      make(map[discover.NodeID]*discover.Node),
		dialing:     make(map[discover.NodeID]Flag),
		randomNodes: make([]*discover.Node, maxdyn/2),
		hist:        new(dialHistory),
	}
	return s
}

func (s *DialState) AddStatic(n *discover.Node) {
	s.static[n.ID] = n
}

// newTasks is called from the main loop to new tasks.
func (s *DialState) NewTasks(nRunning int, now time.Time) []interface{} {
	var newtasks []interface{}
	addDial := func(flag Flag, n *discover.Node) bool {
		_, dialing := s.dialing[n.ID]
		if dialing || s.peers.IsConnected(n.ID) || s.hist.contains(n.ID) {
			return false
		}
		s.dialing[n.ID] = flag
		newtasks = append(newtasks, &DialTask{Flags: flag, Dest: n})
		return true
	}

	// Compute number of dynamic dials necessary at this point.
	needDynDials := s.needDynDials()

	// Expire the dial history on every invocation.
	s.hist.expire(now)

	// Create dials for static nodes if they are not connected.
	for _, n := range s.static {
		addDial(StaticDialedConn, n)
	}

	// Use random nodes from the table for half of the necessary
	// dynamic dials.
	randomCandidates := needDynDials / 2
	if randomCandidates > 0 && s.bootstrapped {
		n := s.ntab.ReadRandomNodes(s.randomNodes)
		for i := 0; i < randomCandidates && i < n; i++ {
			if addDial(DynDialedConn, s.randomNodes[i]) {
				needDynDials--
			}
		}
	}
	// Create dynamic dials from random lookup results, removing tried
	// items from the result buffer.
	i := 0
	for ; i < len(s.lookupBuf) && needDynDials > 0; i++ {
		if addDial(DynDialedConn, s.lookupBuf[i]) {
			needDynDials--
		}
	}
	s.lookupBuf = s.lookupBuf[:copy(s.lookupBuf, s.lookupBuf[i:])]
	// Launch a discovery lookup if more candidates are needed. The
	// first discoverTask bootstraps the table and won't return any
	// results.
	if len(s.lookupBuf) < needDynDials && !s.lookupRunning {
		s.lookupRunning = true
		newtasks = append(newtasks, &DiscoverTask{Bootstrap: !s.bootstrapped})
	}

	// Launch a timer to wait for the next node to expire if all
	// candidates have been tried and no task is currently active.
	// This should prevent cases where the dialer logic is not ticked
	// because there are no pending events.
	if nRunning == 0 && len(newtasks) == 0 && s.hist.Len() > 0 {
		t := &WaitExpireTask{s.hist.min().exp.Sub(now)}
		newtasks = append(newtasks, t)
	}
	return newtasks
}

// TaskDone should be called when a task has finished.
func (s *DialState) taskDone(t interface{}, now time.Time) {
	switch t := t.(type) {
	case *DialTask:
		s.hist.add(t.Dest.ID, now.Add(dialHistoryExpiration))
		delete(s.dialing, t.Dest.ID)
	case *DiscoverTask:
		if t.Bootstrap {
			s.bootstrapped = true
		}
		s.lookupRunning = false
		s.lookupBuf = append(s.lookupBuf, t.Results...)
	case *WaitExpireTask:
		// nothing to do here
	default:
		panic("unknown task type")
	}
}

// computes number of required dynamic dials
func (s *DialState) needDynDials() int {
	need := s.maxDynDials
	for _, p := range s.peers.NumDynPeers() {
		need--
	}
	for _, flag := range s.dialing {
		if flag&DynDialedConn != 0 {
v			need--
		}
	}
	return need
}

func (t *DialTask) String() string {
	return fmt.Sprintf("%v %x %v:%d", t.Flags, t.Dest.ID[:8], t.Dest.IP, t.Dest.TCP)
}

func (t *DiscoverTask) String() (s string) {
	if t.bootstrap {
		s = "discovery bootstrap"
	} else {
		s = "discovery lookup"
	}
	if len(t.results) > 0 {
		s += fmt.Sprintf(" (%d results)", len(t.results))
	}
	return s
}

func (t *WaitExpireTask) String() string {
	return fmt.Sprintf("wait for dial hist expire (%v)", t.Duration)
}

// Use only these methods to access or modify dialHistory.
func (h dialHistory) min() pastDial {
	return h[0]
}
func (h *dialHistory) add(id discover.NodeID, exp time.Time) {
	heap.Push(h, pastDial{id, exp})
}
func (h dialHistory) contains(id discover.NodeID) bool {
	for _, v := range h {
		if v.id == id {
			return true
		}
	}
	return false
}
func (h *dialHistory) expire(now time.Time) {
	for h.Len() > 0 && h.min().exp.Before(now) {
		heap.Pop(h)
	}
}

// heap.Interface boilerplate
func (h dialHistory) Len() int           { return len(h) }
func (h dialHistory) Less(i, j int) bool { return h[i].exp.Before(h[j].exp) }
func (h dialHistory) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *dialHistory) Push(x interface{}) {
	*h = append(*h, x.(pastDial))
}
func (h *dialHistory) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
