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

// Package rlpx implements the RLPx secure transport protocol.
//
// RLPx multiplexes packet streams over an authenticated and encrypted
// network connection.
//
// The wire protocol specification lives at https://github.com/ethereum/devp2p.
//
// Protocols
//
// RLPx transports packet streams for multiple protocols on the same
// connection, ensuring that available bandwidth is fairly distributed
// among them. Negotiation of protocol identifiers is not part of the
// transport layer and is typically done by sending messages with
// protocol identifier 0.
package rlpx

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	defaultHandshakeTimeout      = 5 * time.Second
	defaultReadTimeout           = 10 * time.Second
	defaultReadIdleTimeout       = 25 * time.Second
	defaultWriteTimeout          = 10 * time.Second
	defaultReadBufferSize        = 2 * 1024 * 1024
	defaultReadBufferWaitTimeout = 5 * time.Second
)

// A Config structure is used to configure an RLPx client or server
// connection. After one has been passed to any function in package
// rlpx, it must not be modified. A Config may be reused; the rlpx
// package will also not modify it.
type Config struct {
	// Key is the private key of the server. The key must use the
	// secp256k1 curve, other curves are not supported.
	// This field is required for both client and server connections.
	Key *ecdsa.PrivateKey

	HandshakeTimeout time.Duration // for the key negotiation handshake (default 5s)
	ReadIdleTimeout  time.Duration // applies while waiting for a new frame (default 25s)
	ReadTimeout      time.Duration // for reading the payload data of a single frame (default 10s)
	WriteTimeout     time.Duration // for writing one frame of data (default 10s)

	// ReadBufferSize controls how much data can be buffered for each
	// protocol. The default is 2MB for compatibility with legacy
	// peers.
	//
	// If the read buffer is full, the implementation waits for
	// buffer space to become available. The connection is closed if
	// no space becomes available within the timeout (default 5s).
	ReadBufferSize        uint32
	ReadBufferWaitTimeout time.Duration

	// Forces use of the version 4 handshake.
	ForceV4 bool
}

func (cfg *Config) handshakeTimeout() time.Duration {
	if cfg.HandshakeTimeout != 0 {
		return cfg.HandshakeTimeout
	}
	return defaultHandshakeTimeout
}

func (cfg *Config) readTimeout() time.Duration {
	if cfg.ReadTimeout != 0 {
		return cfg.ReadTimeout
	}
	return defaultReadTimeout
}

func (cfg *Config) readIdleTimeout() time.Duration {
	if cfg.ReadIdleTimeout != 0 {
		return cfg.ReadIdleTimeout
	}
	return defaultReadIdleTimeout
}

func (cfg *Config) writeTimeout() time.Duration {
	if cfg.WriteTimeout != 0 {
		return cfg.WriteTimeout
	}
	return defaultWriteTimeout
}

func (cfg *Config) readBufferWaitTimeout() time.Duration {
	if cfg.ReadBufferWaitTimeout != 0 {
		return cfg.ReadBufferWaitTimeout
	}
	return defaultReadBufferWaitTimeout
}

func (cfg *Config) readBufferSize() uint32 {
	if cfg.ReadBufferSize != 0 {
		return cfg.ReadBufferSize
	}
	return defaultReadBufferSize
}

// Conn represents an RLPx connection.
type Conn struct {
	// readonly fields
	cfg           *Config
	isServer      bool
	fd            net.Conn
	handshake     sync.Once
	handshakeRand handshakeRandSource // for testing

	wmu      sync.Mutex // excludes writes on rw
	rw       *frameRW   // set after handshake
	remoteID *ecdsa.PublicKey
	vsn      uint // negotiated version

	mu      sync.Mutex
	proto   map[uint16]*Protocol
	readErr error
}

// Client returns a new client side RLPx connection using fd as the
// underlying transport. The public key of the remote end must be
// known in advance.
//
// config must not be nil and must contain a
// valid private key.
func Client(fd net.Conn, remotePubkey *ecdsa.PublicKey, config *Config) *Conn {
	c := newConn(fd, config)
	c.remoteID = remotePubkey
	return c
}

// Server returns a new server side RLPx connection using fd as the
// underlying transport. The configuration config must be non-nil and
// must contain a valid private key
func Server(fd net.Conn, config *Config) *Conn {
	c := newConn(fd, config)
	c.isServer = true
	return c
}

func newConn(fd net.Conn, config *Config) *Conn {
	return &Conn{
		fd:    fd,
		cfg:   config,
		proto: make(map[uint16]*Protocol),
	}
}

// Handshake runs the client or server handshake protocol if it has
// not yet been run. Most uses of this package need not call Handshake
// explicitly: the first Read or Write will call it automatically.
func (c *Conn) Handshake() (err error) {
	// TODO: check cfg.Key curve, maybe panic earlier
	c.handshake.Do(func() {
		if c.handshakeRand == nil {
			c.handshakeRand = realRandSource{}
		}
		var (
			ingress, egress secrets
			rid             *ecdsa.PublicKey
			vsn             uint
		)
		c.fd.SetDeadline(time.Now().Add(c.cfg.handshakeTimeout()))
		if c.isServer {
			vsn, rid, ingress, egress, err = c.recipientHandshake()
		} else {
			vsn, ingress, egress, err = c.initiatorHandshake()
		}
		if err != nil {
			return
		}

		c.mu.Lock()
		c.vsn = vsn
		if rid != nil {
			c.remoteID = rid
		}
		c.mu.Unlock()
		c.rw = newFrameRW(c.fd, ingress, egress)
		go readLoop(c)
	})
	if err == nil && c.rw == nil {
		return errors.New("handshake failed")
	}
	return err
}

// LocalAddr returns the local network address of the underlying net.Conn.
func (c *Conn) LocalAddr() net.Addr {
	return c.fd.LocalAddr()
}

// RemoteAddr returns the remote network address of the underlying net.Conn.
func (c *Conn) RemoteAddr() net.Addr {
	return c.fd.RemoteAddr()
}

// RemoteID returns the public key of the remote end.
// If the remote identity is not yet known, it returns nil.
func (c *Conn) RemoteID() *ecdsa.PublicKey {
	c.mu.Lock()
	id := c.remoteID
	c.mu.Unlock()
	return id
}

// Version returns the negotiated RLPx version of the connection.
// The return value is zero before the handshake has executed and
// can be 4 or 5 afterwards.
func (c *Conn) Version() uint {
	c.mu.Lock()
	vsn := c.vsn
	c.mu.Unlock()
	return vsn
}

// Close closes the connection.
func (c *Conn) Close() error {
	// TODO: shut down reader/wr
	return c.fd.Close()
}

// Protocol returns a handle for the given protocol id.
// It can be called at most once for any given id,
// subsequent call with the same id will panic.
func (c *Conn) Protocol(id uint16) *Protocol {
	p := c.getProtocol(id)
	close(p.claimSignal) // panics when claimed twice
	return p
}

// waits until the given protocol is claimed by a call to Protocol.
func (c *Conn) waitForProtocol(id uint16) *Protocol {
	p := c.getProtocol(id)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		return nil
	case <-p.claimSignal:
		return p
	}
}

func (c *Conn) getProtocol(id uint16) *Protocol {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.proto[id] == nil {
		c.proto[id] = newProtocol(c, id)
	}
	return c.proto[id]
}

// Protocol is a handle for the given protocol.
type Protocol struct {
	c           *Conn
	claimed     bool
	id          uint16
	claimSignal chan struct{}

	// for readLoop
	xfers       map[uint16]*packetReader
	readBufSema *bufSema

	// for ReadPacket
	readCond   *sync.Cond // unblocks ReadPacket
	newPackets []*packetReader
	readErr    error

	// for writing
	contextidSeq uint16
}

func newProtocol(c *Conn, id uint16) *Protocol {
	return &Protocol{
		c:           c,
		id:          id,
		claimSignal: make(chan struct{}),
		xfers:       make(map[uint16]*packetReader),
		readBufSema: newBufSema(c.cfg.readBufferSize()),
		readCond:    sync.NewCond(new(sync.Mutex)),
	}
}

func (p *Protocol) feedPacket(pr *packetReader) {
	p.readCond.L.Lock()
	p.newPackets = append(p.newPackets, pr)
	p.readCond.Signal()
	p.readCond.L.Unlock()
}

func (p *Protocol) readClose(err error) {
	p.readCond.L.Lock()
	p.readErr = err
	p.readCond.Broadcast()
	p.readCond.L.Unlock()
}

// ReadHeader waits for a packet to appear. The content of the packet
// can be read from r as it is received. More packets can be read
// immediately, r does not need to be consumed before the next call.
func (p *Protocol) ReadPacket() (totalSize uint32, r io.Reader, err error) {
	// Lazy handshake.
	if err := p.c.Handshake(); err != nil {
		return 0, nil, err
	}
	// Wait for a packet or error.
	p.readCond.L.Lock()
	defer p.readCond.L.Unlock()
	for len(p.newPackets) == 0 && p.readErr == nil {
		p.readCond.Wait()
	}
	if p.readErr != nil {
		return 0, nil, p.readErr
	}
	pr := p.newPackets[0]
	p.newPackets = p.newPackets[:copy(p.newPackets, p.newPackets[1:])]
	return pr.readN, pr, nil
}

// SendPacket sends len bytes from the payload reader on the connection.
func (p *Protocol) SendPacket(len uint32, payload io.Reader) error {
	if err := p.c.Handshake(); err != nil {
		return err
	}
	if len <= staticFrameSize {
		// The message is small enough and can be sent in a single frame.
		buf := makeFrameWriteBuffer()
		if n, err := io.CopyN(buf, payload, int64(len)); err != nil {
			return fmt.Errorf("read from packet payload failed at pos %d: %v", n, err)
		}
		return p.c.sendFrame(regularHeader{p.id, 0}, buf)
	}
	return p.sendChunked(len, payload)
}

func (p *Protocol) sendChunked(size uint32, payload io.Reader) error {
	contextid := p.nextContextID()
	initial := true
	buf := makeFrameWriteBuffer()
	var rpos int64
	for seq := uint16(0); size > 0; seq++ {
		var header interface{}
		if initial {
			header = chunkStartHeader{p.id, contextid, size}
			initial = false
		} else {
			header = regularHeader{p.id, contextid}
		}

		fsize := staticFrameSize
		if size < fsize {
			fsize = size
		}
		if !initial {
			buf.resetForWrite()
		}
		if n, err := io.CopyN(buf, payload, int64(fsize)); err != nil {
			// The remote end is waiting for the rest of the packet
			// but we can't provide it. Since there is no way to cancel
			// partial transfers, our only option is closing the connection.
			// TODO: close the connection
			return fmt.Errorf("read from packet payload failed at pos %d: %v", rpos+n, err)
		}
		rpos += int64(fsize)
		if err := p.c.sendFrame(header, buf); err != nil {
			return err
		}
		size -= fsize
	}
	return nil
}

// returns the next context ID for a chunked transfer.
// never returns 0, which is reserved for single-frame transfers.
func (p *Protocol) nextContextID() uint16 {
	p.contextidSeq++
	return p.contextidSeq
}

func (c *Conn) sendFrame(header interface{}, body *frameBuffer) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.fd.SetWriteDeadline(time.Now().Add(c.cfg.writeTimeout()))
	return c.rw.sendFrame(header, body)
}
