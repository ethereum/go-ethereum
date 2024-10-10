package health

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

type ethClient interface {
	PeerCount(context.Context) (uint64, error)
	BlockByNumber(context.Context, *big.Int) (*types.Block, error)
	SyncProgress(context.Context) (*ethereum.SyncProgress, error)
}
