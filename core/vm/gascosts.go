package vm

import "fmt"

type GasCosts struct {
	RegularGas uint64
	StateGas   uint64
}

// Sum returns the total gas (regular + state).
func (g GasCosts) Sum() uint64 {
	return g.RegularGas + g.StateGas
}

func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
