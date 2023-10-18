package ethclient

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// HeadL1Origin returns the latest L2 block's corresponding L1 origin.
func (ec *Client) HeadL1Origin(ctx context.Context) (*rawdb.L1Origin, error) {
	var res *rawdb.L1Origin

	if err := ec.c.CallContext(ctx, &res, "taiko_headL1Origin"); err != nil {
		return nil, err
	}

	return res, nil
}

// L1OriginByID returns the L2 block's corresponding L1 origin.
func (ec *Client) L1OriginByID(ctx context.Context, blockID *big.Int) (*rawdb.L1Origin, error) {
	var res *rawdb.L1Origin

	if err := ec.c.CallContext(ctx, &res, "taiko_l1OriginByID", hexutil.EncodeBig(blockID)); err != nil {
		return nil, err
	}

	return res, nil
}
