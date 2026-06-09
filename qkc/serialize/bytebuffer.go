// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"encoding/binary"
	"fmt"
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

	var size int = 0
	for i := 0; i < byteSize; i++ {
		size = (size << 8) | int(b[i])
	}

	return size, nil
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
