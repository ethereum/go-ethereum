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

type GasUsed = GasCosts

func (g *GasUsed) Add(costs GasCosts) {
	g.RegularGas += costs.RegularGas
	g.StateGas += costs.StateGas
}

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

// NewGasBudgetReg creates a GasBudget with the given initial regular gas allowance.
func NewGasBudgetReg(gas uint64) GasBudget {
	return GasBudget{RegularGas: gas}
}

func NewGasBudget(regular, state uint64) GasBudget {
	return GasBudget{RegularGas: regular, StateGas: state}
}

// Used returns the total amount of gas consumed so far (regular + state).
func (g GasBudget) Used(initial GasBudget) uint64 {
	return (initial.RegularGas + initial.StateGas) - (g.RegularGas + g.StateGas)
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
// When state gas exceeds the reservoir, the excess spills to regular gas.
func (g GasBudget) CanAfford(cost GasCosts) bool {
	if g.RegularGas < cost.RegularGas {
		return false
	}
	if cost.StateGas > g.StateGas {
		spillover := cost.StateGas - g.StateGas
		if spillover > g.RegularGas-cost.RegularGas {
			return false
		}
	}
	return true
}

// Charge deducts the given gas cost from the budget. It returns the
// pre-charge regular gas value and false if the budget does not have
// sufficient gas to cover the cost.
func (g *GasBudget) Charge(cost GasCosts) (uint64, bool) {
	prior := g.RegularGas
	if !g.CanAfford(cost) {
		return prior, false
	}
	g.RegularGas -= cost.RegularGas
	if cost.StateGas > g.StateGas {
		spillover := cost.StateGas - g.StateGas
		g.StateGas = 0
		g.RegularGas -= spillover
	} else {
		g.StateGas -= cost.StateGas
	}
	return prior, true
}

// Refund adds the given gas budget back. It returns the pre-refund regular gas
// value and whether the budget was actually changed.
func (g *GasBudget) Refund(other GasBudget) (uint64, bool) {
	prior := g.RegularGas
	g.RegularGas += other.RegularGas
	g.StateGas += other.StateGas
	return prior, other.RegularGas != 0 || other.StateGas != 0
}
