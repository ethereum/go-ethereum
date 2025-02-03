{{- /* -*- indent-tabs-mode: t -*- */ -}}
// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

import (
	"math/big"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
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
	{{$field.Name}} {{$field.Type}}{{end}}
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
		 return bind.NewContractInstance(backend, addr, c.abi)
	}

	{{ if .Constructor.Inputs }}
	func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) PackConstructor({{range .Constructor.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) []byte {
		enc, err := {{ decapitalise $contract.Type}}.abi.Pack("" {{range .Constructor.Inputs}}, {{.Name}}{{end}})
		if err != nil {
		   panic(err)
		}
		return enc
	}
	{{ end -}}

	{{range .Calls}}
		// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.ID}}.
		//
		// Solidity: {{.Original.String}}
		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Pack{{.Normalized.Name}}({{range .Normalized.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) []byte {
			enc, err := {{ decapitalise $contract.Type}}.abi.Pack("{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
			if err != nil {
				panic(err)
			}
			return enc
		}

		{{/* Unpack method is needed only when there are return args */}}
		{{if .Normalized.Outputs }}
			{{ if .Structured }}
			type {{.Normalized.Name}}Output struct {
			  {{range .Normalized.Outputs}}
			  {{.Name}} {{bindtype .Type $structs}}{{end}}
			}
			{{ end }}
			func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}(data []byte) ({{if .Structured}} {{.Normalized.Name}}Output,{{else}}{{range .Normalized.Outputs}}{{bindtype .Type $structs}},{{end}}{{end}} error) {
				out, err := {{ decapitalise $contract.Type}}.abi.Unpack("{{.Original.Name}}", data)
				{{if .Structured}}
				outstruct := new({{.Normalized.Name}}Output)
				if err != nil {
					return *outstruct, err
				}
				{{range $i, $t := .Normalized.Outputs}}
				{{if ispointertype .Type}}
					outstruct.{{.Name}} = abi.ConvertType(out[{{$i}}], new({{underlyingbindtype .Type }})).({{bindtype .Type $structs}})
				{{ else }}
					outstruct.{{.Name}} = *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}})
				{{ end }}{{end}}

				return *outstruct, err
				{{else}}
				if err != nil {
					return {{range $i, $_ := .Normalized.Outputs}}{{if ispointertype .Type}}new({{underlyingbindtype .Type }}), {{else}}*new({{bindtype .Type $structs}}), {{end}}{{end}} err
				}
				{{range $i, $t := .Normalized.Outputs}}
				{{ if ispointertype .Type }}
				out{{$i}} := abi.ConvertType(out[{{$i}}], new({{underlyingbindtype .Type}})).({{bindtype .Type $structs}})
				{{ else }}
				out{{$i}} := *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}})
				{{ end }}
				{{end}}

				return {{range $i, $t := .Normalized.Outputs}}out{{$i}}, {{end}} err
				{{end}}
			}
		{{end}}
	{{end}}

	{{range .Events}}
		// {{$contract.Type}}{{.Normalized.Name}} represents a {{.Normalized.Name}} event raised by the {{$contract.Type}} contract.
		type {{$contract.Type}}{{.Normalized.Name}} struct { {{range .Normalized.Inputs}}
			{{capitalise .Name}} {{if .Indexed}}{{bindtopictype .Type $structs}}{{else}}{{bindtype .Type $structs}}{{end}}; {{end}}
			Raw *types.Log // Blockchain specific contextual infos
		}

		const {{$contract.Type}}{{.Normalized.Name}}EventName = "{{.Original.Name}}"

		func ({{$contract.Type}}{{.Normalized.Name}}) ContractEventName() string {
			return {{$contract.Type}}{{.Normalized.Name}}EventName
		}

		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Event(log *types.Log) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			event := "{{.Original.Name}}"
			if log.Topics[0] != {{ decapitalise $contract.Type}}.abi.Events[event].ID {
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
	func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) UnpackError(raw []byte) any {
		// TODO: we should be able to discern the error type from the selector, instead of trying each possible type
		// strip off the error type selector
		raw = raw[4:]
		{{$i := 0}}
		{{range $k, $v := .Errors}}
			{{ if eq $i 0 }}
				if val, err := {{ decapitalise $contract.Type}}.Unpack{{.Normalized.Name}}Error(raw); err == nil {
					return val
			{{ else }}
				} else if val, err := {{ decapitalise $contract.Type}}.Unpack{{.Normalized.Name}}Error(raw); err == nil {
					return val
			{{ end -}}
			{{$i = add $i 1}}
		{{end -}}
		}
		return nil
	}
	{{ end -}}

	{{range .Errors}}
		// {{$contract.Type}}{{.Normalized.Name}} represents a {{.Normalized.Name}} error raised by the {{$contract.Type}} contract.
		type {{$contract.Type}}{{.Normalized.Name}} struct { {{range .Normalized.Inputs}}
			{{capitalise .Name}} {{if .Indexed}}{{bindtopictype .Type $structs}}{{else}}{{bindtype .Type $structs}}{{end}}; {{end}}
		}

		func {{$contract.Type}}{{.Normalized.Name}}ErrorID() common.Hash {
			return common.HexToHash("{{.Original.ID}}")
		}

		func ({{ decapitalise $contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Error(raw []byte) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			errName := "{{.Normalized.Name}}"
			out := new({{$contract.Type}}{{.Normalized.Name}})
			if err := {{ decapitalise $contract.Type}}.abi.UnpackIntoInterface(out, errName, raw); err != nil {
				return nil, err
			}
			return out, nil
		}
	{{end}}
{{end}}
