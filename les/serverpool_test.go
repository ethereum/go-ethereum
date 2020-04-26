// Copyright 2020 The go-ethereum Authors
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

package les

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	lpc "github.com/ethereum/go-ethereum/les/lespay/client"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

const (
	testNodes  = 1000
	testTarget = 5
)

func testNodeID(i int) enode.ID {
	return enode.ID{42, byte(i % 256), byte(i / 256)}
}

func testNodeIndex(id enode.ID) int {
	if id[0] != 42 {
		return -1
	}
	return int(id[1]) + int(id[2])*256
}

type serverPoolTest struct {
	db        ethdb.KeyValueStore
	clock     *mclock.Simulated
	ns        *utils.NodeStateMachine
	vt        *lpc.ValueTracker
	sp        *serverPool
	input     enode.Iterator
	testNodes []spTestNode
	trusted   []string

	cycle, conn, servedConn  int
	serviceCycles, dialCount int
	disconnect               map[int][]int
}

type spTestNode struct {
	connectCycles, waitCycles int
	nextConnCycle, totalConn  int
	connected, service        bool
	peer                      *serverPeer
}

func newServerPoolTest() *serverPoolTest {
	nodes := make([]*enode.Node, testNodes)
	for i := range nodes {
		nodes[i] = enode.SignNull(&enr.Record{}, testNodeID(i))
	}
	return &serverPoolTest{
		clock:     &mclock.Simulated{},
		db:        memorydb.New(),
		input:     enode.CycleNodes(nodes),
		testNodes: make([]spTestNode, testNodes),
	}
}

func (s *serverPoolTest) addTrusted(i int) {
	s.trusted = append(s.trusted, enode.SignNull(&enr.Record{}, testNodeID(i)).String())
}

func (s *serverPoolTest) start() {
	s.ns = utils.NewNodeStateMachine(s.db, []byte("nodestate:"), s.clock, serverPoolSetup)
	s.vt = lpc.NewValueTracker(s.db, s.clock, requestList, time.Minute, 1/float64(time.Hour), 1/float64(time.Hour*100), 1/float64(time.Hour*1000))
	s.sp = newServerPool(s.db, []byte("serverpool:"), s.ns, s.vt, s.input, s.clock, s.trusted, true)
	s.disconnect = make(map[int][]int)
	s.ns.Start()
	s.sp.start()
}

func (s *serverPoolTest) stop() {
	s.ns.Stop()
	s.sp.stop()
	s.vt.Stop()
	for i := range s.testNodes {
		n := &s.testNodes[i]
		if n.connected {
			n.totalConn += s.cycle
		}
		n.connected = false
		n.peer = nil
		n.nextConnCycle = 0
	}
	s.conn, s.servedConn = 0, 0
}

func (s *serverPoolTest) run(count int) {
	for ; count > 0; count-- {
		if dcList := s.disconnect[s.cycle]; dcList != nil {
			for _, idx := range dcList {
				n := &s.testNodes[idx]
				s.sp.unregisterPeer(n.peer)
				n.totalConn += s.cycle
				n.connected = false
				n.peer = nil
				s.conn--
				if n.service {
					s.servedConn--
				}
				n.nextConnCycle = s.cycle + n.waitCycles
			}
			delete(s.disconnect, s.cycle)
		}
		if s.conn < testTarget {
			s.dialCount++
			s.sp.dialIterator.Next()
			dial := s.sp.dialIterator.Node()
			id := dial.ID()
			idx := testNodeIndex(id)
			n := &s.testNodes[idx]
			if !n.connected && n.connectCycles != 0 && s.cycle >= n.nextConnCycle {
				s.conn++
				if n.service {
					s.servedConn++
				}
				n.totalConn -= s.cycle
				n.connected = true
				dc := s.cycle + n.connectCycles
				s.disconnect[dc] = append(s.disconnect[dc], idx)
				n.peer = &serverPeer{peerCommons: peerCommons{Peer: p2p.NewPeer(id, "", nil)}}
				s.sp.registerPeer(n.peer)
				if n.service {
					s.vt.Served(s.vt.GetNode(id), []lpc.ServedRequest{{ReqType: 0, Amount: 100}}, 0)
				}
			}
		}
		s.serviceCycles += s.servedConn
		s.clock.Run(time.Second)
		s.cycle++
	}
}

func (s *serverPoolTest) setNodes(count, conn, wait int, service, trusted bool) (res []int) {
	for ; count > 0; count-- {
		idx := rand.Intn(testNodes)
		for s.testNodes[idx].connectCycles != 0 || s.testNodes[idx].connected {
			idx = rand.Intn(testNodes)
		}
		res = append(res, idx)
		s.testNodes[idx] = spTestNode{
			connectCycles: conn,
			waitCycles:    wait,
			service:       service,
		}
		if trusted {
			s.addTrusted(idx)
		}
	}
	return
}

func (s *serverPoolTest) resetNodes() {
	for i, n := range s.testNodes {
		if n.connected {
			n.totalConn += s.cycle
			s.sp.unregisterPeer(n.peer)
		}
		s.testNodes[i] = spTestNode{totalConn: n.totalConn}
	}
	s.conn, s.servedConn = 0, 0
	s.disconnect = make(map[int][]int)
	s.trusted = nil
}

func (s *serverPoolTest) checkNodes(t *testing.T, nodes []int, minTotal, maxTotal int) {
	var sum int
	for _, idx := range nodes {
		n := &s.testNodes[idx]
		if n.connected {
			n.totalConn += s.cycle
		}
		sum += n.totalConn
		n.totalConn = 0
		if n.connected {
			n.totalConn -= s.cycle
		}
	}
	if sum < minTotal || sum > maxTotal {
		t.Errorf("Total connection amount %d outside expected range %d to %d", sum, minTotal, maxTotal)
	}
}

func TestServerPool(t *testing.T) {
	s := newServerPoolTest()
	nodes := s.setNodes(20, 200, 200, true, false)
	s.setNodes(20, 20, 20, false, false)
	s.start()
	s.run(10000)
	s.stop()
	s.checkNodes(t, nodes, 40000, 50000)
}

func TestServerPoolChangedNodes(t *testing.T) {
	s := newServerPoolTest()
	nodes := s.setNodes(20, 200, 200, true, false)
	s.setNodes(20, 20, 20, false, false)
	s.start()
	s.run(10000)
	s.checkNodes(t, nodes, 40000, 50000)
	for i := 0; i < 3; i++ {
		s.resetNodes()
		nodes := s.setNodes(20, 200, 200, true, false)
		s.setNodes(20, 20, 20, false, false)
		s.run(10000)
		s.checkNodes(t, nodes, 40000, 50000)
	}
	s.stop()
}

func TestServerPoolRestartNoDiscovery(t *testing.T) {
	s := newServerPoolTest()
	nodes := s.setNodes(20, 200, 200, true, false)
	s.setNodes(20, 20, 20, false, false)
	s.start()
	s.run(10000)
	s.stop()
	s.checkNodes(t, nodes, 40000, 50000)
	s.input = nil
	s.start()
	s.run(10000)
	s.stop()
	s.checkNodes(t, nodes, 40000, 50000)
}

func TestServerPoolTrustedNoDiscovery(t *testing.T) {
	s := newServerPoolTest()
	trusted := s.setNodes(200, 200, 200, true, true)
	s.input = nil
	s.start()
	s.run(10000)
	s.stop()
	s.checkNodes(t, trusted, 40000, 50000)
}
