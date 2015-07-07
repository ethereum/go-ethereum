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
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// Environment is is required by the virtual machine to get information from
// it's own isolated environment. For an example see `core.VMEnv`
type Environment interface {
	State() *state.StateDB

	Origin() common.Address
	BlockNumber() *big.Int
	GetHash(n uint64) common.Hash
	Coinbase() common.Address
	Time() uint64
	Difficulty() *big.Int
	GasLimit() *big.Int
	Transfer(from, to Account, amount *big.Int) error
	AddLog(*state.Log)
	AddStructLog(StructLog)
	StructLogs() []StructLog

	VmType() Type

	Depth() int
	SetDepth(i int)

	Call(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	CallCode(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, ContextRef)
}

// StructLog is emited to the Environment each cycle and lists information about the curent internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc      uint64
	Op      OpCode
	Gas     *big.Int
	GasCost *big.Int
	Memory  []byte
	Stack   []*big.Int
	Storage map[common.Hash][]byte
	Err     error
}

type Account interface {
	SubBalance(amount *big.Int)
	AddBalance(amount *big.Int)
	Balance() *big.Int
	Address() common.Address
}

// generic transfer method
func Transfer(from, to Account, amount *big.Int) error {
	if from.Balance().Cmp(amount) < 0 {
		return errors.New("Insufficient balance in account")
	}

	from.SubBalance(amount)
	to.AddBalance(amount)

	return nil
}
