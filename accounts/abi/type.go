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
	"regexp"
	"strconv"
)

const (
	IntTy byte = iota
	UintTy
	BoolTy
	StringTy
	SliceTy
	AddressTy
	FixedBytesTy
	BytesTy
	HashTy
	RealTy
)

// Type is the reflection of the supported argument type
type Type struct {
	IsSlice, IsArray bool
	SliceSize        int

	Elem *Type

	Kind reflect.Kind
	Type reflect.Type
	Size int
	T    byte // Our own type checking

	stringKind string // holds the unparsed string for deriving signatures
}

var (
	// fullTypeRegex parses the abi types
	//
	// Types can be in the format of:
	//
	// 	Input  = Type [ "[" [ Number ] "]" ] Name .
	// 	Type   = [ "u" ] "int" [ Number ] .
	//
	// Examples:
	//
	//      string     int       uint       real
	//      string32   int8      uint8      uint[]
	//      address    int256    uint256    real[2]
	fullTypeRegex = regexp.MustCompile("([a-zA-Z0-9]+)(\\[([0-9]*)?\\])?")
	// typeRegex parses the abi sub types
	typeRegex = regexp.MustCompile("([a-zA-Z]+)([0-9]*)?")
)

// NewType creates a new reflection type of abi type given in t.
func NewType(t string) (typ Type, err error) {
	res := fullTypeRegex.FindAllStringSubmatch(t, -1)[0]
	// check if type is slice and parse type.
	switch {
	case res[3] != "":
		// err is ignored. Already checked for number through the regexp
		typ.SliceSize, _ = strconv.Atoi(res[3])
		typ.IsArray = true
	case res[2] != "":
		typ.IsSlice, typ.SliceSize = true, -1
	case res[0] == "":
		return Type{}, fmt.Errorf("abi: type parse error: %s", t)
	}
	if typ.IsArray || typ.IsSlice {
		sliceType, err := NewType(res[1])
		if err != nil {
			return Type{}, err
		}
		typ.Elem = &sliceType
		typ.stringKind = sliceType.stringKind + t[len(res[1]):]
		return typ, nil
	}

	// parse the type and size of the abi-type.
	parsedType := typeRegex.FindAllStringSubmatch(res[1], -1)[0]
	// varSize is the size of the variable
	var varSize int
	if len(parsedType[2]) > 0 {
		var err error
		varSize, err = strconv.Atoi(parsedType[2])
		if err != nil {
			return Type{}, fmt.Errorf("abi: error parsing variable size: %v", err)
		}
	}
	// varType is the parsed abi type
	varType := parsedType[1]
	// substitute canonical integer
	if varSize == 0 && (varType == "int" || varType == "uint") {
		varSize = 256
		t += "256"
	}
	typ.stringKind = t

	switch varType {
	case "int":
		typ.Kind = reflectIntKind(false, varSize)
		typ.Type = big_t
		typ.Size = varSize
		typ.T = IntTy
	case "uint":
		typ.Kind = reflectIntKind(true, varSize)
		typ.Type = ubig_t
		typ.Size = varSize
		typ.T = UintTy
	case "bool":
		typ.Kind = reflect.Bool
		typ.T = BoolTy
	case "address":
		typ.Kind = reflect.Array
		typ.Type = address_t
		typ.Size = 20
		typ.T = AddressTy
	case "string":
		typ.Kind = reflect.String
		typ.Size = -1
		typ.T = StringTy
	case "bytes":
		sliceType, _ := NewType("uint8")
		typ.Elem = &sliceType
		if varSize == 0 {
			typ.IsSlice = true
			typ.T = BytesTy
			typ.SliceSize = -1
		} else {
			typ.IsArray = true
			typ.T = FixedBytesTy
			typ.SliceSize = varSize
		}
	default:
		return Type{}, fmt.Errorf("unsupported arg type: %s", t)
	}

	return
}

// String implements Stringer
func (t Type) String() (out string) {
	return t.stringKind
}

func (t Type) pack(v reflect.Value) ([]byte, error) {
	// dereference pointer first if it's a pointer
	v = indirect(v)

	if err := typeCheck(t, v); err != nil {
		return nil, err
	}

	if (t.IsSlice || t.IsArray) && t.T != BytesTy && t.T != FixedBytesTy {
		var packed []byte
		for i := 0; i < v.Len(); i++ {
			val, err := t.Elem.pack(v.Index(i))
			if err != nil {
				return nil, err
			}
			packed = append(packed, val...)
		}
		return packBytesSlice(packed, v.Len()), nil
	}

	return packElement(t, v), nil
}

// requireLengthPrefix returns whether the type requires any sort of length
// prefixing.
func (t Type) requiresLengthPrefix() bool {
	return t.T != FixedBytesTy && (t.T == StringTy || t.T == BytesTy || t.IsSlice || t.IsArray)
}
