package tracers

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

type TraceBlock interface {
	GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error)
	GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error)
}

// GetBlockTraceByNumberOrHash replays the block and returns the structured BlockTrace by hash or number.
func (api *API) GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error) {
	var block *types.Block
	if number, ok := blockNrOrHash.Number(); ok {
		block, err = api.blockByNumber(ctx, number)
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	env, err := api.createTraceEnv(ctx, config, block)
	if err != nil {
		return nil, err
	}

	return env.GetBlockTrace(block)
}

func (api *API) GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error) {
	// Try to retrieve the specified block
	var (
		err   error
		block *types.Block
	)
	if number, ok := blockNrOrHash.Number(); ok {
		block, err = api.blockByNumber(ctx, number)
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	block = types.NewBlockWithHeader(block.Header()).WithBody([]*types.Transaction{tx}, nil)

	// create current execution environment.
	env, err := api.createTraceEnv(ctx, config, block)
	if err != nil {
		return nil, err
	}

	return env.GetBlockTrace(block)
}

// Make trace environment for current block.
func (api *API) createTraceEnv(ctx context.Context, config *TraceConfig, block *types.Block) (*core.TraceEnv, error) {
	if config == nil {
		config = &TraceConfig{
			LogConfig: &vm.LogConfig{
				EnableMemory:     false,
				EnableReturnData: true,
			},
		}
	} else if config.Tracer != nil {
		config.Tracer = nil
		log.Warn("Tracer params is unsupported")
	}

	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, err := api.backend.StateAtBlock(ctx, parent, reexec, nil, true, true)
	if err != nil {
		return nil, err
	}

	chaindb := api.backend.ChainDb()
	return core.CreateTraceEnv(api.backend.ChainConfig(), api.chainContext(ctx), api.backend.Engine(), chaindb, statedb, parent, block, true)
}
