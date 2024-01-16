package backends

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

func TestEthBackend_Compatibility(t *testing.T) {
	// This test ensures that the client is able to call to the server.
	// It does not cover the internal logic implemention of the endpoints.
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("suavex", NewEthBackendServer(&mockBackend{})))

	clt := &RemoteEthBackend{client: rpc.DialInProc(srv)}

	_, err := clt.BuildEthBlock(context.Background(), &types.BuildBlockArgs{}, nil)
	require.NoError(t, err)

	_, err = clt.BuildEthBlockFromBundles(context.Background(), &types.BuildBlockArgs{}, nil)
	require.NoError(t, err)

	_, err = clt.Call(context.Background(), common.Address{}, nil)
	require.NoError(t, err)
}

// mockBackend is a backend for the EthBackendServer that returns mock data
type mockBackend struct{}

func (n *mockBackend) CurrentHeader() *types.Header {
	return &types.Header{}
}

func (n *mockBackend) BuildBlockFromTxs(ctx context.Context, buildArgs *suave.BuildBlockArgs, txs types.Transactions) (*types.Block, *big.Int, error) {
	block := types.NewBlock(&types.Header{GasUsed: 1000, BaseFee: big.NewInt(1)}, txs, nil, nil, trie.NewStackTrie(nil))
	return block, big.NewInt(11000), nil
}

func (n *mockBackend) BuildBlockFromBundles(ctx context.Context, buildArgs *suave.BuildBlockArgs, bundles []types.SBundle) (*types.Block, *big.Int, error) {
	var txs types.Transactions
	for _, bundle := range bundles {
		txs = append(txs, bundle.Txs...)
	}
	block := types.NewBlock(&types.Header{GasUsed: 1000, BaseFee: big.NewInt(1)}, txs, nil, nil, trie.NewStackTrie(nil))
	return block, big.NewInt(11000), nil
}

func (n *mockBackend) Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error) {
	return []byte{0x1}, nil
}
