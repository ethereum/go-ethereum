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
	"net"
	"slices"
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

func (s *Suite) listen1(log logger) (*conn, net.PacketConn) {
	c := newConn(s.Dest, log)
	l := c.listen(s.Listen1)
	return c, l
}

func (s *Suite) listen2(log logger) (*conn, net.PacketConn, net.PacketConn) {
	c := newConn(s.Dest, log)
	l1, l2 := c.listen(s.Listen1), c.listen(s.Listen2)
	return c, l1, l2
}

func (s *Suite) AllTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Ping", Fn: s.TestPing},
		{Name: "PingLargeRequestID", Fn: s.TestPingLargeRequestID},
		{Name: "PingMultiIP", Fn: s.TestPingMultiIP},
		{Name: "HandshakeResend", Fn: s.TestHandshakeResend},
		{Name: "TalkRequest", Fn: s.TestTalkRequest},
		{Name: "FindnodeZeroDistance", Fn: s.TestFindnodeZeroDistance},
		{Name: "FindnodeResults", Fn: s.TestFindnodeResults},
	}
}

func (s *Suite) TestPing(t *utesting.T) {
	t.Log(`This test is just a sanity check. It sends PING and expects a PONG response.`)

	conn, l1 := s.listen1(t)
	defer conn.close()

	ping := &v5wire.Ping{ReqID: conn.nextReqID()}
	switch resp := conn.reqresp(l1, ping).(type) {
	case *v5wire.Pong:
		checkPong(t, resp, ping, l1)
	default:
		t.Fatal("expected PONG, got", resp.Name())
	}
}

func checkPong(t *utesting.T, pong *v5wire.Pong, ping *v5wire.Ping, c net.PacketConn) {
	if !bytes.Equal(pong.ReqID, ping.ReqID) {
		t.Fatalf("wrong request ID %x in PONG, want %x", pong.ReqID, ping.ReqID)
	}
	if !pong.ToIP.Equal(laddr(c).IP) {
		t.Fatalf("wrong destination IP %v in PONG, want %v", pong.ToIP, laddr(c).IP)
	}
	if int(pong.ToPort) != laddr(c).Port {
		t.Fatalf("wrong destination port %v in PONG, want %v", pong.ToPort, laddr(c).Port)
	}
}

func (s *Suite) TestPingLargeRequestID(t *utesting.T) {
	t.Log(`This test sends PING with a 9-byte request ID, which isn't allowed by the spec.
The remote node should not respond.`)

	conn, l1 := s.listen1(t)
	defer conn.close()

	ping := &v5wire.Ping{ReqID: make([]byte, 9)}
	switch resp := conn.reqresp(l1, ping).(type) {
	case *v5wire.Pong:
		t.Errorf("PONG response with unknown request ID %x", resp.ReqID)
	case *readError:
		if resp.err == v5wire.ErrInvalidReqID {
			t.Error("response with oversized request ID")
		} else if !netutil.IsTimeout(resp.err) {
			t.Error(resp)
		}
	}
}

func (s *Suite) TestPingMultiIP(t *utesting.T) {
	t.Log(`This test establishes a session from one IP as usual. The session is then reused
on another IP, which shouldn't work. The remote node should respond with WHOAREYOU for
the attempt from a different IP.`)

	conn, l1, l2 := s.listen2(t)
	defer conn.close()

	// Create the session on l1.
	ping := &v5wire.Ping{ReqID: conn.nextReqID()}
	resp := conn.reqresp(l1, ping)
	if resp.Kind() != v5wire.PongMsg {
		t.Fatal("expected PONG, got", resp)
	}
	checkPong(t, resp.(*v5wire.Pong), ping, l1)

	// Send on l2. This reuses the session because there is only one codec.
	t.Log("sending ping from alternate IP", l2.LocalAddr())
	ping2 := &v5wire.Ping{ReqID: conn.nextReqID()}
	conn.write(l2, ping2, nil)
	switch resp := conn.read(l2).(type) {
	case *v5wire.Pong:
		t.Fatalf("remote responded to PING from %v for session on IP %v", laddr(l2).IP, laddr(l1).IP)
	case *v5wire.Whoareyou:
		t.Logf("got WHOAREYOU for new session as expected")
		resp.Node = s.Dest
		conn.write(l2, ping2, resp)
	default:
		t.Fatal("expected WHOAREYOU, got", resp)
	}

	// Catch the PONG on l2.
	switch resp := conn.read(l2).(type) {
	case *v5wire.Pong:
		checkPong(t, resp, ping2, l2)
	default:
		t.Fatal("expected PONG, got", resp)
	}

	// Try on l1 again.
	ping3 := &v5wire.Ping{ReqID: conn.nextReqID()}
	conn.write(l1, ping3, nil)
	switch resp := conn.read(l1).(type) {
	case *v5wire.Pong:
		t.Fatalf("remote responded to PING from %v for session on IP %v", laddr(l1).IP, laddr(l2).IP)
	case *v5wire.Whoareyou:
		t.Logf("got WHOAREYOU for new session as expected")
	default:
		t.Fatal("expected WHOAREYOU, got", resp)
	}
}

// TestHandshakeResend starts a handshake, but doesn't finish it and sends a second ordinary message
// packet instead of a handshake message packet. The remote node should repeat the previous WHOAREYOU
// challenge for the first PING.
func (s *Suite) TestHandshakeResend(t *utesting.T) {
	conn, l1 := s.listen1(t)
	defer conn.close()

	// First PING triggers challenge.
	ping := &v5wire.Ping{ReqID: conn.nextReqID()}
	conn.write(l1, ping, nil)
	var challenge1 *v5wire.Whoareyou
	switch resp := conn.read(l1).(type) {
	case *v5wire.Whoareyou:
		challenge1 = resp
		t.Logf("got WHOAREYOU for PING")
	default:
		t.Fatal("expected WHOAREYOU, got", resp)
	}

	// Send second PING.
	ping2 := &v5wire.Ping{ReqID: conn.nextReqID()}
	conn.write(l1, ping2, nil)
	switch resp := conn.read(l1).(type) {
	case *v5wire.Whoareyou:
		if resp.Nonce != challenge1.Nonce {
			t.Fatalf("wrong nonce %x in WHOAREYOU (want %x)", resp.Nonce[:], challenge1.Nonce[:])
		}
		if !bytes.Equal(resp.ChallengeData, challenge1.ChallengeData) {
			t.Fatalf("wrong ChallengeData in resent WHOAREYOU (want %x)", resp.ChallengeData, challenge1.ChallengeData)
		}
		resp.Node = conn.remote
	default:
		t.Fatal("expected WHOAREYOU, got", resp)
	}
}

func (s *Suite) TestTalkRequest(t *utesting.T) {
	t.Log(`This test sends some examples of TALKREQ with a protocol-id of "test-protocol"
and expects an empty TALKRESP response.`)

	conn, l1 := s.listen1(t)
	defer conn.close()

	// Non-empty request ID.
	id := conn.nextReqID()
	resp := conn.reqresp(l1, &v5wire.TalkRequest{ReqID: id, Protocol: "test-protocol"})
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
	t.Log("sending TALKREQ with empty request-id")
	resp = conn.reqresp(l1, &v5wire.TalkRequest{Protocol: "test-protocol"})
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

func (s *Suite) TestFindnodeZeroDistance(t *utesting.T) {
	t.Log(`This test checks that the remote node returns itself for FINDNODE with distance zero.`)

	conn, l1 := s.listen1(t)
	defer conn.close()

	nodes, err := conn.findnode(l1, []uint{0})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("remote returned more than one node for FINDNODE [0]")
	}
	if nodes[0].ID() != conn.remote.ID() {
		t.Errorf("ID of response node is %v, want %v", nodes[0].ID(), conn.remote.ID())
	}
}

func (s *Suite) TestFindnodeResults(t *utesting.T) {
	t.Log(`This test pings the node under test from multiple other endpoints and node identities
(the 'bystanders'). After waiting for them to be accepted into the remote table, the test checks
that they are returned by FINDNODE.`)

	// Create bystanders.
	nodes := make([]*bystander, 5)
	liveCh := make(chan enode.ID, len(nodes))
	for i := range nodes {
		nodes[i] = newBystander(t, s, liveCh)
		defer nodes[i].close()
	}

	// Prefill each bystander with the full bystander set so background FINDNODE
	// lookups see useful routing data instead of empty responses.
	known := make([]*enode.Node, 0, len(nodes))
	for _, bn := range nodes {
		known = append(known, bn.conn.localNode.Node())
	}
	for _, bn := range nodes {
		bn.known = append([]*enode.Node(nil), known...)
	}

	// Wait until enough bystanders have actually become live, i.e. the remote node
	// has revalidated them by sending PING and receiving our PONG.
	requiredLiveNodes := len(nodes)
	timeout := 60 * time.Second
	timeoutCh := time.After(timeout)
	liveSet := make(map[enode.ID]*enode.Node)
	for len(liveSet) < requiredLiveNodes {
		select {
		case id := <-liveCh:
			for _, bn := range nodes {
				if bn.id() == id {
					liveSet[id] = bn.conn.localNode.Node()
					break
				}
			}
			t.Logf("bystander node %v became live", id)
		case <-timeoutCh:
			t.Errorf("remote revalidated %d bystander nodes in %v, need %d to continue", len(liveSet), timeout, requiredLiveNodes)
			return
		}
	}
	t.Logf("continuing after all %d bystander nodes became live", len(liveSet))

	// Collect live nodes by distance.
	var dists []uint
	expect := make(map[enode.ID]*enode.Node)
	for id, n := range liveSet {
		expect[id] = n
		d := uint(enode.LogDist(n.ID(), s.Dest.ID()))
		if !slices.Contains(dists, d) {
			dists = append(dists, d)
		}
	}

	// Send FINDNODE for all distances.
	t.Log("requesting nodes")
	conn, l1 := s.listen1(t)
	defer conn.close()

	const maxAttempts = 5
	const retryInterval = 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		foundNodes, err := conn.findnode(l1, dists)
		if err != nil {
			t.Fatal(err)
		}
		missing := make(map[enode.ID]struct{})
		for id := range expect {
			missing[id] = struct{}{}
		}
		for _, n := range foundNodes {
			delete(missing, n.ID())
		}
		t.Logf("attempt %d: remote returned %d nodes for distance list %v, missing %d", attempt, len(foundNodes), dists, len(missing))
		if len(missing) == 0 {
			t.Logf("all %d expected live nodes were returned", len(expect))
			return
		}
		if attempt < maxAttempts {
			time.Sleep(retryInterval)
		}
	}
	t.Errorf("missing nodes in FINDNODE result after %d attempts", maxAttempts)
	t.Logf("this can happen if the node has a non-empty table from previous runs")
}

// A bystander is a node whose only purpose is filling a spot in the remote table.
type bystander struct {
	dest  *enode.Node
	conn  *conn
	l     net.PacketConn
	known []*enode.Node

	liveCh chan enode.ID
	sent   map[v5wire.Nonce]v5wire.Packet
	done   sync.WaitGroup
}

func newBystander(t *utesting.T, s *Suite, live chan enode.ID) *bystander {
	conn, l := s.listen1(t)
	conn.setEndpoint(l) // bystander nodes need IP/port to get pinged
	bn := &bystander{
		conn:   conn,
		l:      l,
		dest:   s.Dest,
		liveCh: live,
		sent:   make(map[v5wire.Nonce]v5wire.Packet),
	}
	// Establish an initial session and let the remote learn this node before
	// switching to the passive responder loop below.
	conn.reqresp(l, &v5wire.Ping{
		ReqID:  conn.nextReqID(),
		ENRSeq: conn.localNode.Seq(),
	})
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

	for {
		p, from := bn.conn.readFrom(bn.l)
		switch p := p.(type) {
		case *v5wire.Whoareyou:
			p.Node = bn.dest
			if resp, ok := bn.sent[p.Nonce]; ok {
				nonce := bn.conn.writeTo(bn.l, resp, p, from)
				delete(bn.sent, p.Nonce)
				bn.sent[nonce] = resp
			} else {
				bn.conn.writeTo(bn.l, &v5wire.Ping{
					ReqID:  bn.conn.nextReqID(),
					ENRSeq: bn.conn.localNode.Seq(),
				}, p, from)
			}
		case *v5wire.Ping:
			resp := &v5wire.Pong{
				ReqID:  append([]byte(nil), p.ReqID...),
				ENRSeq: bn.conn.localNode.Seq(),
				ToIP:   from.IP,
				ToPort: uint16(from.Port),
			}
			nonce := bn.conn.writeTo(bn.l, resp, nil, from)
			bn.sent[nonce] = resp
			bn.notifyLive()
		case *v5wire.Findnode:
			resp := &v5wire.Nodes{ReqID: append([]byte(nil), p.ReqID...), RespCount: 1}
			for _, n := range bn.known {
				if slices.Contains(p.Distances, uint(enode.LogDist(n.ID(), bn.id()))) {
					resp.Nodes = append(resp.Nodes, n.Record())
				}
			}
			nonce := bn.conn.writeTo(bn.l, resp, nil, from)
			bn.sent[nonce] = resp
		case *v5wire.TalkRequest:
			resp := &v5wire.TalkResponse{ReqID: append([]byte(nil), p.ReqID...)}
			nonce := bn.conn.writeTo(bn.l, resp, nil, from)
			bn.sent[nonce] = resp
		case *readError:
			if netutil.IsTemporaryError(p.err) || v5wire.IsInvalidHeader(p.err) {
				continue
			}
			bn.conn.logf("shutting down: %v", p.err)
			return
		}
	}
}

func (bn *bystander) notifyLive() {
	if bn.liveCh != nil {
		bn.liveCh <- bn.id()
		bn.liveCh = nil
	}
}
