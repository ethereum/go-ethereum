// Package pulse implements the PulseChain fork
package pulse

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// Apply PrimordialPulse fork changes
func PrimordialPulseFork(state *state.StateDB, treasury *params.Treasury, chainID *big.Int) {
	applySacrificeCredits(state, treasury, chainID)
	replaceDepositContract(state)
}
