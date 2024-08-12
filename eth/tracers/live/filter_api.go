package live

import (
	"context"

	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
)

type filterAPI struct {
	backend tracers.Backend
	filter  *filter
}

type traceConfig struct {
	Tracer string `json:"tracer"`
}

var defaultTraceConfig = &traceConfig{
	Tracer: "callTracer",
}

func (api *filterAPI) Block(ctx context.Context, blockNr rpc.BlockNumber, cfg *traceConfig) ([]*traceResult, error) {
	blknum := uint64(blockNr.Int64())
	if blockNr == rpc.LatestBlockNumber {
		blknum = api.filter.latest.Load()
	}

	tracer := defaultTraceConfig.Tracer
	if cfg != nil {
		tracer = cfg.Tracer
	}

	return api.filter.readBlockTraces(ctx, tracer, blknum)
}
