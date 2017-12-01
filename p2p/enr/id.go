package enr

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type ID string

func (ID) ENRKey() string {
	return "id"
}

func (v ID) EncodeRLP(w io.Writer) error {
	id := string(v)
	return rlp.Encode(w, id)
}

func (v *ID) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*string)(v)); err != nil {
		return err
	}
	return nil
}
