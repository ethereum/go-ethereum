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

// Bind generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention as opposed to having to
// manually maintain hard coded strings that break on runtime.
func Bind(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {
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
				return "", fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
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
				return "", fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
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
				return "", fmt.Errorf("duplicated identifier \"%s\"(normalized \"%s\"), use --alias for renaming", original.Name, normalizedName)
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

	// Generate the contract template data content and render it
	data := &tmplData{
		Package:   pkg,
		Contracts: contracts,
		Libraries: libs,
		Structs:   structs,
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

type binder struct {
	// contracts is the map of each individual contract requested binding
	contracts map[string]*tmplContractV2

	// structs is the map of all redeclared structs shared by passed contracts.
	structs map[string]*tmplStruct

	aliases map[string]string
}

func (b *contractBinder) registerIdentifier(identifiers map[string]bool, original string) (normalized string, err error) {
	normalized = alias(b.binder.aliases, methodNormalizer(original))
	// Name shouldn't start with a digit. It will make the generated code invalid.
	if len(normalized) > 0 && unicode.IsDigit(rune(normalized[0])) {
		normalized = fmt.Sprintf("E%s", normalized)
		normalized = abi.ResolveNameConflict(normalized, func(name string) bool {
			_, ok := identifiers[name]
			return ok
		})
	}

	if _, ok := identifiers[normalized]; ok {
		return "", fmt.Errorf("duplicate symbol '%s'", normalized)
	}
	identifiers[normalized] = true
	return normalized, nil
}

func (b *contractBinder) RegisterCallIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.callIdentifiers, id)
}

func (b *contractBinder) RegisterEventIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.eventIdentifiers, id)
}

func (b *contractBinder) RegisterErrorIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.errorIdentifiers, id)
}

func (b *binder) BindStructType(typ abi.Type) {
	bindStructType(typ, b.structs)
}

type contractBinder struct {
	binder *binder
	calls  map[string]*tmplMethod
	events map[string]*tmplEvent
	errors map[string]*tmplError

	callIdentifiers  map[string]bool
	eventIdentifiers map[string]bool
	errorIdentifiers map[string]bool
}

func (cb *contractBinder) bindMethod(original abi.Method) error {
	normalized := original
	normalizedName, err := cb.RegisterCallIdentifier(original.Name)
	if err != nil {
		return err
	}

	normalized.Name = normalizedName
	normalized.Inputs = make([]abi.Argument, len(original.Inputs))
	copy(normalized.Inputs, original.Inputs)
	for j, input := range normalized.Inputs {
		if input.Name == "" || isKeyWord(input.Name) {
			normalized.Inputs[j].Name = fmt.Sprintf("arg%d", j)
		}
		if hasStruct(input.Type) {
			cb.binder.BindStructType(input.Type)
		}
	}
	normalized.Outputs = make([]abi.Argument, len(original.Outputs))
	copy(normalized.Outputs, original.Outputs)
	for j, output := range normalized.Outputs {
		if output.Name != "" {
			normalized.Outputs[j].Name = capitalise(output.Name)
		}
		if hasStruct(output.Type) {
			cb.binder.BindStructType(output.Type)
		}
	}
	isStructured := structured(original.Outputs)
	// if the call returns multiple values, coallesce them into a struct
	if len(normalized.Outputs) > 1 {
		// Build up dictionary of existing arg names.
		keys := make(map[string]struct{})
		for _, o := range normalized.Outputs {
			if o.Name != "" {
				keys[strings.ToLower(o.Name)] = struct{}{}
			}
		}
		// Assign names to anonymous fields.
		for i, o := range normalized.Outputs {
			if o.Name != "" {
				continue
			}
			o.Name = capitalise(abi.ResolveNameConflict("arg", func(name string) bool { _, ok := keys[name]; return ok }))
			normalized.Outputs[i] = o
			keys[strings.ToLower(o.Name)] = struct{}{}
		}
		isStructured = true
	}

	cb.calls[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: isStructured}
	return nil
}

func (cb *contractBinder) normalizeErrorOrEventFields(originalInputs abi.Arguments) abi.Arguments {
	normalizedArguments := make([]abi.Argument, len(originalInputs))
	copy(normalizedArguments, originalInputs)
	used := make(map[string]bool)

	for i, input := range normalizedArguments {
		if input.Name == "" || isKeyWord(input.Name) {
			normalizedArguments[i].Name = fmt.Sprintf("arg%d", i)
		}
		for index := 0; ; index++ {
			if !used[capitalise(normalizedArguments[i].Name)] {
				used[capitalise(normalizedArguments[i].Name)] = true
				break
			}
			normalizedArguments[i].Name = fmt.Sprintf("%s%d", normalizedArguments[i].Name, index)
		}
		if hasStruct(input.Type) {
			cb.binder.BindStructType(input.Type)
		}
	}
	return normalizedArguments
}

func (cb *contractBinder) bindEvent(original abi.Event) error {
	// Skip anonymous events as they don't support explicit filtering
	if original.Anonymous {
		return nil
	}
	normalizedName, err := cb.RegisterEventIdentifier(original.Name)
	if err != nil {
		return err
	}

	normalized := original
	normalized.Name = normalizedName
	normalized.Inputs = cb.normalizeErrorOrEventFields(original.Inputs)
	cb.events[original.Name] = &tmplEvent{Original: original, Normalized: normalized}
	return nil
}

func (cb *contractBinder) bindError(original abi.Error) error {
	normalizedName, err := cb.RegisterErrorIdentifier(original.Name)
	if err != nil {
		return err
	}

	normalized := original
	normalized.Name = normalizedName
	normalized.Inputs = cb.normalizeErrorOrEventFields(original.Inputs)
	cb.errors[original.Name] = &tmplError{Original: original, Normalized: normalized}
	return nil
}

func BindV2(types []string, abis []string, bytecodes []string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {

	// TODO: validate each alias (ensure it doesn't begin with a digit or other invalid character)

	b := binder{
		contracts: make(map[string]*tmplContractV2),
		structs:   make(map[string]*tmplStruct),
		aliases:   make(map[string]string),
	}
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

		for _, input := range evmABI.Constructor.Inputs {
			if hasStruct(input.Type) {
				bindStructType(input.Type, b.structs)
			}
		}

		cb := contractBinder{
			binder: &b,
			calls:  make(map[string]*tmplMethod),
			events: make(map[string]*tmplEvent),
			errors: make(map[string]*tmplError),

			callIdentifiers:  make(map[string]bool),
			errorIdentifiers: make(map[string]bool),
			eventIdentifiers: make(map[string]bool),
		}
		for _, original := range evmABI.Methods {
			if err := cb.bindMethod(original); err != nil {
				return "", err
			}
		}

		for _, original := range evmABI.Events {
			if err := cb.bindEvent(original); err != nil {
				return "", err
			}
		}
		for _, original := range evmABI.Errors {
			if err := cb.bindError(original); err != nil {
				return "", err
			}
		}

		// replace this with a method call to cb (name it BoundContract()?)
		b.contracts[types[i]] = &tmplContractV2{
			Type:        capitalise(types[i]),
			InputABI:    strings.ReplaceAll(strippedABI, "\"", "\\\""),
			InputBin:    strings.TrimPrefix(strings.TrimSpace(bytecodes[i]), "0x"),
			Constructor: evmABI.Constructor,
			Calls:       cb.calls,
			Events:      cb.events,
			Errors:      cb.errors,
			Libraries:   make(map[string]string),
		}
	}

	invertedLibs := make(map[string]string)
	for pattern, name := range libs {
		invertedLibs[name] = pattern
	}

	data := tmplDataV2{
		Package:   pkg,
		Contracts: b.contracts,
		Libraries: invertedLibs,
		Structs:   b.structs,
	}

	contractsBins := make(map[string]string)
	for typ, contract := range data.Contracts {
		pattern := invertedLibs[typ]
		contractsBins[pattern] = contract.InputBin
	}
	builder := newDepTreeBuilder(nil, contractsBins)
	roots, deps := builder.BuildDepTrees()
	allNodes := append(roots, deps...)
	for _, dep := range allNodes {
		contractType := libs[dep.pattern]
		for subDepPattern, _ := range dep.Flatten() {
			if subDepPattern == dep.pattern {
				// don't include the dep as a dependency of itself
				continue
			}
			subDepType := libs[subDepPattern]
			data.Contracts[contractType].Libraries[subDepType] = subDepPattern
		}
	}
	buffer := new(bytes.Buffer)
	funcs := map[string]interface{}{
		"bindtype":      bindType,
		"bindtopictype": bindTopicType,
		"capitalise":    capitalise,
		"decapitalise":  decapitalise,
		"add": func(val1, val2 int) int {
			return val1 + val2
		},
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
