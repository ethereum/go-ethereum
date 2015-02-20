package types

import "math/big"

type BlockProcessor interface {
	Process(*Block) (*big.Int, error)
}
