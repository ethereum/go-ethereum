package ethclient

import (
	"context"
)

// GetRootHash returns the merkle root of the block headers
func (ec *Client) GetRootHash(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) (string, error) {
	var rootHash string
	if err := ec.c.CallContext(ctx, &rootHash, "eth_getRootHash", startBlockNumber, endBlockNumber); err != nil {
		return "", err
	}
	return rootHash, nil
}
