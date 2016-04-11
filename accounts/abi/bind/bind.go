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
//
// Detailed usage document and tutorial available on the go-ethereum Wiki page:
// https://github.com/ethereum/go-ethereum/wiki/Native-DApps:-Go-bindings-to-Ethereum-contracts
package bind

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/tools/imports"
)

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(types []string, abis []string, bytecodes []string, pkg string) (string, error) {
	// Process each individual contract requested binding
	contracts := make(map[string]*tmplContract)

	for i := 0; i < len(types); i++ {
		// Parse the actual ABI to generate the binding for
		evmABI, err := abi.JSON(strings.NewReader(abis[i]))
		if err != nil {
			return "", err
		}
		// Strip any whitespace from the JSON ABI
		strippedABI := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, abis[i])

		// Extract the call and transact methods, and sort them alphabetically
		var (
			calls     = make(map[string]*tmplMethod)
			transacts = make(map[string]*tmplMethod)
		)
		for _, original := range evmABI.Methods {
			// Normalize the method for capital cases and non-anonymous inputs/outputs
			normalized := original
			normalized.Name = capitalise(original.Name)

			normalized.Inputs = make([]abi.Argument, len(original.Inputs))
			copy(normalized.Inputs, original.Inputs)
			for j, input := range normalized.Inputs {
				if input.Name == "" {
					normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
				}
			}
			normalized.Outputs = make([]abi.Argument, len(original.Outputs))
			copy(normalized.Outputs, original.Outputs)
			for j, output := range normalized.Outputs {
				if output.Name != "" {
					normalized.Outputs[j].Name = capitalise(output.Name)
				}
			}
			// Append the methos to the call or transact lists
			if original.Const {
				calls[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original)}
			} else {
				transacts[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original)}
			}
		}
		contracts[types[i]] = &tmplContract{
			Type:        capitalise(types[i]),
			InputABI:    strippedABI,
			InputBin:    strings.TrimSpace(bytecodes[i]),
			Constructor: evmABI.Constructor,
			Calls:       calls,
			Transacts:   transacts,
		}
	}
	// Generate the contract template data content and render it
	data := &tmplData{
		Package:   pkg,
		Contracts: contracts,
	}
	buffer := new(bytes.Buffer)

	funcs := map[string]interface{}{
		"bindtype": bindType,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).Parse(tmplSource))
	if err := tmpl.Execute(buffer, data); err != nil {
		return "", err
	}
	// Pass the code through goimports to clean it up and double check
	code, err := imports.Process("", buffer.Bytes(), nil)
	if err != nil {
		return "", fmt.Errorf("%v\n%s", err, buffer)
	}
	return string(code), nil
}

// bindType converts a Solidity type to a Go one. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. *big.Int).
func bindType(kind abi.Type) string {
	stringKind := kind.String()

	switch {
	case strings.HasPrefix(stringKind, "address"):
		parts := regexp.MustCompile("address(\\[[0-9]*\\])?").FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return stringKind
		}
		return fmt.Sprintf("%scommon.Address", parts[1])

	case strings.HasPrefix(stringKind, "bytes"):
		parts := regexp.MustCompile("bytes([0-9]*)(\\[[0-9]*\\])?").FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return stringKind
		}
		return fmt.Sprintf("%s[%s]byte", parts[2], parts[1])

	case strings.HasPrefix(stringKind, "int") || strings.HasPrefix(stringKind, "uint"):
		parts := regexp.MustCompile("(u)?int([0-9]*)(\\[[0-9]*\\])?").FindStringSubmatch(stringKind)
		if len(parts) != 4 {
			return stringKind
		}
		switch parts[2] {
		case "8", "16", "32", "64":
			return fmt.Sprintf("%s%sint%s", parts[3], parts[1], parts[2])
		}
		return fmt.Sprintf("%s*big.Int", parts[3])

	case strings.HasPrefix(stringKind, "bool") || strings.HasPrefix(stringKind, "string"):
		parts := regexp.MustCompile("([a-z]+)(\\[[0-9]*\\])?").FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return stringKind
		}
		return fmt.Sprintf("%s%s", parts[2], parts[1])

	default:
		return stringKind
	}
}

// capitalise makes the first character of a string upper case.
func capitalise(input string) string {
	return strings.ToUpper(input[:1]) + input[1:]
}

// structured checks whether a method has enough information to return a proper
// Go struct ot if flat returns are needed.
func structured(method abi.Method) bool {
	if len(method.Outputs) < 2 {
		return false
	}
	for _, out := range method.Outputs {
		if out.Name == "" {
			return false
		}
	}
	return true
}
