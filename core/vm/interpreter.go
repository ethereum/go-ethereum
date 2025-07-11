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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// Config are the configuration options for the Interpreter
type Config struct {
	Tracer                  tracing.Hooks
	NoBaseFee               bool  // Forces the EIP-1559 baseFee to 0 (needed for 0 price calls)
	EnablePreimageRecording bool  // Enables recording of SHA3/keccak preimages
	ExtraEips               []int // Additional EIPS that are to be enabled

	StatelessSelfValidation bool // Generate execution witnesses and self-check against them (testing purpose)
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
	case evm.chainRules.IsOsaka:
		table = &osakaInstructionSet
	case evm.chainRules.IsVerkle:
		// TODO replace with proper instruction set when fork is specified
		table = &verkleInstructionSet
	case evm.chainRules.IsPrague:
		table = &pragueInstructionSet
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
	if len(evm.Config.ExtraEips) > 0 || evm.Config.Tracer.HooksVM() {
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
	if evm.Config.Tracer.HooksVM() {
		for _, op := range table {
			op.execute = executeWithTracer(&evm.Config.Tracer, op.execute)
		}
	}

	evm.Config.ExtraEips = extraEips
	return &EVMInterpreter{evm: evm, table: table, hasher: crypto.NewKeccakState()}
}

// executeWithTracer is a wrapper that enables tracing the given executionFunc
func executeWithTracer(tracer *tracing.Hooks, execF executionFunc) executionFunc {
	return func(pc *uint64, interpreter *EVMInterpreter, callContext *ScopeContext) ([]byte, error) {
		pcCopy := *pc
		op := callContext.Contract.GetOp(pcCopy)
		operation := interpreter.table[op]
		cost := operation.constantGas
		gasCopy := callContext.Contract.Gas + cost

		if tracer.OnGasChange != nil {
			tracer.OnGasChange(gasCopy, gasCopy-cost, tracing.GasChangeCallOpCode)
		}
		if tracer.OnOpcode != nil {
			tracer.OnOpcode(pcCopy, byte(op), gasCopy, cost, callContext, interpreter.returnData,
				interpreter.evm.depth, nil)
		}

		ret, err := execF(pc, interpreter, callContext)
		if err != nil && err != errStopToken && tracer.OnFault != nil {
			tracer.OnFault(pcCopy, byte(op), gasCopy, gasCopy-callContext.Contract.Gas, callContext, interpreter.evm.depth, VMErrorFromErr(err))
		}
		return ret, err
	}
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// ErrExecutionReverted which means revert-and-keep-gas-left.
func (in *EVMInterpreter) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
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
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op          OpCode     // current opcode
		jumpTable   *JumpTable = in.table
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
		res  []byte // result of the opcode execution function
	)
	// Don't move this deferred function, it's placed before the OnOpcode-deferred method,
	// so that it gets executed _after_: the OnOpcode needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
		mem.Free()
	}()
	contract.Input = input

	// This is just temporary until we move everything that can error in to the execution function
	traceAndReturnError := func(err error) error {
		if in.evm.Config.Tracer.OnFault != nil {
			in.evm.Config.Tracer.OnFault(pc, byte(op), contract.Gas, cost, callContext, in.evm.depth, VMErrorFromErr(err))
		}
		return err
	}

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	_ = jumpTable[0] // nil-check the jumpTable out of the loop
	for {
		if in.evm.chainRules.IsEIP4762 && !contract.IsDeployment && !contract.IsSystemCall {
			// if the PC ends up in a new "chunk" of verkleized code, charge the
			// associated costs.
			contractAddr := contract.Address()
			consumed, wanted := in.evm.TxContext.AccessEvents.CodeChunksRangeGas(contractAddr, pc, 1, uint64(len(contract.Code)), false, contract.Gas)
			contract.UseGas(consumed, in.evm.Config.Tracer.OnGasChange, tracing.GasChangeWitnessCodeChunk)
			if consumed < wanted {
				return nil, traceAndReturnError(ErrOutOfGas)
			}
		}

		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := jumpTable[op]
		cost = operation.constantGas // For tracing
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, traceAndReturnError(&ErrStackUnderflow{stackLen: sLen, required: operation.minStack})
		} else if sLen > operation.maxStack {
			return nil, traceAndReturnError(&ErrStackOverflow{stackLen: sLen, limit: operation.maxStack})
		}
		// for tracing: this gas consumption event is emitted in the executeWithTracer wrapper.
		if contract.Gas < cost {
			return nil, traceAndReturnError(ErrOutOfGas)
		} else {
			contract.Gas -= cost
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
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
