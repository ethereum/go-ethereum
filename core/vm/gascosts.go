package vm

import "fmt"

type GasCosts struct {
	RegularGas uint64
	StateGas   uint64

	// StateGasCharged tracks the cumulative state gas charged during execution.
	StateGasCharged uint64
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
	g.StateGasCharged += b.StateGas
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
	g.StateGasCharged += b.StateGasCharged
}

func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
