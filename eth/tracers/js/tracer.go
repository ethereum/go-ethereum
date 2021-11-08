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

// package js is a collection of tracers written in javascript.
package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	tracers2 "github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/js/internal/tracers"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/olebedev/go-duktape.v3"
)

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}

var assetTracers = make(map[string]string)

// init retrieves the JavaScript transaction tracers included in go-ethereum.
func init() {
	for _, file := range tracers.AssetNames() {
		name := camel(strings.TrimSuffix(file, ".js"))
		assetTracers[name] = string(tracers.MustAsset(file))
	}
	tracers2.RegisterLookup(true, newJsTracer)
}

// makeSlice convert an unsafe memory pointer with the given type into a Go byte
// slice.
//
// Note, the returned slice uses the same memory area as the input arguments.
// If those are duktape stack items, popping them off **will** make the slice
// contents change.
func makeSlice(ptr unsafe.Pointer, size uint) []byte {
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(ptr), int(size), int(size)}

	return *(*[]byte)(unsafe.Pointer(&sl))
}

// popSlice pops a buffer off the JavaScript stack and returns it as a slice.
func popSlice(ctx *duktape.Context) []byte {
	blob := common.CopyBytes(makeSlice(ctx.GetBuffer(-1)))
	ctx.Pop()
	return blob
}

// pushBigInt create a JavaScript BigInteger in the VM.
func pushBigInt(n *big.Int, ctx *duktape.Context) {
	ctx.GetGlobalString("bigInt")
	ctx.PushString(n.String())
	ctx.Call(1)
}

// opWrapper provides a JavaScript wrapper around OpCode.
type opWrapper struct {
	op vm.OpCode
}

// pushObject assembles a JSVM object wrapping a swappable opcode and pushes it
// onto the VM stack.
func (ow *opWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushInt(int(ow.op)); return 1 })
	vm.PutPropString(obj, "toNumber")

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushString(ow.op.String()); return 1 })
	vm.PutPropString(obj, "toString")

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushBoolean(ow.op.IsPush()); return 1 })
	vm.PutPropString(obj, "isPush")
}

// memoryWrapper provides a JavaScript wrapper around vm.Memory.
type memoryWrapper struct {
	memory *vm.Memory
}

// slice returns the requested range of memory as a byte slice.
func (mw *memoryWrapper) slice(begin, end int64) []byte {
	if end == begin {
		return []byte{}
	}
	if end < begin || begin < 0 {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound memory", "offset", begin, "end", end)
		return nil
	}
	if mw.memory.Len() < int(end) {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound memory", "available", mw.memory.Len(), "offset", begin, "size", end-begin)
		return nil
	}
	return mw.memory.GetCopy(begin, end-begin)
}

// getUint returns the 32 bytes at the specified address interpreted as a uint.
func (mw *memoryWrapper) getUint(addr int64) *big.Int {
	if mw.memory.Len() < int(addr)+32 || addr < 0 {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound memory", "available", mw.memory.Len(), "offset", addr, "size", 32)
		return new(big.Int)
	}
	return new(big.Int).SetBytes(mw.memory.GetPtr(addr, 32))
}

// pushObject assembles a JSVM object wrapping a swappable memory and pushes it
// onto the VM stack.
func (mw *memoryWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Generate the `slice` method which takes two ints and returns a buffer
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		blob := mw.slice(int64(ctx.GetInt(-2)), int64(ctx.GetInt(-1)))
		ctx.Pop2()

		ptr := ctx.PushFixedBuffer(len(blob))
		copy(makeSlice(ptr, uint(len(blob))), blob)
		return 1
	})
	vm.PutPropString(obj, "slice")

	// Generate the `getUint` method which takes an int and returns a bigint
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		offset := int64(ctx.GetInt(-1))
		ctx.Pop()

		pushBigInt(mw.getUint(offset), ctx)
		return 1
	})
	vm.PutPropString(obj, "getUint")
}

// stackWrapper provides a JavaScript wrapper around vm.Stack.
type stackWrapper struct {
	stack *vm.Stack
}

// peek returns the nth-from-the-top element of the stack.
func (sw *stackWrapper) peek(idx int) *big.Int {
	if len(sw.stack.Data()) <= idx || idx < 0 {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound stack", "size", len(sw.stack.Data()), "index", idx)
		return new(big.Int)
	}
	return sw.stack.Back(idx).ToBig()
}

// pushObject assembles a JSVM object wrapping a swappable stack and pushes it
// onto the VM stack.
func (sw *stackWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushInt(len(sw.stack.Data())); return 1 })
	vm.PutPropString(obj, "length")

	// Generate the `peek` method which takes an int and returns a bigint
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		offset := ctx.GetInt(-1)
		ctx.Pop()

		pushBigInt(sw.peek(offset), ctx)
		return 1
	})
	vm.PutPropString(obj, "peek")
}

// dbWrapper provides a JavaScript wrapper around vm.Database.
type dbWrapper struct {
	db vm.StateDB
}

// pushObject assembles a JSVM object wrapping a swappable database and pushes it
// onto the VM stack.
func (dw *dbWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Push the wrapper for statedb.GetBalance
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		pushBigInt(dw.db.GetBalance(common.BytesToAddress(popSlice(ctx))), ctx)
		return 1
	})
	vm.PutPropString(obj, "getBalance")

	// Push the wrapper for statedb.GetNonce
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ctx.PushInt(int(dw.db.GetNonce(common.BytesToAddress(popSlice(ctx)))))
		return 1
	})
	vm.PutPropString(obj, "getNonce")

	// Push the wrapper for statedb.GetCode
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		code := dw.db.GetCode(common.BytesToAddress(popSlice(ctx)))

		ptr := ctx.PushFixedBuffer(len(code))
		copy(makeSlice(ptr, uint(len(code))), code)
		return 1
	})
	vm.PutPropString(obj, "getCode")

	// Push the wrapper for statedb.GetState
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		hash := popSlice(ctx)
		addr := popSlice(ctx)

		state := dw.db.GetState(common.BytesToAddress(addr), common.BytesToHash(hash))

		ptr := ctx.PushFixedBuffer(len(state))
		copy(makeSlice(ptr, uint(len(state))), state[:])
		return 1
	})
	vm.PutPropString(obj, "getState")

	// Push the wrapper for statedb.Exists
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ctx.PushBoolean(dw.db.Exist(common.BytesToAddress(popSlice(ctx))))
		return 1
	})
	vm.PutPropString(obj, "exists")
}

// contractWrapper provides a JavaScript wrapper around vm.Contract
type contractWrapper struct {
	contract *vm.Contract
}

// pushObject assembles a JSVM object wrapping a swappable contract and pushes it
// onto the VM stack.
func (cw *contractWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Push the wrapper for contract.Caller
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ptr := ctx.PushFixedBuffer(20)
		copy(makeSlice(ptr, 20), cw.contract.Caller().Bytes())
		return 1
	})
	vm.PutPropString(obj, "getCaller")

	// Push the wrapper for contract.Address
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ptr := ctx.PushFixedBuffer(20)
		copy(makeSlice(ptr, 20), cw.contract.Address().Bytes())
		return 1
	})
	vm.PutPropString(obj, "getAddress")

	// Push the wrapper for contract.Value
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		pushBigInt(cw.contract.Value(), ctx)
		return 1
	})
	vm.PutPropString(obj, "getValue")

	// Push the wrapper for contract.Input
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		blob := cw.contract.Input

		ptr := ctx.PushFixedBuffer(len(blob))
		copy(makeSlice(ptr, uint(len(blob))), blob)
		return 1
	})
	vm.PutPropString(obj, "getInput")
}

type frame struct {
	typ   *string
	from  *common.Address
	to    *common.Address
	input []byte
	gas   *uint
	value *big.Int
}

func newFrame() *frame {
	return &frame{
		typ:  new(string),
		from: new(common.Address),
		to:   new(common.Address),
		gas:  new(uint),
	}
}

func (f *frame) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, *f.typ); return 1 })
	vm.PutPropString(obj, "getType")

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, *f.from); return 1 })
	vm.PutPropString(obj, "getFrom")

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, *f.to); return 1 })
	vm.PutPropString(obj, "getTo")

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, f.input); return 1 })
	vm.PutPropString(obj, "getInput")

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, *f.gas); return 1 })
	vm.PutPropString(obj, "getGas")

	vm.PushGoFunction(func(ctx *duktape.Context) int {
		if f.value != nil {
			pushValue(ctx, f.value)
		} else {
			ctx.PushUndefined()
		}
		return 1
	})
	vm.PutPropString(obj, "getValue")
}

type frameResult struct {
	gasUsed    *uint
	output     []byte
	errorValue *string
}

func newFrameResult() *frameResult {
	return &frameResult{
		gasUsed: new(uint),
	}
}

func (r *frameResult) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, *r.gasUsed); return 1 })
	vm.PutPropString(obj, "getGasUsed")

	vm.PushGoFunction(func(ctx *duktape.Context) int { pushValue(ctx, r.output); return 1 })
	vm.PutPropString(obj, "getOutput")

	vm.PushGoFunction(func(ctx *duktape.Context) int {
		if r.errorValue != nil {
			pushValue(ctx, *r.errorValue)
		} else {
			ctx.PushUndefined()
		}
		return 1
	})
	vm.PutPropString(obj, "getError")
}

// jsTracer provides an implementation of Tracer that evaluates a Javascript
// function for each VM execution step.
type jsTracer struct {
	vm  *duktape.Context // Javascript VM instance
	env *vm.EVM          // EVM instance executing the code being traced

	tracerObject int // Stack index of the tracer JavaScript object
	stateObject  int // Stack index of the global state to pull arguments from

	opWrapper       *opWrapper       // Wrapper around the VM opcode
	stackWrapper    *stackWrapper    // Wrapper around the VM stack
	memoryWrapper   *memoryWrapper   // Wrapper around the VM memory
	contractWrapper *contractWrapper // Wrapper around the contract object
	dbWrapper       *dbWrapper       // Wrapper around the VM environment

	pcValue     *uint   // Swappable pc value wrapped by a log accessor
	gasValue    *uint   // Swappable gas value wrapped by a log accessor
	costValue   *uint   // Swappable cost value wrapped by a log accessor
	depthValue  *uint   // Swappable depth value wrapped by a log accessor
	errorValue  *string // Swappable error value wrapped by a log accessor
	refundValue *uint   // Swappable refund value wrapped by a log accessor

	frame       *frame       // Represents entry into call frame. Fields are swappable
	frameResult *frameResult // Represents exit from a call frame. Fields are swappable

	ctx map[string]interface{} // Transaction context gathered throughout execution
	err error                  // Error, if one has occurred

	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption

	activePrecompiles []common.Address // Updated on CaptureStart based on given rules
	traceSteps        bool             // When true, will invoke step() on each opcode
	traceCallFrames   bool             // When true, will invoke enter() and exit() js funcs
}

// New instantiates a new tracer instance. code specifies a Javascript snippet,
// which must evaluate to an expression returning an object with 'step', 'fault'
// and 'result' functions.
func newJsTracer(code string, ctx *tracers2.Context) (tracers2.Tracer, error) {
	if c, ok := assetTracers[code]; ok {
		code = c
	}
	if ctx == nil {
		ctx = new(tracers2.Context)
	}
	tracer := &jsTracer{
		vm:              duktape.New(),
		ctx:             make(map[string]interface{}),
		opWrapper:       new(opWrapper),
		stackWrapper:    new(stackWrapper),
		memoryWrapper:   new(memoryWrapper),
		contractWrapper: new(contractWrapper),
		dbWrapper:       new(dbWrapper),
		pcValue:         new(uint),
		gasValue:        new(uint),
		costValue:       new(uint),
		depthValue:      new(uint),
		refundValue:     new(uint),
		frame:           newFrame(),
		frameResult:     newFrameResult(),
	}
	if ctx.BlockHash != (common.Hash{}) {
		tracer.ctx["blockHash"] = ctx.BlockHash

		if ctx.TxHash != (common.Hash{}) {
			tracer.ctx["txIndex"] = ctx.TxIndex
			tracer.ctx["txHash"] = ctx.TxHash
		}
	}
	// Set up builtins for this environment
	tracer.vm.PushGlobalGoFunction("toHex", func(ctx *duktape.Context) int {
		ctx.PushString(hexutil.Encode(popSlice(ctx)))
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toWord", func(ctx *duktape.Context) int {
		var word common.Hash
		if ptr, size := ctx.GetBuffer(-1); ptr != nil {
			word = common.BytesToHash(makeSlice(ptr, size))
		} else {
			word = common.HexToHash(ctx.GetString(-1))
		}
		ctx.Pop()
		copy(makeSlice(ctx.PushFixedBuffer(32), 32), word[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toAddress", func(ctx *duktape.Context) int {
		var addr common.Address
		if ptr, size := ctx.GetBuffer(-1); ptr != nil {
			addr = common.BytesToAddress(makeSlice(ptr, size))
		} else {
			addr = common.HexToAddress(ctx.GetString(-1))
		}
		ctx.Pop()
		copy(makeSlice(ctx.PushFixedBuffer(20), 20), addr[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toContract", func(ctx *duktape.Context) int {
		var from common.Address
		if ptr, size := ctx.GetBuffer(-2); ptr != nil {
			from = common.BytesToAddress(makeSlice(ptr, size))
		} else {
			from = common.HexToAddress(ctx.GetString(-2))
		}
		nonce := uint64(ctx.GetInt(-1))
		ctx.Pop2()

		contract := crypto.CreateAddress(from, nonce)
		copy(makeSlice(ctx.PushFixedBuffer(20), 20), contract[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toContract2", func(ctx *duktape.Context) int {
		var from common.Address
		if ptr, size := ctx.GetBuffer(-3); ptr != nil {
			from = common.BytesToAddress(makeSlice(ptr, size))
		} else {
			from = common.HexToAddress(ctx.GetString(-3))
		}
		// Retrieve salt hex string from js stack
		salt := common.HexToHash(ctx.GetString(-2))
		// Retrieve code slice from js stack
		var code []byte
		if ptr, size := ctx.GetBuffer(-1); ptr != nil {
			code = common.CopyBytes(makeSlice(ptr, size))
		} else {
			code = common.FromHex(ctx.GetString(-1))
		}
		codeHash := crypto.Keccak256(code)
		ctx.Pop3()
		contract := crypto.CreateAddress2(from, salt, codeHash)
		copy(makeSlice(ctx.PushFixedBuffer(20), 20), contract[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("isPrecompiled", func(ctx *duktape.Context) int {
		addr := common.BytesToAddress(popSlice(ctx))
		for _, p := range tracer.activePrecompiles {
			if p == addr {
				ctx.PushBoolean(true)
				return 1
			}
		}
		ctx.PushBoolean(false)
		return 1
	})
	tracer.vm.PushGlobalGoFunction("slice", func(ctx *duktape.Context) int {
		start, end := ctx.GetInt(-2), ctx.GetInt(-1)
		ctx.Pop2()

		blob := popSlice(ctx)
		size := end - start

		if start < 0 || start > end || end > len(blob) {
			// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
			// runtime goes belly up https://github.com/golang/go/issues/15639.
			log.Warn("Tracer accessed out of bound memory", "available", len(blob), "offset", start, "size", size)
			ctx.PushFixedBuffer(0)
			return 1
		}
		copy(makeSlice(ctx.PushFixedBuffer(size), uint(size)), blob[start:end])
		return 1
	})
	// Push the JavaScript tracer as object #0 onto the JSVM stack and validate it
	if err := tracer.vm.PevalString("(" + code + ")"); err != nil {
		log.Warn("Failed to compile tracer", "err", err)
		return nil, err
	}
	tracer.tracerObject = 0 // yeah, nice, eval can't return the index itself

	hasStep := tracer.vm.GetPropString(tracer.tracerObject, "step")
	tracer.vm.Pop()

	if !tracer.vm.GetPropString(tracer.tracerObject, "fault") {
		return nil, fmt.Errorf("trace object must expose a function fault()")
	}
	tracer.vm.Pop()

	if !tracer.vm.GetPropString(tracer.tracerObject, "result") {
		return nil, fmt.Errorf("trace object must expose a function result()")
	}
	tracer.vm.Pop()

	hasEnter := tracer.vm.GetPropString(tracer.tracerObject, "enter")
	tracer.vm.Pop()
	hasExit := tracer.vm.GetPropString(tracer.tracerObject, "exit")
	tracer.vm.Pop()
	if hasEnter != hasExit {
		return nil, fmt.Errorf("trace object must expose either both or none of enter() and exit()")
	}
	tracer.traceCallFrames = hasEnter && hasExit
	tracer.traceSteps = hasStep

	// Tracer is valid, inject the big int library to access large numbers
	tracer.vm.EvalString(bigIntegerJS)
	tracer.vm.PutGlobalString("bigInt")

	// Push the global environment state as object #1 into the JSVM stack
	tracer.stateObject = tracer.vm.PushObject()

	logObject := tracer.vm.PushObject()

	tracer.opWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "op")

	tracer.stackWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "stack")

	tracer.memoryWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "memory")

	tracer.contractWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "contract")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.pcValue); return 1 })
	tracer.vm.PutPropString(logObject, "getPC")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.gasValue); return 1 })
	tracer.vm.PutPropString(logObject, "getGas")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.costValue); return 1 })
	tracer.vm.PutPropString(logObject, "getCost")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.depthValue); return 1 })
	tracer.vm.PutPropString(logObject, "getDepth")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.refundValue); return 1 })
	tracer.vm.PutPropString(logObject, "getRefund")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int {
		if tracer.errorValue != nil {
			ctx.PushString(*tracer.errorValue)
		} else {
			ctx.PushUndefined()
		}
		return 1
	})
	tracer.vm.PutPropString(logObject, "getError")

	tracer.vm.PutPropString(tracer.stateObject, "log")

	tracer.frame.pushObject(tracer.vm)
	tracer.vm.PutPropString(tracer.stateObject, "frame")

	tracer.frameResult.pushObject(tracer.vm)
	tracer.vm.PutPropString(tracer.stateObject, "frameResult")

	tracer.dbWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(tracer.stateObject, "db")

	return tracer, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (jst *jsTracer) Stop(err error) {
	jst.reason = err
	atomic.StoreUint32(&jst.interrupt, 1)
}

// call executes a method on a JS object, catching any errors, formatting and
// returning them as error objects.
func (jst *jsTracer) call(noret bool, method string, args ...string) (json.RawMessage, error) {
	// Execute the JavaScript call and return any error
	jst.vm.PushString(method)
	for _, arg := range args {
		jst.vm.GetPropString(jst.stateObject, arg)
	}
	code := jst.vm.PcallProp(jst.tracerObject, len(args))
	defer jst.vm.Pop()

	if code != 0 {
		err := jst.vm.SafeToString(-1)
		return nil, errors.New(err)
	}
	// No error occurred, extract return value and return
	if noret {
		return nil, nil
	}
	// Push a JSON marshaller onto the stack. We can't marshal from the out-
	// side because duktape can crash on large nestings and we can't catch
	// C++ exceptions ourselves from Go. TODO(karalabe): Yuck, why wrap?!
	jst.vm.PushString("(JSON.stringify)")
	jst.vm.Eval()

	jst.vm.Swap(-1, -2)
	if code = jst.vm.Pcall(1); code != 0 {
		err := jst.vm.SafeToString(-1)
		return nil, errors.New(err)
	}
	return json.RawMessage(jst.vm.SafeToString(-1)), nil
}

func wrapError(context string, err error) error {
	return fmt.Errorf("%v    in server-side tracer function '%v'", err, context)
}

// CaptureStart implements the Tracer interface to initialize the tracing operation.
func (jst *jsTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	jst.env = env
	jst.ctx["type"] = "CALL"
	if create {
		jst.ctx["type"] = "CREATE"
	}
	jst.ctx["from"] = from
	jst.ctx["to"] = to
	jst.ctx["input"] = input
	jst.ctx["gas"] = gas
	jst.ctx["gasPrice"] = env.TxContext.GasPrice
	jst.ctx["value"] = value

	// Initialize the context
	jst.ctx["block"] = env.Context.BlockNumber.Uint64()
	jst.dbWrapper.db = env.StateDB
	// Update list of precompiles based on current block
	rules := env.ChainConfig().Rules(env.Context.BlockNumber)
	jst.activePrecompiles = vm.ActivePrecompiles(rules)

	// Compute intrinsic gas
	isHomestead := env.ChainConfig().IsHomestead(env.Context.BlockNumber)
	isIstanbul := env.ChainConfig().IsIstanbul(env.Context.BlockNumber)
	intrinsicGas, err := core.IntrinsicGas(input, nil, jst.ctx["type"] == "CREATE", isHomestead, isIstanbul)
	if err != nil {
		return
	}
	jst.ctx["intrinsicGas"] = intrinsicGas
}

// CaptureState implements the Tracer interface to trace a single step of VM execution.
func (jst *jsTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if !jst.traceSteps {
		return
	}
	if jst.err != nil {
		return
	}
	// If tracing was interrupted, set the error and stop
	if atomic.LoadUint32(&jst.interrupt) > 0 {
		jst.err = jst.reason
		jst.env.Cancel()
		return
	}
	jst.opWrapper.op = op
	jst.stackWrapper.stack = scope.Stack
	jst.memoryWrapper.memory = scope.Memory
	jst.contractWrapper.contract = scope.Contract

	*jst.pcValue = uint(pc)
	*jst.gasValue = uint(gas)
	*jst.costValue = uint(cost)
	*jst.depthValue = uint(depth)
	*jst.refundValue = uint(jst.env.StateDB.GetRefund())

	jst.errorValue = nil
	if err != nil {
		jst.errorValue = new(string)
		*jst.errorValue = err.Error()
	}

	if _, err := jst.call(true, "step", "log", "db"); err != nil {
		jst.err = wrapError("step", err)
	}
}

// CaptureFault implements the Tracer interface to trace an execution fault
func (jst *jsTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	if jst.err != nil {
		return
	}
	// Apart from the error, everything matches the previous invocation
	jst.errorValue = new(string)
	*jst.errorValue = err.Error()

	if _, err := jst.call(true, "fault", "log", "db"); err != nil {
		jst.err = wrapError("fault", err)
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (jst *jsTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	jst.ctx["output"] = output
	jst.ctx["time"] = t.String()
	jst.ctx["gasUsed"] = gasUsed

	if err != nil {
		jst.ctx["error"] = err.Error()
	}
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (jst *jsTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if !jst.traceCallFrames {
		return
	}
	if jst.err != nil {
		return
	}
	// If tracing was interrupted, set the error and stop
	if atomic.LoadUint32(&jst.interrupt) > 0 {
		jst.err = jst.reason
		return
	}

	*jst.frame.typ = typ.String()
	*jst.frame.from = from
	*jst.frame.to = to
	jst.frame.input = common.CopyBytes(input)
	*jst.frame.gas = uint(gas)
	jst.frame.value = nil
	if value != nil {
		jst.frame.value = new(big.Int).SetBytes(value.Bytes())
	}

	if _, err := jst.call(true, "enter", "frame"); err != nil {
		jst.err = wrapError("enter", err)
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (jst *jsTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	if !jst.traceCallFrames {
		return
	}
	// If tracing was interrupted, set the error and stop
	if atomic.LoadUint32(&jst.interrupt) > 0 {
		jst.err = jst.reason
		return
	}

	jst.frameResult.output = common.CopyBytes(output)
	*jst.frameResult.gasUsed = uint(gasUsed)
	jst.frameResult.errorValue = nil
	if err != nil {
		jst.frameResult.errorValue = new(string)
		*jst.frameResult.errorValue = err.Error()
	}

	if _, err := jst.call(true, "exit", "frameResult"); err != nil {
		jst.err = wrapError("exit", err)
	}
}

// GetResult calls the Javascript 'result' function and returns its value, or any accumulated error
func (jst *jsTracer) GetResult() (json.RawMessage, error) {
	// Transform the context into a JavaScript object and inject into the state
	obj := jst.vm.PushObject()

	for key, val := range jst.ctx {
		jst.addToObj(obj, key, val)
	}
	jst.vm.PutPropString(jst.stateObject, "ctx")

	// Finalize the trace and return the results
	result, err := jst.call(false, "result", "ctx", "db")
	if err != nil {
		jst.err = wrapError("result", err)
	}
	// Clean up the JavaScript environment
	jst.vm.DestroyHeap()
	jst.vm.Destroy()

	return result, jst.err
}

// addToObj pushes a field to a JS object.
func (jst *jsTracer) addToObj(obj int, key string, val interface{}) {
	pushValue(jst.vm, val)
	jst.vm.PutPropString(obj, key)
}

func pushValue(ctx *duktape.Context, val interface{}) {
	switch val := val.(type) {
	case uint64:
		ctx.PushUint(uint(val))
	case string:
		ctx.PushString(val)
	case []byte:
		ptr := ctx.PushFixedBuffer(len(val))
		copy(makeSlice(ptr, uint(len(val))), val)
	case common.Address:
		ptr := ctx.PushFixedBuffer(20)
		copy(makeSlice(ptr, 20), val[:])
	case *big.Int:
		pushBigInt(val, ctx)
	case int:
		ctx.PushInt(val)
	case uint:
		ctx.PushUint(val)
	case common.Hash:
		ptr := ctx.PushFixedBuffer(32)
		copy(makeSlice(ptr, 32), val[:])
	default:
		panic(fmt.Sprintf("unsupported type: %T", val))
	}
}
