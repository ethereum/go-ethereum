package precompiles

import (
	"maps"

	"github.com/ethereum/go-ethereum/common"
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	Address() common.Address
	RequiredGas(input []byte) uint64  // RequiredPrice calculates the contract gas use
	Run(input []byte) ([]byte, error) // Run runs the precompiled contract
}

// PrecompiledContracts contains the precompiled contracts supported at the given fork.
type PrecompiledContracts map[common.Address]PrecompiledContract

var CustomPrecompiledContracts = PrecompiledContracts{}

// WithCustomPrecompiles merge given precompiles with custom precompiles
func WithCustomPrecompiles(base PrecompiledContracts) PrecompiledContracts {
	result := make(PrecompiledContracts)
	maps.Copy(result, base)
	for k, v := range CustomPrecompiledContracts {
		if _, exists := result[k]; !exists {
			result[k] = v
		} else {
			panic("Precompile address collision")
		}
	}
	return result
}
