// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package wire

import (
	"fmt"
)

// Direction matches the P2P sync Direction enum in
// quarkchain/cluster/p2p_commands.py.
//
//	DIRECTIONS = [GENESIS, TIP]
//	class Direction(IntEnum):
//	    GENESIS = 0
//	    TIP     = 1
type Direction uint8

const (
	DirectionGenesis Direction = 0
	DirectionTip     Direction = 1
)

func (d Direction) String() string {
	switch d {
	case DirectionGenesis:
		return "GENESIS"
	case DirectionTip:
		return "TIP"
	default:
		return fmt.Sprintf("Direction(%d)", uint8(d))
	}
}

// SSZMessage is implemented by every wire message after sszgen runs.
type SSZMessage interface {
	MarshalSSZ() ([]byte, error)
	UnmarshalSSZ([]byte) error
	SizeSSZ() int
}

// NewClusterMessage returns a new empty message struct pointer corresponding
// to the given ClusterOp. Used by the connection layer on both Master and chain
// sides to allocate the concrete type before decoding a frame payload.
func NewClusterMessage(op ClusterOp) (SSZMessage, error) {
	switch op {
	case ClusterOpPing:
		return &PingRequest{}, nil
	case ClusterOpPong:
		return &PongResponse{}, nil
	case ClusterOpConnectToChainsRequest:
		return &ConnectToChainsRequest{}, nil
	case ClusterOpConnectToChainsResponse:
		return &ConnectToChainsResponse{}, nil

	default:
		return nil, fmt.Errorf("unknown ClusterOp: 0x%x", op)
	}
}

// NewCommandMessage returns a new empty message struct pointer corresponding
// to the given CommandOp.
func NewCommandMessage(op CommandOp) (SSZMessage, error) {
	switch op {
	case CommandOpHello:
		return &HelloCommand{}, nil
	default:
		return nil, fmt.Errorf("unknown CommandOp: 0x%x", op)
	}
}
