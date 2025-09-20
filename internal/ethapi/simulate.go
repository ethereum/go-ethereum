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
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// maxSimulateBlocks is the maximum number of blocks that can be simulated
	// in a single request.
	maxSimulateBlocks = 256

	// timestampIncrement is the default increment between block timestamps.
	timestampIncrement = 12
)

// simBlock is a batch of calls to be simulated sequentially.
type simBlock struct {
	BlockOverrides *override.BlockOverrides
	StateOverrides *override.StateOverride
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

// simBlockResult is the result of a simulated block.
type simBlockResult struct {
	fullTx      bool
	chainConfig *params.ChainConfig
	Block       *types.Block
	Calls       []simCallResult
	// senders is a map of transaction hashes to their senders.
	senders map[common.Hash]common.Address
}

func (r *simBlockResult) MarshalJSON() ([]byte, error) {
	blockData := RPCMarshalBlock(r.Block, true, r.fullTx, r.chainConfig)
	blockData["calls"] = r.Calls
	// Set tx sender if user requested full tx objects.
	if r.fullTx {
		if raw, ok := blockData["transactions"].([]any); ok {
			for _, tx := range raw {
				if tx, ok := tx.(*RPCTransaction); ok {
					tx.From = r.senders[tx.Hash]
				} else {
					return nil, errors.New("simulated transaction result has invalid type")
				}
			}
		}
	}
	return json.Marshal(blockData)
}

// simOpts are the inputs to eth_simulateV1.
type simOpts struct {
	BlockStateCalls        []simBlock
	TraceTransfers         bool
	Validation             bool
	ReturnFullTransactions bool
}

// simChainHeadReader implements ChainHeaderReader which is needed as input for FinalizeAndAssemble.
type simChainHeadReader struct {
	context.Context
	Backend
}

func (m *simChainHeadReader) Config() *params.ChainConfig {
	return m.Backend.ChainConfig()
}

func (m *simChainHeadReader) CurrentHeader() *types.Header {
	return m.Backend.CurrentHeader()
}

func (m *simChainHeadReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	header, err := m.Backend.HeaderByNumber(m.Context, rpc.BlockNumber(number))
	if err != nil || header == nil {
		return nil
	}
	if header.Hash() != hash {
		return nil
	}
	return header
}

func (m *simChainHeadReader) GetHeaderByNumber(number uint64) *types.Header {
	header, err := m.Backend.HeaderByNumber(m.Context, rpc.BlockNumber(number))
	if err != nil {
		return nil
	}
	return header
}

func (m *simChainHeadReader) GetHeaderByHash(hash common.Hash) *types.Header {
	header, err := m.Backend.HeaderByHash(m.Context, hash)
	if err != nil {
		return nil
	}
	return header
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
func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]*simBlockResult, error) {
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
		results = make([]*simBlockResult, len(blocks))
		parent  = sim.base
	)
	for bi, block := range blocks {
		result, callResults, senders, err := sim.processBlock(ctx, &block, headers[bi], parent, headers[:bi], timeout)
		if err != nil {
			return nil, err
		}
		headers[bi] = result.Header()
		results[bi] = &simBlockResult{fullTx: sim.fullTx, chainConfig: sim.chainConfig, Block: result, Calls: callResults, senders: senders}
		parent = result.Header()
	}
	return results, nil
}

func (sim *simulator) processBlock(ctx context.Context, block *simBlock, header, parent *types.Header, headers []*types.Header, timeout time.Duration) (*types.Block, []simCallResult, map[common.Hash]common.Address, error) {
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
			excess = eip4844.CalcExcessBlobGas(sim.chainConfig, parent, header.Time)
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
		return nil, nil, nil, err
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
		// senders is a map of transaction hashes to their senders.
		// Transaction objects contain only the signature, and we lose track
		// of the sender when translating the arguments into a transaction object.
		senders = make(map[common.Hash]common.Address)
	)
	tracingStateDB := vm.StateDB(sim.state)
	if hooks := tracer.Hooks(); hooks != nil {
		tracingStateDB = state.NewHookedState(sim.state, hooks)
	}
	evm := vm.NewEVM(blockContext, tracingStateDB, sim.chainConfig, *vmConfig)
	// It is possible to override precompiles with EVM bytecode, or
	// move them to another address.
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	if sim.chainConfig.IsPrague(header.Number, header.Time) || sim.chainConfig.IsVerkle(header.Number, header.Time) {
		core.ProcessParentBlockHash(header.ParentHash, evm)
	}
	if header.ParentBeaconRoot != nil {
		core.ProcessBeaconBlockRoot(*header.ParentBeaconRoot, evm)
	}
	var allLogs []*types.Log
	for i, call := range block.Calls {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		if err := sim.sanitizeCall(&call, sim.state, header, blockContext, &gasUsed); err != nil {
			return nil, nil, nil, err
		}
		var (
			tx     = call.ToTransaction(types.DynamicFeeTxType)
			txHash = tx.Hash()
		)
		txes[i] = tx
		senders[txHash] = call.from()
		tracer.reset(txHash, uint(i))
		sim.state.SetTxContext(txHash, i)
		// EoA check is always skipped, even in validation mode.
		msg := call.ToMessage(header.BaseFee, !sim.validate)
		result, err := applyMessageWithEVM(ctx, evm, msg, timeout, sim.gp)
		if err != nil {
			txErr := txValidationError(err)
			return nil, nil, nil, txErr
		}
		// Update the state with pending changes.
		var root []byte
		if sim.chainConfig.IsByzantium(blockContext.BlockNumber) {
			tracingStateDB.Finalise(true)
		} else {
			root = sim.state.IntermediateRoot(sim.chainConfig.IsEIP158(blockContext.BlockNumber)).Bytes()
		}
		gasUsed += result.UsedGas
		receipts[i] = core.MakeReceipt(evm, result, sim.state, blockContext.BlockNumber, common.Hash{}, blockContext.Time, tx, gasUsed, root)
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
			allLogs = append(allLogs, callRes.Logs...)
		}
		callResults[i] = callRes
	}
	header.GasUsed = gasUsed
	if sim.chainConfig.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = &blobGasUsed
	}
	var requests [][]byte
	// Process EIP-7685 requests
	if sim.chainConfig.IsPrague(header.Number, header.Time) {
		requests = [][]byte{}
		// EIP-6110
		if err := core.ParseDepositLogs(&requests, allLogs, sim.chainConfig); err != nil {
			return nil, nil, nil, err
		}
		// EIP-7002
		if err := core.ProcessWithdrawalQueue(&requests, evm); err != nil {
			return nil, nil, nil, err
		}
		// EIP-7251
		if err := core.ProcessConsolidationQueue(&requests, evm); err != nil {
			return nil, nil, nil, err
		}
	}
	if requests != nil {
		reqHash := types.CalcRequestsHash(requests)
		header.RequestsHash = &reqHash
	}
	blockBody := &types.Body{Transactions: txes, Withdrawals: *block.BlockOverrides.Withdrawals}
	chainHeadReader := &simChainHeadReader{ctx, sim.b}
	b, err := sim.b.Engine().FinalizeAndAssemble(chainHeadReader, header, sim.state, blockBody, receipts)
	if err != nil {
		return nil, nil, nil, err
	}
	repairLogs(callResults, b.Hash())
	return b, callResults, senders, nil
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

func (sim *simulator) sanitizeCall(call *TransactionArgs, state vm.StateDB, header *types.Header, blockContext vm.BlockContext, gasUsed *uint64) error {
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
	return vm.ActivePrecompiledContracts(rules)
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
			block.BlockOverrides = new(override.BlockOverrides)
		}
		if block.BlockOverrides.Number == nil {
			n := new(big.Int).Add(prevNumber, big.NewInt(1))
			block.BlockOverrides.Number = (*hexutil.Big)(n)
		}
		if block.BlockOverrides.Withdrawals == nil {
			block.BlockOverrides.Withdrawals = &types.Withdrawals{}
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
				b := simBlock{
					BlockOverrides: &override.BlockOverrides{
						Number:      (*hexutil.Big)(n),
						Time:        (*hexutil.Uint64)(&t),
						Withdrawals: &types.Withdrawals{},
					},
				}
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
		number := overrides.Number.ToInt()
		timestamp := (uint64)(*overrides.Time)
		if sim.chainConfig.IsShanghai(number, timestamp) {
			withdrawalsHash = &types.EmptyWithdrawalsHash
		}
		var parentBeaconRoot *common.Hash
		if sim.chainConfig.IsCancun(number, timestamp) {
			parentBeaconRoot = &common.Hash{}
			if overrides.BeaconRoot != nil {
				parentBeaconRoot = overrides.BeaconRoot
			}
		}
		// Set difficulty to zero if the given block is post-merge. Without this, all post-merge hardforks would remain inactive.
		// For example, calling eth_simulateV1(..., blockParameter: 0x0) on hoodi network will cause all blocks to have a difficulty of 1 and be treated as pre-merge.
		difficulty := header.Difficulty
		if sim.chainConfig.IsPostMerge(number.Uint64(), timestamp) {
			difficulty = big.NewInt(0)
		}
		header = overrides.MakeHeader(&types.Header{
			UncleHash:        types.EmptyUncleHash,
			ReceiptHash:      types.EmptyReceiptsHash,
			TxHash:           types.EmptyTxsHash,
			Coinbase:         header.Coinbase,
			Difficulty:       difficulty,
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

func (b *simBackend) ChainConfig() *params.ChainConfig {
	return b.b.ChainConfig()
}
