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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

// Config are the configuration options for the EVM
type Config struct {
	Debug     bool
	EnableJit bool
	ForceJit  bool
	Tracer    Tracer
}

// EVM is used to run Ethereum based contracts and will utilise the
// passed environment to query external sources for state information.
// The EVM will run the byte code VM or JIT VM based on the passed
// configuration.
type EVM struct {
	env       Environment
	jumpTable vmJumpTable
	cfg       Config
	gasTable  params.GasTable
}

// New returns a new instance of the EVM.
func New(env Environment, cfg Config) *EVM {
	return &EVM{
		env:       env,
		jumpTable: newJumpTable(env.RuleSet(), env.BlockNumber()),
		cfg:       cfg,
		gasTable:  env.RuleSet().GasTable(env.BlockNumber()),
	}
}

// Run loops and evaluates the contract's code with the given input data
func (evm *EVM) Run(contract *Contract, input []byte) (ret []byte, err error) {
	evm.env.SetDepth(evm.env.Depth() + 1)
	defer evm.env.SetDepth(evm.env.Depth() - 1)

	if contract.CodeAddr != nil {
		if p := Precompiled[contract.CodeAddr.Str()]; p != nil {
			return evm.RunPrecompiled(p, input, contract)
		}
	}

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	codehash := contract.CodeHash // codehash is used when doing jump dest caching
	if codehash == (common.Hash{}) {
		codehash = crypto.Keccak256Hash(contract.Code)
	}
	var program *Program
	if false {
		// JIT disabled due to JIT not being Homestead gas reprice ready.

		// If the JIT is enabled check the status of the JIT program,
		// if it doesn't exist compile a new program in a separate
		// goroutine or wait for compilation to finish if the JIT is
		// forced.
		switch GetProgramStatus(codehash) {
		case progReady:
			return RunProgram(GetProgram(codehash), evm.env, contract, input)
		case progUnknown:
			if evm.cfg.ForceJit {
				// Create and compile program
				program = NewProgram(contract.Code)
				perr := CompileProgram(program)
				if perr == nil {
					return RunProgram(program, evm.env, contract, input)
				}
				glog.V(logger.Info).Infoln("error compiling program", err)
			} else {
				// create and compile the program. Compilation
				// is done in a separate goroutine
				program = NewProgram(contract.Code)
				go func() {
					err := CompileProgram(program)
					if err != nil {
						glog.V(logger.Info).Infoln("error compiling program", err)
						return
					}
				}()
			}
		}
	}

	var (
		caller     = contract.caller
		code       = contract.Code
		instrCount = 0

		op      OpCode         // current opcode
		mem     = NewMemory()  // bound memory
		stack   = newstack()   // local stack
		statedb = evm.env.Db() // current state
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC to be uint256. Practically much less so feasible.
		pc = uint64(0) // program counter

		// jump evaluates and checks whether the given jump destination is a valid one
		// if valid move the `pc` otherwise return an error.
		jump = func(from uint64, to *big.Int) error {
			if !contract.jumpdests.has(codehash, code, to) {
				nop := contract.GetOp(to.Uint64())
				return fmt.Errorf("invalid jump destination (%v) %v", nop, to)
			}

			pc = to.Uint64()

			return nil
		}

		newMemSize *big.Int
		cost       *big.Int
	)
	contract.Input = input

	// User defer pattern to check for an error and, based on the error being nil or not, use all gas and return.
	defer func() {
		if err != nil && evm.cfg.Debug {
			evm.cfg.Tracer.CaptureState(evm.env, pc, op, contract.Gas, cost, mem, stack, contract, evm.env.Depth(), err)
		}
	}()

	if glog.V(logger.Debug) {
		glog.Infof("running byte VM %x\n", codehash[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("byte VM %x done. time: %v instrc: %v\n", codehash[:4], time.Since(tstart), instrCount)
		}()
	}

	for ; ; instrCount++ {
		/*
			if EnableJit && it%100 == 0 {
				if program != nil && progStatus(atomic.LoadInt32(&program.status)) == progReady {
					// move execution
					fmt.Println("moved", it)
					glog.V(logger.Info).Infoln("Moved execution to JIT")
					return runProgram(program, pc, mem, stack, evm.env, contract, input)
				}
			}
		*/

		// Get the memory location of pc
		op = contract.GetOp(pc)
		// calculate the new memory size and gas price for the current executing opcode
		newMemSize, cost, err = calculateGasAndSize(evm.gasTable, evm.env, contract, caller, op, statedb, mem, stack)
		if err != nil {
			return nil, err
		}

		// Use the calculated gas. When insufficient gas is present, use all gas and return an
		// Out Of Gas error
		if !contract.UseGas(cost) {
			return nil, OutOfGasError
		}

		// Resize the memory calculated previously
		mem.Resize(newMemSize.Uint64())
		// Add a log message
		if evm.cfg.Debug {
			evm.cfg.Tracer.CaptureState(evm.env, pc, op, contract.Gas, cost, mem, stack, contract, evm.env.Depth(), nil)
		}

		if opPtr := evm.jumpTable[op]; opPtr.valid {
			if opPtr.fn != nil {
				opPtr.fn(instruction{}, &pc, evm.env, contract, mem, stack)
			} else {
				switch op {
				case PC:
					opPc(instruction{data: new(big.Int).SetUint64(pc)}, &pc, evm.env, contract, mem, stack)
				case JUMP:
					if err := jump(pc, stack.pop()); err != nil {
						return nil, err
					}

					continue
				case JUMPI:
					pos, cond := stack.pop(), stack.pop()

					if cond.Cmp(common.BigTrue) >= 0 {
						if err := jump(pc, pos); err != nil {
							return nil, err
						}

						continue
					}
				case RETURN:
					offset, size := stack.pop(), stack.pop()
					ret := mem.GetPtr(offset.Int64(), size.Int64())

					return ret, nil
				case SUICIDE:
					opSuicide(instruction{}, nil, evm.env, contract, mem, stack)

					fallthrough
				case STOP: // Stop the contract
					return nil, nil
				}
			}
		} else {
			return nil, fmt.Errorf("Invalid opcode %x", op)
		}

		pc++

	}
}

// calculateGasAndSize calculates the required given the opcode and stack items calculates the new memorysize for
// the operation. This does not reduce gas or resizes the memory.
func calculateGasAndSize(gasTable params.GasTable, env Environment, contract *Contract, caller ContractRef, op OpCode, statedb Database, mem *Memory, stack *Stack) (*big.Int, *big.Int, error) {
	var (
		gas                 = new(big.Int)
		newMemSize *big.Int = new(big.Int)
	)
	err := baseCheck(op, stack, gas)
	if err != nil {
		return nil, nil, err
	}

	// stack Check, memory resize & gas phase
	switch op {
	case SUICIDE:
		// if suicide is not nil: homestead gas fork
		if gasTable.CreateBySuicide != nil {
			gas.Set(gasTable.Suicide)
			if !env.Db().Exist(common.BigToAddress(stack.data[len(stack.data)-1])) {
				gas.Add(gas, gasTable.CreateBySuicide)
			}
		}

		if !statedb.HasSuicided(contract.Address()) {
			statedb.AddRefund(params.SuicideRefundGas)
		}
	case EXTCODESIZE:
		gas.Set(gasTable.ExtcodeSize)
	case BALANCE:
		gas.Set(gasTable.Balance)
	case SLOAD:
		gas.Set(gasTable.SLoad)
	case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
		n := int(op - SWAP1 + 2)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
		n := int(op - DUP1 + 1)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case LOG0, LOG1, LOG2, LOG3, LOG4:
		n := int(op - LOG0)
		err := stack.require(n + 2)
		if err != nil {
			return nil, nil, err
		}

		mSize, mStart := stack.data[stack.len()-2], stack.data[stack.len()-1]

		gas.Add(gas, params.LogGas)
		gas.Add(gas, new(big.Int).Mul(big.NewInt(int64(n)), params.LogTopicGas))
		gas.Add(gas, new(big.Int).Mul(mSize, params.LogDataGas))

		newMemSize = calcMemSize(mStart, mSize)

		quadMemGas(mem, newMemSize, gas)
	case EXP:
		gas.Add(gas, new(big.Int).Mul(big.NewInt(int64(len(stack.data[stack.len()-2].Bytes()))), params.ExpByteGas))
	case SSTORE:
		err := stack.require(2)
		if err != nil {
			return nil, nil, err
		}

		var g *big.Int
		y, x := stack.data[stack.len()-2], stack.data[stack.len()-1]
		val := statedb.GetState(contract.Address(), common.BigToHash(x))

		// This checks for 3 scenario's and calculates gas accordingly
		// 1. From a zero-value address to a non-zero value         (NEW VALUE)
		// 2. From a non-zero value address to a zero-value address (DELETE)
		// 3. From a non-zero to a non-zero                         (CHANGE)
		if common.EmptyHash(val) && !common.EmptyHash(common.BigToHash(y)) {
			// 0 => non 0
			g = params.SstoreSetGas
		} else if !common.EmptyHash(val) && common.EmptyHash(common.BigToHash(y)) {
			statedb.AddRefund(params.SstoreRefundGas)

			g = params.SstoreClearGas
		} else {
			// non 0 => non 0 (or 0 => 0)
			g = params.SstoreResetGas
		}
		gas.Set(g)
	case MLOAD:
		newMemSize = calcMemSize(stack.peek(), u256(32))
		quadMemGas(mem, newMemSize, gas)
	case MSTORE8:
		newMemSize = calcMemSize(stack.peek(), u256(1))
		quadMemGas(mem, newMemSize, gas)
	case MSTORE:
		newMemSize = calcMemSize(stack.peek(), u256(32))
		quadMemGas(mem, newMemSize, gas)
	case RETURN:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-2])
		quadMemGas(mem, newMemSize, gas)
	case SHA3:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-2])

		words := toWordSize(stack.data[stack.len()-2])
		gas.Add(gas, words.Mul(words, params.Sha3WordGas))

		quadMemGas(mem, newMemSize, gas)
	case CALLDATACOPY:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-3])

		words := toWordSize(stack.data[stack.len()-3])
		gas.Add(gas, words.Mul(words, params.CopyGas))

		quadMemGas(mem, newMemSize, gas)
	case CODECOPY:
		newMemSize = calcMemSize(stack.peek(), stack.data[stack.len()-3])

		words := toWordSize(stack.data[stack.len()-3])
		gas.Add(gas, words.Mul(words, params.CopyGas))

		quadMemGas(mem, newMemSize, gas)
	case EXTCODECOPY:
		gas.Set(gasTable.ExtcodeCopy)

		newMemSize = calcMemSize(stack.data[stack.len()-2], stack.data[stack.len()-4])

		words := toWordSize(stack.data[stack.len()-4])
		gas.Add(gas, words.Mul(words, params.CopyGas))

		quadMemGas(mem, newMemSize, gas)
	case CREATE:
		newMemSize = calcMemSize(stack.data[stack.len()-2], stack.data[stack.len()-3])

		quadMemGas(mem, newMemSize, gas)
	case CALL, CALLCODE:
		gas.Set(gasTable.Calls)

		if op == CALL {
			if !env.Db().Exist(common.BigToAddress(stack.data[stack.len()-2])) {
				gas.Add(gas, params.CallNewAccountGas)
			}
		}
		if len(stack.data[stack.len()-3].Bytes()) > 0 {
			gas.Add(gas, params.CallValueTransferGas)
		}
		x := calcMemSize(stack.data[stack.len()-6], stack.data[stack.len()-7])
		y := calcMemSize(stack.data[stack.len()-4], stack.data[stack.len()-5])

		newMemSize = common.BigMax(x, y)

		quadMemGas(mem, newMemSize, gas)

		cg := callGas(gasTable, contract.Gas, gas, stack.data[stack.len()-1])
		// Replace the stack item with the new gas calculation. This means that
		// either the original item is left on the stack or the item is replaced by:
		// (availableGas - gas) * 63 / 64
		// We replace the stack item so that it's available when the opCall instruction is
		// called. This information is otherwise lost due to the dependency on *current*
		// available gas.
		stack.data[stack.len()-1] = cg
		gas.Add(gas, cg)

	case DELEGATECALL:
		gas.Set(gasTable.Calls)

		x := calcMemSize(stack.data[stack.len()-5], stack.data[stack.len()-6])
		y := calcMemSize(stack.data[stack.len()-3], stack.data[stack.len()-4])

		newMemSize = common.BigMax(x, y)

		quadMemGas(mem, newMemSize, gas)

		cg := callGas(gasTable, contract.Gas, gas, stack.data[stack.len()-1])
		// Replace the stack item with the new gas calculation. This means that
		// either the original item is left on the stack or the item is replaced by:
		// (availableGas - gas) * 63 / 64
		// We replace the stack item so that it's available when the opCall instruction is
		// called.
		stack.data[stack.len()-1] = cg
		gas.Add(gas, cg)

	}

	return newMemSize, gas, nil
}

// RunPrecompile runs and evaluate the output of a precompiled contract defined in contracts.go
func (evm *EVM) RunPrecompiled(p *PrecompiledAccount, input []byte, contract *Contract) (ret []byte, err error) {
	gas := p.Gas(len(input))
	if contract.UseGas(gas) {
		ret = p.Call(input)

		return ret, nil
	} else {
		return nil, OutOfGasError
	}
}
