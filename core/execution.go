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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// Execution is the execution environment for the given call or create action.
type Execution struct {
	env     vm.Environment
	address *common.Address
	input   []byte
	evm     vm.VirtualMachine

	Gas, price, value *big.Int
}

// NewExecution returns a new execution environment that handles all calling
// and creation logic defined by the YP.
func NewExecution(env vm.Environment, address *common.Address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	exe := &Execution{env: env, address: address, input: input, Gas: gas, price: gasPrice, value: value}
	exe.evm = vm.NewVm(env)
	return exe
}

// Call executes within the given context
func (self *Execution) Call(codeAddr common.Address, caller vm.ContextRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.env.State().GetCode(codeAddr)

	return self.exec(&codeAddr, code, caller)
}

// Create creates a new contract and runs the initialisation procedure of the
// contract. This returns the returned code for the contract and is stored
// elsewhere.
func (self *Execution) Create(caller vm.ContextRef) (ret []byte, err error, account *state.StateObject) {
	// Input must be nil for create
	code := self.input
	self.input = nil
	ret, err = self.exec(nil, code, caller)
	// Here we get an error if we run into maximum stack depth,
	// See: https://github.com/ethereum/yellowpaper/pull/131
	// and YP definitions for CREATE instruction
	if err != nil {
		return nil, err, nil
	}
	account = self.env.State().GetStateObject(*self.address)
	return
}

// exec executes the given code and executes within the contextAddr context.
func (self *Execution) exec(contextAddr *common.Address, code []byte, caller vm.ContextRef) (ret []byte, err error) {
	env := self.env
	evm := self.evm
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(self.Gas, self.price)

		return nil, vm.DepthError
	}

	if !env.CanTransfer(env.State().GetStateObject(caller.Address()), self.value) {
		caller.ReturnGas(self.Gas, self.price)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", self.value, env.State().GetBalance(caller.Address()))
	}

	var createAccount bool
	if self.address == nil {
		// Generate a new address
		nonce := env.State().GetNonce(caller.Address())
		env.State().SetNonce(caller.Address(), nonce+1)

		addr := crypto.CreateAddress(caller.Address(), nonce)

		self.address = &addr
		createAccount = true
	}
	snapshot := env.State().Copy()

	var (
		from = env.State().GetStateObject(caller.Address())
		to   *state.StateObject
	)
	if createAccount {
		to = env.State().CreateAccount(*self.address)
	} else {
		to = env.State().GetOrNewStateObject(*self.address)
	}
	vm.Transfer(from, to, self.value)

	context := vm.NewContext(caller, to, self.value, self.Gas, self.price)
	context.SetCallCode(contextAddr, code)

	ret, err = evm.Run(context, self.input)
	if err != nil {
		env.State().Set(snapshot)
	}

	return
}
