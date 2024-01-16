package api

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

type API interface {
	NewSession(ctx context.Context) (string, error)
	AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error)
}
