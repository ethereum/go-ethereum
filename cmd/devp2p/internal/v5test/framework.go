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
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v5wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// errorPacket represents an error during packet reading.
// This exists to facilitate type-switching on the result of conn.read.
type errorPacket struct {
	err error
}

func (p *errorPacket) Kind() byte      { return 99 }
func (p *errorPacket) Name() string    { return fmt.Sprintf("error: %v", p.err) }
func (p *errorPacket) SetReqID([]byte) {}
func (p *errorPacket) Error() string   { return p.err.Error() }
func (p *errorPacket) Unwrap() error   { return p.err }

// This is the response timeout used in tests.
const waitTime = 300 * time.Millisecond

// conn is a connection to the node under test.
type conn struct {
	l1, l2     net.PacketConn
	localNode  *enode.LocalNode
	localKey   *ecdsa.PrivateKey
	remote     *enode.Node
	remoteAddr *net.UDPAddr

	codec         *v5wire.Codec
	lastRequest   v5wire.Packet
	lastChallenge *v5wire.Whoareyou
	idCounter     uint32
}

// newConn sets up a connection to the given node.
func newConn(dest *enode.Node, listen1, listen2 string) *conn {
	l1, err := net.ListenPacket("udp", fmt.Sprintf("%v:0", listen1))
	if err != nil {
		panic(err)
	}
	l2, err := net.ListenPacket("udp", fmt.Sprintf("%v:0", listen2))
	if err != nil {
		panic(err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	db, err := enode.OpenDB("")
	if err != nil {
		panic(err)
	}
	ln := enode.NewLocalNode(db, key)
	ln.SetStaticIP(laddr(l1).IP)
	ln.SetFallbackUDP(laddr(l1).Port)

	return &conn{
		l1:         l1,
		l2:         l2,
		localKey:   key,
		localNode:  ln,
		remote:     dest,
		remoteAddr: &net.UDPAddr{IP: dest.IP(), Port: dest.UDP()},
		codec:      v5wire.NewCodec(ln, key, mclock.System{}),
	}
}

// close shuts down the listener.
func (tc *conn) close() {
	tc.l1.Close()
	tc.l2.Close()
	tc.localNode.Database().Close()
}

// nextReqID creates a request id.
func (tc *conn) nextReqID() []byte {
	id := make([]byte, 4)
	tc.idCounter++
	binary.BigEndian.PutUint32(id, tc.idCounter)
	return id
}

// reqresp performs a request/response interaction on the given connection.
// The request is retried if a handshake is requested.
func (tc *conn) reqresp(c net.PacketConn, req v5wire.Packet) v5wire.Packet {
	tc.write(c, req, nil)
	resp := tc.read(c)
	if resp.Kind() == v5wire.WhoareyouPacket {
		challenge := resp.(*v5wire.Whoareyou)
		challenge.Node = tc.remote
		tc.write(c, req, challenge)
		return tc.read(c)
	}
	return resp
}

// write sends a packet on the given connection.
func (tc *conn) write(c net.PacketConn, p v5wire.Packet, challenge *v5wire.Whoareyou) {
	packet, _, err := tc.codec.Encode(tc.remote.ID(), tc.remoteAddr.String(), p, challenge)
	if err != nil {
		panic(fmt.Errorf("can't encode %v packet: %v", p.Name(), err))
	}
	if _, err := c.WriteTo(packet, tc.remoteAddr); err != nil {
		panic(fmt.Errorf("can't send %v: %v", p.Name(), err))
	}
}

// read waits for an incoming packet on the given connection.
func (tc *conn) read(c net.PacketConn) v5wire.Packet {
	buf := make([]byte, 1280)
	if err := c.SetReadDeadline(time.Now().Add(waitTime)); err != nil {
		return &errorPacket{err}
	}
	n, fromAddr, err := c.ReadFrom(buf)
	if err != nil {
		return &errorPacket{err}
	}
	_, _, p, err := tc.codec.Decode(buf[:n], fromAddr.String())
	if err != nil {
		return &errorPacket{err}
	}
	return p
}

func laddr(c net.PacketConn) *net.UDPAddr {
	return c.LocalAddr().(*net.UDPAddr)
}
