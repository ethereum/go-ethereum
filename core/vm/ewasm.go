// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// This file lists the EEI functions, so that they can be bound to any
// ewasm-compatible module, as well as the types of these functions

package vm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/params"
	"github.com/go-interpreter/wagon/wasm"

	"github.com/go-interpreter/wagon/exec"
)

type terminationType int

// List of termination reasons
const (
	TerminateFinish = iota
	TerminateRevert
	TerminateSuicide
	TerminateInvalid
)

// InterpreterEWASM implements the Interpreter interface for ewasm.
type InterpreterEWASM struct {
	vm *exec.VM

	evm *EVM

	StateDB StateDB

	gasTable params.GasTable

	contract *Contract

	returnData []byte

	terminationType terminationType

	staticMode bool
}

// NewEWASMInterpreter creates a new wagon-based ewasm interpreter. It
// currently takes a *vm.EVM pointer as a proxy to the client's internal
// state; this will be fixed in subsequent updates.
func NewEWASMInterpreter(evm *EVM, cfg Config) Interpreter {
	return &InterpreterEWASM{StateDB: evm.StateDB, evm: evm, gasTable: evm.chainConfig.GasTable(evm.BlockNumber)}
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
func (in *InterpreterEWASM) Run(contract *Contract, input []byte, ro bool) ([]byte, error) {
	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	in.contract = contract
	in.contract.Input = input
	initialGas := contract.Gas

	module, err := wasm.ReadModule(bytes.NewReader(contract.Code), WrappedModuleResolver(in))
	if err != nil {
		return nil, fmt.Errorf("Error decoding module at address %s: %v", contract.Address(), err)
	}

	// The module should not have any start function
	if module.Start != nil {
		return nil, fmt.Errorf("A contract should not have a start function: found #%d", module.Start.Index)
	}

	vm, err := exec.NewVM(module)
	if err != nil {
		return nil, fmt.Errorf("could not create the vm: %v", err)
	}
	vm.RecoverPanic = true
	in.vm = vm

	// Look for the "main" function and execute it after checking it
	// has the right kind of signature.
	for name, entry := range module.Export.Entries {
		if name == "main" && entry.Kind == wasm.ExternalFunction {

			// Check input and output types
			sig := module.FunctionIndexSpace[entry.Index].Sig
			if len(sig.ParamTypes) == 0 && len(sig.ReturnTypes) == 0 {
				_, err = vm.ExecCode(int64(entry.Index))

				if err != nil {
					in.terminationType = TerminateInvalid
				}

				if in.StateDB.HasSuicided(contract.Address()) {
					if initialGas-contract.Gas-params.TxGas < 2*params.SuicideRefundGas {
						in.StateDB.AddRefund((initialGas - contract.Gas - params.TxGas) / 2)
					} else {
						in.StateDB.AddRefund(params.SuicideRefundGas)
					}
					err = nil
				}

				return in.returnData, err
			}

			// Found a main but it doesn't have the right signature - fail
			break
		}
	}

	return nil, errors.New("Could not find a suitable 'main' function in that contract")
}

// CanRun checks the binary for a WASM header and accepts the binary blob
// if it matches.
func (in *InterpreterEWASM) CanRun(file []byte) bool {
	// Check the header
	if len(file) <= 8 || string(file[:4]) != "\000asm" {
		return false
	}

	// Check the version
	ver := binary.LittleEndian.Uint32(file[4:])
	if ver != 1 {
		return false
	}

	return true
}
