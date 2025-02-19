// Copyright 2024 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"go/format"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// underlyingBindType returns a string representation of the Go type
// that corresponds to the given ABI type, panicking if it is not a
// pointer.
func underlyingBindType(typ abi.Type) string {
	goType := typ.GetType()
	if goType.Kind() != reflect.Pointer {
		panic("trying to retrieve underlying bind type of non-pointer type.")
	}
	return goType.Elem().String()
}

// isPointerType returns true if the underlying type is a pointer.
func isPointerType(typ abi.Type) bool {
	return typ.GetType().Kind() == reflect.Pointer
}

// OLD:
// binder is used during the conversion of an ABI definition into Go bindings
// (as part of the execution of BindV2). In contrast to contractBinder, binder
// contains binding-generation-state that is shared between contracts:
//
// a global struct map of structs emitted by all contracts is tracked and expanded.
// Structs generated in the bindings are not prefixed with the contract name
// that uses them (to keep the generated bindings less verbose).
//
// This contrasts to other per-contract state (constructor/method/event/error,
// pack/unpack methods) which are guaranteed to be unique because of their
// association with the uniquely-named owning contract (whether prefixed in the
// generated symbol name, or as a member method on a contract struct).
//
// In addition, binder contains the input alias map. In BindV2, a binder is
// instantiated to produce a set of tmplContractV2 and tmplStruct objects from
// the provided ABI definition. These are used as part of the input to rendering
// the binding template.

// NEW:
// binder is used to translate an ABI definition into a set of data-structures
// that will be used to render the template and produce Go bindings.  This can
// be thought of as the "backend" that sanitizes the ABI definition to a format
// that can be directly rendered with minimal complexity in the template.
//
// The input data to the template rendering consists of:
//   - the set of all contracts requested for binding, each containing
//     methods/events/errors to emit pack/unpack methods for.
//   - the set of structures defined by the contracts, and created
//     as part of the binding process.
type binder struct {
	// contracts is the map of each individual contract requested binding.
	// It is keyed by the contract name provided in the ABI definition.
	contracts map[string]*tmplContractV2

	// structs is the map of all emitted structs from contracts being bound.
	// it is keyed by a unique identifier generated from the name of the owning contract
	// and the solidity type signature of the struct
	structs map[string]*tmplStruct

	// aliases is a map for renaming instances of named events/functions/errors
	// to specified values. it is keyed by source symbol name, and values are
	// what the replacement name should be.
	aliases map[string]string
}

// BindStructType registers the type to be emitted as a struct in the
// bindings.
func (b *binder) BindStructType(typ abi.Type) {
	bindStructType(typ, b.structs)
}

// contractBinder holds state for binding of a single contract. It is a type
// registry for compiling maps of identifiers that will be emitted in generated
// bindings.
type contractBinder struct {
	binder *binder

	// all maps are keyed by the original (non-normalized) name of the symbol in question
	// from the provided ABI definition.
	calls            map[string]*tmplMethod
	events           map[string]*tmplEvent
	errors           map[string]*tmplError
	callIdentifiers  map[string]bool
	eventIdentifiers map[string]bool
	errorIdentifiers map[string]bool
}

func newContractBinder(binder *binder) *contractBinder {
	return &contractBinder{
		binder,
		make(map[string]*tmplMethod),
		make(map[string]*tmplEvent),
		make(map[string]*tmplError),
		make(map[string]bool),
		make(map[string]bool),
		make(map[string]bool),
	}
}

// registerIdentifier applies alias renaming, name normalization (conversion
// from snake to camel-case), and registers the normalized name in the specified identifier map.
// It returns an error if the normalized name already exists in the map.
func (cb *contractBinder) registerIdentifier(identifiers map[string]bool, original string) (normalized string, err error) {
	normalized = abi.ToCamelCase(alias(cb.binder.aliases, original))

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

// bindMethod registers a method to be emitted in the bindings. The name, inputs
// and outputs are normalized. If any inputs are struct-type their structs are
// registered to be emitted in the bindings. Any methods that return more than
// one output have their results gathered into a struct.
func (cb *contractBinder) bindMethod(original abi.Method) error {
	normalized := original
	normalizedName, err := cb.registerIdentifier(cb.callIdentifiers, original.Name)
	if err != nil {
		return err
	}
	normalized.Name = normalizedName

	normalized.Inputs = normalizeArgs(original.Inputs)
	for _, input := range normalized.Inputs {
		if hasStruct(input.Type) {
			cb.binder.BindStructType(input.Type)
		}
	}
	normalized.Outputs = normalizeArgs(original.Outputs)
	for _, output := range normalized.Outputs {
		if hasStruct(output.Type) {
			cb.binder.BindStructType(output.Type)
		}
	}

	var isStructured bool
	// If the call returns multiple values, gather them into a struct
	if len(normalized.Outputs) > 1 {
		isStructured = true
	}
	cb.calls[original.Name] = &tmplMethod{
		Original:   original,
		Normalized: normalized,
		Structured: isStructured,
	}
	return nil
}

// normalize a set of arguments by stripping underscores, giving a generic name
// in the case where the arg name collides with a reserved Go keyword, and finally
// converting to camel-case.
func normalizeArgs(args abi.Arguments) abi.Arguments {
	args = slices.Clone(args)
	used := make(map[string]bool)

	for i, input := range args {
		if isKeyWord(input.Name) {
			args[i].Name = fmt.Sprintf("arg%d", i)
		}
		args[i].Name = abi.ToCamelCase(args[i].Name)
		if args[i].Name == "" {
			args[i].Name = fmt.Sprintf("arg%d", i)
		} else {
			args[i].Name = strings.ToLower(args[i].Name[:1]) + args[i].Name[1:]
		}

		for index := 0; ; index++ {
			if !used[args[i].Name] {
				used[args[i].Name] = true
				break
			}
			args[i].Name = fmt.Sprintf("%s%d", args[i].Name, index)
		}
	}
	return args
}

// normalizeErrorOrEventFields normalizes errors/events for emitting through
// bindings: Any anonymous fields are given generated names.
func (cb *contractBinder) normalizeErrorOrEventFields(originalInputs abi.Arguments) abi.Arguments {
	normalizedArguments := normalizeArgs(originalInputs)
	for _, input := range normalizedArguments {
		if hasStruct(input.Type) {
			cb.binder.BindStructType(input.Type)
		}
	}
	return normalizedArguments
}

// bindEvent normalizes an event and registers it to be emitted in the bindings.
func (cb *contractBinder) bindEvent(original abi.Event) error {
	// Skip anonymous events as they don't support explicit filtering
	if original.Anonymous {
		return nil
	}
	normalizedName, err := cb.registerIdentifier(cb.eventIdentifiers, original.Name)
	if err != nil {
		return err
	}

	normalized := original
	normalized.Name = normalizedName
	normalized.Inputs = cb.normalizeErrorOrEventFields(original.Inputs)
	cb.events[original.Name] = &tmplEvent{Original: original, Normalized: normalized}
	return nil
}

// bindError normalizes an error and registers it to be emitted in the bindings.
func (cb *contractBinder) bindError(original abi.Error) error {
	normalizedName, err := cb.registerIdentifier(cb.errorIdentifiers, original.Name)
	if err != nil {
		return err
	}

	normalized := original
	normalized.Name = normalizedName
	normalized.Inputs = cb.normalizeErrorOrEventFields(original.Inputs)
	cb.errors[original.Name] = &tmplError{Original: original, Normalized: normalized}
	return nil
}

// parseLibraryDeps extracts references to library dependencies from the unlinked
// hex string deployment bytecode.
func parseLibraryDeps(unlinkedCode string) (res []string) {
	reMatchSpecificPattern, err := regexp.Compile(`__\$([a-f0-9]+)\$__`)
	if err != nil {
		panic(err)
	}
	for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(unlinkedCode, -1) {
		res = append(res, match[1])
	}
	return res
}

// iterSorted iterates the map in the lexicographic order of the keys calling
// onItem on each. If the callback returns an error, iteration is halted and
// the error is returned from iterSorted.
func iterSorted[V any](inp map[string]V, onItem func(string, V) error) error {
	var sortedKeys []string
	for key := range inp {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		if err := onItem(key, inp[key]); err != nil {
			return err
		}
	}
	return nil
}

// BindV2 generates a Go wrapper around a contract ABI. This wrapper isn't meant
// to be used as is in client code, but rather as an intermediate struct which
// enforces compile time type safety and naming convention as opposed to having to
// manually maintain hard coded strings that break on runtime.
func BindV2(types []string, abis []string, bytecodes []string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {
	b := binder{
		contracts: make(map[string]*tmplContractV2),
		structs:   make(map[string]*tmplStruct),
		aliases:   aliases,
	}
	for i := 0; i < len(types); i++ {
		// Parse the actual ABI to generate the binding for
		evmABI, err := abi.JSON(strings.NewReader(abis[i]))
		if err != nil {
			return "", err
		}

		for _, input := range evmABI.Constructor.Inputs {
			if hasStruct(input.Type) {
				bindStructType(input.Type, b.structs)
			}
		}

		cb := newContractBinder(&b)
		err = iterSorted(evmABI.Methods, func(_ string, original abi.Method) error {
			return cb.bindMethod(original)
		})
		if err != nil {
			return "", err
		}
		err = iterSorted(evmABI.Events, func(_ string, original abi.Event) error {
			return cb.bindEvent(original)
		})
		if err != nil {
			return "", err
		}
		err = iterSorted(evmABI.Errors, func(_ string, original abi.Error) error {
			return cb.bindError(original)
		})
		if err != nil {
			return "", err
		}
		b.contracts[types[i]] = newTmplContractV2(types[i], abis[i], bytecodes[i], evmABI.Constructor, cb)
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

	for typ, contract := range data.Contracts {
		for _, depPattern := range parseLibraryDeps(contract.InputBin) {
			data.Contracts[typ].Libraries[libs[depPattern]] = depPattern
		}
	}
	buffer := new(bytes.Buffer)
	funcs := map[string]interface{}{
		"bindtype":           bindType,
		"bindtopictype":      bindTopicType,
		"capitalise":         abi.ToCamelCase,
		"decapitalise":       decapitalise,
		"ispointertype":      isPointerType,
		"underlyingbindtype": underlyingBindType,
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
