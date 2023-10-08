// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// precompileManager is used as a default PrecompileManager for the EVM.
type precompileManager struct {
}

// NewPrecompileManager returns a new PrecompileManager for the current chain rules.
func NewPrecompileManager() PrecompileManager {
	return &precompileManager{}
}

// Get returns the precompiled contract deployed at the given address.
func (pm *precompileManager) Get(addr common.Address, rules *params.Rules) (PrecompiledContract, bool) {
	return nil, false
}

// GetActive sets the chain rules on the precompile manager and returns the list of active
// precompile addresses.
func (pm *precompileManager) GetActive(rules params.Rules) []common.Address {
	return nil
}

// Run runs the given precompiled contract with the given input data and returns the remaining gas.
func (pm *precompileManager) Run(
	evm PrecompileEVM, p PrecompiledContract, input []byte,
	caller common.Address, value *big.Int, suppliedGas uint64, _ bool,
) (ret []byte, remainingGas uint64, err error) {
	gasCost := p.RequiredGas(input)
	if gasCost > suppliedGas {
		return nil, 0, ErrOutOfGas
	}

	suppliedGas -= gasCost
	output, err := p.Run(context.Background(), evm, input, caller, value)

	return output, suppliedGas, err
}
