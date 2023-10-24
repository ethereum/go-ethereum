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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// maxMulticallBlocks is the maximum number of blocks that can be simulated
	// in a single request.
	maxMulticallBlocks = 256
)

// mcBlock is a batch of calls to be simulated sequentially.
type mcBlock struct {
	BlockOverrides *BlockOverrides
	StateOverrides *StateOverride
	Calls          []TransactionArgs
}

type mcBlockResult struct {
	Number       hexutil.Uint64 `json:"number"`
	Hash         common.Hash    `json:"hash"`
	Time         hexutil.Uint64 `json:"timestamp"`
	GasLimit     hexutil.Uint64 `json:"gasLimit"`
	GasUsed      hexutil.Uint64 `json:"gasUsed"`
	FeeRecipient common.Address `json:"feeRecipient"`
	BaseFee      *hexutil.Big   `json:"baseFeePerGas"`
	PrevRandao   common.Hash    `json:"prevRandao"`
	Calls        []mcCallResult `json:"calls"`
}

func mcBlockResultFromHeader(header *types.Header, callResults []mcCallResult) mcBlockResult {
	return mcBlockResult{
		Number:       hexutil.Uint64(header.Number.Uint64()),
		Hash:         header.Hash(),
		Time:         hexutil.Uint64(header.Time),
		GasLimit:     hexutil.Uint64(header.GasLimit),
		GasUsed:      hexutil.Uint64(header.GasUsed),
		FeeRecipient: header.Coinbase,
		BaseFee:      (*hexutil.Big)(header.BaseFee),
		PrevRandao:   header.MixDigest,
		Calls:        callResults,
	}
}

type mcCallResult struct {
	ReturnValue hexutil.Bytes  `json:"returnData"`
	Logs        []*types.Log   `json:"logs"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Status      hexutil.Uint64 `json:"status"`
	Error       *callError     `json:"error,omitempty"`
}

func (r *mcCallResult) MarshalJSON() ([]byte, error) {
	type callResultAlias mcCallResult
	// Marshal logs to be an empty array instead of nil when empty
	if r.Logs == nil {
		r.Logs = []*types.Log{}
	}
	return json.Marshal((*callResultAlias)(r))
}

type mcOpts struct {
	BlockStateCalls []mcBlock
	TraceTransfers  bool
	Validation      bool
}

type multicall struct {
	blockNrOrHash rpc.BlockNumberOrHash
	b             Backend
	hashes        []common.Hash
}

func (mc *multicall) execute(ctx context.Context, opts mcOpts) ([]mcBlockResult, error) {
	state, base, err := mc.b.StateAndHeaderByNumberOrHash(ctx, mc.blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	// Setup context so it may be cancelled before the calls completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var (
		cancel  context.CancelFunc
		timeout = mc.b.RPCEVMTimeout()
		blocks  = opts.BlockStateCalls
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	headers, err := makeHeaders(mc.b.ChainConfig(), blocks, base)
	if err != nil {
		return nil, err
	}
	var (
		results = make([]mcBlockResult, len(blocks))
		// Each tx and all the series of txes shouldn't consume more gas than cap
		gp          = new(core.GasPool).AddGas(mc.b.RPCGasCap())
		precompiles = mc.activePrecompiles(ctx, base)
		numHashes   = headers[len(headers)-1].Number.Uint64() - base.Number.Uint64() + 256
	)
	// Cache for the block hashes.
	mc.hashes = make([]common.Hash, numHashes)
	for bi, block := range blocks {
		header := headers[bi]
		blockContext := core.NewEVMBlockContext(header, NewChainContext(ctx, mc.b), nil)
		// Respond to BLOCKHASH requests.
		blockContext.GetHash = func(n uint64) common.Hash {
			h, err := mc.getBlockHash(ctx, n, base, headers)
			if err != nil {
				log.Warn(err.Error())
				return common.Hash{}
			}
			return h
		}
		// State overrides are applied prior to execution of a block
		if err := block.StateOverrides.Apply(state, precompiles); err != nil {
			return nil, err
		}
		var (
			gasUsed     uint64
			txes        = make([]*types.Transaction, len(block.Calls))
			callResults = make([]mcCallResult, len(block.Calls))
		)
		for i, call := range block.Calls {
			if call.Nonce == nil {
				nonce := state.GetNonce(call.from())
				call.Nonce = (*hexutil.Uint64)(&nonce)
			}
			// Let the call run wild unless explicitly specified.
			if call.Gas == nil {
				remaining := blockContext.GasLimit - gasUsed
				call.Gas = (*hexutil.Uint64)(&remaining)
			}
			if call.GasPrice == nil && call.MaxFeePerGas == nil && call.MaxPriorityFeePerGas == nil {
				call.MaxFeePerGas = (*hexutil.Big)(big.NewInt(0))
				call.MaxPriorityFeePerGas = (*hexutil.Big)(big.NewInt(0))
			}
			// TODO: check chainID and against current header for london fees
			if err := call.validateAll(); err != nil {
				return nil, err
			}
			tx := call.ToTransaction(true)
			txes[i] = tx
			// TODO: repair log block hashes post execution.
			vmConfig := &vm.Config{
				NoBaseFee: true,
				// Block hash will be repaired after execution.
				Tracer: newTracer(opts.TraceTransfers, blockContext.BlockNumber.Uint64(), common.Hash{}, tx.Hash(), uint(i)),
			}
			result, err := applyMessage(ctx, mc.b, call, state, header, timeout, gp, &blockContext, vmConfig, precompiles, opts.Validation)
			if err != nil {
				callErr := callErrorFromError(err)
				callResults[i] = mcCallResult{Error: callErr, Status: hexutil.Uint64(types.ReceiptStatusFailed)}
				continue
			}
			// If the result contains a revert reason, try to unpack it.
			if len(result.Revert()) > 0 {
				result.Err = newRevertError(result)
			}
			logs := vmConfig.Tracer.(*tracer).Logs()
			callRes := mcCallResult{ReturnValue: result.Return(), Logs: logs, GasUsed: hexutil.Uint64(result.UsedGas)}
			if result.Failed() {
				callRes.Status = hexutil.Uint64(types.ReceiptStatusFailed)
				if errors.Is(result.Err, vm.ErrExecutionReverted) {
					callRes.Error = &callError{Message: result.Err.Error(), Code: -32000}
				} else {
					callRes.Error = &callError{Message: result.Err.Error(), Code: -32015}
				}
			} else {
				callRes.Status = hexutil.Uint64(types.ReceiptStatusSuccessful)
			}
			callResults[i] = callRes
			gasUsed += result.UsedGas
			state.Finalise(true)
		}
		var (
			parentHash common.Hash
			err        error
		)
		parentHash, err = mc.getBlockHash(ctx, header.Number.Uint64()-1, base, headers)
		if err != nil {
			return nil, err
		}
		header.ParentHash = parentHash
		header.Root = state.IntermediateRoot(true)
		header.GasUsed = gasUsed
		header.TxHash = types.DeriveSha(types.Transactions(txes), trie.NewStackTrie(nil))
		results[bi] = mcBlockResultFromHeader(header, callResults)
		repairLogs(results, header.Hash())
	}
	return results, nil
}

// getBlockHash returns the hash for the block of the given number. Block can be
// part of the canonical chain, a simulated block or a phantom block.
// Note getBlockHash assumes `n` is smaller than the last already simulated block
// and smaller than the last block to be simulated.
func (mc *multicall) getBlockHash(ctx context.Context, n uint64, base *types.Header, headers []*types.Header) (common.Hash, error) {
	// getIndex returns the index of the hash in the hashes cache.
	// The cache potentially includes 255 blocks prior to the base.
	getIndex := func(n uint64) int {
		first := base.Number.Uint64() - 255
		return int(n - first)
	}
	index := getIndex(n)
	if h := mc.hashes[index]; h != (common.Hash{}) {
		return h, nil
	}
	h, err := mc.computeBlockHash(ctx, n, base, headers)
	if err != nil {
		return common.Hash{}, err
	}
	if h != (common.Hash{}) {
		mc.hashes[index] = h
	}
	return h, nil
}

func (mc *multicall) computeBlockHash(ctx context.Context, n uint64, base *types.Header, headers []*types.Header) (common.Hash, error) {
	if n == base.Number.Uint64() {
		return base.Hash(), nil
	} else if n < base.Number.Uint64() {
		h, err := mc.b.HeaderByNumber(ctx, rpc.BlockNumber(n))
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to load block hash for number %d. Err: %v\n", n, err)
		}
		return h.Hash(), nil
	}
	h := base
	for i, _ := range headers {
		tmp := headers[i]
		// BLOCKHASH will only allow numbers prior to current block
		// so no need to check that condition.
		if tmp.Number.Uint64() == n {
			hash := tmp.Hash()
			return hash, nil
		} else if tmp.Number.Uint64() > n {
			// Phantom block.
			lastNonPhantomHash, err := mc.getBlockHash(ctx, h.Number.Uint64(), base, headers)
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

func (mc *multicall) activePrecompiles(ctx context.Context, base *types.Header) vm.PrecompiledContracts {
	var (
		blockContext = core.NewEVMBlockContext(base, NewChainContext(ctx, mc.b), nil)
		rules        = mc.b.ChainConfig().Rules(blockContext.BlockNumber, blockContext.Random != nil, blockContext.Time)
	)
	return vm.ActivePrecompiledContracts(rules).Copy()
}

// repairLogs updates the block hash in the logs present in a multicall
// result object. This is needed as during execution when logs are collected
// the block hash is not known.
func repairLogs(results []mcBlockResult, blockHash common.Hash) {
	for i := range results {
		for j := range results[i].Calls {
			for k := range results[i].Calls[j].Logs {
				results[i].Calls[j].Logs[k].BlockHash = blockHash
			}
		}
	}
}
func makeHeaders(config *params.ChainConfig, blocks []mcBlock, base *types.Header) ([]*types.Header, error) {
	res := make([]*types.Header, len(blocks))
	var (
		prevNumber    = base.Number.Uint64()
		prevTimestamp = base.Time
		header        = base
	)
	for bi, block := range blocks {
		overrides := new(BlockOverrides)
		if block.BlockOverrides != nil {
			overrides = block.BlockOverrides
		}
		// Sanitize block number and timestamp
		if overrides.Number == nil {
			n := new(big.Int).Add(big.NewInt(int64(prevNumber)), big.NewInt(1))
			overrides.Number = (*hexutil.Big)(n)
		} else if overrides.Number.ToInt().Uint64() <= prevNumber {
			return nil, fmt.Errorf("block numbers must be in order")
		}
		prevNumber = overrides.Number.ToInt().Uint64()

		if overrides.Time == nil {
			t := prevTimestamp + 1
			overrides.Time = (*hexutil.Uint64)(&t)
		} else if time := (*uint64)(overrides.Time); *time <= prevTimestamp {
			return nil, fmt.Errorf("timestamps must be in order")
		}
		prevTimestamp = uint64(*overrides.Time)

		var baseFee *big.Int
		if config.IsLondon(overrides.Number.ToInt()) {
			baseFee = eip1559.CalcBaseFee(config, header)
		}
		header = &types.Header{
			UncleHash:  types.EmptyUncleHash,
			Coinbase:   base.Coinbase,
			Difficulty: base.Difficulty,
			GasLimit:   base.GasLimit,
			//MixDigest:  header.MixDigest,
			BaseFee: baseFee,
		}
		overrides.ApplyToHeader(header)
		res[bi] = header
	}
	return res, nil
}
