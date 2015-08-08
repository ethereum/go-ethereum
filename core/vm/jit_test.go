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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

const maxRun = 1000

type vmBench struct {
	precompile bool // compile prior to executing
	nojit      bool // ignore jit (sets DisbaleJit = true
	forcejit   bool // forces the jit, precompile is ignored

	code  []byte
	input []byte
}

func runVmBench(test vmBench, b *testing.B) {
	db, _ := ethdb.NewMemDatabase()
	sender := state.NewStateObject(common.Address{}, db)

	if test.precompile && !test.forcejit {
		NewProgram(test.code)
	}
	env := NewEnv()

	DisableJit = test.nojit
	ForceJit = test.forcejit

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		context := NewContext(sender, sender, big.NewInt(100), big.NewInt(10000), big.NewInt(0))
		context.Code = test.code
		context.CodeAddr = &common.Address{}
		_, err := New(env).Run(context, test.input)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}

var benchmarks = map[string]vmBench{
	"pushes": vmBench{
		false, false, false,
		common.Hex2Bytes("600a600a01600a600a01600a600a01600a600a01600a600a01600a600a01600a600a01600a600a01600a600a01600a600a01"), nil,
	},
}

func BenchmarkPushes(b *testing.B) {
	runVmBench(benchmarks["pushes"], b)
}

type Env struct {
	gasLimit *big.Int
	depth    int
}

func NewEnv() *Env {
	return &Env{big.NewInt(10000), 0}
}

func (self *Env) Origin() common.Address { return common.Address{} }
func (self *Env) BlockNumber() *big.Int  { return big.NewInt(0) }
func (self *Env) AddStructLog(log StructLog) {
}
func (self *Env) StructLogs() []StructLog {
	return nil
}

//func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() common.Address { return common.Address{} }
func (self *Env) Time() uint64             { return uint64(time.Now().Unix()) }
func (self *Env) Difficulty() *big.Int     { return big.NewInt(0) }
func (self *Env) State() *state.StateDB    { return nil }
func (self *Env) GasLimit() *big.Int       { return self.gasLimit }
func (self *Env) VmType() Type             { return StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Sha3([]byte(big.NewInt(int64(n)).String())))
}
func (self *Env) AddLog(log *state.Log) {
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) CanTransfer(from Account, balance *big.Int) bool {
	return from.Balance().Cmp(balance) >= 0
}
func (self *Env) Transfer(from, to Account, amount *big.Int) error {
	return nil
}
func (self *Env) Call(caller ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) CallCode(caller ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) Create(caller ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, ContextRef) {
	return nil, nil, nil
}
