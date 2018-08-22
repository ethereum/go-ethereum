// Copyright 2016 The go-ethereum Authors
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
)

// indirect recursively dereferences the value until it either gets the value
// or finds a big.Int
func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr && v.Elem().Type() != derefbigT {
		return indirect(v.Elem())
	}
	return v
}

// reflectIntKind returns the reflect using the given size and
// unsignedness.
func reflectIntKindAndType(unsigned bool, size int) (reflect.Kind, reflect.Type) {
	switch size {
	case 8:
		if unsigned {
			return reflect.Uint8, uint8T
		}
		return reflect.Int8, int8T
	case 16:
		if unsigned {
			return reflect.Uint16, uint16T
		}
		return reflect.Int16, int16T
	case 32:
		if unsigned {
			return reflect.Uint32, uint32T
		}
		return reflect.Int32, int32T
	case 64:
		if unsigned {
			return reflect.Uint64, uint64T
		}
		return reflect.Int64, int64T
	}
	return reflect.Ptr, bigT
}

// mustArrayToBytesSlice creates a new byte slice with the exact same size as value
// and copies the bytes in value to the new slice.
func mustArrayToByteSlice(value reflect.Value) reflect.Value {
	slice := reflect.MakeSlice(reflect.TypeOf([]byte{}), value.Len(), value.Len())
	reflect.Copy(slice, value)
	return slice
}

// set attempts to assign src to dst by either setting, copying or otherwise.
//
// set is a bit more lenient when it comes to assignment and doesn't force an as
// strict ruleset as bare `reflect` does.
func set(dst, src reflect.Value, output Argument) error {
	dstType := dst.Type()
	srcType := src.Type()
	switch {
	case dstType.AssignableTo(srcType):
		dst.Set(src)
	case dstType.Kind() == reflect.Interface:
		dst.Set(src)
	case dstType.Kind() == reflect.Ptr:
		return set(dst.Elem(), src, output)
	default:
		return fmt.Errorf("abi: cannot unmarshal %v in to %v", src.Type(), dst.Type())
	}
	return nil
}

// requireAssignable assures that `dest` is a pointer and it's not an interface.
func requireAssignable(dst, src reflect.Value) error {
	if dst.Kind() != reflect.Ptr && dst.Kind() != reflect.Interface {
		return fmt.Errorf("abi: cannot unmarshal %v into %v", src.Type(), dst.Type())
	}
	return nil
}

// requireUnpackKind verifies preconditions for unpacking `args` into `kind`
func requireUnpackKind(v reflect.Value, t reflect.Type, k reflect.Kind,
	args Arguments) error {

	switch k {
	case reflect.Struct:
	case reflect.Slice, reflect.Array:
		if minLen := args.LengthNonIndexed(); v.Len() < minLen {
			return fmt.Errorf("abi: insufficient number of elements in the list/array for unpack, want %d, got %d",
				minLen, v.Len())
		}
	default:
		return fmt.Errorf("abi: cannot unmarshal tuple into %v", t)
	}
	return nil
}

// mapAbiToStringField maps abi to struct fields.
// first round: for each Exportable field that contains a `abi:""` tag
//   and this field name exists in the arguments, pair them together.
// second round: for each argument field that has not been already linked,
//   find what variable is expected to be mapped into, if it exists and has not been
//   used, pair them.
func mapAbiToStructFields(args Arguments, value reflect.Value) (map[string]string, error) {

	typ := value.Type()

	abi2struct := make(map[string]string)
	struct2abi := make(map[string]string)

	// first round ~~~
	for i := 0; i < typ.NumField(); i++ {
		structFieldName := typ.Field(i).Name

		// skip private struct fields.
		if structFieldName[:1] != strings.ToUpper(structFieldName[:1]) {
			continue
		}

		// skip fields that have no abi:"" tag.
		var ok bool
		var tagName string
		if tagName, ok = typ.Field(i).Tag.Lookup("abi"); !ok {
			continue
		}

		// check if tag is empty.
		if tagName == "" {
			return nil, fmt.Errorf("struct: abi tag in '%s' is empty", structFieldName)
		}

		// check which argument field matches with the abi tag.
		found := false
		for _, abiField := range args.NonIndexed() {
			if abiField.Name == tagName {
				if abi2struct[abiField.Name] != "" {
					return nil, fmt.Errorf("struct: abi tag in '%s' already mapped", structFieldName)
				}
				// pair them
				abi2struct[abiField.Name] = structFieldName
				struct2abi[structFieldName] = abiField.Name
				found = true
			}
		}

		// check if this tag has been mapped.
		if !found {
			return nil, fmt.Errorf("struct: abi tag '%s' defined but not found in abi", tagName)
		}

	}

	// second round ~~~
	for _, arg := range args {

		abiFieldName := arg.Name
		structFieldName := Capitalise(abiFieldName)

		if structFieldName == "" {
			return nil, fmt.Errorf("abi: purely underscored output cannot unpack to struct")
		}

		// this abi has already been paired, skip it... unless there exists another, yet unassigned
		// struct field with the same field name. If so, raise an error:
		//    abi: [ { "name": "value" } ]
		//    struct { Value  *big.Int , Value1 *big.Int `abi:"value"`}
		if abi2struct[abiFieldName] != "" {
			if abi2struct[abiFieldName] != structFieldName &&
				struct2abi[structFieldName] == "" &&
				value.FieldByName(structFieldName).IsValid() {
				return nil, fmt.Errorf("abi: multiple variables maps to the same abi field '%s'", abiFieldName)
			}
			continue
		}

		// return an error if this struct field has already been paired.
		if struct2abi[structFieldName] != "" {
			return nil, fmt.Errorf("abi: multiple outputs mapping to the same struct field '%s'", structFieldName)
		}

		if value.FieldByName(structFieldName).IsValid() {
			// pair them
			abi2struct[abiFieldName] = structFieldName
			struct2abi[structFieldName] = abiFieldName
		} else {
			// not paired, but annotate as used, to detect cases like
			//   abi : [ { "name": "value" }, { "name": "_value" } ]
			//   struct { Value *big.Int }
			struct2abi[structFieldName] = abiFieldName
		}

	}

	return abi2struct, nil
}
