// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"fmt"
	"math"
)

// GasPool tracks the amount of gas available for transaction execution
// within a block, along with the cumulative gas consumed.
type GasPool struct {
	remaining      uint64
	initial        uint64
	cumulativeUsed uint64

	// EIP-8037: per-dimension cumulative sums for Amsterdam.
	// Block gas used = max(cumulativeRegular, cumulativeState).
	cumulativeRegular uint64
	cumulativeState   uint64
}

// NewGasPool initializes the gasPool with the given amount.
func NewGasPool(amount uint64) *GasPool {
	return &GasPool{
		remaining: amount,
		initial:   amount,
	}
}

// SubGas deducts the given amount from the pool if enough gas is
// available and returns an error otherwise.
func (gp *GasPool) SubGas(amount uint64) error {
	if gp.remaining < amount {
		return ErrGasLimitReached
	}
	gp.remaining -= amount
	return nil
}

// ReturnGas adds the refunded gas back to the pool and updates
// the cumulative gas usage accordingly.
func (gp *GasPool) ReturnGas(returned uint64, gasUsed uint64) error {
	if gp.remaining > math.MaxUint64-returned {
		return fmt.Errorf("%w: remaining: %d, returned: %d", ErrGasLimitOverflow, gp.remaining, returned)
	}
	// The returned gas calculation differs across forks.
	//
	// - Pre-Amsterdam:
	//   returned = purchased - remaining (refund included)
	//
	// - Post-Amsterdam:
	//   returned = purchased - gasUsed (refund excluded)
	gp.remaining += returned

	// gasUsed = max(txGasUsed - gasRefund, calldataFloorGasCost)
	// regardless of Amsterdam is activated or not.
	gp.cumulativeUsed += gasUsed
	return nil
}

// ReturnGasAmsterdam handles 2D gas accounting for Amsterdam (EIP-8037).
// It undoes the SubGas deduction fully and accumulates per-dimension block totals.
func (gp *GasPool) ReturnGasAmsterdam(returned, txRegular, txState, receiptGasUsed uint64) error {
	if gp.remaining > math.MaxUint64-returned {
		return fmt.Errorf("%w: remaining: %d, returned: %d", ErrGasLimitOverflow, gp.remaining, returned)
	}
	// Undo SubGas deduction fully (Amsterdam uses cumulative tracking)
	gp.remaining += returned
	// Accumulate 2D block dimensions
	gp.cumulativeRegular += txRegular
	gp.cumulativeState += txState
	gp.cumulativeUsed += receiptGasUsed
	return nil
}

// Gas returns the amount of gas remaining in the pool.
func (gp *GasPool) Gas() uint64 {
	return gp.remaining
}

// CumulativeUsed returns the cumulative gas consumed for receipt tracking.
// For Amsterdam blocks, this is the sum of per-tx tx_gas_used_after_refund
// (what users pay), not the 2D block-level metric.
func (gp *GasPool) CumulativeUsed() uint64 {
	return gp.cumulativeUsed
}

// Used returns the amount of consumed gas. For Amsterdam blocks with
// 2D gas accounting (EIP-8037), returns max(sum_regular, sum_state).
func (gp *GasPool) Used() uint64 {
	if gp.cumulativeRegular > 0 || gp.cumulativeState > 0 {
		return max(gp.cumulativeRegular, gp.cumulativeState)
	}
	if gp.initial < gp.remaining {
		panic(fmt.Sprintf("gas used underflow: %v %v", gp.initial, gp.remaining))
	}
	return gp.initial - gp.remaining
}

// Snapshot returns the deep-copied object as the snapshot.
func (gp *GasPool) Snapshot() *GasPool {
	return &GasPool{
		initial:           gp.initial,
		remaining:         gp.remaining,
		cumulativeUsed:    gp.cumulativeUsed,
		cumulativeRegular: gp.cumulativeRegular,
		cumulativeState:   gp.cumulativeState,
	}
}

// Set sets the content of gasPool with the provided one.
func (gp *GasPool) Set(other *GasPool) {
	gp.initial = other.initial
	gp.remaining = other.remaining
	gp.cumulativeUsed = other.cumulativeUsed
	gp.cumulativeRegular = other.cumulativeRegular
	gp.cumulativeState = other.cumulativeState
}

// AmsterdamDimensions returns the per-dimension cumulative gas values
// for 2D gas accounting (EIP-8037).
func (gp *GasPool) AmsterdamDimensions() (regular, state uint64) {
	return gp.cumulativeRegular, gp.cumulativeState
}

func (gp *GasPool) String() string {
	return fmt.Sprintf("initial: %d, remaining: %d, cumulative used: %d", gp.initial, gp.remaining, gp.cumulativeUsed)
}
