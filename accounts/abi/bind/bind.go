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
	"go/format"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

		// Extract the call and transact methods; events; and sort them alphabetically
		var (
			calls     = make(map[string]*tmplMethod)
			transacts = make(map[string]*tmplMethod)
			events    = make(map[string]*tmplEvent)
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
				calls[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original.Outputs)}
			} else {
				transacts[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: structured(original.Outputs)}
			}
		}
		for _, original := range evmABI.Events {
			// Skip anonymous events as they don't support explicit filtering
			if original.Anonymous {
				continue
			}
			// Normalize the event for capital cases and non-anonymous outputs
			normalized := original
			normalized.Name = methodNormalizer[lang](original.Name)

			normalized.Inputs = make([]abi.Argument, len(original.Inputs))
			copy(normalized.Inputs, original.Inputs)
			for j, input := range normalized.Inputs {
				// Indexed fields are input, non-indexed ones are outputs
				if input.Indexed {
					if input.Name == "" {
						normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
					}
				}
			}
			// Append the event to the accumulator list
			events[original.Name] = &tmplEvent{Original: original, Normalized: normalized}
		}
		contracts[types[i]] = &tmplContract{
			Type:        capitalise(types[i]),
			InputABI:    strings.Replace(strippedABI, "\"", "\\\"", -1),
			InputBin:    strings.TrimSpace(bytecodes[i]),
			Constructor: evmABI.Constructor,
			Calls:       calls,
			Transacts:   transacts,
			Events:      events,
		}
	}
	// Generate the contract template data content and render it
	data := &tmplData{
		Package:   pkg,
		Contracts: contracts,
	}
	buffer := new(bytes.Buffer)

	funcs := map[string]interface{}{
		"bindtype":      bindType[lang],
		"bindtopictype": bindTopicType[lang],
		"namedtype":     namedType[lang],
		"capitalise":    capitalise,
		"decapitalise":  decapitalise,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).Parse(tmplSource[lang]))
	if err := tmpl.Execute(buffer, data); err != nil {
		return "", err
	}
	// For Go bindings pass the code through gofmt to clean it up
	if lang == LangGo {
		code, err := format.Source(buffer.Bytes())
		if err != nil {
			return "", fmt.Errorf("%v\n%s", err, buffer)
		}
		return string(code), nil
	}
	// For all others just return as is for now
	return buffer.String(), nil
}

// bindType is a set of type binders that convert Solidity types to some supported
// programming language types.
var bindType = map[Lang]func(kind abi.Type) string{
	LangGo:   bindTypeGo,
	LangJava: bindTypeJava,
}

// Helper function for the binding generators.
// It reads the unmatched characters after the inner type-match,
//  (since the inner type is a prefix of the total type declaration),
//  looks for valid arrays (possibly a dynamic one) wrapping the inner type,
//  and returns the sizes of these arrays.
//
// Returned array sizes are in the same order as solidity signatures; inner array size first.
// Array sizes may also be "", indicating a dynamic array.
func wrapArray(stringKind string, innerLen int, innerMapping string) (string, []string) {
	remainder := stringKind[innerLen:]
	//find all the sizes
	matches := regexp.MustCompile(`\[(\d*)\]`).FindAllStringSubmatch(remainder, -1)
	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		//get group 1 from the regex match
		parts = append(parts, match[1])
	}
	return innerMapping, parts
}

// Translates the array sizes to a Go-lang declaration of a (nested) array of the inner type.
// Simply returns the inner type if arraySizes is empty.
func arrayBindingGo(inner string, arraySizes []string) string {
	out := ""
	//prepend all array sizes, from outer (end arraySizes) to inner (start arraySizes)
	for i := len(arraySizes) - 1; i >= 0; i-- {
		out += "[" + arraySizes[i] + "]"
	}
	out += inner
	return out
}

// bindTypeGo converts a Solidity type to a Go one. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. *big.Int).
func bindTypeGo(kind abi.Type) string {
	stringKind := kind.String()
	innerLen, innerMapping := bindUnnestedTypeGo(stringKind)
	return arrayBindingGo(wrapArray(stringKind, innerLen, innerMapping))
}

// The inner function of bindTypeGo, this finds the inner type of stringKind.
// (Or just the type itself if it is not an array or slice)
// The length of the matched part is returned, with the translated type.
func bindUnnestedTypeGo(stringKind string) (int, string) {

	switch {
	case strings.HasPrefix(stringKind, "address"):
		return len("address"), "common.Address"

	case strings.HasPrefix(stringKind, "bytes"):
		parts := regexp.MustCompile(`bytes([0-9]*)`).FindStringSubmatch(stringKind)
		return len(parts[0]), fmt.Sprintf("[%s]byte", parts[1])

	case strings.HasPrefix(stringKind, "int") || strings.HasPrefix(stringKind, "uint"):
		parts := regexp.MustCompile(`(u)?int([0-9]*)`).FindStringSubmatch(stringKind)
		switch parts[2] {
		case "8", "16", "32", "64":
			return len(parts[0]), fmt.Sprintf("%sint%s", parts[1], parts[2])
		}
		return len(parts[0]), "*big.Int"

	case strings.HasPrefix(stringKind, "bool"):
		return len("bool"), "bool"

	case strings.HasPrefix(stringKind, "string"):
		return len("string"), "string"

	default:
		return len(stringKind), stringKind
	}
}

// Translates the array sizes to a Java declaration of a (nested) array of the inner type.
// Simply returns the inner type if arraySizes is empty.
func arrayBindingJava(inner string, arraySizes []string) string {
	// Java array type declarations do not include the length.
	return inner + strings.Repeat("[]", len(arraySizes))
}

// bindTypeJava converts a Solidity type to a Java one. Since there is no clear mapping
// from all Solidity types to Java ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. BigDecimal).
func bindTypeJava(kind abi.Type) string {
	stringKind := kind.String()
	innerLen, innerMapping := bindUnnestedTypeJava(stringKind)
	return arrayBindingJava(wrapArray(stringKind, innerLen, innerMapping))
}

// The inner function of bindTypeJava, this finds the inner type of stringKind.
// (Or just the type itself if it is not an array or slice)
// The length of the matched part is returned, with the translated type.
func bindUnnestedTypeJava(stringKind string) (int, string) {

	switch {
	case strings.HasPrefix(stringKind, "address"):
		parts := regexp.MustCompile(`address(\[[0-9]*\])?`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return len(stringKind), stringKind
		}
		if parts[1] == "" {
			return len("address"), "Address"
		}
		return len(parts[0]), "Addresses"

	case strings.HasPrefix(stringKind, "bytes"):
		parts := regexp.MustCompile(`bytes([0-9]*)`).FindStringSubmatch(stringKind)
		if len(parts) != 2 {
			return len(stringKind), stringKind
		}
		return len(parts[0]), "byte[]"

	case strings.HasPrefix(stringKind, "int") || strings.HasPrefix(stringKind, "uint"):
		//Note that uint and int (without digits) are also matched,
		// these are size 256, and will translate to BigInt (the default).
		parts := regexp.MustCompile(`(u)?int([0-9]*)`).FindStringSubmatch(stringKind)
		if len(parts) != 3 {
			return len(stringKind), stringKind
		}

		namedSize := map[string]string{
			"8":  "byte",
			"16": "short",
			"32": "int",
			"64": "long",
		}[parts[2]]

		//default to BigInt
		if namedSize == "" {
			namedSize = "BigInt"
		}
		return len(parts[0]), namedSize

	case strings.HasPrefix(stringKind, "bool"):
		return len("bool"), "boolean"

	case strings.HasPrefix(stringKind, "string"):
		return len("string"), "String"

	default:
		return len(stringKind), stringKind
	}
}

// bindTopicType is a set of type binders that convert Solidity types to some
// supported programming language topic types.
var bindTopicType = map[Lang]func(kind abi.Type) string{
	LangGo:   bindTopicTypeGo,
	LangJava: bindTopicTypeJava,
}

// bindTypeGo converts a Solidity topic type to a Go one. It is almost the same
// funcionality as for simple types, but dynamic types get converted to hashes.
func bindTopicTypeGo(kind abi.Type) string {
	bound := bindTypeGo(kind)
	if bound == "string" || bound == "[]byte" {
		bound = "common.Hash"
	}
	return bound
}

// bindTypeGo converts a Solidity topic type to a Java one. It is almost the same
// funcionality as for simple types, but dynamic types get converted to hashes.
func bindTopicTypeJava(kind abi.Type) string {
	bound := bindTypeJava(kind)
	if bound == "String" || bound == "Bytes" {
		bound = "Hash"
	}
	return bound
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
	case "boolean":
		return "Bool"
	case "boolean[]":
		return "Bools"
	case "BigInt[]":
		return "BigInts"
	default:
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
	}
}

// methodNormalizer is a name transformer that modifies Solidity method names to
// conform to target language naming concentions.
var methodNormalizer = map[Lang]func(string) string{
	LangGo:   abi.ToCamelCase,
	LangJava: decapitalise,
}

// capitalise makes a camel-case string which starts with an upper case character.
func capitalise(input string) string {
	return abi.ToCamelCase(input)
}

// decapitalise makes a camel-case string which starts with a lower case character.
func decapitalise(input string) string {
	if len(input) == 0 {
		return input
	}

	goForm := abi.ToCamelCase(input)
	return strings.ToLower(goForm[:1]) + goForm[1:]
}

// structured checks whether a list of ABI data types has enough information to
// operate through a proper Go struct or if flat returns are needed.
func structured(args abi.Arguments) bool {
	if len(args) < 2 {
		return false
	}
	exists := make(map[string]bool)
	for _, out := range args {
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
