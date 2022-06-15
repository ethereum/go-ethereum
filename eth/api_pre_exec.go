package eth

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"

	txtrace "github.com/DeBankDeFi/etherlib/pkg/txtracev1"
	// "github.com/DeBankDeFi/eth/txtrace"
)

type PreExecTx struct {
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

type PreExecResult struct {
	Trace *[]txtrace.ActionTrace `json:"trace"`
	Logs  []*types.Log           `json:"logs"`
}

// PreExecAPI provides pre exec info for rpc
type PreExecAPI struct {
	e *Ethereum
}

func NewPreExecAPI(e *Ethereum) *PreExecAPI {
	return &PreExecAPI{e: e}
}

func (api *PreExecAPI) GetLogs(ctx context.Context, origin *PreExecTx) (*types.Receipt, error) {
	state, header, err := api.e.APIBackend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if state == nil || err != nil {
		return nil, err
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
		Nonce:                origin.Nonce,
		Data:                 origin.Data,
		Input:                origin.Input,
	}
	// Get a new instance of the EVM.
	msg, err := txArgs.ToMessage(0, header.BaseFee)
	if err != nil {
		return nil, err
	}
	evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true})
	if err != nil {
		return nil, err
	}
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	receipt := types.NewReceipt(bytes.NewBufferString("").Bytes(), result.Failed(), result.UsedGas)
	receipt.GasUsed = result.UsedGas
	receipt.Logs = state.Logs()
	receipt.BlockHash = header.Hash()
	receipt.BlockNumber = header.Number
	return receipt, nil
}

// TraceTransaction tracing pre-exec transaction object.
func (api *PreExecAPI) TraceTransaction(ctx context.Context, origin *PreExecTx) (interface{}, error) {
	state, header, err := api.e.APIBackend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if state == nil || err != nil {
		return nil, err
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
		Nonce:                origin.Nonce,
		Data:                 origin.Data,
		Input:                origin.Input,
	}
	// Get a new instance of the EVM.
	msg, err := txArgs.ToMessage(0, header.BaseFee)
	if err != nil {
		return nil, err
	}
	tracer := txtrace.NewOeTracer(nil)
	tracer.SetFrom(msg.From())
	tracer.SetTo(msg.To())
	tracer.SetValue(*msg.Value())
	tracer.SetTxIndex(uint(0))
	tracer.SetBlockNumber(header.Number)
	tracer.SetBlockHash(header.Hash())
	evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true, Debug: true, Tracer: tracer})
	if err != nil {
		return nil, err
	}
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	tracer.SetGasUsed(result.UsedGas)
	tracer.Finalize()
	return tracer.GetResult(), nil
}

func (api *PreExecAPI) TraceMany(ctx context.Context, origins []PreExecTx) (*PreExecResult, error) {
	state, header, err := api.e.APIBackend.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if state == nil || err != nil {
		return nil, err
	}
	for i := 0; i < len(origins)-1; i++ {
		origin := origins[i]
		txArgs := ethapi.TransactionArgs{
			ChainID:              (*hexutil.Big)(big.NewInt(1)),
			From:                 origin.From,
			To:                   origin.To,
			Gas:                  origin.Gas,
			GasPrice:             origin.GasPrice,
			MaxFeePerGas:         origin.MaxFeePerGas,
			MaxPriorityFeePerGas: origin.MaxPriorityFeePerGas,
			Value:                origin.Value,
			Nonce:                origin.Nonce,
			Data:                 origin.Data,
			Input:                origin.Input,
		}
		// Get a new instance of the EVM.
		msg, err := txArgs.ToMessage(0, header.BaseFee)
		if err != nil {
			return nil, err
		}
		evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true})
		if err != nil {
			return nil, err
		}
		// Execute the message.
		gp := new(core.GasPool).AddGas(math.MaxUint64)
		_, err = core.ApplyMessage(evm, msg, gp)
		if err := vmError(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
		}
	}
	origin := origins[len(origins)-1]
	txArgs := ethapi.TransactionArgs{
		ChainID:              (*hexutil.Big)(big.NewInt(1)),
		From:                 origin.From,
		To:                   origin.To,
		Gas:                  origin.Gas,
		GasPrice:             origin.GasPrice,
		MaxFeePerGas:         origin.MaxFeePerGas,
		MaxPriorityFeePerGas: origin.MaxPriorityFeePerGas,
		Value:                origin.Value,
		Nonce:                origin.Nonce,
		Data:                 origin.Data,
		Input:                origin.Input,
	}
	// Get a new instance of the EVM.
	msg, err := txArgs.ToMessage(0, header.BaseFee)
	if err != nil {
		return nil, err
	}
	tracer := txtrace.NewOeTracer(nil)
	tracer.SetFrom(msg.From())
	tracer.SetTo(msg.To())
	tracer.SetValue(*msg.Value())
	tracer.SetTxIndex(uint(0))
	tracer.SetBlockNumber(header.Number)
	tracer.SetBlockHash(header.Hash())
	evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true, Debug: true, Tracer: tracer})
	if err != nil {
		return nil, err
	}
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	txHash := common.BigToHash(big.NewInt(1))
	state.Prepare(txHash, len(origins)-1)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	tracer.SetGasUsed(result.UsedGas)
	tracer.Finalize()
	return &PreExecResult{Trace: tracer.GetResult(), Logs: state.GetLogs(txHash, header.Hash())}, nil
}
