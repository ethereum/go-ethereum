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
	"github.com/holiman/uint256"
)

// Config are the configuration options for the Interpreter
type Config struct {
	Tracer                  *tracing.Hooks
	NoBaseFee               bool  // Forces the EIP-1559 baseFee to 0 (needed for 0 price calls)
	EnablePreimageRecording bool  // Enables recording of SHA3/keccak preimages
	ExtraEips               []int // Additional EIPS that are to be enabled

	StatelessSelfValidation bool // Generate execution witnesses and self-check against them (testing purpose)
	EnableWitnessStats      bool // Whether trie access statistics collection is enabled
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

// ContractCode returns the code of the contract being executed.
func (ctx *ScopeContext) ContractCode() []byte {
	return ctx.Contract.Code
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// ErrExecutionReverted which means revert-and-keep-gas-left.
func (evm *EVM) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	// Increment the call depth which is restricted to 1024
	evm.depth++
	defer func() { evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !evm.readOnly {
		evm.readOnly = true
		defer func() { evm.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	evm.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op          OpCode     // current opcode
		jumpTable   *JumpTable = evm.table
		mem                    = NewMemory() // bound memory
		stack                  = newstack()  // local stack
		callContext            = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract,
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		// copies used by tracer
		pcCopy    uint64 // needed for the deferred EVMLogger
		gasCopy   uint64 // for EVMLogger to log gas remaining before execution
		logged    bool   // deferred EVMLogger should ignore already logged steps
		res       []byte // result of the opcode execution function
		debug     = evm.Config.Tracer != nil
		isEIP4762 = evm.chainRules.IsEIP4762
	)
	// Don't move this deferred function, it's placed before the OnOpcode-deferred method,
	// so that it gets executed _after_: the OnOpcode needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
		mem.Free()
	}()
	contract.Input = input

	if debug {
		defer func() { // this deferred method handles exit-with-error
			if err == nil {
				return
			}
			if !logged && evm.Config.Tracer.OnOpcode != nil {
				evm.Config.Tracer.OnOpcode(pcCopy, byte(op), gasCopy, cost, callContext, evm.returnData, evm.depth, VMErrorFromErr(err))
			}
			if logged && evm.Config.Tracer.OnFault != nil {
				evm.Config.Tracer.OnFault(pcCopy, byte(op), gasCopy, cost, callContext, evm.depth, VMErrorFromErr(err))
			}
		}()
	}
	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	_ = jumpTable[0] // nil-check the jumpTable out of the loop
	for {
		if debug {
			// Capture pre-execution values for tracing.
			logged, pcCopy, gasCopy = false, pc, contract.Gas
		}

		if isEIP4762 && !contract.IsDeployment && !contract.IsSystemCall {
			// if the PC ends up in a new "chunk" of verkleized code, charge the
			// associated costs.
			contractAddr := contract.Address()
			consumed, wanted := evm.TxContext.AccessEvents.CodeChunksRangeGas(contractAddr, pc, 1, uint64(len(contract.Code)), false, contract.Gas)
			contract.UseGas(consumed, evm.Config.Tracer, tracing.GasChangeWitnessCodeChunk)
			if consumed < wanted {
				return nil, ErrOutOfGas
			}
		}

		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := jumpTable[op]
		cost = operation.constantGas // For tracing
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		// for tracing: this gas consumption event is emitted below in the debug section.
		if contract.Gas < cost {
			return nil, ErrOutOfGas
		} else {
			contract.Gas -= cost
		}

		// All ops with a dynamic memory usage also has a dynamic gas cost.
		var memorySize uint64
		if operation.dynamicGas != nil {
			// calculate the new memory size and expand the memory to fit
			// the operation
			// Memory check needs to be done prior to evaluating the dynamic gas portion,
			// to detect calculation overflows
			if operation.memorySize != nil {
				memSize, overflow := operation.memorySize(stack)
				if overflow {
					return nil, ErrGasUintOverflow
				}
				// memory is expanded in words of 32 bytes. Gas
				// is also calculated in words.
				if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
					return nil, ErrGasUintOverflow
				}
			}
			// Consume the gas and return an error if not enough gas is available.
			// cost is explicitly set so that the capture state defer method can get the proper cost
			var dynamicCost uint64
			dynamicCost, err = operation.dynamicGas(evm, contract, stack, mem, memorySize)
			cost += dynamicCost // for tracing
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
			}
			// for tracing: this gas consumption event is emitted below in the debug section.
			if contract.Gas < dynamicCost {
				return nil, ErrOutOfGas
			} else {
				contract.Gas -= dynamicCost
			}
		}

		// Do tracing before potential memory expansion
		if debug {
			if evm.Config.Tracer.OnGasChange != nil {
				evm.Config.Tracer.OnGasChange(gasCopy, gasCopy-cost, tracing.GasChangeCallOpCode)
			}
			if evm.Config.Tracer.OnOpcode != nil {
				evm.Config.Tracer.OnOpcode(pc, byte(op), gasCopy, cost, callContext, evm.returnData, evm.depth, VMErrorFromErr(err))
				logged = true
			}
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}

		// execute the operation
		res, err = operation.execute(&pc, evm, callContext)
		if err != nil {
			break
		}
		pc++
	}

	if err == errStopToken {
		err = nil // clear stop token error
	}

	return res, err
}
