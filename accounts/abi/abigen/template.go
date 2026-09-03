// Copyright 2016 The go-ethereum Authors
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

package abigen

import (
	_ "embed"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// tmplData is the data structure required to fill the binding template.
type tmplData struct {
	Package   string                   // Name of the package to place the generated file in
	Contracts map[string]*tmplContract // List of contracts to generate into this file
	Libraries map[string]string        // Map the bytecode's link pattern to the library name
	Structs   map[string]*tmplStruct   // Contract struct type definitions
}

// tmplContract contains the data needed to generate an individual contract binding.
type tmplContract struct {
	Type        string                 // Type name of the main contract binding
	InputABI    string                 // JSON ABI used as the input to generate the binding from
	InputBin    string                 // Optional EVM bytecode used to generate deploy code from
	FuncSigs    map[string]string      // Optional map: string signature -> 4-byte signature
	Constructor abi.Method             // Contract constructor for deploy parametrization
	Calls       map[string]*tmplMethod // Contract calls that only read state data
	Transacts   map[string]*tmplMethod // Contract calls that write state data
	Fallback    *tmplMethod            // Additional special fallback function
	Receive     *tmplMethod            // Additional special receive function
	Events      map[string]*tmplEvent  // Contract events accessors
	Libraries   map[string]string      // Same as tmplData, but filtered to only keep direct deps that the contract needs
	Library     bool                   // Indicator whether the contract is a library
}

type tmplContractV2 struct {
	Type        string                 // Type name of the main contract binding
	InputABI    string                 // JSON ABI used as the input to generate the binding from
	InputBin    string                 // Optional EVM bytecode used to generate deploy code from
	Constructor abi.Method             // Contract constructor for deploy parametrization
	Calls       map[string]*tmplMethod // All contract methods (excluding fallback, receive)
	Events      map[string]*tmplEvent  // Contract events accessors
	Libraries   map[string]string      // all direct library dependencies
	Errors      map[string]*tmplError  // all errors defined
}

func newTmplContractV2(typ string, abiStr string, bytecode string, constructor abi.Method, cb *contractBinder) *tmplContractV2 {
	// Strip any whitespace from the JSON ABI
	strippedABI := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, abiStr)
	return &tmplContractV2{
		abi.ToCamelCase(typ),
		strings.ReplaceAll(strippedABI, "\"", "\\\""),
		strings.TrimPrefix(strings.TrimSpace(bytecode), "0x"),
		constructor,
		cb.calls,
		cb.events,
		make(map[string]string),
		cb.errors,
	}
}

type tmplDataV2 struct {
	Package   string                     // Name of the package to use for the generated bindings
	Contracts map[string]*tmplContractV2 // Contracts that will be emitted in the bindings (keyed by contract name)
	Libraries map[string]string          // Map of the contract's name to link pattern
	Structs   map[string]*tmplStruct     // Contract struct type definitions
}

// tmplMethod is a wrapper around an abi.Method that contains a few preprocessed
// and cached data fields.
type tmplMethod struct {
	Original   abi.Method // Original method as parsed by the abi package
	Normalized abi.Method // Normalized version of the parsed method (capitalized names, non-anonymous args/returns)
	Structured bool       // Whether the returns should be accumulated into a struct
}

// tmplEvent is a wrapper around an abi.Event that contains a few preprocessed
// and cached data fields.
type tmplEvent struct {
	Original   abi.Event // Original event as parsed by the abi package
	Normalized abi.Event // Normalized version of the parsed fields
}

// tmplError is a wrapper around an abi.Error that contains a few preprocessed
// and cached data fields.
type tmplError struct {
	Original   abi.Error
	Normalized abi.Error
}

// tmplField is a wrapper around a struct field with binding language
// struct type definition and relative filed name.
type tmplField struct {
	Type    string   // Field type representation depends on target binding language
	Name    string   // Field name converted from the raw user-defined field name
	SolKind abi.Type // Raw abi type information
}

// tmplStruct is a wrapper around an abi.tuple and contains an auto-generated
// struct name.
type tmplStruct struct {
	Name   string       // Auto-generated struct name(before solidity v0.5.11) or raw name.
	Fields []*tmplField // Struct fields definition depends on the binding language.
}

// tmplSource is the Go source template that the generated Go contract binding
// is based on.
//
//go:embed source.go.tpl
var tmplSource string

// tmplSourceV2 is the Go source template that the generated Go contract binding
// for abigen v2 is based on.
//
//go:embed source2.go.tpl
var tmplSourceV2 string
