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
	"bytes"
	"crypto/ecdsa"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/p2p/quic"
	"github.com/ethereum/go-ethereum/rlp"
)

// quicTransport is the transport for connections carried over a QUIC stream.
// It adapts a quic.Conn's framing to Msg with locks and read/write deadlines.
type quicTransport struct {
	rmu, wmu sync.Mutex
	wbuf     bytes.Buffer
	conn     *quic.Conn
	protos   []Protocol // protocols this connection may run
}

func newQUICTransport(conn *quic.Conn, protos []Protocol) transport {
	return &quicTransport{conn: conn, protos: protos}
}

func (q *quicTransport) ReadMsg() (Msg, error) {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	var msg Msg
	q.conn.SetReadDeadline(time.Now().Add(frameReadTimeout))
	code, data, wireSize, err := q.conn.ReadFrame()
	if err == nil {
		// Protocol messages are dispatched to subprotocol handlers asynchronously,
		// but the connection may reuse the returned 'data' buffer on the next call
		// to ReadFrame. Copy the message data to avoid this being an issue.
		data = common.CopyBytes(data)
		msg = Msg{
			ReceivedAt: time.Now(),
			Code:       code,
			Size:       uint32(len(data)),
			meterSize:  uint32(wireSize),
			Payload:    bytes.NewReader(data),
		}
	}
	return msg, err
}

func (q *quicTransport) WriteMsg(msg Msg) error {
	q.wmu.Lock()
	defer q.wmu.Unlock()

	// Copy message data to write buffer.
	q.wbuf.Reset()
	if _, err := io.CopyN(&q.wbuf, msg.Payload, int64(msg.Size)); err != nil {
		return err
	}

	// Write the message.
	q.conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	_, err := q.conn.WriteFrame(msg.Code, q.wbuf.Bytes())
	return err
}

func (q *quicTransport) close(err error) {
	q.wmu.Lock()
	defer q.wmu.Unlock()

	// Tell the remote end why we're disconnecting if possible.
	if reason, ok := err.(DiscReason); ok && reason != DiscNetworkError {
		// We do not use the WriteMsg func since we want a custom deadline.
		deadline := time.Now().Add(discWriteTimeout)
		if err := q.conn.SetWriteDeadline(deadline); err == nil {
			q.wbuf.Reset()
			rlp.Encode(&q.wbuf, []any{reason})
			q.conn.WriteFrame(discMsg, q.wbuf.Bytes())
		}
	}
	q.conn.Close()
}

func (q *quicTransport) doEncHandshake(prv *ecdsa.PrivateKey) (*enode.Node, error) {
	q.conn.SetDeadline(time.Now().Add(handshakeTimeout))
	pubkey, err := q.conn.Handshake(prv, q.chosenCaps())
	if err != nil {
		return nil, err
	}
	if q.conn.Nonce() != nil {
		return browserNodeFromConn(q.conn)
	}
	return nodeFromConn(pubkey, q.conn), nil
}

// chosenCaps matches the capabilities announced in the CONNECT request against
// our protocols, keeping the highest common version of each.
func (q *quicTransport) chosenCaps() []string {
	var caps []Cap
	for _, s := range q.conn.Caps() {
		if cap, ok := parseCap(s); ok {
			caps = append(caps, cap)
		}
	}
	var chosen []string
	for _, proto := range matchProtocols(q.protos, caps, nil) {
		chosen = append(chosen, proto.cap().String())
	}
	slices.Sort(chosen)
	return chosen
}

func (q *quicTransport) doProtoHandshake(_ *protoHandshake) (*protoHandshake, error) {
	// A QUIC connection does not perform the Hello exchange. The peer handshake is
	// assembled from the identity established during the encryption handshake and
	// the capabilities chosen by the server in it. QUIC always uses snappy
	// (version 5).
	q.conn.SetSnappy(true)
	var caps []Cap
	for _, s := range q.conn.Caps() {
		if cap, ok := parseCap(s); ok {
			caps = append(caps, cap)
		}
	}
	return &protoHandshake{Version: baseProtocolVersion, Caps: caps, ID: q.conn.PeerID()}, nil
}

var browserScheme = enr.SchemeMap{
	"browser": browserID{},
}

type browserID struct{}

func (b browserID) NodeAddr(r *enr.Record) []byte {
	var id enode.ID
	r.Load(enr.WithEntry("browserid", &id))
	return id[:]
}

func (b browserID) Verify(r *enr.Record, sig []byte) error {
	return nil
}

// browserNodeFromConn creates the node of a browser peer from its random ID
// and remote IP.
func browserNodeFromConn(conn *quic.Conn) (*enode.Node, error) {
	var r enr.Record
	if ip := netutil.AddrAddr(conn.RemoteAddr()); ip.Is4() {
		r.Set(enr.IPv4Addr(ip))
	} else if ip.IsValid() {
		r.Set(enr.IPv6Addr(ip))
	}
	return signBrowser(&r, enode.ID(crypto.Keccak256Hash(conn.PeerID())))
}

func signBrowser(r *enr.Record, id enode.ID) (*enode.Node, error) {
	r.Set(enr.ID("browser"))
	r.Set(enr.WithEntry("browserid", id))
	if err := r.SetSig(browserID{}, []byte{}); err != nil {
		return nil, err
	}
	return enode.New(browserScheme, r)
}

// isBrowser returns whether the remote node uses the browser identity scheme
func (c *conn) isBrowser() bool {
	var scheme enr.ID
	if c.node != nil {
		c.node.Load(&scheme)
	}
	return scheme == "browser"
}

// browserProtocols returns the subprotocols that browser peers can run
func browserProtocols(protos []Protocol) []Protocol {
	var allowed []Protocol
	for _, p := range protos {
		if p.AllowBrowser {
			allowed = append(allowed, p)
		}
	}
	return allowed
}
