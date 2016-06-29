// Copyright 2015 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/crypto"
)

const maxRun = 1000

func TestSegmenting(t *testing.T) {
	prog := NewProgram([]byte{byte(PUSH1), 0x1, byte(PUSH1), 0x1, 0x0})
	err := CompileProgram(prog)
	if err != nil {
		t.Fatal(err)
	}

	if instr, ok := prog.instructions[0].(pushSeg); ok {
		if len(instr.data) != 2 {
			t.Error("expected 2 element width pushSegment, got", len(instr.data))
		}
	} else {
		t.Errorf("expected instr[0] to be a pushSeg, got %T", prog.instructions[0])
	}

	prog = NewProgram([]byte{byte(PUSH1), 0x1, byte(PUSH1), 0x1, byte(JUMP)})
	err = CompileProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := prog.instructions[1].(jumpSeg); ok {
	} else {
		t.Errorf("expected instr[1] to be jumpSeg, got %T", prog.instructions[1])
	}

	prog = NewProgram([]byte{byte(PUSH1), 0x1, byte(PUSH1), 0x1, byte(PUSH1), 0x1, byte(JUMP)})
	err = CompileProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	if instr, ok := prog.instructions[0].(pushSeg); ok {
		if len(instr.data) != 2 {
			t.Error("expected 2 element width pushSegment, got", len(instr.data))
		}
	} else {
		t.Errorf("expected instr[0] to be a pushSeg, got %T", prog.instructions[0])
	}
	if _, ok := prog.instructions[2].(jumpSeg); ok {
	} else {
		t.Errorf("expected instr[1] to be jumpSeg, got %T", prog.instructions[1])
	}
}

func TestCompiling(t *testing.T) {
	prog := NewProgram([]byte{0x60, 0x10})
	err := CompileProgram(prog)
	if err != nil {
		t.Error("didn't expect compile error")
	}

	if len(prog.instructions) != 1 {
		t.Error("expected 1 compiled instruction, got", len(prog.instructions))
	}
}

func TestResetInput(t *testing.T) {
	var sender account

	env := NewEnv(false, true)
	contract := NewContract(sender, sender, big.NewInt(100), big.NewInt(10000), big.NewInt(0))
	contract.CodeAddr = &common.Address{}

	program := NewProgram([]byte{})
	RunProgram(program, env, contract, []byte{0xbe, 0xef})
	if contract.Input != nil {
		t.Errorf("expected input to be nil, got %x", contract.Input)
	}
}

func TestPcMappingToInstruction(t *testing.T) {
	program := NewProgram([]byte{byte(PUSH2), 0xbe, 0xef, byte(ADD)})
	CompileProgram(program)
	if program.mapping[3] != 1 {
		t.Error("expected mapping PC 4 to me instr no. 2, got", program.mapping[4])
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

type vmBench struct {
	precompile bool // compile prior to executing
	nojit      bool // ignore jit (sets DisbaleJit = true
	forcejit   bool // forces the jit, precompile is ignored

	code  []byte
	input []byte
}

type account struct{}

func (account) SubBalance(amount *big.Int)                          {}
func (account) AddBalance(amount *big.Int)                          {}
func (account) SetAddress(common.Address)                           {}
func (account) Value() *big.Int                                     { return nil }
func (account) SetBalance(*big.Int)                                 {}
func (account) SetNonce(uint64)                                     {}
func (account) Balance() *big.Int                                   { return nil }
func (account) Address() common.Address                             { return common.Address{} }
func (account) ReturnGas(*big.Int, *big.Int)                        {}
func (account) SetCode([]byte)                                      {}
func (account) ForEachStorage(cb func(key, value common.Hash) bool) {}

func runVmBench(test vmBench, b *testing.B) {
	var sender account

	if test.precompile && !test.forcejit {
		NewProgram(test.code)
	}
	env := NewEnv(test.nojit, test.forcejit)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		context := NewContract(sender, sender, big.NewInt(100), big.NewInt(10000), big.NewInt(0))
		context.Code = test.code
		context.CodeAddr = &common.Address{}
		_, err := env.Vm().Run(context, test.input)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}

type Env struct {
	gasLimit *big.Int
	depth    int
	evm      *EVM
}

func NewEnv(noJit, forceJit bool) *Env {
	env := &Env{gasLimit: big.NewInt(10000), depth: 0}
	env.evm = New(env, Config{
		EnableJit: !noJit,
		ForceJit:  forceJit,
	})
	return env
}

func (self *Env) RuleSet() RuleSet       { return ruleSet{new(big.Int)} }
func (self *Env) Vm() Vm                 { return self.evm }
func (self *Env) Origin() common.Address { return common.Address{} }
func (self *Env) BlockNumber() *big.Int  { return big.NewInt(0) }
func (self *Env) AddStructLog(log StructLog) {
}
func (self *Env) StructLogs() []StructLog {
	return nil
}

//func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() common.Address { return common.Address{} }
func (self *Env) MakeSnapshot() Database   { return nil }
func (self *Env) SetSnapshot(Database)     {}
func (self *Env) Time() *big.Int           { return big.NewInt(time.Now().Unix()) }
func (self *Env) Difficulty() *big.Int     { return big.NewInt(0) }
func (self *Env) Db() Database             { return nil }
func (self *Env) GasLimit() *big.Int       { return self.gasLimit }
func (self *Env) VmType() Type             { return StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
}
func (self *Env) AddLog(log *Log) {
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) CanTransfer(from common.Address, balance *big.Int) bool {
	return true
}
func (self *Env) Transfer(from, to Account, amount *big.Int) {}
func (self *Env) Call(caller ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) CallCode(caller ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) Create(caller ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return nil, common.Address{}, nil
}
func (self *Env) DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return nil, nil
}
