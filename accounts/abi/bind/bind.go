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
	"github.com/ethereum/go-ethereum/log"
)

func isKeyWord(arg string) bool {
	switch arg {
	case "break":
	case "case":
	case "chan":
	case "const":
	case "continue":
	case "default":
	case "defer":
	case "else":
	case "fallthrough":
	case "for":
	case "func":
	case "go":
	case "goto":
	case "if":
	case "import":
	case "interface":
	case "iota":
	case "map":
	case "make":
	case "new":
	case "package":
	case "range":
	case "return":
	case "select":
	case "struct":
	case "switch":
	case "type":
	case "var":
	default:
		return false
	}

	return true
}

func add(val1, val2 int) int {
	return val1 + val2
}

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention as opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {
	data, err := bind(types, abis, bytecodes, fsigs, pkg, libs, aliases)
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)

	funcs := map[string]interface{}{
		"bindtype":      bindType,
		"bindtopictype": bindTopicType,
		"capitalise":    capitalise,
		"decapitalise":  decapitalise,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).Parse(tmplSource))
	if err := tmpl.Execute(buffer, data); err != nil {
		return "", err
	}
	// Pass the code through gofmt to clean it up
	code, err := format.Source(buffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("%v\n%s", err, buffer)
	}
	return string(code), nil
}

func BindV2(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {
	data, err := bind(types, abis, bytecodes, fsigs, pkg, libs, aliases)

	if err != nil {
		return "", err
	}
	for _, c := range data.Contracts {
		// We want pack/unpack methods for all existing methods.
		for name, t := range c.Transacts {
			c.Calls[name] = t
		}
		c.Transacts = nil

		// Make sure we return one argument. If multiple exist
		// merge them into a struct.
		for _, call := range c.Calls {
			if call.Structured {
				continue
			}
			if len(call.Normalized.Outputs) < 2 {
				continue
			}
			// Build up dictionary of existing arg names.
			keys := make(map[string]struct{})
			for _, o := range call.Normalized.Outputs {
				if o.Name != "" {
					keys[strings.ToLower(o.Name)] = struct{}{}
				}
			}
			// Assign names to anonymous fields.
			for i, o := range call.Normalized.Outputs {
				if o.Name != "" {
					continue
				}
				o.Name = capitalise(abi.ResolveNameConflict("arg", func(name string) bool { _, ok := keys[name]; return ok }))
				call.Normalized.Outputs[i] = o
				keys[strings.ToLower(o.Name)] = struct{}{}
			}
			call.Structured = true
		}
	}
	buffer := new(bytes.Buffer)
	funcs := map[string]interface{}{
		"bindtype":      bindType,
		"bindtopictype": bindTopicType,
		"capitalise":    capitalise,
		"decapitalise":  decapitalise,
		"add":           add,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).Parse(tmplSourceV2))
	if err := tmpl.Execute(buffer, data); err != nil {
		return "", err
	}
	// Pass the code through gofmt to clean it up
	code, err := format.Source(buffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("%v\n%s", err, buffer)
	}
	return string(code), nil
}

func bind(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, libs map[string]string, aliases map[string]string) (*tmplData, error) {
	var (
		// contracts is the map of each individual contract requested binding
		contracts = make(map[string]*tmplContract)

		// structs is the map of all redeclared structs shared by passed contracts.
		structs = make(map[string]*tmplStruct)

		// isLib is the map used to flag each encountered library as such
		isLib = make(map[string]struct{})
	)
	for i := 0; i < len(types); i++ {
		// Parse the actual ABI to generate the binding for
		evmABI, err := abi.JSON(strings.NewReader(abis[i]))
		if err != nil {
			return nil, err
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
			errors    = make(map[string]*tmplError)
			fallback  *tmplMethod
			receive   *tmplMethod

			// identifiers are used to detect duplicated identifiers of functions
			// and events. For all calls, transacts and events, abigen will generate
			// corresponding bindings. However we have to ensure there is no
			// identifier collisions in the bindings of these categories.
			callIdentifiers     = make(map[string]bool)
			transactIdentifiers = make(map[string]bool)
			eventIdentifiers    = make(map[string]bool)
		)

		for _, input := range evmABI.Constructor.Inputs {
			if hasStruct(input.Type) {
				bindStructType(input.Type, structs)
			}
		}

		for _, original := range evmABI.Methods {
			// Normalize the method for capital cases and non-anonymous inputs/outputs
			normalized := original
			normalizedName := methodNormalizer(alias(aliases, original.Name))
			// Ensure there is no duplicated identifier
			var identifiers = callIdentifiers
			if !original.IsConstant() {
				identifiers = transactIdentifiers
			}
			// Name shouldn't start with a digit. It will make the generated code invalid.
			if len(normalizedName) > 0 && unicode.IsDigit(rune(normalizedName[0])) {
				normalizedName = fmt.Sprintf("M%s", normalizedName)
				normalizedName = abi.ResolveNameConflict(normalizedName, func(name string) bool {
					_, ok := identifiers[name]
					return ok
				})
			}
			if identifiers[normalizedName] {
				return nil, fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
			}
			identifiers[normalizedName] = true

			normalized.Name = normalizedName
			normalized.Inputs = make([]abi.Argument, len(original.Inputs))
			copy(normalized.Inputs, original.Inputs)
			for j, input := range normalized.Inputs {
				if input.Name == "" || isKeyWord(input.Name) {
					normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
				}
				if hasStruct(input.Type) {
					bindStructType(input.Type, structs)
				}
			}
			normalized.Outputs = make([]abi.Argument, len(original.Outputs))
			copy(normalized.Outputs, original.Outputs)
			for j, output := range normalized.Outputs {
				if output.Name != "" {
					normalized.Outputs[j].Name = capitalise(output.Name)
				}
				if hasStruct(output.Type) {
					bindStructType(output.Type, structs)
				}
			}
			// Append the methods to the call or transact lists
			if original.IsConstant() {
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

			// Ensure there is no duplicated identifier
			normalizedName := methodNormalizer(alias(aliases, original.Name))
			// Name shouldn't start with a digit. It will make the generated code invalid.
			if len(normalizedName) > 0 && unicode.IsDigit(rune(normalizedName[0])) {
				normalizedName = fmt.Sprintf("E%s", normalizedName)
				normalizedName = abi.ResolveNameConflict(normalizedName, func(name string) bool {
					_, ok := eventIdentifiers[name]
					return ok
				})
			}
			if eventIdentifiers[normalizedName] {
				return nil, fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
			}
			eventIdentifiers[normalizedName] = true
			normalized.Name = normalizedName

			used := make(map[string]bool)
			normalized.Inputs = make([]abi.Argument, len(original.Inputs))
			copy(normalized.Inputs, original.Inputs)
			for j, input := range normalized.Inputs {
				if input.Name == "" || isKeyWord(input.Name) {
					normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
				}
				// Event is a bit special, we need to define event struct in binding,
				// ensure there is no camel-case-style name conflict.
				for index := 0; ; index++ {
					if !used[capitalise(normalized.Inputs[j].Name)] {
						used[capitalise(normalized.Inputs[j].Name)] = true
						break
					}
					normalized.Inputs[j].Name = fmt.Sprintf("%s%d", normalized.Inputs[j].Name, index)
				}
				if hasStruct(input.Type) {
					bindStructType(input.Type, structs)
				}
			}
			// Append the event to the accumulator list
			events[original.Name] = &tmplEvent{Original: original, Normalized: normalized}
		}
		for _, original := range evmABI.Errors {
			// TODO: I copied this from events (above in this function).  I think it should be correct but not totally sure
			// even if it is correct, should consider deduplicating this into its own function.

			// Normalize the error for capital cases and non-anonymous outputs
			normalized := original

			// Ensure there is no duplicated identifier
			normalizedName := methodNormalizer(alias(aliases, original.Name))
			// Name shouldn't start with a digit. It will make the generated code invalid.
			if len(normalizedName) > 0 && unicode.IsDigit(rune(normalizedName[0])) {
				normalizedName = fmt.Sprintf("E%s", normalizedName)
				normalizedName = abi.ResolveNameConflict(normalizedName, func(name string) bool {
					_, ok := eventIdentifiers[name]
					return ok
				})
			}
			if eventIdentifiers[normalizedName] {
				return nil, fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
			}
			eventIdentifiers[normalizedName] = true
			normalized.Name = normalizedName

			used := make(map[string]bool)
			normalized.Inputs = make([]abi.Argument, len(original.Inputs))
			copy(normalized.Inputs, original.Inputs)
			for j, input := range normalized.Inputs {
				if input.Name == "" || isKeyWord(input.Name) {
					normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
				}
				// Event is a bit special, we need to define event struct in binding,
				// ensure there is no camel-case-style name conflict.
				for index := 0; ; index++ {
					if !used[capitalise(normalized.Inputs[j].Name)] {
						used[capitalise(normalized.Inputs[j].Name)] = true
						break
					}
					normalized.Inputs[j].Name = fmt.Sprintf("%s%d", normalized.Inputs[j].Name, index)
				}
				if hasStruct(input.Type) {
					bindStructType(input.Type, structs)
				}
			}
			errors[original.Name] = &tmplError{Original: original, Normalized: normalized}
		}
		// Add two special fallback functions if they exist
		if evmABI.HasFallback() {
			fallback = &tmplMethod{Original: evmABI.Fallback}
		}
		if evmABI.HasReceive() {
			receive = &tmplMethod{Original: evmABI.Receive}
		}

		contracts[types[i]] = &tmplContract{
			Type:         capitalise(types[i]),
			InputABI:     strings.ReplaceAll(strippedABI, "\"", "\\\""),
			InputBin:     strings.TrimPrefix(strings.TrimSpace(bytecodes[i]), "0x"),
			Constructor:  evmABI.Constructor,
			Calls:        calls,
			Transacts:    transacts,
			Fallback:     fallback,
			Receive:      receive,
			Events:       events,
			Libraries:    make(map[string]string),
			AllLibraries: make(map[string]string),
			Errors:       errors,
		}

		// Function 4-byte signatures are stored in the same sequence
		// as types, if available.
		if len(fsigs) > i {
			contracts[types[i]].FuncSigs = fsigs[i]
		}
		// Parse library references.
		for pattern, name := range libs {
			matched, err := regexp.MatchString("__\\$"+pattern+"\\$__", contracts[types[i]].InputBin)
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

	// compute the full set of libraries that each contract depends on.
	for _, contract := range contracts {
		if contract.Library {
			continue
		}
		// recursively traverse the library dependency graph
		// of the contract, flattening it into a set.
		//
		// For abigenv2, we do not generate contract deploy
		// methods (which in v1 recursively deploy their
		// library dependencies).  So, the entire set of
		// library dependencies is required, and we will
		// determine the order to deploy and link them at
		// runtime.
		var findDeps func(contract *tmplContract) map[string]struct{}
		findDeps = func(contract *tmplContract) map[string]struct{} {
			// 1) match all libraries that this contract depends on
			re, err := regexp.Compile(`__\\$([a-f0-9]+)\\$__`)
			if err != nil {
				panic(err)
			}
			libBin := contracts[contract.Type].InputBin
			matches := re.FindAllStringSubmatch(libBin, -1)
			result := make(map[string]struct{})

			// 2) recurse, gathering nested library dependencies
			for _, match := range matches {
				pattern := match[1]
				result[pattern] = struct{}{}
				depContract := contracts[libs[pattern]]
				for subPattern, _ := range findDeps(depContract) {
					result[subPattern] = struct{}{}
				}
			}
			return result
		}
		// take the set of library patterns, convert it to a map of type -> pattern
		deps := findDeps(contract)
		contract.AllLibraries = make(map[string]string)
		for contractPattern, _ := range deps {
			contractType := libs[contractPattern]
			contract.AllLibraries[contractType] = contractPattern
		}
	}

	// map of contract name -> pattern
	invertedLibs := make(map[string]string)
	// assume that this is invertible/onto because I assume library names are unique now
	// TODO: check that they've been sanitized at this point.
	for pattern, name := range libs {
		invertedLibs[name] = pattern
	}

	// Generate the contract template data content and render it
	data := &tmplData{
		Package:      pkg,
		Contracts:    contracts,
		Libraries:    libs,
		InvertedLibs: invertedLibs,
		Structs:      structs,
	}
	return data, nil
}

// bindBasicType converts basic solidity types(except array, slice and tuple) to Go ones.
func bindBasicType(kind abi.Type) string {
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

// bindType converts solidity types to Go ones. Since there is no clear mapping
// from all Solidity types to Go ones (e.g. uint17), those that cannot be exactly
// mapped will use an upscaled type (e.g. BigDecimal).
func bindType(kind abi.Type, structs map[string]*tmplStruct) string {
	switch kind.T {
	case abi.TupleTy:
		return structs[kind.TupleRawName+kind.String()].Name
	case abi.ArrayTy:
		return fmt.Sprintf("[%d]", kind.Size) + bindType(*kind.Elem, structs)
	case abi.SliceTy:
		return "[]" + bindType(*kind.Elem, structs)
	default:
		return bindBasicType(kind)
	}
}

// bindTopicType converts a Solidity topic type to a Go one. It is almost the same
// functionality as for simple types, but dynamic types get converted to hashes.
func bindTopicType(kind abi.Type, structs map[string]*tmplStruct) string {
	bound := bindType(kind, structs)

	// todo(rjl493456442) according solidity documentation, indexed event
	// parameters that are not value types i.e. arrays and structs are not
	// stored directly but instead a keccak256-hash of an encoding is stored.
	//
	// We only convert strings and bytes to hash, still need to deal with
	// array(both fixed-size and dynamic-size) and struct.
	if bound == "string" || bound == "[]byte" {
		bound = "common.Hash"
	}
	return bound
}

// bindStructType converts a Solidity tuple type to a Go one and records the mapping
// in the given map.
// Notably, this function will resolve and record nested struct recursively.
func bindStructType(kind abi.Type, structs map[string]*tmplStruct) string {
	switch kind.T {
	case abi.TupleTy:
		// We compose a raw struct name and a canonical parameter expression
		// together here. The reason is before solidity v0.5.11, kind.TupleRawName
		// is empty, so we use canonical parameter expression to distinguish
		// different struct definition. From the consideration of backward
		// compatibility, we concat these two together so that if kind.TupleRawName
		// is not empty, it can have unique id.
		id := kind.TupleRawName + kind.String()
		if s, exist := structs[id]; exist {
			return s.Name
		}
		var (
			names  = make(map[string]bool)
			fields []*tmplField
		)
		for i, elem := range kind.TupleElems {
			name := capitalise(kind.TupleRawNames[i])
			name = abi.ResolveNameConflict(name, func(s string) bool { return names[s] })
			names[name] = true
			fields = append(fields, &tmplField{Type: bindStructType(*elem, structs), Name: name, SolKind: *elem})
		}
		name := kind.TupleRawName
		if name == "" {
			name = fmt.Sprintf("Struct%d", len(structs))
		}
		name = capitalise(name)

		structs[id] = &tmplStruct{
			Name:   name,
			Fields: fields,
		}
		return name
	case abi.ArrayTy:
		return fmt.Sprintf("[%d]", kind.Size) + bindStructType(*kind.Elem, structs)
	case abi.SliceTy:
		return "[]" + bindStructType(*kind.Elem, structs)
	default:
		return bindBasicType(kind)
	}
}

// alias returns an alias of the given string based on the aliasing rules
// or returns itself if no rule is matched.
func alias(aliases map[string]string, n string) string {
	if alias, exist := aliases[n]; exist {
		return alias
	}
	return n
}

// methodNormalizer is a name transformer that modifies Solidity method names to
// conform to Go naming conventions.
var methodNormalizer = abi.ToCamelCase

// capitalise makes a camel-case string which starts with an upper case character.
var capitalise = abi.ToCamelCase

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

// hasStruct returns an indicator whether the given type is struct, struct slice
// or struct array.
func hasStruct(t abi.Type) bool {
	switch t.T {
	case abi.SliceTy:
		return hasStruct(*t.Elem)
	case abi.ArrayTy:
		return hasStruct(*t.Elem)
	case abi.TupleTy:
		return true
	default:
		return false
	}
}
