// Copyright 2016 The go-ethereum Authors
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

package ethapi

import (
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type ruleSet struct{}

func (self *ruleSet) IsHomestead(*big.Int) bool    { return true }
func (*ruleSet) GasTable(*big.Int) params.GasTable { return params.GasTableHomesteadGasRepriceFork }

type Env struct {
	gasLimit *big.Int
	depth    int
	evm      *vm.EVM
}

func NewEnv(config *vm.Config) *Env {
	env := &Env{gasLimit: big.NewInt(10000), depth: 0}
	env.evm = vm.New(env, *config)
	return env
}

func (self *Env) RuleSet() vm.RuleSet    { return &ruleSet{} }
func (self *Env) Vm() vm.Vm              { return self.evm }
func (self *Env) Origin() common.Address { return common.Address{} }
func (self *Env) BlockNumber() *big.Int  { return big.NewInt(0) }

//func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() common.Address { return common.Address{} }
func (self *Env) SnapshotDatabase() int    { return 0 }
func (self *Env) RevertToSnapshot(int)     {}
func (self *Env) Time() *big.Int           { return big.NewInt(time.Now().Unix()) }
func (self *Env) Difficulty() *big.Int     { return big.NewInt(0) }
func (self *Env) Db() vm.Database          { return nil }
func (self *Env) GasLimit() *big.Int       { return self.gasLimit }
func (self *Env) VmType() vm.Type          { return vm.StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
}
func (self *Env) AddLog(log *vm.Log) {
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) CanTransfer(from common.Address, balance *big.Int) bool {
	return true
}
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) {}
func (self *Env) Call(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) CallCode(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (self *Env) Create(caller vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return nil, common.Address{}, nil
}
func (self *Env) DelegateCall(me vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return nil, nil
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
func (account) SetCode(common.Hash, []byte)                         {}
func (account) ForEachStorage(cb func(key, value common.Hash) bool) {}

func runTrace(tracer *JavascriptTracer) (interface{}, error) {
	env := NewEnv(&vm.Config{Debug: true, Tracer: tracer})

	contract := vm.NewContract(account{}, account{}, big.NewInt(0), env.GasLimit(), big.NewInt(1))
	contract.Code = []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, 0x0}

	_, err := env.Vm().Run(contract, []byte{})
	if err != nil {
		return nil, err
	}

	return tracer.GetResult()
}

func TestTracing(t *testing.T) {
	tracer, err := NewJavascriptTracer("{count: 0, step: function() { this.count += 1; }, result: function() { return this.count; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}

	value, ok := ret.(float64)
	if !ok {
		t.Errorf("Expected return value to be float64, was %T", ret)
	}
	if value != 3 {
		t.Errorf("Expected return value to be 3, got %v", value)
	}
}

func TestStack(t *testing.T) {
	tracer, err := NewJavascriptTracer("{depths: [], step: function(log) { this.depths.push(log.stack.length()); }, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}

	expected := []int{0, 1, 2}
	if !reflect.DeepEqual(ret, expected) {
		t.Errorf("Expected return value to be %#v, got %#v", expected, ret)
	}
}

func TestOpcodes(t *testing.T) {
	tracer, err := NewJavascriptTracer("{opcodes: [], step: function(log) { this.opcodes.push(log.op.toString()); }, result: function() { return this.opcodes; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"PUSH1", "PUSH1", "STOP"}
	if !reflect.DeepEqual(ret, expected) {
		t.Errorf("Expected return value to be %#v, got %#v", expected, ret)
	}
}

func TestHalt(t *testing.T) {
	timeout := errors.New("stahp")
	tracer, err := NewJavascriptTracer("{step: function() { while(1); }, result: function() { return null; }}")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		tracer.Stop(timeout)
	}()

	if _, err = runTrace(tracer); err.Error() != "stahp    in server-side tracer function 'step'" {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestHaltBetweenSteps(t *testing.T) {
	tracer, err := NewJavascriptTracer("{step: function() {}, result: function() { return null; }}")
	if err != nil {
		t.Fatal(err)
	}

	env := NewEnv(&vm.Config{Debug: true, Tracer: tracer})
	contract := vm.NewContract(&account{}, &account{}, big.NewInt(0), big.NewInt(0), big.NewInt(0))

	tracer.CaptureState(env, 0, 0, big.NewInt(0), big.NewInt(0), nil, nil, contract, 0, nil)
	timeout := errors.New("stahp")
	tracer.Stop(timeout)
	tracer.CaptureState(env, 0, 0, big.NewInt(0), big.NewInt(0), nil, nil, contract, 0, nil)

	if _, err := tracer.GetResult(); err.Error() != "stahp    in server-side tracer function 'step'" {
		t.Errorf("Expected timeout error, got %v", err)
	}
}
