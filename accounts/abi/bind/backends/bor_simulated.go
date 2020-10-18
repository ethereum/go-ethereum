package backends

import (
	"context"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
)

func (fb *filterBackend) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.BorReceipt, error) {
	number := rawdb.ReadHeaderNumber(fb.db, hash)
	if number == nil {
		return nil, nil
	}
	receipt := rawdb.ReadRawBorReceipt(fb.db, hash, *number)
	if receipt == nil {
		return nil, nil
	}

	return receipt, nil
}

func (fb *filterBackend) GetBorBlockLogs(ctx context.Context, hash common.Hash) ([]*types.Log, error) {
	receipt, err := fb.GetBorBlockReceipt(ctx, hash)
	if err != nil || receipt == nil {
		return nil, err
	}

	return receipt.Logs, nil
}
