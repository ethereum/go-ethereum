package eth

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"

	txtrace2 "github.com/DeBankDeFi/etherlib/pkg/txtracev2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// PreExecAPI provides pre exec info for rpc
type PreExecAPI struct {
	e *Ethereum
}

func NewPreExecAPI(e *Ethereum) *PreExecAPI {
	return &PreExecAPI{e: e}
}

const (
	UnKnown            = 1000
	InsufficientBalane = 1001
	Reverted           = 1002
)

type PreArgs struct {
	ChainId              *big.Int        `json:"chainId,omitempty"`
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`
	Data                 *hexutil.Bytes  `json:"data"`
	Input                *hexutil.Bytes  `json:"input"`
}

type PreError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func toPreError(err error, result *core.ExecutionResult) PreError {
	preErr := PreError{
		Code: UnKnown,
	}
	if err != nil {
		preErr.Msg = err.Error()
	}
	if result != nil && result.Err != nil {
		preErr.Msg = result.Err.Error()
	}
	if strings.HasPrefix(preErr.Msg, "execution reverted") {
		preErr.Code = Reverted
		if result != nil {
			preErr.Msg, _ = abi.UnpackRevert(result.Revert())
		}
	}
	if strings.HasPrefix(preErr.Msg, "out of gas") {
		preErr.Code = Reverted
	}
	if strings.HasPrefix(preErr.Msg, "insufficient funds for transfer") {
		preErr.Code = InsufficientBalane
	}
	if strings.HasPrefix(preErr.Msg, "insufficient balance for transfer") {
		preErr.Code = InsufficientBalane
	}
	if strings.HasPrefix(preErr.Msg, "insufficient funds for gas * price") {
		preErr.Code = InsufficientBalane
	}
	return preErr
}

type PreResult struct {
	Trace     txtrace2.ActionTraceList `json:"trace"`
	Logs      []*types.Log             `json:"logs"`
	StateDiff txtrace2.StateDiff       `json:"stateDiff"`
	Error     PreError                 `json:"error,omitempty"`
	GasUsed   uint64                   `json:"gasUsed"`
}

func (api *PreExecAPI) TraceMany(ctx context.Context, origins []PreArgs) ([]PreResult, error) {
	preResList := make([]PreResult, 0)
	state, header, err := api.e.APIBackend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if state == nil || err != nil {
		return nil, err
	}
	for i := 0; i < len(origins); i++ {
		origin := origins[i]
		if origin.Nonce == nil {
			preResList = append(preResList, PreResult{
				Error: PreError{
					Code: UnKnown,
					Msg:  "nonce is nil",
				},
			})
			continue
		}
		if i > 0 && (uint64)(*origin.Nonce) <= (uint64)(*origins[i-1].Nonce) {
			preResList = append(preResList, PreResult{
				Error: PreError{
					Code: UnKnown,
					Msg:  fmt.Sprintf("nonce decreases, tx index %d has nonce %d, tx index %d has nonce %d", i-1, (uint64)(*origins[i-1].Nonce), i, (uint64)(*origin.Nonce)),
				},
			})
			continue
		}
		txArgs := ethapi.TransactionArgs{
			ChainID:              (*hexutil.Big)(big.NewInt(1)),
			From:                 origin.From,
			To:                   origin.To,
			Gas:                  origin.Gas,
			GasPrice:             origin.GasPrice,
			MaxFeePerGas:         origin.MaxFeePerGas,
			MaxPriorityFeePerGas: origin.MaxPriorityFeePerGas,
			Value:                origin.Value,
			Data:                 origin.Data,
			Input:                origin.Input,
		}
		// Get a new instance of the EVM.
		msg, err := txArgs.ToMessage(api.e.APIBackend.RPCGasCap(), header.BaseFee)
		if err != nil {
			preResList = append(preResList, PreResult{
				Error: PreError{
					Code: UnKnown,
					Msg:  err.Error(),
				},
			})
			continue
		}
		txHash := common.BigToHash(big.NewInt(int64(i)))
		tracer := txtrace2.NewOeTracer(nil, header.Hash(), header.Number, txHash, uint64(i))
		evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true, Tracer: tracer, PreExec: true})
		evm.Context.BaseFee = big.NewInt(0)
		evm.Context.BlockNumber.Add(evm.Context.BlockNumber, big.NewInt(rand.Int63n(6)+6))
		evm.Context.Time += uint64(rand.Int63n(60) + 30)
		if err != nil {
			preResList = append(preResList, PreResult{
				Error: PreError{
					Code: UnKnown,
					Msg:  err.Error(),
				},
			})
			continue
		}
		// Execute the message.
		gp := new(core.GasPool).AddGas(math.MaxUint64)
		state.SetTxContext(txHash, i)
		result, err := core.ApplyMessage(evm, msg, gp)
		if err := vmError(); err != nil {
			preRes := PreResult{
				Error: toPreError(err, result),
			}
			if result != nil {
				preRes.GasUsed = result.UsedGas
			}
			preResList = append(preResList, preRes)
			continue
		}
		if err != nil {
			preRes := PreResult{
				Error: toPreError(err, result),
			}
			if result != nil {
				preRes.GasUsed = result.UsedGas
			}
			preResList = append(preResList, preRes)
			continue
		}
		preRes := PreResult{
			Trace:     tracer.GetTraces(),
			Logs:      state.GetLogs(txHash, header.Number.Uint64(), header.Hash()),
			StateDiff: tracer.GetStateDiff(),
		}
		if result != nil {
			preRes.GasUsed = result.UsedGas
			if result.Failed() {
				preRes.Error = toPreError(err, result)
			}
		}

		if preRes.Error.Msg == "" && len(preRes.Trace) > 0 && (preRes.Trace)[0].Error != "" {
			preRes.Error = PreError{
				Code: Reverted,
				Msg:  (preRes.Trace)[0].Error,
			}
		}
		preResList = append(preResList, preRes)
	}
	return preResList, nil
}
