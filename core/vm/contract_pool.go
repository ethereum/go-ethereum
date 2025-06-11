package vm

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var contractPool = sync.Pool{
	New: func() any {
		return &Contract{}
	},
}

// GetContract returns a contract from the pool or creates a new one
func GetContract(caller common.Address, address common.Address, value *uint256.Int, gas uint64, jumpDests map[common.Hash]bitvec) *Contract {
	contract := contractPool.Get().(*Contract)

	// Reset the contract with new values
	contract.caller = caller
	contract.address = address
	contract.jumpdests = jumpDests
	if contract.jumpdests == nil {
		// Initialize the jump analysis map if it's nil, mostly for tests
		contract.jumpdests = make(map[common.Hash]bitvec)
	}
	contract.Gas = gas
	contract.value = value

	contract.Code = nil
	contract.CodeHash = common.Hash{}
	contract.Input = nil
	contract.IsDeployment = false
	contract.IsSystemCall = false

	contract.analysis = nil

	return contract
}

// ReturnContract returns a contract to the pool
func ReturnContract(contract *Contract) {
	if contract == nil {
		return
	}
	contractPool.Put(contract)
}
