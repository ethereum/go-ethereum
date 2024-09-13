// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// maxSimulateBlocks is the maximum number of blocks that can be simulated
	// in a single request.
	maxSimulateBlocks = 256

	// timestampIncrement is the default increment between block timestamps.
	timestampIncrement = 1
)

// simBlock is a batch of calls to be simulated sequentially.
type simBlock struct {
	BlockOverrides *BlockOverrides
	StateOverrides *StateOverride
	Calls          []TransactionArgs
}

// simCallResult is the result of a simulated call.
type simCallResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

func (r *simCallResult) MarshalJSON() ([]byte, error) {
	type callResultAlias simCallResult
	// Marshal logs to be an empty array instead of nil when empty
	if r.Logs == nil {
		r.Logs = []*types.Log{}
	}
	return json.Marshal((*callResultAlias)(r))
}

// simOpts are the inputs to eth_simulateV1.
type simOpts struct {
	BlockStateCalls        []simBlock
	TraceTransfers         bool
	Validation             bool
	ReturnFullTransactions bool
}

// simulator is a stateful object that simulates a series of blocks.
// it is not safe for concurrent use.
type simulator struct {
	b              Backend
	state          *state.StateDB
	base           *types.Header
	chainConfig    *params.ChainConfig
	gp             *core.GasPool
	traceTransfers bool
	validate       bool
	fullTx         bool
}

// execute runs the simulation of a series of blocks.
func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		cancel  context.CancelFunc
		timeout = sim.b.RPCEVMTimeout()
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	var err error
	blocks, err = sim.sanitizeChain(blocks)
	if err != nil {
		return nil, err
	}
	// Prepare block headers with preliminary fields for the response.
	headers, err := sim.makeHeaders(blocks)
	if err != nil {
		return nil, err
	}
	var (
		results = make([]map[string]interface{}, len(blocks))
		parent  = sim.base
		// Assume same total difficulty for all simulated blocks.
		td = sim.b.GetTd(ctx, sim.base.Hash())
	)
	for bi, block := range blocks {
		result, callResults, err := sim.processBlock(ctx, &block, headers[bi], parent, headers[:bi], timeout)
		if err != nil {
			return nil, err
		}
		enc := RPCMarshalBlock(result, true, sim.fullTx, sim.chainConfig)
		enc["totalDifficulty"] = (*hexutil.Big)(td)
		enc["calls"] = callResults
		results[bi] = enc

		parent = headers[bi]
	}
	return results, nil
}

func (sim *simulator) processBlock(ctx context.Context, block *simBlock, header, parent *types.Header, headers []*types.Header, timeout time.Duration) (*types.Block, []simCallResult, error) {
	// Set header fields that depend only on parent block.
	// Parent hash is needed for evm.GetHashFn to work.
	header.ParentHash = parent.Hash()
	if sim.chainConfig.IsLondon(header.Number) {
		// In non-validation mode base fee is set to 0 if it is not overridden.
		// This is because it creates an edge case in EVM where gasPrice < baseFee.
		// Base fee could have been overridden.
		if header.BaseFee == nil {
			if sim.validate {
				header.BaseFee = eip1559.CalcBaseFee(sim.chainConfig, parent)
			} else {
				header.BaseFee = big.NewInt(0)
			}
		}
	}
	if sim.chainConfig.IsCancun(header.Number, header.Time) {
		var excess uint64
		if sim.chainConfig.IsCancun(parent.Number, parent.Time) {
			excess = eip4844.CalcExcessBlobGas(*parent.ExcessBlobGas, *parent.BlobGasUsed)
		} else {
			excess = eip4844.CalcExcessBlobGas(0, 0)
		}
		header.ExcessBlobGas = &excess
	}
	blockContext := core.NewEVMBlockContext(header, sim.newSimulatedChainContext(ctx, headers), nil)
	if block.BlockOverrides.BlobBaseFee != nil {
		blockContext.BlobBaseFee = block.BlockOverrides.BlobBaseFee.ToInt()
	}
	precompiles := sim.activePrecompiles(sim.base)
	// State overrides are applied prior to execution of a block
	if err := block.StateOverrides.Apply(sim.state, precompiles); err != nil {
		return nil, nil, err
	}
	var (
		gasUsed, blobGasUsed uint64
		txes                 = make([]*types.Transaction, len(block.Calls))
		callResults          = make([]simCallResult, len(block.Calls))
		receipts             = make([]*types.Receipt, len(block.Calls))
		// Block hash will be repaired after execution.
		tracer   = newTracer(sim.traceTransfers, blockContext.BlockNumber.Uint64(), common.Hash{}, common.Hash{}, 0)
		vmConfig = &vm.Config{
			NoBaseFee: !sim.validate,
			Tracer:    tracer.Hooks(),
		}
		evm = vm.NewEVM(blockContext, vm.TxContext{GasPrice: new(big.Int)}, sim.state, sim.chainConfig, *vmConfig)
	)
	sim.state.SetLogger(tracer.Hooks())
	// It is possible to override precompiles with EVM bytecode, or
	// move them to another address.
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	for i, call := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if err := sim.sanitizeCall(&call, sim.state, header, blockContext, &gasUsed); err != nil {
			return nil, nil, err
		}
		tx := call.ToTransaction(types.DynamicFeeTxType)
		txes[i] = tx
		tracer.reset(tx.Hash(), uint(i))
		// EoA check is always skipped, even in validation mode.
		msg := call.ToMessage(header.BaseFee, !sim.validate, true)
		evm.Reset(core.NewEVMTxContext(msg), sim.state)
		result, err := applyMessageWithEVM(ctx, evm, msg, sim.state, timeout, sim.gp)
		if err != nil {
			txErr := txValidationError(err)
			return nil, nil, txErr
		}
		// Update the state with pending changes.
		var root []byte
		if sim.chainConfig.IsByzantium(blockContext.BlockNumber) {
			sim.state.Finalise(true)
		} else {
			root = sim.state.IntermediateRoot(sim.chainConfig.IsEIP158(blockContext.BlockNumber)).Bytes()
		}
		gasUsed += result.UsedGas
		receipts[i] = core.MakeReceipt(evm, result, sim.state, blockContext.BlockNumber, common.Hash{}, tx, gasUsed, root)
		blobGasUsed += receipts[i].BlobGasUsed
		logs := tracer.Logs()
		callRes := simCallResult{ReturnValue: result.Return(), Logs: logs, GasUsed: hexutil.Uint64(result.UsedGas)}
		if result.Failed() {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusFailed)
			if errors.Is(result.Err, vm.ErrExecutionReverted) {
				// If the result contains a revert reason, try to unpack it.
				revertErr := newRevertError(result.Revert())
				callRes.Error = &callError{Message: revertErr.Error(), Code: errCodeReverted, Data: revertErr.ErrorData().(string)}
			} else {
				callRes.Error = &callError{Message: result.Err.Error(), Code: errCodeVMError}
			}
		} else {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
		}
		callResults[i] = callRes
	}
	header.Root = sim.state.IntermediateRoot(true)
	header.GasUsed = gasUsed
	if sim.chainConfig.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = &blobGasUsed
	}
	var withdrawals types.Withdrawals
	if sim.chainConfig.IsShanghai(header.Number, header.Time) {
		withdrawals = make([]*types.Withdrawal, 0)
	}
	b := types.NewBlock(header, &types.Body{Transactions: txes, Withdrawals: withdrawals}, receipts, trie.NewStackTrie(nil))
	repairLogs(callResults, b.Hash())
	return b, callResults, nil
}

// repairLogs updates the block hash in the logs present in the result of
// a simulated block. This is needed as during execution when logs are collected
// the block hash is not known.
func repairLogs(calls []simCallResult, hash common.Hash) {
	for i := range calls {
		for j := range calls[i].Logs {
			calls[i].Logs[j].BlockHash = hash
		}
	}
}

func (sim *simulator) sanitizeCall(call *TransactionArgs, state *state.StateDB, header *types.Header, blockContext vm.BlockContext, gasUsed *uint64) error {
	if call.Nonce == nil {
		nonce := state.GetNonce(call.from())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	// Let the call run wild unless explicitly specified.
	if call.Gas == nil {
		remaining := blockContext.GasLimit - *gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	if *gasUsed+uint64(*call.Gas) > blockContext.GasLimit {
		return &blockGasLimitReachedError{fmt.Sprintf("block gas limit reached: %d >= %d", gasUsed, blockContext.GasLimit)}
	}
	if err := call.CallDefaults(sim.gp.Gas(), header.BaseFee, sim.chainConfig.ChainID); err != nil {
		return err
	}
	return nil
}

func (sim *simulator) activePrecompiles(base *types.Header) vm.PrecompiledContracts {
	var (
		isMerge = (base.Difficulty.Sign() == 0)
		rules   = sim.chainConfig.Rules(base.Number, isMerge, base.Time)
	)
	return maps.Clone(vm.ActivePrecompiledContracts(rules))
}

// sanitizeChain checks the chain integrity. Specifically it checks that
// block numbers and timestamp are strictly increasing, setting default values
// when necessary. Gaps in block numbers are filled with empty blocks.
// Note: It modifies the block's override object.
func (sim *simulator) sanitizeChain(blocks []simBlock) ([]simBlock, error) {
	var (
		res           = make([]simBlock, 0, len(blocks))
		base          = sim.base
		prevNumber    = base.Number
		prevTimestamp = base.Time
	)
	for _, block := range blocks {
		if block.BlockOverrides == nil {
			block.BlockOverrides = new(BlockOverrides)
		}
		if block.BlockOverrides.Number == nil {
			n := new(big.Int).Add(prevNumber, big.NewInt(1))
			block.BlockOverrides.Number = (*hexutil.Big)(n)
		}
		diff := new(big.Int).Sub(block.BlockOverrides.Number.ToInt(), prevNumber)
		if diff.Cmp(common.Big0) <= 0 {
			return nil, &invalidBlockNumberError{fmt.Sprintf("block numbers must be in order: %d <= %d", block.BlockOverrides.Number.ToInt().Uint64(), prevNumber)}
		}
		if total := new(big.Int).Sub(block.BlockOverrides.Number.ToInt(), base.Number); total.Cmp(big.NewInt(maxSimulateBlocks)) > 0 {
			return nil, &clientLimitExceededError{message: "too many blocks"}
		}
		if diff.Cmp(big.NewInt(1)) > 0 {
			// Fill the gap with empty blocks.
			gap := new(big.Int).Sub(diff, big.NewInt(1))
			// Assign block number to the empty blocks.
			for i := uint64(0); i < gap.Uint64(); i++ {
				n := new(big.Int).Add(prevNumber, big.NewInt(int64(i+1)))
				t := prevTimestamp + timestampIncrement
				b := simBlock{BlockOverrides: &BlockOverrides{Number: (*hexutil.Big)(n), Time: (*hexutil.Uint64)(&t)}}
				prevTimestamp = t
				res = append(res, b)
			}
		}
		// Only append block after filling a potential gap.
		prevNumber = block.BlockOverrides.Number.ToInt()
		var t uint64
		if block.BlockOverrides.Time == nil {
			t = prevTimestamp + timestampIncrement
			block.BlockOverrides.Time = (*hexutil.Uint64)(&t)
		} else {
			t = uint64(*block.BlockOverrides.Time)
			if t <= prevTimestamp {
				return nil, &invalidBlockTimestampError{fmt.Sprintf("block timestamps must be in order: %d <= %d", t, prevTimestamp)}
			}
		}
		prevTimestamp = t
		res = append(res, block)
	}
	return res, nil
}

// makeHeaders makes header object with preliminary fields based on a simulated block.
// Some fields have to be filled post-execution.
// It assumes blocks are in order and numbers have been validated.
func (sim *simulator) makeHeaders(blocks []simBlock) ([]*types.Header, error) {
	var (
		res    = make([]*types.Header, len(blocks))
		base   = sim.base
		header = base
	)
	for bi, block := range blocks {
		if block.BlockOverrides == nil || block.BlockOverrides.Number == nil {
			return nil, errors.New("empty block number")
		}
		overrides := block.BlockOverrides

		var withdrawalsHash *common.Hash
		if sim.chainConfig.IsShanghai(overrides.Number.ToInt(), (uint64)(*overrides.Time)) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		var parentBeaconRoot *common.Hash
		if sim.chainConfig.IsCancun(overrides.Number.ToInt(), (uint64)(*overrides.Time)) {
			parentBeaconRoot = &common.Hash{}
		}
		header = overrides.MakeHeader(&types.Header{
			UncleHash:        types.EmptyUncleHash,
			ReceiptHash:      types.EmptyReceiptsHash,
			TxHash:           types.EmptyTxsHash,
			Coinbase:         header.Coinbase,
			Difficulty:       header.Difficulty,
			GasLimit:         header.GasLimit,
			WithdrawalsHash:  withdrawalsHash,
			ParentBeaconRoot: parentBeaconRoot,
		})
		res[bi] = header
	}
	return res, nil
}

func (sim *simulator) newSimulatedChainContext(ctx context.Context, headers []*types.Header) *ChainContext {
	return NewChainContext(ctx, &simBackend{base: sim.base, b: sim.b, headers: headers})
}

type simBackend struct {
	b       ChainContextBackend
	base    *types.Header
	headers []*types.Header
}

func (b *simBackend) Engine() consensus.Engine {
	return b.b.Engine()
}

func (b *simBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if uint64(number) == b.base.Number.Uint64() {
		return b.base, nil
	}
	if uint64(number) < b.base.Number.Uint64() {
		// Resolve canonical header.
		return b.b.HeaderByNumber(ctx, number)
	}
	// Simulated block.
	for _, header := range b.headers {
		if header.Number.Uint64() == uint64(number) {
			return header, nil
		}
	}
	return nil, errors.New("header not found")
}
