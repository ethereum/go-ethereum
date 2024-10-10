package live

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/internal/ethapi"
)

type ethAPI struct {
	backend tracing.Backend
	live    *live
}

func (n *ethAPI) GetTransactionBySenderAndNonce(ctx context.Context, sender common.Address, nonce hexutil.Uint) (*ethapi.RPCTransaction, error) {
	// TODO:
	// 1. return nil if sender is a contract

	// Retrieve from txpool first
	if tx := n.backend.GetPoolTransactionBySenderAndNonce(sender, uint64(nonce)); tx != nil {
		return ethapi.NewRPCPendingTransaction(tx, n.backend.CurrentHeader(), n.backend.ChainConfig()), nil
	}

	txHash, err := n.live.kvdb.Get(append(sender.Bytes(), encodeNumber(uint64(nonce))...))
	if err != nil {
		return nil, nil
	}

	found, tx, blockHash, blockNumber, index, err := n.backend.GetTransaction(ctx, common.BytesToHash(txHash))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("transaction not found")
	}

	header, err := n.backend.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	return ethapi.NewRPCTransaction(tx, blockHash, blockNumber, header.Time, index, header.BaseFee, n.backend.ChainConfig()), nil
}
