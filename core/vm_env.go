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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type VMEnv struct {
	state  *state.StateDB
	header *types.Header
	msg    Message
	depth  int
	chain  *ChainManager
	typ    vm.Type
	// structured logging
	logs []vm.StructLog
}

func NewEnv(state *state.StateDB, chain *ChainManager, msg Message, header *types.Header) *VMEnv {
	return &VMEnv{
		chain:  chain,
		state:  state,
		header: header,
		msg:    msg,
		typ:    vm.StdVmTy,
	}
}

func (self *VMEnv) Origin() common.Address   { f, _ := self.msg.From(); return f }
func (self *VMEnv) BlockNumber() *big.Int    { return self.header.Number }
func (self *VMEnv) Coinbase() common.Address { return self.header.Coinbase }
func (self *VMEnv) Time() uint64             { return self.header.Time }
func (self *VMEnv) Difficulty() *big.Int     { return self.header.Difficulty }
func (self *VMEnv) GasLimit() *big.Int       { return self.header.GasLimit }
func (self *VMEnv) Value() *big.Int          { return self.msg.Value() }
func (self *VMEnv) State() *state.StateDB    { return self.state }
func (self *VMEnv) Depth() int               { return self.depth }
func (self *VMEnv) SetDepth(i int)           { self.depth = i }
func (self *VMEnv) VmType() vm.Type          { return self.typ }
func (self *VMEnv) SetVmType(t vm.Type)      { self.typ = t }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	if block := self.chain.GetBlockByNumber(n); block != nil {
		return block.Hash()
	}

	return common.Hash{}
}

func (self *VMEnv) AddLog(log *state.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *VMEnv) Call(me vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := NewExecution(self, &addr, data, gas, price, value)
	return exe.Call(addr, me)
}
func (self *VMEnv) CallCode(me vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	maddr := me.Address()
	exe := NewExecution(self, &maddr, data, gas, price, value)
	return exe.Call(addr, me)
}

func (self *VMEnv) Create(me vm.ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := NewExecution(self, nil, data, gas, price, value)
	return exe.Create(me)
}

func (self *VMEnv) StructLogs() []vm.StructLog {
	return self.logs
}

func (self *VMEnv) AddStructLog(log vm.StructLog) {
	self.logs = append(self.logs, log)
}
