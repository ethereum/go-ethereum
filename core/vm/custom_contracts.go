package vm

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ActivePrecompiles returns the precompiles enabled with the current configuration.
func (evm *EVM) ActivePrecompiles() []common.Address {
	addrs := make([]common.Address, len(evm.precompiles))
	i := 0
	for addr, _ := range evm.precompiles {
		addrs[i] = addr
		i++
	}
	return addrs
}

// Precompile returns a precompiled contract for the given address. This
// function returns false if the address is not a registered precompile.
func (evm *EVM) Precompile(addr common.Address) (PrecompiledContract, bool) {
	p, ok := evm.precompiles[addr]
	return p, ok
}

// WithPrecompiles sets the precompiled contracts and the slice of actives precompiles.
// IMPORTANT: This function does NOT validate the precompiles provided to the EVM. The caller should
// use the ValidatePrecompiles function for this purpose prior to calling WithPrecompiles.
func (evm *EVM) WithPrecompiles(precompiles map[common.Address]PrecompiledContract) {
	evm.precompiles = precompiles
}

// ValidatePrecompiles validates the precompile map against the active
// precompile slice.
// It returns an error if the precompiled contract map has a different length
// than the slice of active contract addresses. This function also checks for
// duplicates, invalid addresses and empty precompile contract instances.
func ValidatePrecompiles(
	precompiles PrecompiledContracts,
) error {
	dupActivePrecompiles := make(map[common.Address]bool)

	for addr, precompile := range precompiles {
		if dupActivePrecompiles[addr] {
			return fmt.Errorf("duplicate active precompile: %s", addr)
		}

		if precompile == nil {
			return fmt.Errorf("precompile contract cannot be nil: %s", addr)
		}

		if bytes.Equal(addr.Bytes(), common.Address{}.Bytes()) {
			return fmt.Errorf("precompile cannot be the zero address: %s", addr)
		}

		if !bytes.Equal(addr.Bytes(), precompile.Address().Bytes()) {
			return fmt.Errorf("precompile address mismatch: %s != %s", addr, precompile.Address())
		}

		dupActivePrecompiles[addr] = true
	}

	return nil
}
