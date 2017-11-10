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
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// Callable method given a `Name` and whether the method is a constant.
// If the method is `Const` no transaction needs to be created for this
// particular Method call. It can easily be simulated using a local VM.
// For example a `Balance()` method only needs to retrieve something
// from the storage and therefor requires no Tx to be send to the
// network. A method such as `Transact` does require a Tx and thus will
// be flagged `true`.
// Input specifies the required input parameters for this gives method.
type Method struct {
	Name    string
	Const   bool
	Inputs  []Argument
	Outputs []Argument
}

func (method Method) pack(args ...interface{}) ([]byte, error) {
	// Make sure arguments match up and pack them
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(method.Inputs))
	}
	// variable input is the output appended at the end of packed
	// output. This is used for strings and bytes types input.
	var variableInput []byte

	// input offset is the bytes offset for packed output
	inputOffset := 0
	for _, input := range method.Inputs {
		if input.Type.T == ArrayTy {
			inputOffset += (32 * input.Type.Size)
		} else {
			inputOffset += 32
		}
	}

	var ret []byte
	for i, a := range args {
		input := method.Inputs[i]
		// pack the input
		packed, err := input.Type.pack(reflect.ValueOf(a))
		if err != nil {
			return nil, fmt.Errorf("`%s` %v", method.Name, err)
		}

		// check for a slice type (string, bytes, slice)
		if input.Type.requiresLengthPrefix() {
			// calculate the offset
			offset := inputOffset + len(variableInput)

			// set the offset
			ret = append(ret, packNum(reflect.ValueOf(offset))...)
			// Append the packed output to the variable input. The variable input
			// will be appended at the end of the input.
			variableInput = append(variableInput, packed...)
		} else {
			// append the packed value to the input
			ret = append(ret, packed...)
		}
	}
	// append the variable input at the end of the packed input
	ret = append(ret, variableInput...)

	return ret, nil
}

// unpacks a method return tuple into a struct of corresponding go types
//
// Unpacking can be done into a struct or a slice/array.
func (method Method) tupleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
		kind  = value.Kind()
	)
	if err := requireUnpackKind(value, typ, kind, method.Outputs, false); err != nil {
		return err
	}

	j := 0
	for i := 0; i < len(method.Outputs); i++ {
		toUnpack := method.Outputs[i]
		marshalledValue, err := toGoType((i+j)*32, toUnpack.Type, output)
		if err != nil {
			return err
		}
		if toUnpack.Type.T == ArrayTy {
			// combined index ('i' + 'j') need to be adjusted only by size of array, thus
			// we need to decrement 'j' because 'i' was incremented
			j += toUnpack.Type.Size - 1
		}
		reflectValue := reflect.ValueOf(marshalledValue)

		switch kind {
		case reflect.Struct:
			for j := 0; j < typ.NumField(); j++ {
				field := typ.Field(j)
				// TODO read tags: `abi:"fieldName"`
				if field.Name == strings.ToUpper(method.Outputs[i].Name[:1])+method.Outputs[i].Name[1:] {
					if err := set(value.Field(j), reflectValue, method.Outputs[i]); err != nil {
						return err
					}
				}
			}
		case reflect.Slice, reflect.Array:
			v := value.Index(i)
			if err := requireAssignable(v, reflectValue); err != nil {
				return err
			}
			if err := set(v.Elem(), reflectValue, method.Outputs[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (method Method) isTupleReturn() bool { return len(method.Outputs) > 1 }

func (method Method) singleUnpack(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	value := valueOf.Elem()

	marshalledValue, err := toGoType(0, method.Outputs[0].Type, output)
	if err != nil {
		return err
	}
	if err := set(value, reflect.ValueOf(marshalledValue), method.Outputs[0]); err != nil {
		return err
	}
	return nil
}

// Sig returns the methods string signature according to the ABI spec.
//
// Example
//
//     function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (m Method) Sig() string {
	types := make([]string, len(m.Inputs))
	i := 0
	for _, input := range m.Inputs {
		types[i] = input.Type.String()
		i++
	}
	return fmt.Sprintf("%v(%v)", m.Name, strings.Join(types, ","))
}

func (m Method) String() string {
	inputs := make([]string, len(m.Inputs))
	for i, input := range m.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Name, input.Type)
	}
	outputs := make([]string, len(m.Outputs))
	for i, output := range m.Outputs {
		if len(output.Name) > 0 {
			outputs[i] = fmt.Sprintf("%v ", output.Name)
		}
		outputs[i] += output.Type.String()
	}
	constant := ""
	if m.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", m.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

func (m Method) Id() []byte {
	return crypto.Keccak256([]byte(m.Sig()))[:4]
}
