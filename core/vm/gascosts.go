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
)

// GasCosts denotes a vector of gas costs in the multidimensional metering
// paradigm. It represents the cost charged by an individual operation.
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

// GasBudget is the unified gas-state structure used throughout the EVM.
// It carries two pairs of fields:
//
//   - RegularGas / StateGas: the running balance during execution, or the
//     leftover balance the caller must absorb after a sub-call.
//   - UsedRegularGas / UsedStateGas: per-frame accumulators tracking gross
//     consumption. UsedStateGas is signed so it can be decremented by inline
//     state-gas refunds (e.g., SSTORE 0->A->0).
//
// The same struct serves three roles:
//
//   - During execution: Charge / ChargeRegular / ChargeState / RefundState
//     and RefundRegular mutate the running balance and the usage accumulators
//     in lockstep.
//
//   - At frame exit: Preserved / ExitSuccess / ExitRevert / ExitHalt /
//     Exit produce a new GasBudget in "leftover" form that packages
//     the result for the caller.
//
//   - At absorption: the caller's Absorb method merges the child's leftover
//     budget into its own running budget.
type GasBudget struct {
	RegularGas     uint64 // remaining regular-gas balance (or leftover for caller to absorb)
	StateGas       uint64 // remaining state-gas reservoir (or leftover for caller to absorb)
	UsedRegularGas uint64 // gross regular gas consumed in this frame
	UsedStateGas   int64  // signed net state-gas consumed in this frame
}

// NewGasBudget initializes a fresh GasBudget for execution / forwarding,
// with both usage accumulators set to zero.
func NewGasBudget(regular, state uint64) GasBudget {
	return GasBudget{RegularGas: regular, StateGas: state}
}

// Used returns the total scalar gas consumed relative to an initial budget
// (= (initial.regular + initial.state) − (current.regular + current.state)).
// This is the payment scalar (EIP-8037's tx_gas_used_before_refund).
//
// TODO(rjl493456442) the total used gas can be calculated via g.UsedRegularGas
// and g.UsedStateGas.
func (g GasBudget) Used(initial GasBudget) uint64 {
	return (initial.RegularGas + initial.StateGas) - (g.RegularGas + g.StateGas)
}

// Copy returns a deep copy of the budget.
func (g GasBudget) Copy() GasBudget {
	return g
}

// String returns a visual representation of the budget.
func (g GasBudget) String() string {
	return fmt.Sprintf("<%v,%v,used=<%v,%v>>", g.RegularGas, g.StateGas, g.UsedRegularGas, g.UsedStateGas)
}

// CanAfford reports whether the running balance can cover the given cost.
// State-gas charges that exceed the reservoir spill into regular gas.
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

// Charge deducts a combined regular+state cost from the running balance and
// updates the usage accumulators. State-gas in excess of the reservoir spills
// into regular_gas.
func (g *GasBudget) Charge(cost GasCosts) (GasBudget, bool) {
	prior := *g
	if !g.CanAfford(cost) {
		return prior, false
	}
	// Charge regular gas
	g.RegularGas -= cost.RegularGas
	g.UsedRegularGas += cost.RegularGas

	// Charge state gas
	if cost.StateGas > g.StateGas {
		spillover := cost.StateGas - g.StateGas
		g.StateGas = 0
		g.RegularGas -= spillover
	} else {
		g.StateGas -= cost.StateGas
	}
	g.UsedStateGas += int64(cost.StateGas)
	return prior, true
}

// AsTracing converts the GasBudget into the tracing-facing Gas vector.
func (g GasBudget) AsTracing() tracing.Gas {
	return tracing.Gas{Regular: g.RegularGas, State: g.StateGas}
}

// ChargeRegular is a convenience that deducts a regular-only cost.
func (g *GasBudget) ChargeRegular(r uint64) (GasBudget, bool) {
	return g.Charge(GasCosts{RegularGas: r})
}

// ChargeState is a convenience that deducts a state-only cost (spills to
// regular when the reservoir is exhausted). Returns false on OOG.
func (g *GasBudget) ChargeState(s uint64) (GasBudget, bool) {
	return g.Charge(GasCosts{StateGas: s})
}

// IsZero returns an indicator if the gas budget has been exhausted.
func (g *GasBudget) IsZero() bool {
	return g.RegularGas == 0 && g.StateGas == 0
}

// RefundState applies an inline state-gas refund (e.g., SSTORE 0->A->0).
// The reservoir is credited and the signed usage counter is decremented
// in lockstep, preserving the per-frame invariant:
//
//	StateGas + UsedStateGas == initialStateGas + spillover_so_far
//
// which the revert path relies on for the correct gross refund.
func (g *GasBudget) RefundState(s uint64) {
	g.StateGas += s
	g.UsedStateGas -= int64(s)
}

// RefundRegular applies an inline regular-gas refund.
func (g *GasBudget) RefundRegular(s uint64) {
	g.RegularGas += s
	g.UsedRegularGas -= s
}

// Forward drains `regular` regular gas and the entire state reservoir from
// the parent's running budget and returns the initial GasBudget for a child
// frame. The parent's UsedRegularGas is bumped by the forwarded amount so
// that the absorb-on-return path correctly reclaims the unused portion.
//
// Used by frame boundaries where the regular forward has NOT been pre-
// deducted: tx-level dispatch (state_transition) and CREATE / CREATE2. The
// CALL family pre-deducts the forward via the dynamic gas table for tracer-
// reporting reasons and therefore constructs its child budget directly.
//
// Caller must ensure `regular` does not exceed the running balance and
// apply any EIP-150 1/64 retention before calling Forward.
func (g *GasBudget) Forward(regular uint64) GasBudget {
	g.RegularGas -= regular
	g.UsedRegularGas += regular

	child := GasBudget{
		RegularGas: regular,
		StateGas:   g.StateGas,
	}
	g.StateGas = 0
	return child
}

// ForwardAll forwards the parent's full remaining budget (both regular and
// state) to a child frame. Equivalent to Forward(g.RegularGas) — used at
// the tx boundary where there is no 1/64 retention.
func (g *GasBudget) ForwardAll() GasBudget {
	return g.Forward(g.RegularGas)
}

// ============================================================================
// Exit-form constructors. These take a post-execution running budget and
// produce a new GasBudget in "leftover form" — the value the caller should
// absorb to update its own state.
// ============================================================================

// Preserved produces a leftover form with the running balance preserved and
// usage zeroed. Use this for pre-execution validation failures (depth,
// balance, EIP-158 zero-value-to-nonexistent) where no execution actually
// occurred.
func (g GasBudget) Preserved() GasBudget {
	return GasBudget{
		RegularGas: g.RegularGas,
		StateGas:   g.StateGas,
		// UsedRegularGas / UsedStateGas stay at zero.
	}
}

// ExitSuccess produces the leftover form for a successful frame. Inline
// state-gas refunds have already been folded into StateGas / UsedStateGas
// during execution; the running budget IS the exit budget on success.
func (g GasBudget) ExitSuccess() GasBudget {
	return g
}

// ExitRevert produces the leftover form for a REVERT exit. Per EIP-8037,
// the gross state-gas charged in the failing subtree is refunded to the
// caller's reservoir via the signed-counter formula
//
//	leftover.StateGas = StateGas + UsedStateGas
//
// and UsedStateGas is zeroed because the state effect is routed through
// StateGas. UsedRegularGas is unchanged.
func (g GasBudget) ExitRevert() GasBudget {
	reservoir := int64(g.StateGas) + g.UsedStateGas
	if reservoir < 0 {
		// Defensive: invariant guarantees non-negativity. Clamping prevents
		// a hypothetical bug from sign-extending to a huge uint64.
		reservoir = 0
	}
	return GasBudget{
		RegularGas:     g.RegularGas,
		StateGas:       uint64(reservoir),
		UsedRegularGas: g.UsedRegularGas,
		UsedStateGas:   0,
	}
}

// ExitHalt produces the leftover form for an exceptional halt. Remaining
// regular gas is burned into UsedRegularGas; the state dimension follows the
// same revert-style formula as ExitRevert because the spec routes both halt
// and revert through incorporate_child_on_error:
//
//	parent.state_gas_left = parent + child.state_gas_used + child.state_gas_left
//
// which in our model means returning `StateGas + UsedStateGas` to the parent
// and zeroing the per-frame counter.
func (g GasBudget) ExitHalt() GasBudget {
	reservoir := int64(g.StateGas) + g.UsedStateGas
	if reservoir < 0 {
		reservoir = 0
	}
	return GasBudget{
		RegularGas:     0,
		StateGas:       uint64(reservoir),
		UsedRegularGas: g.UsedRegularGas + g.RegularGas,
		UsedStateGas:   0,
	}
}

// Exit dispatches on err to the appropriate exit-form constructor
// for the post-evm.Run path:
//
//   - err == nil                  → ExitSuccess
//   - err == ErrExecutionReverted → ExitRevert
//   - any other err               → ExitHalt
//
// Soft validation failures (occurring BEFORE evm.Run) should call Preserved
// directly instead of going through this dispatcher.
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
// budget. The caller's UsedRegularGas is reclaimed by the unused forwarded
// regular gas (which was pre-charged in full at call entry); the state
// reservoir is overwritten with the child's leftover; and the child's signed
// net state-gas usage is added to the caller's accumulator.
//
// Invariant maintained by all callers: at the moment of this call, the
// caller's UsedRegularGas already accounts for the FULL forwarded regular
// gas (as if the child had consumed all of it). On halt, child.RegularGas
// is 0 so the reclaim is a no-op.
func (g *GasBudget) Absorb(child GasBudget) {
	g.UsedRegularGas -= child.RegularGas
	g.RegularGas += child.RegularGas
	g.StateGas = child.StateGas
	g.UsedStateGas += child.UsedStateGas
}
