package eth

import (
	"context"
	"fmt"
	"hash"
	"math"
	"math/big"
	"math/rand"
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

func (h *helpHash) Update(key, val []byte) error {
	h.hashed.Write(key)
	h.hashed.Write(val)
	return nil
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
	msg     *core.Message
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

func (api *PreExecAPI) getBlockAndMsg(origin *PreExecTx, header *types.Header) (*types.Block, *core.Message) {
	toAddr := common.HexToAddress(origin.To)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    hexutil.MustDecodeUint64(origin.Nonce),
		To:       &toAddr,
		Value:    hexutil.MustDecodeBig(origin.Value),
		Gas:      hexutil.MustDecodeUint64(origin.Gas),
		GasPrice: hexutil.MustDecodeBig(origin.GasPrice),
		Data:     hexutil.MustDecode(origin.Data),
	})

	number := big.NewInt(header.Number.Int64() + 1)
	block := types.NewBlock(
		&types.Header{Number: number},
		[]*types.Transaction{tx}, nil, nil, newHash())

	msg, _ := core.TransactionToMessage(tx, types.MakeSigner(api.e.APIBackend.ChainConfig(), header.Number), header.BaseFee)
	msg.IsPre = true
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
	d.block, d.msg = api.getBlockAndMsg(origin, d.header)
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

	d.stateDb.SetTxContext(d.tx.Hash(), 0)
	receipt, err := core.ApplyTransactionForPreExec(
		bc.Config(), bc, nil, gp, d.stateDb, d.header, d.tx, &gas, *bc.GetVMConfig())
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
	vmenv := vm.NewEVM(core.NewEVMBlockContext(d.header, bc, nil), txContext, d.stateDb, bc.Config(), vm.Config{Tracer: tracer, PreExec: true})
	vmenv.Context.BaseFee = big.NewInt(0)
	vmenv.Context.BlockNumber.Add(vmenv.Context.BlockNumber, big.NewInt(rand.Int63n(6)+6))
	vmenv.Context.Time += uint64(rand.Int63n(60) + 30)

	// Call Prepare to clear out the statedb access list
	d.stateDb.SetTxContext(d.tx.Hash(), 0)

	tracer.SetMessage(d.block.Number(), d.block.Hash(), d.tx.Hash(), uint(txIndex), d.msg.From, d.msg.To, *d.msg.Value)

	_, err = core.ApplyMessage(vmenv, d.msg, new(core.GasPool).AddGas(d.msg.GasLimit))
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
