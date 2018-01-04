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

// Lang is a target programming language selector to generate bindings for.
type Lang int

const (
	LangGo Lang = iota
	LangJava
	LangObjC
)

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(types []string, abis []string, bytecodes []string, pkg string, lang Lang) (string, error) {
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
			normalized.Name = methodNormalizer[lang](original.Name)

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
			// Append the methods to the call or transact lists
			if original.Const {
				calls[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original)}
			} else {
				transacts[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original)}
			}
		}
		contracts[types[i]] = &tmplContract{
			Type:        capitalise(types[i]),
			InputABI:    strings.Replace(strippedABI, "\"", "\\\"", -1),
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
		"bindtype":     bindType[lang],
		"namedtype":    namedType[lang],
		"capitalise":   capitalise,
		"decapitalise": decapitalise,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).Parse(tmplSource[lang]))
	if err := tmpl.Execute(buffer, data); err != nil {
		return "", err
	}
	// For Go bindings pass the code through goimports to clean it up and double check
	if lang == LangGo {
		code, err := imports.Process(".", buffer.Bytes(), nil)
		if err != nil {
			return "", fmt.Errorf("%v\n%s", err, buffer)
		}
		return string(code), nil
	}
	// For all others just return as is for now
	return buffer.String(), nil
}

// bindType is a set of type binders that convert Solidity types to some supported
// programming language.
var bindType = map[Lang]func(kind abi.Type) string{
	LangGo:   bindTypeGo,
	LangJava: bindTypeJava,
}

// bindTypeGo converts a Solidity type to a Go one. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. *big.Int).
func bindTypeGo(kind abi.Type) string {
	stringKind := kind.String()

	switch {
	case strings.HasPrefix(stringKind, "address"):
		parts := regexp.MustCompile(`address(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return stringKind
		}
		return fmt.Sprintf("%scommon.Address", parts[1])

	case strings.HasPrefix(stringKind, "bytes"):
		parts := regexp.MustCompile(`bytes([0-9]*)(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return stringKind
		}
		return fmt.Sprintf("%s[%s]byte", parts[2], parts[1])

	case strings.HasPrefix(stringKind, "int") || strings.HasPrefix(stringKind, "uint"):
		parts := regexp.MustCompile(`(u)?int([0-9]*)(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 4 {
			return stringKind
		}
		switch parts[2] {
		case "8", "16", "32", "64":
			return fmt.Sprintf("%s%sint%s", parts[3], parts[1], parts[2])
		}
		return fmt.Sprintf("%s*big.Int", parts[3])

	case strings.HasPrefix(stringKind, "bool") || strings.HasPrefix(stringKind, "string"):
		parts := regexp.MustCompile(`([a-z]+)(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return stringKind
		}
		return fmt.Sprintf("%s%s", parts[2], parts[1])

	default:
		return stringKind
	}
}

// bindTypeJava converts a Solidity type to a Java one. Since there is no clear mapping
// from all Solidity types to Java ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. BigDecimal).
func bindTypeJava(kind abi.Type) string {
	stringKind := kind.String()

	switch {
	case strings.HasPrefix(stringKind, "address"):
		parts := regexp.MustCompile(`address(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return stringKind
		}
		if parts[1] == "" {
			return fmt.Sprintf("Address")
		}
		return fmt.Sprintf("Addresses")

	case strings.HasPrefix(stringKind, "bytes"):
		parts := regexp.MustCompile(`bytes([0-9]*)(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return stringKind
		}
		if parts[2] != "" {
			return "byte[][]"
		}
		return "byte[]"

	case strings.HasPrefix(stringKind, "int") || strings.HasPrefix(stringKind, "uint"):
		parts := regexp.MustCompile(`(u)?int([0-9]*)(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 4 {
			return stringKind
		}
		switch parts[2] {
		case "8", "16", "32", "64":
			if parts[1] == "" {
				if parts[3] == "" {
					return fmt.Sprintf("int%s", parts[2])
				}
				return fmt.Sprintf("int%s[]", parts[2])
			}
		}
		if parts[3] == "" {
			return fmt.Sprintf("BigInt")
		}
		return fmt.Sprintf("BigInts")

	case strings.HasPrefix(stringKind, "bool"):
		parts := regexp.MustCompile(`bool(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return stringKind
		}
		if parts[1] == "" {
			return fmt.Sprintf("bool")
		}
		return fmt.Sprintf("bool[]")

	case strings.HasPrefix(stringKind, "string"):
		parts := regexp.MustCompile(`string(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return stringKind
		}
		if parts[1] == "" {
			return fmt.Sprintf("String")
		}
		return fmt.Sprintf("String[]")

	default:
		return stringKind
	}
}

// namedType is a set of functions that transform language specific types to
// named versions that my be used inside method names.
var namedType = map[Lang]func(string, abi.Type) string{
	LangGo:   func(string, abi.Type) string { panic("this shouldn't be needed") },
	LangJava: namedTypeJava,
}

// namedTypeJava converts some primitive data types to named variants that can
// be used as parts of method names.
func namedTypeJava(javaKind string, solKind abi.Type) string {
	switch javaKind {
	case "byte[]":
		return "Binary"
	case "byte[][]":
		return "Binaries"
	case "string":
		return "String"
	case "string[]":
		return "Strings"
	case "bool":
		return "Bool"
	case "bool[]":
		return "Bools"
	case "BigInt":
		parts := regexp.MustCompile(`(u)?int([0-9]*)(\[[0-9]*\])?`).FindStringSubmatch(solKind.String())
		if len(parts) != 4 {
			return javaKind
		}
		switch parts[2] {
		case "8", "16", "32", "64":
			if parts[3] == "" {
				return capitalise(fmt.Sprintf("%sint%s", parts[1], parts[2]))
			}
			return capitalise(fmt.Sprintf("%sint%ss", parts[1], parts[2]))

		default:
			return javaKind
		}
	default:
		return javaKind
	}
}

// methodNormalizer is a name transformer that modifies Solidity method names to
// conform to target language naming concentions.
var methodNormalizer = map[Lang]func(string) string{
	LangGo:   capitalise,
	LangJava: decapitalise,
}

// capitalise makes the first character of a string upper case, also removing any
// prefixing underscores from the variable names.
func capitalise(input string) string {
	for len(input) > 0 && input[0] == '_' {
		input = input[1:]
	}
	if len(input) == 0 {
		return ""
	}
	return strings.ToUpper(input[:1]) + input[1:]
}

// decapitalise makes the first character of a string lower case.
func decapitalise(input string) string {
	return strings.ToLower(input[:1]) + input[1:]
}

// structured checks whether a method has enough information to return a proper
// Go struct or if flat returns are needed.
func structured(method abi.Method) bool {
	if len(method.Outputs) < 2 {
		return false
	}
	exists := make(map[string]bool)
	for _, out := range method.Outputs {
		// If the name is anonymous, we can't organize into a struct
		if out.Name == "" {
			return false
		}
		// If the field name is empty when normalized or collides (var, Var, _var, _Var),
		// we can't organize into a struct
		field := capitalise(out.Name)
		if field == "" || exists[field] {
			return false
		}
		exists[field] = true
	}
	return true
}
