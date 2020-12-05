package filters

import (
	"bytes"
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func (api *PublicFilterAPI) GetBorBlockLogs(ctx context.Context, crit FilterCriteria) ([]*types.Log, error) {
	var filter *BorBlockLogsFilter
	if crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter
		filter = NewBorBlockLogsFilter(api.backend, *crit.BlockHash, crit.Addresses, crit.Topics)
	} else {
		// Convert the RPC block numbers into internal representations
		begin := rpc.LatestBlockNumber.Int64()
		if crit.FromBlock != nil {
			begin = crit.FromBlock.Int64()
		}
		end := rpc.LatestBlockNumber.Int64()
		if crit.ToBlock != nil {
			end = crit.ToBlock.Int64()
		}
		// Construct the range filter
		filter = NewBorBlockLogsRangeFilter(api.backend, begin, end, crit.Addresses, crit.Topics)
	}

	// Run the filter and return all the logs
	logs, err := filter.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), err
}

// NewDeposits send a notification each time a new deposit received from bridge.
func (api *PublicFilterAPI) NewDeposits(ctx context.Context, crit ethereum.StateSyncFilter) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()
	go func() {
		stateSyncData := make(chan *types.StateSyncData)
		stateSyncSub := api.events.SubscribeNewDeposits(stateSyncData)

		for {
			select {
			case h := <-stateSyncData:
				if crit.ID == h.ID || bytes.Compare(crit.Contract.Bytes(), h.Contract.Bytes()) == 0 ||
					(crit.ID == 0 && crit.Contract == common.Address{}) {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				stateSyncSub.Unsubscribe()
				return
			case <-notifier.Closed():
				stateSyncSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}
