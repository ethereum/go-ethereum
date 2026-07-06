// Copyright 2026-2027, QuarkChain.

// =============================================================================
// TEMPORARY PLACEHOLDER FILE — DELETE after real types merge
// =============================================================================
//
// RawBytes is a placeholder used during pyquarkchain → Go migration.
// Delete this file once real types (RootBlock, MinorBlockHeader, etc.) are ported.
// Replace `*RawBytes` fields in messages.go with real typed pointers.
//
// DO NOT REVIEW AS PRODUCTION CODE.
package wire

import (
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

	*w = append(*w, r...)
	return nil
}

// Deserialize consumes all remaining bytes from the buffer.
//
// SAFETY: This is only safe when RawBytes is the LAST field in its parent
// struct.  If RawBytes appears before other fields, consuming the remaining
// bytes will corrupt subsequent fields.  Structs with non-last RawBytes fields
// are marked with a WARNING in messages.go and cannot be correctly deserialized
// until the real Go type is ported.
//
// TEMPORARY: Delete this placeholder once real types (RootBlock, MinorBlockHeader) are ported.
func (r *RawBytes) Deserialize(bb *serialize.ByteBuffer) error {
	const maxRawBytesSize = 100 * 1024 * 1024 // 100 MB, matches Serialize
	if bb.Remaining() > maxRawBytesSize {
		return fmt.Errorf("RawBytes.Deserialize: size %d exceeds max %d", bb.Remaining(), maxRawBytesSize)
	}
	bytes, err := bb.ReadRemaining()
	if err != nil {
		return err
	}
	*r = RawBytes(bytes)
	return nil
}

// Compile-time check: RawBytes implements Serializable (required by serialize package).
// This ensures serialize.Serialize(&buf, &struct{Field *RawBytes}) works.
var _ serialize.Serializable = (*RawBytes)(nil)
