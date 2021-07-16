package abi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

// SolidityEncodePacked implements the non-standard encoding available in solidity.
// Since encoding is ambigious there is no decoding function.
//
// Using EncodePacked to pack data before hashing or signing is generally unsafe because
// the following arguments produce the same output.
// abi.SolidityEncodePacked([]Type{String,String}, []interface{}{"hello", "world01"})
// abi.SolidityEncodePacked([]Type{String,String}, []interface{}{"helloworld", "01"})
// '0x68656c6c6f776f726c643031'
func SolidityEncodePacked(args []Type, values []interface{}) ([]byte, error) {
	enc := make([]byte, 0)
	var index int
	for _, arg := range args {
		switch arg.T {
		case TupleTy:
			return []byte{}, errors.New("Type not supported in abi.EncodePacked()")
		case ArrayTy, SliceTy:
			packed, err := encodePackArray(arg, values[index:arg.Size])
			if err != nil {
				return []byte{}, err
			}
			enc = append(enc, packed...)
			index += arg.Size
		default:
			packed, err := encodePackElement(arg, reflect.ValueOf(values[index]))
			if err != nil {
				return []byte{}, err
			}
			enc = append(enc, packed...)
			index++
		}
	}
	return enc, nil
}

func encodePackArray(t Type, values []interface{}) ([]byte, error) {
	encoded := make([]byte, 0, t.Size*32)
	for i := 0; i < t.Size; i++ {
		packed, err := encodePackElement(*t.Elem, reflect.ValueOf(values[i]))
		if err != nil {
			return []byte{}, err
		}
		// Array elements are packed with padding
		padded := common.LeftPadBytes(packed, 32)
		encoded = append(encoded, padded...)
	}
	return encoded, nil
}

func encodePackElement(t Type, value reflect.Value) ([]byte, error) {
	value = indirect(value)

	switch t.T {
	case IntTy, UintTy:
		return encodePackedNum(t, value), nil
	case StringTy:
		return encodePackedByteSlice(t, []byte(value.String())), nil
	case AddressTy, FixedBytesTy:
		if value.Kind() == reflect.Array {
			value = mustArrayToByteSlice(value)
		}
		return encodePackedByteSlice(t, value.Bytes()), nil
	case BoolTy:
		if value.Bool() {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case BytesTy:
		if value.Kind() == reflect.Array {
			value = mustArrayToByteSlice(value)
		}
		if value.Type() != reflect.TypeOf([]byte{}) {
			return []byte{}, errors.New("Bytes type is neither slice nor array")
		}
		return encodePackedByteSlice(t, value.Bytes()), nil
	default:
		return []byte{}, fmt.Errorf("Could not encode pack element, unknown type: %v", t.T)
	}
}

func encodePackedNum(t Type, value reflect.Value) []byte {
	bytes := make([]byte, 8)
	switch kind := value.Kind(); kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		binary.BigEndian.PutUint64(bytes, value.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		binary.BigEndian.PutUint64(bytes, uint64(value.Int()))
	case reflect.Ptr:
		big := new(big.Int).Set(value.Interface().(*big.Int))
		bytes = big.Bytes()
	default:
		panic(fmt.Sprintf("abi: fatal error: %v", kind))
	}
	return encodePackedByteSlice(t, bytes)
}

func encodePackedByteSlice(t Type, value []byte) []byte {
	size := t.Size / 8
	// If size is not set in the type, use the length of the value to pad
	if size == 0 {
		size = len(value)
	}
	padded := common.LeftPadBytes(value, size)
	return padded[len(padded)-size:]
}
