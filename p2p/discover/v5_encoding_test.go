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
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// To regenerate discv5 test vectors, run
//
//     go test -run TestVectors -write-test-vectors
//
var writeTestVectorsFlag = flag.Bool("write-test-vectors", false, "Overwrite discv5 test vectors in testdata/")

var (
	testKeyA, _ = crypto.HexToECDSA("eef77acb6c6a6eebc5b363a475ac583ec7eccdb42b6481424c60f59aa326547f")
	testKeyB, _ = crypto.HexToECDSA("66fb62bfbd66b9177a138c1e5cddbe4f7c30c343e94e68df8769459cb1cde628")
	testIDnonce = [32]byte{5, 6, 7, 8, 9, 10, 11, 12}
)

func TestDeriveKeysV5(t *testing.T) {
	t.Parallel()

	var (
		n1        = enode.ID{1}
		n2        = enode.ID{2}
		challenge = &whoareyouV5{}
		db, _     = enode.OpenDB("")
		ln        = enode.NewLocalNode(db, testKeyA)
		c         = newWireCodec(ln, testKeyA, mclock.System{})
	)
	defer db.Close()

	sec1 := c.deriveKeys(n1, n2, testKeyA, &testKeyB.PublicKey, challenge)
	sec2 := c.deriveKeys(n1, n2, testKeyB, &testKeyA.PublicKey, challenge)
	if sec1 == nil || sec2 == nil {
		t.Fatal("key agreement failed")
	}
	if !reflect.DeepEqual(sec1, sec2) {
		t.Fatalf("keys not equal:\n  %+v\n  %+v", sec1, sec2)
	}
}

// This test checks the basic handshake flow where A talks to B and A has no secrets.
func TestHandshakeV5(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	// A -> B   RANDOM PACKET
	packet, _ := net.nodeA.encode(t, net.nodeB, &findnodeV5{})
	resp := net.nodeB.expectDecode(t, p_unknownV5, packet)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{
		AuthTag:   resp.(*unknownV5).AuthTag,
		IDNonce:   testIDnonce,
		RecordSeq: 0,
	}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// A -> B   FINDNODE (handshake packet)
	findnode, _ := net.nodeA.encodeWithChallenge(t, net.nodeB, challenge, &findnodeV5{})
	net.nodeB.expectDecode(t, p_findnodeV5, findnode)
	if len(net.nodeB.c.sc.handshakes) > 0 {
		t.Fatalf("node B didn't remove handshake from challenge map")
	}

	// A <- B   NODES
	nodes, _ := net.nodeB.encode(t, net.nodeA, &nodesV5{Total: 1})
	net.nodeA.expectDecode(t, p_nodesV5, nodes)
}

// This test checks that handshake attempts are removed within the timeout.
func TestHandshakeV5_timeout(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	// A -> B   RANDOM PACKET
	packet, _ := net.nodeA.encode(t, net.nodeB, &findnodeV5{})
	resp := net.nodeB.expectDecode(t, p_unknownV5, packet)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{
		AuthTag:   resp.(*unknownV5).AuthTag,
		IDNonce:   testIDnonce,
		RecordSeq: 0,
	}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// A -> B   FINDNODE (handshake packet) after timeout
	net.clock.Run(handshakeTimeout + 1)
	findnode, _ := net.nodeA.encodeWithChallenge(t, net.nodeB, challenge, &findnodeV5{})
	net.nodeB.expectDecodeErr(t, errUnexpectedHandshake, findnode)
}

// This test checks handshake behavior when no record is sent in the auth response.
func TestHandshakeV5_norecord(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	// A -> B   RANDOM PACKET
	packet, _ := net.nodeA.encode(t, net.nodeB, &findnodeV5{})
	resp := net.nodeB.expectDecode(t, p_unknownV5, packet)

	// A <- B   WHOAREYOU
	nodeA := net.nodeA.n()
	if nodeA.Seq() == 0 {
		t.Fatal("need non-zero sequence number")
	}
	challenge := &whoareyouV5{
		AuthTag:   resp.(*unknownV5).AuthTag,
		IDNonce:   testIDnonce,
		RecordSeq: nodeA.Seq(),
		node:      nodeA,
	}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// A -> B   FINDNODE
	findnode, _ := net.nodeA.encodeWithChallenge(t, net.nodeB, challenge, &findnodeV5{})
	net.nodeB.expectDecode(t, p_findnodeV5, findnode)

	// A <- B   NODES
	nodes, _ := net.nodeB.encode(t, net.nodeA, &nodesV5{Total: 1})
	net.nodeA.expectDecode(t, p_nodesV5, nodes)
}

// In this test, A tries to send FINDNODE with existing secrets but B doesn't know
// anything about A.
func TestHandshakeV5_rekey(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	session := &session{
		readKey:  []byte("BBBBBBBBBBBBBBBB"),
		writeKey: []byte("AAAAAAAAAAAAAAAA"),
	}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), session)

	// A -> B   FINDNODE (encrypted with zero keys)
	findnode, authTag := net.nodeA.encode(t, net.nodeB, &findnodeV5{})
	net.nodeB.expectDecode(t, p_unknownV5, findnode)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{AuthTag: authTag, IDNonce: testIDnonce}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// Check that new keys haven't been stored yet.
	sa := net.nodeA.c.sc.session(net.nodeB.id(), net.nodeB.addr())
	if !bytes.Equal(sa.writeKey, session.writeKey) || !bytes.Equal(sa.readKey, session.readKey) {
		t.Fatal("node A stored keys too early")
	}
	if s := net.nodeB.c.sc.session(net.nodeA.id(), net.nodeA.addr()); s != nil {
		t.Fatal("node B stored keys too early")
	}

	// A -> B   FINDNODE encrypted with new keys
	findnode, _ = net.nodeA.encodeWithChallenge(t, net.nodeB, challenge, &findnodeV5{})
	net.nodeB.expectDecode(t, p_findnodeV5, findnode)

	// A <- B   NODES
	nodes, _ := net.nodeB.encode(t, net.nodeA, &nodesV5{Total: 1})
	net.nodeA.expectDecode(t, p_nodesV5, nodes)
}

// In this test A and B have different keys before the handshake.
func TestHandshakeV5_rekey2(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	initKeysA := &session{
		readKey:  []byte("BBBBBBBBBBBBBBBB"),
		writeKey: []byte("AAAAAAAAAAAAAAAA"),
	}
	initKeysB := &session{
		readKey:  []byte("CCCCCCCCCCCCCCCC"),
		writeKey: []byte("DDDDDDDDDDDDDDDD"),
	}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), initKeysA)
	net.nodeB.c.sc.storeNewSession(net.nodeA.id(), net.nodeA.addr(), initKeysB)

	// A -> B   FINDNODE encrypted with initKeysA
	findnode, authTag := net.nodeA.encode(t, net.nodeB, &findnodeV5{Distances: []uint{3}})
	net.nodeB.expectDecode(t, p_unknownV5, findnode)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{AuthTag: authTag, IDNonce: testIDnonce}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// A -> B   FINDNODE (handshake packet)
	findnode, _ = net.nodeA.encodeWithChallenge(t, net.nodeB, challenge, &findnodeV5{})
	net.nodeB.expectDecode(t, p_findnodeV5, findnode)

	// A <- B   NODES
	nodes, _ := net.nodeB.encode(t, net.nodeA, &nodesV5{Total: 1})
	net.nodeA.expectDecode(t, p_nodesV5, nodes)
}

// This test checks some malformed packets.
func TestDecodeErrorsV5(t *testing.T) {
	t.Parallel()
	net := newHandshakeTest()
	defer net.close()

	net.nodeA.expectDecodeErr(t, errTooShort, []byte{})
	// TODO some more tests would be nice :)
	// - check invalid authdata sizes
	// - check invalid handshake data sizes
}

// This test checks that all test vectors can be decoded.
func TestTestVectorsV5(t *testing.T) {
	var (
		idA     = enode.PubkeyToIDV4(&testKeyA.PublicKey)
		idB     = enode.PubkeyToIDV4(&testKeyB.PublicKey)
		addr    = "127.0.0.1"
		session = &session{
			writeKey: hexutil.MustDecode("0x00000000000000000000000000000000"),
			readKey:  hexutil.MustDecode("0x01010101010101010101010101010101"),
		}
		challenge0 = &whoareyouV5{
			AuthTag:   packetNonce{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			IDNonce:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			RecordSeq: 0,
		}
		challenge1 = &whoareyouV5{
			AuthTag:   packetNonce{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			IDNonce:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			RecordSeq: 1,
		}
	)

	type testVectorTest struct {
		name      string               // test vector name
		packet    packetV5             // the packet to be encoded
		challenge *whoareyouV5         // handshake challenge passed to encoder
		prep      func(*handshakeTest) // called before encode/decode
	}
	tests := []testVectorTest{
		{
			name: "v5.1-whoareyou",
			packet: &whoareyouV5{
				AuthTag: packetNonce{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			},
		},
		{
			name: "v5.1-ping-message",
			packet: &pingV5{
				ReqID:  []byte{0, 0, 0, 1},
				ENRSeq: 2,
			},
			prep: func(net *handshakeTest) {
				net.nodeA.c.sc.storeNewSession(idB, addr, session)
				net.nodeB.c.sc.storeNewSession(idA, addr, session.keysFlipped())
			},
		},
		{
			name: "v5.1-ping-handshake",
			packet: &pingV5{
				ReqID:  []byte{0, 0, 0, 1},
				ENRSeq: 2,
			},
			challenge: challenge1,
			prep: func(net *handshakeTest) {
				c := *challenge1
				c.node = net.nodeA.n()
				net.nodeB.c.sc.storeSentHandshake(idA, addr, &c)
			},
		},
		{
			name: "v5.1-ping-handshake-enr",
			packet: &pingV5{
				ReqID:  []byte{0, 0, 0, 1},
				ENRSeq: 2,
			},
			challenge: challenge0,
			prep: func(net *handshakeTest) {
				c := *challenge0
				c.node = net.nodeA.n()
				net.nodeB.c.sc.storeSentHandshake(idA, addr, &c)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			net := newHandshakeTest()
			net.nodeA.c.sc.nonceFunc = func(counter uint32) packetNonce {
				return packetNonce{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
			}
			defer net.close()

			if test.prep != nil {
				test.prep(net)
			}

			file := filepath.Join("testdata", test.name+".txt")
			if *writeTestVectorsFlag {
				d, nonce := net.nodeA.encodeWithChallenge(t, net.nodeB, test.challenge, test.packet)
				comment := testVectorComment(net, test.packet, test.challenge, nonce)
				writeTestVector(file, comment, d)
			}
			enc := hexFile(file)
			net.nodeB.expectDecode(t, test.packet.kind(), enc)
		})
	}
}

// testVectorComment creates the commentary for discv5 test vector files.
func testVectorComment(net *handshakeTest, p packetV5, challenge *whoareyouV5, nonce packetNonce) string {
	o := new(strings.Builder)
	fmt.Fprintf(o, "src-node-id = %#x\n", net.nodeA.id().Bytes())
	fmt.Fprintf(o, "dest-node-id = %#x\n", net.nodeB.id().Bytes())

	printWhoareyou := func(p *whoareyouV5) {
		fmt.Fprintf(o, "whoareyou.auth-tag = %#x\n", p.AuthTag[:])
		fmt.Fprintf(o, "whoareyou.id-nonce = %#x\n", p.IDNonce[:])
		fmt.Fprintf(o, "whoareyou.enr-seq = %d\n", p.RecordSeq)
	}

	switch p := p.(type) {
	case *whoareyouV5:
		// WHOAREYOU packet.
		printWhoareyou(p)
	case *pingV5:
		if challenge != nil {
			// Handshake message packet.
			printWhoareyou(challenge)
		} else {
			// Ordinary message packet.
			fmt.Fprintf(o, "read-key = %#x\n", net.nodeB.c.sc.readKey(net.nodeA.id(), net.nodeA.addr()))
		}
		fmt.Fprintf(o, "auth-tag = %#x\n", nonce[:])
		fmt.Fprintf(o, "ping.req-id = %#x\n", p.ReqID)
		fmt.Fprintf(o, "ping.enr-seq = %d", p.ENRSeq)
	default:
		panic(fmt.Errorf("unhandled packet type %T", p))
	}

	return o.String()
}

// This benchmark checks performance of handshake packet decoding.
func BenchmarkV5_DecodeHandshakePingSecp256k1(b *testing.B) {
	net := newHandshakeTest()
	defer net.close()

	var (
		idA       = net.nodeA.id()
		challenge = &whoareyouV5{node: net.nodeB.n()}
		message   = &pingV5{ReqID: []byte("reqid")}
	)
	enc, _, err := net.nodeA.c.encode(net.nodeB.id(), "", message, challenge)
	if err != nil {
		b.Fatal("can't encode handshake packet")
	}
	challenge.node = nil // force ENR signature verification in decoder
	b.ResetTimer()

	input := make([]byte, len(enc))
	for i := 0; i < b.N; i++ {
		copy(input, enc)
		net.nodeB.c.sc.storeSentHandshake(idA, "", challenge)
		_, _, _, err := net.nodeB.c.decode(input, "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// This benchmark checks how long it takes to decode an encrypted ping packet.
func BenchmarkV5_DecodePing(b *testing.B) {
	net := newHandshakeTest()
	defer net.close()

	session := &session{
		readKey:  []byte{233, 203, 93, 195, 86, 47, 177, 186, 227, 43, 2, 141, 244, 230, 120, 17},
		writeKey: []byte{79, 145, 252, 171, 167, 216, 252, 161, 208, 190, 176, 106, 214, 39, 178, 134},
	}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), session)
	net.nodeB.c.sc.storeNewSession(net.nodeA.id(), net.nodeA.addr(), session.keysFlipped())
	addrB := net.nodeA.addr()
	ping := &pingV5{ReqID: []byte("reqid"), ENRSeq: 5}
	enc, _, err := net.nodeA.c.encode(net.nodeB.id(), addrB, ping, nil)
	if err != nil {
		b.Fatalf("can't encode: %v", err)
	}
	b.ResetTimer()

	input := make([]byte, len(enc))
	for i := 0; i < b.N; i++ {
		copy(input, enc)
		_, _, packet, _ := net.nodeB.c.decode(input, addrB)
		if _, ok := packet.(*pingV5); !ok {
			b.Fatalf("wrong packet type %T", packet)
		}
	}
}

var pp = spew.NewDefaultConfig()

type handshakeTest struct {
	nodeA, nodeB handshakeTestNode
	clock        mclock.Simulated
}

type handshakeTestNode struct {
	ln *enode.LocalNode
	c  *wireCodec
}

func newHandshakeTest() *handshakeTest {
	t := new(handshakeTest)
	t.nodeA.init(testKeyA, net.IP{127, 0, 0, 1}, &t.clock)
	t.nodeB.init(testKeyB, net.IP{127, 0, 0, 1}, &t.clock)
	return t
}

func (t *handshakeTest) close() {
	t.nodeA.ln.Database().Close()
	t.nodeB.ln.Database().Close()
}

func (n *handshakeTestNode) init(key *ecdsa.PrivateKey, ip net.IP, clock mclock.Clock) {
	db, _ := enode.OpenDB("")
	n.ln = enode.NewLocalNode(db, key)
	n.ln.SetStaticIP(ip)
	if n.ln.Node().Seq() != 1 {
		panic(fmt.Errorf("unexpected seq %d", n.ln.Node().Seq()))
	}
	n.c = newWireCodec(n.ln, key, clock)
}

func (n *handshakeTestNode) encode(t testing.TB, to handshakeTestNode, p packetV5) ([]byte, packetNonce) {
	t.Helper()
	return n.encodeWithChallenge(t, to, nil, p)
}

func (n *handshakeTestNode) encodeWithChallenge(t testing.TB, to handshakeTestNode, c *whoareyouV5, p packetV5) ([]byte, packetNonce) {
	t.Helper()
	// Copy challenge and add destination node. This avoids sharing 'c' among the two codecs.
	var challenge *whoareyouV5
	if c != nil {
		challengeCopy := *c
		challenge = &challengeCopy
		challenge.node = to.n()
	}
	// Encode to destination.
	enc, nonce, err := n.c.encode(to.id(), to.addr(), p, challenge)
	if err != nil {
		t.Fatal(fmt.Errorf("(%s) %v", n.ln.ID().TerminalString(), err))
	}
	t.Logf("(%s) -> (%s)   %s\n%s", n.ln.ID().TerminalString(), to.id().TerminalString(), p.name(), hex.Dump(enc))
	return enc, nonce
}

func (n *handshakeTestNode) expectDecode(t *testing.T, ptype byte, p []byte) packetV5 {
	t.Helper()
	dec, err := n.decode(p)
	if err != nil {
		t.Fatal(fmt.Errorf("(%s) %v", n.ln.ID().TerminalString(), err))
	}
	t.Logf("(%s) %#v", n.ln.ID().TerminalString(), pp.NewFormatter(dec))
	if dec.kind() != ptype {
		t.Fatalf("expected packet type %d, got %d", ptype, dec.kind())
	}
	return dec
}

func (n *handshakeTestNode) expectDecodeErr(t *testing.T, wantErr error, p []byte) {
	t.Helper()
	if _, err := n.decode(p); !reflect.DeepEqual(err, wantErr) {
		t.Fatal(fmt.Errorf("(%s) got err %q, want %q", n.ln.ID().TerminalString(), err, wantErr))
	}
}

func (n *handshakeTestNode) decode(input []byte) (packetV5, error) {
	_, _, p, err := n.c.decode(input, "127.0.0.1")
	return p, err
}

func (n *handshakeTestNode) n() *enode.Node {
	return n.ln.Node()
}

func (n *handshakeTestNode) addr() string {
	return n.ln.Node().IP().String()
}

func (n *handshakeTestNode) id() enode.ID {
	return n.ln.ID()
}
