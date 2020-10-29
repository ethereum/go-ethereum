package ethclient

import (
	"context"

	ethereum "github.com/maticnetwork/bor"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/types"
)

// GetRootHash returns the merkle root of the block headers
func (ec *Client) GetRootHash(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) (string, error) {
	var rootHash string
	if err := ec.c.CallContext(ctx, &rootHash, "eth_getRootHash", startBlockNumber, endBlockNumber); err != nil {
		return "", err
	}
	return rootHash, nil
}

// GetBorBlockReceipt returns bor block receipt
func (ec *Client) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := ec.c.CallContext(ctx, &r, "eth_getBorBlockReceipt", hash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return r, err
}
