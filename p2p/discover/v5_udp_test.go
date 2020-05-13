// Copyright 2019 The go-ethereum Authors
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

package discover

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

// Real sockets, real crypto: this test checks end-to-end connectivity for UDPv5.
func TestEndToEndV5(t *testing.T) {
	t.Parallel()

	var nodes []*UDPv5
	for i := 0; i < 5; i++ {
		var cfg Config
		if len(nodes) > 0 {
			bn := nodes[0].Self()
			cfg.Bootnodes = []*enode.Node{bn}
		}
		node := startLocalhostV5(t, cfg)
		nodes = append(nodes, node)
		defer node.Close()
	}

	last := nodes[len(nodes)-1]
	target := nodes[rand.Intn(len(nodes)-2)].Self()
	results := last.Lookup(target.ID())
	if len(results) == 0 || results[0].ID() != target.ID() {
		t.Fatalf("lookup returned wrong results: %v", results)
	}
}

func startLocalhostV5(t *testing.T, cfg Config) *UDPv5 {
	cfg.PrivateKey = newkey()
	db, _ := enode.OpenDB("")
	ln := enode.NewLocalNode(db, cfg.PrivateKey)

	// Prefix logs with node ID.
	lprefix := fmt.Sprintf("(%s)", ln.ID().TerminalString())
	lfmt := log.TerminalFormat(false)
	cfg.Log = testlog.Logger(t, log.LvlTrace)
	cfg.Log.SetHandler(log.FuncHandler(func(r *log.Record) error {
		t.Logf("%s %s", lprefix, lfmt.Format(r))
		return nil
	}))

	// Listen.
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	realaddr := socket.LocalAddr().(*net.UDPAddr)
	ln.SetStaticIP(realaddr.IP)
	ln.Set(enr.UDP(realaddr.Port))
	udp, err := ListenV5(socket, ln, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return udp
}

// This test checks that incoming PING calls are handled correctly.
func TestUDPv5_pingHandling(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	test.packetIn(&pingV5{ReqID: []byte("foo")})
	test.waitPacketOut(func(p *pongV5, addr *net.UDPAddr, authTag []byte) {
		if !bytes.Equal(p.ReqID, []byte("foo")) {
			t.Error("wrong request ID in response:", p.ReqID)
		}
		if p.ENRSeq != test.table.self().Seq() {
			t.Error("wrong ENR sequence number in response:", p.ENRSeq)
		}
	})
}

// This test checks that incoming 'unknown' packets trigger the handshake.
func TestUDPv5_unknownPacket(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	authTag := [12]byte{1, 2, 3}
	check := func(p *whoareyouV5, wantSeq uint64) {
		t.Helper()
		if !bytes.Equal(p.AuthTag, authTag[:]) {
			t.Error("wrong token in WHOAREYOU:", p.AuthTag, authTag[:])
		}
		if p.IDNonce == ([32]byte{}) {
			t.Error("all zero ID nonce")
		}
		if p.RecordSeq != wantSeq {
			t.Errorf("wrong record seq %d in WHOAREYOU, want %d", p.RecordSeq, wantSeq)
		}
	}

	// Unknown packet from unknown node.
	test.packetIn(&unknownV5{AuthTag: authTag[:]})
	test.waitPacketOut(func(p *whoareyouV5, addr *net.UDPAddr, _ []byte) {
		check(p, 0)
	})

	// Make node known.
	n := test.getNode(test.remotekey, test.remoteaddr).Node()
	test.table.addSeenNode(wrapNode(n))

	test.packetIn(&unknownV5{AuthTag: authTag[:]})
	test.waitPacketOut(func(p *whoareyouV5, addr *net.UDPAddr, _ []byte) {
		check(p, n.Seq())
	})
}

// This test checks that incoming FINDNODE calls are handled correctly.
func TestUDPv5_findnodeHandling(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	// Create test nodes and insert them into the table.
	nodes := nodesAtDistance(test.table.self().ID(), 253, 10)
	fillTable(test.table, wrapNodes(nodes))

	// Requesting with distance zero should return the node's own record.
	test.packetIn(&findnodeV5{ReqID: []byte{0}, Distance: 0})
	test.expectNodes([]byte{0}, 1, []*enode.Node{test.udp.Self()})

	// Requesting with distance > 256 caps it at 256.
	test.packetIn(&findnodeV5{ReqID: []byte{1}, Distance: 4234098})
	test.expectNodes([]byte{1}, 1, nil)

	// This request gets no nodes because the corresponding bucket is empty.
	test.packetIn(&findnodeV5{ReqID: []byte{2}, Distance: 254})
	test.expectNodes([]byte{2}, 1, nil)

	// This request gets all test nodes.
	test.packetIn(&findnodeV5{ReqID: []byte{3}, Distance: 253})
	test.expectNodes([]byte{3}, 4, nodes)
}

func (test *udpV5Test) expectNodes(wantReqID []byte, wantTotal uint8, wantNodes []*enode.Node) {
	nodeSet := make(map[enode.ID]*enr.Record)
	for _, n := range wantNodes {
		nodeSet[n.ID()] = n.Record()
	}
	for {
		test.waitPacketOut(func(p *nodesV5, addr *net.UDPAddr, authTag []byte) {
			if len(p.Nodes) > 3 {
				test.t.Fatalf("too many nodes in response")
			}
			if p.Total != wantTotal {
				test.t.Fatalf("wrong total response count %d", p.Total)
			}
			if !bytes.Equal(p.ReqID, wantReqID) {
				test.t.Fatalf("wrong request ID in response: %v", p.ReqID)
			}
			for _, record := range p.Nodes {
				n, _ := enode.New(enode.ValidSchemesForTesting, record)
				want := nodeSet[n.ID()]
				if want == nil {
					test.t.Fatalf("unexpected node in response: %v", n)
				}
				if !reflect.DeepEqual(record, want) {
					test.t.Fatalf("wrong record in response: %v", n)
				}
				delete(nodeSet, n.ID())
			}
		})
		if len(nodeSet) == 0 {
			return
		}
	}
}

// This test checks that outgoing PING calls work.
func TestUDPv5_pingCall(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	remote := test.getNode(test.remotekey, test.remoteaddr).Node()
	done := make(chan error, 1)

	// This ping times out.
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {})
	if err := <-done; err != errTimeout {
		t.Fatalf("want errTimeout, got %q", err)
	}

	// This ping works.
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetInFrom(test.remotekey, test.remoteaddr, &pongV5{ReqID: p.ReqID})
	})
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// This ping gets a reply from the wrong endpoint.
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		wrongAddr := &net.UDPAddr{IP: net.IP{33, 44, 55, 22}, Port: 10101}
		test.packetInFrom(test.remotekey, wrongAddr, &pongV5{ReqID: p.ReqID})
	})
	if err := <-done; err != errTimeout {
		t.Fatalf("want errTimeout for reply from wrong IP, got %q", err)
	}
}

// This test checks that outgoing FINDNODE calls work and multiple NODES
// replies are aggregated.
func TestUDPv5_findnodeCall(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	// Launch the request:
	var (
		distance = 230
		remote   = test.getNode(test.remotekey, test.remoteaddr).Node()
		nodes    = nodesAtDistance(remote.ID(), distance, 8)
		done     = make(chan error, 1)
		response []*enode.Node
	)
	go func() {
		var err error
		response, err = test.udp.findnode(remote, distance)
		done <- err
	}()

	// Serve the responses:
	test.waitPacketOut(func(p *findnodeV5, addr *net.UDPAddr, authTag []byte) {
		if p.Distance != uint(distance) {
			t.Fatalf("wrong bucket: %d", p.Distance)
		}
		test.packetIn(&nodesV5{
			ReqID: p.ReqID,
			Total: 2,
			Nodes: nodesToRecords(nodes[:4]),
		})
		test.packetIn(&nodesV5{
			ReqID: p.ReqID,
			Total: 2,
			Nodes: nodesToRecords(nodes[4:]),
		})
	})

	// Check results:
	if err := <-done; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, nodes) {
		t.Fatalf("wrong nodes in response")
	}

	// TODO: check invalid IPs
	// TODO: check invalid/unsigned record
}

// This test checks that pending calls are re-sent when a handshake happens.
func TestUDPv5_callResend(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	remote := test.getNode(test.remotekey, test.remoteaddr).Node()
	done := make(chan error, 2)
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()

	// Ping answered by WHOAREYOU.
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetIn(&whoareyouV5{AuthTag: authTag})
	})
	// Ping should be re-sent.
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetIn(&pongV5{ReqID: p.ReqID})
	})
	// Answer the other ping.
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetIn(&pongV5{ReqID: p.ReqID})
	})
	if err := <-done; err != nil {
		t.Fatalf("unexpected ping error: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("unexpected ping error: %v", err)
	}
}

// This test ensures we don't allow multiple rounds of WHOAREYOU for a single call.
func TestUDPv5_multipleHandshakeRounds(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	remote := test.getNode(test.remotekey, test.remoteaddr).Node()
	done := make(chan error, 1)
	go func() {
		_, err := test.udp.ping(remote)
		done <- err
	}()

	// Ping answered by WHOAREYOU.
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetIn(&whoareyouV5{AuthTag: authTag})
	})
	// Ping answered by WHOAREYOU again.
	test.waitPacketOut(func(p *pingV5, addr *net.UDPAddr, authTag []byte) {
		test.packetIn(&whoareyouV5{AuthTag: authTag})
	})
	if err := <-done; err != errTimeout {
		t.Fatalf("unexpected ping error: %q", err)
	}
}

// This test checks that calls with n replies may take up to n * respTimeout.
func TestUDPv5_callTimeoutReset(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)
	defer test.close()

	// Launch the request:
	var (
		distance = 230
		remote   = test.getNode(test.remotekey, test.remoteaddr).Node()
		nodes    = nodesAtDistance(remote.ID(), distance, 8)
		done     = make(chan error, 1)
	)
	go func() {
		_, err := test.udp.findnode(remote, distance)
		done <- err
	}()

	// Serve two responses, slowly.
	test.waitPacketOut(func(p *findnodeV5, addr *net.UDPAddr, authTag []byte) {
		time.Sleep(respTimeout - 50*time.Millisecond)
		test.packetIn(&nodesV5{
			ReqID: p.ReqID,
			Total: 2,
			Nodes: nodesToRecords(nodes[:4]),
		})

		time.Sleep(respTimeout - 50*time.Millisecond)
		test.packetIn(&nodesV5{
			ReqID: p.ReqID,
			Total: 2,
			Nodes: nodesToRecords(nodes[4:]),
		})
	})
	if err := <-done; err != nil {
		t.Fatalf("unexpected error: %q", err)
	}
}

// This test checks that lookup works.
func TestUDPv5_lookup(t *testing.T) {
	t.Parallel()
	test := newUDPV5Test(t)

	// Lookup on empty table returns no nodes.
	if results := test.udp.Lookup(lookupTestnet.target.id()); len(results) > 0 {
		t.Fatalf("lookup on empty table returned %d results: %#v", len(results), results)
	}

	// Ensure the tester knows all nodes in lookupTestnet by IP.
	for d, nn := range lookupTestnet.dists {
		for i, key := range nn {
			n := lookupTestnet.node(d, i)
			test.getNode(key, &net.UDPAddr{IP: n.IP(), Port: n.UDP()})
		}
	}

	// Seed table with initial node.
	fillTable(test.table, []*node{wrapNode(lookupTestnet.node(256, 0))})

	// Start the lookup.
	resultC := make(chan []*enode.Node, 1)
	go func() {
		resultC <- test.udp.Lookup(lookupTestnet.target.id())
		test.close()
	}()

	// Answer lookup packets.
	for done := false; !done; {
		done = test.waitPacketOut(func(p packetV5, to *net.UDPAddr, authTag []byte) {
			recipient, key := lookupTestnet.nodeByAddr(to)
			switch p := p.(type) {
			case *pingV5:
				test.packetInFrom(key, to, &pongV5{ReqID: p.ReqID})
			case *findnodeV5:
				nodes := lookupTestnet.neighborsAtDistance(recipient, p.Distance, 3)
				response := &nodesV5{ReqID: p.ReqID, Total: 1, Nodes: nodesToRecords(nodes)}
				test.packetInFrom(key, to, response)
			}
		})
	}

	// Verify result nodes.
	checkLookupResults(t, lookupTestnet, <-resultC)
}

// This test checks the local node can be utilised to set key-values.
func TestUDPv5_LocalNode(t *testing.T) {
	t.Parallel()
	var cfg Config
	node := startLocalhostV5(t, cfg)
	defer node.Close()
	localNd := node.LocalNode()

	// set value in node's local record
	testVal := [4]byte{'A', 'B', 'C', 'D'}
	localNd.Set(enr.WithEntry("testing", &testVal))

	// retrieve the value from self to make sure it matches.
	outputVal := [4]byte{}
	if err := node.Self().Load(enr.WithEntry("testing", &outputVal)); err != nil {
		t.Errorf("Could not load value from record: %v", err)
	}
	if testVal != outputVal {
		t.Errorf("Wanted %#x to be retrieved from the record but instead got %#x", testVal, outputVal)
	}
}

// udpV5Test is the framework for all tests above.
// It runs the UDPv5 transport on a virtual socket and allows testing outgoing packets.
type udpV5Test struct {
	t                   *testing.T
	pipe                *dgramPipe
	table               *Table
	db                  *enode.DB
	udp                 *UDPv5
	localkey, remotekey *ecdsa.PrivateKey
	remoteaddr          *net.UDPAddr
	nodesByID           map[enode.ID]*enode.LocalNode
	nodesByIP           map[string]*enode.LocalNode
}

type testCodec struct {
	test *udpV5Test
	id   enode.ID
	ctr  uint64
}

type testCodecFrame struct {
	NodeID  enode.ID
	AuthTag []byte
	Ptype   byte
	Packet  rlp.RawValue
}

func (c *testCodec) encode(toID enode.ID, addr string, p packetV5, _ *whoareyouV5) ([]byte, []byte, error) {
	c.ctr++
	authTag := make([]byte, 8)
	binary.BigEndian.PutUint64(authTag, c.ctr)
	penc, _ := rlp.EncodeToBytes(p)
	frame, err := rlp.EncodeToBytes(testCodecFrame{c.id, authTag, p.kind(), penc})
	return frame, authTag, err
}

func (c *testCodec) decode(input []byte, addr string) (enode.ID, *enode.Node, packetV5, error) {
	frame, p, err := c.decodeFrame(input)
	if err != nil {
		return enode.ID{}, nil, nil, err
	}
	if p.kind() == p_whoareyouV5 {
		frame.NodeID = enode.ID{} // match wireCodec behavior
	}
	return frame.NodeID, nil, p, nil
}

func (c *testCodec) decodeFrame(input []byte) (frame testCodecFrame, p packetV5, err error) {
	if err = rlp.DecodeBytes(input, &frame); err != nil {
		return frame, nil, fmt.Errorf("invalid frame: %v", err)
	}
	switch frame.Ptype {
	case p_unknownV5:
		dec := new(unknownV5)
		err = rlp.DecodeBytes(frame.Packet, &dec)
		p = dec
	case p_whoareyouV5:
		dec := new(whoareyouV5)
		err = rlp.DecodeBytes(frame.Packet, &dec)
		p = dec
	default:
		p, err = decodePacketBodyV5(frame.Ptype, frame.Packet)
	}
	return frame, p, err
}

func newUDPV5Test(t *testing.T) *udpV5Test {
	test := &udpV5Test{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{10, 0, 1, 99}, Port: 30303},
		nodesByID:  make(map[enode.ID]*enode.LocalNode),
		nodesByIP:  make(map[string]*enode.LocalNode),
	}
	test.db, _ = enode.OpenDB("")
	ln := enode.NewLocalNode(test.db, test.localkey)
	ln.SetStaticIP(net.IP{10, 0, 0, 1})
	ln.Set(enr.UDP(30303))
	test.udp, _ = ListenV5(test.pipe, ln, Config{
		PrivateKey:   test.localkey,
		Log:          testlog.Logger(t, log.LvlTrace),
		ValidSchemes: enode.ValidSchemesForTesting,
	})
	test.udp.codec = &testCodec{test: test, id: ln.ID()}
	test.table = test.udp.tab
	test.nodesByID[ln.ID()] = ln
	// Wait for initial refresh so the table doesn't send unexpected findnode.
	<-test.table.initDone
	return test
}

// handles a packet as if it had been sent to the transport.
func (test *udpV5Test) packetIn(packet packetV5) {
	test.t.Helper()
	test.packetInFrom(test.remotekey, test.remoteaddr, packet)
}

// handles a packet as if it had been sent to the transport by the key/endpoint.
func (test *udpV5Test) packetInFrom(key *ecdsa.PrivateKey, addr *net.UDPAddr, packet packetV5) {
	test.t.Helper()

	ln := test.getNode(key, addr)
	codec := &testCodec{test: test, id: ln.ID()}
	enc, _, err := codec.encode(test.udp.Self().ID(), addr.String(), packet, nil)
	if err != nil {
		test.t.Errorf("%s encode error: %v", packet.name(), err)
	}
	if test.udp.dispatchReadPacket(addr, enc) {
		<-test.udp.readNextCh // unblock UDPv5.dispatch
	}
}

// getNode ensures the test knows about a node at the given endpoint.
func (test *udpV5Test) getNode(key *ecdsa.PrivateKey, addr *net.UDPAddr) *enode.LocalNode {
	id := encodePubkey(&key.PublicKey).id()
	ln := test.nodesByID[id]
	if ln == nil {
		db, _ := enode.OpenDB("")
		ln = enode.NewLocalNode(db, key)
		ln.SetStaticIP(addr.IP)
		ln.Set(enr.UDP(addr.Port))
		test.nodesByID[id] = ln
	}
	test.nodesByIP[string(addr.IP)] = ln
	return ln
}

func (test *udpV5Test) waitPacketOut(validate interface{}) (closed bool) {
	test.t.Helper()
	fn := reflect.ValueOf(validate)
	exptype := fn.Type().In(0)

	dgram, err := test.pipe.receive()
	if err == errClosed {
		return true
	}
	if err == errTimeout {
		test.t.Fatalf("timed out waiting for %v", exptype)
		return false
	}
	ln := test.nodesByIP[string(dgram.to.IP)]
	if ln == nil {
		test.t.Fatalf("attempt to send to non-existing node %v", &dgram.to)
		return false
	}
	codec := &testCodec{test: test, id: ln.ID()}
	frame, p, err := codec.decodeFrame(dgram.data)
	if err != nil {
		test.t.Errorf("sent packet decode error: %v", err)
		return false
	}
	if !reflect.TypeOf(p).AssignableTo(exptype) {
		test.t.Errorf("sent packet type mismatch, got: %v, want: %v", reflect.TypeOf(p), exptype)
		return false
	}
	fn.Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(&dgram.to), reflect.ValueOf(frame.AuthTag)})
	return false
}

func (test *udpV5Test) close() {
	test.t.Helper()

	test.udp.Close()
	test.db.Close()
	for id, n := range test.nodesByID {
		if id != test.udp.Self().ID() {
			n.Database().Close()
		}
	}
	if len(test.pipe.queue) != 0 {
		test.t.Fatalf("%d unmatched UDP packets in queue", len(test.pipe.queue))
	}
}
