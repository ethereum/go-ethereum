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
// https://github.com/XinFinOrg/XDPoSChain/wiki/Native-DApps:-Go-bindings-to-Ethereum-contracts
package bind

import (
	"bytes"
	"fmt"
	"go/format"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// Lang is a target programming language selector to generate bindings for.
type Lang int

const (
	LangGo Lang = iota
)

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, lang Lang, libs map[string]string) (string, error) {
	// Process each individual contract requested binding
	contracts := make(map[string]*tmplContract)

	// Map used to flag each encountered library as such
	isLib := make(map[string]struct{})

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

		// Extract the call and transact methods; events, struct definitions; and sort them alphabetically
		var (
			calls     = make(map[string]*tmplMethod)
			transacts = make(map[string]*tmplMethod)
			events    = make(map[string]*tmplEvent)
			structs   = make(map[string]*tmplStruct)
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
				if _, exist := structs[input.Type.String()]; input.Type.T == abi.TupleTy && !exist {
					bindStructType[lang](input.Type, structs)
				}
			}
			normalized.Outputs = make([]abi.Argument, len(original.Outputs))
			copy(normalized.Outputs, original.Outputs)
			for j, output := range normalized.Outputs {
				if output.Name != "" {
					normalized.Outputs[j].Name = capitalise(output.Name)
				}
				if _, exist := structs[output.Type.String()]; output.Type.T == abi.TupleTy && !exist {
					bindStructType[lang](output.Type, structs)
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
					if _, exist := structs[input.Type.String()]; input.Type.T == abi.TupleTy && !exist {
						bindStructType[lang](input.Type, structs)
					}
				}
			}
			// Append the event to the accumulator list
			events[original.Name] = &tmplEvent{Original: original, Normalized: normalized}
		}

		contracts[types[i]] = &tmplContract{
			Type:        capitalise(types[i]),
			InputABI:    strings.ReplaceAll(strippedABI, "\"", "\\\""),
			InputBin:    strings.TrimPrefix(strings.TrimSpace(bytecodes[i]), "0x"),
			Constructor: evmABI.Constructor,
			Calls:       calls,
			Transacts:   transacts,
			Events:      events,
			Libraries:   make(map[string]string),
			Structs:     structs,
		}
		// Function 4-byte signatures are stored in the same sequence
		// as types, if available.
		if len(fsigs) > i {
			contracts[types[i]].FuncSigs = fsigs[i]
		}
		// Parse library references.
		for pattern, name := range libs {
			matched, err := regexp.Match("__\\$"+pattern+"\\$__", []byte(contracts[types[i]].InputBin))
			if err != nil {
				log.Error("Could not search for pattern", "pattern", pattern, "contract", contracts[types[i]], "err", err)
			}
			if matched {
				contracts[types[i]].Libraries[pattern] = name
				// keep track that this type is a library
				if _, ok := isLib[name]; !ok {
					isLib[name] = struct{}{}
				}
			}
		}
	}
	// Check if that type has already been identified as a library
	for i := 0; i < len(types); i++ {
		_, ok := isLib[types[i]]
		contracts[types[i]].Library = ok
	}
	// Generate the contract template data content and render it
	data := &tmplData{
		Package:   pkg,
		Contracts: contracts,
		Libraries: libs,
	}
	buffer := new(bytes.Buffer)

	funcs := map[string]interface{}{
		"bindtype":      bindType[lang],
		"bindtopictype": bindTopicType[lang],
		"namedtype":     namedType[lang],
		"formatmethod":  formatMethod,
		"formatevent":   formatEvent,
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
var bindType = map[Lang]func(kind abi.Type, structs map[string]*tmplStruct) string{
	LangGo: bindTypeGo,
}

// bindBasicTypeGo converts basic solidity types(except array, slice and tuple) to Go one.
func bindBasicTypeGo(kind abi.Type) string {
	switch kind.T {
	case abi.AddressTy:
		return "common.Address"
	case abi.IntTy, abi.UintTy:
		parts := regexp.MustCompile(`(u)?int([0-9]*)`).FindStringSubmatch(kind.String())
		switch parts[2] {
		case "8", "16", "32", "64":
			return fmt.Sprintf("%sint%s", parts[1], parts[2])
		}
		return "*big.Int"
	case abi.FixedBytesTy:
		return fmt.Sprintf("[%d]byte", kind.Size)
	case abi.BytesTy:
		return "[]byte"
	case abi.FunctionTy:
		return "[24]byte"
	default:
		// string, bool types
		return kind.String()
	}
}

// bindTypeGo converts solidity types to Go ones. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. BigDecimal).
func bindTypeGo(kind abi.Type, structs map[string]*tmplStruct) string {
	switch kind.T {
	case abi.TupleTy:
		return structs[kind.String()].Name
	case abi.ArrayTy:
		return fmt.Sprintf("[%d]", kind.Size) + bindTypeGo(*kind.Elem, structs)
	case abi.SliceTy:
		return "[]" + bindTypeGo(*kind.Elem, structs)
	default:
		return bindBasicTypeGo(kind)
	}
}

// bindTopicType is a set of type binders that convert Solidity types to some
// supported programming language topic types.
var bindTopicType = map[Lang]func(kind abi.Type, structs map[string]*tmplStruct) string{
	LangGo: bindTopicTypeGo,
}

// bindTopicTypeGo converts a Solidity topic type to a Go one. It is almost the same
// funcionality as for simple types, but dynamic types get converted to hashes.
func bindTopicTypeGo(kind abi.Type, structs map[string]*tmplStruct) string {
	bound := bindTypeGo(kind, structs)
	if bound == "string" || bound == "[]byte" {
		bound = "common.Hash"
	}
	return bound
}

// bindStructType is a set of type binders that convert Solidity tuple types to some supported
// programming language struct definition.
var bindStructType = map[Lang]func(kind abi.Type, structs map[string]*tmplStruct) string{
	LangGo: bindStructTypeGo,
}

// bindStructTypeGo converts a Solidity tuple type to a Go one and records the mapping
// in the given map.
// Notably, this function will resolve and record nested struct recursively.
func bindStructTypeGo(kind abi.Type, structs map[string]*tmplStruct) string {
	switch kind.T {
	case abi.TupleTy:
		if s, exist := structs[kind.String()]; exist {
			return s.Name
		}
		var fields []*tmplField
		for i, elem := range kind.TupleElems {
			field := bindStructTypeGo(*elem, structs)
			fields = append(fields, &tmplField{Type: field, Name: capitalise(kind.TupleRawNames[i]), SolKind: *elem})
		}
		name := fmt.Sprintf("Struct%d", len(structs))
		structs[kind.String()] = &tmplStruct{
			Name:   name,
			Fields: fields,
		}
		return name
	case abi.ArrayTy:
		return fmt.Sprintf("[%d]", kind.Size) + bindStructTypeGo(*kind.Elem, structs)
	case abi.SliceTy:
		return "[]" + bindStructTypeGo(*kind.Elem, structs)
	default:
		return bindBasicTypeGo(kind)
	}
}

// namedType is a set of functions that transform language specific types to
// named versions that my be used inside method names.
var namedType = map[Lang]func(string, abi.Type) string{
	LangGo: func(string, abi.Type) string { panic("this shouldn't be needed") },
}

// methodNormalizer is a name transformer that modifies Solidity method names to
// conform to target language naming concentions.
var methodNormalizer = map[Lang]func(string) string{
	LangGo: abi.ToCamelCase,
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

// resolveArgName converts a raw argument representation into a user friendly format.
func resolveArgName(arg abi.Argument, structs map[string]*tmplStruct) string {
	var (
		prefix   string
		embedded string
		typ      = &arg.Type
	)
loop:
	for {
		switch typ.T {
		case abi.SliceTy:
			prefix += "[]"
		case abi.ArrayTy:
			prefix += fmt.Sprintf("[%d]", typ.Size)
		default:
			embedded = typ.String()
			break loop
		}
		typ = typ.Elem
	}
	if s, exist := structs[embedded]; exist {
		return prefix + s.Name
	} else {
		return arg.Type.String()
	}
}

// formatMethod transforms raw method representation into a user friendly one.
func formatMethod(method abi.Method, structs map[string]*tmplStruct) string {
	inputs := make([]string, len(method.Inputs))
	for i, input := range method.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", resolveArgName(input, structs), input.Name)
	}
	outputs := make([]string, len(method.Outputs))
	for i, output := range method.Outputs {
		outputs[i] = resolveArgName(output, structs)
		if len(output.Name) > 0 {
			outputs[i] += fmt.Sprintf(" %v", output.Name)
		}
	}
	constant := ""
	if method.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", method.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

// formatEvent transforms raw event representation into a user friendly one.
func formatEvent(event abi.Event, structs map[string]*tmplStruct) string {
	inputs := make([]string, len(event.Inputs))
	for i, input := range event.Inputs {
		if input.Indexed {
			inputs[i] = fmt.Sprintf("%v indexed %v", resolveArgName(input, structs), input.Name)
		} else {
			inputs[i] = fmt.Sprintf("%v %v", resolveArgName(input, structs), input.Name)
		}
	}
	return fmt.Sprintf("event %v(%v)", event.Name, strings.Join(inputs, ", "))
}
