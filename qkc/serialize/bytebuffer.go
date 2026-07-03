// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"encoding/binary"
	"fmt"
	"math"
)

type ByteBuffer struct {
	data     *[]byte
	position int
	size     int
}

func NewByteBuffer(bytes []byte) *ByteBuffer {
	bb := ByteBuffer{&bytes, 0, len(bytes)}
	return &bb
}

func (bb *ByteBuffer) GetOffset() int {
	return bb.position
}

func (bb *ByteBuffer) getBytes(size int) ([]byte, error) {
	if size > bb.size-bb.position {
		return nil, fmt.Errorf("deser: buffer is shorter than expected")
	}

	bytes := (*bb.data)[bb.position : bb.position+size]
	bb.position += size
	return bytes, nil
}

func (bb *ByteBuffer) GetUInt8() (uint8, error) {
	bytes, err := bb.getBytes(1)
	if err != nil {
		return 0, err
	}

	return uint8(bytes[0]), nil
}

func (bb *ByteBuffer) GetUInt16() (uint16, error) {
	bytes, err := bb.getBytes(2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(bytes), nil
}

func (bb *ByteBuffer) GetUInt32() (uint32, error) {
	bytes, err := bb.getBytes(4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(bytes), nil
}

func (bb *ByteBuffer) GetUInt64() (uint64, error) {
	bytes, err := bb.getBytes(8)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(bytes), nil
}

func (bb *ByteBuffer) getLen(byteSize int) (int, error) {
	if byteSize < 1 {
		return 0, fmt.Errorf("deser: bytesize in GetVarBytes should larger than 0")
	}

	b, err := bb.getBytes(byteSize)
	if err != nil {
		return 0, err
	}

	// Accumulate in uint64 and bound the result. Using a signed int with an
	// unbounded left shift let a high length-prefix byte land in the sign bit
	// once the prefix reached the platform int width (4 bytes on 32-bit, 8 on
	// 64-bit), silently yielding a NEGATIVE length. A negative length slips past
	// downstream "len <= remaining" checks and then panics (slice bounds /
	// reflect.MakeSlice: negative len) on attacker-controlled input. Reject any
	// prefix that overflows uint64 or does not fit a non-negative int. Hardening
	// divergence from goquarkchain; it does not change any representable length.
	var size uint64 = 0
	for i := 0; i < byteSize; i++ {
		if size > math.MaxUint64>>8 {
			return 0, fmt.Errorf("deser: length prefix of %d bytes overflows", byteSize)
		}
		size = (size << 8) | uint64(b[i])
	}
	if size > uint64(math.MaxInt) {
		return 0, fmt.Errorf("deser: length prefix %d exceeds maximum %d", size, uint64(math.MaxInt))
	}

	return int(size), nil
}

func (bb *ByteBuffer) GetVarBytes(byteSizeOfSliceLen int) ([]byte, error) {
	size, err := bb.getLen(byteSizeOfSliceLen)
	if err != nil {
		return nil, err
	}

	bs, err := bb.getBytes(size)
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, size, size)
	copy(bytes, bs)
	return bytes, nil
}

func (bb *ByteBuffer) Remaining() int {
	return bb.size - bb.position
}

// ReadRemaining returns all unread bytes in the buffer.
// Used by types that need to consume the rest of the buffer (e.g. opaque
// placeholder types for forward-compatible deserialization).
func (bb *ByteBuffer) ReadRemaining() ([]byte, error) {
	return bb.getBytes(bb.Remaining())
}
