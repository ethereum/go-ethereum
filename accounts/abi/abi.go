// Copyright 2015 The go-ethereum Authors
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

package abi

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Executer is an executer method for performing state executions. It takes one
// argument which is the input data and expects output data to be returned as
// multiple 32 byte word length concatenated slice
type Executer func(datain []byte) []byte

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Methods map[string]Method
	Events  map[string]Event
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// tests, tests whether the given input would result in a successful
// call. Checks argument list count and matches input to `input`.
func (abi ABI) pack(name string, args ...interface{}) ([]byte, error) {
	method := abi.Methods[name]

	// variable input is the output appended at the end of packed
	// output. This is used for strings and bytes types input.
	var variableInput []byte

	var ret []byte
	for i, a := range args {
		input := method.Inputs[i]
		// pack the input
		packed, err := input.Type.pack(a)
		if err != nil {
			return nil, fmt.Errorf("`%s` %v", name, err)
		}

		// check for a string or bytes input type
		switch input.Type.T {
		case StringTy, BytesTy:
			// calculate the offset
			offset := len(method.Inputs)*32 + len(variableInput)
			// set the offset
			ret = append(ret, packNum(reflect.ValueOf(offset), UintTy)...)
			// Append the packed output to the variable input. The variable input
			// will be appended at the end of the input.
			variableInput = append(variableInput, packed...)
		default:
			// append the packed value to the input
			ret = append(ret, packed...)
		}
	}
	// append the variable input at the end of the packed input
	ret = append(ret, variableInput...)

	return ret, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	method, exist := abi.Methods[name]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", name)
	}

	// start with argument count match
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(method.Inputs))
	}

	arguments, err := abi.pack(name, args...)
	if err != nil {
		return nil, err
	}

	// Set function id
	packed := abi.Methods[name].Id()
	packed = append(packed, arguments...)

	return packed, nil
}

// toGoType parses the input and casts it to the proper type defined by the ABI
// argument in T.
func toGoType(i int, t Argument, output []byte) (interface{}, error) {
	index := i * 32

	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), index+32)
	}

	// Parse the given index output and check whether we need to read
	// a different offset and length based on the type (i.e. string, bytes)
	var returnOutput []byte
	switch t.Type.T {
	case StringTy, BytesTy: // variable arrays are written at the end of the return bytes
		// parse offset from which we should start reading
		offset := int(common.BytesToBig(output[index : index+32]).Uint64())
		if offset+32 > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32)
		}
		// parse the size up until we should be reading
		size := int(common.BytesToBig(output[offset : offset+32]).Uint64())
		if offset+32+size > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32+size)
		}

		// get the bytes for this return value
		returnOutput = output[offset+32 : offset+32+size]
	default:
		returnOutput = output[index : index+32]
	}

	// cast bytes to abi return type
	switch t.Type.T {
	case IntTy:
		return common.BytesToBig(returnOutput), nil
	case UintTy:
		return common.BytesToBig(returnOutput), nil
	case BoolTy:
		return common.BytesToBig(returnOutput).Uint64() > 0, nil
	case AddressTy:
		return common.BytesToAddress(returnOutput), nil
	case HashTy:
		return common.BytesToHash(returnOutput), nil
	case BytesTy, FixedBytesTy:
		return returnOutput, nil
	case StringTy:
		return string(returnOutput), nil
	}
	return nil, fmt.Errorf("abi: unknown type %v", t.Type.T)
}

// Call will unmarshal the output of the call in v. It will return an error if
// invalid type is given or if the output is too short to conform to the ABI
// spec.
//
// Call supports all of the available types and accepts a struct or an interface
// slice if the return is a tuple.
func (abi ABI) Call(executer Executer, v interface{}, name string, args ...interface{}) error {
	callData, err := abi.Pack(name, args...)
	if err != nil {
		return err
	}

	return abi.unmarshal(v, name, executer(callData))
}

// these variable are used to determine certain types during type assertion for
// assignment.
var (
	r_interSlice = reflect.TypeOf([]interface{}{})
	r_hash       = reflect.TypeOf(common.Hash{})
	r_bytes      = reflect.TypeOf([]byte{})
	r_byte       = reflect.TypeOf(byte(0))
)

// unmarshal output in v according to the abi specification
func (abi ABI) unmarshal(v interface{}, name string, output []byte) error {
	var method = abi.Methods[name]

	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}

	value := reflect.ValueOf(v).Elem()
	typ := value.Type()

	if len(method.Outputs) > 1 {
		switch value.Kind() {
		// struct will match named return values to the struct's field
		// names
		case reflect.Struct:
			for i := 0; i < len(method.Outputs); i++ {
				marshalledValue, err := toGoType(i, method.Outputs[i], output)
				if err != nil {
					return err
				}
				reflectValue := reflect.ValueOf(marshalledValue)

				for j := 0; j < typ.NumField(); j++ {
					field := typ.Field(j)
					// TODO read tags: `abi:"fieldName"`
					if field.Name == strings.ToUpper(method.Outputs[i].Name[:1])+method.Outputs[i].Name[1:] {
						if err := set(value.Field(j), reflectValue, method.Outputs[i]); err != nil {
							return err
						}
					}
				}
			}
		case reflect.Slice:
			if !value.Type().AssignableTo(r_interSlice) {
				return fmt.Errorf("abi: cannot marshal tuple in to slice %T (only []interface{} is supported)", v)
			}

			// create a new slice and start appending the unmarshalled
			// values to the new interface slice.
			z := reflect.MakeSlice(typ, 0, len(method.Outputs))
			for i := 0; i < len(method.Outputs); i++ {
				marshalledValue, err := toGoType(i, method.Outputs[i], output)
				if err != nil {
					return err
				}
				z = reflect.Append(z, reflect.ValueOf(marshalledValue))
			}
			value.Set(z)
		default:
			return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
		}

	} else {
		marshalledValue, err := toGoType(0, method.Outputs[0], output)
		if err != nil {
			return err
		}
		if err := set(value, reflect.ValueOf(marshalledValue), method.Outputs[0]); err != nil {
			return err
		}
	}

	return nil
}

// set attempts to assign src to dst by either setting, copying or otherwise.
//
// set is a bit more lenient when it comes to assignment and doesn't force an as
// strict ruleset as bare `reflect` does.
func set(dst, src reflect.Value, output Argument) error {
	dstType := dst.Type()
	srcType := src.Type()

	switch {
	case dstType.AssignableTo(src.Type()):
		dst.Set(src)
	case dstType.Kind() == reflect.Array && srcType.Kind() == reflect.Slice:
		if !dstType.Elem().AssignableTo(r_byte) {
			return fmt.Errorf("abi: cannot unmarshal %v in to array of elem %v", src.Type(), dstType.Elem())
		}

		if dst.Len() < output.Type.Size {
			return fmt.Errorf("abi: cannot unmarshal src (len=%d) in to dst (len=%d)", output.Type.Size, dst.Len())
		}
		reflect.Copy(dst, src)
	default:
		return fmt.Errorf("abi: cannot unmarshal %v in to %v", src.Type(), dst.Type())
	}
	return nil
}

func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type    string
		Name    string
		Const   bool
		Indexed bool
		Inputs  []Argument
		Outputs []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = Method{
				Name:    field.Name,
				Const:   field.Const,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:   field.Name,
				Inputs: field.Inputs,
			}
		}
	}

	return nil
}
