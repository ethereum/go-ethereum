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

package p2p

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// devp2p message codes
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
	getPeersMsg  = 0x04
	peersMsg     = 0x05
)

const (
	baseProtocolVersion    = 4
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 2 * 1024
)

var errMsgTooBig = errors.New("encoded message size exceeds uint32")

// DiscReason indicates why a connection is being disconnected.
type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscSubprotocolError = 0x10
)

var discReasonToString = [...]string{
	DiscRequested:           "Disconnect requested",
	DiscNetworkError:        "Network error",
	DiscProtocolError:       "Breach of protocol",
	DiscUselessPeer:         "Useless peer",
	DiscTooManyPeers:        "Too many peers",
	DiscAlreadyConnected:    "Already connected",
	DiscIncompatibleVersion: "Incompatible P2P protocol version",
	DiscInvalidIdentity:     "Invalid node identity",
	DiscQuitting:            "Client quitting",
	DiscUnexpectedIdentity:  "Unexpected identity",
	DiscSelf:                "Connected to self",
	DiscReadTimeout:         "Read timeout",
	DiscSubprotocolError:    "Subprotocol error",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return fmt.Sprintf("Unknown Reason(%d)", d)
	}
	return discReasonToString[d]
}

func (d DiscReason) Error() string {
	return d.String()
}

func discReasonForError(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	peerError, ok := err.(*peerError)
	if ok {
		switch peerError.code {
		case errInvalidMsgCode, errInvalidMsg:
			return DiscProtocolError
		default:
			return DiscSubprotocolError
		}
	}
	return DiscSubprotocolError
}

// protoHandshake is the RLP structure of the protocol handshake.
type protoHandshake struct {
	Version    uint64
	Name       string
	Caps       []Cap
	ListenPort uint64
	ID         discover.NodeID
}

// devConn implements the devp2p the messaging layer atop RLPx.
type devConn struct {
	*rlpx.Conn
	// contains negotiated protocol sessions.
	// protocol zero is pre-negotiated and carries the
	// built-in devp2p packets.
	protocols []*devProtocol
}

// devProtocol represents a running subprotocol.
type devProtocol struct {
	p *rlpx.Protocol
}

func newDevConn(fd net.Conn, key *ecdsa.PrivateKey, remote *ecdsa.PublicKey) *devConn {
	c := new(devConn)
	if remote == nil {
		c.Conn = rlpx.Server(fd, &rlpx.Config{Key: key})
	} else {
		c.Conn = rlpx.Client(fd, remote, &rlpx.Config{Key: key})
	}
	c.protocols = []*devProtocol{{c.Conn.Protocol(0)}}
	return c
}

func (t *devConn) addProtocols(n int) {
	for i := 0; i < n; i++ {
		p := t.Conn.Protocol(uint16(len(t.protocols)))
		t.protocols = append(t.protocols, &devProtocol{p})
	}
}

// protoHandshake negotiates RLPx subprotocols.
// the protocol handshake is the first authenticated message
// and also verifies whether the RLPx encryption handshake 'worked' and the
// remote side actually provided the right public key.
func (t *devConn) doProtoHandshake(our *protoHandshake) (their *protoHandshake, err error) {
	// Writing our handshake happens concurrently, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	go func() { werr <- Send(t.protocols[0], handshakeMsg, our) }()
	if their, err = readProtocolHandshake(t.protocols[0], our); err != nil {
		<-werr // make sure the write terminates too
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	return their, nil
}

func readProtocolHandshake(rw MsgReader, our *protoHandshake) (*protoHandshake, error) {
	msg, err := rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Size > baseProtocolMaxMsgSize {
		return nil, fmt.Errorf("message too big")
	}
	if msg.Code == discMsg {
		// Disconnect before protocol handshake is valid according to the
		// spec and we send it ourself if the posthanshake checks fail.
		// We can't return the reason directly, though, because it is echoed
		// back otherwise. Wrap it in a string instead.
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}
	var hs protoHandshake
	if err := msg.Decode(&hs); err != nil {
		return nil, err
	}
	// validate handshake info
	if hs.Version != our.Version {
		return nil, DiscIncompatibleVersion
	}
	if (hs.ID == discover.NodeID{}) {
		return nil, DiscInvalidIdentity
	}
	return &hs, nil
}

func (t *devConn) close(err error) {
	// Tell the remote end why we're disconnecting if possible.
	// TODO: if t.DidHandshake()
	if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
		SendItems(t.protocols[0], discMsg, r)
	}
	t.Close()
}

func (p *devProtocol) WriteMsg(msg Msg) error {
	codelen, code, _ := rlp.EncodeToReader(msg.Code)
	if msg.Size > math.MaxUint32-uint32(codelen) {
		return errMsgTooBig
	}
	plen := msg.Size + uint32(codelen)
	return p.p.SendPacket(plen, io.MultiReader(code, msg.Payload))
}

func (p *devProtocol) ReadMsg() (msg Msg, err error) {
	len, r, err := p.p.ReadPacket()
	if err != nil {
		return msg, err
	}
	// Parse the message code, which is prepended to the protocol payload.
	// r must be recognized as buffered by package rlp to prevent it from
	// reading into the payload. The interface assertion ensures that it is.
	// The input limit is 9, which is as large as an encoded uint64 can get.
	s := rlp.NewStream(r.(rlp.ByteReader), 9)
	if err := s.Decode(&msg.Code); err != nil {
		return msg, err
	}
	// Remaining data in r belongs to the protocol.
	msg.Payload = r
	msg.Size = len - uint32(rlp.IntSize(msg.Code))
	msg.ReceivedAt = time.Now()
	return msg, nil
}
