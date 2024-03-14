package api

import (
	"context"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

var _ API = (*APIClient)(nil)

type APIClient struct {
	rpc rpcClient
}

func NewClient(endpoint string) (*APIClient, error) {
	clt, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return NewClientFromRPC(clt), nil
}

type rpcClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

func NewClientFromRPC(rpc rpcClient) *APIClient {
	return &APIClient{rpc: rpc}
}

func (a *APIClient) NewSession(ctx context.Context, args *BuildBlockArgs) (string, error) {
	var id string
	err := a.rpc.CallContext(ctx, &id, "suavex_newSession", args)
	return id, err
}

func (a *APIClient) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error) {
	var receipt *SimulateTransactionResult
	err := a.rpc.CallContext(ctx, &receipt, "suavex_addTransaction", sessionId, tx)
	return receipt, err
}

func (a *APIClient) BuildBlock(ctx context.Context, sessionId string) error {
	return a.rpc.CallContext(ctx, nil, "suavex_buildBlock", sessionId)
}

func (a *APIClient) Bid(ctx context.Context, sessioId string, blsPubKey phase0.BLSPubKey) (*SubmitBlockRequest, error) {
	var req *SubmitBlockRequest
	err := a.rpc.CallContext(ctx, &req, "suavex_bid", sessioId, blsPubKey)
	return req, err
}

func (a *APIClient) GetBalance(ctx context.Context, sessionId string, addr common.Address) (*big.Int, error) {
	var balance *big.Int
	err := a.rpc.CallContext(ctx, &balance, "suavex_getBalance", sessionId, addr)
	return balance, err
}
