// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

import (
	"math/big"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = bind.Bind
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
	var {{$contract.Type}}LibraryDeps = []*bind.MetaData{
	{{range $name, $pattern := .Libraries -}}
		{{$name}}MetaData,
	{{ end}}
	}

	// TODO: convert this type to value type after everything works.
	// {{.Type}}MetaData contains all meta data concerning the {{.Type}} contract.
	var {{.Type}}MetaData = &bind.MetaData{
		ABI: "{{.InputABI}}",
		Pattern: "{{index $.Libraries .Type}}",
		{{if .InputBin -}}
		Bin: "0x{{.InputBin}}",
		{{end}}
	}

	// {{.Type}} is an auto generated Go binding around an Ethereum contract.
	type {{.Type}} struct {
		abi abi.ABI
	}

	// New{{.Type}} creates a new instance of {{.Type}}.
	func New{{.Type}}() (*{{.Type}}, error) {
		parsed, err := {{.Type}}MetaData.GetAbi()
		if err != nil {
			return nil, err
		}
		return &{{.Type}}{abi: *parsed}, nil
	}

	func (_{{$contract.Type}} *{{$contract.Type}}) PackConstructor({{range .Constructor.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) ([]byte, error) {
		return _{{$contract.Type}}.abi.Pack("" {{range .Constructor.Inputs}}, {{.Name}}{{end}})
	}

	{{range .Calls}}
		// {{.Normalized.Name}} is a free data retrieval call binding the contract method 0x{{printf "%x" .Original.ID}}.
		//
		// Solidity: {{.Original.String}}
		func (_{{$contract.Type}} *{{$contract.Type}}) Pack{{.Normalized.Name}}({{range .Normalized.Inputs}} {{.Name}} {{bindtype .Type $structs}}, {{end}}) ([]byte, error) {
			return _{{$contract.Type}}.abi.Pack("{{.Original.Name}}" {{range .Normalized.Inputs}}, {{.Name}}{{end}})
		}

		{{/* Unpack method is needed only when there are return args */}}
		{{if .Normalized.Outputs }}
		    {{ if .Structured }}
		    type {{.Normalized.Name}}Output struct {
		      {{range .Normalized.Outputs}}
		      {{.Name}} {{bindtype .Type $structs}}{{end}}
		    }
		    {{ end }}
			func (_{{$contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}(data []byte) ({{if .Structured}} {{.Normalized.Name}}Output,{{else}}{{range .Normalized.Outputs}}{{bindtype .Type $structs}},{{end}}{{end}} error) {
				out, err := _{{$contract.Type}}.abi.Unpack("{{.Original.Name}}", data)
				{{if .Structured}}
				outstruct := new({{.Normalized.Name}}Output)
				if err != nil {
					return *outstruct, err
				}
				{{range $i, $t := .Normalized.Outputs}}
				outstruct.{{.Name}} = *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}}){{end}}

				return *outstruct, err
				{{else}}
				if err != nil {
					return {{range $i, $_ := .Normalized.Outputs}}*new({{bindtype .Type $structs}}), {{end}} err
				}
				{{range $i, $t := .Normalized.Outputs}}
				out{{$i}} := *abi.ConvertType(out[{{$i}}], new({{bindtype .Type $structs}})).(*{{bindtype .Type $structs}}){{end}}

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

		func (_{{$contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Event(log *types.Log) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			event := "{{.Original.Name}}"
			if log.Topics[0] != _{{$contract.Type}}.abi.Events[event].ID {
				return nil, errors.New("event signature mismatch")
			}
			out := new({{$contract.Type}}{{.Normalized.Name}})
			if len(log.Data) > 0 {
				if err := _{{$contract.Type}}.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
					return nil, err
				}
			}
			var indexed abi.Arguments
			for _, arg := range _{{$contract.Type}}.abi.Events[event].Inputs {
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
    func (_{{$contract.Type}} *{{$contract.Type}}) UnpackError(raw []byte) any {
        {{$i := 0}}
        {{range $k, $v := .Errors}}
            {{ if eq $i 0 }}
                if val, err := _{{$contract.Type}}.Unpack{{.Normalized.Name}}Error(raw); err == nil {
                    return val
            {{ else }}
                } else if val, err := _{{$contract.Type}}.Unpack{{.Normalized.Name}}Error(raw); err == nil {
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

		func (_{{$contract.Type}} *{{$contract.Type}}) Unpack{{.Normalized.Name}}Error(raw []byte) (*{{$contract.Type}}{{.Normalized.Name}}, error) {
			errName := "{{.Normalized.Name}}"
			out := new({{$contract.Type}}{{.Normalized.Name}})
            if err := _{{$contract.Type}}.abi.UnpackIntoInterface(out, errName, raw); err != nil {
                // TODO: output can be non-pointer type.
                return nil, err
            }
			return out, nil
		}
    {{end}}
{{end}}