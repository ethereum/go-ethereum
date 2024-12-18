package bind

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"go/format"
	"strings"
	"text/template"
	"unicode"
)

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
	normalized = methodNormalizer(alias(b.binder.aliases, original))
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

// RegisterCallIdentifier applies registerIdentifier for contract methods.
func (b *contractBinder) RegisterCallIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.callIdentifiers, id)
}

// RegisterEventIdentifier applies registerIdentifier for contract events.
func (b *contractBinder) RegisterEventIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.eventIdentifiers, id)
}

// RegisterErrorIdentifier applies registerIdentifier for contract errors.
func (b *contractBinder) RegisterErrorIdentifier(id string) (string, error) {
	return b.registerIdentifier(b.errorIdentifiers, id)
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

// bindMethod registers a method to be emitted in the bindings.
// The name, inputs and outputs are normalized.  If any inputs are
// struct-type their structs are registered to be emitted in the bindings.
// Any methods that return more than one output have their result coalesced
// into a struct.
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

// normalizeErrorOrEventFields normalizes errors/events for emitting through bindings:
// Any anonymous fields are given generated names.
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

// bindEvent normalizes an event and registers it to be emitted in the bindings.
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

// bindEvent normalizes an error and registers it to be emitted in the bindings.
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
		// Strip any whitespace from the JSON ABI
		strippedABI := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, abis[i])
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
