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

package test

import (
	"crypto/rand"
	"flag"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
)

const (
	expiration  = 20 * time.Second
	wrongPacket = 66
	macSize     = 256 / 8
)

var (
	remote   = flag.String("remote", "", "Node to run the test against")
	waitTime = flag.Int("waitTime", 500, "ms to wait for response")
)

type pingWithJunk struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
	JunkData1  uint
	JunkData2  []byte
}

func (req *pingWithJunk) Name() string { return "PING/v4" }
func (req *pingWithJunk) Kind() byte   { return v4wire.PingPacket }

type pingWrongType struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
}

func (req *pingWrongType) Name() string { return "WRONG/v4" }
func (req *pingWrongType) Kind() byte   { return wrongPacket }

func TestMain(m *testing.M) {
	if os.Getenv("CI") != "" {
		os.Exit(0)
	}
	flag.Parse()
	if *remote == "" {
		fmt.Fprintf(os.Stderr, "Need -remote to run this test\n")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func futureExpiration() uint64 {
	return uint64(time.Now().Add(expiration).Unix())
}

// This test just sends a PING packet and expects a response.
func BasicPing(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	req := v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

// This test sends a PING packet with wrong 'to' field and expects a PONG response.
func PingWrongTo(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	wrongEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	req := v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         wrongEndpoint,
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

// This test sends a PING packet with wrong 'from' field and expects a PONG response.
func PingWrongFrom(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	wrongEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	req := v4wire.Ping{
		Version:    4,
		From:       wrongEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

// This test sends a PING packet with additional data at the end and expects a PONG
// response. The remote node should respond because EIP-8 mandates ignoring additional
// trailing data.
func PingExtraData(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	req := pingWithJunk{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

// This test sends a PING packet with additional data and wrong 'from' field
// and expects a PONG response.
func PingExtraDataWrongFrom(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	wrongEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	req := pingWithJunk{
		Version:    4,
		From:       wrongEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

// This test sends a PING packet with an expiration in the past.
// The remote node should not respond.
func PingPastExpiration(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	req := v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: -futureExpiration(),
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

// This test sends an invalid packet. The remote node should not respond.
func WrongPacketType(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	req := pingWrongType{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("send", err)
	}
	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

// This test verifies that the default behaviour of ignoring 'from' fields is unaffected by
// the bonding process. After bonding, it pings the target with a different from endpoint.
func BondThenPingWithWrongFrom(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()
	bond(t, te)

	wrongEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	req2 := v4wire.Ping{
		Version:    4,
		From:       wrongEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &req2); err != nil {
		t.Fatal("send 2nd", err)
	}

	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read 2nd", err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong after bonding", reply.Name())
	}
}

// This test just sends FINDNODE. The remote node should not reply
// because the endpoint proof has not completed.
func FindnodeWithoutEndpointProof(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	req := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(req.Target[:])
	if err := te.send(te.l1, &req); err != nil {
		t.Fatal("sending find nodes", err)
	}
	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no response, got", reply)
	}
}

// BasicFindnode sends a FINDNODE request after performing the endpoint
// proof. The remote node should respond.
func BasicFindnode(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()
	bond(t, te)

	//now call find neighbours
	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	if err := te.send(te.l1, &findnode); err != nil {
		t.Fatal("sending findnode", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read find nodes", err)
	}
	if reply.Kind() != v4wire.NeighborsPacket {
		t.Fatal("Expected neighbors, got", reply.Name())
	}
}

// This test sends an unsolicited NEIGHBORS packet after the endpoint proof, then sends
// FINDNODE to read the remote table. The remote node should not return the node contained
// in the unsolicited NEIGHBORS packet.
func UnsolicitedNeighbors(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()
	bond(t, te)

	// Send unsolicited NEIGHBORS response.
	fakeKey, _ := crypto.GenerateKey()
	encFakeKey := v4wire.EncodePubkey(&fakeKey.PublicKey)
	neighbors := v4wire.Neighbors{
		Expiration: futureExpiration(),
		Nodes: []v4wire.Node{{
			ID:  encFakeKey,
			IP:  net.IP{1, 2, 3, 4},
			UDP: 30303,
			TCP: 30303,
		}},
	}
	if err := te.send(te.l1, &neighbors); err != nil {
		t.Fatal("NeighborsReq", err)
	}

	// Check if the remote node included the fake node.
	findnode := v4wire.Findnode{
		Expiration: futureExpiration(),
		Target:     encFakeKey,
	}
	if err := te.send(te.l1, &findnode); err != nil {
		t.Fatal("sending findnode", err)
	}
	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read find nodes", err)
	}
	if reply.Kind() != v4wire.NeighborsPacket {
		t.Fatal("Expected neighbors, got", reply.Name())
	}
	nodes := reply.(*v4wire.Neighbors).Nodes
	if contains(nodes, encFakeKey) {
		t.Fatal("neighbors response contains node from earlier unsolicited neighbors response")
	}
}

// This test sends FINDNODE with an expiration timestamp in the past.
// The remote node should not respond.
func FindnodePastExpiration(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()
	bond(t, te)

	findnode := v4wire.Findnode{
		Expiration: -futureExpiration(),
	}
	rand.Read(findnode.Target[:])
	if err := te.send(te.l1, &findnode); err != nil {
		t.Fatal("sending find nodes", err)
	}
	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

// bond performs the endpoint proof with the remote node.
func bond(t *testing.T, te *testenv) {
	ping := v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &ping); err != nil {
		t.Fatal("ping failed", err)
	}
	var gotPing, gotPong bool
	for !gotPing || !gotPong {
		req, hash, err := te.read(te.l1)
		if err != nil {
			t.Fatal(err)
		}
		switch req.(type) {
		case *v4wire.Ping:
			te.send(te.l1, &v4wire.Pong{
				To:         te.remoteEndpoint(),
				ReplyTok:   hash,
				Expiration: futureExpiration(),
			})
			gotPing = true
		case *v4wire.Pong:
			// TODO: maybe verify pong data here
			gotPong = true
		}
	}
}

// This test attempts to perform a traffic amplification attack against a
// 'victim' endpoint using FINDNODE. In this attack scenario, the attacker
// attempts to complete the endpoint proof non-interactively by sending a PONG
// with mismatching reply token from the 'victim' endpoint. The attack works if
// the remote node does not verify the PONG reply token field correctly. The
// attacker could then perform traffic amplification by sending many FINDNODE
// requests to the discovery node, which would reply to the 'victim' address.
func FindnodeAmplificationInvalidPongHash(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	// Send PING to start endpoint verification.
	ping := v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	}
	if err := te.send(te.l1, &ping); err != nil {
		t.Fatal(err)
	}

	var gotPing, gotPong bool
	for !gotPing || !gotPong {
		req, _, err := te.read(te.l1)
		if err != nil {
			t.Fatal(err)
		}
		switch req.(type) {
		case *v4wire.Ping:
			// Send PONG from this node ID, but with invalid ReplyTok.
			te.send(te.l1, &v4wire.Pong{
				To:         te.remoteEndpoint(),
				ReplyTok:   make([]byte, macSize),
				Expiration: futureExpiration(),
			})
			gotPing = true
		case *v4wire.Pong:
			gotPong = true
		}
	}

	// Now send FINDNODE. The remote node should not respond because our
	// PONG did not reference the PING hash.
	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	if err := te.send(te.l1, &findnode); err != nil {
		t.Fatal(err)
	}

	// If we receive a NEIGHBORS response, the attack worked and the test fails.
	reply, _, _ := te.read(te.l1)
	if reply != nil && reply.Kind() == v4wire.NeighborsPacket {
		t.Error("Got neighbors")
	}
}

// This test attempts to perform a traffic amplification attack using FINDNODE.
// The attack works if the remote node does not verify the IP address of FINDNODE
// against the endpoint verification proof done by PING/PONG.
func FindnodeAmplificationWrongIP(t *testing.T) {
	te := newTestEnv(*remote)
	defer te.close()

	// Do the endpoint proof from the l1 IP.
	bond(t, te)

	// Now send FINDNODE from the same node ID, but different IP address.
	// The remote node should not respond.
	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	if err := te.send(te.l2, &findnode); err != nil {
		t.Fatal(err)
	}
	// If we receive a NEIGHBORS response, the attack worked and the test fails.
	reply, _, _ := te.read(te.l2)
	if reply != nil {
		t.Error("Got NEIGHORS response for FINDNODE from wrong IP")
	}
}

func TestPing(t *testing.T) {
	t.Run("BasicPing", BasicPing)
	t.Run("WrongTo", PingWrongTo)
	t.Run("WrongFrom", PingWrongFrom)
	t.Run("ExtraData", PingExtraData)
	t.Run("ExtraDataWrongFrom", PingExtraDataWrongFrom)
	t.Run("PastExpiration", PingPastExpiration)
	t.Run("WrongPacketType", WrongPacketType)
	t.Run("BondThenPingWithWrongFrom", BondThenPingWithWrongFrom)
}

func TestAmplification(t *testing.T) {
	t.Run("InvalidPongHash", FindnodeAmplificationInvalidPongHash)
	t.Run("WrongIP", FindnodeAmplificationWrongIP)
}

func TestFindnode(t *testing.T) {
	t.Run("WithoutEndpointProof", FindnodeWithoutEndpointProof)
	t.Run("BasicFindnode", BasicFindnode)
	t.Run("UnsolicitedNeighbors", UnsolicitedNeighbors)
	t.Run("PastExpiration", FindnodePastExpiration)
}
