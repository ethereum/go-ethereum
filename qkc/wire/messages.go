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

// PingRequest (ClusterOp.PING, 0x81) initializes a chain connection.
type PingRequest struct {
	ID              []byte   `ssz-max:"4294967295"`
	FullShardIDList []uint32 `ssz-max:"4294967295"`
	HasRootTip      bool
	RootTip         []byte `ssz-max:"4294967295"` // SSZ encoding of RootBlock when HasRootTip is true.
}

// PongResponse (ClusterOp.PONG, 0x82) is the chain's response to PING and
// echoes the request RPCID in the frame header.
type PongResponse struct {
	ID              []byte   `ssz-max:"4294967295"`
	FullShardIDList []uint32 `ssz-max:"4294967295"`
}

// ChainInfo describes a remote chain for ConnectToChainsRequest.
type ChainInfo struct {
	ID              []byte `ssz-max:"4294967295"`
	Host            []byte `ssz-max:"4294967295"`
	Port            uint16
	FullShardIDList []uint32 `ssz-max:"4294967295"`
}

// ConnectToChainsRequest (opcode 0x83) asks a chain to connect to the listed
// chains and report each result.
type ConnectToChainsRequest struct {
	ChainInfoList []ChainInfo `ssz-max:"4294967295"`
}

// ConnectToChainsResponse (opcode 0x84) reports one result per requested
// chain, in request order. Empty means success; non-empty contains the error
// payload.
type ConnectToChainsResponse struct {
	ResultList [][]byte `ssz-max:"4294967295,4294967295" ssz-size:"?,?"`
}

// HelloCommand (CommandOp.HELLO, 0x00) is the non-RPC Master↔Master handshake.
type HelloCommand struct {
	Version              uint32
	NetworkID            uint32
	PeerID               [HashLength]byte
	PeerIP               [UInt128Length]byte
	PeerPort             uint16
	ChainMaskList        []uint32 `ssz-max:"4294967295"` // Chain masks hosted by this Master.
	RootBlockHeader      []byte   `ssz-max:"4294967295"` // SSZ encoding of RootBlockHeader.
	GenesisRootBlockHash [HashLength]byte
}
