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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// Config are the configuration options for the Interpreter
type Config struct {
	Tracer                  *tracing.Hooks
	NoBaseFee               bool  // Forces the EIP-1559 baseFee to 0 (needed for 0 price calls)
	EnablePreimageRecording bool  // Enables recording of SHA3/keccak preimages
	ExtraEips               []int // Additional EIPS that are to be enabled
}

// ScopeContext contains the things that are per-call, such as stack and memory,
// but not transients like pc and gas
type ScopeContext struct {
	Memory   *Memory
	Stack    *Stack
	Contract *Contract
}

// MemoryData returns the underlying memory slice. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) MemoryData() []byte {
	if ctx.Memory == nil {
		return nil
	}
	return ctx.Memory.Data()
}

// StackData returns the stack data. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) StackData() []uint256.Int {
	if ctx.Stack == nil {
		return nil
	}
	return ctx.Stack.Data()
}

// Caller returns the current caller.
func (ctx *ScopeContext) Caller() common.Address {
	return ctx.Contract.Caller()
}

// Address returns the address where this scope of execution is taking place.
func (ctx *ScopeContext) Address() common.Address {
	return ctx.Contract.Address()
}

// CallValue returns the value supplied with this call.
func (ctx *ScopeContext) CallValue() *uint256.Int {
	return ctx.Contract.Value()
}

// CallInput returns the input/calldata with this call. Callers must not modify
// the contents of the returned data.
func (ctx *ScopeContext) CallInput() []byte {
	return ctx.Contract.Input
}

// EVMInterpreter represents an EVM interpreter
type EVMInterpreter struct {
	evm   *EVM
	table *JumpTable

	hasher    crypto.KeccakState // Keccak256 hasher instance shared across opcodes
	hasherBuf common.Hash        // Keccak256 hasher result array shared across opcodes

	readOnly   bool   // Whether to throw on stateful modifications
	returnData []byte // Last CALL's return data for subsequent reuse
}

// NewEVMInterpreter returns a new instance of the Interpreter.
func NewEVMInterpreter(evm *EVM) *EVMInterpreter {
	// If jump table was not initialised we set the default one.
	var table *JumpTable
	switch {
	case evm.chainRules.IsCancun:
		table = &cancunInstructionSet
	case evm.chainRules.IsShanghai:
		table = &shanghaiInstructionSet
	case evm.chainRules.IsMerge:
		table = &mergeInstructionSet
	case evm.chainRules.IsLondon:
		table = &londonInstructionSet
	case evm.chainRules.IsBerlin:
		table = &berlinInstructionSet
	case evm.chainRules.IsIstanbul:
		table = &istanbulInstructionSet
	case evm.chainRules.IsConstantinople:
		table = &constantinopleInstructionSet
	case evm.chainRules.IsByzantium:
		table = &byzantiumInstructionSet
	case evm.chainRules.IsEIP158:
		table = &spuriousDragonInstructionSet
	case evm.chainRules.IsEIP150:
		table = &tangerineWhistleInstructionSet
	case evm.chainRules.IsHomestead:
		table = &homesteadInstructionSet
	default:
		table = &frontierInstructionSet
	}
	var extraEips []int
	if len(evm.Config.ExtraEips) > 0 {
		// Deep-copy jumptable to prevent modification of opcodes in other tables
		table = copyJumpTable(table)
	}
	for _, eip := range evm.Config.ExtraEips {
		if err := EnableEIP(eip, table); err != nil {
			// Disable it, so caller can check if it's activated or not
			log.Error("EIP activation failed", "eip", eip, "error", err)
		} else {
			extraEips = append(extraEips, eip)
		}
	}
	evm.Config.ExtraEips = extraEips
	return &EVMInterpreter{evm: evm, table: table}
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// ErrExecutionReverted which means revert-and-keep-gas-left.
func (in *EVMInterpreter) run(state *InterpreterState, input []byte, readOnly bool) (ret []byte, err error) {
	defer func() {
		state.finished = true
	}()

	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(state.Contract.Code) == 0 {
		return nil, nil
	}

	gethState := NewGethState(state.Contract, state.Memory, state.Stack, state.pc)
	gethState.Contract.Input = input

	debug := in.evm.Config.Tracer != nil
	var logged bool

	if debug {
		defer func() { // this deferred method handles exit-with-error
			if err == nil {
				return
			}
			if !logged && in.evm.Config.Tracer.OnOpcode != nil {
				in.evm.Config.Tracer.OnOpcode(gethState.pcCopy, byte(gethState.op), gethState.gasCopy, gethState.cost, gethState.CallContext, in.returnData, in.evm.depth, VMErrorFromErr(err))
			}
			if logged && in.evm.Config.Tracer.OnFault != nil {
				in.evm.Config.Tracer.OnFault(gethState.pcCopy, byte(gethState.op), gethState.gasCopy, gethState.cost, gethState.CallContext, in.evm.depth, VMErrorFromErr(err))
			}
		}()
	}

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	steps := 0
	for {
		steps++
		if in.evm.abort.Load() {
			break
		}

		if !in.Step(gethState) {
			break
		}
	}

	if gethState.Err == errStopToken {
		gethState.Err = nil // clear stop token error
	}

	return gethState.Result, gethState.Err
}

func (in *EVMInterpreter) Step(state *GethState) bool {
	debug := in.evm.Config.Tracer != nil
	if debug {
		// Capture pre-execution values for tracing.
		state.logged, state.pcCopy, state.gasCopy = false, state.Pc, state.Contract.Gas
	}
	// Get the operation from the jump table and validate the stack to ensure there are
	// enough stack items available to perform the operation.
	state.op = state.Contract.GetOp(state.Pc)
	operation := in.table[state.op]
	cost := operation.constantGas // For tracing
	// Validate stack
	if sLen := state.Stack.len(); sLen < operation.minStack {
		state.Err = &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		return false

	} else if sLen > operation.maxStack {
		state.Err = &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		return false
	}
	if !state.Contract.UseGas(cost, in.evm.Config.Tracer, tracing.GasChangeIgnored) {
		state.Err = ErrOutOfGas
		return false
	}

	if operation.dynamicGas != nil {
		// All ops with a dynamic memory usage also has a dynamic gas cost.
		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		// Memory check needs to be done prior to evaluating the dynamic gas portion,
		// to detect calculation overflows
		if operation.memorySize != nil {
			memSize, overflow := operation.memorySize(state.Stack)
			if overflow {
				state.Err = ErrGasUintOverflow
				return false
			}
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
				state.Err = ErrGasUintOverflow
				return false
			}
		}
		// Consume the gas and return an error if not enough gas is available.
		// cost is explicitly set so that the capture state defer method can get the proper cost
		var dynamicCost uint64
		dynamicCost, state.Err = operation.dynamicGas(in.evm, state.Contract, state.Stack, state.Memory, memorySize)
		cost += dynamicCost // for tracing
		if state.Err != nil {
			state.Err = fmt.Errorf("%w: %v", ErrOutOfGas, state.Err)
			return false
		}
		if !state.Contract.UseGas(dynamicCost, in.evm.Config.Tracer, tracing.GasChangeIgnored) {
			state.Err = ErrOutOfGas
			return false
		}

		// Do tracing before memory expansion
		if debug {
			if in.evm.Config.Tracer.OnGasChange != nil {
				in.evm.Config.Tracer.OnGasChange(state.gasCopy, state.gasCopy-cost, tracing.GasChangeCallOpCode)
			}
			if in.evm.Config.Tracer.OnOpcode != nil {
				in.evm.Config.Tracer.OnOpcode(state.Pc, byte(state.op), state.gasCopy, cost, state.CallContext, in.returnData, in.evm.depth, VMErrorFromErr(state.Err))
				state.logged = true
			}
		}
		if memorySize > 0 {
			state.Memory.Resize(memorySize)
		}
	} else if debug {
		if in.evm.Config.Tracer.OnGasChange != nil {
			in.evm.Config.Tracer.OnGasChange(state.gasCopy, state.gasCopy-cost, tracing.GasChangeCallOpCode)
		}
		if in.evm.Config.Tracer.OnOpcode != nil {
			in.evm.Config.Tracer.OnOpcode(state.Pc, byte(state.op), state.gasCopy, cost, state.CallContext, in.returnData, in.evm.depth, VMErrorFromErr(state.Err))
			state.logged = true
		}
	}

	// execute the operation
	state.Result, state.Err = operation.execute(&state.Pc, in, state.CallContext)
	if state.Err != nil {
		return false
	}
	state.Pc++

	return state.Err == nil
}
