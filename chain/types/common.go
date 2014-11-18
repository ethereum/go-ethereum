package types

import (
	"math/big"
	"github.com/ethereum/go-ethereum/state"
)

type BlockProcessor interface {
	ProcessWithParent(*Block, *Block) (*big.Int, state.Messages, error)
}
