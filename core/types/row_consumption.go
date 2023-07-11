package types

import (
	"fmt"
	"io"

	"github.com/scroll-tech/go-ethereum/rlp"
)

type RowConsumption struct {
	Rows uint64
}

func (rc *RowConsumption) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rc.Rows)
}

func (rc *RowConsumption) DecodeRLP(s *rlp.Stream) error {
	_, size, err := s.Kind()
	if err != nil {
		return err
	}
	if size <= 8 {
		return s.Decode(&rc.Rows)
	} else {
		return fmt.Errorf("invalid input size %d for origin", size)
	}
}
