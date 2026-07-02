package types

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type Uint32 uint32

const (
	rlpUint32Prefix = byte(0x84) // RLP "string of length 4"
	rlpUint32Len    = 5
)

func (u *Uint32) GetValue() uint32 { return uint32(*u) }

func (u *Uint32) EncodeRLP(w io.Writer) error {
	b := [rlpUint32Len]byte{rlpUint32Prefix}
	binary.BigEndian.PutUint32(b[1:], uint32(*u))
	_, err := w.Write(b[:])
	return err
}

func (u *Uint32) DecodeRLP(s *rlp.Stream) error {
	data, err := s.Raw()
	if err != nil {
		return err
	}
	if len(data) != rlpUint32Len {
		return fmt.Errorf("Uint32 RLP: expected %d bytes, got %d", rlpUint32Len, len(data))
	}
	if data[0] != rlpUint32Prefix {
		return fmt.Errorf("Uint32 RLP: expected prefix 0x%02x, got 0x%02x", rlpUint32Prefix, data[0])
	}
	*u = Uint32(binary.BigEndian.Uint32(data[1:]))
	return nil
}
