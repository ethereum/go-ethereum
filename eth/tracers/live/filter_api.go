package live

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

var errTxNotFound = errors.New("transaction not found")

type filterAPI struct {
	backend tracing.Backend
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

	results := make([]interface{}, 0, len(traces))
	if tracer == "parityTracer" {
		// Convert from []interface{} to []traceResult
		for _, trace := range traces {
			if parityTraces, ok := trace.Result.([]interface{}); ok {
				results = append(results, parityTraces...)
			} else {
				return nil, errors.New("unexpected trace result type")
			}
		}
		return results, nil
	}

	txHashes := make([]common.Hash, 0)
	block, err := api.backend.BlockByNumber(ctx, blockNr)
	if err != nil {
		return nil, err
	}
	for _, tx := range block.Transactions() {
		txHashes = append(txHashes, tx.Hash())
	}
	if len(traces) != len(txHashes) {
		return nil, errors.New("traces and transactions mismatch")
	}

	for i, trace := range traces {
		trace.TxHash = &txHashes[i]
		results = append(results, trace)
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
