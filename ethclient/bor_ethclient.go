package ethclient

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	zeroAddress = "0x0000000000000000000000000000000000000000"
)

// GetRootHash returns the merkle root of the block headers
func (ec *Client) GetRootHash(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) (string, error) {
	var rootHash string
	if err := ec.c.CallContext(ctx, &rootHash, "bor_getRootHash", startBlockNumber, endBlockNumber); err != nil {
		return "", err
	}

	return rootHash, nil
}

// GetRootHash returns the merkle root of the block headers
func (ec *Client) GetVoteOnHash(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64, hash string, milestoneID string) (bool, error) {
	var value bool
	if err := ec.c.CallContext(ctx, &value, "bor_getVoteOnHash", startBlockNumber, endBlockNumber, hash, milestoneID); err != nil {
		return false, err
	}

	return value, nil
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
