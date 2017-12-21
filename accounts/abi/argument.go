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
	"reflect"
	"strings"
)

// Argument holds the name of the argument and the corresponding type.
// Types are used when packing and testing arguments.
type Argument struct {
	Name    string
	Type    Type
	Indexed bool // indexed is only used by events
}

type Arguments []Argument

// UnmarshalJSON implements json.Unmarshaler interface
func (a *Argument) UnmarshalJSON(data []byte) error {
	var extarg struct {
		Name    string
		Type    string
		Indexed bool
	}
	err := json.Unmarshal(data, &extarg)
	if err != nil {
		return fmt.Errorf("argument json err: %v", err)
	}

	a.Type, err = NewType(extarg.Type)
	if err != nil {
		return err
	}
	a.Name = extarg.Name
	a.Indexed = extarg.Indexed

	return nil
}

func countNonIndexedArguments(args []Argument) int {
	out := 0
	for i := range args {
		if !args[i].Indexed {
			out++
		}
	}
	return out
}
func (a *Arguments) isTuple() bool {
	return a != nil && len(*a) > 1
}

func (a *Arguments) Unpack(v interface{}, data []byte) error {
	if a.isTuple() {
		return a.unpackTuple(v, data)
	}
	return a.unpackAtomic(v, data)
}

func (a *Arguments) unpackTuple(v interface{}, output []byte) error {
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
/* !TODO add this back
	if err := requireUnpackKind(value, typ, kind, (*a), false); err != nil {
		return err
	}
*/
	// `i` counts the nonindexed arguments.
	// `j` counts the number of complex types.
	// both `i` and `j` are used to to correctly compute `data` offset.

	i, j := -1, 0
	for _, arg := range(*a) {

		if arg.Indexed {
			// can't read, continue
			continue
		}
		i++
		marshalledValue, err := toGoType((i+j)*32, arg.Type, output)
		if err != nil {
			return err
		}

		if arg.Type.T == ArrayTy {
			// combined index ('i' + 'j') need to be adjusted only by size of array, thus
			// we need to decrement 'j' because 'i' was incremented
			j += arg.Type.Size - 1
		}

		reflectValue := reflect.ValueOf(marshalledValue)

		switch kind {
		case reflect.Struct:
			for j := 0; j < typ.NumField(); j++ {
				field := typ.Field(j)
				// TODO read tags: `abi:"fieldName"`
				if field.Name == strings.ToUpper(arg.Name[:1])+arg.Name[1:] {
					if err := set(value.Field(j), reflectValue, arg); err != nil {
						return err
					}
				}
			}
		case reflect.Slice, reflect.Array:
			if value.Len() < i {
				return fmt.Errorf("abi: insufficient number of arguments for unpack, want %d, got %d", len(*a), value.Len())
			}
			v := value.Index(i)
			if err := requireAssignable(v, reflectValue); err != nil {
				return err
			}
			reflectValue := reflect.ValueOf(marshalledValue)
			return set(v.Elem(), reflectValue, arg)
		default:
			return fmt.Errorf("abi:[2] cannot unmarshal tuple in to %v", typ)
		}
	}
	return nil
}

func (a *Arguments) unpackAtomic(v interface{}, output []byte) error {
	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}
	arg := (*a)[0]
	if arg.Indexed {
		return fmt.Errorf("abi: attempting to unpack indexed variable into element.")
	}

	value := valueOf.Elem()

	marshalledValue, err := toGoType(0, arg.Type, output)
	if err != nil {
		return err
	}
	if err := set(value, reflect.ValueOf(marshalledValue), arg); err != nil {
		return err
	}
	return nil
}

func (arguments *Arguments) Pack(args ...interface{}) ([]byte, error) {
	// Make sure arguments match up and pack them
	if arguments == nil {
		return nil, fmt.Errorf("arguments are nil, programmer error!")
	}

	abiArgs := *arguments
	if len(args) != len(abiArgs) {
		return nil, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(abiArgs))
	}

	// variable input is the output appended at the end of packed
	// output. This is used for strings and bytes types input.
	var variableInput []byte

	// input offset is the bytes offset for packed output
	inputOffset := 0
	for _, abiArg := range abiArgs {
		if abiArg.Type.T == ArrayTy {
			inputOffset += (32 * abiArg.Type.Size)
		} else {
			inputOffset += 32
		}
	}

	var ret []byte
	for i, a := range args {
		input := abiArgs[i]
		// pack the input
		packed, err := input.Type.pack(reflect.ValueOf(a))
		if err != nil {
			return nil, err
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
