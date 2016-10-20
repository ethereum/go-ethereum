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

// Call executes within the given contract
func Call(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas, gasPrice)

		return nil, vm.DepthError
	}
	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	snapshotPreTransfer := env.SnapshotDatabase()
	var (
		from = env.Db().GetAccount(caller.Address())
		to   vm.Account
	)
	if !env.Db().Exist(addr) {
		if vm.Precompiled[addr.Str()] == nil && env.ChainConfig().IsEIP158(env.BlockNumber()) && value.BitLen() == 0 {
			caller.ReturnGas(gas, gasPrice)
			return nil, nil
		}

		to = env.Db().CreateAccount(addr)
	} else {
		to = env.Db().GetAccount(addr)
	}
	env.Transfer(from, to, value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	ret, err = env.Vm().Run(contract, input)
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		contract.UseGas(contract.Gas)

		env.RevertToSnapshot(snapshotPreTransfer)
	}
	return ret, err
}

// CallCode executes the given address' code as the given contract address
func CallCode(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas, gasPrice)

		return nil, vm.DepthError
	}
	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	var (
		snapshotPreTransfer = env.SnapshotDatabase()
		to                  = env.Db().GetAccount(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	ret, err = env.Vm().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.RevertToSnapshot(snapshotPreTransfer)
	}

	return ret, err
}

// Create creates a new contract with the given code
func Create(env vm.Environment, caller vm.ContractRef, code []byte, gas, gasPrice, value *big.Int) (ret []byte, address common.Address, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, vm.DepthError
	}
	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	// Create a new account on the state
	nonce := env.Db().GetNonce(caller.Address())
	env.Db().SetNonce(caller.Address(), nonce+1)

	snapshotPreTransfer := env.SnapshotDatabase()
	var (
		addr = crypto.CreateAddress(caller.Address(), nonce)
		from = env.Db().GetAccount(caller.Address())
		to   = env.Db().CreateAccount(addr)
	)
	if env.ChainConfig().IsEIP158(env.BlockNumber()) {
		env.Db().SetNonce(addr, 1)
	}
	env.Transfer(from, to, value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(&addr, crypto.Keccak256Hash(code), code)
	defer contract.Finalise()

	ret, err = env.Vm().Run(contract, nil)
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil {
		dataGas := big.NewInt(int64(len(ret)))
		dataGas.Mul(dataGas, params.CreateDataGas)
		if contract.UseGas(dataGas) {
			env.Db().SetCode(addr, ret)
		} else {
			err = vm.CodeStoreOutOfGasError
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil && (env.ChainConfig().IsHomestead(env.BlockNumber()) || err != vm.CodeStoreOutOfGasError) {
		contract.UseGas(contract.Gas)

		env.RevertToSnapshot(snapshotPreTransfer)

		// Nothing should be returned when an error is thrown.
		return nil, addr, err
	}

	return ret, addr, err
}

// DelegateCall is equivalent to CallCode except that sender and value propagates from parent scope to child scope
func DelegateCall(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice *big.Int) (ret []byte, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas, gasPrice)
		return nil, vm.DepthError
	}

	var (
		snapshot = env.SnapshotDatabase()
		to       = env.Db().GetAccount(caller.Address())
	)

	// Iinitialise a new contract and make initialise the delegate values
	contract := vm.NewContract(caller, to, caller.Value(), gas, gasPrice).AsDelegate()
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	ret, err = env.Vm().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.RevertToSnapshot(snapshot)
	}

	return ret, err
}

// generic transfer method
func Transfer(from, to vm.Account, amount *big.Int) {
	from.SubBalance(amount)
	to.AddBalance(amount)
}
