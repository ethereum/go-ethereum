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

package p2p

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/snappy"
	"github.com/quic-go/webtransport-go"
)

const (
	quicMaxMsgSize = (1 << 24) - 1
	quicProofLabel = "ENR-key-proof-v1"
	quicNonceLen   = 32
)

var errQUICMsgTooLarge = errors.New("quic: message too large")

// quicWire is a wireConn carried by a single bidirectional QUIC stream.
// Encryption is provided by TLS, so messages are framed in plaintext as
// [2B code (BE)] [3B len (BE)] [payload].
type quicWire struct {
	fd       net.Conn
	session  *webtransport.Session
	dialDest *ecdsa.PublicKey
	nonce    []byte // nonce from CONNECT

	// nodeID is a random identity minted for an unverified inbound peer.
	nodeID []byte

	// These are the buffers for snappy compression.
	// Compression is enabled if they are non-nil.
	snappyReadBuffer  []byte
	snappyWriteBuffer []byte
}

var _ wireConn = (*quicWire)(nil)

func newQUICTransport(fd net.Conn, session *webtransport.Session, nonce []byte, dialDest *ecdsa.PublicKey) transport {
	return &wireTransport{conn: &quicWire{fd: fd, session: session, nonce: nonce, dialDest: dialDest}}
}

// overrideID replaces the peer's self-reported protocol handshake ID with the
// random identity minted for an unverified inbound peer.
func (w *quicWire) overrideID() []byte { return w.nodeID }

// Handshake establishes node identity. The dialer sends a nonce in the CONNECT
// request; the server proves ownership of its node key by signing that nonce,
// which the dialer verifies against the ENR it dialed. Mutual authentication is
// not yet implemented, so the server does not verify the dialer and instead
// assigns it a random identity rather than trusting a self-reported one.
func (w *quicWire) Handshake(prv *ecdsa.PrivateKey) (*ecdsa.PublicKey, error) {
	if w.dialDest == nil {
		// inbound: prove our identity, then assign the peer a random one.
		if err := w.writeProof(prv); err != nil {
			return nil, err
		}
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		w.nodeID = crypto.FromECDSAPub(&key.PublicKey)[1:]
		return &key.PublicKey, nil
	}
	// dialed: verify the server's proof over the nonce we sent.
	remote, err := w.readProof()
	if err != nil {
		return nil, err
	}
	if !remote.Equal(w.dialDest) {
		return nil, errors.New("quic: node identity mismatch")
	}
	return remote, nil
}

func (w *quicWire) proofDigest() []byte {
	return crypto.Keccak256([]byte(quicProofLabel), w.nonce)
}

func (w *quicWire) writeProof(prv *ecdsa.PrivateKey) error {
	sig, err := crypto.Sign(w.proofDigest(), prv)
	if err != nil {
		return err
	}
	_, err = w.fd.Write(sig)
	return err
}

func (w *quicWire) readProof() (*ecdsa.PublicKey, error) {
	sig := make([]byte, crypto.SignatureLength)
	if _, err := io.ReadFull(w.fd, sig); err != nil {
		return nil, err
	}
	return crypto.SigToPub(w.proofDigest(), sig)
}

func (w *quicWire) Read() (uint64, []byte, int, error) {
	var head [5]byte
	if _, err := io.ReadFull(w.fd, head[:]); err != nil {
		return 0, nil, 0, err
	}

	code := uint64(binary.BigEndian.Uint16(head[:2]))
	size := readUint24(head[2:])

	data := make([]byte, size)
	if _, err := io.ReadFull(w.fd, data); err != nil {
		return 0, nil, 0, err
	}
	wireSize := len(data)

	if w.snappyReadBuffer != nil {
		actualSize, err := snappy.DecodedLen(data)
		if err != nil {
			return 0, nil, 0, err
		}
		if actualSize > quicMaxMsgSize {
			return 0, nil, 0, errQUICMsgTooLarge
		}
		w.snappyReadBuffer = growslice(w.snappyReadBuffer, actualSize)
		if data, err = snappy.Decode(w.snappyReadBuffer, data); err != nil {
			return 0, nil, 0, err
		}
	}
	return code, data, wireSize, nil
}

func (w *quicWire) Write(code uint64, data []byte) (uint32, error) {
	if code > math.MaxUint16 {
		return 0, fmt.Errorf("quic: invalid message code %d", code)
	}
	if len(data) > quicMaxMsgSize {
		return 0, errQUICMsgTooLarge
	}
	if w.snappyWriteBuffer != nil {
		// Ensure the buffer has sufficient size.
		// Package snappy will allocate its own buffer if the provided
		// one is smaller than MaxEncodedLen.
		w.snappyWriteBuffer = growslice(w.snappyWriteBuffer, snappy.MaxEncodedLen(len(data)))
		data = snappy.Encode(w.snappyWriteBuffer, data)
		// Compression can expand incompressible data past the limit.
		if len(data) > quicMaxMsgSize {
			return 0, errQUICMsgTooLarge
		}
	}
	var head [5]byte
	binary.BigEndian.PutUint16(head[:2], uint16(code))
	putUint24(uint32(len(data)), head[2:])
	if _, err := w.fd.Write(head[:]); err != nil {
		return 0, err
	}
	if _, err := w.fd.Write(data); err != nil {
		return 0, err
	}
	return uint32(len(data)), nil
}

func (w *quicWire) SetSnappy(snappy bool) {
	if snappy {
		w.snappyReadBuffer = []byte{}
		w.snappyWriteBuffer = []byte{}
	} else {
		w.snappyReadBuffer = nil
		w.snappyWriteBuffer = nil
	}
}

func (w *quicWire) SetReadDeadline(t time.Time) error {
	return w.fd.SetReadDeadline(t)
}

func (w *quicWire) SetWriteDeadline(t time.Time) error {
	return w.fd.SetWriteDeadline(t)
}

func (w *quicWire) SetDeadline(t time.Time) error {
	return w.fd.SetDeadline(t)
}

func (w *quicWire) Close() error {
	return w.session.CloseWithError(0, "")
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
