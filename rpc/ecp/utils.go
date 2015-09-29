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

package ecp

import (
	"math/big"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// Implements this type the error interface
func isErrorType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Implements(errorType)
}

func suitableCallbacks(typ reflect.Type) callbacks {
	callbacks := make(callbacks)
METHODS:
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		if method.PkgPath != "" { // method must be exported
			continue
		}

		var h callback
		h.method = method
		numIn := mtype.NumIn()

		// determine arguments, ignore first arg since it's the receiver type
		h.args = make([]reflect.Type, numIn-1)
		for i := 1; i < numIn; i++ {
			argType := mtype.In(i)
			if !isExportedOrBuiltinType(argType) {
				continue METHODS
			}
			h.args[i-1] = argType
		}

		// determine if callback can return an error
		h.errPos = -1
		for i := 0; i < mtype.NumOut(); i++ {
			if isErrorType(mtype.Out(i)) {
				h.errPos = i
				break
			}
		}

		// only support methods which return an error as the last returned value (or no error).
		if h.errPos == -1 || (h.errPos >= 0 && h.errPos == mtype.NumOut()-1) {
			callbacks[mname] = &h
		}
	}

	return callbacks
}

var bigIntType = reflect.TypeOf(new(big.Int))

func isBigInt(typ reflect.Type) bool {
	return typ == bigIntType
}

// in the future we might consider adding support for more conversion as the API gets defined
func convert(pos int, val interface{}, to reflect.Type) (reflect.Value, error) {
	from := reflect.TypeOf(val)
	if from.Kind() == to.Kind() {
		return reflect.ValueOf(val), nil
	}

	plainType := to
	for plainType.Kind() == reflect.Ptr {
		plainType = plainType.Elem()
	}

	if plainType.Kind() == reflect.Struct {
		return convertStruct(pos, val, to, plainType)
	}

	retPtr := reflect.New(plainType)
	if from.ConvertibleTo(plainType) {
		ret := reflect.Indirect(retPtr)
		ret.Set(reflect.ValueOf(val).Convert(plainType))
		if to.Kind() == reflect.Ptr {
			return retPtr, nil
		}
		return ret, nil
	}

	if bytes, ok := val.([]byte); ok && to.Kind() == reflect.Int64 {
		if i, err := strconv.ParseInt(string(bytes), 10, 64); err == nil {
			return reflect.ValueOf(i), nil
		}
	}

	if bytes, ok := val.([]byte); ok && isBigInt(to) {
		v := new(big.Int)
		v.SetBytes(bytes)
		return reflect.ValueOf(v), nil
	}

	return reflect.Zero(to), &invalidArgumentError{1 + pos, to, reflect.TypeOf(val)}
}

func convertStruct(pos int, val interface{}, to, plainType reflect.Type) (reflect.Value, error) {
	if fields, ok := val.([]*message); ok {
		if len(fields) != plainType.NumField() {
			return reflect.Zero(to), &invalidStructArgumentError{to, plainType.NumField(), len(fields)}
		}

		ptrVal := reflect.New(plainType)
		v := reflect.Indirect(ptrVal)
		for i := 0; i < v.NumField(); i++ {
			val, err := convert(pos, fields[i].Val, v.Field(i).Type())
			if err != nil {
				return reflect.Zero(to), err
			}
			v.Field(i).Set(val)
		}

		if to.Kind() == reflect.Ptr {
			return ptrVal, nil
		}

		return v, nil
	}

	return reflect.Zero(to), &invalidArgumentError{1 + pos, to, reflect.TypeOf(val)}
}
