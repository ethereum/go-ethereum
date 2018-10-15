// Copyright 2017 The go-ethereum Authors
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

package tracers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type account struct{}

func (account) SubBalance(amount *big.Int)                          {}
func (account) AddBalance(amount *big.Int)                          {}
func (account) SetAddress(common.Address)                           {}
func (account) Value() *big.Int                                     { return nil }
func (account) SetBalance(*big.Int)                                 {}
func (account) SetNonce(uint64)                                     {}
func (account) Balance() *big.Int                                   { return nil }
func (account) Address() common.Address                             { return common.Address{} }
func (account) ReturnGas(*big.Int)                                  {}
func (account) SetCode(common.Hash, []byte)                         {}
func (account) ForEachStorage(cb func(key, value common.Hash) bool) {}

type dummyStatedb struct {
}

func (dummyStatedb) CreateAccount(common.Address)                              { panic("implement me") }
func (dummyStatedb) SubBalance(common.Address, *big.Int)                       { panic("implement me") }
func (dummyStatedb) AddBalance(common.Address, *big.Int)                       { panic("implement me") }
func (dummyStatedb) GetBalance(common.Address) *big.Int                        { panic("implement me") }
func (dummyStatedb) GetNonce(common.Address) uint64                            { panic("implement me") }
func (dummyStatedb) SetNonce(common.Address, uint64)                           { panic("implement me") }
func (dummyStatedb) GetCodeHash(common.Address) common.Hash                    { panic("implement me") }
func (dummyStatedb) GetCode(common.Address) []byte                             { panic("implement me") }
func (dummyStatedb) SetCode(common.Address, []byte)                            { panic("implement me") }
func (dummyStatedb) GetCodeSize(common.Address) int                            { panic("implement me") }
func (dummyStatedb) AddRefund(uint64)                                          { panic("implement me") }
func (dummyStatedb) SubRefund(uint64)                                          { panic("implement me") }
func (dummyStatedb) GetRefund() uint64                                         { return 1337 }
func (dummyStatedb) GetCommittedState(common.Address, common.Hash) common.Hash { panic("implement me") }
func (dummyStatedb) GetState(common.Address, common.Hash) common.Hash          { panic("implement me") }
func (dummyStatedb) SetState(common.Address, common.Hash, common.Hash)         { panic("implement me") }
func (dummyStatedb) Suicide(common.Address) bool                               { panic("implement me") }
func (dummyStatedb) HasSuicided(common.Address) bool                           { panic("implement me") }
func (dummyStatedb) Exist(common.Address) bool                                 { panic("implement me") }
func (dummyStatedb) Empty(common.Address) bool                                 { panic("implement me") }
func (dummyStatedb) RevertToSnapshot(int)                                      { panic("implement me") }
func (dummyStatedb) Snapshot() int                                             { panic("implement me") }
func (dummyStatedb) AddLog(*types.Log)                                         { panic("implement me") }
func (dummyStatedb) AddPreimage(common.Hash, []byte)                           { panic("implement me") }
func (dummyStatedb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	panic("implement me")
}

func runTrace(tracer *Tracer) (json.RawMessage, error) {
	env := vm.NewEVM(vm.Context{BlockNumber: big.NewInt(1)}, dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})

	contract := vm.NewContract(account{}, account{}, big.NewInt(0), 10000)
	contract.Code = []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, 0x0}

	_, err := env.Interpreter().Run(contract, []byte{}, false)
	if err != nil {
		return nil, err
	}
	return tracer.GetResult()
}

func TestTracing(t *testing.T) {
	tracer, err := New("{count: 0, step: function() { this.count += 1; }, fault: function() {}, result: function() { return this.count; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("3")) {
		t.Errorf("Expected return value to be 3, got %s", string(ret))
	}
}

func TestStack(t *testing.T) {
	tracer, err := New("{depths: [], step: function(log) { this.depths.push(log.stack.length()); }, fault: function() {}, result: function() { return this.depths; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("[0,1,2]")) {
		t.Errorf("Expected return value to be [0,1,2], got %s", string(ret))
	}
}

func TestOpcodes(t *testing.T) {
	tracer, err := New("{opcodes: [], step: function(log) { this.opcodes.push(log.op.toString()); }, fault: function() {}, result: function() { return this.opcodes; }}")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := runTrace(tracer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ret, []byte("[\"PUSH1\",\"PUSH1\",\"STOP\"]")) {
		t.Errorf("Expected return value to be [\"PUSH1\",\"PUSH1\",\"STOP\"], got %s", string(ret))
	}
}

func TestHalt(t *testing.T) {
	t.Skip("duktape doesn't support abortion")

	timeout := errors.New("stahp")
	tracer, err := New("{step: function() { while(1); }, result: function() { return null; }}")
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
	tracer, err := New("{step: function() {}, fault: function() {}, result: function() { return null; }}")
	if err != nil {
		t.Fatal(err)
	}

	env := vm.NewEVM(vm.Context{BlockNumber: big.NewInt(1)}, dummyStatedb{}, params.TestChainConfig, vm.Config{Debug: true, Tracer: tracer})
	contract := vm.NewContract(&account{}, &account{}, big.NewInt(0), 0)

	tracer.CaptureState(env, 0, 0, 0, 0, nil, nil, contract, 0, nil)
	timeout := errors.New("stahp")
	tracer.Stop(timeout)
	tracer.CaptureState(env, 0, 0, 0, 0, nil, nil, contract, 0, nil)

	if _, err := tracer.GetResult(); err.Error() != timeout.Error() {
		t.Errorf("Expected timeout error, got %v", err)
	}
}
