// Copyright 2016 The go-ethereum Authors
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	state.StateDBI
}

// CallContext provides a basic interface for the EVM calling conventions. The EVM
// depends on this context being implemented for doing subcalls and initialising new EVM contracts.
type CallContext interface {
	// Call calls another contract.
	Call(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// CallCode takes another contracts code and execute within our own context
	CallCode(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// DelegateCall is same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *EVM, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create creates a new contract
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error)
}

type (
	// PrecompileManager allows the EVM to execute a precompiled contract.
	PrecompileManager interface {
		// `Get` returns the precompiled contract at `addr`. Returns nil if no
		// contract is found at `addr`.
		Get(addr common.Address, rules *params.Rules) (PrecompiledContract, bool)

		GetActive(params.Rules) []common.Address

		// `Run` runs a precompiled contract and returns the remaining gas.
		Run(evm PrecompileEVM, p PrecompiledContract, input []byte, caller common.Address,
			value *big.Int, suppliedGas uint64, readonly bool,
		) (ret []byte, remainingGas uint64, err error)
	}

	// PrecompileEVM is the interface through which stateful precompiles can call back into the EVM.
	PrecompileEVM interface {
		GetStateDB() StateDB

		Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)
		StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)
		Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
		Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *uint256.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
		GetContext() *BlockContext
	}
)
