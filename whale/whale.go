// Package whale implements the WhaleChain fork
package whale

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// Apply PrimordialWhale fork changes
func PrimordialWhaleFork(state *state.StateDB, treasury *params.Treasury, chainID *big.Int) {
	applySacrificeCredits(state, treasury, chainID)
	replaceDepositContract(state)
}
