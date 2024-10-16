package exex

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// State provides read access to Geth's internal state object.
type State interface {
	// Balance retrieves the balance of the given account, or 0 if the account
	// is not found in the state.
	Balance(addr common.Address) *uint256.Int

	// Nonce retrieves the nonce of the given account, or 0 if the account is
	// not found in the state.
	Nonce(addr common.Address) uint64

	// Code retrieves the bytecode associated with the given account, or a nil
	// slice if the account is not found.
	Code(addr common.Address) []byte

	// Storage retrieves the value associated with a specific storage slot key
	// within a specific account.
	Storage(addr common.Address, slot common.Hash) common.Hash
}
