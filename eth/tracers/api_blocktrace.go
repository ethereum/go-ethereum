package tracers

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var errNoScrollTracerWrapper = errors.New("no ScrollTracerWrapper")

type TraceBlock interface {
	GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error)
	GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error)
}

type scrollTracerWrapper interface {
	CreateTraceEnvAndGetBlockTrace(*params.ChainConfig, core.ChainContext, consensus.Engine, ethdb.Database, *state.StateDB, *types.Header, *types.Block, bool) (*types.BlockTrace, error)
}

// GetBlockTraceByNumberOrHash replays the block and returns the structured BlockTrace by hash or number.
func (api *API) GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error) {
	if api.scrollTracerWrapper == nil {
		return nil, errNoScrollTracerWrapper
	}

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

	return api.createTraceEnvAndGetBlockTrace(ctx, config, block)
}

func (api *API) GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error) {
	if api.scrollTracerWrapper == nil {
		return nil, errNoScrollTracerWrapper
	}

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

	return api.createTraceEnvAndGetBlockTrace(ctx, config, block)
}

// Make trace environment for current block, and then get the trace for the block.
func (api *API) createTraceEnvAndGetBlockTrace(ctx context.Context, config *TraceConfig, block *types.Block) (*types.BlockTrace, error) {
	if config == nil {
		config = &TraceConfig{
			Config: &logger.Config{
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
	statedb, release, err := api.backend.StateAtBlock(ctx, parent, reexec, nil, true, true)
	if err != nil {
		return nil, err
	}
	defer release()

	chaindb := api.backend.ChainDb()
	return api.scrollTracerWrapper.CreateTraceEnvAndGetBlockTrace(api.backend.ChainConfig(), api.chainContext(ctx), api.backend.Engine(), chaindb, statedb, parent.Header(), block, true)
}
