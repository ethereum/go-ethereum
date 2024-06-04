package tracers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
	"time"
)

// Rip7560API is the collection of tracing APIs exposed over the private debugging endpoint.
type Rip7560API struct {
	backend Backend
}

func NewRip7560API(backend Backend) *Rip7560API {
	return &Rip7560API{backend: backend}
}

// TraceRip7560Validation mostly copied from 'tracers/api.go' file
func (api *Rip7560API) TraceRip7560Validation(
	ctx context.Context,
	args ethapi.TransactionArgs,
	blockNrOrHash rpc.BlockNumberOrHash,
	config *TraceCallConfig,
) (interface{}, error) {
	number, _ := blockNrOrHash.Number()
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	reexec := defaultTraceReexec
	statedb, release, err := api.backend.StateAtBlock(ctx, block, reexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()

	vmctx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
	if err := args.CallDefaults(api.backend.RPCGasCap(), vmctx.BaseFee, api.backend.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	var (
		msg         = args.ToMessage(vmctx.BaseFee)
		tx          = args.ToTransaction()
		traceConfig *TraceConfig
	)
	if config != nil {
		traceConfig = &config.TraceConfig
	}
	traceResult, err := api.traceTx(ctx, tx, msg, new(Context), vmctx, statedb, traceConfig)
	if err != nil {
		return nil, err
	}
	log.Println("TraceRip7560Validation result")
	log.Println(string(traceResult.(json.RawMessage)))
	return traceResult, err
}

//////// copy-pasted code

// blockByNumber is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *Rip7560API) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := api.backend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// chainContext constructs the context reader which is used by the evm for reading
// the necessary chain context.
func (api *Rip7560API) chainContext(ctx context.Context) core.ChainContext {
	return ethapi.NewChainContext(ctx, api.backend)
}

func (api *Rip7560API) traceTx(ctx context.Context, tx *types.Transaction, message *core.Message, txctx *Context, vmctx vm.BlockContext, statedb *state.StateDB, config *TraceConfig) (interface{}, error) {
	var (
		tracer  *Tracer
		err     error
		timeout = defaultTraceTimeout
		usedGas uint64
	)
	if config == nil {
		config = &TraceConfig{}
	}
	// Default tracer is the struct logger
	//if config.Tracer == nil {
	//	logger := logger.NewStructLogger(config.Config)
	//	tracer = &Tracer{
	//		Hooks:     logger.Hooks(),
	//		GetResult: logger.GetResult,
	//		Stop:      logger.Stop,
	//	}
	//} else {
	tracer, err = DefaultDirectory.New("rip7560Validation", txctx, config.TracerConfig)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	vmenv := vm.NewEVM(vmctx, vm.TxContext{GasPrice: big.NewInt(0)}, statedb, api.backend.ChainConfig(), vm.Config{Tracer: tracer.Hooks, NoBaseFee: true})
	statedb.SetLogger(tracer.Hooks)

	// Define a meaningful timeout of a single transaction trace
	if config.Timeout != nil {
		if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
			return nil, err
		}
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			// Stop evm execution. Note cancellation is not necessarily immediate.
			vmenv.Cancel()
		}
	}()
	defer cancel()

	// Call Prepare to clear out the statedb access list
	statedb.SetTxContext(txctx.TxHash, txctx.TxIndex)
	message.IsRip7560Frame = true
	_, err = core.ApplyTransactionWithEVM(message, api.backend.ChainConfig(), new(core.GasPool).AddGas(message.GasLimit), statedb, vmctx.BlockNumber, txctx.BlockHash, tx, &usedGas, vmenv)
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	return tracer.GetResult()
}
