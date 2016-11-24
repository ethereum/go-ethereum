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
	"github.com/ethereum/go-ethereum/params"
)

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

// Environment provides information about external sources for the EVM.
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
func NewEnvironment(context Context, statedb StateDB, chainConfig *params.ChainConfig, vmCfg Config) *Environment {
	env := &Environment{
		Context:     context,
		StateDB:     statedb,
		vmConfig:    vmCfg,
		chainConfig: chainConfig,
	}
	env.evm = New(env, vmCfg)
	return env
}

func (env *Environment) Call(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	return env.CallContext.Call(env, caller, addr, data, gas, value)
}

// Take another's contract code and execute within our own context
func (env *Environment) CallCode(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	return env.CallContext.CallCode(env, caller, addr, data, gas, value)
}

// Same as CallCode except sender and value is propagated from parent to child scope
func (env *Environment) DelegateCall(caller ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, nil
	}

	return env.CallContext.DelegateCall(env, caller, addr, data, gas)
}

// Create a new contract
func (env *Environment) Create(caller ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error) {
	if env.vmConfig.Test && env.Depth > 0 {
		caller.ReturnGas(gas)

		return nil, common.Address{}, nil
	}

	return env.CallContext.Create(env, caller, data, gas, value)
}

// ChainConfig returns the environment's chain configuration
func (env *Environment) ChainConfig() *params.ChainConfig { return env.chainConfig }

// EVM returns the environments EVM
func (env *Environment) EVM() Vm { return env.evm }
