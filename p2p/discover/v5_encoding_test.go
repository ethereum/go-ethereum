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
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

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

	// A -> B   FINDNODE
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

	// A -> B   FINDNODE after timeout
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

	initKeys := &handshakeSecrets{
		readKey:  []byte("BBBBBBBBBBBBBBBB"),
		writeKey: []byte("AAAAAAAAAAAAAAAA"),
	}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), initKeys.readKey, initKeys.writeKey)

	// A -> B   FINDNODE (encrypted with zero keys)
	findnode, authTag := net.nodeA.encode(t, net.nodeB, &findnodeV5{})
	net.nodeB.expectDecode(t, p_unknownV5, findnode)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{AuthTag: authTag, IDNonce: testIDnonce}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// Check that new keys haven't been stored yet.
	if s := net.nodeA.c.sc.session(net.nodeB.id(), net.nodeB.addr()); !bytes.Equal(s.writeKey, initKeys.writeKey) || !bytes.Equal(s.readKey, initKeys.readKey) {
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

	initKeysA := &handshakeSecrets{
		readKey:  []byte("BBBBBBBBBBBBBBBB"),
		writeKey: []byte("AAAAAAAAAAAAAAAA"),
	}
	initKeysB := &handshakeSecrets{
		readKey:  []byte("CCCCCCCCCCCCCCCC"),
		writeKey: []byte("DDDDDDDDDDDDDDDD"),
	}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), initKeysA.readKey, initKeysA.writeKey)
	net.nodeB.c.sc.storeNewSession(net.nodeA.id(), net.nodeA.addr(), initKeysB.readKey, initKeysA.writeKey)

	// A -> B   FINDNODE encrypted with initKeysA
	findnode, authTag := net.nodeA.encode(t, net.nodeB, &findnodeV5{Distance: 3})
	net.nodeB.expectDecode(t, p_unknownV5, findnode)

	// A <- B   WHOAREYOU
	challenge := &whoareyouV5{AuthTag: authTag, IDNonce: testIDnonce}
	whoareyou, _ := net.nodeB.encode(t, net.nodeA, challenge)
	net.nodeA.expectDecode(t, p_whoareyouV5, whoareyou)

	// A -> B   FINDNODE encrypted with new keys
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
}

// This benchmark checks performance of authHeader decoding, verification and key derivation.
func BenchmarkV5_DecodeAuthSecp256k1(b *testing.B) {
	net := newHandshakeTest()
	defer net.close()

	var (
		idA       = net.nodeA.id()
		addrA     = net.nodeA.addr()
		challenge = &whoareyouV5{AuthTag: []byte("authresp"), RecordSeq: 0, node: net.nodeB.n()}
		nonce     = make([]byte, gcmNonceSize)
	)
	header, _, _ := net.nodeA.c.makeAuthHeader(nonce, challenge)
	challenge.node = nil // force ENR signature verification in decoder
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := net.nodeB.c.decodeAuthResp(idA, addrA, header, challenge)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// This benchmark checks how long it takes to decode an encrypted ping packet.
func BenchmarkV5_DecodePing(b *testing.B) {
	net := newHandshakeTest()
	defer net.close()

	r := []byte{233, 203, 93, 195, 86, 47, 177, 186, 227, 43, 2, 141, 244, 230, 120, 17}
	w := []byte{79, 145, 252, 171, 167, 216, 252, 161, 208, 190, 176, 106, 214, 39, 178, 134}
	net.nodeA.c.sc.storeNewSession(net.nodeB.id(), net.nodeB.addr(), r, w)
	net.nodeB.c.sc.storeNewSession(net.nodeA.id(), net.nodeA.addr(), w, r)
	addrB := net.nodeA.addr()
	ping := &pingV5{ReqID: []byte("reqid"), ENRSeq: 5}
	enc, _, err := net.nodeA.c.encode(net.nodeB.id(), addrB, ping, nil)
	if err != nil {
		b.Fatalf("can't encode: %v", err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, p, _ := net.nodeB.c.decode(enc, addrB)
		if _, ok := p.(*pingV5); !ok {
			b.Fatalf("wrong packet type %T", p)
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
	n.c = newWireCodec(n.ln, key, clock)
}

func (n *handshakeTestNode) encode(t testing.TB, to handshakeTestNode, p packetV5) ([]byte, []byte) {
	t.Helper()
	return n.encodeWithChallenge(t, to, nil, p)
}

func (n *handshakeTestNode) encodeWithChallenge(t testing.TB, to handshakeTestNode, c *whoareyouV5, p packetV5) ([]byte, []byte) {
	t.Helper()
	// Copy challenge and add destination node. This avoids sharing 'c' among the two codecs.
	var challenge *whoareyouV5
	if c != nil {
		challengeCopy := *c
		challenge = &challengeCopy
		challenge.node = to.n()
	}
	// Encode to destination.
	enc, authTag, err := n.c.encode(to.id(), to.addr(), p, challenge)
	if err != nil {
		t.Fatal(fmt.Errorf("(%s) %v", n.ln.ID().TerminalString(), err))
	}
	t.Logf("(%s) -> (%s)   %s\n%s", n.ln.ID().TerminalString(), to.id().TerminalString(), p.name(), hex.Dump(enc))
	return enc, authTag
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
