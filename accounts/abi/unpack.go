// Copyright 2017 The go-ethereum Authors
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

var (
	maxUint256 = big.NewInt(0).Add(
		big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil),
		big.NewInt(-1))
	maxInt256 = big.NewInt(0).Add(
		big.NewInt(0).Exp(big.NewInt(2), big.NewInt(255), nil),
		big.NewInt(-1))
)

// reads the integer based on its kind
func readInteger(typ byte, kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint8:
		return b[len(b)-1]
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
		// the only case lefts for integer is int256/uint256.
		// big.SetBytes can't tell if a number is negative, positive on itself.
		// On EVM, if the returned number > max int256, it is negative.
		ret := new(big.Int).SetBytes(b)
		if typ == UintTy {
			return ret
		}

		if ret.Cmp(maxInt256) > 0 {
			ret.Add(maxUint256, big.NewInt(0).Neg(ret))
			ret.Add(ret, big.NewInt(1))
			ret.Neg(ret)
		}
		return ret
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
		return [24]byte{}, fmt.Errorf("abi: invalid type in call to make function type byte array")
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
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array")
	}
	// convert
	array := reflect.New(t.Type).Elem()

	reflect.Copy(array, reflect.ValueOf(word[0:t.Size]))
	return array.Interface(), nil

}

// iteratively unpack elements
func forEachUnpack(t Type, output []byte, start, size int) (interface{}, error) {
	if size < 0 {
		return nil, fmt.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+32*size > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)", len(output), start+32*size)
	}

	// this value will become our slice or our array, depending on the type
	var refSlice reflect.Value

	if t.T == SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(t.Type, size, size)
	} else if t.T == ArrayTy {
		// declare our array
		refSlice = reflect.New(t.Type).Elem()
	} else {
		return nil, fmt.Errorf("abi: invalid type in array/slice unpacking stage")
	}

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := getTypeSize(*t.Elem)

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
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

func forTupleUnpack(t Type, output []byte) (interface{}, error) {
	retval := reflect.New(t.Type).Elem()
	virtualArgs := 0
	for index, elem := range t.TupleElems {
		marshalledValue, err := toGoType((index+virtualArgs)*32, *elem, output)
		if elem.T == ArrayTy && !isDynamicType(*elem) {
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
			virtualArgs += getTypeSize(*elem)/32 - 1
		} else if elem.T == TupleTy && !isDynamicType(*elem) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(*elem)/32 - 1
		}
		if err != nil {
			return nil, err
		}
		retval.Field(index).Set(reflect.ValueOf(marshalledValue))
	}
	return retval.Interface(), nil
}

// toGoType parses the output bytes and recursively assigns the value of these bytes
// into a go type with accordance with the ABI spec.
func toGoType(index int, t Type, output []byte) (interface{}, error) {
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), index+32)
	}

	var (
		returnOutput  []byte
		begin, length int
		err           error
	)

	// if we require a length prefix, find the beginning word and size returned.
	if t.requiresLengthPrefix() {
		begin, length, err = lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, err
		}
	} else {
		returnOutput = output[index : index+32]
	}

	switch t.T {
	case TupleTy:
		if isDynamicType(t) {
			begin, err := tuplePointsTo(index, output)
			if err != nil {
				return nil, err
			}
			return forTupleUnpack(t, output[begin:])
		} else {
			return forTupleUnpack(t, output[index:])
		}
	case SliceTy:
		return forEachUnpack(t, output[begin:], 0, length)
	case ArrayTy:
		if isDynamicType(*t.Elem) {
			offset := int64(binary.BigEndian.Uint64(returnOutput[len(returnOutput)-8:]))
			return forEachUnpack(t, output[offset:], 0, t.Size)
		}
		return forEachUnpack(t, output[index:], 0, t.Size)
	case StringTy: // variable arrays are written at the end of the return bytes
		return string(output[begin : begin+length]), nil
	case IntTy, UintTy:
		return readInteger(t.T, t.Kind, returnOutput), nil
	case BoolTy:
		return readBool(returnOutput)
	case AddressTy:
		return common.BytesToAddress(returnOutput), nil
	case HashTy:
		return common.BytesToHash(returnOutput), nil
	case BytesTy:
		return output[begin : begin+length], nil
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
	bigOffsetEnd := big.NewInt(0).SetBytes(output[index : index+32])
	bigOffsetEnd.Add(bigOffsetEnd, common.Big32)
	outputLength := big.NewInt(int64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)", bigOffsetEnd, outputLength)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}

	offsetEnd := int(bigOffsetEnd.Uint64())
	lengthBig := big.NewInt(0).SetBytes(output[offsetEnd-32 : offsetEnd])

	totalSize := big.NewInt(0)
	totalSize.Add(totalSize, bigOffsetEnd)
	totalSize.Add(totalSize, lengthBig)
	if totalSize.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi length larger than int64: %v", totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %v require %v", outputLength, totalSize)
	}
	start = int(bigOffsetEnd.Uint64())
	length = int(lengthBig.Uint64())
	return
}

// tuplePointsTo resolves the location reference for dynamic tuple.
func tuplePointsTo(index int, output []byte) (start int, err error) {
	offset := big.NewInt(0).SetBytes(output[index : index+32])
	outputLen := big.NewInt(int64(len(output)))

	if offset.Cmp(big.NewInt(int64(len(output)))) > 0 {
		return 0, fmt.Errorf("abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)", offset, outputLen)
	}
	if offset.BitLen() > 63 {
		return 0, fmt.Errorf("abi offset larger than int64: %v", offset)
	}
	return int(offset.Uint64()), nil
}
