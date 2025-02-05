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
	ethProtoLen  = 17
	snapProtoLen = 8
)

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
func getProto(code uint64) Proto {
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
func protoOffset(proto Proto) uint64 {
	switch proto {
	case baseProto:
		return 0
	case ethProto:
		return baseProtoLen
	case snapProto:
		return baseProtoLen + ethProtoLen
	default:
		panic("unhandled protocol")
	}
}
