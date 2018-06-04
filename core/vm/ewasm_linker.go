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

package vm

import (
	"fmt"

	"github.com/go-interpreter/wagon/wasm"
)

var eeiFunctionList = []string{
	"useGas",
	"getAddress",
	"getExternalBalance",
	"getBlockHash",
	"call",
	"callDataCopy",
	"getCallDataSize",
	"callCode",
	"callDelegate",
	"callStatic",
	"storageStore",
	"storageLoad",
	"getCaller",
	"getCallValue",
	"codeCopy",
	"getCodeSize",
	"getBlockCoinbase",
	"create",
	"getBlockDifficulty",
	"externalCodeCopy",
	"getExternalCodeSize",
	"getGasLeft",
	"getBlockGasLimit",
	"getTxGasPrice",
	"log",
	"getBlockNumber",
	"getTxOrigin",
	"finish",
	"revert",
	"getReturnDataSize",
	"returnDataCopy",
	"selfDestruct",
	"getBlockTimestamp",
}

var debugFunctionList = []string{
	"printMemHex",
	"printStorageHex",
}

// ModuleResolver matches all EEI functions to native go functions
func ModuleResolver(interpreter *InterpreterEWASM, name string) (*wasm.Module, error) {
	if name == "debug" {
		debugModule := wasm.NewModule()
		debugModule.Types = eeiTypes
		debugModule.FunctionIndexSpace = getDebugFuncs(interpreter)
		entries := make(map[string]wasm.ExportEntry)
		for idx, name := range debugFunctionList {
			entries[name] = wasm.ExportEntry{
				FieldStr: name,
				Kind:     wasm.ExternalFunction,
				Index:    uint32(idx),
			}
		}
		debugModule.Export = &wasm.SectionExports{
			Entries: entries,
		}
		return debugModule, nil
	}

	if name != "env" && name != "ethereum" {
		return nil, fmt.Errorf("Unknown module name: %s", name)
	}

	m := wasm.NewModule()
	m.Types = eeiTypes
	m.FunctionIndexSpace = eeiFuncs(interpreter)

	entries := make(map[string]wasm.ExportEntry)

	for idx, name := range eeiFunctionList {
		entries[name] = wasm.ExportEntry{
			FieldStr: name,
			Kind:     wasm.ExternalFunction,
			Index:    uint32(idx),
		}
	}

	m.Export = &wasm.SectionExports{
		Entries: entries,
	}

	return m, nil
}

// WrappedModuleResolver returns a module resolver function that whose
// EEI functions are closure-bound to a given interpreter.
// This is the first step to closure hell, the plan is to improve PR #59
// in wagon to be able to pass some context.
func WrappedModuleResolver(in *InterpreterEWASM) wasm.ResolveFunc {
	return func(name string) (*wasm.Module, error) {
		return ModuleResolver(in, name)
	}
}
