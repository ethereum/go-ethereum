package live

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

var errTxNotFound = errors.New("transaction not found")

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
	tracer, err := api.getTracerOrDefault(cfg)
	if err != nil {
		return nil, err
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

func (api *filterAPI) getTracerOrDefault(cfg *traceConfig) (string, error) {
	if cfg == nil {
		return defaultTraceConfig.Tracer, nil
	}
	tracer := cfg.Tracer

	if !api.isSupportedTracer(tracer) {
		return "", errors.New("tracer not found")
	}
	return tracer, nil
}

func (api *filterAPI) Transaction(ctx context.Context, hash common.Hash, cfg *traceConfig) (interface{}, error) {
	tracer, err := api.getTracerOrDefault(cfg)
	if err != nil {
		return nil, err
	}

	found, _, _, blknum, index, err := api.backend.GetTransaction(ctx, hash)
	if err != nil {
		return nil, ethapi.NewTxIndexingError()
	}
	if !found {
		return nil, errTxNotFound
	}
	traces, err := api.filter.readBlockTraces(ctx, tracer, blknum)
	if err != nil {
		return nil, err
	}

	if index >= uint64(len(traces)) {
		return nil, nil
	}

	return traces[index].Result, nil
}
