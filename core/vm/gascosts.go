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
}

func (g GasCosts) Max() uint64 {
	return max(g.RegularGas, g.StateGas)
}

func (g GasCosts) Sum() uint64 {
	return g.RegularGas + g.StateGas
}

// Sub returns true if the operation would underflow
func (g GasCosts) Underflow(b GasCosts) bool {
	if b.RegularGas > g.RegularGas {
		return true
	}
	if b.StateGas > g.StateGas {
		if b.StateGas > g.RegularGas {
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
}

func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
