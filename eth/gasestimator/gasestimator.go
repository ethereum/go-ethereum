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

package gasestimator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Options are the contextual parameters to execute the requested call.
//
// Whilst it would be possible to pass a blockchain object that aggregates all
// these together, it would be excessively hard to test. Splitting the parts out
// allows testing without needing a proper live chain.
type Options struct {
	Config *params.ChainConfig // Chain configuration for hard fork selection
	Chain  core.ChainContext   // Chain context to access past block hashes
	Header *types.Header       // Header defining the block context to execute in
	State  *state.StateDB      // Pre-state on top of which to estimate the gas

	ErrorRatio float64 // Allowed overestimation ratio for faster estimation termination
}

// Estimate returns the lowest possible gas limit that allows the transaction to
// run successfully with the provided context options. It returns an error if the
// transaction would always revert, or if there are unexpected failures.
func Estimate(ctx context.Context, call *core.Message, opts *Options, gasCap uint64) (uint64, []byte, error) {
	// Binary search the gas limit, as it may need to be higher than the amount used
	var (
		lo uint64 // lowest-known gas limit where tx execution fails
		hi uint64 // lowest-known gas limit where tx execution succeeds
	)
	// Determine the highest gas limit can be used during the estimation.
	hi = opts.Header.GasLimit
	if call.GasLimit >= params.TxGas {
		hi = call.GasLimit
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if call.GasFeeCap != nil {
		feeCap = call.GasFeeCap
	} else if call.GasPrice != nil {
		feeCap = call.GasPrice
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		balance := opts.State.GetBalance(call.From).ToBig()

		available := balance
		if call.Value != nil {
			if call.Value.Cmp(available) >= 0 {
				return 0, nil, core.ErrInsufficientFundsForTransfer
			}
			available.Sub(available, call.Value)
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := call.Value
			if transfer == nil {
				transfer = new(big.Int)
			}
			log.Debug("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer, "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}
	// Recap the highest gas allowance with specified gascap.
	if gasCap != 0 && hi > gasCap {
		log.Debug("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}
	// If the transaction is a plain value transfer, short circuit estimation and
	// directly try 21000. Returning 21000 without any execution is dangerous as
	// some tx field combos might bump the price up even for plain transfers (e.g.
	// unused access list items). Ever so slightly wasteful, but safer overall.
	if len(call.Data) == 0 {
		if call.To != nil && opts.State.GetCodeSize(*call.To) == 0 {
			failed, _, err := execute(ctx, call, opts, params.TxGas)
			if !failed && err == nil {
				return params.TxGas, nil, nil
			}
		}
	}
	// We first execute the transaction at the highest allowable gas limit, since if this fails we
	// can return error immediately.
	failed, result, err := execute(ctx, call, opts, hi)
	if err != nil {
		return 0, nil, err
	}
	if failed {
		if result != nil && !errors.Is(result.Err, vm.ErrOutOfGas) {
			return 0, result.Revert(), result.Err
		}
		return 0, nil, fmt.Errorf("gas required exceeds allowance (%d)", hi)
	}
	// For almost any transaction, the gas consumed by the unconstrained execution
	// above lower-bounds the gas limit required for it to succeed. One exception
	// is those that explicitly check gas remaining in order to execute within a
	// given limit, but we probably don't want to return the lowest possible gas
	// limit for these cases anyway.
	lo = result.UsedGas - 1

	// There's a fairly high chance for the transaction to execute successfully
	// with gasLimit set to the first execution's usedGas + gasRefund. Explicitly
	// check that gas amount and use as a limit for the binary search.
	optimisticGasLimit := (result.UsedGas + result.RefundedGas + params.CallStipend) * 64 / 63
	if optimisticGasLimit < hi {
		failed, _, err = execute(ctx, call, opts, optimisticGasLimit)
		if err != nil {
			// This should not happen under normal conditions since if we make it this far the
			// transaction had run without error at least once before.
			log.Error("Execution error in estimate gas", "err", err)
			return 0, nil, err
		}
		if failed {
			lo = optimisticGasLimit
		} else {
			hi = optimisticGasLimit
		}
	}
	// Binary search for the smallest gas limit that allows the tx to execute successfully.
	for lo+1 < hi {
		if opts.ErrorRatio > 0 {
			// It is a bit pointless to return a perfect estimation, as changing
			// network conditions require the caller to bump it up anyway. Since
			// wallets tend to use 20-25% bump, allowing a small approximation
			// error is fine (as long as it's upwards).
			if float64(hi-lo)/float64(hi) < opts.ErrorRatio {
				break
			}
		}
		mid := (hi + lo) / 2
		if mid > lo*2 {
			// Most txs don't need much higher gas limit than their gas used, and most txs don't
			// require near the full block limit of gas, so the selection of where to bisect the
			// range here is skewed to favor the low side.
			mid = lo * 2
		}
		failed, _, err = execute(ctx, call, opts, mid)
		if err != nil {
			// This should not happen under normal conditions since if we make it this far the
			// transaction had run without error at least once before.
			log.Error("Execution error in estimate gas", "err", err)
			return 0, nil, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi, nil, nil
}

// execute is a helper that executes the transaction under a given gas limit and
// returns true if the transaction fails for a reason that might be related to
// not enough gas. A non-nil error means execution failed due to reasons unrelated
// to the gas limit.
func execute(ctx context.Context, call *core.Message, opts *Options, gasLimit uint64) (bool, *core.ExecutionResult, error) {
	// Configure the call for this specific execution (and revert the change after)
	defer func(gas uint64) { call.GasLimit = gas }(call.GasLimit)
	call.GasLimit = gasLimit

	// Execute the call and separate execution faults caused by a lack of gas or
	// other non-fixable conditions
	result, err := run(ctx, call, opts)
	if err != nil {
		if errors.Is(err, core.ErrIntrinsicGas) {
			return true, nil, nil // Special case, raise gas limit
		}
		return true, nil, err // Bail out
	}
	return result.Failed(), result, nil
}

// run assembles the EVM as defined by the consensus rules and runs the requested
// call invocation.
func run(ctx context.Context, call *core.Message, opts *Options) (*core.ExecutionResult, error) {
	// Assemble the call and the call context
	var (
		msgContext = core.NewEVMTxContext(call)
		evmContext = core.NewEVMBlockContext(opts.Header, opts.Chain, nil)

		dirtyState = opts.State.Copy()
		evm        = vm.NewEVM(evmContext, msgContext, dirtyState, opts.Config, vm.Config{NoBaseFee: true})
	)
	// Monitor the outer context and interrupt the EVM upon cancellation. To avoid
	// a dangling goroutine until the outer estimation finishes, create an internal
	// context for the lifetime of this method call.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()
	// Execute the call, returning a wrapped error or the result
	result, err := core.ApplyMessage(evm, call, new(core.GasPool).AddGas(math.MaxUint64))
	if vmerr := dirtyState.Error(); vmerr != nil {
		return nil, vmerr
	}
	if err != nil {
		return result, fmt.Errorf("failed with %d gas: %w", call.GasLimit, err)
	}
	return result, nil
}
