package exex

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/exex"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/holiman/uint256"
)

// stateAdapter is an adapter to convert Geth's internal state db (unstable
// and legacy API) into the exex state interface (stable API).
type stateAdapter struct {
	state *state.StateDB
}

// wrapState wraps a Geth internal state object into an exex stable API.
func wrapState(state *state.StateDB) exex.State {
	return &stateAdapter{state: state}
}

// Balance retrieves the balance of the given account, or 0 if the account is
// not found in the state.
func (a *stateAdapter) Balance(addr common.Address) *uint256.Int {
	return a.state.GetBalance(addr)
}

// Nonce retrieves the nonce of the given account, or 0 if the account is not
// found in the state.
func (a *stateAdapter) Nonce(addr common.Address) uint64 {
	return a.state.GetNonce(addr)
}

// Code retrieves the bytecode associated with the given account, or a nil slice
// if the account is not found.
func (a *stateAdapter) Code(addr common.Address) []byte {
	return common.CopyBytes(a.state.GetCode(addr))
}

// Storage retrieves the value associated with a specific storage slot key within
// a specific account.
func (a *stateAdapter) Storage(addr common.Address, slot common.Hash) common.Hash {
	return a.state.GetState(addr, slot)
}
