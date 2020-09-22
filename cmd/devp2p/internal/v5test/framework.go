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

const waitTime = 300 * time.Millisecond

type testenv struct {
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

type errorPacket struct {
	err error
}

func (p *errorPacket) Kind() byte      { return 99 }
func (p *errorPacket) Name() string    { return fmt.Sprintf("error: %v", p.err) }
func (p *errorPacket) SetReqID([]byte) {}
func (p *errorPacket) Error() string   { return p.err.Error() }
func (p *errorPacket) Unwrap() error   { return p.err }

func newTestEnv(dest *enode.Node, listen1, listen2 string) *testenv {
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

	return &testenv{
		l1:         l1,
		l2:         l2,
		localKey:   key,
		localNode:  ln,
		remote:     dest,
		remoteAddr: &net.UDPAddr{IP: dest.IP(), Port: dest.UDP()},
		codec:      v5wire.NewCodec(ln, key, mclock.System{}),
	}
}

func (te *testenv) close() {
	te.l1.Close()
	te.l2.Close()
	te.localNode.Database().Close()
}

func (te *testenv) nextReqID() []byte {
	id := make([]byte, 4)
	te.idCounter++
	binary.BigEndian.PutUint32(id, te.idCounter)
	return id
}

func (te *testenv) reqresp(c net.PacketConn, req v5wire.Packet) v5wire.Packet {
	te.write(c, req, nil)
	resp := te.read(c)
	if resp.Kind() == v5wire.WhoareyouPacket {
		challenge := resp.(*v5wire.Whoareyou)
		challenge.Node = te.remote
		te.write(c, req, challenge)
		return te.read(c)
	}
	return resp
}

func (te *testenv) write(c net.PacketConn, p v5wire.Packet, challenge *v5wire.Whoareyou) {
	packet, _, err := te.codec.Encode(te.remote.ID(), te.remoteAddr.String(), p, challenge)
	if err != nil {
		panic(fmt.Errorf("can't encode %v packet: %v", p.Name(), err))
	}
	if _, err := c.WriteTo(packet, te.remoteAddr); err != nil {
		panic(fmt.Errorf("can't send %v: %v", p.Name(), err))
	}
}

func (te *testenv) read(c net.PacketConn) v5wire.Packet {
	buf := make([]byte, 1280)
	if err := c.SetReadDeadline(time.Now().Add(waitTime)); err != nil {
		return &errorPacket{err}
	}
	n, fromAddr, err := c.ReadFrom(buf)
	if err != nil {
		return &errorPacket{err}
	}
	_, _, p, err := te.codec.Decode(buf[:n], fromAddr.String())
	if err != nil {
		return &errorPacket{err}
	}
	return p
}

func laddr(c net.PacketConn) *net.UDPAddr {
	return c.LocalAddr().(*net.UDPAddr)
}
