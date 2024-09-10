package libevm_test

import (
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/libevm"
)

// IMPORTANT: if any of these break then the libevm copy MUST be updated.

// These two interfaces MUST be identical.
var (
	// Each assignment demonstrates that the methods of the LHS interface are a
	// (non-strict) subset of the RHS interface's; both being possible
	// proves that they are identical.
	_ vm.PrecompiledContract     = (libevm.PrecompiledContract)(nil)
	_ libevm.PrecompiledContract = (vm.PrecompiledContract)(nil)
)

// StateReader MUST be a subset vm.StateDB.
var _ libevm.StateReader = (vm.StateDB)(nil)
