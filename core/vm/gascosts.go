package vm

import "fmt"

type GasCosts struct {
	RegularGas uint64
	StateGas   uint64
}

// GasUsed tracks per-frame gas usage metrics for EIP-8037 2D block gas accounting.
type GasUsed struct {
	// RegularGasUsed accumulates all gas charged via charge_gas() in the spec:
	// State-gas spillover (when StateGas is exhausted and the excess
	// is paid from RegularGas) does NOT increment RegularGasUsed.
	RegularGasUsed uint64
	// StateGasCharged accumulates all gas charged via charge_state_gas()
	// On child error the charged state gas is restored to the parent's state gas reservoir.
	StateGasCharged uint64
}

func (g *GasUsed) Add(cost GasCosts) {
	g.RegularGasUsed += cost.RegularGas
	g.StateGasCharged += cost.StateGas
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
	if b.StateGas > g.StateGas {
		diff := b.StateGas - g.StateGas
		g.StateGas = 0
		g.RegularGas -= diff
		// Note: spillover does NOT increment RegularGasUsed, matching the spec
	} else {
		g.StateGas -= b.StateGas
	}
}

// Add doesn't check for overflows
func (g *GasCosts) Add(b GasCosts) {
	g.RegularGas += b.RegularGas
	g.StateGas += b.StateGas
}

func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
