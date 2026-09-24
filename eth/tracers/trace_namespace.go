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

package tracers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	traceFilterBlockLimit  = 1000
	traceFilterResultLimit = 10000
	traceBatchTimeout      = 30 * time.Second
)

// TraceAPI implements the Parity trace namespace independently of debug APIs.
type TraceAPI struct{ api *API }

// NewTraceAPI constructs the trace namespace service.
func NewTraceAPI(backend Backend) *TraceAPI { return &TraceAPI{api: NewAPI(backend)} }

// Call executes an unsigned call on the selected block's post-state.
func (api *TraceAPI) Call(ctx context.Context, args TraceCallArgs, kinds TraceTypes, number *rpc.BlockNumber) (*TraceExecution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	block, st, release, err := api.callState(ctx, number)
	if err != nil {
		return nil, err
	}
	defer release()
	return api.call(ctx, args, kinds, block, st, 0)
}

// CallMany executes calls sequentially, retaining successful writes in the sequence.
func (api *TraceAPI) CallMany(ctx context.Context, calls TraceCalls, number *rpc.BlockNumber) ([]*TraceExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, traceBatchTimeout)
	defer cancel()
	if len(calls) > traceFilterResultLimit {
		return nil, traceInvalid("too many calls (limit %d)", traceFilterResultLimit)
	}
	for _, call := range calls {
		if err := call.Types.validate(); err != nil {
			return nil, err
		}
	}
	block, st, release, err := api.callState(ctx, number)
	if err != nil {
		return nil, err
	}
	defer release()
	results := make([]*TraceExecution, 0, len(calls))
	for i, call := range calls {
		result, err := api.call(ctx, call.Call, call.Types, block, st, i)
		if err != nil {
			var rpcErr rpc.Error
			if errors.As(err, &rpcErr) {
				return nil, &traceRPCError{rpcErr.ErrorCode(), fmt.Sprintf("call %d: %v", i, err)}
			}
			return nil, fmt.Errorf("call %d: %w", i, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// RawTransaction validates and executes the supplied signed transaction at latest state.
func (api *TraceAPI) RawTransaction(ctx context.Context, input hexutil.Bytes, kinds TraceTypes) (*TraceExecution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return nil, traceInvalid("invalid transaction: %v", err)
	}
	block, st, release, err := api.callState(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer release()
	if cap := api.api.backend.RPCGasCap(); cap != 0 && tx.Gas() > cap {
		return nil, traceInvalid("transaction gas exceeds RPC cap %d", cap)
	}
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
	if err != nil {
		return nil, &traceRPCError{-32003, fmt.Sprintf("invalid signed transaction: %v", err)}
	}
	vmctx := core.NewEVMBlockContext(block.Header(), api.api.chainContext(ctx), nil)
	return api.execute(ctx, tx, msg, kinds, vmctx, st, common.Hash{}, 0, false)
}

// ReplayTransaction returns one replay envelope or null for an unknown transaction.
func (api *TraceAPI) ReplayTransaction(ctx context.Context, hash common.Hash, kinds TraceTypes) (*TraceExecution, error) {
	if err := kinds.validate(); err != nil {
		return nil, err
	}
	result, _, _, err := api.transaction(ctx, hash, kinds)
	return result, err
}

// ReplayBlockTransactions returns one envelope per transaction in block order.
func (api *TraceAPI) ReplayBlockTransactions(ctx context.Context, number rpc.BlockNumber, kinds TraceTypes) ([]*TraceExecution, error) {
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
func (api *TraceAPI) Transaction(ctx context.Context, hash common.Hash) ([]*TraceFrame, error) {
	result, block, index, err := api.transaction(ctx, hash, TraceTypes{"trace"})
	if err != nil || result == nil {
		return nil, err
	}
	localizeTrace(result.Trace, block, &hash, &index)
	return result.Trace, nil
}

// Get returns one tree path, with the empty path selecting the root.
func (api *TraceAPI) Get(ctx context.Context, hash common.Hash, position TracePosition) (*TraceFrame, error) {
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
func (api *TraceAPI) Block(ctx context.Context, number rpc.BlockNumber) ([]*TraceFrame, error) {
	block, err := api.block(ctx, number)
	if err != nil {
		return nil, err
	}
	return api.blockTraces(ctx, block)
}

// Filter scans a bounded canonical snapshot, applying pagination after matching.
func (api *TraceAPI) Filter(ctx context.Context, filter TraceFilter) ([]*TraceFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, traceBatchTimeout)
	defer cancel()
	// Resolve both bounds against one head. As for eth_getLogs, a bound beyond
	// the head or a reversed range is invalid rather than clamped.
	head := api.api.backend.CurrentHeader()
	resolve := func(number *rpc.BlockNumber) (*types.Header, error) {
		n := rpc.LatestBlockNumber
		if number != nil {
			n = *number
		}
		switch {
		case n == rpc.PendingBlockNumber:
			return nil, traceInvalid("pending is not a valid trace_filter bound")
		case n == rpc.LatestBlockNumber:
			return head, nil
		case n >= 0 && uint64(n) > head.Number.Uint64():
			return nil, traceInvalid("block %d is beyond the current head %d", n, head.Number.Uint64())
		}
		h, err := api.api.backend.HeaderByNumber(ctx, n)
		if (err != nil && traceTagMissing(n, err)) || (err == nil && h == nil) {
			return nil, traceInvalid("block %s not found", n)
		}
		if err != nil {
			return nil, err
		}
		return h, nil
	}
	from, err := resolve(filter.FromBlock)
	if err != nil {
		return nil, err
	}
	to, err := resolve(filter.ToBlock)
	if err != nil {
		return nil, err
	}
	lo, hi := from.Number.Uint64(), to.Number.Uint64()
	if lo > hi {
		return nil, traceInvalid("fromBlock exceeds toBlock")
	}
	if hi-lo >= traceFilterBlockLimit {
		return nil, traceInvalid("trace_filter range exceeds %d blocks", traceFilterBlockLimit)
	}
	count := uint64(traceFilterResultLimit)
	if filter.Count != nil {
		count = *filter.Count
		if count > traceFilterResultLimit {
			return nil, traceInvalid("trace_filter count exceeds %d", traceFilterResultLimit)
		}
	}
	result := make([]*TraceFrame, 0)
	if count == 0 {
		return result, nil
	}
	// Walk parent hashes once, so a concurrent reorg cannot mix branches by height.
	blocks := make([]*types.Block, hi-lo+1)
	hash := to.Hash()
	for i := len(blocks) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		block, err := api.api.blockByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		blocks[i] = block
		hash = block.ParentHash()
	}
	if blocks[0].Hash() != from.Hash() {
		return nil, errors.New("canonical chain changed while resolving trace_filter bounds")
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
		for _, frame := range frames {
			if !traceMatches(frame, filter) {
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

func (api *TraceAPI) block(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.PendingBlockNumber {
		return nil, traceInvalid("pending tracing is not supported")
	}
	// The backend resolves earliest to the lowest available block, as eth_* does.
	block, err := api.api.backend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, traceBlockError(number, err)
	}
	if block == nil {
		return nil, &traceRPCError{-32001, fmt.Sprintf("block %s not found", number)}
	}
	return block, nil
}

func traceBlockError(number rpc.BlockNumber, err error) error {
	// Preserve other backend failures, including the typed pruned-history error.
	if traceTagMissing(number, err) {
		return &traceRPCError{-32001, err.Error()}
	}
	return err
}

// traceTagMissing reports whether the backend signalled an unavailable safe or
// finalized block. These tags report absence as errors rather than nil headers.
func traceTagMissing(number rpc.BlockNumber, err error) bool {
	return (number == rpc.SafeBlockNumber || number == rpc.FinalizedBlockNumber) && err.Error() == number.String()+" block not found"
}

func (api *TraceAPI) callState(ctx context.Context, number *rpc.BlockNumber) (*types.Block, *state.StateDB, StateReleaseFunc, error) {
	n := rpc.LatestBlockNumber
	if number != nil {
		n = *number
	}
	block, err := api.block(ctx, n)
	if err != nil {
		return nil, nil, nil, err
	}
	st, release, err := api.api.backend.StateAtBlock(ctx, block, nil, true, false)
	if err != nil {
		return nil, nil, nil, traceStateError(err)
	}
	return block, st, release, nil
}

func (api *TraceAPI) call(ctx context.Context, input TraceCallArgs, kinds TraceTypes, block *types.Block, st *state.StateDB, index int) (*TraceExecution, error) {
	args := input.TransactionArgs
	if args.AuthorizationList != nil && args.IsEIP4844() {
		return nil, traceInvalid("authorizationList conflicts with blob fields")
	}
	if args.GasPrice != nil && (args.AuthorizationList != nil || args.IsEIP4844()) {
		return nil, traceInvalid("gasPrice conflicts with blob or authorization fields")
	}
	if input.Type != nil {
		if *input.Type > types.SetCodeTxType {
			return nil, traceInvalid("unsupported unsigned transaction type")
		}
		if *input.Type < types.DynamicFeeTxType && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
			return nil, traceInvalid("dynamic fee fields require transaction type 2 or later")
		}
		if *input.Type >= types.DynamicFeeTxType && args.GasPrice != nil {
			return nil, traceInvalid("gasPrice conflicts with transaction type %d", *input.Type)
		}
		if *input.Type == types.LegacyTxType && args.AccessList != nil {
			return nil, traceInvalid("accessList conflicts with transaction type 0")
		}
		if args.IsEIP4844() && *input.Type != types.BlobTxType {
			return nil, traceInvalid("blob fields require transaction type 3")
		}
		if args.AuthorizationList != nil && *input.Type != types.SetCodeTxType {
			return nil, traceInvalid("authorizationList requires transaction type 4")
		}
		if *input.Type == types.BlobTxType && args.BlobHashes == nil {
			return nil, traceInvalid("transaction type 3 requires blobVersionedHashes")
		}
		if *input.Type == types.SetCodeTxType && args.AuthorizationList == nil {
			return nil, traceInvalid("transaction type 4 requires authorizationList")
		}
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return nil, traceInvalid("data and input disagree")
	}
	from := common.Address{}
	if args.From != nil {
		from = *args.From
	}
	// A supplied nonce is neither validated nor used: execution, including
	// CREATE address derivation, uses the sender's state nonce.
	nonce := hexutil.Uint64(st.GetNonce(from))
	args.Nonce = &nonce
	vmctx := core.NewEVMBlockContext(block.Header(), api.api.chainContext(ctx), nil)
	if err := args.CallDefaults(api.api.backend.RPCGasCap(), vmctx.BaseFee, api.api.backend.ChainConfig().ChainID); err != nil {
		return nil, traceInvalid("invalid call: %v", err)
	}
	msg := args.ToMessage(vmctx.BaseFee, true)
	tx := args.ToTransaction(types.DynamicFeeTxType)
	if input.Type != nil {
		switch *input.Type {
		case types.LegacyTxType:
			tx = types.NewTx(&types.LegacyTx{Nonce: msg.Nonce, To: msg.To, Gas: msg.GasLimit, GasPrice: msg.GasPrice.ToBig(), Value: msg.Value.ToBig(), Data: msg.Data})
		case types.AccessListTxType:
			tx = types.NewTx(&types.AccessListTx{ChainID: api.api.backend.ChainConfig().ChainID, Nonce: msg.Nonce, To: msg.To, Gas: msg.GasLimit, GasPrice: msg.GasPrice.ToBig(), Value: msg.Value.ToBig(), Data: msg.Data, AccessList: msg.AccessList})
		}
	}
	// As in eth_call, a zero effective gas price runs with BASEFEE 0, and a
	// zero blob fee cap (supplied or defaulted) with BLOBBASEFEE 0. NoBaseFee
	// skips fee validation only when both fee caps are zero.
	if msg.GasPrice.Sign() == 0 {
		vmctx.BaseFee = new(big.Int)
	}
	if msg.BlobGasFeeCap != nil && msg.BlobGasFeeCap.BitLen() == 0 {
		vmctx.BlobBaseFee = new(big.Int)
	}
	return api.execute(ctx, tx, msg, kinds, vmctx, st, common.Hash{}, index, true)
}

func (api *TraceAPI) transaction(ctx context.Context, hash common.Hash, kinds TraceTypes) (*TraceExecution, *types.Block, uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, traceBatchTimeout)
	defer cancel()
	found, _, blockHash, number, index := api.api.backend.GetCanonicalTransaction(hash)
	if !found {
		if !api.api.backend.TxIndexDone() {
			return nil, nil, 0, ethapi.NewTxIndexingError()
		}
		// Genesis has no transactions. If any later block lacks lookup history,
		// a missing hash cannot establish absence from the canonical chain.
		if tail := rawdb.ReadTxIndexTail(api.api.backend.ChainDb()); tail != nil && *tail > 1 {
			return nil, nil, 0, &traceRPCError{4444, "transaction lookup history unavailable"}
		}
		return nil, nil, 0, nil
	}
	block, err := api.api.blockByNumberAndHash(ctx, rpc.BlockNumber(number), blockHash)
	if err != nil {
		return nil, nil, 0, err
	}
	st, vmctx, release, err := api.blockState(ctx, block)
	if err != nil {
		return nil, nil, 0, err
	}
	defer release()
	for i, tx := range block.Transactions() {
		msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
		if err != nil {
			return nil, nil, 0, err
		}
		modes := TraceTypes{}
		if uint64(i) == index {
			modes = kinds
		}
		result, err := api.execute(ctx, tx, msg, modes, vmctx, st, blockHash, i, false)
		if err != nil {
			return nil, nil, 0, err
		}
		if uint64(i) == index {
			result.TransactionHash = &hash
			return result, block, index, nil
		}
	}
	return nil, nil, 0, errors.New("transaction index points outside block")
}

func (api *TraceAPI) blockState(ctx context.Context, block *types.Block) (*state.StateDB, vm.BlockContext, StateReleaseFunc, error) {
	parent, err := api.api.blockByHash(ctx, block.ParentHash())
	if err != nil {
		return nil, vm.BlockContext{}, nil, err
	}
	st, release, err := api.api.backend.StateAtBlock(ctx, parent, nil, true, false)
	if err != nil {
		return nil, vm.BlockContext{}, nil, traceStateError(err)
	}
	config := api.api.backend.ChainConfig()
	vmctx := core.NewEVMBlockContext(block.Header(), api.api.chainContext(ctx), nil)
	if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(st)
	}
	evm := vm.NewEVM(vmctx, st, config, vm.Config{})
	core.PreExecution(ctx, block.BeaconRoot(), parent.Header(), config, evm, block.Number(), block.Time())
	evm.Release()
	if err := st.Error(); err != nil {
		release()
		return nil, vm.BlockContext{}, nil, traceStateError(err)
	}
	return st, vmctx, release, nil
}

func (api *TraceAPI) replayBlock(ctx context.Context, block *types.Block, kinds TraceTypes) ([]*TraceExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, traceBatchTimeout)
	defer cancel()
	results := make([]*TraceExecution, 0, len(block.Transactions()))
	if len(block.Transactions()) == 0 {
		return results, nil
	}
	st, vmctx, release, err := api.blockState(ctx, block)
	if err != nil {
		return nil, err
	}
	defer release()
	for i, tx := range block.Transactions() {
		msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
		if err != nil {
			return nil, err
		}
		result, err := api.execute(ctx, tx, msg, kinds, vmctx, st, block.Hash(), i, false)
		if err != nil {
			return nil, err
		}
		hash := tx.Hash()
		result.TransactionHash = &hash
		results = append(results, result)
	}
	return results, nil
}

func (api *TraceAPI) execute(ctx context.Context, tx *types.Transaction, msg *core.Message, kinds TraceTypes, vmctx vm.BlockContext, st *state.StateDB, blockHash common.Hash, index int, unsigned bool) (*TraceExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTraceTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	capture := newTraceCapture(st, kinds, api.api.backend.ChainConfig().Rules(vmctx.BlockNumber, vmctx.Random != nil, vmctx.Time))
	hooks := capture.hooks()
	evm := vm.NewEVM(vmctx, state.NewHookedState(st, hooks), api.api.backend.ChainConfig(), vm.Config{Tracer: hooks, NoBaseFee: unsigned})
	defer evm.Release()
	capture.stop = evm.Cancel
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
	if capture.err != nil {
		return nil, capture.err
	}
	if err := st.Error(); err != nil {
		return nil, traceStateError(err)
	}
	if err != nil {
		// EVM reverts and exceptional halts are execution results. Errors from
		// ApplyTransactionWithEVM reject the message before execution; a failed
		// state read must take precedence over any resulting validation error.
		return nil, &traceRPCError{-32003, err.Error()}
	}
	result := capture.result()
	if err := st.Error(); err != nil {
		return nil, traceStateError(err)
	}
	if capture.before != nil && capture.before.Error() != nil {
		return nil, traceStateError(capture.before.Error())
	}
	return result, nil
}

func (api *TraceAPI) blockTraces(ctx context.Context, block *types.Block) ([]*TraceFrame, error) {
	replays, err := api.replayBlock(ctx, block, TraceTypes{"trace"})
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
	config := api.api.backend.ChainConfig()
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
		frame := &TraceFrame{Type: "reward", Action: traceRewardAction{author, (*hexutil.Big)(value), kind}, TraceAddress: []uint64{}}
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
		frame.traceLocation = &traceLocation{block.Hash(), block.NumberU64(), hash, index}
	}
}

func traceMatches(frame *TraceFrame, filter TraceFilter) bool {
	var from, to *common.Address
	switch action := frame.Action.(type) {
	case traceCallAction:
		from, to = &action.From, &action.To
	case traceCreateAction:
		from = &action.From
		if result, ok := frame.Result.(traceCreateResult); ok && frame.Error == "" {
			to = &result.Address
		}
	case traceSuicideAction:
		from, to = &action.Address, &action.RefundAddress
	case traceRewardAction:
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
	if filter.Mode == TraceFilterUnion && fromSet && toSet {
		return contains(filter.FromAddress, from) || contains(filter.ToAddress, to)
	}
	return fromOK && toOK
}

func traceStateError(err error) error {
	for _, text := range []string{"historical state", "missing trie node", "state not found", "pruned"} {
		if strings.Contains(strings.ToLower(err.Error()), text) {
			return &traceRPCError{4444, err.Error()}
		}
	}
	return err
}
