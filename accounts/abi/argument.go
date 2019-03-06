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

type ArgumentMarshaling struct {
	Name       string
	Type       string
	Components []ArgumentMarshaling
	Indexed    bool
}

// UnmarshalJSON implements json.Unmarshaler interface
func (argument *Argument) UnmarshalJSON(data []byte) error {
	var arg ArgumentMarshaling
	err := json.Unmarshal(data, &arg)
	if err != nil {
		return fmt.Errorf("argument json err: %v", err)
	}

	argument.Type, err = NewType(arg.Type, arg.Components)
	if err != nil {
		return err
	}
	argument.Name = arg.Name
	argument.Indexed = arg.Indexed

	return nil
}

// LengthNonIndexed returns the number of arguments when not counting 'indexed' ones. Only events
// can ever have 'indexed' arguments, it should always be false on arguments for method input/output
func (arguments Arguments) LengthNonIndexed() int {
	out := 0
	for _, arg := range arguments {
		if !arg.Indexed {
			out++
		}
	}
	return out
}

// NonIndexed returns the arguments with indexed arguments filtered out
func (arguments Arguments) NonIndexed() Arguments {
	var ret []Argument
	for _, arg := range arguments {
		if !arg.Indexed {
			ret = append(ret, arg)
		}
	}
	return ret
}

// isTuple returns true for non-atomic constructs, like (uint,uint) or uint[]
func (arguments Arguments) isTuple() bool {
	return len(arguments) > 1
}

// Unpack performs the operation hexdata -> Go format
func (arguments Arguments) Unpack(v interface{}, data []byte) error {
	// make sure the passed value is arguments pointer
	if reflect.Ptr != reflect.ValueOf(v).Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}
	marshalledValues, err := arguments.UnpackValues(data)
	if err != nil {
		return err
	}
	if arguments.isTuple() {
		return arguments.unpackTuple(v, marshalledValues)
	}
	return arguments.unpackAtomic(v, marshalledValues[0])
}

// unpack sets the unmarshalled value to go format.
// Note the dst here must be settable.
func unpack(t *Type, dst interface{}, src interface{}) error {
	var (
		dstVal = reflect.ValueOf(dst).Elem()
		srcVal = reflect.ValueOf(src)
	)

	if t.T != TupleTy && !((t.T == SliceTy || t.T == ArrayTy) && t.Elem.T == TupleTy) {
		return set(dstVal, srcVal)
	}

	switch t.T {
	case TupleTy:
		if dstVal.Kind() != reflect.Struct {
			return fmt.Errorf("abi: invalid dst value for unpack, want struct, got %s", dstVal.Kind())
		}
		fieldmap, err := mapArgNamesToStructFields(t.TupleRawNames, dstVal)
		if err != nil {
			return err
		}
		for i, elem := range t.TupleElems {
			fname := fieldmap[t.TupleRawNames[i]]
			field := dstVal.FieldByName(fname)
			if !field.IsValid() {
				return fmt.Errorf("abi: field %s can't found in the given value", t.TupleRawNames[i])
			}
			if err := unpack(elem, field.Addr().Interface(), srcVal.Field(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	case SliceTy:
		if dstVal.Kind() != reflect.Slice {
			return fmt.Errorf("abi: invalid dst value for unpack, want slice, got %s", dstVal.Kind())
		}
		slice := reflect.MakeSlice(dstVal.Type(), srcVal.Len(), srcVal.Len())
		for i := 0; i < slice.Len(); i++ {
			if err := unpack(t.Elem, slice.Index(i).Addr().Interface(), srcVal.Index(i).Interface()); err != nil {
				return err
			}
		}
		dstVal.Set(slice)
	case ArrayTy:
		if dstVal.Kind() != reflect.Array {
			return fmt.Errorf("abi: invalid dst value for unpack, want array, got %s", dstVal.Kind())
		}
		array := reflect.New(dstVal.Type()).Elem()
		for i := 0; i < array.Len(); i++ {
			if err := unpack(t.Elem, array.Index(i).Addr().Interface(), srcVal.Index(i).Interface()); err != nil {
				return err
			}
		}
		dstVal.Set(array)
	}
	return nil
}

// unpackAtomic unpacks ( hexdata -> go ) a single value
func (arguments Arguments) unpackAtomic(v interface{}, marshalledValues interface{}) error {
	if arguments.LengthNonIndexed() == 0 {
		return nil
	}
	argument := arguments.NonIndexed()[0]
	elem := reflect.ValueOf(v).Elem()

	if elem.Kind() == reflect.Struct {
		fieldmap, err := mapArgNamesToStructFields([]string{argument.Name}, elem)
		if err != nil {
			return err
		}
		field := elem.FieldByName(fieldmap[argument.Name])
		if !field.IsValid() {
			return fmt.Errorf("abi: field %s can't be found in the given value", argument.Name)
		}
		return unpack(&argument.Type, field.Addr().Interface(), marshalledValues)
	}
	return unpack(&argument.Type, elem.Addr().Interface(), marshalledValues)
}

// unpackTuple unpacks ( hexdata -> go ) a batch of values.
func (arguments Arguments) unpackTuple(v interface{}, marshalledValues []interface{}) error {
	var (
		value = reflect.ValueOf(v).Elem()
		typ   = value.Type()
		kind  = value.Kind()
	)
	if err := requireUnpackKind(value, typ, kind, arguments); err != nil {
		return err
	}

	// If the interface is a struct, get of abi->struct_field mapping
	var abi2struct map[string]string
	if kind == reflect.Struct {
		var (
			argNames []string
			err      error
		)
		for _, arg := range arguments.NonIndexed() {
			argNames = append(argNames, arg.Name)
		}
		abi2struct, err = mapArgNamesToStructFields(argNames, value)
		if err != nil {
			return err
		}
	}
	for i, arg := range arguments.NonIndexed() {
		switch kind {
		case reflect.Struct:
			field := value.FieldByName(abi2struct[arg.Name])
			if !field.IsValid() {
				return fmt.Errorf("abi: field %s can't be found in the given value", arg.Name)
			}
			if err := unpack(&arg.Type, field.Addr().Interface(), marshalledValues[i]); err != nil {
				return err
			}
		case reflect.Slice, reflect.Array:
			if value.Len() < i {
				return fmt.Errorf("abi: insufficient number of arguments for unpack, want %d, got %d", len(arguments), value.Len())
			}
			v := value.Index(i)
			if err := requireAssignable(v, reflect.ValueOf(marshalledValues[i])); err != nil {
				return err
			}
			if err := unpack(&arg.Type, v.Addr().Interface(), marshalledValues[i]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("abi:[2] cannot unmarshal tuple in to %v", typ)
		}
	}
	return nil

}

// UnpackValues can be used to unpack ABI-encoded hexdata according to the ABI-specification,
// without supplying a struct to unpack into. Instead, this method returns a list containing the
// values. An atomic argument will be a list with one element.
func (arguments Arguments) UnpackValues(data []byte) ([]interface{}, error) {
	retval := make([]interface{}, 0, arguments.LengthNonIndexed())
	virtualArgs := 0
	for index, arg := range arguments.NonIndexed() {
		marshalledValue, err := toGoType((index+virtualArgs)*32, arg.Type, data)
		if arg.Type.T == ArrayTy && !isDynamicType(arg.Type) {
			// If we have a static array, like [3]uint256, these are coded as
			// just like uint256,uint256,uint256.
			// This means that we need to add two 'virtual' arguments when
			// we count the index from now on.
			//
			// Array values nested multiple levels deep are also encoded inline:
			// [2][3]uint256: uint256,uint256,uint256,uint256,uint256,uint256
			//
			// Calculate the full array size to get the correct offset for the next argument.
			// Decrement it by 1, as the normal index increment is still applied.
			virtualArgs += getTypeSize(arg.Type)/32 - 1
		} else if arg.Type.T == TupleTy && !isDynamicType(arg.Type) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(arg.Type)/32 - 1
		}
		if err != nil {
			return nil, err
		}
		retval = append(retval, marshalledValue)
	}
	return retval, nil
}

// PackValues performs the operation Go format -> Hexdata
// It is the semantic opposite of UnpackValues
func (arguments Arguments) PackValues(args []interface{}) ([]byte, error) {
	return arguments.Pack(args...)
}

// Pack performs the operation Go format -> Hexdata
func (arguments Arguments) Pack(args ...interface{}) ([]byte, error) {
	// Make sure arguments match up and pack them
	abiArgs := arguments
	if len(args) != len(abiArgs) {
		return nil, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(abiArgs))
	}
	// variable input is the output appended at the end of packed
	// output. This is used for strings and bytes types input.
	var variableInput []byte

	// input offset is the bytes offset for packed output
	inputOffset := 0
	for _, abiArg := range abiArgs {
		inputOffset += getTypeSize(abiArg.Type)
	}
	var ret []byte
	for i, a := range args {
		input := abiArgs[i]
		// pack the input
		packed, err := input.Type.pack(reflect.ValueOf(a))
		if err != nil {
			return nil, err
		}
		// check for dynamic types
		if isDynamicType(input.Type) {
			// set the offset
			ret = append(ret, packNum(reflect.ValueOf(inputOffset))...)
			// calculate next offset
			inputOffset += len(packed)
			// append to variable input
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

// ToCamelCase converts an under-score string to a camel-case string
func ToCamelCase(input string) string {
	parts := strings.Split(input, "_")
	for i, s := range parts {
		if len(s) > 0 {
			parts[i] = strings.ToUpper(s[:1]) + s[1:]
		}
	}
	return strings.Join(parts, "")
}
