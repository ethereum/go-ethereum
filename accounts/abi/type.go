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
	SliceTy
	AddressTy
	HashTy
	RealTy
)

// Type is the reflection of the supported argument type
type Type struct {
	Kind       reflect.Kind
	Type       reflect.Type
	Size       int
	T          byte   // Our own type checking
	stringKind string // holds the unparsed string for deriving signatures
}

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
	freg, err := regexp.Compile("([a-zA-Z0-9]+)(\\[([0-9]*)?\\])?")
	if err != nil {
		return Type{}, err
	}
	res := freg.FindAllStringSubmatch(t, -1)[0]
	var (
		isslice bool
		size    int
	)
	switch {
	case res[3] != "":
		// err is ignored. Already checked for number through the regexp
		size, _ = strconv.Atoi(res[3])
		isslice = true
	case res[2] != "":
		isslice = true
		size = -1
	case res[0] == "":
		return Type{}, fmt.Errorf("type parse error for `%s`", t)
	}

	treg, err := regexp.Compile("([a-zA-Z]+)([0-9]*)?")
	if err != nil {
		return Type{}, err
	}

	parsedType := treg.FindAllStringSubmatch(res[1], -1)[0]
	vsize, _ := strconv.Atoi(parsedType[2])
	vtype := parsedType[1]
	// substitute canonical representation
	if vsize == 0 && (vtype == "int" || vtype == "uint") {
		vsize = 256
		t += "256"
	}

	if isslice {
		typ.Kind = reflect.Slice
		typ.Size = size
		switch vtype {
		case "int":
			typ.Type = big_ts
		case "uint":
			typ.Type = ubig_ts
		default:
			return Type{}, fmt.Errorf("unsupported arg slice type: %s", t)
		}
	} else {
		switch vtype {
		case "int":
			typ.Kind = reflect.Ptr
			typ.Type = big_t
			typ.Size = 256
			typ.T = IntTy
		case "uint":
			typ.Kind = reflect.Ptr
			typ.Type = ubig_t
			typ.Size = 256
			typ.T = UintTy
		case "bool":
			typ.Kind = reflect.Bool
		case "real": // TODO
			typ.Kind = reflect.Invalid
		case "address":
			typ.Kind = reflect.Slice
			typ.Type = address_t
			typ.Size = 20
			typ.T = AddressTy
		case "string":
			typ.Kind = reflect.String
			typ.Size = -1
			if vsize > 0 {
				typ.Size = 32
			}
		case "hash":
			typ.Kind = reflect.Slice
			typ.Size = 32
			typ.Type = hash_t
			typ.T = HashTy
		case "bytes":
			typ.Kind = reflect.Slice
			typ.Type = byte_ts
			typ.Size = vsize
		default:
			return Type{}, fmt.Errorf("unsupported arg type: %s", t)
		}
	}
	typ.stringKind = t

	return
}

func (t Type) String() (out string) {
	return t.stringKind
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
		if t.Type != ubig_t {
			return nil, fmt.Errorf("type mismatch: %s for %T", t.Type, v)
		}
		return packNum(value, t.T), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t.Type != ubig_t {
			return nil, fmt.Errorf("type mismatch: %s for %T", t.Type, v)
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
		return []byte(common.LeftPadString(t.String(), 32)), nil
	case reflect.Slice:
		if t.Size > -1 && value.Len() > t.Size {
			return nil, fmt.Errorf("%v out of bound. %d for %d", value.Kind(), value.Len(), t.Size)
		}

		// Address is a special slice. The slice acts as one rather than a list of elements.
		if t.T == AddressTy {
			return common.LeftPadBytes(v.([]byte), 32), nil
		}

		// Signed / Unsigned check
		if (t.T != IntTy && isSigned(value)) || (t.T == UintTy && isSigned(value)) {
			return nil, fmt.Errorf("slice of incompatible types.")
		}

		var packed []byte
		for i := 0; i < value.Len(); i++ {
			packed = append(packed, packNum(value.Index(i), t.T)...)
		}
		return packed, nil
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

	return nil, fmt.Errorf("ABI: bad input given %T", value.Kind())
}
