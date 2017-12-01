package enr

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type DiscV5 uint32

func (DiscV5) ENRKey() string {
	return "discv5"
}

func (v DiscV5) EncodeRLP(w io.Writer) error {
	port := uint32(v)
	return rlp.Encode(w, port)
}

func (v *DiscV5) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*uint32)(v)); err != nil {
		return err
	}
	return nil
}
