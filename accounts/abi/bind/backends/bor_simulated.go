package backends

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

func (b *SimulatedBackend) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	receipt, err := b.GetBorBlockReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	if receipt == nil {
		return nil, nil
	}

	return receipt, nil
}

func (b *SimulatedBackend) GetVoteOnHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64, hash string, milestoneId string) (bool, error) {
	return false, nil
}

func (b *SimulatedBackend) GetBorBlockLogs(ctx context.Context, hash common.Hash) ([]*types.Log, error) {
	receipt, err := b.GetBorBlockReceipt(ctx, hash)
	if err != nil || receipt == nil {
		return nil, err
	}

	return receipt.Logs, nil
}

// SubscribeStateSyncEvent subscribes to state sync events
func (b *SimulatedBackend) SubscribeStateSyncEvent(ch chan<- core.StateSyncEvent) event.Subscription {
	return b.SubscribeStateSyncEvent(ch)
}
