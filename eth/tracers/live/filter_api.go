package live

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
)

type filterAPI struct {
	filter *filter
}

func (api *filterAPI) Block(ctx context.Context, blockNr rpc.BlockNumber) ([]*traceResult, error) {
	blknum := uint64(blockNr.Int64())
	if blockNr == rpc.LatestBlockNumber {
		blknum = api.filter.latest
	}

	return api.filter.readBlockTraces(blknum)
}
