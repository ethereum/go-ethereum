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

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/log"
)

// GasCosts denotes a vector of gas costs in the multidimensional metering
// paradigm. It represents the cost charged by an individual operation.
type GasCosts struct {
	ExecutionGas uint64
	StateGas     uint64
}

// Sum returns the total gas (execution + state).
func (g GasCosts) Sum() uint64 {
	return g.ExecutionGas + g.StateGas
}

// String returns a visual representation of the gas vector.
func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.ExecutionGas, g.StateGas)
}

// GasBudget is the unified gas-state structure used throughout the EVM.
// It carries two pairs of fields:
//
//   - ExecutionGas / StateGas: the running balance during execution, or the
//     leftover balance the caller must absorb after a sub-call.
//   - UsedExecutionGas / UsedStateGas: per-frame accumulators tracking gross
//     consumption. UsedStateGas is signed so it can be decremented by inline
//     state-gas refunds (e.g., SSTORE 0->A->0).
type GasBudget struct {
	ExecutionGas     uint64 // remaining execution-gas balance (or leftover for caller to absorb)
	StateGas         uint64 // remaining state-gas reservoir (or leftover for caller to absorb)
	UsedExecutionGas uint64 // gross execution gas consumed in this frame
	UsedStateGas     int64  // signed net state-gas consumed in this frame

	// Spilled tracks how much of this frame's execution gas (gas_left)
	// has been borrowed to cover state-gas charges that exceeded the
	// reservoir.
	Spilled uint64
}

// NewGasBudget initializes a fresh GasBudget for execution / forwarding,
// with both usage accumulators set to zero.
func NewGasBudget(execution, state uint64) GasBudget {
	return GasBudget{ExecutionGas: execution, StateGas: state}
}

// Used returns the total scalar gas consumed relative to an initial budget.
func (g GasBudget) Used(initial GasBudget) uint64 {
	return (initial.ExecutionGas + initial.StateGas) - (g.ExecutionGas + g.StateGas)
}

// String returns a visual representation of the budget.
func (g GasBudget) String() string {
	return fmt.Sprintf("<%v,%v,used=<%v,%v>,borrowed=%v>", g.ExecutionGas, g.StateGas, g.UsedExecutionGas, g.UsedStateGas, g.Spilled)
}

// Charge deducts a combined execution+state cost from the running balance and
// updates the usage accumulators.
func (g *GasBudget) Charge(cost GasCosts) (GasBudget, bool) {
	prior := *g
	ok := g.charge(cost)
	return prior, ok
}

// ChargeExecutionOnly deducts a execution-only cost. It's always preferred for
// performance consideration if the opcode doesn't have any state cost.
func (g *GasBudget) ChargeExecutionOnly(r uint64) error {
	if g.ExecutionGas < r {
		return ErrOutOfGas
	}
	g.ExecutionGas -= r
	g.UsedExecutionGas += r
	return nil
}

// CanAfford reports whether the running budget can cover the given cost vector
// without going out of gas.
func (g GasBudget) CanAfford(cost GasCosts) bool {
	if g.ExecutionGas < cost.ExecutionGas {
		return false
	}
	execution := g.ExecutionGas - cost.ExecutionGas
	if cost.StateGas > g.StateGas {
		return cost.StateGas-g.StateGas <= execution
	}
	return true
}

// charge deducts both the state and execution cost.
func (g *GasBudget) charge(cost GasCosts) bool {
	if g.ExecutionGas < cost.ExecutionGas {
		return false
	}
	execution := g.ExecutionGas - cost.ExecutionGas
	state := g.StateGas
	spilled := g.Spilled

	if cost.StateGas > state {
		spillover := cost.StateGas - state
		if spillover > execution {
			return false
		}
		execution -= spillover
		state = 0
		spilled += spillover
	} else {
		state -= cost.StateGas
	}
	g.ExecutionGas = execution
	g.StateGas = state
	g.UsedExecutionGas += cost.ExecutionGas
	g.UsedStateGas += int64(cost.StateGas)
	g.Spilled = spilled
	return true
}

// AsTracing converts the GasBudget into the tracing-facing Gas vector.
func (g GasBudget) AsTracing() tracing.Gas {
	return tracing.Gas{Execution: g.ExecutionGas, State: g.StateGas}
}

// ChargeExecution is a convenience that deducts a execution-only cost.
func (g *GasBudget) ChargeExecution(r uint64) (GasBudget, bool) {
	return g.Charge(GasCosts{ExecutionGas: r})
}

// ChargeState is a convenience that deducts a state-only cost.
func (g *GasBudget) ChargeState(s uint64) (GasBudget, bool) {
	return g.Charge(GasCosts{StateGas: s})
}

// IsZero returns an indicator if the gas budget has been exhausted.
func (g *GasBudget) IsZero() bool {
	return g.ExecutionGas == 0 && g.StateGas == 0
}

// RefundState applies an inline state-gas refund (e.g., SSTORE 0->A->0).
func (g *GasBudget) RefundState(s uint64) {
	repay := min(s, g.Spilled)
	g.ExecutionGas += repay
	g.Spilled -= repay
	g.StateGas += s - repay
	g.UsedStateGas -= int64(s)
}

// DrainExecution burns the remaining execution-gas.
func (g *GasBudget) DrainExecution() {
	g.UsedExecutionGas += g.ExecutionGas
	g.ExecutionGas = 0
}

// Forward drains `execution` gas and the entire state reservoir from
// the parent's running budget and returns the initial GasBudget for a child
// frame. The parent's UsedExecutionGas is bumped by the forwarded amount so
// that the absorb-on-return path correctly reclaims the unused portion.
func (g *GasBudget) Forward(execution uint64) GasBudget {
	g.ExecutionGas -= execution
	g.UsedExecutionGas += execution

	child := GasBudget{
		ExecutionGas: execution,
		StateGas:     g.StateGas,
	}
	g.StateGas = 0
	return child
}

// ForwardAll forwards the parent's full remaining budget (both execution and
// state) to a child frame. Equivalent to Forward(g.ExecutionGas) — used at
// the tx boundary where there is no 1/64 retention.
func (g *GasBudget) ForwardAll() GasBudget {
	return g.Forward(g.ExecutionGas)
}

// ============================================================================
// Exit-form constructors. These take a post-execution running budget and
// produce a new GasBudget in "leftover form", the value the caller should
// absorb to update its own state.
// ============================================================================

// ExitSuccess produces the leftover form for a successful frame.
func (g GasBudget) ExitSuccess() GasBudget {
	return g
}

// ExitRevert produces the leftover for a REVERT exit. The frame's state
// changes are discarded, so all state gas it charged is refilled with LIFO
// mechanism: up to Spilled is returned to ExecutionGas (the execution gas it
// borrowed), and the remainder restores the reservoir.
func (g GasBudget) ExitRevert() GasBudget {
	reservoir := int64(g.StateGas) + g.UsedStateGas - int64(g.Spilled)
	if reservoir < 0 {
		// Reservoir should never be negative. By construction it equals
		// the initial state-gas allocation.
		reservoir = 0
		log.Warn("Negative reservoir at revert", "remaining", g.StateGas, "used", g.UsedStateGas, "borrowed", g.Spilled)
	}
	return GasBudget{
		ExecutionGas:     g.ExecutionGas + g.Spilled,
		StateGas:         uint64(reservoir),
		UsedExecutionGas: g.UsedExecutionGas,
		UsedStateGas:     0,
		Spilled:          0,
	}
}

// ExitHalt produces the leftover for an exceptional halt. As with a revert, the
// frame's state changes are rolled back and its state gas is refilled with LIFO
// mechanism. The difference is that the frame's execution gas is consumed rather
// than returned. The portion refilled to ExecutionGas is therefore burned along
// with the rest of execution gas, leaving only the reservoir portion to survive,
// which equals the reservoir's value at the start of the frame.
func (g GasBudget) ExitHalt() GasBudget {
	reservoir := int64(g.StateGas) + g.UsedStateGas - int64(g.Spilled)
	if reservoir < 0 {
		// Reservoir should never be negative. By construction it equals
		// the initial state-gas allocation.
		reservoir = 0
		log.Warn("Negative reservoir at halt", "remaining", g.StateGas, "used", g.UsedStateGas, "borrowed", g.Spilled)
	}
	return GasBudget{
		ExecutionGas:     0,
		StateGas:         uint64(reservoir),
		UsedExecutionGas: g.UsedExecutionGas + g.ExecutionGas + g.Spilled,
		UsedStateGas:     0,
		Spilled:          0,
	}
}

// Exit dispatches on err to the appropriate exit-form constructor
// for the post-evm.Run path:
//
//   - err == nil                  → ExitSuccess
//   - err == ErrExecutionReverted → ExitRevert
//   - any other err               → ExitHalt
func (g GasBudget) Exit(err error) GasBudget {
	switch {
	case err == nil:
		return g.ExitSuccess()
	case err == ErrExecutionReverted:
		return g.ExitRevert()
	default:
		return g.ExitHalt()
	}
}

// Absorb merges a sub-call's leftover GasBudget into this (caller's) running
// budget. Additionally, it does an EIP-8037 spillover correction:
// state-gas that spilled into the execution pool inside the child frame is
// excluded from the UsedExecutionGas.
func (g *GasBudget) Absorb(child GasBudget) {
	g.UsedExecutionGas -= child.ExecutionGas
	g.ExecutionGas += child.ExecutionGas
	g.StateGas = child.StateGas
	g.UsedStateGas += child.UsedStateGas

	g.UsedExecutionGas -= child.Spilled
	g.Spilled += child.Spilled
}
