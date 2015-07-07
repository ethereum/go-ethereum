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

package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ContextRef interface {
	ReturnGas(*big.Int, *big.Int)
	Address() common.Address
	SetCode([]byte)
}

type Context struct {
	caller ContextRef
	self   ContextRef

	jumpdests destinations // result of JUMPDEST analysis.

	Code     []byte
	CodeAddr *common.Address

	value, Gas, UsedGas, Price *big.Int

	Args []byte
}

// Create a new context for the given data items.
func NewContext(caller ContextRef, object ContextRef, value, gas, price *big.Int) *Context {
	c := &Context{caller: caller, self: object, Args: nil}

	if parent, ok := caller.(*Context); ok {
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

func (c *Context) GetOp(n uint64) OpCode {
	return OpCode(c.GetByte(n))
}

func (c *Context) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

func (c *Context) Return(ret []byte) []byte {
	// Return the remaining gas to the caller
	c.caller.ReturnGas(c.Gas, c.Price)

	return ret
}

/*
 * Gas functions
 */
func (c *Context) UseGas(gas *big.Int) (ok bool) {
	ok = UseGas(c.Gas, gas)
	if ok {
		c.UsedGas.Add(c.UsedGas, gas)
	}
	return
}

// Implement the caller interface
func (c *Context) ReturnGas(gas, price *big.Int) {
	// Return the gas to the context
	c.Gas.Add(c.Gas, gas)
	c.UsedGas.Sub(c.UsedGas, gas)
}

/*
 * Set / Get
 */
func (c *Context) Address() common.Address {
	return c.self.Address()
}

func (self *Context) SetCode(code []byte) {
	self.Code = code
}

func (self *Context) SetCallCode(addr *common.Address, code []byte) {
	self.Code = code
	self.CodeAddr = addr
}
