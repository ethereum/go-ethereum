package api

import (
	"context"
	"math/big"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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

	res0, err := c.NewSession(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, res0, "1")

	txn := types.NewTransaction(0, common.Address{}, big.NewInt(1), 1, big.NewInt(1), []byte{})
	_, err = c.AddTransaction(context.Background(), "1", txn)
	require.NoError(t, err)

	_, err = c.GetBalance(context.Background(), "1", common.Address{})
	require.NoError(t, err)

	_, err = c.AddTransactions(context.Background(), "1", []*types.Transaction{txn})
	require.NoError(t, err)

	bundle := &Bundle{
		Txs: []*types.Transaction{txn},
	}
	_, err = c.AddBundles(context.Background(), "1", []*Bundle{bundle})
	require.NoError(t, err)
}

type nullSessionManager struct{}

func (nullSessionManager) NewSession(ctx context.Context, args *BuildBlockArgs) (string, error) {
	return "1", ctx.Err()
}

func (nullSessionManager) AddTransaction(sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error) {
	return &SimulateTransactionResult{Logs: []*SimulatedLog{}}, nil
}

func (nullSessionManager) AddTransactions(sessionId string, txs types.Transactions) ([]*SimulateTransactionResult, error) {
	return nil, nil
}

func (nullSessionManager) AddBundles(sessionId string, bundles []*Bundle) ([]*SimulateBundleResult, error) {
	return nil, nil
}

func (nullSessionManager) BuildBlock(sessionId string) error {
	return nil
}

func (nullSessionManager) Bid(sessioId string, blsPubKey phase0.BLSPubKey) (*SubmitBlockRequest, error) {
	return nil, nil
}

func (nullSessionManager) GetBalance(sessionId string, addr common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}
