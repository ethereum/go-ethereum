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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Vm is the basic interface for an implementation of the EVM.
type Vm interface {
	// Run should execute the given contract with the input given in in
	// and return the contract execution return bytes or an error if it
	// failed.
	Run(c *Contract, in []byte) ([]byte, error)
}

// RuleSet is an interface that defines the current rule set during the
// execution of the EVM instructions (e.g. whether it's homestead)
type RuleSet interface {
	IsHomestead(*big.Int) bool
}

// Database is a EVM database for full state querying.
type Database interface {
	GetAccount(common.Address) Account
	CreateAccount(common.Address) Account

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(*big.Int)
	GetRefund() *big.Int

	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool

	AddLog(*Log)
}

// Account represents a contract or basic ethereum account.
type Account interface {
	SubBalance(amount *big.Int)
	AddBalance(amount *big.Int)
	SetBalance(*big.Int)
	SetNonce(uint64)
	Balance() *big.Int
	Address() common.Address
	ReturnGas(uint64)
	SetCode(common.Hash, []byte)
	ForEachStorage(cb func(key, value common.Hash) bool)
	Value() *big.Int
}

// Context provides the EVM with auxilary information. Once provided it shouldn't be modified.
type Context struct {
	CallContext

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	Coinbase common.Address // Provides information for COINBASE
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	GasLimit    *big.Int // Provides information for GASLIMIT
	BlockNumber *big.Int // Provides information for NUMBER
	Time        *big.Int // Provides information for TIME
	Difficulty  *big.Int // Provides information for DIFFICULTY
}

// CallContext provides a basic interface for the EVM calling conventions. The EVM Environment
// depends on this context being implemented for doing subcalls and initialising new EVM contracts.
type CallContext interface {
	// Call another contract
	Call(env *Environment, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Take another's contract code and execute within our own context
	CallCode(env *Environment, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *Environment, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create a new contract
	Create(env *Environment, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error)
}

// Backend is the basic interface for keeping track of state and taking care of
// returning ancestory data.
type Backend interface {
	// GetHash returns the hash corresponding to n
	GetHash(n uint64) common.Hash
	// Creates a restorable snapshot
	SnapshotDatabase() int
	// Set database to previous snapshot
	RevertToSnapshot(int)
	// Get returns the database
	Get() Database
}

type Environment struct {
	Context

	Backend Backend

	ruleSet  RuleSet
	vmConfig Config

	evm Vm

	Depth int
}

func NewEnvironment(context Context, backend Backend, ruleSet RuleSet, vmCfg Config) *Environment {
	env := &Environment{
		Context:  context,
		Backend:  backend,
		vmConfig: vmCfg,
		ruleSet:  ruleSet,
	}
	env.evm = New(env, vmCfg)
	return env
}

func (env *Environment) Call(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas.Uint64())

		return nil, nil
	}

	return env.CallContext.Call(env, caller, addr, data, gas, value)
}

// Take another's contract code and execute within our own context
func (env *Environment) CallCode(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas.Uint64())

		return nil, nil
	}

	return env.CallContext.CallCode(env, caller, addr, data, gas, value)
}

// Same as CallCode except sender and value is propagated from parent to child scope
func (env *Environment) DelegateCall(caller ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas.Uint64())

		return nil, nil
	}

	return env.CallContext.DelegateCall(env, caller, addr, data, gas)
}

// Create a new contract
func (env *Environment) Create(caller ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas.Uint64())

		return nil, common.Address{}, nil
	}

	return env.CallContext.Create(env, caller, data, gas, value)
}
func (env *Environment) RuleSet() RuleSet { return env.ruleSet }
func (env *Environment) Vm() Vm           { return env.evm }
func (env *Environment) Db() Database     { return env.Backend.Get() }
func (env *Environment) GetHash(n uint64) common.Hash {
	return env.Backend.GetHash(n)
}
