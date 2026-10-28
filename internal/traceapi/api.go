// Copyright 2026 The go-ethereum Authors
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

// Package traceapi implements the trace RPC namespace.
package traceapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	filterBlockLimit  = 1000
	filterResultLimit = 10000
	callManyLimit     = 10000
	batchTimeout      = 30 * time.Second
	replayTimeout     = 5 * time.Second
)

// Backend provides the chain access of the trace namespace.
type Backend interface {
	tracers.Backend
	Pending() (*types.Block, types.Receipts, *state.StateDB)
	RPCEVMTimeout() time.Duration
}

// APIs returns the trace namespace service.
func APIs(backend Backend) []rpc.API {
	return []rpc.API{{Namespace: "trace", Service: NewAPI(backend)}}
}

// API implements the Parity trace namespace.
type API struct {
	backend Backend
}

// NewAPI constructs the trace namespace service.
func NewAPI(backend Backend) *API { return &API{backend: backend} }

// chainContext constructs the context reader which is used by the evm for reading
// the necessary chain context.
func (api *API) chainContext(ctx context.Context) core.ChainContext {
	return ethapi.NewChainContext(ctx, api.backend)
}

// blockByNumber is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *API) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := api.backend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// blockByHash is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *API) blockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := api.backend.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return block, nil
}

// blockByNumberAndHash is the wrapper of the chain access function offered by
// the backend. It will return an error if the block is not found.
func (api *API) blockByNumberAndHash(ctx context.Context, number rpc.BlockNumber, hash common.Hash) (*types.Block, error) {
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block.Hash() == hash {
		return block, nil
	}
	return api.blockByHash(ctx, hash)
}

// Call executes an unsigned call on the selected block's post-state.
func (api *API) Call(ctx context.Context, args CallArgs, kinds Outputs, block *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride, blockOverrides *override.BlockOverrides) (*Execution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	env, err := api.newCallEnv(ctx, block)
	if err != nil {
		return nil, err
	}
	defer env.release()
	if err := env.applyOverrides(api.backend.ChainConfig(), stateOverrides, blockOverrides); err != nil {
		return nil, err
	}
	return api.call(ctx, args, kinds, env, 0)
}

// CallMany executes calls sequentially, retaining successful writes in the sequence.
func (api *API) CallMany(ctx context.Context, calls CallManyArgs, block *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride, blockOverrides *override.BlockOverrides) ([]*Execution, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	if len(calls) > callManyLimit {
		return nil, limitExceeded("too many calls (limit %d)", callManyLimit)
	}
	for _, call := range calls {
		if err := call.Types.validate(); err != nil {
			return nil, err
		}
	}
	env, err := api.newCallEnv(ctx, block)
	if err != nil {
		return nil, err
	}
	defer env.release()
	if err := env.applyOverrides(api.backend.ChainConfig(), stateOverrides, blockOverrides); err != nil {
		return nil, err
	}
	results := make([]*Execution, 0, len(calls))
	for i, call := range calls {
		// Each item is a separate transaction on the preceding post-state.
		// A failing item fails the whole request, identified by its index.
		result, err := api.call(ctx, call.Call, call.Types, env, i)
		if err != nil {
			var rpcErr rpc.Error
			if errors.As(err, &rpcErr) {
				return nil, &itemError{rpcErr, i}
			}
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// RawTransaction validates and executes the supplied signed transaction at latest state.
func (api *API) RawTransaction(ctx context.Context, input hexutil.Bytes, kinds Outputs) (*Execution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return nil, invalidParams("invalid transaction: %v", err)
	}
	block, err := api.block(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, err
	}
	st, release, err := api.backend.StateAtBlock(ctx, block, nil, true, false)
	if err != nil {
		return nil, stateError(err)
	}
	defer release()
	if err := api.checkGasCap(tx.Gas()); err != nil {
		return nil, err
	}
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
	if err != nil {
		return nil, rawRejection(fmt.Errorf("invalid signed transaction: %w", err))
	}
	vmctx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
	return api.execute(ctx, api.backend.RPCEVMTimeout(), tx, msg, kinds, vmctx, st, nil, common.Hash{}, 0, false, rawRejection)
}

// ReplayTransaction returns one replay envelope or null for an unknown transaction.
func (api *API) ReplayTransaction(ctx context.Context, hash common.Hash, kinds Outputs) (*Execution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	result, _, _, err := api.transaction(ctx, hash, kinds)
	return result, err
}

// ReplayBlockTransactions returns one envelope per transaction in block order.
func (api *API) ReplayBlockTransactions(ctx context.Context, number rpc.BlockNumber, kinds Outputs) ([]*Execution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	block, err := api.block(ctx, number)
	if err != nil {
		return nil, err
	}
	return api.replayBlock(ctx, block, kinds)
}

// Transaction returns the localized preorder tree for a mined transaction.
func (api *API) Transaction(ctx context.Context, hash common.Hash) ([]*TraceFrame, error) {
	result, block, index, err := api.transaction(ctx, hash, Outputs{"trace"})
	if err != nil || result == nil {
		return nil, err
	}
	localizeTrace(result.Trace, block, &hash, &index)
	return result.Trace, nil
}

// Get returns one tree path, with the empty path selecting the root.
func (api *API) Get(ctx context.Context, hash common.Hash, position Path) (*TraceFrame, error) {
	frames, err := api.Transaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	for _, frame := range frames {
		if len(frame.TraceAddress) != len(position) {
			continue
		}
		match := true
		for i, n := range position {
			if frame.TraceAddress[i] != uint64(n) {
				match = false
				break
			}
		}
		if match {
			return frame, nil
		}
	}
	return nil, nil
}

// Block returns localized transaction traces followed by historical PoW rewards.
func (api *API) Block(ctx context.Context, number rpc.BlockNumber) ([]*TraceFrame, error) {
	block, err := api.block(ctx, number)
	if err != nil {
		return nil, err
	}
	return api.blockTraces(ctx, block)
}

// Filter scans a bounded canonical snapshot, applying pagination after matching.
func (api *API) Filter(ctx context.Context, filter FilterArgs) ([]*TraceFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	// Resolve both bounds against one head. As for eth_getLogs, a bound beyond
	// the head or a reversed range is invalid rather than clamped.
	// The header chain can be ahead of execution during sync. Resolve latest
	// against the executed head, as the other trace methods and eth_* do.
	head, err := api.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, err
	}
	if head == nil {
		return nil, notFound("latest block not found")
	}
	resolve := func(number *rpc.BlockNumber) (*types.Header, error) {
		n := rpc.LatestBlockNumber
		if number != nil {
			n = *number
		}
		switch {
		case n == rpc.PendingBlockNumber:
			return nil, invalidParams("pending is not a valid trace_filter bound")
		case n == rpc.LatestBlockNumber:
			return head, nil
		case n >= 0 && uint64(n) > head.Number.Uint64():
			return nil, invalidParams("block %d is beyond the current head %d", n, head.Number.Uint64())
		}
		h, err := api.backend.HeaderByNumber(ctx, n)
		if errors.Is(err, ethapi.ErrSafeBlockNotFound) || errors.Is(err, ethapi.ErrFinalizedBlockNotFound) || (err == nil && h == nil) {
			return nil, invalidParams("block %s not found", n)
		}
		if err != nil {
			return nil, err
		}
		if h.Number.Cmp(head.Number) > 0 {
			return nil, invalidParams("block %s is beyond the current head %d", n, head.Number.Uint64())
		}
		return h, nil
	}
	var from, to *types.Header
	var selected *types.Block
	if filter.BlockHash != nil {
		if filter.FromBlock != nil || filter.ToBlock != nil {
			return nil, invalidParams("blockHash is mutually exclusive with fromBlock and toBlock")
		}
		selected, err = api.selectBlockByHash(ctx, *filter.BlockHash, true)
		if err != nil {
			return nil, err
		}
		// The executed head was read before the canonical check, so a reorg
		// cannot lift it over a block that was never executed.
		if selected.NumberU64() > head.Number.Uint64() {
			return nil, notFound("block %s is not executed", filter.BlockHash.Hex())
		}
		from, to = selected.Header(), selected.Header()
	} else {
		var err error
		from, err = resolve(filter.FromBlock)
		if err != nil {
			return nil, err
		}
		to, err = resolve(filter.ToBlock)
		if err != nil {
			return nil, err
		}
	}
	lo, hi := from.Number.Uint64(), to.Number.Uint64()
	if lo > hi {
		return nil, invalidParams("fromBlock exceeds toBlock")
	}
	if hi-lo >= filterBlockLimit {
		return nil, invalidParams("trace_filter range exceeds %d blocks", filterBlockLimit)
	}
	count := uint64(filterResultLimit)
	if filter.Count != nil {
		count = *filter.Count
		if count > filterResultLimit {
			return nil, invalidParams("trace_filter count exceeds %d", filterResultLimit)
		}
	}
	result := make([]*TraceFrame, 0)
	if count == 0 {
		return result, nil
	}
	blocks := []*types.Block{selected}
	if selected == nil {
		// Walk parent hashes once, so a concurrent reorg cannot mix branches by height.
		blocks = make([]*types.Block, hi-lo+1)
		hash := to.Hash()
		for i := len(blocks) - 1; i >= 0; i-- {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			block, err := api.blockByHash(ctx, hash)
			if err != nil {
				return nil, err
			}
			blocks[i] = block
			hash = block.ParentHash()
		}
		if blocks[0].Hash() != from.Hash() {
			return nil, errors.New("canonical chain changed while resolving trace_filter bounds")
		}
	}
	after := uint64(0)
	if filter.After != nil {
		after = *filter.After
	}
	for _, block := range blocks {
		frames, err := api.blockTraces(ctx, block)
		if err != nil {
			return nil, err
		}
		// Replay reads ancestors and historical state by canonical number, so
		// the traces of a block reorganized out while it ran may mix branches.
		if _, err := api.selectBlockByHash(ctx, block.Hash(), true); err != nil {
			return nil, err
		}
		for _, frame := range frames {
			if !matchesFilter(frame, filter) {
				continue
			}
			if after > 0 {
				after--
				continue
			}
			if uint64(len(result)) == count {
				return nil, errors.New("trace_filter result limit exceeded; use count and after")
			}
			result = append(result, frame)
			if filter.Count != nil && uint64(len(result)) == count {
				return result, nil
			}
		}
	}
	return result, nil
}

func (api *API) block(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.PendingBlockNumber {
		block, _, err := api.pending()
		return block, err
	}
	// The backend resolves earliest to the lowest available block, as eth_* does.
	block, err := api.backend.BlockByNumber(ctx, number)
	if errors.Is(err, ethapi.ErrSafeBlockNotFound) || errors.Is(err, ethapi.ErrFinalizedBlockNotFound) {
		return nil, notFound("%v", err)
	}
	// Preserve other backend failures, including the typed pruned-history error.
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, notFound("block %s not found", number)
	}
	return block, nil
}

// pending returns the pending block and its post-state. Only a real pending
// environment, the next block built from pending transactions, selects
// pending; without one it is an invalid parameter rather than latest.
func (api *API) pending() (*types.Block, *state.StateDB, error) {
	block, _, st := api.backend.Pending()
	if block == nil || st == nil {
		return nil, nil, invalidParams("pending block is not available")
	}
	return block, st, nil
}

// callEnv is the shared state and block environment of unsigned calls.
type callEnv struct {
	state       *state.StateDB
	vmctx       vm.BlockContext
	precompiles vm.PrecompiledContracts // nil unless state overrides replaced them
	release     tracers.StateReleaseFunc
}

// blockByNumberOrHash selects a block by number, tag or hash. The pending
// block comes with its post-state, which already includes the pending
// transactions; for any other block the state is nil.
func (api *API) blockByNumberOrHash(ctx context.Context, selector rpc.BlockNumberOrHash) (*types.Block, *state.StateDB, error) {
	if hash, ok := selector.Hash(); ok {
		block, err := api.selectBlockByHash(ctx, hash, selector.RequireCanonical)
		return block, nil, err
	}
	if number, ok := selector.Number(); ok {
		if number == rpc.PendingBlockNumber {
			return api.pending()
		}
		block, err := api.block(ctx, number)
		return block, nil, err
	}
	return nil, nil, invalidParams("invalid arguments; neither block nor hash specified")
}

// newCallEnv selects a block by number, tag or hash, latest by default, and
// returns its post-state and block environment. The caller releases the state.
func (api *API) newCallEnv(ctx context.Context, selector *rpc.BlockNumberOrHash) (*callEnv, error) {
	sel := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if selector != nil {
		sel = *selector
	}
	block, st, err := api.blockByNumberOrHash(ctx, sel)
	if err != nil {
		return nil, err
	}
	release := tracers.StateReleaseFunc(func() {})
	if st == nil {
		if st, release, err = api.backend.StateAtBlock(ctx, block, nil, true, false); err != nil {
			return nil, stateError(err)
		}
	}
	return &callEnv{state: st, vmctx: core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil), release: release}, nil
}

// applyOverrides applies the optional overrides as eth_call does: block
// overrides replace fields of the block environment, then state overrides
// apply before the first call.
func (env *callEnv) applyOverrides(config *params.ChainConfig, stateOverrides *override.StateOverride, blockOverrides *override.BlockOverrides) error {
	if err := blockOverrides.Apply(&env.vmctx); err != nil {
		return invalidParams("invalid block overrides: %v", err)
	}
	if stateOverrides != nil {
		env.precompiles = vm.ActivePrecompiledContracts(config.Rules(env.vmctx.BlockNumber, env.vmctx.Random != nil, env.vmctx.Time))
		if err := stateOverrides.Apply(env.state, env.precompiles); err != nil {
			return invalidParams("invalid state overrides: %v", err)
		}
	}
	return nil
}

// selectBlockByHash returns the block with the given hash, which must be
// canonical if requested.
func (api *API) selectBlockByHash(ctx context.Context, hash common.Hash, canonical bool) (*types.Block, error) {
	block, err := api.backend.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, notFound("block %s not found", hash.Hex())
	}
	if canonical {
		header, err := api.backend.HeaderByNumber(ctx, rpc.BlockNumber(block.NumberU64()))
		if err != nil {
			return nil, err
		}
		if header == nil || header.Hash() != hash {
			return nil, notFound("block %s is not canonical", hash.Hex())
		}
	}
	return block, nil
}

// checkCallArgs rejects call objects that no transaction can carry at any
// block. eth_call leaves most of these to message validation after the fork
// checks; here they take precedence over block-dependent rejections.
func checkCallArgs(args *ethapi.TransactionArgs) error {
	if args.AuthorizationList != nil && args.IsEIP4844() {
		return invalidParams("authorizationList conflicts with blob fields")
	}
	// A legacy gasPrice serves as both fee caps, also for an authorization or blob call, but not beside
	// dynamic fee fields. An explicit type never changes execution: the fields present decide.
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return invalidParams("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return invalidParams("data and input disagree")
	}
	// An omitted fee cap defaults to zero.
	if args.MaxPriorityFeePerGas != nil {
		feeCap := new(big.Int)
		if args.MaxFeePerGas != nil {
			feeCap = args.MaxFeePerGas.ToInt()
		}
		if args.MaxPriorityFeePerGas.ToInt().Cmp(feeCap) > 0 {
			return invalidParams("%v: maxPriorityFeePerGas %v, maxFeePerGas %v", core.ErrTipAboveFeeCap, args.MaxPriorityFeePerGas.ToInt(), feeCap)
		}
	}
	// A blob fee cap or hash list requires at least one blob hash.
	if args.IsEIP4844() && len(args.BlobHashes) == 0 {
		return invalidParams("%v", core.ErrMissingBlobHashes)
	}
	for i, hash := range args.BlobHashes {
		if !kzg4844.IsValidVersionedHash(hash[:]) {
			return invalidParams("blob %d has invalid hash version", i)
		}
	}
	if args.AuthorizationList != nil && len(args.AuthorizationList) == 0 {
		return invalidParams("%v", core.ErrEmptyAuthList)
	}
	return nil
}

func (api *API) call(ctx context.Context, input CallArgs, kinds Outputs, env *callEnv, index int) (*Execution, error) {
	args := input.TransactionArgs
	if err := checkCallArgs(&args); err != nil {
		return nil, err
	}
	// As in eth_call, CallDefaults runs omitted or over-cap gas at the RPC gas cap,
	// and omitted gas at the maximum without a cap.
	vmctx := env.vmctx
	gasCap, chainID := api.backend.RPCGasCap(), api.backend.ChainConfig().ChainID
	if gasCap == 0 {
		gasCap = math.MaxUint64
	}
	// Before London only gasPrice priced a transaction, so dynamic fee fields are rejected there, even
	// with zero values, rather than dropped: CallDefaults prices such a call at gasPrice 0. The other
	// invalid-parameter checks of CallDefaults take precedence over this block-dependent one.
	if (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) && !api.backend.ChainConfig().IsLondon(vmctx.BlockNumber) {
		rest := args
		rest.MaxFeePerGas, rest.MaxPriorityFeePerGas = nil, nil
		if err := rest.CallDefaults(gasCap, vmctx.BaseFee, chainID); err != nil {
			return nil, invalidParams("invalid call: %v", err)
		}
		return nil, callRejection(fmt.Errorf("%w: maxFeePerGas or maxPriorityFeePerGas before London", core.ErrTxTypeNotSupported))
	}
	if err := args.CallDefaults(gasCap, vmctx.BaseFee, chainID); err != nil {
		return nil, invalidParams("invalid call: %v", err)
	}
	// An explicit type adds no requirement of its own: a supplied feature
	// that is not active at the selected fork is rejected by message validation.
	// A supplied nonce is neither validated nor used: the message skips the
	// nonce check, and execution, including CREATE address derivation, uses
	// the sender's state nonce.
	// As in eth_call, execution may spend more than params.MaxTxGas after Amsterdam.
	msg := args.ToMessage(vmctx.BaseFee, true).SkipExecutionGasCapCheck()
	// The transaction only identifies the call to the state and its receipt. Its
	// nonce is the item index, so that callMany items never share a hash and the
	// logs collected under it.
	nonce := hexutil.Uint64(index)
	args.Nonce = &nonce
	// A legacy gasPrice prices an authorization call as both fee caps, but as a
	// legacy transaction it would drop the authorizations the tracer discovers
	// from it.
	if args.GasPrice != nil && args.AuthorizationList != nil {
		args.MaxFeePerGas, args.MaxPriorityFeePerGas, args.GasPrice = args.GasPrice, args.GasPrice, nil
	}
	tx := args.ToTransaction(types.DynamicFeeTxType)
	// As in eth_call, a zero effective gas price runs with BASEFEE 0, and a
	// zero blob fee cap (supplied or defaulted) with BLOBBASEFEE 0. NoBaseFee
	// skips fee validation only when both fee caps are zero.
	if msg.GasPrice.Sign() == 0 {
		vmctx.BaseFee = new(big.Int)
	}
	if msg.BlobGasFeeCap != nil && msg.BlobGasFeeCap.BitLen() == 0 {
		vmctx.BlobBaseFee = new(big.Int)
	}
	return api.execute(ctx, api.backend.RPCEVMTimeout(), tx, msg, kinds, vmctx, env.state, env.precompiles, common.Hash{}, index, true, callRejection)
}

// checkGasCap rejects a signed gas limit above the RPC gas cap, since capping it
// would change the transaction.
func (api *API) checkGasCap(gas uint64) error {
	if cap := api.backend.RPCGasCap(); cap != 0 && gas > cap {
		return limitExceeded("gas %d exceeds the RPC gas cap %d", gas, cap)
	}
	return nil
}

func (api *API) transaction(ctx context.Context, hash common.Hash, kinds Outputs) (*Execution, *types.Block, uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	found, _, blockHash, number, index := api.backend.GetCanonicalTransaction(hash)
	if !found {
		if !api.backend.TxIndexDone() {
			return nil, nil, 0, ethapi.NewTxIndexingError()
		}
		// Genesis has no transactions. If any later block lacks lookup history,
		// a missing hash cannot establish absence from the canonical chain.
		if tail := rawdb.ReadTxIndexTail(api.backend.ChainDb()); tail != nil && *tail > 1 {
			return nil, nil, 0, &rpcError{errCodePrunedHistory, "transaction lookup history unavailable"}
		}
		return nil, nil, 0, nil
	}
	block, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(number), blockHash)
	if err != nil {
		return nil, nil, 0, err
	}
	// Preceding transactions only build the target's pre-state, so they run
	// without tracer hooks.
	tx, vmctx, st, release, err := api.backend.StateAtTransaction(ctx, block, int(index))
	if err != nil {
		return nil, nil, 0, stateError(err)
	}
	defer release()
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
	if err != nil {
		return nil, nil, 0, err
	}
	result, err := api.execute(ctx, replayTimeout, tx, msg, kinds, vmctx, st, nil, blockHash, int(index), false, replayRejection)
	if err != nil {
		return nil, nil, 0, err
	}
	result.TransactionHash = &hash
	return result, block, index, nil
}

func (api *API) replayBlock(ctx context.Context, block *types.Block, kinds Outputs) ([]*Execution, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	results := make([]*Execution, 0, len(block.Transactions()))
	if len(block.Transactions()) == 0 {
		return results, nil
	}
	_, vmctx, st, release, err := api.backend.StateAtTransaction(ctx, block, 0)
	if err != nil {
		return nil, stateError(err)
	}
	defer release()
	for i, tx := range block.Transactions() {
		msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
		if err != nil {
			return nil, err
		}
		result, err := api.execute(ctx, replayTimeout, tx, msg, kinds, vmctx, st, nil, block.Hash(), i, false, replayRejection)
		if err != nil {
			return nil, err
		}
		hash := tx.Hash()
		result.TransactionHash = &hash
		results = append(results, result)
	}
	return results, nil
}

// execute applies one message, within the timeout when it is positive.
// Precompiles replace the active set when not nil; reject maps a validation
// failure, which leaves the message unexecuted, to its RPC error.
func (api *API) execute(ctx context.Context, timeout time.Duration, tx *types.Transaction, msg *core.Message, kinds Outputs, vmctx vm.BlockContext, st *state.StateDB, precompiles vm.PrecompiledContracts, blockHash common.Hash, index int, unsigned bool, reject func(error) error) (*Execution, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rules := api.backend.ChainConfig().Rules(vmctx.BlockNumber, vmctx.Random != nil, vmctx.Time)
	active := precompiles
	if active == nil {
		active = vm.ActivePrecompiledContracts(rules)
	}
	c := newCapture(kinds, rules, active)
	hooks := c.hooks()
	evm := vm.NewEVM(vmctx, state.NewHookedState(st, hooks), api.backend.ChainConfig(), vm.Config{Tracer: hooks, NoBaseFee: unsigned})
	defer evm.Release()
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	c.stop = evm.Cancel
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { evm.Cancel(); close(stopped) })
	// Release returns the EVM to a pool. Join a running cancellation callback
	// first so it cannot cancel another request that reuses the EVM.
	defer func() {
		if !stop() {
			<-stopped
		}
	}()
	st.SetTxContext(tx.Hash(), index, uint32(index+1))
	_, _, err := core.ApplyTransactionWithEVM(ctx, msg, core.NewGasPool(msg.GasLimit), st, vmctx.BlockNumber, blockHash, vmctx.Time, tx, evm)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if c.err != nil {
		return nil, c.err
	}
	if err := st.Error(); err != nil {
		return nil, stateError(err)
	}
	if err != nil {
		// EVM reverts and exceptional halts are execution results. Errors from
		// ApplyTransactionWithEVM reject the message before execution; a failed
		// state read must take precedence over any resulting validation error.
		return nil, reject(err)
	}
	result := c.result()
	if err := st.Error(); err != nil {
		return nil, stateError(err)
	}
	return result, nil
}

func (api *API) blockTraces(ctx context.Context, block *types.Block) ([]*TraceFrame, error) {
	replays, err := api.replayBlock(ctx, block, Outputs{"trace"})
	if err != nil {
		return nil, err
	}
	frames := make([]*TraceFrame, 0)
	for i, replay := range replays {
		index := uint64(i)
		localizeTrace(replay.Trace, block, replay.TransactionHash, &index)
		frames = append(frames, replay.Trace...)
	}
	// Clique and post-Merge blocks do not pay Ethash issuance rewards.
	config := api.backend.ChainConfig()
	if block.NumberU64() == 0 || block.Difficulty().Sign() == 0 || config.Ethash == nil {
		return frames, nil
	}
	reward := big.NewInt(5e18)
	if config.IsByzantium(block.Number()) {
		reward = big.NewInt(3e18)
	}
	if config.IsConstantinople(block.Number()) {
		reward = big.NewInt(2e18)
	}
	appendReward := func(author common.Address, value *big.Int, kind string) {
		frame := &TraceFrame{Type: "reward", Action: rewardAction{author, (*hexutil.Big)(value), kind}, TraceAddress: []uint64{}}
		localizeTrace([]*TraceFrame{frame}, block, nil, nil)
		frames = append(frames, frame)
	}
	miner := new(big.Int).Add(reward, new(big.Int).Mul(new(big.Int).Div(new(big.Int).Set(reward), big.NewInt(32)), big.NewInt(int64(len(block.Uncles())))))
	appendReward(block.Coinbase(), miner, "block")
	for _, uncle := range block.Uncles() {
		amount := new(big.Int).Add(uncle.Number, big.NewInt(8))
		amount.Sub(amount, block.Number())
		amount.Mul(amount, reward)
		amount.Div(amount, big.NewInt(8))
		appendReward(uncle.Coinbase, amount, "uncle")
	}
	return frames, nil
}

func localizeTrace(frames []*TraceFrame, block *types.Block, hash *common.Hash, index *uint64) {
	for _, frame := range frames {
		frame.location = &location{block.Hash(), block.NumberU64(), hash, index}
	}
}

func matchesFilter(frame *TraceFrame, filter FilterArgs) bool {
	var from, to *common.Address
	switch action := frame.Action.(type) {
	case callAction:
		from, to = &action.From, &action.To
	case createAction:
		from = &action.From
		if result, ok := frame.Result.(createResult); ok && frame.Error == "" {
			to = &result.Address
		}
	case suicideAction:
		from, to = &action.Address, &action.RefundAddress
	case rewardAction:
		to = &action.Author
	}
	contains := func(list []common.Address, address *common.Address) bool {
		if address == nil {
			return false
		}
		for _, item := range list {
			if item == *address {
				return true
			}
		}
		return false
	}
	// Empty lists are unrestricted; union matches either populated list.
	fromSet, toSet := len(filter.FromAddress) > 0, len(filter.ToAddress) > 0
	fromOK, toOK := !fromSet || contains(filter.FromAddress, from), !toSet || contains(filter.ToAddress, to)
	if filter.Mode == FilterUnion && fromSet && toSet {
		return contains(filter.FromAddress, from) || contains(filter.ToAddress, to)
	}
	return fromOK && toOK
}
