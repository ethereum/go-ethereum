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

	"github.com/ethereum/go-ethereum/common"
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
	IsSlice   bool
	SliceSize int

	Kind       reflect.Kind
	Type       reflect.Type
	Size       int
	T          byte   // Our own type checking
	stringKind string // holds the unparsed string for deriving signatures
}

var (
	fullTypeRegex = regexp.MustCompile("([a-zA-Z0-9]+)(\\[([0-9]*)?\\])?")
	typeRegex     = regexp.MustCompile("([a-zA-Z]+)([0-9]*)?")
)

// NewType returns a fully parsed Type given by the input string or an error if it  can't be parsed.
//
// Strings can be in the format of:
//
// 	Input  = Type [ "[" [ Number ] "]" ] Name .
// 	Type   = [ "u" ] "int" [ Number ] .
//
// Examples:
//
//      string     int       uint       real
//      string32   int8      uint8      uint[]
//      address    int256    uint256    real[2]
func NewType(t string) (typ Type, err error) {
	// 1. full string 2. type 3. (opt.) is slice 4. (opt.) size
	// parse the full representation of the abi-type definition; including:
	// * full string
	// * type
	// 	* is slice
	//	* slice size
	res := fullTypeRegex.FindAllStringSubmatch(t, -1)[0]

	// check if type is slice and parse type.
	switch {
	case res[3] != "":
		// err is ignored. Already checked for number through the regexp
		typ.SliceSize, _ = strconv.Atoi(res[3])
		typ.IsSlice = true
	case res[2] != "":
		typ.IsSlice, typ.SliceSize = true, -1
	case res[0] == "":
		return Type{}, fmt.Errorf("abi: type parse error: %s", t)
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

	switch varType {
	case "int":
		typ.Kind = reflect.Int
		typ.Type = big_t
		typ.Size = varSize
		typ.T = IntTy
	case "uint":
		typ.Kind = reflect.Uint
		typ.Type = ubig_t
		typ.Size = varSize
		typ.T = UintTy
	case "bool":
		typ.Kind = reflect.Bool
		typ.T = BoolTy
	case "real": // TODO
		typ.Kind = reflect.Invalid
	case "address":
		typ.Type = address_t
		typ.Size = 20
		typ.T = AddressTy
	case "string":
		typ.Kind = reflect.String
		typ.Size = -1
		typ.T = StringTy
		if varSize > 0 {
			typ.Size = 32
		}
	case "hash":
		typ.Kind = reflect.Array
		typ.Size = 32
		typ.Type = hash_t
		typ.T = HashTy
	case "bytes":
		typ.Kind = reflect.Array
		typ.Type = byte_ts
		typ.Size = varSize
		if varSize == 0 {
			typ.T = BytesTy
		} else {
			typ.T = FixedBytesTy
		}
	default:
		return Type{}, fmt.Errorf("unsupported arg type: %s", t)
	}
	typ.stringKind = t

	return
}

func (t Type) String() (out string) {
	return t.stringKind
}

// packBytesSlice packs the given bytes as [L, V] as the canonical representation
// bytes slice
func packBytesSlice(bytes []byte, l int) []byte {
	len := packNum(reflect.ValueOf(l), UintTy)
	return append(len, common.RightPadBytes(bytes, (l+31)/32*32)...)
}

// Test the given input parameter `v` and checks if it matches certain
// criteria
// * Big integers are checks for ptr types and if the given value is
//   assignable
// * Integer are checked for size
// * Strings, addresses and bytes are checks for type and size
func (t Type) pack(v interface{}) ([]byte, error) {
	value := reflect.ValueOf(v)
	switch kind := value.Kind(); kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// check input is unsigned
		if t.Type != ubig_t {
			return nil, fmt.Errorf("abi: type mismatch: %s for %T", t.Type, v)
		}

		// no implicit type casting
		if int(value.Type().Size()*8) != t.Size {
			return nil, fmt.Errorf("abi: cannot use type %T as type uint%d", v, t.Size)
		}

		return packNum(value, t.T), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t.Type != ubig_t {
			return nil, fmt.Errorf("type mismatch: %s for %T", t.Type, v)
		}

		// no implicit type casting
		if int(value.Type().Size()*8) != t.Size {
			return nil, fmt.Errorf("abi: cannot use type %T as type uint%d", v, t.Size)
		}
		return packNum(value, t.T), nil
	case reflect.Ptr:
		// If the value is a ptr do a assign check (only used by
		// big.Int for now)
		if t.Type == ubig_t && value.Type() != ubig_t {
			return nil, fmt.Errorf("type mismatch: %s for %T", t.Type, v)
		}
		return packNum(value, t.T), nil
	case reflect.String:
		if t.Size > -1 && value.Len() > t.Size {
			return nil, fmt.Errorf("%v out of bound. %d for %d", value.Kind(), value.Len(), t.Size)
		}

		return packBytesSlice([]byte(value.String()), value.Len()), nil
	case reflect.Slice:
		// Byte slice is a special case, it gets treated as a single value
		if t.T == BytesTy {
			return packBytesSlice(value.Bytes(), value.Len()), nil
		}

		if t.SliceSize > -1 && value.Len() > t.SliceSize {
			return nil, fmt.Errorf("%v out of bound. %d for %d", value.Kind(), value.Len(), t.Size)
		}

		// Signed / Unsigned check
		if value.Type() == big_t && (t.T != IntTy && isSigned(value)) || (t.T == UintTy && isSigned(value)) {
			return nil, fmt.Errorf("slice of incompatible types.")
		}

		var packed []byte
		for i := 0; i < value.Len(); i++ {
			val, err := t.pack(value.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			packed = append(packed, val...)
		}
		return packBytesSlice(packed, value.Len()), nil
	case reflect.Bool:
		if value.Bool() {
			return common.LeftPadBytes(common.Big1.Bytes(), 32), nil
		} else {
			return common.LeftPadBytes(common.Big0.Bytes(), 32), nil
		}
	case reflect.Array:
		if v, ok := value.Interface().(common.Address); ok {
			return common.LeftPadBytes(v[:], 32), nil
		} else if v, ok := value.Interface().(common.Hash); ok {
			return v[:], nil
		}
	}

	return nil, fmt.Errorf("ABI: bad input given %v", value.Kind())
}
