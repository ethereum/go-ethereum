package eth

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"math"
	"math/big"
	"strings"

	txtrace "github.com/DeBankDeFi/etherlib/pkg/txtracev1"
	txtrace2 "github.com/DeBankDeFi/etherlib/pkg/txtracev2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	// "github.com/DeBankDeFi/eth/txtrace"
	"golang.org/x/crypto/sha3"
)

type helpHash struct {
	hashed hash.Hash
}

func newHash() *helpHash {

	return &helpHash{hashed: sha3.NewLegacyKeccak256()}
}

func (h *helpHash) Reset() {
	h.hashed.Reset()
}

func (h *helpHash) Update(key, val []byte) {
	h.hashed.Write(key)
	h.hashed.Write(val)
}

func (h *helpHash) Hash() common.Hash {
	return common.BytesToHash(h.hashed.Sum(nil))
}

type PreExecTx struct {
	ChainId                                     *big.Int
	From, To, Data, Value, Gas, GasPrice, Nonce string
}

type preData struct {
	block   *types.Block
	tx      *types.Transaction
	msg     types.Message
	stateDb *state.StateDB
	header  *types.Header
}

// PreExecAPI provides pre exec info for rpc
type PreExecAPI struct {
	e *Ethereum
}

func NewPreExecAPI(e *Ethereum) *PreExecAPI {
	return &PreExecAPI{e: e}
}

func (api *PreExecAPI) getBlockAndMsg(origin *PreExecTx, number *big.Int) (*types.Block, types.Message) {
	fromAddr := common.HexToAddress(origin.From)
	toAddr := common.HexToAddress(origin.To)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    hexutil.MustDecodeUint64(origin.Nonce),
		To:       &toAddr,
		Value:    hexutil.MustDecodeBig(origin.Value),
		Gas:      hexutil.MustDecodeUint64(origin.Gas),
		GasPrice: hexutil.MustDecodeBig(origin.GasPrice),
		Data:     hexutil.MustDecode(origin.Data),
	})

	number.Add(number, big.NewInt(1))
	block := types.NewBlock(
		&types.Header{Number: number},
		[]*types.Transaction{tx}, nil, nil, newHash())

	msg := types.NewMessage(
		fromAddr,
		&toAddr,
		hexutil.MustDecodeUint64(origin.Nonce),
		hexutil.MustDecodeBig(origin.Value),
		hexutil.MustDecodeUint64(origin.Gas),
		hexutil.MustDecodeBig(origin.GasPrice),
		tx.GasFeeCap(),
		tx.GasTipCap(),
		hexutil.MustDecode(origin.Data),
		nil, false, true,
	)
	return block, msg
}

func (api *PreExecAPI) prepareData(ctx context.Context, origin *PreExecTx) (*preData, error) {
	var (
		d   preData
		err error
	)
	bc := api.e.blockchain
	d.header, err = api.e.APIBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, err
	}
	latestNumber := d.header.Number
	parent := api.e.blockchain.GetBlockByNumber(latestNumber.Uint64())
	d.stateDb, err = state.New(parent.Header().Root, bc.StateCache(), bc.Snapshots())
	if err != nil {
		return nil, err
	}
	d.block, d.msg = api.getBlockAndMsg(origin, latestNumber)
	d.tx = d.block.Transactions()[0]
	return &d, nil
}

func (api *PreExecAPI) GetLogs(ctx context.Context, origin *PreExecTx) (*types.Receipt, error) {
	var (
		bc = api.e.blockchain
	)
	d, err := api.prepareData(ctx, origin)
	if err != nil {
		return nil, err
	}
	gas := d.tx.Gas()
	gp := new(core.GasPool).AddGas(gas)

	d.stateDb.Prepare(d.tx.Hash(), 0)
	receipt, err := core.ApplyTransactionForPreExec(
		bc.Config(), bc, nil, gp, d.stateDb, d.header, d.tx, d.msg, &gas, *bc.GetVMConfig())
	if err != nil {
		return nil, err
	}
	return receipt, receipt.Err
}

// TraceTransaction tracing pre-exec transaction object.
func (api *PreExecAPI) TraceTransaction(ctx context.Context, origin *PreExecTx) (interface{}, error) {
	var (
		bc     = api.e.blockchain
		tracer *txtrace.OeTracer
		err    error
	)
	d, err := api.prepareData(ctx, origin)
	if err != nil {
		return nil, err
	}
	txContext := core.NewEVMTxContext(d.msg)
	txIndex := 0

	tracer = txtrace.NewOeTracer(nil)
	// Run the transaction with tracing enabled.
	vmenv := vm.NewEVM(core.NewEVMBlockContext(d.header, bc, nil), txContext, d.stateDb, bc.Config(), vm.Config{Debug: true, Tracer: tracer})
	vmenv.Context.BaseFee = big.NewInt(0)

	// Call Prepare to clear out the statedb access list
	d.stateDb.Prepare(d.tx.Hash(), txIndex)

	tracer.SetMessage(d.block.Number(), d.block.Hash(), d.tx.Hash(), uint(txIndex), d.msg.From(), d.msg.To(), *d.msg.Value())

	_, err = core.ApplyMessage(vmenv, d.msg, new(core.GasPool).AddGas(d.msg.Gas()))
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %v", err)
	}

	tracer.Finalize()
	return tracer.GetResult(), nil
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
		stateOld := state.Copy()
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
		msg, err := txArgs.ToMessage(0, nil)
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
		evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true, Debug: true, Tracer: tracer})
		evm.Context.BaseFee = big.NewInt(0)
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
		state.Prepare(txHash, i)
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
			Logs:      state.GetLogs(txHash, header.Hash()),
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
		if preRes.Error.Code == 0 {
			gasLimit := header.GasLimit
			gasCap := api.e.APIBackend.RPCGasCap()
			if gasLimit > gasCap {
				gasLimit = gasCap
			}
			usedGas := api.doEstimateGas(ctx, txArgs, header, stateOld, gasLimit)
			log.Debug("doEstimateGas", "usedGas", usedGas)
			if usedGas > 0 {
				preRes.GasUsed = usedGas
			}
		}
		preResList = append(preResList, preRes)
	}
	return preResList, nil
}

func (api *PreExecAPI) doEstimateGas(ctx context.Context, args ethapi.TransactionArgs, header *types.Header, state *state.StateDB, gasLimit uint64) uint64 {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo uint64 = params.TxGas - 1
		hi uint64
	)
	// Use zero address if sender unspecified.
	if args.From == nil {
		args.From = new(common.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = gasLimit
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		balance := state.GetBalance(*args.From) // from can't be nil
		available := new(big.Int).Set(balance)
		if args.Value != nil {
			if args.Value.ToInt().Cmp(available) >= 0 {
				return 0
			}
			available.Sub(available, args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}
	// Recap the highest gas allowance with specified gascap.
	if gasLimit != 0 && hi > gasLimit {
		hi = gasLimit
	}
	cap := hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, *core.ExecutionResult, error) {
		args.Gas = (*hexutil.Uint64)(&gas)

		result, err := api.doCallCopy(ctx, args, header, state)
		log.Debug("doCallCopy", "result", result, "err", err)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, err // Bail out
		}
		return result.Failed(), result, nil
	}

	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, result, err := executable(hi)
		if err != nil {
			return 0
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas {
				if len(result.Revert()) > 0 {
					return 0
				}
				return 0
			}
			// Otherwise, the specified gas cap is too low
			return 0
		}
	}
	return hi
}

func (api *PreExecAPI) doCallCopy(ctx context.Context, args ethapi.TransactionArgs, header *types.Header, state *state.StateDB) (*core.ExecutionResult, error) {
	stateNew := state.Copy()
	// Get a new instance of the EVM.
	msg, err := args.ToMessage(api.e.APIBackend.RPCGasCap(), header.BaseFee)
	if err != nil {
		return nil, err
	}
	evm, vmError, err := api.e.APIBackend.GetEVM(ctx, msg, stateNew, header, &vm.Config{NoBaseFee: true})
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
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	return result, nil
}
