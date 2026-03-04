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

// Gas returns the amount of gas remaining in the pool.
func (gp *GasPool) Gas() uint64 {
	return gp.remaining
}

// CumulativeUsed returns the amount of cumulative consumed gas (refunded included).
func (gp *GasPool) CumulativeUsed() uint64 {
	return gp.cumulativeUsed
}

// Used returns the amount of consumed gas.
func (gp *GasPool) Used() uint64 {
	if gp.initial < gp.remaining {
		panic("gas used underflow")
	}
	return gp.initial - gp.remaining
}

// Snapshot returns the deep-copied object as the snapshot.
func (gp *GasPool) Snapshot() *GasPool {
	return &GasPool{
		initial:        gp.initial,
		remaining:      gp.remaining,
		cumulativeUsed: gp.cumulativeUsed,
	}
}

// Set sets the content of gasPool with the provided one.
func (gp *GasPool) Set(other *GasPool) {
	gp.initial = other.initial
	gp.remaining = other.remaining
	gp.cumulativeUsed = other.cumulativeUsed
}

func (gp *GasPool) String() string {
	return fmt.Sprintf("initial: %d, remaining: %d, cumulative used: %d", gp.initial, gp.remaining, gp.cumulativeUsed)
}
