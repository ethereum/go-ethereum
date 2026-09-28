// Copyright 2023 The go-ethereum Authors
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

package ethtest

import (
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

// Unexported devp2p message codes from p2p/peer.go.
const (
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

// Unexported devp2p protocol lengths from p2p package.
const (
	baseProtoLen = 16
	// snapProtoLen accommodates snap/2 (EIP-8189) which extends snap/1 with two
	// additional message codes (GetBlockAccessLists=0x08, BlockAccessLists=0x09).
	// Using 10 is safe for snap/1 connections because the extra codes are simply
	// never used on that protocol version.
	snapProtoLen = 10
)

// ethProtoLens are the message counts of the eth versions the suite speaks,
// mirroring the unexported lengths in eth/protocols/eth. The snap message codes
// start right after the eth range, so they move with the negotiated eth version.
var ethProtoLens = map[uint]uint64{
	eth.ETH69: 18,
	eth.ETH70: 18,
	eth.ETH71: 20,
	eth.ETH72: 22,
}

// Unexported handshake structure from p2p/peer.go.
type protoHandshake struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte
	Rest       []rlp.RawValue `rlp:"tail"`
}

type Hello = protoHandshake

// Proto is an enum representing devp2p protocol types.
type Proto int

const (
	baseProto Proto = iota
	ethProto
	snapProto
)

// getProto returns the protocol a certain message code is associated with
// (assuming the negotiated capabilities are exactly {eth,snap})
func (c *Conn) getProto(code uint64) Proto {
	ethProtoLen := ethProtoLens[c.negotiatedProtoVersion]
	switch {
	case code < baseProtoLen:
		return baseProto
	case code < baseProtoLen+ethProtoLen:
		return ethProto
	case code < baseProtoLen+ethProtoLen+snapProtoLen:
		return snapProto
	default:
		panic("unhandled msg code beyond last protocol")
	}
}

// protoOffset will return the offset at which the specified protocol's messages
// begin.
func (c *Conn) protoOffset(proto Proto) uint64 {
	switch proto {
	case baseProto:
		return 0
	case ethProto:
		return baseProtoLen
	case snapProto:
		return baseProtoLen + ethProtoLens[c.negotiatedProtoVersion]
	default:
		panic("unhandled protocol")
	}
}
