// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package v5test

import (
	"bytes"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/discover/v5wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/netutil"
)

// Suite is the discv5 test suite.
type Suite struct {
	Dest             *enode.Node
	Listen1, Listen2 string // listening addresses
}

func (s *Suite) listen(log logger) *conn {
	return newConn(s.Dest, s.Listen1, s.Listen2, log)
}

func (s *Suite) AllTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Ping", Fn: s.TestPing},
		{Name: "PingLargeRequestID", Fn: s.TestPingLargeRequestID},
		{Name: "TalkRequest", Fn: s.TestTalkRequest},
		{Name: "FindnodeZeroDistance", Fn: s.TestFindnodeZeroDistance},
		{Name: "FindnodeResults", Fn: s.TestFindnodeResults},
	}
}

// This test sends PING and expects a PONG response.
func (s *Suite) TestPing(t *utesting.T) {
	conn := s.listen(t)
	defer conn.close()

	ping := &v5wire.Ping{ReqID: conn.nextReqID()}
	switch resp := conn.reqresp(conn.l1, ping).(type) {
	case *v5wire.Pong:
		if !bytes.Equal(resp.ReqID, ping.ReqID) {
			t.Fatalf("wrong request ID %x in PONG, want %x", resp.ReqID, ping.ReqID)
		}
		if !resp.ToIP.Equal(laddr(conn.l1).IP) {
			t.Fatalf("wrong destination IP %v in PONG, want %v", resp.ToIP, laddr(conn.l1).IP)
		}
		if int(resp.ToPort) != laddr(conn.l1).Port {
			t.Fatalf("wrong destination port %v in PONG, want %v", resp.ToPort, laddr(conn.l1).Port)
		}
	default:
		t.Fatal("expected PONG, got", resp.Name())
	}
}

// This test sends PING with a 9-byte request ID, which isn't allowed by the spec.
// The remote node should not respond.
func (s *Suite) TestPingLargeRequestID(t *utesting.T) {
	conn := s.listen(t)
	defer conn.close()

	ping := &v5wire.Ping{ReqID: make([]byte, 9)}
	switch resp := conn.reqresp(conn.l1, ping).(type) {
	case *v5wire.Pong:
		t.Errorf("remote responded to PING with 9-byte request ID %x", resp.ReqID)
	case *readError:
		if !netutil.IsTimeout(resp.err) {
			t.Error(resp)
		}
	}
}

// This test sends TALKREQ and expects an empty TALKRESP response.
func (s *Suite) TestTalkRequest(t *utesting.T) {
	conn := s.listen(t)
	defer conn.close()

	// Non-empty request ID.
	id := conn.nextReqID()
	resp := conn.reqresp(conn.l1, &v5wire.TalkRequest{ReqID: id, Protocol: "test-protocol"})
	switch resp := resp.(type) {
	case *v5wire.TalkResponse:
		if !bytes.Equal(resp.ReqID, id) {
			t.Fatalf("wrong request ID %x in TALKRESP, want %x", resp.ReqID, id)
		}
		if len(resp.Message) > 0 {
			t.Fatalf("non-empty message %x in TALKRESP", resp.Message)
		}
	default:
		t.Fatal("expected TALKRESP, got", resp.Name())
	}

	// Empty request ID.
	resp = conn.reqresp(conn.l1, &v5wire.TalkRequest{Protocol: "test-protocol"})
	switch resp := resp.(type) {
	case *v5wire.TalkResponse:
		if len(resp.ReqID) > 0 {
			t.Fatalf("wrong request ID %x in TALKRESP, want empty byte array", resp.ReqID)
		}
		if len(resp.Message) > 0 {
			t.Fatalf("non-empty message %x in TALKRESP", resp.Message)
		}
	default:
		t.Fatal("expected TALKRESP, got", resp.Name())
	}
}

// This test checks that the remote node returns itself for FINDNODE with distance zero.
func (s *Suite) TestFindnodeZeroDistance(t *utesting.T) {
	conn := s.listen(t)
	defer conn.close()

	id := conn.nextReqID()
	resp := conn.reqresp(conn.l1, &v5wire.Findnode{ReqID: id, Distances: []uint{0}})
	switch resp := resp.(type) {
	case *v5wire.Nodes:
		if !bytes.Equal(resp.ReqID, id) {
			t.Fatalf("wrong request ID %x in NODES, want %x", resp.ReqID, id)
		}
		if len(resp.Nodes) != 1 {
			t.Error("invalid number of entries in NODES response")
		}
		nodes, err := checkRecords(resp.Nodes)
		if err != nil {
			t.Errorf("invalid node in NODES response: %v", err)
		}
		if nodes[0].ID() != conn.remote.ID() {
			t.Errorf("ID of response node is %v, want %v", nodes[0].ID(), conn.remote.ID())
		}
	default:
		t.Fatal("expected NODES, got", resp.Name())
	}
}

// In this test, multiple nodes ping the node under test. After waiting for them to be
// accepted into the remote table, the test checks that they are returned by FINDNODE.
func (s *Suite) TestFindnodeResults(t *utesting.T) {
	conn := s.listen(t)
	defer conn.close()

	// Create bystanders.
	nodes := make([]*bystander, 5)
	added := make(chan enode.ID, len(nodes))
	for i := range nodes {
		nodes[i] = newBystander(t, s, added)
		defer nodes[i].close()
	}

	// Get them added to the remote table.
	timeout := 60 * time.Second
	timeoutCh := time.After(timeout)
	for count := 0; count < len(nodes); {
		select {
		case id := <-added:
			t.Logf("bystander node %v added to remote table", id)
			count++
		case <-timeoutCh:
			t.Errorf("remote added %d bystander nodes in %v, need %d to continue", count, timeout, len(nodes))
			t.Logf("this can happen if the node has a non-empty table from previous runs")
			return
		}
	}
	t.Logf("all %d bystander nodes were added", len(nodes))

	// Collect our nodes by distance.
	var dists []uint
	expect := make(map[enode.ID]*enode.Node)
	for _, bn := range nodes {
		n := bn.conn.localNode.Node()
		expect[n.ID()] = n
		d := uint(enode.LogDist(n.ID(), s.Dest.ID()))
		if !containsUint(dists, d) {
			dists = append(dists, uint(d))
		}
	}

	// Send FINDNODE for all distances.
	foundNodes, err := conn.findnode(conn.l1, dists)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("remote returned %d nodes for distance list %v", len(foundNodes), dists)
	for _, n := range foundNodes {
		delete(expect, n.ID())
	}
	if len(expect) > 0 {
		t.Errorf("missing %d nodes in FINDNODE result", len(expect))
		t.Logf("this can happen if the test is run multiple times in quick succession")
		t.Logf("and the remote node hasn't removed dead nodes from previous runs yet")
	} else {
		t.Logf("all %d expected nodes were returned", len(nodes))
	}
}

// A bystander is a node whose only purpose is filling a spot in the remote table.
type bystander struct {
	dest *enode.Node
	conn *conn
	log  *utesting.T

	addedCh chan enode.ID
	done    sync.WaitGroup
}

func newBystander(t *utesting.T, s *Suite, added chan enode.ID) *bystander {
	bn := &bystander{
		conn:    s.listen(t),
		dest:    s.Dest,
		addedCh: added,
	}
	bn.done.Add(1)
	go bn.loop()
	return bn
}

// id returns the node ID of the bystander.
func (bn *bystander) id() enode.ID {
	return bn.conn.localNode.ID()
}

// close shuts down loop.
func (bn *bystander) close() {
	bn.conn.close()
	bn.done.Wait()
}

// loop answers packets from the remote node until quit.
func (bn *bystander) loop() {
	defer bn.done.Done()

	var (
		lastPing time.Time
		wasAdded bool
	)
	for {
		// Ping the remote node.
		if !wasAdded && time.Since(lastPing) > 10*time.Second {
			bn.conn.reqresp(bn.conn.l1, &v5wire.Ping{
				ReqID:  bn.conn.nextReqID(),
				ENRSeq: bn.dest.Seq(),
			})
			lastPing = time.Now()
		}
		// Answer packets.
		switch p := bn.conn.read(bn.conn.l1).(type) {
		case *v5wire.Ping:
			bn.conn.write(bn.conn.l1, &v5wire.Pong{
				ReqID:  p.ReqID,
				ENRSeq: bn.conn.localNode.Seq(),
				ToIP:   bn.dest.IP(),
				ToPort: uint16(bn.dest.UDP()),
			}, nil)
			wasAdded = true
			bn.notifyAdded()
		case *v5wire.Findnode:
			bn.conn.write(bn.conn.l1, &v5wire.Nodes{ReqID: p.ReqID, Total: 1}, nil)
			wasAdded = true
			bn.notifyAdded()
		case *v5wire.TalkRequest:
			bn.conn.write(bn.conn.l1, &v5wire.TalkResponse{ReqID: p.ReqID}, nil)
		case *readError:
			if !netutil.IsTemporaryError(p.err) {
				bn.conn.logf("shutting down: %v", p.err)
				return
			}
		}
	}
}

func (bn *bystander) notifyAdded() {
	if bn.addedCh != nil {
		bn.addedCh <- bn.id()
		bn.addedCh = nil
	}
}
