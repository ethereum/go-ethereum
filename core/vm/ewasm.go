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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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

	metering bool

	meteringContract   *Contract
	meteringModule     *wasm.Module
	meteringStartIndex int64
}

// NewEWASMInterpreter creates a new wagon-based ewasm interpreter. It
// currently takes a *vm.EVM pointer as a proxy to the client's internal
// state; this will be fixed in subsequent updates.
func NewEWASMInterpreter(evm *EVM, cfg Config) Interpreter {
	metering := cfg.EWASMInterpreter["metering"] == "true"

	inter := &InterpreterEWASM{
		StateDB:  evm.StateDB,
		evm:      evm,
		gasTable: evm.chainConfig.GasTable(evm.BlockNumber),
		metering: metering,
	}

	if metering {
		meteringContractAddress := common.HexToAddress(sentinelContractAddress)
		meteringCode := evm.StateDB.GetCode(meteringContractAddress)

		var err error
		inter.meteringModule, err = wasm.ReadModule(bytes.NewReader(meteringCode), WrappedModuleResolver(inter))
		if err != nil {
			panic(fmt.Sprintf("Error loading the metering contract: %v", err))
		}
		// TODO when the metering contract abides by that rule, check that it
		// only exports "main" and "memory".
		inter.meteringStartIndex = int64(inter.meteringModule.Export.Entries["main"].Index)
		mainSig := inter.meteringModule.FunctionIndexSpace[inter.meteringStartIndex].Sig
		if len(mainSig.ParamTypes) != 0 || len(mainSig.ReturnTypes) != 0 {
			panic(fmt.Sprintf("Invalid main function for the metering contract: index=%d sig=%v", inter.meteringStartIndex, mainSig))
		}
	}

	return inter
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
		in.terminationType = TerminateInvalid
		return nil, fmt.Errorf("Error decoding module at address %s: %v", contract.Address().Hex(), err)
	}

	vm, err := exec.NewVM(module)
	if err != nil {
		in.terminationType = TerminateInvalid
		return nil, fmt.Errorf("could not create the vm: %v", err)
	}
	vm.RecoverPanic = true
	in.vm = vm

	mainIndex, err := validateModule(module)
	if err != nil {
		in.terminationType = TerminateInvalid
		return nil, err
	}


	// Check input and output types
	sig := module.FunctionIndexSpace[mainIndex].Sig
	if len(sig.ParamTypes) == 0 && len(sig.ReturnTypes) == 0 {
		_, err = vm.ExecCode(int64(mainIndex))

		if err != nil && err != errExecutionReverted {
			in.terminationType = TerminateInvalid
		}

		if in.StateDB.HasSuicided(contract.Address()) {
			in.StateDB.AddRefund(params.SuicideRefundGas)
			err = nil
		}

		return in.returnData, err
	}

	in.terminationType = TerminateInvalid
	return nil, errors.New("Could not find a suitable 'main' function in that contract")
}

// CanRun checks the binary for a WASM header and accepts the binary blob
// if it matches.
func (in *InterpreterEWASM) CanRun(file []byte) bool {
	// Check the header
	if len(file) < 4 || string(file[:4]) != "\000asm" {
		return false
	}


	return true
}

// PreContractCreation meters the contract's its init code before it
// is run.
func (in *InterpreterEWASM) PreContractCreation(code []byte, contract *Contract) ([]byte, error) {
	savedContract := in.contract
	in.contract = contract

	defer func() {
		in.contract = savedContract
	}()

	if in.metering {
		metered, _, err := sentinel(in, code)
		if len(metered) < 5 || err != nil {
			return nil, fmt.Errorf("Error metering the init contract code, err=%v", err)
		}
		return metered, nil
	}
	return code, nil
}

func validateModule(m *wasm.Module) (int, error) {
	// A module should not have a start section
	if m.Start != nil {
		return -1, fmt.Errorf("Module has a start section")
	}

	// Only two exports are authorized: "main" and "memory"
	if m.Export == nil {
		return -1, fmt.Errorf("Module has no exports instead of 2")
	}
	if len(m.Export.Entries) != 2 {
		return -1, fmt.Errorf("Module has %d exports instead of 2", len(m.Export.Entries))
	}

	mainIndex := -1
	for name, entry := range m.Export.Entries {
		switch name {
		case "main":
			if entry.Kind != wasm.ExternalFunction {
				return -1, fmt.Errorf("Main is not a function in module")
			}
			mainIndex = int(entry.Index)
			break
		case "memory":
			if entry.Kind != wasm.ExternalMemory {
				return -1, fmt.Errorf("'memory' is not a memory in module")
			}
			break
		default:
			return -1, fmt.Errorf("A symbol named %s has been exported. Only main and memory should exist", name)
		}
	}

	if m.Import != nil {
	OUTER:
		for _, entry := range m.Import.Entries {
			if entry.ModuleName == "ethereum" {
				if entry.Type.Kind() == wasm.ExternalFunction {
					for _, name := range eeiFunctionList {
						if name == entry.FieldName {
							continue OUTER
						}
					}
					return -1, fmt.Errorf("%s could not be found in the list of ethereum-provided functions", entry.FieldName)
				}
			}
		}
	}

	return mainIndex, nil
}

// PostContractCreation meters the contract once its init code has
// been run. It also validates the module's format before it is to
// be committed to disk.
func (in *InterpreterEWASM) PostContractCreation(code []byte) ([]byte, error) {
	// If a REVERT has been encountered, then return the code and
	if in.terminationType == TerminateRevert {
		return nil, errExecutionReverted
	}
		if in.metering {
			code, _, err := sentinel(in, code)
			if len(code) < 5 || err != nil {
				return nil, fmt.Errorf("Error metering the generated contract code, err=%v", err)
			}

			if len(code) < 8 {
				return nil, fmt.Errorf("Invalid contract code")
			}
		}

		if len(code) > 8 {
			// Check the validity of the module
			m, err := wasm.DecodeModule(bytes.NewReader(code))
			if err != nil {
				return nil, fmt.Errorf("Error decoding the module produced by init code: %v", err)
			}

			_, err = validateModule(m)
			if err != nil {
				in.terminationType = TerminateInvalid
				return nil, err
			}
		}
	}

	return code, nil
}
