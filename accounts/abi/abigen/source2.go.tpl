// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

import (
	"bytes"
	"math/big"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

{{$structs := .Structs}}
{{range $structs}}
	// {{.Name}} is an auto generated low-level Go binding around an user-defined struct.
	type {{.Name}} struct {
	{{range $field := .Fields}}
	{{capitalise $field.Name}} {{$field.Type}}{{end}}
	}
{{end}}

{{range $contract := .Contracts}}
	// {{.Type}}MetaData contains all meta data concerning the {{.Type}} contract.
	var {{.Type}}MetaData = bind.MetaData{
		ABI: "{{.InputABI}}",
		{{if (index $.Libraries .Type) -}}
		ID: "{{index $.Libraries .Type}}",
		{{ else -}}
		ID: "{{.Type}}",
		{{end -}}
		{{if .InputBin -}}
		Bin: "0x{{.InputBin}}",
		{{end -}}
		{{if .Libraries -}}
		Deps: []*bind.MetaData{
		{{- range $name, $pattern := .Libraries}}
			&{{$name}}MetaData,
		{{- end}}
		},
		{{end}}
	}

	// {{.Type}} is an auto generated Go binding around an Ethereum contract.
	type {{.Type}} struct {
		abi abi.ABI
	}

	// New{{.Type}} creates a new instance of {{.Type}}.
	func New{{.Type}}() *{{.Type}} {
		parsed, err := {{.Type}}MetaData.ParseABI()
		if err != nil {
			panic(errors.New("invalid ABI: " + err.Error()))
		}
		return &{{.Type}}{abi: *parsed}
	}

	// Instance creates a wrapper for a deployed contract instance at the given address.
	// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
	func (c *{{.Type}}) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
		 return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
	}

	{{ if .Constructor.Inputs }}
	// PackConstructor is the Go binding used to pack the parameters required for
	// contract deployment.
	//
	// Solidity: {{.Constructor.String}}
	func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) PackConstructor({{range .Constructor.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) []byte {
		enc, err := {{ decapitalise $contract.Type}}.abi.Pack("" {{range .Constructor.Inputs}}, {{.Name}}{{end}})
		if err != nil {
		   panic(err)
		}
		return enc
	}
	{{ end }}

	{{range .Calls}}
		// Pack{{.Normalized.Name}} is the Go binding used to pack the parameters required for calling
		// the contract method with ID 0x{{printf "%x" .Original.ID}}.  This method will panic if any
		// invalid/nil inputs are passed.
		//
		// Solidity: {{.Original.String}}
		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Pack{{.Normalized.Name}}({{range .Normalized.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) []byte {
			enc, err := {{ decapitalise $contract.Type}}.abi.Pack("{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
			if err != nil {
				panic(err)
			}
			return enc
		}

		// TryPack{{.Normalized.Name}} is the Go binding used to pack the parameters required for calling
		// the contract method with ID 0x{{printf "%x" .Original.ID}}.  This method will return an error
		// if any inputs are invalid/nil.
		//
		// Solidity: {{.Original.String}}
		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) TryPack{{.Normalized.Name}}({{range .Normalized.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) ([]byte, error) {
			return {{ decapitalise $contract.Type}}.abi.Pack("{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
		}

		{{/* Unpack method is needed only when there are return args */}}
		{{if .Normalized.Outputs }}
			{{ if .Structured }}
			// {{.Normalized.Name}}Output serves as a container for the return parameters of contract
			// method {{ .Normalized.Name }}.
			type {{.Normalized.Name}}Output struct {
			  {{range .Normalized.Outputs}}
			  {{capitalise .Name}} {{bindtype .Type $structs}}{{end}}
			}
			{{ end }}

			// Unpack{{.Normalized.Name}} is the Go binding that unpacks the parameters returned
			// from invoking the contract method with ID 0x{{printf "%x" .Original.ID}}.
			//
			// Solidity: {{.Original.String}}
			func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}(data []byte) (
				{{- if .Structured}} {{.Normalized.Name}}Output,{{else}}
				{{- range .Normalized.Outputs}} {{bindtype .Type $structs}},{{- end }}
				{{- end }} error) {
				out, err := {{ decapitalise $contract.Type}}.abi.Unpack("{{.Original.Name}}", data)
				{{- if .Structured}}
				outstruct := new({{.Normalized.Name}}Output)
				if err != nil {
					return *outstruct, err
				}
				{{- range $i, $t := .Normalized.Outputs}}
				{{- if ispointertype .Type}}
					outstruct.{{capitalise .Name}} = abi.ConvertType(out[{{$i}}], new({{underlyingbindtype .Type }})).({{bindtype .Type $structs}})
				{{- else }}
					outstruct.{{capitalise .Name}} = *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}})
				{{- end }}
				{{- end }}
				return *outstruct, nil{{else}}
				if err != nil {
					return {{range $i, $_ := .Normalized.Outputs}}{{if ispointertype .Type}}new({{underlyingbindtype .Type }}), {{else}}*new({{bindtype .Type $structs}}), {{end}}{{end}} err
				}
				{{- range $i, $t := .Normalized.Outputs}}
				{{- if ispointertype .Type }}
				out{{$i}} := abi.ConvertType(out[{{$i}}], new({{underlyingbindtype .Type}})).({{bindtype .Type $structs}})
				{{- else }}
				out{{$i}} := *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}})
				{{- end }}
				{{- end}}
				return {{range $i, $t := .Normalized.Outputs}}out{{$i}}, {{end}} nil
				{{- end}}
			}
		{{end}}
	{{end}}

	{{range .Events}}
		// {{$contract.Type}}{{.Normalized.Name}} represents a {{.Original.Name}} event raised by the {{$contract.Type}} contract.
		type {{$contract.Type}}{{.Normalized.Name}} struct {
			{{- range .Normalized.Inputs}}
				{{ capitalise .Name}}
				{{- if .Indexed}} {{ bindtopictype .Type $structs}}{{- else}} {{ bindtype .Type $structs}}{{ end }}
			{{- end}}
			Raw *types.Log // Blockchain specific contextual infos
		}

		const {{$contract.Type}}{{.Normalized.Name}}EventName = "{{.Original.Name}}"

		// ContractEventName returns the user-defined event name.
		func ({{$contract.Type}}{{.Normalized.Name}}) ContractEventName() string {
			return {{$contract.Type}}{{.Normalized.Name}}EventName
		}

		// Unpack{{.Normalized.Name}}Event is the Go binding that unpacks the event data emitted
		// by contract.
		//
		// Solidity: {{.Original.String}}
		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Event(log *types.Log) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			event := "{{.Original.Name}}"
			if len(log.Topics) == 0 || log.Topics[0] != {{ decapitalise $contract.Type}}.abi.Events[event].ID {
				return nil, errors.New("event signature mismatch")
			}
			out := new({{$contract.Type}}{{.Normalized.Name}})
			if len(log.Data) > 0 {
				if err := {{ decapitalise $contract.Type}}.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
					return nil, err
				}
			}
			var indexed abi.Arguments
			for _, arg := range {{ decapitalise $contract.Type}}.abi.Events[event].Inputs {
				if arg.Indexed {
					indexed = append(indexed, arg)
				}
			}
			if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
				return nil, err
			}
			out.Raw = log
			return out, nil
		}
	{{end}}

	{{ if .Errors }}
	// UnpackError attempts to decode the provided error data using user-defined
	// error definitions.
	func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) UnpackError(raw []byte) (any, error) {
		{{- range $k, $v := .Errors}}
		if bytes.Equal(raw[:4], {{ decapitalise $contract.Type}}.abi.Errors["{{.Normalized.Name}}"].ID.Bytes()[:4]) {
			return {{ decapitalise $contract.Type}}.Unpack{{.Normalized.Name}}Error(raw[4:])
		}
		{{- end }}
		return nil, errors.New("Unknown error")
	}
	{{ end }}

	{{range .Errors}}
		// {{$contract.Type}}{{.Normalized.Name}} represents a {{.Original.Name}} error raised by the {{$contract.Type}} contract.
		type {{$contract.Type}}{{.Normalized.Name}} struct { {{range .Normalized.Inputs}}
			{{capitalise .Name}} {{if .Indexed}}{{bindtopictype .Type $structs}}{{else}}{{bindtype .Type $structs}}{{end}}; {{end}}
		}

		// ErrorID returns the hash of canonical representation of the error's signature.
		//
		// Solidity: {{.Original.String}}
		func {{$contract.Type}}{{.Normalized.Name}}ErrorID() common.Hash {
			return common.HexToHash("{{.Original.ID}}")
		}

		// Unpack{{.Normalized.Name}}Error is the Go binding used to decode the provided
		// error data into the corresponding Go error struct.
		//
		// Solidity: {{.Original.String}}
		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Error(raw []byte) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			out := new({{$contract.Type}}{{.Normalized.Name}})
			if err := {{ decapitalise $contract.Type}}.abi.UnpackIntoInterface(out, "{{.Normalized.Name}}", raw); err != nil {
				return nil, err
			}
			return out, nil
		}
	{{end}}
{{end}}
