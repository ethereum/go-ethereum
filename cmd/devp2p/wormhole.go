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

func discv5WormholeSend(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	// Create discv5 session.
	unhandled := make(chan discover.ReadPacket)
	disc := startV5WithUnhandled(ctx, unhandled)
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

	conn := newUnhandledWrapper(unhandled, disc)
	go handleUnhandledLoop(conn)

	if sess, err := kcp.NewConn(fmt.Sprintf("%v:%d", n.IP(), n.UDP()), nil, 10, 3, conn); err == nil {
		log.Info("Transmitting data")
		for i := 0; i < 10; i++ {
			n, err := sess.Write([]byte("this is a very large file"))
			log.Info("Sent data", "n", n, "err", err)
		}
		log.Info("Closing session")
		sess.Close()
	} else {
		log.Error("Could not establish kcp session", "err", err)
	}
	return nil
}

func discv5WormholeReceive(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	// Setup discv5 protocol.
	unhandled := make(chan discover.ReadPacket)
	disc := startV5WithUnhandled(ctx, unhandled)
	disc.RegisterTalkHandler("wrm", handleWormholeTalkrequest)
	defer close(unhandled)
	defer disc.Close()

	// Print ENR.
	fmt.Println(disc.Self())

	// Create wrapped connection based on discv5 unhandled channel.
	conn := newUnhandledWrapper(unhandled, disc)
	l, err := kcp.ServeConn(nil, 10, 3, conn)
	if err != nil {
		return err
	}

	// Sping up routine to buffer packets on unhandled channel.
	go handleUnhandledLoop(conn)

	for {
		log.Info("Waiting for KCP conn")
		s, err := l.Accept()
		if err != nil {
			log.Error("Error", "err", err)
			return err
		}

		log.Info("KCP socket accepted")
		go func(net.Conn) {
			for {
				buf := make([]byte, 2048)
				n, err := s.Read(buf)
				if err != nil {
					log.Error("Error", "err", err)
					return
				}
				log.Info("Read KCP data", "data", string(buf[:n]))
			}
		}(s)
	}
}

func handleWormholeTalkrequest(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
	log.Info("Handling talk request", "from", addr, "id", id, "data", fmt.Sprintf("%x", data))
	return []byte("ok")
}

func handleUnhandledLoop(wrapper *unhandledWrapper) {
	for {
		select {
		case packet := <-wrapper.unhandled:
			if len(packet.Data) > 10 {
				log.Info("Unhandled packet handled", "from", packet.Addr, "size", len(packet.Data),
					"data", fmt.Sprintf("%#x...", packet.Data[:10]))
			} else {
				log.Info("Unhandled packet handled", "from", packet.Addr, "size", len(packet.Data))
			}

			wrapper.readMu.Lock()
			// This is a bit hacky: setting the remote addr here.
			// Ideally we shouldn't need to do it on _every_ single packet really.
			wrapper.remote = packet.Addr
			wrapper.inqueue = append(wrapper.inqueue, packet.Data...)
			wrapper.flag.Broadcast()
			wrapper.readMu.Unlock()

		}
	}
}

func newUnhandledWrapper(packetCh chan discover.ReadPacket, disc *discover.UDPv5) *unhandledWrapper {
	x := sync.Mutex{}
	cond := sync.NewCond(&x)
	return &unhandledWrapper{
		unhandled: packetCh,
		readMu:    &x,
		flag:      cond,
		disc:      disc,
	}
}

type unhandledWrapper struct {
	unhandled chan discover.ReadPacket
	inqueue   []byte
	remote    *net.UDPAddr
	disc      *discover.UDPv5
	readMu    *sync.Mutex
	flag      *sync.Cond
}

func (o *unhandledWrapper) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// TODO: We must deliver from our wrapper.inqueue here. Make sure not to
	// modify that thing from two threads at once.

	o.readMu.Lock()
	for len(o.inqueue) == 0 {
		o.flag.Wait()
		fmt.Printf("Woke up reader\n")
	}
	defer o.readMu.Unlock()
	n = copy(p, o.inqueue)
	o.inqueue = make([]byte, 0)
	log.Info("Packet conn delivered to reader", "n", n)
	return n, o.remote, nil
}

func (o *unhandledWrapper) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		return 0, err
	}
	return o.disc.WriteTo(udpAddr, p)
}

func (o *unhandledWrapper) LocalAddr() net.Addr {
	panic("not implemented")
}

func (o *unhandledWrapper) Close() error                       { return nil }
func (o *unhandledWrapper) SetDeadline(t time.Time) error      { return nil }
func (o *unhandledWrapper) SetReadDeadline(t time.Time) error  { return nil }
func (o *unhandledWrapper) SetWriteDeadline(t time.Time) error { return nil }
