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
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
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
	s.SetStreamMode(true)
	s.SetWindowSize(10000, 10000)

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
	fName := filepath.Base(filepath.Clean(file.Name()))
	xfer := &xferStart{Size: uint64(fileInfo.Size()), Filename: fName}
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
	progress := newDownloadWriter(sess, int64(xfer.Size))

	inbuf := bufio.NewReader(file)
	if _, err := io.CopyN(progress, inbuf, fileInfo.Size()); err != nil {
		return fmt.Errorf("copy error: %v", err)
	}
	if err := progress.dstBuf.Flush(); err != nil {
		return fmt.Errorf("copy error: %v", err)
	}

	// KCP writes are a bit 'async', so wait for the remote send ACK in the end.
	log.Info("Done writing to KCP, waiting for FIN-ACK")
	sess.SetReadDeadline(time.Now().Add(30 * time.Minute))
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

	path := "./" // filePath is where received files are stored.
	if ctx.Args().Len() > 0 {
		path = ctx.Args().First()
		fInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("first argument must be a directory: %w", err)
		}
		if !fInfo.IsDir() {
			return fmt.Errorf("first argument must be a directory")
		}
	}
	rootPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	log.Info("File destination set", "directory", rootPath)
	return doReceive(rootPath, ctx)
}

//func promptUser(q string) (string, error) {
//	var ans string
//	fmt.Print(q)
//	_, err := fmt.Scanln(&ans)
//	return ans, err
//}

func doReceive(root string, ctx *cli.Context) error {
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
		/*
			Unfortunately, we cannot do user-prompt here, because the TALKREQUEST has a very short
			timeout-window. TODO fix.
		*/
		//ans, err := promptUser(fmt.Sprintf("Incoming request:\n  Filename: %v\n  File size %d bytes\nAccept transfer? (Y/n) > ",
		//	startReq.Filename, startReq.Size))
		//if err != nil {
		//	log.Error("Error doing prompt", "err", err)
		//} else if ans == "Y" || ans == "y" {
		startReq.fromNode = id
		startReq.fromAddr = addr
		startSession <- &startReq
		resp, _ := rlp.EncodeToBytes(&xferResponse{Accept: true})
		return resp
		//}
		//resp, _ := rlp.EncodeToBytes(&xferResponse{Accept: false})
		//return resp
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

	file, err := createTmpDest(root, xfer.Filename)
	if err != nil {
		return err
	}
	fName := filepath.Base(file.Name())
	if err != nil {
		return err
	}

	progress := newDownloadWriter(file, int64(xfer.Size))
	if _, err := io.CopyN(progress, s, int64(xfer.Size)); err != nil {
		// Clean up
		progress.Close()
		os.Remove(filepath.Join(root, fName))
		return fmt.Errorf("copy failed: %v", err)
	}

	// Send FIN-ACK.
	s.Write([]byte("ACK"))

	kcpStatsDump(kcp.DefaultSnmp)
	progress.Close()
	if err := os.Rename(filepath.Join(root, fName), filepath.Join(root, xfer.Filename)); err != nil {
		log.Error("Error renaming file", "err", err)
	}
	log.Info("Transfer OK", "file", filepath.Join(root, xfer.Filename))
	return nil
}

// createTmpDest creates a temporary destination-file, but avoid overwriting an
// existing file. If the incoming file is "file.txt", then it will create
// "file.txt.0.tmp" first, and if that already exists, "file,txt.1.tmp" etc.
// This method will return error if the filename is not the canonical represention,
// (i.e: if the name contains more than just the filename).
func createTmpDest(rootPath, filename string) (*os.File, error) {
	if filepath.Clean(filename) != filename {
		return nil, fmt.Errorf("filename not canonical, possibly path-traversal attempt: %v != %v", filename, filepath.Clean(filename))
	}
	for i := 0; i < 20; i++ {
		fullPath := fmt.Sprintf("%v.%d.tmp", filepath.Join(rootPath, filename), i)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// We can create a file here
			file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}
			return file, nil
		}
	}
	return nil, errors.New("file creation aborted after 20 retries, please clean up destination path")
}

// xferStart is sent in the initial TALKREQUEST, and contains some metadata
// about the file file that is being requested to send.
type xferStart struct {
	Size     uint64
	Filename string
	// Key [16]byte

	fromNode enode.ID
	fromAddr *net.UDPAddr
}

// xferResponse is the response to the xferStart request
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
		return fmt.Errorf("talk request error: %w", err)
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
	// log.Info("KCP packet out", "len", n, "err", err)
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

type downloadWriter struct {
	file    io.WriteCloser
	dstBuf  *bufio.Writer
	size    int64
	written int64
	lastpct int64
}

func newDownloadWriter(dst io.WriteCloser, size int64) *downloadWriter {
	return &downloadWriter{
		file:   dst,
		dstBuf: bufio.NewWriter(dst),
		size:   size,
	}
}

func (w *downloadWriter) Write(buf []byte) (int, error) {
	n, err := w.dstBuf.Write(buf)

	// Report progress.
	w.written += int64(n)
	pct := w.written * 10 / w.size * 10
	if pct != w.lastpct {
		if w.lastpct != 0 {
			fmt.Print("...")
		}
		fmt.Print(pct, "%")
		w.lastpct = pct
	}
	return n, err
}

func (w *downloadWriter) Close() error {
	if w.lastpct > 0 {
		fmt.Println() // Finish the progress line.
	}
	flushErr := w.dstBuf.Flush()
	closeErr := w.file.Close()
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}
