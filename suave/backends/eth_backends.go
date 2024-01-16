package backends

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	builder "github.com/ethereum/go-ethereum/suave/builder/api"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	_ EthBackend = &EthMock{}
	_ EthBackend = &RemoteEthBackend{}
)

type EthMock struct {
	*builder.MockServer
}

func (e *EthMock) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	block := types.NewBlock(&types.Header{GasUsed: 1000}, txs, nil, nil, trie.NewStackTrie(nil))
	return engine.BlockToExecutableData(block, big.NewInt(11000), nil), nil
}

func (e *EthMock) BuildEthBlockFromBundles(ctx context.Context, args *suave.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error) {
	var txs types.Transactions
	for _, bundle := range bundles {
		txs = append(txs, bundle.Txs...)
	}
	block := types.NewBlock(&types.Header{GasUsed: 1000}, txs, nil, nil, trie.NewStackTrie(nil))
	return engine.BlockToExecutableData(block, big.NewInt(11000), nil), nil
}

func (e *EthMock) Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error) {
	return nil, nil
}

type RemoteEthBackend struct {
	endpoint string
	client   *rpc.Client

	*builder.APIClient
}

func NewRemoteEthBackend(endpoint string) *RemoteEthBackend {
	r := &RemoteEthBackend{
		endpoint: endpoint,
	}

	r.APIClient = builder.NewClientFromRPC(r)
	return r
}

func (e *RemoteEthBackend) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if e.client == nil {
		// should lock
		var err error
		client, err := rpc.DialContext(ctx, e.endpoint)
		if err != nil {
			return err
		}
		e.client = client
	}

	err := e.client.CallContext(ctx, &result, method, args...)
	if err != nil {
		client := e.client
		e.client = nil
		client.Close()
		return err
	}

	return nil
}

func (e *RemoteEthBackend) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	var result engine.ExecutionPayloadEnvelope
	err := e.CallContext(ctx, &result, "suavex_buildEthBlock", args, txs)

	return &result, err
}

func (e *RemoteEthBackend) BuildEthBlockFromBundles(ctx context.Context, args *suave.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error) {
	var result engine.ExecutionPayloadEnvelope
	err := e.CallContext(ctx, &result, "suavex_buildEthBlockFromBundles", args, bundles)

	return &result, err
}

func (e *RemoteEthBackend) Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error) {
	var result []byte
	err := e.CallContext(ctx, &result, "suavex_call", contractAddr, input)

	return result, err
}
