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

package vm

import "fmt"

// GasCosts denotes a vector of gas costs in the
// multidimensional metering paradigm. It represents the cost
// charged by an individual operation.
type GasCosts struct {
	RegularGas uint64
	StateGas   uint64
}

// Sum returns the total gas (regular + state).
func (g GasCosts) Sum() uint64 {
	return g.RegularGas + g.StateGas
}

// String returns a visual representation of the gas vector.
func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}

// GasBudget denotes a vector of remaining gas allowances available
// for EVM execution in the multidimensional metering paradigm.
// Unlike GasCosts which represents the price of an operation,
// GasBudget tracks how much gas is left to spend.
type GasBudget struct {
	RegularGas uint64 // The leftover gas for execution and state gas usage
	StateGas   uint64 // The state gas reservoir
}

// NewGasBudget creates a GasBudget with the given initial regular gas allowance.
func NewGasBudget(gas uint64) GasBudget {
	return GasBudget{RegularGas: gas}
}

// Used returns the amount of regular gas consumed so far.
func (g GasBudget) Used(initial GasBudget) uint64 {
	return initial.RegularGas - g.RegularGas
}

// Exhaust sets all remaining gas to zero, preserving the initial amount
// for usage tracking.
func (g *GasBudget) Exhaust() {
	g.RegularGas = 0
	g.StateGas = 0
}

func (g *GasBudget) Copy() GasBudget {
	return GasBudget{RegularGas: g.RegularGas, StateGas: g.StateGas}
}

// String returns a visual representation of the gas budget vector.
func (g GasBudget) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}

// CanAfford reports whether the budget has sufficient gas to cover the cost.
func (g GasBudget) CanAfford(cost GasCosts) bool {
	return g.RegularGas >= cost.RegularGas
}

// Charge deducts the given gas cost from the budget. It returns the
// pre-charge gas value and false if the budget does not have sufficient
// gas to cover the cost.
func (g *GasBudget) Charge(cost GasCosts) (uint64, bool) {
	prior := g.RegularGas
	if prior < cost.RegularGas {
		return prior, false
	}
	g.RegularGas -= cost.RegularGas
	return prior, true
}

// Refund adds the given gas budget back. It returns the pre-refund gas
// value and whether the budget was actually changed.
func (g *GasBudget) Refund(other GasBudget) (uint64, bool) {
	prior := g.RegularGas
	g.RegularGas += other.RegularGas
	return prior, g.RegularGas != prior
}
