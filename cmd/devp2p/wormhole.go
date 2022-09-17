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

	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/urfave/cli/v2"
	"github.com/xtaci/kcp-go"
)

const (
	ecParityShards = 1
	ecDataShards   = 1
)

func setupKCP(s *kcp.UDPSession) {
	s.SetMtu(1200)
	s.SetStreamMode(true)

	// https://github.com/skywind3000/kcp/blob/master/README.en.md#protocol-configuration
	// Normal Mode: ikcp_nodelay(kcp, 0, 40, 0, 0);
	// Turbo Mode: ikcp_nodelay(kcp, 1, 10, 2, 1);

	s.SetNoDelay(1, 10, 2, 1)
	// s.SetNoDelay(0, 40, 0, 0)
}

func discv5WormholeSend(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int(verbosityFlag.Name)), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	// Create discv5 session.
	unhandled := make(chan discover.ReadPacket, 10)
	disc, socket := startV5WithUnhandled(ctx, unhandled)
	defer disc.Close()
	defer close(unhandled)

	// Send request
	n := getNodeArg(ctx)
	resp, err := disc.TalkRequest(n, "wrm", []byte("rand"))
	if err != nil {
		return err
	}
	if string(resp) != "ok" {
		return fmt.Errorf("talk request rejected: %s", string(resp))
	}

	addr := &net.UDPAddr{IP: n.IP(), Port: n.UDP()}

	conn := newUnhandledWrapper(addr, socket)
	go enqueueUnhandledPackets(unhandled, conn)

	sess, err := kcp.NewConn3(0, addr, nil, ecDataShards, ecParityShards, conn)
	if err != nil {
		log.Error("Could not establish kcp session", "err", err)
		return err
	}
	defer sess.Close()

	setupKCP(sess)

	log.Info("Transmitting data")
	for i := 0; i < 10; i++ {
		n, err := sess.Write([]byte("this is a very large file"))
		log.Info("Sent data", "n", n, "err", err)
	}
	if _, err := sess.Write([]byte("FIN")); err != nil {
		return fmt.Errorf("unable to close connection: %s", err)
	}
	return nil
}

func discv5WormholeReceive(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int(verbosityFlag.Name)), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	startSession := make(chan *net.UDPAddr)

	// Setup discv5 protocol.
	unhandled := make(chan discover.ReadPacket, 10)
	disc, socket := startV5WithUnhandled(ctx, unhandled)
	disc.RegisterTalkHandler("wrm", func(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
		startSession <- addr
		return []byte("ok")
	})
	defer close(unhandled)
	defer disc.Close()

	// Print ENR.
	fmt.Println(disc.Self())

	// Wait for talk request, then start the session.
	addr := <-startSession
	conn := newUnhandledWrapper(addr, socket)
	go enqueueUnhandledPackets(unhandled, conn)

	s, err := kcp.NewConn3(0, addr, nil, ecDataShards, ecParityShards, conn)
	if err != nil {
		log.Error("Could not establish kcp session", "err", err)
		return err
	}
	defer s.Close()

	log.Info("KCP socket accepted")
	setupKCP(s)

	for {
		buf := make([]byte, 2048)
		n, err := s.Read(buf)
		if err != nil {
			return err
		}
		if string(buf[:n]) == "FIN" {
			log.Trace("connection finished")
			return nil
		}

		log.Info("Read KCP data", "data", string(buf[:n]))
	}
}

func handleWormholeTalkrequest(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
	log.Info("Handling talk request", "from", addr, "id", id, "data", fmt.Sprintf("%x", data))
	return []byte("ok")
}

func enqueueUnhandledPackets(ch <-chan discover.ReadPacket, o *unhandledWrapper) {
	for packet := range ch {
		o.enqueue(packet.Data)
	}
}

type unhandledWrapper struct {
	out net.PacketConn

	mu      sync.Mutex
	flag    *sync.Cond
	inqueue [][]byte
	remote  *net.UDPAddr
}

func newUnhandledWrapper(remote *net.UDPAddr, out net.PacketConn) *unhandledWrapper {
	o := &unhandledWrapper{out: out, remote: remote}
	o.flag = sync.NewCond(&o.mu)
	return o
}

func (o *unhandledWrapper) enqueue(p []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.inqueue = append(o.inqueue, p)
	o.flag.Broadcast()
}

// ReadFrom delivers a single packet from o.inqueue into the buffer p.
// If a packet does not fit into the buffer, the remaining bytes of the packet
// are discarded.
func (o *unhandledWrapper) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	o.mu.Lock()
	for len(o.inqueue) == 0 {
		o.flag.Wait()
	}

	// Move packet data into p.
	n = copy(p, o.inqueue[0])

	// Delete the packet from inqueue.
	copy(o.inqueue, o.inqueue[1:])
	o.inqueue = o.inqueue[:len(o.inqueue)-1]

	o.mu.Unlock()

	log.Info("KCP read", "buf", len(p), "n", n, "remaining-in-q", len(o.inqueue))
	return n, o.remote, nil
}

func (o *unhandledWrapper) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = o.out.WriteTo(p, addr)
	log.Info("KCP write", "buf", len(p), "n", n, "err", err)
	return n, err
}

func (o *unhandledWrapper) LocalAddr() net.Addr {
	panic("not implemented")
}

func (o *unhandledWrapper) Close() error                       { return nil }
func (o *unhandledWrapper) SetDeadline(t time.Time) error      { return nil }
func (o *unhandledWrapper) SetReadDeadline(t time.Time) error  { return nil }
func (o *unhandledWrapper) SetWriteDeadline(t time.Time) error { return nil }
