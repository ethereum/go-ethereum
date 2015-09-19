// Copyright 2015 The go-ethereum Authors
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

package rlpx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
)

const (
	staticFrameSize     uint32 = 8 * 1024
	frameHeaderSize            = 16 // encoded header
	frameHeaderFullSize        = 32 // encoded header + MAC
)

var (
	errProtocolClaimTimeout = errors.New("protocol for pending message was not claimed in time")
	errUnexpectedChunkStart = errors.New("received chunk start header for existing transfer")
	errChunkTooLarge        = errors.New("chunk size larger than remaining message size")
)

// readLoop runs in its own goroutine for each connection,
// dispatching frames to protocols.
func readLoop(c *Conn) (err error) {
	defer func() {
		// When the loop ends, forward the error to all protocols so
		// their next ReadPacket fails. Active chunked transfers also
		// need to cancel immediately so shutdown is not delayed.
		c.mu.Lock()
		for _, p := range c.proto {
			p.readClose(err)
			for _, pr := range p.xfers {
				pr.close(err)
			}
		}
		c.readErr = err
		c.mu.Unlock()
	}()

	// Local cache of claimed protocols.
	protos := make(map[uint16]*Protocol)

	for {
		// Read the next frame header.
		c.fd.SetReadDeadline(time.Now().Add(c.cfg.readIdleTimeout()))
		fsize, hdr, err := c.rw.readFrameHeader()
		if err != nil {
			return err
		}
		// Grab the protocol, checking the local cache before
		// interacting with the claims machinery in Conn.
		proto := protos[hdr.protocol]
		if proto == nil {
			if proto = c.waitForProtocol(hdr.protocol); proto == nil {
				return errProtocolClaimTimeout
			}
			protos[proto.id] = proto
		}
		// Wait until there is enough buffer space for the body
		// before reading it.
		err = proto.readBufSema.waitAcquire(fsize, c.cfg.readBufferWaitTimeout())
		if err != nil {
			return err
		}
		// Read the body of the frame.
		c.fd.SetReadDeadline(time.Now().Add(c.cfg.readTimeout()))
		body, err := c.rw.readFrameBody(fsize)
		if err != nil {
			return err
		}
		// Dispatch the frame to the protocol.
		// This shouldn't block.
		if pr := proto.xfers[hdr.contextID]; pr != nil {
			if hdr.chunkStart {
				return errUnexpectedChunkStart
			}
			end, err := pr.feed(body)
			if end {
				delete(proto.xfers, hdr.contextID)
			}
			if err != nil {
				return err
			}
		} else {
			pr, err := frameToPacket(proto, hdr, body)
			if err != nil {
				return err
			}
			if pr.bufN > 0 {
				// Track as ongoing transfer if there is still something
				// to buffer after the initial frame.
				proto.xfers[hdr.contextID] = pr
			}
			proto.feedPacket(pr)
		}
	}
}

// frameToPacket handles the initial frame for a new packet.
func frameToPacket(proto *Protocol, hdr frameHeader, frame frameBuffer) (pr *packetReader, err error) {
	if hdr.chunkStart {
		if uint32(len(frame)) > hdr.totalSize {
			return nil, fmt.Errorf("initial chunk size %d larger than total size %d", len(frame), hdr.totalSize)
		}
		if uint32(len(frame)) < hdr.totalSize {
			return newPacketReader(proto.readBufSema, hdr.totalSize, frame), nil
		}
	}
	return newPacketReader(proto.readBufSema, uint32(len(frame)), frame), nil
}

// packetReader is the payload of a packet.
// frames are appended to it as they are read from the connection.
type packetReader struct {
	// all of these can be accessed without locking
	// because Read is not safe for concurrent use.
	readBufs []frameBuffer
	origBufs []frameBuffer
	bufSema  *bufSema
	readN    uint32 // how much can still be read

	// these fields are protected by cond.L
	cond    *sync.Cond    // wakes waitFrame
	newBufs []frameBuffer // buffer inbox
	err     error         // error inbox
	bufN    uint32        // how much still needs to be buffered
}

func newPacketReader(bsem *bufSema, psize uint32, initialFrame frameBuffer) *packetReader {
	pr := &packetReader{
		bufSema: bsem,
		cond:    sync.NewCond(new(sync.Mutex)),
		readN:   psize,
		bufN:    psize,
	}
	if len(initialFrame) > 0 {
		pr.bufN -= uint32(len(initialFrame))
		pr.readBufs = []frameBuffer{initialFrame}
		pr.origBufs = []frameBuffer{initialFrame}
	}
	return pr
}

func (pr *packetReader) Read(rslice []byte) (int, error) {
	if err := pr.waitFrame(); err != nil {
		return 0, err
	}
	n := 0
	for i := 0; i < len(pr.readBufs) && n < len(rslice); i++ {
		nn, _ := pr.readBufs[i].Read(rslice[n:])
		n += nn
	}
	pr.afterRead(n)
	return n, nil
}

func (pr *packetReader) ReadByte() (byte, error) {
	if err := pr.waitFrame(); err != nil {
		return 0, err
	}
	b, _ := pr.readBufs[0].ReadByte()
	pr.afterRead(1)
	return b, nil
}

// blocks until at least one frame is available,
// then transfers any new frame buffers that have appeared
// to readBufs/origBufs.
func (pr *packetReader) waitFrame() error {
	if len(pr.readBufs) > 0 {
		return nil
	}
	if pr.readN == 0 {
		return io.EOF
	}
	pr.cond.L.Lock()
	defer pr.cond.L.Unlock()
	for len(pr.newBufs) == 0 && pr.err == nil {
		pr.cond.Wait()
	}
	pr.readBufs = append(pr.readBufs, pr.newBufs...)
	pr.origBufs = append(pr.origBufs, pr.newBufs...)
	pr.newBufs = pr.newBufs[:0]
	return pr.err
}

// removes drained buffers and decrements the read buffer semaphore.
func (pr *packetReader) afterRead(n int) {
	pr.readN -= uint32(n)
	drained := 0
	drainedLen := uint32(0)
	for i, buf := range pr.readBufs {
		if len(buf) != 0 {
			break
		}
		drained++
		drainedLen += uint32(len(pr.origBufs[i]))
	}
	if drained > 0 {
		pr.readBufs = pr.readBufs[:copy(pr.readBufs, pr.readBufs[drained:])]
		pr.origBufs = pr.origBufs[:copy(pr.origBufs, pr.origBufs[drained:])]
		pr.bufSema.release(drainedLen)
	}
}

func (pr *packetReader) close(err error) {
	pr.cond.L.Lock()
	pr.err = err
	pr.cond.Signal() // wake up waitFrame
	pr.cond.L.Unlock()
}

func (pr *packetReader) feed(frame frameBuffer) (end bool, err error) {
	pr.cond.L.Lock()
	defer pr.cond.L.Unlock()
	if uint32(len(frame)) > pr.bufN {
		pr.err = errChunkTooLarge
		end = true
	} else {
		pr.bufN -= uint32(len(frame))
		pr.newBufs = append(pr.newBufs, frame)
		end = pr.bufN == 0
	}
	pr.cond.Signal() // wake up waitFrame
	return end, pr.err
}

// represents a frame header that has been read.
type frameHeader struct {
	protocol, contextID uint16
	chunkStart          bool   // initial frame of chunked message
	totalSize           uint32 // total number of bytes of chunked message
}

// header types for sending
type chunkStartHeader struct {
	Protocol, ContextID uint16
	TotalSize           uint32
}
type regularHeader struct {
	Protocol, ContextID uint16
}

func decodeHeader(b []byte) (fsize uint32, h frameHeader, err error) {
	fsize = readInt24(b)
	if fsize == 0 {
		return 0, h, errors.New("zero-sized frame")
	}
	b = b[3:]
	lc, rest, err := rlp.SplitList(b)
	if err != nil {
		return fsize, h, err
	}
	// This is silly. rlp.DecodeBytes errors for data
	// after the value, so we need to pass a slice
	// containing just the value.
	hlist := b[:len(b)-len(rest)]

	switch cnt, _ := rlp.CountValues(lc); cnt {
	case 1:
		var in struct{ Protocol uint16 }
		err = rlp.DecodeBytes(hlist, &in)
		h.protocol = in.Protocol
	case 2:
		var in regularHeader
		err = rlp.DecodeBytes(hlist, &in)
		h.protocol = in.Protocol
		h.contextID = in.ContextID
	case 3:
		var in chunkStartHeader
		err = rlp.DecodeBytes(hlist, &in)
		h.protocol = in.Protocol
		h.contextID = in.ContextID
		h.totalSize = in.TotalSize
		h.chunkStart = true
	default:
		err = fmt.Errorf("too many list elements")
	}
	return fsize, h, err
}

// frameRW implements the framed wire protocol.
type frameRW struct {
	conn io.ReadWriter
	// for reading
	headbuf          []byte
	dec              cipher.Stream
	ingressMacCipher cipher.Block
	ingressMac       hash.Hash
	// for writing
	enc             cipher.Stream
	egressMacCipher cipher.Block
	egressMac       hash.Hash
}

func newFrameRW(conn io.ReadWriter, ingress, egress secrets) *frameRW {
	return &frameRW{
		conn:             conn,
		headbuf:          make([]byte, 32),
		enc:              cipher.NewCTR(mustBlockCipher("egress.encKey", egress.encKey), egress.encIV),
		egressMacCipher:  mustBlockCipher("egress.macKey", egress.macKey),
		egressMac:        egress.mac,
		dec:              cipher.NewCTR(mustBlockCipher("ingress.encKey", ingress.encKey), ingress.encIV),
		ingressMacCipher: mustBlockCipher("ingress.macKey", ingress.macKey),
		ingressMac:       ingress.mac,
	}
}

func mustBlockCipher(what string, key []byte) cipher.Block {
	c, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("invalid %s: %v", what, err))
	}
	return c
}

// sends a frame on the connection. the body buffer must placeholder bytes
// for the encoded frame header and its MAC.
func (rw *frameRW) sendFrame(hdr interface{}, body *frameBuffer) error {
	wbuf := *body
	usize := uint32(len(wbuf))
	if usize < frameHeaderFullSize {
		panic(fmt.Sprintf("invalid body buffer, size < %d", frameHeaderFullSize))
	}
	if usize-frameHeaderFullSize > maxUint24 {
		return errors.New("frame size overflows uint24")
	}

	// Write and encrypt the frame header to the buffer.
	headbuf := wbuf[:frameHeaderSize]
	putInt24(headbuf, usize-frameHeaderFullSize)
	headbufAfterSize := headbuf[3:3]
	rlp.Encode(&headbufAfterSize, hdr)
	rw.enc.XORKeyStream(headbuf, headbuf)
	copy(wbuf[frameHeaderSize:], updateMAC(rw.egressMac, rw.egressMacCipher, headbuf))

	// Write and encrypt frame data to the buffer.
	wbuf.pad16()
	rw.enc.XORKeyStream(wbuf[frameHeaderFullSize:], wbuf[frameHeaderFullSize:])
	rw.egressMac.Write(wbuf[frameHeaderFullSize:])
	fmacseed := rw.egressMac.Sum(nil)
	wbuf = append(wbuf, zero[:frameHeaderSize]...)
	copy(wbuf[len(wbuf)-16:], updateMAC(rw.egressMac, rw.egressMacCipher, fmacseed))

	// Send the whole buffered frame on the socket.
	_, err := rw.conn.Write(wbuf)
	*body = wbuf
	return err
}

func (rw *frameRW) readFrameHeader() (fsize uint32, hdr frameHeader, err error) {
	// Read the header and verify its MAC.
	if _, err := io.ReadFull(rw.conn, rw.headbuf); err != nil {
		return 0, hdr, err
	}
	shouldMAC := updateMAC(rw.ingressMac, rw.ingressMacCipher, rw.headbuf[:16])
	if !hmac.Equal(shouldMAC, rw.headbuf[16:]) {
		return 0, hdr, errors.New("bad header MAC")
	}
	rw.dec.XORKeyStream(rw.headbuf[:16], rw.headbuf[:16])

	// Parse the header.
	fsize, hdr, err = decodeHeader(rw.headbuf)
	if err != nil {
		err = fmt.Errorf("can't decode frame header: %v", err)
	}
	return fsize, hdr, err
}

func (rw *frameRW) readFrameBody(fsize uint32) (frameBuffer, error) {
	// Grab a buffer for the content.
	var rsize = fsize
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding // frame size rounded up to 16 byte boundary
	}
	fb := makeFrameReadBuffer(rsize + 16)
	if _, err := io.ReadFull(rw.conn, fb); err != nil {
		return nil, err
	}

	// Verify the body MAC and decrypt the content.
	mac, bb := fb[len(fb)-16:], fb[:len(fb)-16]
	rw.ingressMac.Write(bb)
	fmacseed := rw.ingressMac.Sum(nil)
	shouldMAC := updateMAC(rw.ingressMac, rw.ingressMacCipher, fmacseed)
	if !hmac.Equal(shouldMAC, mac) {
		return nil, errors.New("bad frame body MAC")
	}
	rw.dec.XORKeyStream(bb, bb)
	return bb[:fsize], nil
}

// updateMAC reseeds the given hash with encrypted seed.
// it returns the first 16 bytes of the hash sum after seeding.
func updateMAC(mac hash.Hash, block cipher.Block, seed []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, mac.Sum(aesbuf[:0]))
	for i := range aesbuf {
		aesbuf[i] ^= seed[i]
	}
	mac.Write(aesbuf)
	return mac.Sum(nil)[:16]
}

type frameBuffer []byte

func makeFrameWriteBuffer() *frameBuffer {
	buf := make(frameBuffer, frameHeaderFullSize, frameHeaderFullSize+staticFrameSize)
	return &buf
}

func makeFrameReadBuffer(size uint32) frameBuffer {
	return make(frameBuffer, size)
}

// resetForWrite truncates the buffer so it contains just enough space
// for an encoded frame header. it must be called before writing
// payload content for a new frame.
func (buf *frameBuffer) resetForWrite() {
	*buf = append((*buf)[:0], zero[:frameHeaderFullSize]...)
}

func (buf *frameBuffer) Write(s []byte) (n int, err error) {
	*buf = append(*buf, s...)
	return len(s), nil
}

func (buf *frameBuffer) Read(s []byte) (int, error) {
	if buf == nil || len(*buf) == 0 {
		return 0, io.EOF
	}
	n := copy(s, *buf)
	*buf = (*buf)[n:]
	return n, nil
}

func (buf *frameBuffer) ReadByte() (byte, error) {
	if buf == nil || len(*buf) == 0 {
		return 0, io.EOF
	}
	b := (*buf)[0]
	*buf = (*buf)[1:]
	return b, nil
}

func (buf *frameBuffer) pad16() {
	if padding := len(*buf) % 16; padding > 0 {
		*buf = append(*buf, zero[:16-padding]...)
	}
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(s []byte, v uint32) {
	s[0] = byte(v >> 16)
	s[1] = byte(v >> 8)
	s[2] = byte(v)
}
