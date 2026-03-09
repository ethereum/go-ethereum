package vm

import "fmt"

type GasCosts struct {
	RegularGas uint64
	StateGas   uint64

	// TotalStateGasCharged tracks the cumulative state gas charged during
	// execution, including gas that spilled from the reservoir to regular gas.
	// This is needed for EIP-8037 block gas accounting where the state gas
	// dimension counts ALL state creation charges, not just reservoir consumption.
	TotalStateGasCharged uint64

	// RevertedStateGasSpill tracks state gas that was charged from regular gas
	// (spilled) during execution of a call that subsequently reverted. When a
	// call fails, its state changes are undone, but the regular gas was already
	// consumed. Block gas accounting must exclude this amount from the regular
	// gas dimension since it was for state operations that didn't persist.
	// This gas is refunded to the user (invisible to both block and receipt).
	RevertedStateGasSpill uint64

	// CollisionConsumedGas tracks regular gas consumed on CREATE/CREATE2 address
	// collision. On collision, the child's regular gas is consumed (user pays)
	// but must be excluded from block regular gas accounting to preserve 2D
	// block gas semantics. Unlike RevertedStateGasSpill, this is NOT refunded.
	CollisionConsumedGas uint64
}

func (g GasCosts) Max() uint64 {
	return max(g.RegularGas, g.StateGas)
}

func (g GasCosts) Sum() uint64 {
	return g.RegularGas + g.StateGas
}

// Underflow returns true if the operation would underflow.
// When state gas exceeds the reservoir, the excess spills to regular gas.
// The check accounts for regular gas already consumed by b.RegularGas.
func (g GasCosts) Underflow(b GasCosts) bool {
	if b.RegularGas > g.RegularGas {
		return true
	}
	if b.StateGas > g.StateGas {
		spillover := b.StateGas - g.StateGas
		remainingRegular := g.RegularGas - b.RegularGas
		if spillover > remainingRegular {
			return true
		}
	}
	return false
}

// Sub doesn't check for underflows
func (g *GasCosts) Sub(b GasCosts) {
	g.RegularGas -= b.RegularGas
	g.TotalStateGasCharged += b.StateGas
	if b.StateGas > g.StateGas {
		diff := b.StateGas - g.StateGas
		g.StateGas = 0
		g.RegularGas -= diff
	} else {
		g.StateGas -= b.StateGas
	}
}

// Add doesn't check for overflows
func (g *GasCosts) Add(b GasCosts) {
	g.RegularGas += b.RegularGas
	g.StateGas += b.StateGas
	g.TotalStateGasCharged += b.TotalStateGasCharged
	g.RevertedStateGasSpill += b.RevertedStateGasSpill
	g.CollisionConsumedGas += b.CollisionConsumedGas
}

// RevertStateGas handles state gas accounting when a call reverts (EIP-8037).
// It computes how much state gas was charged from regular gas (spill) during the
// call, and either returns it for REVERT errors or tracks it for non-REVERT errors.
func (g *GasCosts) RevertStateGas(savedTotalStateGas, savedStateGas uint64, isRevert bool) {
	chargedDuringCall := g.TotalStateGasCharged - savedTotalStateGas
	fromReservoir := savedStateGas - g.StateGas
	spilledFromRegular := chargedDuringCall - fromReservoir

	if isRevert {
		// REVERT: return the spilled state gas to regular gas since the caller
		// keeps unused gas and state operations were undone.
		g.RegularGas += spilledFromRegular
	} else {
		// Non-REVERT: regular gas is zeroed, but block accounting must exclude
		// the spill from the regular gas dimension.
		g.RevertedStateGasSpill += spilledFromRegular
	}
	g.TotalStateGasCharged = savedTotalStateGas
	g.StateGas = savedStateGas
}

func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
