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

// Package bind generates Ethereum contract Go bindings.
package bind

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/tools/imports"
)

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(jsonABI string, pkg string, kind string) (string, error) {
	// Parse the actual ABI to generate the binding for
	abi, err := abi.JSON(strings.NewReader(jsonABI))
	if err != nil {
		return "", err
	}
	// Generate the contract type, fields and methods
	code := new(bytes.Buffer)
	kind = strings.ToUpper(kind[:1]) + kind[1:]
	fmt.Fprintf(code, "%s\n", bindContract(kind, jsonABI))

	methods := make([]string, 0, len(abi.Methods))
	for name, _ := range abi.Methods {
		methods = append(methods, name)
	}
	sort.Strings(methods)

	for _, method := range methods {
		fmt.Fprintf(code, "%s\n", bindMethod(kind, abi.Methods[method]))
	}
	// Format the code with goimports and return
	buffer := new(bytes.Buffer)

	fmt.Fprintf(buffer, "package %s\n\n", pkg)
	fmt.Fprintf(buffer, "%s\n\n", string(code.Bytes()))

	blob, err := imports.Process("", buffer.Bytes(), nil)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

// bindContract generates the basic wrapper code for interacting with an Ethereum
// contract via the abi package. All contract methods will call into the generic
// ones generated here.
func bindContract(kind string, abi string) string {
	code := ""

	// Generate the hard coded ABI used for Ethereum interaction
	code += fmt.Sprintf("// Ethereum ABI used to generate the binding from.\nconst %sABI = `%s`\n\n", kind, strings.TrimSpace(abi))

	// Generate the Go struct with all the maintenance fields
	code += fmt.Sprintf("// %s is an auto generated Go binding around an Ethereum contract.\n", kind)
	code += fmt.Sprintf("type %s struct {\n", kind)
	code += fmt.Sprintf("  %sCaller     // Read-only binding to the contract\n", kind)
	code += fmt.Sprintf("  %sTransactor // Write-only binding to the contract\n", kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// %sCaller is an auto generated read-only Go binding around an Ethereum contract.\n", kind)
	code += fmt.Sprintf("type %sCaller struct {\n", kind)
	code += fmt.Sprintf("  common *common%s // Contract binding common to callers and transactors\n", kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// %sTransactor is an auto generated write-only Go binding around an Ethereum contract.\n", kind)
	code += fmt.Sprintf("type %sTransactor struct {\n", kind)
	code += fmt.Sprintf("  common *common%s // Contract binding common to callers and transactors\n", kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// common%s is an auto generated Go binding around an Ethereum contract.\n", kind)
	code += fmt.Sprintf("type common%s struct {\n", kind)
	code += fmt.Sprintf("  contract *bind.BoundContract // Generic contract wrapper for the low level calls\n")
	code += fmt.Sprintf("}\n\n")

	// Generate the constructor to create a bound contract
	code += fmt.Sprintf("// New%s creates a new instance of %s, bound to a specific deployed contract.\n", kind, kind)
	code += fmt.Sprintf("func New%s(address common.Address, backend bind.ContractBackend) (*%s, error) {\n", kind, kind)
	code += fmt.Sprintf("  common, err := newCommon%s(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))\n", kind)
	code += fmt.Sprintf("  if err != nil {\n")
	code += fmt.Sprintf("    return nil, err\n")
	code += fmt.Sprintf("  }\n")
	code += fmt.Sprintf("  return &%s{%sCaller: %sCaller{common: common}, %sTransactor: %sTransactor{common: common}}, nil\n", kind, kind, kind, kind, kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// New%sCaller creates a new read-only instance of %s, bound to a specific deployed contract.\n", kind, kind)
	code += fmt.Sprintf("func New%sCaller(address common.Address, caller bind.ContractCaller) (*%sCaller, error) {\n", kind, kind)
	code += fmt.Sprintf("  common, err := newCommon%s(address, caller, nil)\n", kind)
	code += fmt.Sprintf("  if err != nil {\n")
	code += fmt.Sprintf("    return nil, err\n")
	code += fmt.Sprintf("  }\n")
	code += fmt.Sprintf("  return &%sCaller{common: common}, nil\n", kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// New%sTransactor creates a new write-only instance of %s, bound to a specific deployed contract.\n", kind, kind)
	code += fmt.Sprintf("func New%sTransactor(address common.Address, transactor bind.ContractTransactor) (*%sTransactor, error) {\n", kind, kind)
	code += fmt.Sprintf("  common, err := newCommon%s(address, nil, transactor)\n", kind)
	code += fmt.Sprintf("  if err != nil {\n")
	code += fmt.Sprintf("    return nil, err\n")
	code += fmt.Sprintf("  }\n")
	code += fmt.Sprintf("  return &%sTransactor{common: common}, nil\n", kind)
	code += fmt.Sprintf("}\n\n")

	code += fmt.Sprintf("// newCommon%s creates an internal instance of %s, bound to a specific deployed contract.\n", kind, kind)
	code += fmt.Sprintf("func newCommon%s(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*common%s, error) {\n", kind, kind)
	code += fmt.Sprintf("  parsed, err := abi.JSON(strings.NewReader(%sABI))\n", kind)
	code += fmt.Sprintf("  if err != nil {\n")
	code += fmt.Sprintf("    return nil, err\n")
	code += fmt.Sprintf("  }\n")
	code += fmt.Sprintf("  return &common%s{\n", kind)
	code += fmt.Sprintf("    contract: bind.NewBoundContract(address, parsed, caller, transactor),\n")
	code += fmt.Sprintf("  }, nil\n")
	code += fmt.Sprintf("}")

	return code
}

// bindMethod
func bindMethod(kind string, method abi.Method) string {
	var (
		name     = strings.ToUpper(method.Name[:1]) + method.Name[1:]
		prologue = new(bytes.Buffer)
	)
	// Generate the argument and return list for the function
	args := make([]string, 0, len(method.Inputs))
	for i, arg := range method.Inputs {
		param := arg.Name
		if param == "" {
			param = fmt.Sprintf("arg%d", i)
		}
		args = append(args, fmt.Sprintf("%s %s", param, bindType(arg.Type)))
	}
	returns, _ := bindReturn(prologue, name, method.Outputs)

	// Generate the docs to help with coding against the binding
	callTypeDoc := "free data retrieval call"
	if !method.Const {
		callTypeDoc = "paid mutator transaction"
	}
	docs := fmt.Sprintf("// %s is a %s binding the contract method 0x%x.\n", name, callTypeDoc, method.Id())
	docs += fmt.Sprintf("// \n")
	docs += fmt.Sprintf("// Solidity: %s", strings.TrimPrefix(method.String(), "function "))

	// Generate the method itself for both the read/write version and the combo too
	code := fmt.Sprintf("%s\n", prologue)
	if method.Const {
		code += fmt.Sprintf("%s\nfunc (_%s *%sCaller) %s(%s) (%s) {\n%s\n}\n", docs, kind, kind, name, strings.Join(args, ","), strings.Join(returns, ","), bindCallBody(kind, method.Name, args, returns))
	} else {
		args = append([]string{"auth *bind.AuthOpts"}, args...)
		code += fmt.Sprintf("%s\nfunc (_%s *%sTransactor) %s(%s) (*types.Transaction, error) {\n%s\n}\n", docs, kind, kind, name, strings.Join(args, ","), bindTransactionBody(kind, method.Name, args))
	}
	return code
}

// bindType converts a Solidity type to a Go one. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. *big.Int).
func bindType(kind abi.Type) string {
	stringKind := kind.String()

	switch {
	case stringKind == "address":
		return "common.Address"

	case stringKind == "hash":
		return "common.Hash"

	case strings.HasPrefix(stringKind, "bytes"):
		if stringKind == "bytes" {
			return "[]byte"
		}
		return fmt.Sprintf("[%s]byte", stringKind[5:])

	case strings.HasPrefix(stringKind, "int"):
		switch stringKind[:3] {
		case "8", "16", "32", "64":
			return stringKind
		}
		return "*big.Int"

	case strings.HasPrefix(stringKind, "uint"):
		switch stringKind[:4] {
		case "8", "16", "32", "64":
			return stringKind
		}
		return "*big.Int"

	default:
		return stringKind
	}
}

// bindReturn creates the list of return parameters for a method invocation. If
// all the fields of the return type are named, and there is more than one value
// being returned, the returns are wrapped in a result struct.
func bindReturn(prologue *bytes.Buffer, method string, outputs []abi.Argument) ([]string, string) {
	// Generate the anonymous return list for when a struct is not needed/possible
	var (
		returns   = make([]string, 0, len(outputs)+1)
		anonymous = false
	)
	for _, ret := range outputs {
		returns = append(returns, bindType(ret.Type))
		if ret.Name == "" {
			anonymous = true
		}
	}
	if anonymous || len(returns) < 2 {
		returns = append(returns, "error")
		return returns, ""
	}
	// If the returns are named and numerous, wrap in a result struct
	wrapper, impl := bindReturnStruct(method, outputs)
	prologue.WriteString(impl + "\n")
	return []string{"*" + wrapper, "error"}, wrapper
}

// bindReturnStruct creates a Go structure with the specified fields to be used
// as the return type from a method call.
func bindReturnStruct(method string, returns []abi.Argument) (string, string) {
	fields := make([]string, 0, len(returns))
	for _, ret := range returns {
		fields = append(fields, fmt.Sprintf("%s %s", strings.ToUpper(ret.Name[:1])+ret.Name[1:], bindType(ret.Type)))
	}
	kind := fmt.Sprintf("%sResult", method)
	docs := fmt.Sprintf("// %s is the result of the %s invocation.", kind, method)

	return kind, fmt.Sprintf("%s\ntype %s struct {\n%s\n}", docs, kind, strings.Join(fields, "\n"))
}

// bindCallBody creates the Go code to declare a batch of return values, invoke
// an Ethereum method call with the requested parameters, parse the binary output
// into the return values and return them.
func bindCallBody(kind string, method string, params []string, returns []string) string {
	body := ""

	// Allocate memory for each of the return values
	rets := make([]string, 0, len(returns)-1)
	if len(returns) > 1 {
		body += "var ("
		for i, kind := range returns[:len(returns)-1] { // Omit the final error
			name := fmt.Sprintf("ret%d", i)

			rets = append(rets, name)
			body += fmt.Sprintf("%s = new(%s)\n", name, strings.TrimPrefix(kind, "*"))
		}
		body += ")\n"
	}
	// Assemble a single collector variable for the result ABI initialization
	result := strings.Join(rets, ",")
	if len(returns) > 2 {
		result = "[]interface{}{" + result + "}"
	}
	// Extract the parameter list into a flat variable name list
	inputs := make([]string, len(params))
	for i, param := range params {
		inputs[i] = strings.Split(param, " ")[0]
	}
	input := ""
	if len(inputs) > 0 {
		input = "," + strings.Join(inputs, ",")
	}
	// Request executing the contract call and return the results with the errors
	body += fmt.Sprintf("err := _%s.common.contract.Call(%s, \"%s\" %s)\n", kind, result, method, input)

	outs := make([]string, 0, len(returns))
	for i, ret := range returns[:len(returns)-1] { // Handle th final error separately
		if strings.HasPrefix(ret, "*") {
			outs = append(outs, rets[i])
		} else {
			outs = append(outs, "*"+rets[i])
		}
	}
	outs = append(outs, "err")

	body += fmt.Sprintf("return %s", strings.Join(outs, ","))

	return body
}

// bindTransactionBody creates the Go code to invoke an Ethereum transaction call
// with the requested parameters, and return the assembled transaction object.
func bindTransactionBody(kind string, method string, params []string) string {
	// Extract the parameter list into a flat variable name list
	inputs := make([]string, len(params)-1) // Omit the auth options
	for i, param := range params[1:] {
		inputs[i] = strings.Split(param, " ")[0]
	}
	input := ""
	if len(inputs) > 0 {
		input = "," + strings.Join(inputs, ",")
	}
	// Request executing the contract call and return the results with the errors
	return fmt.Sprintf("return _%s.common.contract.Transact(auth, \"%s\" %s)", kind, method, input)
}
