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

import "github.com/ethereum/go-ethereum/common"

// The go:generate directive below runs from this package directory; it cds to
// the repository root so the generator paths are stable and can be copied
// into documentation.
//go:generate sh -c "cd ../.. && go run github.com/ferranbt/fastssz/sszgen --path qkc/wire/messages.go --output qkc/wire/messages_ssz.go --include common/types.go"

// =============================================================================
// Wire-level constants
// =============================================================================
//
// These match the fixed-width fields in the message schemas.

// SignatureLength is the byte length of a signature65.
const SignatureLength = 65

// UInt128Length is the byte length of a uint128.
const UInt128Length = 16

// UInt256Length is the byte length of a uint256; numeric fields use little-endian.
const UInt256Length = 32

// Data carries an already encoded object unchanged through SSZ: RLP for geth
// objects, SSZ for QKC root objects.
type Data []byte

// =============================================================================
// §1  Cluster initialisation
// =============================================================================

// TODO(ssz-limits): The 4294967295 values below are broad numeric bounds
// supplied to fastssz for variable-length fields, not intended 4 GB message
// sizes. Actual message size is primarily constrained by the per-frame size
// limit. Revisit these bounds if QKC defines protocol-specific field limits.

// PingRequest (ClusterOp.PING, 0x81) initializes a chain connection.
// Chain↔chain PINGs use HasRootTip=false. pyquarkchain sent a dummy root block
// on these links, but the receiver ignored it.
type PingRequest struct {
	// ChainID identifies the dialing chain on chain↔chain links, whose frames
	// have no metadata. Master→chain handlers ignore it; zero is a valid chain
	// ID and does not mean root.
	ChainID    uint32
	HasRootTip bool
	RootTip    Data `ssz-max:"4294967295"` // SSZ-encoded RootBlock when HasRootTip is true.
}

// PongResponse (ClusterOp.PONG, 0x82) is the chain's response to PING and
// echoes the request RPCID in the frame header.
type PongResponse struct {
	ChainID uint32
}

// ChainInfo describes a remote chain for ConnectToChainsRequest.
type ChainInfo struct {
	Host    []byte `ssz-max:"4294967295"`
	Port    uint16
	ChainID uint32
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
	PeerID               common.Hash
	PeerIP               [UInt128Length]byte
	PeerPort             uint16
	RootBlockHeader      Data `ssz-max:"4294967295"` // SSZ-encoded RootBlockHeader.
	GenesisRootBlockHash common.Hash
}
