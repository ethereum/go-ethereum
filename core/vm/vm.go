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
		jumpTable: newJumpTable(env.ChainConfig(), env.BlockNumber()),
		cfg:       cfg,
		gasTable:  env.ChainConfig().GasTable(env.BlockNumber()),
	}
}

// Run loops and evaluates the contract's code with the given input data
func (evm *EVM) Run(contract *Contract, input []byte) ([]byte, error) {
	evm.env.SetDepth(evm.env.Depth() + 1)
	defer evm.env.SetDepth(evm.env.Depth() - 1)

	if contract.CodeAddr != nil {
		if p, exist := PrecompiledContracts[*contract.CodeAddr]; exist {
			return RunPrecompiled(p, input, contract)
		}
	}

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	codehash := contract.CodeHash // codehash is used as an identifier for the programs
	if codehash == (common.Hash{}) {
		codehash = crypto.Keccak256Hash(contract.Code)
	}
	// If the JIT is enabled check the status of the JIT program,
	// if it doesn't exist compile a new program in a separate
	// goroutine or wait for compilation to finish if the JIT is
	// forced.
	switch GetProgramStatus(codehash) {
	case progReady:
		return evm.runProgram(GetProgram(codehash), contract, input)
	case progUnknown:
		// Create and compile program
		program := NewProgram(contract.Code)
		CompileProgram(program)

		return evm.runProgram(program, contract, input)
	case progCompile:
		// if the program is already compling wait for the compilation to finish
		// and use the program instead of defaulting to the regular byte code vm.
		<-WaitCompile(codehash)

		return evm.runProgram(GetProgram(codehash), contract, input)
	}
	return nil, fmt.Errorf("Unexpected return using program %x", codehash)
}

func (evm *EVM) runProgram(program *Program, contract *Contract, input []byte) ([]byte, error) {
	contract.Input = input

	var (
		pc         uint64 = 0 //program.mapping[pcstart]
		instrCount uint64 = 0
		mem               = NewMemory()
		stack             = newstack()
		env               = evm.env
	)

	if glog.V(logger.Debug) {
		glog.Infof("running JIT program %x\n", program.Id[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("JIT program %x done. time: %v instrc: %v\n", program.Id[:4], time.Since(tstart), instrCount)
		}()
	}

	homestead := env.ChainConfig().IsHomestead(env.BlockNumber())
	for pc < uint64(len(program.instructions)) {
		instrCount++

		instr := program.instructions[pc]
		if instr.Op() == DELEGATECALL && !homestead {
			return nil, fmt.Errorf("Invalid opcode 0x%x", instr.Op())
		}

		ret, err := instr.do(evm, program, &pc, env, contract, mem, stack)
		if err != nil {
			//gas := new(big.Int).SetUint64(contract.gas64)
			//evm.cfg.Tracer.CaptureState(evm.env, pc, instr.Op(), gas, cost, mem, stack, contract, evm.env.Depth(), err)

			return nil, err
		}

		if instr.halts() {
			return ret, nil
		}
	}

	contract.Input = nil

	return nil, nil
}
