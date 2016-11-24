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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// EVMCallContext is the backbone of the Ethereum Virtual Machine allowing
// Subcalls, creation and delegate calls. It limits the EVM call depth and
// takes care of all value transfers during the transations.
//
// EVMCallContext implements the CallContext interface.
type EVMCallContext struct {
	// CanTransfer returns whether the address has enough ether to make the transfer happen
	CanTransfer func(db vm.StateDB, addr common.Address, amount *big.Int) bool
	// Transfer transfers amount of ether from sender to recipient.
	Transfer func(db vm.StateDB, sender, recipient common.Address, amount *big.Int)
	// Get Hash returns the nth hash in the blockchain.
	GetHashFn func(n uint64) common.Hash
}

func (c EVMCallContext) GetHash(n uint64) common.Hash {
	return c.GetHashFn(n)
}

// Call executes within the given contract
func (c EVMCallContext) Call(env *vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, value *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, vm.DepthError
	}
	if !c.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.StateDB.GetBalance(caller.Address()))
	}

	var (
		to                  vm.Account
		snapshotPreTransfer = env.StateDB.Snapshot()
	)
	if !env.StateDB.Exist(addr) {
		if vm.Precompiled[addr.Str()] == nil && env.ChainConfig().IsEIP158(env.BlockNumber) && value.BitLen() == 0 {
			caller.ReturnGas(gas)
			return nil, nil
		}

		to = env.StateDB.CreateAccount(addr)
	} else {
		to = env.StateDB.GetAccount(addr)
	}
	c.Transfer(env.StateDB, caller.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err = env.EVM().Run(contract, input)
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshotPreTransfer)
	}
	return ret, err
}

// CallCode executes the given address' code as the given contract address
func (c EVMCallContext) CallCode(env *vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, value *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, vm.DepthError
	}
	if !c.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.StateDB.GetBalance(caller.Address()))
	}

	var (
		snapshotPreTransfer = env.StateDB.Snapshot()
		to                  = env.StateDB.GetAccount(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err = env.EVM().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshotPreTransfer)
	}

	return ret, err
}

// Create creates a new contract with the given code
func (c EVMCallContext) Create(env *vm.Environment, caller vm.ContractRef, code []byte, gas, value *big.Int) (ret []byte, address common.Address, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, common.Address{}, vm.DepthError
	}
	if !c.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, common.Address{}, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.StateDB.GetBalance(caller.Address()))
	}

	// Create a new account on the state
	nonce := env.StateDB.GetNonce(caller.Address())
	env.StateDB.SetNonce(caller.Address(), nonce+1)

	snapshotPreTransfer := env.StateDB.Snapshot()
	var (
		addr = crypto.CreateAddress(caller.Address(), nonce)
		to   = env.StateDB.CreateAccount(addr)
	)
	if env.ChainConfig().IsEIP158(env.BlockNumber) {
		env.StateDB.SetNonce(addr, 1)
	}
	c.Transfer(env.StateDB, caller.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, crypto.Keccak256Hash(code), code)
	defer contract.Finalise()

	ret, err = env.EVM().Run(contract, nil)
	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		dataGas := big.NewInt(int64(len(ret)))
		dataGas.Mul(dataGas, params.CreateDataGas)
		if contract.UseGas(dataGas) {
			env.StateDB.SetCode(addr, ret)
		} else {
			err = vm.CodeStoreOutOfGasError
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded ||
		(err != nil && (env.ChainConfig().IsHomestead(env.BlockNumber) || err != vm.CodeStoreOutOfGasError)) {
		contract.UseGas(contract.Gas)
		env.StateDB.RevertToSnapshot(snapshotPreTransfer)

		// Nothing should be returned when an error is thrown.
		return nil, addr, err
	}

	return ret, addr, err
}

// DelegateCall is equivalent to CallCode except that sender and value propagates from parent scope to child scope
func (c EVMCallContext) DelegateCall(env *vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)
		return nil, vm.DepthError
	}

	var (
		snapshot = env.StateDB.Snapshot()
		to       = env.StateDB.GetAccount(caller.Address())
	)

	// Iinitialise a new contract and make initialise the delegate values
	contract := vm.NewContract(caller, to, caller.Value(), gas).AsDelegate()
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err = env.EVM().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshot)
	}

	return ret, err
}
