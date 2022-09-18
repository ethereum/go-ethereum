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
	"io"
	"net"
	"os"

	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/urfave/cli/v2"
	"github.com/xtaci/kcp-go"
)

const (
	ecParityShards = 3
	ecDataShards   = 10
)

func setupKCP(s *kcp.UDPSession) {
	s.SetMtu(1200)
	// s.SetStreamMode(true)

	// https://github.com/skywind3000/kcp/blob/master/README.en.md#protocol-configuration
	// Normal Mode: ikcp_nodelay(kcp, 0, 40, 0, 0);
	// Turbo Mode: ikcp_nodelay(kcp, 1, 10, 2, 1);

	// s.SetNoDelay(1, 10, 2, 1)
	s.SetNoDelay(0, 40, 0, 0)
}

func discv5WormholeSend(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int(verbosityFlag.Name)), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("send command needs destination node and file name as arguments")
	}
	n := getNodeArg(ctx)
	filePath := ctx.Args().Get(1)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// Create discv5 session.
	unhandled := make(chan discover.ReadPacket, 10)
	disc, socket := startV5WithUnhandled(ctx, unhandled)
	defer disc.Close()
	defer close(unhandled)

	// Send request
	xfer := &xferStart{Size: uint64(fileInfo.Size())}
	if err := requestTransfer(disc, n, xfer); err != nil {
		return err
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
	if _, err := io.CopyN(sess, file, fileInfo.Size()); err != nil {
		return fmt.Errorf("copy error: %v", err)
	}

	// KCP writes are a bit 'async', so wait for the remote send ACK in the end.
	sess.SetReadDeadline(time.Now().Add(5 * time.Second))
	ackbuf := make([]byte, 3)
	_, err = sess.Read(ackbuf)
	if err != nil {
		log.Error("FIN-ACK read error", "err", err)
	}
	kcpStatsDump(kcp.DefaultSnmp)
	return nil
}

func discv5WormholeReceive(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int(verbosityFlag.Name)), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	if ctx.Args().Len() < 1 {
		return fmt.Errorf("receive command needs filename as argument %d", ctx.Args().Len())
	}
	filePath := ctx.Args().First()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	err = doReceive(ctx, file)
	file.Close()
	if err != nil {
		os.Remove(filePath)
	}
	return err
}

func doReceive(ctx *cli.Context, file *os.File) error {
	startSession := make(chan *xferStart)

	// Setup discv5 protocol.
	unhandled := make(chan discover.ReadPacket, 10)
	disc, socket := startV5WithUnhandled(ctx, unhandled)
	disc.RegisterTalkHandler("wrm", func(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
		var startReq xferStart
		err := rlp.DecodeBytes(data, &startReq)
		if err != nil {
			resp, _ := rlp.EncodeToBytes(&xferResponse{Accept: false, Error: "bad request"})
			return resp
		}

		startReq.fromNode = id
		startReq.fromAddr = addr
		startSession <- &startReq

		resp, _ := rlp.EncodeToBytes(&xferResponse{Accept: true})
		return resp
	})

	defer close(unhandled)
	defer disc.Close()

	// Print ENR.
	fmt.Println(disc.Self())

	// Wait for talk request, then start the session.
	xfer := <-startSession
	conn := newUnhandledWrapper(xfer.fromAddr, socket)
	go enqueueUnhandledPackets(unhandled, conn)

	s, err := kcp.NewConn3(0, xfer.fromAddr, nil, ecDataShards, ecParityShards, conn)
	if err != nil {
		log.Error("Could not establish kcp session", "err", err)
		return err
	}
	defer s.Close()

	log.Info("KCP socket accepted")
	setupKCP(s)

	if _, err := io.CopyN(file, s, int64(xfer.Size)); err != nil {
		return fmt.Errorf("copy failed: %v", err)
	}

	s.Write([]byte("ACK"))
	kcpStatsDump(kcp.DefaultSnmp)
	fmt.Println("transfer OK")
	return nil
}

type xferStart struct {
	Size uint64
	// Key [16]byte

	fromNode enode.ID
	fromAddr *net.UDPAddr
}

type xferResponse struct {
	Accept bool
	Error  string
	// Key [16]byte
}

func requestTransfer(disc *discover.UDPv5, node *enode.Node, xfer *xferStart) error {
	req, err := rlp.EncodeToBytes(xfer)
	if err != nil {
		return err
	}
	resp, err := disc.TalkRequest(node, "wrm", req)
	if err != nil {
		return err
	}
	var xresp xferResponse
	if err := rlp.DecodeBytes(resp, &xresp); err != nil {
		return err
	}
	if !xresp.Accept {
		return fmt.Errorf("transfer not accepted: %s", xresp.Error)
	}
	return nil
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
	defer o.mu.Unlock()

	// Move packet data into p.
	n = copy(p, o.inqueue[0])

	// Delete the packet from inqueue.
	copy(o.inqueue, o.inqueue[1:])
	o.inqueue = o.inqueue[:len(o.inqueue)-1]

	// log.Info("KCP read", "buf", len(p), "n", n, "remaining-in-q", len(o.inqueue))
	// kcpStatsDump(kcp.DefaultSnmp)
	return n, o.remote, nil
}

func (o *unhandledWrapper) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = o.out.WriteTo(p, addr)
	// log.Info("KCP write", "buf", len(p), "n", n, "err", err)
	return n, err
}

func (o *unhandledWrapper) LocalAddr() net.Addr {
	panic("not implemented")
}

func (o *unhandledWrapper) Close() error                       { return nil }
func (o *unhandledWrapper) SetDeadline(t time.Time) error      { return nil }
func (o *unhandledWrapper) SetReadDeadline(t time.Time) error  { return nil }
func (o *unhandledWrapper) SetWriteDeadline(t time.Time) error { return nil }

func kcpStatsDump(snmp *kcp.Snmp) {
	header := snmp.Header()
	for i, value := range snmp.ToSlice() {
		fmt.Printf("%s: %s\n", header[i], value)
	}
}
