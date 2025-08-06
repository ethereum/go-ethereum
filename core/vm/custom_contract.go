package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// AccountRef implements ContractRef.
//
// Account references are used during EVM initialisation and
// its primary use is to fetch addresses. Removing this object
// proves difficult because of the cached jump destinations which
// are fetched from the parent contract (i.e. the caller), which
// is a ContractRef.
type AccountRef common.Address

// Address casts AccountRef to an Address
func (ar AccountRef) Address() common.Address { return (common.Address)(ar) }

// NewPrecompile returns a new instance of a precompiled contract environment for the execution of EVM.
func NewPrecompile(caller, address common.Address, value *uint256.Int, gas uint64) *Contract {
	c := NewContract(caller, address, value, gas, nil)
	c.isPrecompile = true
	return c
}

// IsPrecompile returns true if the contract is a precompiled contract environment
func (c *Contract) IsPrecompile() bool {
	return c.isPrecompile
}
