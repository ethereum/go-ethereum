// Copyright 2026 The go-ethereum Authors
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

package quic

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/snappy"
	"github.com/quic-go/webtransport-go"
)

const (
	quicMaxMsgSize = (1 << 24) - 1
	quicProofLabel = "ENR-key-proof-v1"
)

var errQUICMsgTooLarge = errors.New("quic: message too large")

// Conn is a devp2p connection carried by a single bidirectional WebTransport
// stream.
type Conn struct {
	*webtransport.Stream
	session *webtransport.Session

	nonce  []byte   // id-proof challenge from CONNECT
	caps   []string // capabilities announced in CONNECT, replaced by the chosen ones in the handshake
	nodeID []byte   // random peer identity assigned by the handshake

	// These are the buffers for snappy compression.
	// Compression is enabled if they are non-nil.
	snappyReadBuffer  []byte
	snappyWriteBuffer []byte
}

var _ net.Conn = (*Conn)(nil)

func (c *Conn) Close() error {
	c.Stream.Close()
	select {
	case <-c.session.Context().Done(): // wait for browser to close the session
	case <-time.After(2 * time.Second):
	}
	return c.session.CloseWithError(0, "")
}
func (c *Conn) LocalAddr() net.Addr  { return c.session.LocalAddr() }
func (c *Conn) RemoteAddr() net.Addr { return c.session.RemoteAddr() }

// NewConn wraps a WebTransport session and its first bidirectional stream.
func NewConn(session *webtransport.Session, str *webtransport.Stream, nonce []byte, caps []string) *Conn {
	return &Conn{Stream: str, session: session, nonce: nonce, caps: caps}
}

func (c *Conn) Caps() []string { return c.caps }

// PeerID returns the random peer identity assigned by the handshake.
func (c *Conn) PeerID() []byte { return c.nodeID }

// Handshake establishes connection over the QUIC stream. This must be called
// before any data is written or read from the connection. caps are the
// capabilities chosen by the server.
func (c *Conn) Handshake(prv *ecdsa.PrivateKey, caps []string) error {
	if c.nonce == nil {
		return errors.New("not supported")
	}
	if err := c.writeProof(prv, c.nonce, caps); err != nil {
		return err
	}
	// assign random node id
	c.nodeID = make([]byte, 64)
	rand.Read(c.nodeID)
	return nil
}

func proofDigest(challenge []byte) []byte {
	return crypto.Keccak256([]byte(quicProofLabel), challenge)
}

// handshake is the first message the server sends on the stream.
type handshake struct {
	Proof []byte
	Caps  []string
}

// writeProof writes `id-proof` and chosen capabilities. Only called when
// serving as a server.
func (c *Conn) writeProof(prv *ecdsa.PrivateKey, challenge []byte, caps []string) error {
	sig, err := crypto.Sign(proofDigest(challenge), prv)
	if err != nil {
		return err
	}
	enc, err := rlp.EncodeToBytes(&handshake{Proof: sig, Caps: caps})
	if err != nil {
		return err
	}
	if _, err := c.Stream.Write(enc); err != nil {
		return err
	}
	c.caps = caps
	return nil
}

// ReadFrame reads one framed devp2p message from the stream.
func (c *Conn) ReadFrame() (uint64, []byte, int, error) {
	var head [4]byte
	if _, err := io.ReadFull(c.Stream, head[:]); err != nil {
		return 0, nil, 0, err
	}

	code := uint64(head[0])
	size := readUint24(head[1:])

	data := make([]byte, size)
	if _, err := io.ReadFull(c.Stream, data); err != nil {
		return 0, nil, 0, err
	}
	wireSize := len(data)

	if c.snappyReadBuffer != nil {
		actualSize, err := snappy.DecodedLen(data)
		if err != nil {
			return 0, nil, 0, err
		}
		if actualSize > quicMaxMsgSize {
			return 0, nil, 0, errQUICMsgTooLarge
		}
		c.snappyReadBuffer = growslice(c.snappyReadBuffer, actualSize)
		if data, err = snappy.Decode(c.snappyReadBuffer, data); err != nil {
			return 0, nil, 0, err
		}
	}
	return code, data, wireSize, nil
}

// WriteFrame writes one framed devp2p message to the stream.
func (c *Conn) WriteFrame(code uint64, data []byte) (uint32, error) {
	if code > math.MaxUint8 {
		return 0, fmt.Errorf("quic: invalid code size")
	}
	if len(data) > quicMaxMsgSize {
		return 0, errQUICMsgTooLarge
	}
	if c.snappyWriteBuffer != nil {
		// Ensure the buffer has sufficient size.
		// Package snappy will allocate its own buffer if the provided
		// one is smaller than MaxEncodedLen.
		c.snappyWriteBuffer = growslice(c.snappyWriteBuffer, snappy.MaxEncodedLen(len(data)))
		data = snappy.Encode(c.snappyWriteBuffer, data)
		// Compression can expand incompressible data past the limit.
		if len(data) > quicMaxMsgSize {
			return 0, errQUICMsgTooLarge
		}
	}
	var head [4]byte
	head[0] = byte(code)
	putUint24(uint32(len(data)), head[1:])
	if _, err := c.Stream.Write(head[:]); err != nil {
		return 0, err
	}
	if _, err := c.Stream.Write(data); err != nil {
		return 0, err
	}
	return uint32(len(data)), nil
}

// SnappyEnabled reports whether snappy compression is active.
func (c *Conn) SnappyEnabled() bool { return c.snappyReadBuffer != nil }

func (c *Conn) SetSnappy(snappy bool) {
	if snappy {
		c.snappyReadBuffer = []byte{}
		c.snappyWriteBuffer = []byte{}
	} else {
		c.snappyReadBuffer = nil
		c.snappyWriteBuffer = nil
	}
}

// todo below code is copied from buffer.go
func readUint24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putUint24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

// growslice ensures b has the wanted length by either expanding it to its capacity
// or allocating a new slice if b has insufficient capacity.
func growslice(b []byte, wantLength int) []byte {
	if len(b) >= wantLength {
		return b
	}
	if cap(b) >= wantLength {
		return b[:cap(b)]
	}
	return make([]byte, wantLength)
}
