package bind

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"go/format"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"unicode"
)

// underlyingBindType returns the underlying Go type represented by the given type, panicking if it is not a pointer type.
func underlyingBindType(typ abi.Type) string {
	goType := typ.GetType()
	if goType.Kind() != reflect.Pointer {
		panic("trying to retrieve underlying bind type of non-pointer type.")
	}
	return goType.Elem().String()
}

// isPointerType returns true if the
func isPointerType(typ abi.Type) bool {
	return typ.GetType().Kind() == reflect.Pointer
}

type binder struct {
	// contracts is the map of each individual contract requested binding
	contracts map[string]*tmplContractV2

	// structs is the map of all redeclared structs shared by passed contracts.
	structs map[string]*tmplStruct

	// aliases is a map for renaming instances of named events/functions/errors to specified values
	aliases map[string]string
}

// registerIdentifier applies alias renaming, name normalization (conversion to camel case), and registers the normalized
// name in the specified identifier map.  It returns an error if the normalized name already exists in the map.
func (b *contractBinder) registerIdentifier(identifiers map[string]bool, original string) (normalized string, err error) {
	normalized = abi.ToCamelCase(alias(b.binder.aliases, original))
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

// BindStructType register the type to be emitted as a struct in the
// bindings.
func (b *binder) BindStructType(typ abi.Type) {
	bindStructType(typ, b.structs)
}

// contractBinder holds state for binding of a single contract
type contractBinder struct {
	binder *binder
	calls  map[string]*tmplMethod
	events map[string]*tmplEvent
	errors map[string]*tmplError

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

// bindMethod registers a method to be emitted in the bindings.
// The name, inputs and outputs are normalized.  If any inputs are
// struct-type their structs are registered to be emitted in the bindings.
// Any methods that return more than one output have their result coalesced
// into a struct.
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
	for j, output := range normalized.Outputs {
		if output.Name != "" {
			normalized.Outputs[j].Name = abi.ToCamelCase(output.Name)
		}
		if hasStruct(output.Type) {
			cb.binder.BindStructType(output.Type)
		}
	}
	isStructured := structured(original.Outputs)
	// if the call returns multiple values, coallesce them into a struct
	if len(normalized.Outputs) > 1 {
		isStructured = true
	}

	cb.calls[original.Name] = &tmplMethod{Original: original, Normalized: normalized, Structured: isStructured}
	return nil
}

// normalize a set of arguments by stripping underscores, giving a generic name in the case where
// the arg name collides with a reserved Go keyword, and finally converting to camel-case.
func normalizeArgs(args abi.Arguments) abi.Arguments {
	args = slices.Clone(args)
	used := make(map[string]bool)

	for i, input := range args {
		if isKeyWord(input.Name) {
			args[i].Name = fmt.Sprintf("Arg%d", i)
		}
		args[i].Name = abi.ToCamelCase(args[i].Name)
		if args[i].Name == "" {
			args[i].Name = fmt.Sprintf("Arg%d", i)
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

// normalizeErrorOrEventFields normalizes errors/events for emitting through bindings:
// Any anonymous fields are given generated names.
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

		// TODO: normalize these args, add unit tests that fail in the current commit.
		for _, input := range evmABI.Constructor.Inputs {
			if hasStruct(input.Type) {
				bindStructType(input.Type, b.structs)
			}
		}

		cb := newContractBinder(&b)
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
		"bindtype":      bindType,
		"bindtopictype": bindTopicType,
		"capitalise":    abi.ToCamelCase,
		"decapitalise":  decapitalise,
		"add": func(val1, val2 int) int {
			return val1 + val2
		},
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