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
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// maxSimulateBlocks is the maximum number of blocks that can be simulated
	// in a single request.
	maxSimulateBlocks = 256
)

// simBlock is a batch of calls to be simulated sequentially.
type simBlock struct {
	BlockOverrides *BlockOverrides
	StateOverrides *StateOverride
	Calls          []TransactionArgs
}

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

type simOpts struct {
	BlockStateCalls        []simBlock
	TraceTransfers         bool
	Validation             bool
	ReturnFullTransactions bool
}

type simulator struct {
	b              Backend
	hashes         []common.Hash
	state          *state.StateDB
	base           *types.Header
	traceTransfers bool
	validate       bool
	fullTx         bool
}

func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]map[string]interface{}, error) {
	// Setup context so it may be cancelled before the calls completed
	// or, in case of unmetered gas, setup a context with a timeout.
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
	blocks, err = sim.sanitizeBlockOrder(blocks)
	if err != nil {
		return nil, err
	}
	headers, err := sim.makeHeaders(blocks)
	if err != nil {
		return nil, err
	}
	var (
		results = make([]map[string]interface{}, len(blocks))
		// Each tx and all the series of txes shouldn't consume more gas than cap
		gp          = new(core.GasPool).AddGas(sim.b.RPCGasCap())
		precompiles = sim.activePrecompiles(ctx, sim.base)
		numHashes   = headers[len(headers)-1].Number.Uint64() - sim.base.Number.Uint64() + 256
		parent      = sim.base
	)
	// Cache for the block hashes.
	sim.hashes = make([]common.Hash, numHashes)
	for bi, block := range blocks {
		result, err := sim.processBlock(ctx, &block, headers[bi], parent, headers[:bi], gp, precompiles, timeout)
		if err != nil {
			return nil, err
		}
		results[bi] = result
		parent = headers[bi]
	}
	return results, nil
}

func (sim *simulator) processBlock(ctx context.Context, block *simBlock, header, parent *types.Header, headers []*types.Header, gp *core.GasPool, precompiles vm.PrecompiledContracts, timeout time.Duration) (map[string]interface{}, error) {
	// Set header fields that depend only on parent block.
	config := sim.b.ChainConfig()
	// Parent hash is needed for evm.GetHashFn to work.
	header.ParentHash = parent.Hash()
	if config.IsLondon(header.Number) {
		// In non-validation mode base fee is set to 0 if it is not overridden.
		// This is because it creates an edge case in EVM where gasPrice < baseFee.
		// Base fee could have been overridden.
		if header.BaseFee == nil {
			if sim.validate {
				header.BaseFee = eip1559.CalcBaseFee(config, parent)
			} else {
				header.BaseFee = big.NewInt(0)
			}
		}
	}
	if config.IsCancun(header.Number, header.Time) {
		var excess uint64
		if config.IsCancun(parent.Number, parent.Time) {
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
	// State overrides are applied prior to execution of a block
	if err := block.StateOverrides.Apply(sim.state, precompiles); err != nil {
		return nil, err
	}
	var (
		gasUsed, blobGasUsed uint64
		txes                 = make([]*types.Transaction, len(block.Calls))
		callResults          = make([]simCallResult, len(block.Calls))
		receipts             = make([]*types.Receipt, len(block.Calls))
		tracer               = newTracer(sim.traceTransfers, blockContext.BlockNumber.Uint64(), common.Hash{}, common.Hash{}, 0)
		vmConfig             = &vm.Config{
			NoBaseFee: !sim.validate,
			// Block hash will be repaired after execution.
			Tracer: tracer.Hooks(),
		}
		evm = vm.NewEVM(blockContext, vm.TxContext{GasPrice: new(big.Int)}, sim.state, config, *vmConfig)
	)
	sim.state.SetLogger(tracer.Hooks())
	// It is possible to override precompiles with EVM bytecode, or
	// move them to another address.
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	for i, call := range block.Calls {
		// TODO: Pre-estimate nonce and gas
		// TODO: Move gas fees sanitizing to beginning of func
		if err := sim.sanitizeCall(&call, sim.state, &gasUsed, blockContext); err != nil {
			return nil, err
		}
		if err := call.CallDefaults(gp.Gas(), header.BaseFee, config.ChainID); err != nil {
			return nil, err
		}
		tx := call.ToTransaction(call.GasPrice == nil)
		txes[i] = tx
		// EoA check is always skipped, even in validation mode.
		msg := call.ToMessage(header.BaseFee, !sim.validate, true)
		tracer.reset(tx.Hash(), uint(i))
		evm.Reset(core.NewEVMTxContext(msg), sim.state)
		result, err := applyMessageWithEVM(ctx, evm, msg, sim.state, timeout, gp)
		if err != nil {
			txErr := txValidationError(err)
			return nil, txErr
		}
		// Update the state with pending changes.
		var root []byte
		if config.IsByzantium(blockContext.BlockNumber) {
			sim.state.Finalise(true)
		} else {
			root = sim.state.IntermediateRoot(config.IsEIP158(blockContext.BlockNumber)).Bytes()
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
	if config.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = &blobGasUsed
	}
	var withdrawals types.Withdrawals
	if config.IsShanghai(header.Number, header.Time) {
		withdrawals = make([]*types.Withdrawal, 0)
	}
	b := types.NewBlock(header, &types.Body{Transactions: txes, Withdrawals: withdrawals}, receipts, trie.NewStackTrie(nil))
	res := RPCMarshalBlock(b, true, sim.fullTx, config)
	res["totalDifficulty"] = (*hexutil.Big)(sim.b.GetTd(ctx, sim.base.Hash()))
	repairLogs(callResults, res["hash"].(common.Hash))
	res["calls"] = callResults

	return res, nil
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

func (sim *simulator) sanitizeCall(call *TransactionArgs, state *state.StateDB, gasUsed *uint64, blockContext vm.BlockContext) error {
	if call.Nonce == nil {
		nonce := state.GetNonce(call.from())
		call.Nonce = (*hexutil.Uint64)(&nonce)
	}
	var gas uint64
	if call.Gas != nil {
		gas = uint64(*call.Gas)
	}
	if *gasUsed+gas > blockContext.GasLimit {
		return &blockGasLimitReachedError{fmt.Sprintf("block gas limit reached: %d >= %d", gasUsed, blockContext.GasLimit)}
	}
	// Let the call run wild unless explicitly specified.
	if call.Gas == nil {
		remaining := blockContext.GasLimit - *gasUsed
		call.Gas = (*hexutil.Uint64)(&remaining)
	}
	return nil
}

func (sim *simulator) activePrecompiles(ctx context.Context, base *types.Header) vm.PrecompiledContracts {
	var (
		blockContext = core.NewEVMBlockContext(base, NewChainContext(ctx, sim.b), nil)
		rules        = sim.b.ChainConfig().Rules(blockContext.BlockNumber, blockContext.Random != nil, blockContext.Time)
	)
	return vm.ActivePrecompiledContracts(rules).Copy()
}

// sanitizeBlockOrder iterates the blocks checking that block numbers
// are strictly increasing. When necessary it will generate empty blocks.
// It modifies the block's override object.
func (sim *simulator) sanitizeBlockOrder(blocks []simBlock) ([]simBlock, error) {
	var (
		res        = make([]simBlock, 0, len(blocks))
		base       = sim.base
		prevNumber = base.Number
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
				b := simBlock{BlockOverrides: &BlockOverrides{Number: (*hexutil.Big)(n)}}
				res = append(res, b)
			}
		}
		// Only append block after filling a potential gap.
		prevNumber = block.BlockOverrides.Number.ToInt()
		res = append(res, block)
	}
	return res, nil
}

// makeHeaders makes header object with preliminary fields based on a simulated block.
// Some fields have to be filled post-execution.
// It assumes blocks are in order and numbers have been validated.
func (sim *simulator) makeHeaders(blocks []simBlock) ([]*types.Header, error) {
	var (
		res           = make([]*types.Header, len(blocks))
		config        = sim.b.ChainConfig()
		base          = sim.base
		prevTimestamp = base.Time
		header        = base
	)
	for bi, block := range blocks {
		if block.BlockOverrides == nil || block.BlockOverrides.Number == nil {
			return nil, errors.New("empty block number")
		}
		overrides := block.BlockOverrides
		if overrides.Time == nil {
			t := prevTimestamp + 12
			overrides.Time = (*hexutil.Uint64)(&t)
		} else if time := (*uint64)(overrides.Time); *time <= prevTimestamp {
			return nil, &invalidBlockTimestampError{fmt.Sprintf("block timestamps must be in order: %d <= %d", *time, prevTimestamp)}
		}
		prevTimestamp = uint64(*overrides.Time)

		var withdrawalsHash *common.Hash
		if config.IsShanghai(overrides.Number.ToInt(), (uint64)(*overrides.Time)) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		var parentBeaconRoot *common.Hash
		if config.IsCancun(overrides.Number.ToInt(), (uint64)(*overrides.Time)) {
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
