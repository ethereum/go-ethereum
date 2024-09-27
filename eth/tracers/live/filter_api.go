package live

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	return api.readBlockTraces(ctx, tracer, blknum, tracer == "parityTracer")
}

func (api *filterAPI) readBlockTraces(ctx context.Context, tracer string, blknum uint64, isParity bool) ([]interface{}, error) {
	traces, err := api.filter.readBlockTraces(ctx, tracer, blknum)
	if err != nil {
		return nil, err
	}

	results := make([]interface{}, 0, len(traces))
	if isParity {
		// Convert from []interface{} to []traceResult
		for i, trace := range traces {
			if parityTraces, ok := trace.Result.([]interface{}); ok {
				results = append(results, parityTraces...)
			} else {
				return nil, fmt.Errorf("invalid trace result type at index: %d", i)
			}
		}
		return results, nil
	}

	block, err := api.backend.BlockByNumber(ctx, rpc.BlockNumber(blknum))
	if err != nil {
		return nil, err
	}
	txHashes := make([]common.Hash, 0)
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

// traceFilterConfig represents the arguments for trace_filter
type traceFilterConfig struct {
	FromBlock   *hexutil.Uint64   `json:"fromBlock"`
	ToBlock     *hexutil.Uint64   `json:"toBlock"`
	FromAddress []*common.Address `json:"fromAddress"`
	ToAddress   []*common.Address `json:"toAddress"`
	Mode        TraceFilterMode   `json:"mode"`
	After       *uint64           `json:"after"`
	Count       *uint64           `json:"count"`
}

type TraceFilterMode string

const (
	// TraceFilterModeUnion is default mode for TraceFilter.
	// Unions results referred to addresses from FromAddress or ToAddress
	TraceFilterModeUnion = "union"
	// TraceFilterModeIntersection retrieves results referred to addresses provided both in FromAddress and ToAddress
	TraceFilterModeIntersection = "intersection"
)

// Filter returns traces for the given filter configuration.
func (api *filterAPI) Filter(ctx context.Context, req traceFilterConfig, cfg *traceConfig) (interface{}, error) {
	tracer, err := api.getTracerOrDefault(cfg)
	if err != nil {
		return nil, err
	}
	isParity := tracer == "parityTracer"

	if !isParity && len(req.FromAddress)+len(req.ToAddress) > 0 {
		return nil, errors.New("invalid parameters: filter with fromAddress or toAddress is only supported in parityTracer")
	}

	var (
		fromBlock = uint64(0)
		toBlock   = uint64(0)
		count     = uint64(^uint(0))
		after     = uint64(0)
		// fromAddrs = extractAddres(req.FromAddress)
		// toAddrs   = extractAddres(req.ToAddress)
	)

	if req.FromBlock != nil {
		fromBlock = uint64(*req.FromBlock)
	}
	if req.ToBlock != nil {
		toBlock = uint64(*req.ToBlock)
	} else {
		toBlock = api.filter.latest.Load()
	}

	if fromBlock > toBlock {
		return nil, errors.New("invalid parameters: fromBlock cannot be greater than toBlock")
	}

	if req.Count != nil {
		count = *req.Count
	}
	if req.After != nil {
		after = *req.After
	}

	return exportLimitedTraces(func(blknum uint64) ([]interface{}, error) { return api.readBlockTraces(ctx, tracer, blknum, isParity) }, fromBlock, toBlock, count, after)
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

func extractAddres(addrs []*common.Address) map[common.Address]struct{} {
	result := make(map[common.Address]struct{}, len(addrs))
	for _, addr := range addrs {
		if addr != nil {
			result[*addr] = struct{}{}
		}
	}
	return result
}

func exportLimitedTraces(gen func(blknum uint64) ([]interface{}, error), fromBlock, toBlock, count, after uint64) ([]interface{}, error) {
	var (
		nExported uint64                         // Number of traces exported
		nSkipped  uint64                         // Number of traces skipped
		results   = make([]interface{}, 0, 1024) // 1024 is the initial capacity
	)

	for blknum := fromBlock; blknum <= toBlock && nExported < count; blknum++ {
		traces, err := gen(uint64(blknum))
		if err != nil {
			return nil, err
		}

		nTraces := uint64(len(traces))
		if after > nSkipped {
			skip := min(after-nSkipped, nTraces)
			nSkipped += skip
			if skip == nTraces {
				// Skip if the whole block is skipped
				continue
			}
			traces = traces[skip:]
		}

		// Export at most the remaining traces
		maxExport := min(count-nExported, uint64(len(traces)))
		results = append(results, traces[:maxExport]...)
		nExported += maxExport
	}
	return results, nil
}
