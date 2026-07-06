// Copyright 2026-2027, QuarkChain.

package wire

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// =============================================================================
// Custom wire types for Python format compatibility
// =============================================================================
//
// Python uses 4-byte length prefix for nested slices in some wire messages.
// Go's serialize framework defaults to 1-byte prefix for nested elements.
// These custom types enforce 4-byte prefix to match Python wire format.

// PrependedSizeBytes4 is []byte with 4-byte length prefix (matches Python PrependedSizeBytesSerializer(4)).
type PrependedSizeBytes4 []byte

func (p PrependedSizeBytes4) Serialize(w *[]byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(p)))
	*w = append(*w, lenBuf...)
	*w = append(*w, p...)
	return nil
}

func (p *PrependedSizeBytes4) Deserialize(bb *serialize.ByteBuffer) error {
	length, err := bb.GetUInt32()
	if err != nil {
		return err
	}

	if length > math.MaxInt32 || int(length) > bb.Remaining() {
		return fmt.Errorf("PrependedSizeBytes4.Deserialize: length %d exceeds remaining %d", length, bb.Remaining())
	}

	bytes, err := bb.ReadBytes(int(length))
	if err != nil {
		return err
	}

	*p = PrependedSizeBytes4(bytes)
	return nil
}

var _ serialize.Serializable = (*PrependedSizeBytes4)(nil)

// PrependedSizeHashList4 is [][HashLength]byte with 4-byte length prefix (matches Python PrependedSizeListSerializer(4, hash256)).
type PrependedSizeHashList4 [][HashLength]byte

func (p PrependedSizeHashList4) Serialize(w *[]byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(p)))
	*w = append(*w, lenBuf...)

	for _, hash := range p {
		*w = append(*w, hash[:]...)
	}
	return nil
}

func (p *PrependedSizeHashList4) Deserialize(bb *serialize.ByteBuffer) error {
	length, err := bb.GetUInt32()
	if err != nil {
		return err
	}

	if length > math.MaxInt32/HashLength || int(length)*HashLength > bb.Remaining() {
		return fmt.Errorf("PrependedSizeHashList4.Deserialize: length %d exceeds capacity", length)
	}

	list := make([][HashLength]byte, length)
	for i := 0; i < int(length); i++ {
		hashBytes, err := bb.ReadBytes(HashLength)
		if err != nil {
			return err
		}
		list[i] = [HashLength]byte(hashBytes)
	}

	*p = PrependedSizeHashList4(list)
	return nil
}

var _ serialize.Serializable = (*PrependedSizeHashList4)(nil)
