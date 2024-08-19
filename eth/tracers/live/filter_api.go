package live

import (
	"context"
	"errors"

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

func (api *filterAPI) isSupportedTracer(tracer string) bool {
	_, ok := api.filter.tracer.Tracers()[tracer]
	return ok
}

func (api *filterAPI) Block(ctx context.Context, blockNr rpc.BlockNumber, cfg *traceConfig) ([]interface{}, error) {
	tracer := defaultTraceConfig.Tracer
	if cfg != nil {
		tracer = cfg.Tracer
	}

	if !api.isSupportedTracer(tracer) {
		return nil, errors.New("tracer not found")
	}

	blknum := uint64(blockNr.Int64())
	if blockNr == rpc.LatestBlockNumber {
		blknum = api.filter.latest.Load()
	}

	traces, err := api.filter.readBlockTraces(ctx, tracer, blknum)
	if err != nil {
		return nil, err
	}
	results := make([]interface{}, len(traces))
	for i, trace := range traces {
		if tracer == "parityTracer" {
			results[i] = trace.Result
		} else {
			results[i] = trace
		}
	}

	return results, nil
}
