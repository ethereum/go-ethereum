// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type Execution struct {
	env     vm.Environment
	address *common.Address
	input   []byte
	evm     vm.VirtualMachine

	Gas, price, value *big.Int
}

func NewExecution(env vm.Environment, address *common.Address, input []byte, gas, gasPrice, value *big.Int) *Execution {
	exe := &Execution{env: env, address: address, input: input, Gas: gas, price: gasPrice, value: value}
	exe.evm = vm.NewVm(env)
	return exe
}

func (self *Execution) Call(codeAddr common.Address, caller vm.ContextRef) ([]byte, error) {
	// Retrieve the executing code
	code := self.env.State().GetCode(codeAddr)

	return self.exec(&codeAddr, code, caller)
}

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

func (self *Execution) exec(contextAddr *common.Address, code []byte, caller vm.ContextRef) (ret []byte, err error) {
	env := self.env
	evm := self.evm
	if env.Depth() > int(params.CallCreateDepth.Int64()) {
		caller.ReturnGas(self.Gas, self.price)

		return nil, vm.DepthError
	}

	vsnapshot := env.State().Copy()
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

	err = env.Transfer(from, to, self.value)
	if err != nil {
		env.State().Set(vsnapshot)

		caller.ReturnGas(self.Gas, self.price)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", self.value, from.Balance())
	}

	context := vm.NewContext(caller, to, self.value, self.Gas, self.price)
	context.SetCallCode(contextAddr, code)

	ret, err = evm.Run(context, self.input)
	if err != nil {
		env.State().Set(snapshot)
	}

	return
}
