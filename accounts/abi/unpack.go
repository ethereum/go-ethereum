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
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

// unpacker is a utility interface that enables us to have
// abstraction between events and methods and also to properly
// "unpack" them; e.g. events use Inputs, methods use Outputs.
type unpacker interface {
	tupleUnpack(v interface{}, output []byte) error
	singleUnpack(v interface{}, output []byte) error
	isTupleReturn() bool
}

// reads the integer based on its kind
func readInteger(kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint8:
		return uint8(b[len(b)-1])
	case reflect.Uint16:
		return binary.BigEndian.Uint16(b[len(b)-2:])
	case reflect.Uint32:
		return binary.BigEndian.Uint32(b[len(b)-4:])
	case reflect.Uint64:
		return binary.BigEndian.Uint64(b[len(b)-8:])
	case reflect.Int8:
		return int8(b[len(b)-1])
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
	default:
		return new(big.Int).SetBytes(b)
	}
}

// reads a bool
func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errBadBool
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errBadBool
	}
}

// A function type is simply the address with the function selection signature at the end.
// This enforces that standard by always presenting it as a 24-array (address + sig = 24 bytes)
func readFunctionType(t Type, word []byte) (funcTy [24]byte, err error) {
	if t.T != FunctionTy {
		return [24]byte{}, fmt.Errorf("abi: invalid type in call to make function type byte array.")
	}
	if garbage := binary.BigEndian.Uint64(word[24:32]); garbage != 0 {
		err = fmt.Errorf("abi: got improperly encoded function type, got %v", word)
	} else {
		copy(funcTy[:], word[0:24])
	}
	return
}

// through reflection, creates a fixed array to be read from
func readFixedBytes(t Type, word []byte) (interface{}, error) {
	if t.T != FixedBytesTy {
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array.")
	}
	// convert
	array := reflect.New(t.Type).Elem()

	reflect.Copy(array, reflect.ValueOf(word[0:t.Size]))
	return array.Interface(), nil

}

// iteratively unpack elements
func forEachUnpack(t Type, output []byte, start, size int) (interface{}, error) {
	if start+32*size > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)", len(output), start+32*size)
	}

	// this value will become our slice or our array, depending on the type
	var refSlice reflect.Value
	slice := output[start : start+size*32]

	if t.T == SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(t.Type, size, size)
	} else if t.T == ArrayTy {
		// declare our array
		refSlice = reflect.New(t.Type).Elem()
	} else {
		return nil, fmt.Errorf("abi: invalid type in array/slice unpacking stage")
	}

	for i, j := start, 0; j*32 < len(slice); i, j = i+32, j+1 {
		// this corrects the arrangement so that we get all the underlying array values
		if t.Elem.T == ArrayTy && j != 0 {
			i = start + t.Elem.Size*32*j
		}
		inter, err := toGoType(i, *t.Elem, output)
		if err != nil {
			return nil, err
		}
		// append the item to our reflect slice
		refSlice.Index(j).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

// toGoType parses the output bytes and recursively assigns the value of these bytes
// into a go type with accordance with the ABI spec.
func toGoType(index int, t Type, output []byte) (interface{}, error) {
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), index+32)
	}

	var (
		returnOutput []byte
		begin, end   int
		err          error
	)

	// if we require a length prefix, find the beginning word and size returned.
	if t.requiresLengthPrefix() {
		begin, end, err = lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, err
		}
	} else {
		returnOutput = output[index : index+32]
	}

	switch t.T {
	case SliceTy:
		return forEachUnpack(t, output, begin, end)
	case ArrayTy:
		return forEachUnpack(t, output, index, t.Size)
	case StringTy: // variable arrays are written at the end of the return bytes
		return string(output[begin : begin+end]), nil
	case IntTy, UintTy:
		return readInteger(t.Kind, returnOutput), nil
	case BoolTy:
		return readBool(returnOutput)
	case AddressTy:
		return common.BytesToAddress(returnOutput), nil
	case HashTy:
		return common.BytesToHash(returnOutput), nil
	case BytesTy:
		return output[begin : begin+end], nil
	case FixedBytesTy:
		return readFixedBytes(t, returnOutput)
	case FunctionTy:
		return readFunctionType(t, returnOutput)
	default:
		return nil, fmt.Errorf("abi: unknown type %v", t.T)
	}
}

// interprets a 32 byte slice as an offset and then determines which indice to look to decode the type.
func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	offset := int(binary.BigEndian.Uint64(output[index+24 : index+32]))
	if offset+32 > len(output) {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go slice: offset %d would go over slice boundary (len=%d)", len(output), offset+32)
	}
	length = int(binary.BigEndian.Uint64(output[offset+24 : offset+32]))
	if offset+32+length > len(output) {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32+length)
	}
	start = offset + 32

	//fmt.Printf("LENGTH PREFIX INFO: \nsize: %v\noffset: %v\nstart: %v\n", length, offset, start)
	return
}

// checks for proper formatting of byte output
func bytesAreProper(output []byte) error {
	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	} else if len(output)%32 != 0 {
		return fmt.Errorf("abi: improperly formatted output")
	} else {
		return nil
	}
}
