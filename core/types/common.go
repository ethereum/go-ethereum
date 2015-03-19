package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/state"
)

type BlockProcessor interface {
	Process(*Block) (*big.Int, state.Logs, error)
}
