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

// Opcode values are one byte on the wire and split into two ranges:
//
//   - CommandOp: P2P command opcodes, 0x00–0x7F.
//   - ClusterOp: cluster RPC opcodes (master ↔ slave, slave ↔ slave),
//     0x80–0xFF, defined relative to ClusterOpBase.
//
// The opcode set is not finalised yet. This file defines only the
// framework and one example opcode per range. Add opcodes as the
// corresponding protocol messages are implemented, assigning unique
// values within the appropriate range.

// ClusterOpBase is the offset added to all cluster op values on wire.
// Values below ClusterOpBase belong to CommandOp (P2P).
const ClusterOpBase ClusterOp = 128

// ClusterOp  —  cluster RPC opcodes (master ↔ slave, slave ↔ slave)
type ClusterOp byte

const (
	// Cluster connection opcodes.
	ClusterOpPing                    ClusterOp = 1 + ClusterOpBase // 0x81
	ClusterOpPong                    ClusterOp = 2 + ClusterOpBase // 0x82
	ClusterOpConnectToSlavesRequest  ClusterOp = 3 + ClusterOpBase // 0x83
	ClusterOpConnectToSlavesResponse ClusterOp = 4 + ClusterOpBase // 0x84
)

// CommandOp  —  P2P command opcodes (peer ↔ peer, cluster_peer_id != 0)
type CommandOp byte

const (
	// CommandOpHello is the P2P connection handshake opcode.
	CommandOpHello CommandOp = 0x00
)

// =============================================================================
// Opcode range classification
// =============================================================================

// InCommandOpRange reports whether op lies in the P2P command range (< ClusterOpBase).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InCommandOpRange(op byte) bool { return op < byte(ClusterOpBase) }

// InClusterOpRange reports whether op lies in the cluster range (≥ ClusterOpBase).
// This only classifies the range, not validity. Callers must reject unrecognised
// values, e.g. via a default case in opcode switch statements.
func InClusterOpRange(op byte) bool { return op >= byte(ClusterOpBase) }
