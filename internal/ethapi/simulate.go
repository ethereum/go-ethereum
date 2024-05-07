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
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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

type simBlockResult struct {
	Number        hexutil.Uint64    `json:"number"`
	Hash          common.Hash       `json:"hash"`
	Time          hexutil.Uint64    `json:"timestamp"`
	GasLimit      hexutil.Uint64    `json:"gasLimit"`
	GasUsed       hexutil.Uint64    `json:"gasUsed"`
	FeeRecipient  common.Address    `json:"feeRecipient"`
	BaseFee       *hexutil.Big      `json:"baseFeePerGas"`
	PrevRandao    common.Hash       `json:"prevRandao"`
	Withdrawals   types.Withdrawals `json:"withdrawals"`
	BlobGasUsed   hexutil.Uint64    `json:"blobGasUsed"`
	ExcessBlobGas hexutil.Uint64    `json:"excessBlobGas"`
	Calls         []simCallResult   `json:"calls"`
}

func simBlockResultFromHeader(header *types.Header, callResults []simCallResult) simBlockResult {
	r := simBlockResult{
		Number:       hexutil.Uint64(header.Number.Uint64()),
		Hash:         header.Hash(),
		Time:         hexutil.Uint64(header.Time),
		GasLimit:     hexutil.Uint64(header.GasLimit),
		GasUsed:      hexutil.Uint64(header.GasUsed),
		FeeRecipient: header.Coinbase,
		BaseFee:      (*hexutil.Big)(header.BaseFee),
		PrevRandao:   header.MixDigest,
		// Withdrawals will be always empty in the context of a simulated block.
		Withdrawals: make(types.Withdrawals, 0),
		Calls:       callResults,
	}
	if header.BlobGasUsed != nil && header.ExcessBlobGas != nil {
		r.BlobGasUsed = hexutil.Uint64(*header.BlobGasUsed)
		r.ExcessBlobGas = hexutil.Uint64(*header.ExcessBlobGas)
	}
	return r
}

// repairLogs updates the block hash in the logs present in the result of
// a simulated block. This is needed as during execution when logs are collected
// the block hash is not known.
func (b *simBlockResult) repairLogs() {
	for i := range b.Calls {
		for j := range b.Calls[i].Logs {
			b.Calls[i].Logs[j].BlockHash = b.Hash
		}
	}
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
	BlockStateCalls []simBlock
	TraceTransfers  bool
	Validation      bool
}

type simulator struct {
	b              Backend
	hashes         []common.Hash
	state          *state.StateDB
	base           *types.Header
	traceTransfers bool
	validate       bool
}

func (sim *simulator) execute(ctx context.Context, blocks []simBlock) ([]simBlockResult, error) {
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
		results = make([]simBlockResult, len(blocks))
		// Each tx and all the series of txes shouldn't consume more gas than cap
		gp          = new(core.GasPool).AddGas(sim.b.RPCGasCap())
		precompiles = sim.activePrecompiles(ctx, sim.base)
		numHashes   = headers[len(headers)-1].Number.Uint64() - sim.base.Number.Uint64() + 256
	)
	// Cache for the block hashes.
	sim.hashes = make([]common.Hash, numHashes)
	for bi, block := range blocks {
		result, err := sim.processBlock(ctx, &block, headers[bi], headers, bi, gp, precompiles, timeout)
		if err != nil {
			return nil, err
		}
		results[bi] = *result
	}
	return results, nil
}

func (sim *simulator) processBlock(ctx context.Context, block *simBlock, header *types.Header, headers []*types.Header, blockIndex int, gp *core.GasPool, precompiles vm.PrecompiledContracts, timeout time.Duration) (*simBlockResult, error) {
	blockContext := core.NewEVMBlockContext(header, NewChainContext(ctx, sim.b), nil)
	if block.BlockOverrides != nil && block.BlockOverrides.BlobBaseFee != nil {
		blockContext.BlobBaseFee = block.BlockOverrides.BlobBaseFee.ToInt()
	}
	// Respond to BLOCKHASH requests.
	blockContext.GetHash = func(n uint64) common.Hash {
		h, err := sim.getBlockHash(ctx, n, sim.base, headers)
		if err != nil {
			log.Warn(err.Error())
			return common.Hash{}
		}
		return h
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
		config               = sim.b.ChainConfig()
		parent               = sim.base
		vmConfig             = &vm.Config{
			NoBaseFee: !sim.validate,
			// Block hash will be repaired after execution.
			Tracer: tracer.Hooks(),
		}
		evm = vm.NewEVM(blockContext, vm.TxContext{GasPrice: new(big.Int)}, sim.state, config, *vmConfig)
	)
	if blockIndex > 0 {
		parent = headers[blockIndex-1]
	}
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
		tx := call.ToTransaction(call.GasPrice == nil)
		txes[i] = tx

		if err := call.CallDefaults(gp.Gas(), header.BaseFee, config.ChainID); err != nil {
			return nil, err
		}
		msg := call.ToMessage(header.BaseFee, !sim.validate)
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
		// If the result contains a revert reason, try to unpack it.
		if len(result.Revert()) > 0 {
			result.Err = newRevertError(result.Revert())
		}
		logs := tracer.Logs()
		callRes := simCallResult{ReturnValue: result.Return(), Logs: logs, GasUsed: hexutil.Uint64(result.UsedGas)}
		if result.Failed() {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusFailed)
			if errors.Is(result.Err, vm.ErrExecutionReverted) {
				callRes.Error = &callError{Message: result.Err.Error(), Code: errCodeReverted}
			} else {
				callRes.Error = &callError{Message: result.Err.Error(), Code: errCodeVMError}
			}
		} else {
			callRes.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
		}
		callResults[i] = callRes
	}
	var (
		parentHash common.Hash
		err        error
	)
	parentHash, err = sim.getBlockHash(ctx, header.Number.Uint64()-1, sim.base, headers)
	if err != nil {
		return nil, err
	}
	header.ParentHash = parentHash
	header.Root = sim.state.IntermediateRoot(true)
	header.GasUsed = gasUsed
	if len(txes) > 0 {
		header.TxHash = types.DeriveSha(types.Transactions(txes), trie.NewStackTrie(nil))
	}
	if len(receipts) > 0 {
		header.ReceiptHash = types.DeriveSha(types.Receipts(receipts), trie.NewStackTrie(nil))
		header.Bloom = types.CreateBloom(types.Receipts(receipts))
	}
	if config.IsLondon(header.Number) {
		// BaseFee depends on parent's gasUsed, hence can't be pre-computed.
		// Phantom blocks are ignored.
		header.BaseFee = eip1559.CalcBaseFee(config, parent)
	}
	if config.IsCancun(header.Number, header.Time) {
		header.BlobGasUsed = &blobGasUsed
		var excess uint64
		if config.IsCancun(parent.Number, parent.Time) {
			excess = eip4844.CalcExcessBlobGas(*parent.ExcessBlobGas, *parent.BlobGasUsed)
		} else {
			excess = eip4844.CalcExcessBlobGas(0, 0)
		}
		header.ExcessBlobGas = &excess
	}
	result := simBlockResultFromHeader(header, callResults)
	result.repairLogs()
	return &result, nil
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
	// TODO: check chainID and against current header for london fees
	if call.GasPrice == nil && call.MaxFeePerGas == nil && call.MaxPriorityFeePerGas == nil {
		call.MaxFeePerGas = (*hexutil.Big)(big.NewInt(0))
		call.MaxPriorityFeePerGas = (*hexutil.Big)(big.NewInt(0))
	}
	return nil
}

// getBlockHash returns the hash for the block of the given number. Block can be
// part of the canonical chain, a simulated block or a phantom block.
// Note getBlockHash assumes `n` is smaller than the last already simulated block
// and smaller than the last block to be simulated.
func (sim *simulator) getBlockHash(ctx context.Context, n uint64, base *types.Header, headers []*types.Header) (common.Hash, error) {
	// getIndex returns the index of the hash in the hashes cache.
	// The cache potentially includes 255 blocks prior to the base.
	getIndex := func(n uint64) int {
		first := base.Number.Uint64() - 255
		return int(n - first)
	}
	index := getIndex(n)
	if h := sim.hashes[index]; h != (common.Hash{}) {
		return h, nil
	}
	h, err := sim.computeBlockHash(ctx, n, base, headers)
	if err != nil {
		return common.Hash{}, err
	}
	if h != (common.Hash{}) {
		sim.hashes[index] = h
	}
	return h, nil
}

func (sim *simulator) computeBlockHash(ctx context.Context, n uint64, base *types.Header, headers []*types.Header) (common.Hash, error) {
	if n == base.Number.Uint64() {
		return base.Hash(), nil
	} else if n < base.Number.Uint64() {
		h, err := sim.b.HeaderByNumber(ctx, rpc.BlockNumber(n))
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to load block hash for number %d. Err: %v\n", n, err)
		}
		return h.Hash(), nil
	}
	h := base
	for i := range headers {
		tmp := headers[i]
		// BLOCKHASH will only allow numbers prior to current block
		// so no need to check that condition.
		if tmp.Number.Uint64() == n {
			hash := tmp.Hash()
			return hash, nil
		} else if tmp.Number.Uint64() > n {
			// Phantom block.
			lastNonPhantomHash, err := sim.getBlockHash(ctx, h.Number.Uint64(), base, headers)
			if err != nil {
				return common.Hash{}, err
			}
			// keccak(rlp(lastNonPhantomBlockHash, blockNumber))
			hashData, err := rlp.EncodeToBytes([][]byte{lastNonPhantomHash.Bytes(), big.NewInt(int64(n)).Bytes()})
			if err != nil {
				return common.Hash{}, err
			}
			return crypto.Keccak256Hash(hashData), nil
		}
		h = tmp
	}
	return common.Hash{}, errors.New("requested block is in future")
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
	res := make([]*types.Header, len(blocks))
	var (
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

		var baseFee *big.Int
		if config.IsLondon(overrides.Number.ToInt()) {
			baseFee = eip1559.CalcBaseFee(config, header)
		}
		header = overrides.MakeHeader(&types.Header{
			UncleHash:   types.EmptyUncleHash,
			ReceiptHash: types.EmptyReceiptsHash,
			TxHash:      types.EmptyTxsHash,
			Coinbase:    base.Coinbase,
			Difficulty:  base.Difficulty,
			GasLimit:    base.GasLimit,
			//MixDigest:  header.MixDigest,
			BaseFee: baseFee,
		})
		res[bi] = header
	}
	return res, nil
}
