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
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

// Config are the configuration options for the EVM
type Config struct {
	// Debug enabled debugging EVM options
	Debug bool
	// EnableJit enabled the JIT VM
	EnableJit bool
	// ForceJit forces the JIT VM
	ForceJit bool
	// Tracer is the op code logger
	Tracer Tracer
	// NoRecursion disabled EVM call, callcode,
	// delegate call and create.
	NoRecursion bool
	// Disable gas metering
	DisableGasMetering bool
}

// EVM is used to run Ethereum based contracts and will utilise the
// passed environment to query external sources for state information.
// The EVM will run the byte code VM or JIT VM based on the passed
// configuration.
type EVM struct {
	env       *Environment
	jumpTable vmJumpTable
	cfg       Config
	gasTable  params.GasTable

	// done is an atomic int and is used for
	// cancellation during RunWithContext.
	done int32
}

// New returns a new instance of the EVM.
func New(env *Environment, cfg Config) *EVM {
	return &EVM{
		env:       env,
		jumpTable: newJumpTable(env.ChainConfig(), env.BlockNumber),
		cfg:       cfg,
		gasTable:  env.ChainConfig().GasTable(env.BlockNumber),
	}
}

// RunWithContext allows the EVM to be ran with a cancellation method by passing in a context.Context. The EVM
// behaves exactly the same as an EVM without a context.
//
// RunWithContext is only used for the initial call and shouldn't be called more than once.
func (evm *EVM) RunWithContext(ctx context.Context, contract *Contract, input []byte) (ret []byte, err error) {
	go func() {
		<-ctx.Done()
		atomic.StoreInt32(&evm.done, 1)
	}()
	return evm.Run(contract, input)
}

// Run loops and evaluates the contract's code with the given input data
func (evm *EVM) Run(contract *Contract, input []byte) (ret []byte, err error) {
	evm.env.Depth++
	defer func() { evm.env.Depth-- }()

	if contract.CodeAddr != nil {
		if p := PrecompiledContracts[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
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

	var (
		op    OpCode        // current opcode
		mem   = NewMemory() // bound memory
		stack = newstack()  // local stack
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost *big.Int
	)
	contract.Input = input

	// User defer pattern to check for an error and, based on the error being nil or not, use all gas and return.
	defer func() {
		if err != nil && evm.cfg.Debug {
			evm.cfg.Tracer.CaptureState(evm.env, pc, op, contract.Gas, cost, mem, stack, contract, evm.env.Depth, err)
		}
	}()

	if glog.V(logger.Debug) {
		glog.Infof("evm running: %x\n", codehash[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("evm done: %x. time: %v\n", codehash[:4], time.Since(tstart))
		}()
	}

	// The EVM main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SUICIDE is executed, an error accured during
	// the execution of one of the operations or until the evm.done is set by
	// the parent context.Context.
	for atomic.LoadInt32(&evm.done) == 0 {
		// Get the memory location of pc
		op = contract.GetOp(pc)

		// get the operation from the jump table matching the opcode
		operation := evm.jumpTable[op]

		// if the op is invalid abort the process and return an error
		if !operation.valid {
			return nil, fmt.Errorf("Invalid opcode %x", op)
		}

		// validate the stack and make sure there enough stack items available
		// to perform the operation
		if err := operation.validateStack(stack); err != nil {
			return nil, err
		}

		var memorySize *big.Int
		// calculate the new memory size and expand the memory to fit
		// the operation
		if operation.memorySize != nil {
			memorySize = operation.memorySize(stack)
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			memorySize.Mul(toWordSize(memorySize), big.NewInt(32))
		}

		if !evm.cfg.DisableGasMetering {
			// consume the gas and return an error if not enough gas is available.
			// cost is explicitly set so that the capture state defer method cas get the proper cost
			cost = operation.gasCost(evm.gasTable, evm.env, contract, stack, mem, memorySize)
			if !contract.UseGas(cost) {
				return nil, OutOfGasError
			}
		}
		if memorySize != nil {
			mem.Resize(memorySize.Uint64())
		}

		if evm.cfg.Debug {
			evm.cfg.Tracer.CaptureState(evm.env, pc, op, contract.Gas, cost, mem, stack, contract, evm.env.Depth, err)
		}

		// execute the operation
		res, err := operation.execute(&pc, evm.env, contract, mem, stack)
		switch {
		case err != nil:
			return nil, err
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}
	}
	return nil, nil
}
