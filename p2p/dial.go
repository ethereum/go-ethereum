// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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
	"container/heap"
	"crypto/rand"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	// This is the amount of time spent waiting in between
	// redialing a certain node.
	dialHistoryExpiration = 30 * time.Second

	// Discovery lookups are throttled and can only run
	// once every few seconds.
	lookupInterval = 4 * time.Second
)

// dialstate schedules dials and discovery lookups.
// it get's a chance to compute new tasks on every iteration
// of the main loop in Server.run.
type dialstate struct {
	maxDynDials int
	ntab        discoverTable

	lookupRunning bool
	bootstrapped  bool

	dialing     map[discover.NodeID]connFlag
	lookupBuf   []*discover.Node // current discovery lookup results
	randomNodes []*discover.Node // filled from Table
	static      map[discover.NodeID]*discover.Node
	hist        *dialHistory
}

type discoverTable interface {
	Self() *discover.Node
	Close()
	Bootstrap([]*discover.Node)
	Lookup(target discover.NodeID) []*discover.Node
	ReadRandomNodes([]*discover.Node) int
}

// the dial history remembers recent dials.
type dialHistory []pastDial

// pastDial is an entry in the dial history.
type pastDial struct {
	id  discover.NodeID
	exp time.Time
}

type task interface {
	Do(*Server)
}

// A dialTask is generated for each node that is dialed.
type dialTask struct {
	flags connFlag
	dest  *discover.Node
}

// discoverTask runs discovery table operations.
// Only one discoverTask is active at any time.
//
// If bootstrap is true, the task runs Table.Bootstrap,
// otherwise it performs a random lookup and leaves the
// results in the task.
type discoverTask struct {
	bootstrap bool
	results   []*discover.Node
}

// A waitExpireTask is generated if there are no other tasks
// to keep the loop in Server.run ticking.
type waitExpireTask struct {
	time.Duration
}

func newDialState(static []*discover.Node, ntab discoverTable, maxdyn int) *dialstate {
	s := &dialstate{
		maxDynDials: maxdyn,
		ntab:        ntab,
		static:      make(map[discover.NodeID]*discover.Node),
		dialing:     make(map[discover.NodeID]connFlag),
		randomNodes: make([]*discover.Node, maxdyn/2),
		hist:        new(dialHistory),
	}
	for _, n := range static {
		s.static[n.ID] = n
	}
	return s
}

func (s *dialstate) addStatic(n *discover.Node) {
	s.static[n.ID] = n
}

func (s *dialstate) newTasks(nRunning int, peers map[discover.NodeID]*Peer, now time.Time) []task {
	var newtasks []task
	addDial := func(flag connFlag, n *discover.Node) bool {
		_, dialing := s.dialing[n.ID]
		if dialing || peers[n.ID] != nil || s.hist.contains(n.ID) {
			return false
		}
		s.dialing[n.ID] = flag
		newtasks = append(newtasks, &dialTask{flags: flag, dest: n})
		return true
	}

	// Compute number of dynamic dials necessary at this point.
	needDynDials := s.maxDynDials
	for _, p := range peers {
		if p.rw.is(dynDialedConn) {
			needDynDials--
		}
	}
	for _, flag := range s.dialing {
		if flag&dynDialedConn != 0 {
			needDynDials--
		}
	}

	// Expire the dial history on every invocation.
	s.hist.expire(now)

	// Create dials for static nodes if they are not connected.
	for _, n := range s.static {
		addDial(staticDialedConn, n)
	}

	// Use random nodes from the table for half of the necessary
	// dynamic dials.
	randomCandidates := needDynDials / 2
	if randomCandidates > 0 && s.bootstrapped {
		n := s.ntab.ReadRandomNodes(s.randomNodes)
		for i := 0; i < randomCandidates && i < n; i++ {
			if addDial(dynDialedConn, s.randomNodes[i]) {
				needDynDials--
			}
		}
	}
	// Create dynamic dials from random lookup results, removing tried
	// items from the result buffer.
	i := 0
	for ; i < len(s.lookupBuf) && needDynDials > 0; i++ {
		if addDial(dynDialedConn, s.lookupBuf[i]) {
			needDynDials--
		}
	}
	s.lookupBuf = s.lookupBuf[:copy(s.lookupBuf, s.lookupBuf[i:])]
	// Launch a discovery lookup if more candidates are needed. The
	// first discoverTask bootstraps the table and won't return any
	// results.
	if len(s.lookupBuf) < needDynDials && !s.lookupRunning {
		s.lookupRunning = true
		newtasks = append(newtasks, &discoverTask{bootstrap: !s.bootstrapped})
	}

	// Launch a timer to wait for the next node to expire if all
	// candidates have been tried and no task is currently active.
	// This should prevent cases where the dialer logic is not ticked
	// because there are no pending events.
	if nRunning == 0 && len(newtasks) == 0 && s.hist.Len() > 0 {
		t := &waitExpireTask{s.hist.min().exp.Sub(now)}
		newtasks = append(newtasks, t)
	}
	return newtasks
}

func (s *dialstate) taskDone(t task, now time.Time) {
	switch t := t.(type) {
	case *dialTask:
		s.hist.add(t.dest.ID, now.Add(dialHistoryExpiration))
		delete(s.dialing, t.dest.ID)
	case *discoverTask:
		if t.bootstrap {
			s.bootstrapped = true
		}
		s.lookupRunning = false
		s.lookupBuf = append(s.lookupBuf, t.results...)
	}
}

func (t *dialTask) Do(srv *Server) {
	addr := &net.TCPAddr{IP: t.dest.IP, Port: int(t.dest.TCP)}
	glog.V(logger.Debug).Infof("dialing %v\n", t.dest)
	fd, err := srv.Dialer.Dial("tcp", addr.String())
	if err != nil {
		glog.V(logger.Detail).Infof("dial error: %v", err)
		return
	}
	mfd := newMeteredConn(fd, false)

	srv.setupConn(mfd, t.flags, t.dest)
}
func (t *dialTask) String() string {
	return fmt.Sprintf("%v %x %v:%d", t.flags, t.dest.ID[:8], t.dest.IP, t.dest.TCP)
}

func (t *discoverTask) Do(srv *Server) {
	if t.bootstrap {
		srv.ntab.Bootstrap(srv.BootstrapNodes)
		return
	}
	// newTasks generates a lookup task whenever dynamic dials are
	// necessary. Lookups need to take some time, otherwise the
	// event loop spins too fast.
	next := srv.lastLookup.Add(lookupInterval)
	if now := time.Now(); now.Before(next) {
		time.Sleep(next.Sub(now))
	}
	srv.lastLookup = time.Now()
	var target discover.NodeID
	rand.Read(target[:])
	t.results = srv.ntab.Lookup(target)
}

func (t *discoverTask) String() (s string) {
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

func (t waitExpireTask) Do(*Server) {
	time.Sleep(t.Duration)
}
func (t waitExpireTask) String() string {
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
