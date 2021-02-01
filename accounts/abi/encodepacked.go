package abi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

// EncodePacked implements the non-standard encoding available in solidity.
// Since encoding is ambigious there is no decoding function.
// See
func EncodePacked(args []Type, values []interface{}) ([]byte, error) {
	enc := make([]byte, 0)
	for idx, arg := range args {
		switch arg.T {
		case TupleTy:
			return []byte{}, errors.New("Not implemented")
		case ArrayTy, SliceTy:
		default:
			packed, err := encodePackElement(arg, reflect.ValueOf(values[idx]))
			if err != nil {
				return []byte{}, err
			}
			enc = append(enc, packed...)
		}
	}
	return enc, nil
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
		panic("abi: fatal error")
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
