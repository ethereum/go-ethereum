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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/robertkrimen/otto"
)

// fakeBig is used to provide an interface to Javascript for 'big.NewInt'
type fakeBig struct{}

// NewInt creates a new big.Int with the specified int64 value.
func (fb *fakeBig) NewInt(x int64) *big.Int {
	return big.NewInt(x)
}

// OpCodeWrapper provides a JavaScript-friendly wrapper around OpCode, to convince Otto to treat it
// as an object, instead of a number.
type opCodeWrapper struct {
	op vm.OpCode
}

// toNumber returns the ID of this opcode as an integer
func (ocw *opCodeWrapper) toNumber() int {
	return int(ocw.op)
}

// toString returns the string representation of the opcode
func (ocw *opCodeWrapper) toString() string {
	return ocw.op.String()
}

// isPush returns true if the op is a Push
func (ocw *opCodeWrapper) isPush() bool {
	return ocw.op.IsPush()
}

// MarshalJSON serializes the opcode as JSON
func (ocw *opCodeWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(ocw.op.String())
}

// toValue returns an otto.Value for the opCodeWrapper
func (ocw *opCodeWrapper) toValue(vm *otto.Otto) otto.Value {
	value, _ := vm.ToValue(ocw)
	obj := value.Object()
	obj.Set("toNumber", ocw.toNumber)
	obj.Set("toString", ocw.toString)
	obj.Set("isPush", ocw.isPush)
	return value
}

// memoryWrapper provides a JS wrapper around vm.Memory
type memoryWrapper struct {
	memory *vm.Memory
}

// slice returns the requested range of memory as a byte slice
func (mw *memoryWrapper) slice(begin, end int64) []byte {
	return mw.memory.Get(begin, end-begin)
}

// getUint returns the 32 bytes at the specified address interpreted
// as an unsigned integer
func (mw *memoryWrapper) getUint(addr int64) *big.Int {
	ret := big.NewInt(0)
	ret.SetBytes(mw.memory.GetPtr(addr, 32))
	return ret
}

// toValue returns an otto.Value for the memoryWrapper
func (mw *memoryWrapper) toValue(vm *otto.Otto) otto.Value {
	value, _ := vm.ToValue(mw)
	obj := value.Object()
	obj.Set("slice", mw.slice)
	obj.Set("getUint", mw.getUint)
	return value
}

// stackWrapper provides a JS wrapper around vm.Stack
type stackWrapper struct {
	stack *vm.Stack
}

// peek returns the nth-from-the-top element of the stack.
func (sw *stackWrapper) peek(idx int) *big.Int {
	return sw.stack.Data()[len(sw.stack.Data())-idx-1]
}

// length returns the length of the stack
func (sw *stackWrapper) length() int {
	return len(sw.stack.Data())
}

// toValue returns an otto.Value for the stackWrapper
func (sw *stackWrapper) toValue(vm *otto.Otto) otto.Value {
	value, _ := vm.ToValue(sw)
	obj := value.Object()
	obj.Set("peek", sw.peek)
	obj.Set("length", sw.length)
	return value
}

// dbWrapper provides a JS wrapper around vm.Database
type dbWrapper struct {
	db vm.StateDB
}

// getBalance retrieves an account's balance
func (dw *dbWrapper) getBalance(addr common.Address) *big.Int {
	return dw.db.GetBalance(addr)
}

// getNonce retrieves an account's nonce
func (dw *dbWrapper) getNonce(addr common.Address) uint64 {
	return dw.db.GetNonce(addr)
}

// getCode retrieves an account's code
func (dw *dbWrapper) getCode(addr common.Address) []byte {
	return dw.db.GetCode(addr)
}

// getState retrieves an account's state data for the given hash
func (dw *dbWrapper) getState(addr common.Address, hash common.Hash) common.Hash {
	return dw.db.GetState(addr, hash)
}

// exists returns true iff the account exists
func (dw *dbWrapper) exists(addr common.Address) bool {
	return dw.db.Exist(addr)
}

// toValue returns an otto.Value for the dbWrapper
func (dw *dbWrapper) toValue(vm *otto.Otto) otto.Value {
	value, _ := vm.ToValue(dw)
	obj := value.Object()
	obj.Set("getBalance", dw.getBalance)
	obj.Set("getNonce", dw.getNonce)
	obj.Set("getCode", dw.getCode)
	obj.Set("getState", dw.getState)
	obj.Set("exists", dw.exists)
	return value
}

// contractWrapper provides a JS wrapper around vm.Contract
type contractWrapper struct {
	contract *vm.Contract
}

func (c *contractWrapper) caller() common.Address {
	return c.contract.Caller()
}

func (c *contractWrapper) address() common.Address {
	return c.contract.Address()
}

func (c *contractWrapper) value() *big.Int {
	return c.contract.Value()
}

func (c *contractWrapper) calldata() []byte {
	return c.contract.Input
}

func (c *contractWrapper) toValue(vm *otto.Otto) otto.Value {
	value, _ := vm.ToValue(c)
	obj := value.Object()
	obj.Set("caller", c.caller)
	obj.Set("address", c.address)
	obj.Set("value", c.value)
	obj.Set("calldata", c.calldata)
	return value
}

// JavascriptTracer provides an implementation of Tracer that evaluates a
// Javascript function for each VM execution step.
type JavascriptTracer struct {
	vm            *otto.Otto             // Javascript VM instance
	traceobj      *otto.Object           // User-supplied object to call
	log           map[string]interface{} // (Reusable) map for the `log` arg to `step`
	logvalue      otto.Value             // JS view of `log`
	memory        *memoryWrapper         // Wrapper around the VM memory
	memvalue      otto.Value             // JS view of `memory`
	stack         *stackWrapper          // Wrapper around the VM stack
	stackvalue    otto.Value             // JS view of `stack`
	db            *dbWrapper             // Wrapper around the VM environment
	dbvalue       otto.Value             // JS view of `db`
	contract      *contractWrapper       // Wrapper around the contract object
	contractvalue otto.Value             // JS view of `contract`
	err           error                  // Error, if one has occurred
}

// NewJavascriptTracer instantiates a new JavascriptTracer instance.
// code specifies a Javascript snippet, which must evaluate to an expression
// returning an object with 'step' and 'result' functions.
func NewJavascriptTracer(code string) (*JavascriptTracer, error) {
	vm := otto.New()
	vm.Interrupt = make(chan func(), 1)

	// Set up builtins for this environment
	vm.Set("big", &fakeBig{})
	vm.Set("toHex", hexutil.Encode)

	jstracer, err := vm.Object("(" + code + ")")
	if err != nil {
		return nil, err
	}

	// Check the required functions exist
	step, err := jstracer.Get("step")
	if err != nil {
		return nil, err
	}
	if !step.IsFunction() {
		return nil, fmt.Errorf("Trace object must expose a function step()")
	}

	result, err := jstracer.Get("result")
	if err != nil {
		return nil, err
	}
	if !result.IsFunction() {
		return nil, fmt.Errorf("Trace object must expose a function result()")
	}

	// Create the persistent log object
	log := make(map[string]interface{})
	logvalue, _ := vm.ToValue(log)

	// Create persistent wrappers for memory and stack
	mem := &memoryWrapper{}
	stack := &stackWrapper{}
	db := &dbWrapper{}
	contract := &contractWrapper{}

	return &JavascriptTracer{
		vm:            vm,
		traceobj:      jstracer,
		log:           log,
		logvalue:      logvalue,
		memory:        mem,
		memvalue:      mem.toValue(vm),
		stack:         stack,
		stackvalue:    stack.toValue(vm),
		db:            db,
		dbvalue:       db.toValue(vm),
		contract:      contract,
		contractvalue: contract.toValue(vm),
		err:           nil,
	}, nil
}

// Stop terminates execution of any JavaScript
func (jst *JavascriptTracer) Stop(err error) {
	jst.vm.Interrupt <- func() {
		panic(err)
	}
}

// callSafely executes a method on a JS object, catching any panics and
// returning them as error objects.
func (jst *JavascriptTracer) callSafely(method string, argumentList ...interface{}) (ret interface{}, err error) {
	defer func() {
		if caught := recover(); caught != nil {
			switch caught := caught.(type) {
			case error:
				err = caught
			case string:
				err = errors.New(caught)
			case fmt.Stringer:
				err = errors.New(caught.String())
			default:
				panic(caught)
			}
		}
	}()

	value, err := jst.traceobj.Call(method, argumentList...)
	ret, _ = value.Export()
	return ret, err
}

func wrapError(context string, err error) error {
	var message string
	switch err := err.(type) {
	case *otto.Error:
		message = err.String()
	default:
		message = err.Error()
	}
	return fmt.Errorf("%v    in server-side tracer function '%v'", message, context)
}

// CaptureState implements the Tracer interface to trace a single step of VM execution
func (jst *JavascriptTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	if jst.err == nil {
		jst.memory.memory = memory
		jst.stack.stack = stack
		jst.db.db = env.StateDB
		jst.contract.contract = contract

		ocw := &opCodeWrapper{op}

		jst.log["pc"] = pc
		jst.log["op"] = ocw.toValue(jst.vm)
		jst.log["gas"] = gas
		jst.log["gasPrice"] = cost
		jst.log["memory"] = jst.memvalue
		jst.log["stack"] = jst.stackvalue
		jst.log["contract"] = jst.contractvalue
		jst.log["depth"] = depth
		jst.log["account"] = contract.Address()
		jst.log["err"] = err

		_, err := jst.callSafely("step", jst.logvalue, jst.dbvalue)
		if err != nil {
			jst.err = wrapError("step", err)
		}
	}
	return nil
}

// CaptureEnd is called after the call finishes
func (jst *JavascriptTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	//TODO! @Arachnid please figure out of there's anything we can use this method for
	return nil
}

// GetResult calls the Javascript 'result' function and returns its value, or any accumulated error
func (jst *JavascriptTracer) GetResult() (result interface{}, err error) {
	if jst.err != nil {
		return nil, jst.err
	}

	result, err = jst.callSafely("result")
	if err != nil {
		err = wrapError("result", err)
	}
	return
}
