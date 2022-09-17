// Copyright 2022 The go-ethereum Authors
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

package main

import (
	"fmt"
	"net"

	"crypto/sha1"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/urfave/cli/v2"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
	"sync"
	"time"
)

func discv5WormholeSend(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV5(ctx)
	defer disc.Close()
	fmt.Println(disc.Ping(n))
	resp, err := disc.TalkRequest(n, "wrm", []byte("rand"))
	log.Info("Talkrequest", "resp", fmt.Sprintf("%v (%x)", string(resp), resp), "err", err)

	// taken from https://github.com/xtaci/kcp-go/blob/master/examples/echo.go#L51
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	if sess, err := kcp.DialWithOptions(fmt.Sprintf("%v:%d", n.IP(), n.UDP()), block, 10, 3); err == nil {
		log.Info("Transmitting data")
		n, err := sess.Write([]byte("this is a very large file"))
		log.Info("Sent data", "n", n, "err", err)
		log.Info("Closing session")
		sess.Close()
	} else {
		log.Error("Could not establish kcp session", "err", err)
	}
	return nil
}

func discv5WormholeReceive(ctx *cli.Context) error {
	var unhandled = make(chan discover.ReadPacket)

	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	kcpWrapper := newUnhandledWrapper(unhandled)
	disc := startV5WithUnhandled(ctx, unhandled)
	defer disc.Close()
	defer close(unhandled)

	fmt.Println(disc.Self())

	disc.RegisterTalkHandler("wrm", handleWormholeTalkrequest)
	kcp.ServeConn(block, 10, 3, kcpWrapper)
	handleUnhandledLoop(kcpWrapper)
	return nil
}

// TalkRequestHandler callback processes a talk request and optionally returns a reply
//type TalkRequestHandler func(enode.ID, *net.UDPAddr, []byte) []byte

func handleWormholeTalkrequest(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
	log.Info("Handling talk request", "from", addr, "id", id, "data", fmt.Sprintf("%x", data))
	return []byte("oll korrekt!")
}

func handleUnhandledLoop(wrapper *ourPacketConn) {
	for {
		select {
		case packet := <-wrapper.unhandled:
			log.Info("Unhandled packet handled", "from", packet.Addr, "data", fmt.Sprintf("%v %#x", string(packet.Data), packet.Data))
			wrapper.inqueue = append(wrapper.inqueue, packet.Data...)
		}
	}
}

func newUnhandledWrapper(packetCh chan discover.ReadPacket) *ourPacketConn {
	return &ourPacketConn{
		unhandled: packetCh,
	}
}

type ourPacketConn struct {
	unhandled chan discover.ReadPacket
	inqueue   []byte
	readMu    sync.Mutex
}

func (o *ourPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// TODO: We must deliver from our wrapper.inqueue here. Make sure not to
	// modify that thing from two threads at once.
	o.readMu.Lock()
	defer o.readMu.Unlock()
	if len(o.inqueue) > 0 {
		n = copy(p, o.inqueue)
		return n, nil, nil
	}
	return 0, nil, nil
}

func (o *ourPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	//TODO implement me
	panic("implement me")
}

func (o *ourPacketConn) LocalAddr() net.Addr {
	//TODO implement me
	panic("implement me")
}

func (o *ourPacketConn) Close() error                       { return nil }
func (o *ourPacketConn) SetDeadline(t time.Time) error      { return nil }
func (o *ourPacketConn) SetReadDeadline(t time.Time) error  { return nil }
func (o *ourPacketConn) SetWriteDeadline(t time.Time) error { return nil }
