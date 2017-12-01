package enr

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type Secp256k1 []byte

func (Secp256k1) ENRKey() string {
	return "secp256k1"
}

func (v Secp256k1) EncodeRLP(w io.Writer) error {
	blob := []byte(v)
	return rlp.Encode(w, blob)
}

func (v *Secp256k1) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*[]byte)(v)); err != nil {
		return err
	}
	return nil
}
