// Copyright 2026-2027, QuarkChain.

// =============================================================================
// WIRE MIGRATION SHIM (NOT PART OF PROTOCOL SPEC)
// =============================================================================
//
// This file exists solely to support incremental migration from Python
// QuarkChain Serializable types to Go native structs.
//
// It is an IMPLEMENTATION-ONLY COMPATIBILITY LAYER.
//
// -----------------------------------------------------------------------------
// IMPORTANT DISTINCTION
// -----------------------------------------------------------------------------
//
// This file is NOT part of the wire protocol specification.
//
// The actual protocol contract is defined in package wire message structs.
// RawBytes is only a temporary bridge for unported complex types.
//
// -----------------------------------------------------------------------------
// Migration Strategy
// -----------------------------------------------------------------------------
//
// Many Python-side Serializable types (e.g. RootBlock, MinorBlockHeader,
// TypedTransaction, CrossShardTransactionList, TokenBalanceMap, etc.)
// have not yet been ported to Go.
//
// During migration, these types are represented as:
//
//	*RawBytes
//
// This allows:
//   - wire format to remain stable
//   - incremental type replacement
//   - independent migration of each message type
//
// -----------------------------------------------------------------------------
// RawBytes Semantics
// -----------------------------------------------------------------------------
//
// RawBytes is a bounded-length passthrough placeholder.
//
// It represents an opaque byte segment whose internal structure is defined
// by the Python FIELDS schema but is not yet implemented in Go.
//
// Wire behavior:
//   - Serialize: writes 4-byte length prefix + raw bytes (matches Python PrependedSizeBytesSerializer(4))
//   - Deserialize: reads 4-byte length prefix + corresponding bytes
//
// This design allows RawBytes to be used in ANY struct position (not just last field).
//
// -----------------------------------------------------------------------------
// SAFETY CONSTRAINTS
// -----------------------------------------------------------------------------
//
// RawBytes MUST obey the following rules:
//
//  1. MUST NOT be partially decoded or inspected
//  2. MUST NOT be used in stable protocol definitions
//  3. MUST be removed once real Go types are introduced
//
// Any violation of these rules results in undefined wire behavior.
//
// -----------------------------------------------------------------------------
// Lifecycle
// -----------------------------------------------------------------------------
//
// This file is TEMPORARY and will be removed after full migration.
//
// Migration completion steps:
//  1. Replace all *RawBytes fields with concrete types
//  2. Verify wire compatibility via round-trip tests
//  3. Delete this file entirely
//
// -----------------------------------------------------------------------------
// WARNING
// -----------------------------------------------------------------------------
//
// This file is NOT production protocol logic.
// It is a migration tool and must be treated as unstable internal code.
package wire

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// RawBytes is a transparent byte passthrough placeholder for unported complex types.
type RawBytes []byte

func (r RawBytes) Serialize(w *[]byte) error {
	const maxRawBytesSize = 100 * 1024 * 1024 // 100 MB
	if len(r) > maxRawBytesSize {
		return fmt.Errorf("RawBytes.Serialize: size %d exceeds max %d", len(r), maxRawBytesSize)
	}

	// Write 4-byte length prefix (matches Python PrependedSizeBytesSerializer(4))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(r)))
	*w = append(*w, lenBuf...)
	*w = append(*w, r...)
	return nil
}

// Deserialize reads 4-byte length prefix and corresponding bytes.
//
// TEMPORARY: Delete this placeholder once real types (RootBlock, MinorBlockHeader) are ported.
func (r *RawBytes) Deserialize(bb *serialize.ByteBuffer) error {
	const maxRawBytesSize = 100 * 1024 * 1024 // 100 MB, matches Serialize

	// Read 4-byte length prefix
	length, err := bb.GetUInt32()
	if err != nil {
		return err
	}

	if length > maxRawBytesSize {
		return fmt.Errorf("RawBytes.Deserialize: length %d exceeds max %d", length, maxRawBytesSize)
	}

	// Read the actual bytes
	bytes, err := bb.ReadBytes(int(length))
	if err != nil {
		return err
	}

	*r = RawBytes(bytes)
	return nil
}

// Compile-time check: RawBytes implements Serializable (required by serialize package).
// This ensures serialize.Serialize(&buf, &struct{Field *RawBytes}) works.
var _ serialize.Serializable = (*RawBytes)(nil)
