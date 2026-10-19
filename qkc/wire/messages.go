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

// Package wire defines the SSZ container types for wire messages. See README.md
// for the schema rules and the repo-root command for regenerating their codecs.
package wire

// The go:generate directive below runs from this package directory; it cds to
// the repository root so the generator paths are stable and can be copied
// into documentation.
//go:generate sh -c "cd ../.. && go run github.com/ferranbt/fastssz/sszgen --path qkc/wire/messages.go --output qkc/wire/messages_ssz.go"

// =============================================================================
// Wire-level constants
// =============================================================================
//
// These match the fixed-width fields in the message schemas.

// HashLength is the byte length of a hash256.
const HashLength = 32

// SignatureLength is the byte length of a signature65.
const SignatureLength = 65

// UInt128Length is the byte length of a uint128.
const UInt128Length = 16

// =============================================================================
// §1  Cluster initialisation
// =============================================================================

// PingRequest (ClusterOp.PING, 0x81) — sent by master to initialise a slave.
type PingRequest struct {
	// For []byte, ssz-max is the maximum byte length; for []T, it is the
	// maximum element count. These broad schema limits do not set frame caps.
	ID              []byte   `ssz-max:"4294967295"`
	FullShardIDList []uint32 `ssz-max:"4294967295"`
	HasRootTip      bool
	RootTip         []byte `ssz-max:"4294967295"` // SSZ encoding of RootBlock when HasRootTip is true.
}

// PongResponse (ClusterOp.PONG, 0x82) — slave's reply to PING.
type PongResponse struct {
	ID              []byte   `ssz-max:"4294967295"`
	FullShardIDList []uint32 `ssz-max:"4294967295"`
}

// SlaveInfo (used by ConnectToSlavesRequest) — describes a remote slave.
type SlaveInfo struct {
	ID              []byte `ssz-max:"4294967295"`
	Host            []byte `ssz-max:"4294967295"`
	Port            uint16
	FullShardIDList []uint32 `ssz-max:"4294967295"`
}

// ConnectToSlavesRequest (ClusterOp.CONNECT_TO_SLAVES_REQUEST, 0x83).
type ConnectToSlavesRequest struct {
	SlaveInfoList []SlaveInfo `ssz-max:"4294967295"`
}

// ConnectToSlavesResponse (ClusterOp.CONNECT_TO_SLAVES_RESPONSE, 0x84).
// Each ResultList entry is empty on success or contains the error payload.
type ConnectToSlavesResponse struct {
	ResultList [][]byte `ssz-max:"4294967295,4294967295" ssz-size:"?,?"`
}

// HelloCommand (CommandOp.HELLO, 0x00) — initial inter-cluster handshake.
type HelloCommand struct {
	Version              uint32
	NetworkID            uint32
	PeerID               [HashLength]byte
	PeerIP               [UInt128Length]byte
	PeerPort             uint16
	ChainMaskList        []uint32 `ssz-max:"4294967295"`
	RootBlockHeader      []byte   `ssz-max:"4294967295"` // SSZ encoding of RootBlockHeader.
	GenesisRootBlockHash [HashLength]byte
}
