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

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	ReturnGas(*big.Int, *big.Int)
	Address() common.Address
	Value() *big.Int
	SetCode([]byte)
	EachStorage(cb func(key, value []byte))
}

// Contract represents an ethereum contract in the state database. It contains
// the the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// CallerAddress is the result of the caller which initialised this
	// contract. However when the "call method" is delegated this value
	// needs to be initialised to that of the caller's caller.
	CallerAddress common.Address
	caller        ContractRef
	self          ContractRef

	jumpdests destinations // result of JUMPDEST analysis.

	Code     []byte
	Input    []byte
	CodeAddr *common.Address

	value, Gas, UsedGas, Price *big.Int

	Args []byte

	DelegateCall bool
}

// NewContract returns a new contract environment for the execution of EVM.
func NewContract(caller ContractRef, object ContractRef, value, gas, price *big.Int) *Contract {
	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object, Args: nil}

	if parent, ok := caller.(*Contract); ok {
		// Reuse JUMPDEST analysis from parent context if available.
		c.jumpdests = parent.jumpdests
	} else {
		c.jumpdests = make(destinations)
	}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	c.Gas = gas //new(big.Int).Set(gas)
	c.value = new(big.Int).Set(value)
	// In most cases price and value are pointers to transaction objects
	// and we don't want the transaction's values to change.
	c.Price = new(big.Int).Set(price)
	c.UsedGas = new(big.Int)

	return c
}

// AsDelegate sets the contract to be a delegate call and returns the current
// contract (for chaining calls)
func (c *Contract) AsDelegate() *Contract {
	c.DelegateCall = true
	// NOTE: caller must, at all times be a contract. It should never happen
	// that caller is something other than a Contract.
	c.CallerAddress = c.caller.(*Contract).CallerAddress
	return c
}

// GetOp returns the n'th element in the contract's byte array
func (c *Contract) GetOp(n uint64) OpCode {
	return OpCode(c.GetByte(n))
}

// GetByte returns the n'th byte in the contract's byte array
func (c *Contract) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

// Caller returns the caller of the contract.
//
// Caller will recursively call caller when the contract is a delegate
// call, including that of caller's caller.
func (c *Contract) Caller() common.Address {
	return c.CallerAddress
}

// Finalise finalises the contract and returning any remaining gas to the original
// caller.
func (c *Contract) Finalise() {
	// Return the remaining gas to the caller
	c.caller.ReturnGas(c.Gas, c.Price)
}

// UseGas attempts the use gas and subtracts it and returns true on success
func (c *Contract) UseGas(gas *big.Int) (ok bool) {
	ok = useGas(c.Gas, gas)
	if ok {
		c.UsedGas.Add(c.UsedGas, gas)
	}
	return
}

// ReturnGas adds the given gas back to itself.
func (c *Contract) ReturnGas(gas, price *big.Int) {
	// Return the gas to the context
	c.Gas.Add(c.Gas, gas)
	c.UsedGas.Sub(c.UsedGas, gas)
}

// Address returns the contracts address
func (c *Contract) Address() common.Address {
	return c.self.Address()
}

// Value returns the contracts value (sent to it from it's caller)
func (c *Contract) Value() *big.Int {
	return c.value
}

// SetCode sets the code to the contract
func (self *Contract) SetCode(code []byte) {
	self.Code = code
}

// SetCallCode sets the code of the contract and address of the backing data
// object
func (self *Contract) SetCallCode(addr *common.Address, code []byte) {
	self.Code = code
	self.CodeAddr = addr
}

// EachStorage iterates the contract's storage and calls a method for every key
// value pair.
func (self *Contract) EachStorage(cb func(key, value []byte)) {
	self.caller.EachStorage(cb)
}
