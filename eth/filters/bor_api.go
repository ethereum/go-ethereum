package filters

import (
	"context"

	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/rpc"
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
