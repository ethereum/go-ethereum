package api

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	srv := rpc.NewServer()

	builderAPI := NewServer(&nullSessionManager{})
	srv.RegisterName("suavex", builderAPI)

	c := NewClientFromRPC(rpc.DialInProc(srv))

	res0, err := c.NewSession(context.Background())
	require.NoError(t, err)
	require.Equal(t, res0, "1")

	txn := types.NewTransaction(0, common.Address{}, big.NewInt(1), 1, big.NewInt(1), []byte{})
	_, err = c.AddTransaction(context.Background(), "1", txn)
	require.NoError(t, err)
}

type nullSessionManager struct{}

func (nullSessionManager) NewSession(ctx context.Context) (string, error) {
	return "1", ctx.Err()
}

func (nullSessionManager) AddTransaction(sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	return &types.SimulateTransactionResult{Logs: []*types.SimulatedLog{}}, nil
}
