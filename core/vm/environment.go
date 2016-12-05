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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type (
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	TransferFunc    func(StateDB, common.Address, common.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

// Context provides the EVM with auxilary information. Once provided it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    *big.Int       // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// Environment provides information about external sources for the EVM
//
// The Environment should never be reused and is not thread safe.
type Environment struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	Depth int

	// evm is the ethereum virtual machine
	evm Vm
	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	vmConfig    Config
}

// NewEnvironment retutrns a new EVM environment.
func NewEnvironment(context Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *Environment {
	env := &Environment{
		Context:     context,
		StateDB:     statedb,
		vmConfig:    vmConfig,
		chainConfig: chainConfig,
	}
	env.evm = New(env, vmConfig)
	return env
}

// Call executes the contract associated with the addr with the given input as paramaters. It also handles any
// necessary value transfer required and takes the necessary steps to create accounts and reverses the state in
// case of an execution error or failed value transfer.
func (env *Environment) Call(caller ContractRef, addr common.Address, input []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.NoRecursion && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, DepthError
	}
	if !env.Context.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, ErrInsufficientBalance
	}

	var (
		to                  Account
		snapshotPreTransfer = env.StateDB.Snapshot()
	)
	if !env.StateDB.Exist(addr) {
		if Precompiled[addr.Str()] == nil && env.ChainConfig().IsEIP158(env.BlockNumber) && value.BitLen() == 0 {
			caller.ReturnGas(gas)
			return nil, nil
		}

		to = env.StateDB.CreateAccount(addr)
	} else {
		to = env.StateDB.GetAccount(addr)
	}
	env.Transfer(env.StateDB, caller.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err := env.EVM().Run(contract, input)
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshotPreTransfer)
	}
	return ret, err
}

// CallCode executes the contract associated with the addr with the given input as paramaters. It also handles any
// necessary value transfer required and takes the necessary steps to create accounts and reverses the state in
// case of an execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address' code with the caller as context.
func (env *Environment) CallCode(caller ContractRef, addr common.Address, input []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.NoRecursion && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, DepthError
	}
	if !env.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, fmt.Errorf("insufficient funds to transfer value. Req %v, has %v", value, env.StateDB.GetBalance(caller.Address()))
	}

	var (
		snapshotPreTransfer = env.StateDB.Snapshot()
		to                  = env.StateDB.GetAccount(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err := env.EVM().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshotPreTransfer)
	}

	return ret, err
}

// DelegateCall executes the contract associated with the addr with the given input as paramaters.
// It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address' code with the caller as context
// and the caller is set to the caller of the caller.
func (env *Environment) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas *big.Int) ([]byte, error) {
	if env.vmConfig.NoRecursion && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)
		return nil, DepthError
	}

	var (
		snapshot = env.StateDB.Snapshot()
		to       = env.StateDB.GetAccount(caller.Address())
	)

	// Iinitialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, caller.Value(), gas).AsDelegate()
	contract.SetCallCode(&addr, env.StateDB.GetCodeHash(addr), env.StateDB.GetCode(addr))
	defer contract.Finalise()

	ret, err := env.EVM().Run(contract, input)
	if err != nil {
		contract.UseGas(contract.Gas)

		env.StateDB.RevertToSnapshot(snapshot)
	}

	return ret, err
}

// Create creates a new contract using code as deployment code.
func (env *Environment) Create(caller ContractRef, code []byte, gas, value *big.Int) ([]byte, common.Address, error) {
	if env.vmConfig.NoRecursion && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, common.Address{}, nil
	}

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(gas)

		return nil, common.Address{}, DepthError
	}
	if !env.CanTransfer(env.StateDB, caller.Address(), value) {
		caller.ReturnGas(gas)

		return nil, common.Address{}, ErrInsufficientBalance
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
	env.Transfer(env.StateDB, caller.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, crypto.Keccak256Hash(code), code)
	defer contract.Finalise()

	ret, err := env.EVM().Run(contract, nil)
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
			err = CodeStoreOutOfGasError
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded ||
		(err != nil && (env.ChainConfig().IsHomestead(env.BlockNumber) || err != CodeStoreOutOfGasError)) {
		contract.UseGas(contract.Gas)
		env.StateDB.RevertToSnapshot(snapshotPreTransfer)

		// Nothing should be returned when an error is thrown.
		return nil, addr, err
	}
	// If the vm returned with an error the return value should be set to nil.
	// This isn't consensus critical but merely to for behaviour reasons such as
	// tests, RPC calls, etc.
	if err != nil {
		ret = nil
	}

	return ret, addr, err
}

// ChainConfig returns the environment's chain configuration
func (env *Environment) ChainConfig() *params.ChainConfig { return env.chainConfig }

// EVM returns the environments EVM
func (env *Environment) EVM() Vm { return env.evm }
