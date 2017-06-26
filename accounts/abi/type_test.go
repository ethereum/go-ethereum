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
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// typeWithoutStringer is a alias for the Type type which simply doesn't implement
// the stringer interface to allow printing type details in the tests below.
type typeWithoutStringer Type

// Tests that all allowed types get recognized by the type parser.
func TestTypeRegexp(t *testing.T) {
	tests := []struct {
		blob string
		kind Type
	}{
		{"bool", Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}},
<<<<<<< HEAD
		{"bool[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Bool, T: BoolTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}},
		{"bool[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Bool, T: BoolTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}},
=======
		{"bool[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}},
		{"bool[2]", Type{Size: 2, Kind: reflect.Array, T: ArrayTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}},
		{"bool[2][]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][]"}},
		{"bool[][]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][]"}},
		{"bool[][2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][2]"}},
		{"bool[2][2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][2]"}},
		{"bool[2][][2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][]"}, stringKind: "bool[2][][2]"}},
		{"bool[2][2][2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][2]"}, stringKind: "bool[2][2][2]"}},
		{"bool[][][]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][]"}, stringKind: "bool[][][]"}},
		{"bool[][2][]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Bool, T: BoolTy, stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][2]"}, stringKind: "bool[][2][]"}},
>>>>>>> 50ee5a2... accounts/abi: need to figure out what we're doing with bytes but 2D arrays/slices work now type creation wise.
		{"int8", Type{Kind: reflect.Int8, Type: int8_t, Size: 8, T: IntTy, stringKind: "int8"}},
		{"int16", Type{Kind: reflect.Int16, Type: int16_t, Size: 16, T: IntTy, stringKind: "int16"}},
		{"int32", Type{Kind: reflect.Int32, Type: int32_t, Size: 32, T: IntTy, stringKind: "int32"}},
		{"int64", Type{Kind: reflect.Int64, Type: int64_t, Size: 64, T: IntTy, stringKind: "int64"}},
		{"int256", Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}},
		{"int8[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Int8, Type: int8_t, Size: 8, T: IntTy, stringKind: "int8"}, stringKind: "int8[]"}},
		{"int8[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Int8, Type: int8_t, Size: 8, T: IntTy, stringKind: "int8"}, stringKind: "int8[2]"}},
		{"int16[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Int16, Type: int16_t, Size: 16, T: IntTy, stringKind: "int16"}, stringKind: "int16[]"}},
		{"int16[2]", Type{Size: 2, Kind: reflect.Array, T: ArrayTy, Elem: &Type{Kind: reflect.Int16, Type: int16_t, Size: 16, T: IntTy, stringKind: "int16"}, stringKind: "int16[2]"}},
		{"int32[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Int32, Type: int32_t, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[]"}},
		{"int32[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Int32, Type: int32_t, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[2]"}},
		{"int64[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Int64, Type: int64_t, Size: 64, T: IntTy, stringKind: "int64"}, stringKind: "int64[]"}},
		{"int64[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Int64, Type: int64_t, Size: 64, T: IntTy, stringKind: "int64"}, stringKind: "int64[2]"}},
		{"int256[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[]"}},
		{"int256[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[2]"}},
		{"uint8", Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}},
		{"uint16", Type{Kind: reflect.Uint16, Type: uint16_t, Size: 16, T: UintTy, stringKind: "uint16"}},
		{"uint32", Type{Kind: reflect.Uint32, Type: uint32_t, Size: 32, T: UintTy, stringKind: "uint32"}},
		{"uint64", Type{Kind: reflect.Uint64, Type: uint64_t, Size: 64, T: UintTy, stringKind: "uint64"}},
		{"uint256", Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: UintTy, stringKind: "uint256"}},
		{"uint8[]", Type{Kind: reflect.Slice, T: SliceTy, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "uint8[]"}},
		{"uint8[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "uint8[2]"}},
		{"uint16[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Uint16, Type: uint16_t, Size: 16, T: UintTy, stringKind: "uint16"}, stringKind: "uint16[]"}},
		{"uint16[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Uint16, Type: uint16_t, Size: 16, T: UintTy, stringKind: "uint16"}, stringKind: "uint16[2]"}},
		{"uint32[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Uint32, Type: uint32_t, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[]"}},
		{"uint32[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Uint32, Type: uint32_t, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[2]"}},
		{"uint64[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Uint64, Type: uint64_t, Size: 64, T: UintTy, stringKind: "uint64"}, stringKind: "uint64[]"}},
		{"uint64[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Uint64, Type: uint64_t, Size: 64, T: UintTy, stringKind: "uint64"}, stringKind: "uint64[2]"}},
		{"uint256[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[]"}},
		{"uint256[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[2]"}},
		{"bytes32", Type{Kind: reflect.Array, T: FixedBytesTy, Size: 32, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "bytes32"}},
		{"bytes[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Slice, T: BytesTy, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "bytes"}, stringKind: "bytes[]"}},
		{"bytes[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{T: BytesTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "bytes"}, stringKind: "bytes[2]"}},
		{"bytes32[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Array, T: FixedBytesTy, Size: 32, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "bytes32"}, stringKind: "bytes32[]"}},
		{"bytes32[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Array, T: FixedBytesTy, Size: 32, Elem: &Type{Kind: reflect.Uint8, Type: uint8_t, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "bytes32"}, stringKind: "bytes32[2]"}},
		{"string", Type{Kind: reflect.String, T: StringTy, stringKind: "string"}},
		{"string[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.String, T: StringTy, stringKind: "string"}, stringKind: "string[]"}},
		{"string[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.String, T: StringTy, stringKind: "string"}, stringKind: "string[2]"}},
		{"address", Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}},
<<<<<<< HEAD
		{"address[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Array, Type: address_t, T: AddressTy, Size: 20, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[]"}},
		{"address[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Array, Type: address_t, T: AddressTy, Size: 20, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[2]"}},
=======
		{"address[]", Type{T: SliceTy, Kind: reflect.Slice, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[]"}},
		{"address[2]", Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[2]"}},
>>>>>>> 50ee5a2... accounts/abi: need to figure out what we're doing with bytes but 2D arrays/slices work now type creation wise.

		// TODO when fixed types are implemented properly
		// {"fixed", Type{}},
		// {"fixed128x128", Type{}},
		// {"fixed[]", Type{}},
		// {"fixed[2]", Type{}},
		// {"fixed128x128[]", Type{}},
		// {"fixed128x128[2]", Type{}},
	}
	for i, tt := range tests {
		typ, err := NewType(tt.blob)
		if err != nil {
			t.Errorf("type %d: failed to parse type string: %v", i, err)
		}
		if !reflect.DeepEqual(typ, tt.kind) {
			t.Errorf("type %d: parsed type mismatch:\n  have %+v\n  want %+v", i, typeWithoutStringer(typ), typeWithoutStringer(tt.kind))
		}
	}
}

func TestTypeCheck(t *testing.T) {
	for i, test := range []struct {
		typ   string
		input interface{}
		err   string
	}{
		{"uint", big.NewInt(1), ""},
		{"int", big.NewInt(1), ""},
		{"uint30", big.NewInt(1), ""},
		{"uint30", uint8(1), "abi: cannot use uint8 as type ptr as argument"},
		{"uint16", uint16(1), ""},
		{"uint16", uint8(1), "abi: cannot use uint8 as type uint16 as argument"},
		{"uint16[]", []uint16{1, 2, 3}, ""},
		{"uint16[]", [3]uint16{1, 2, 3}, ""},
		{"uint16[]", []uint32{1, 2, 3}, "abi: cannot use []uint32 as type [0]uint16 as argument"},
		{"uint16[3]", [3]uint32{1, 2, 3}, "abi: cannot use [3]uint32 as type [3]uint16 as argument"},
		{"uint16[3]", [4]uint16{1, 2, 3}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"uint16[3]", []uint16{1, 2, 3}, ""},
		{"uint16[3]", []uint16{1, 2, 3, 4}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"address[]", []common.Address{{1}}, ""},
		{"address[1]", []common.Address{{1}}, ""},
		{"address[1]", [1]common.Address{{1}}, ""},
		{"address[2]", [1]common.Address{{1}}, "abi: cannot use [1]array as type [2]array as argument"},
		{"bytes32", [32]byte{}, ""},
		{"bytes32", [33]byte{}, "abi: cannot use [33]uint8 as type [32]uint8 as argument"},
		{"bytes32", common.Hash{1}, ""},
		{"bytes31", [31]byte{}, ""},
		{"bytes31", [32]byte{}, "abi: cannot use [32]uint8 as type [31]uint8 as argument"},
		{"bytes", []byte{0, 1}, ""},
		{"bytes", [2]byte{0, 1}, "abi: cannot use array as type slice as argument"},
		{"bytes", common.Hash{1}, "abi: cannot use array as type slice as argument"},
		{"string", "hello world", ""},
<<<<<<< HEAD
=======
		{"string", string(""), ""},
		{"string", []byte{}, "abi: cannot use slice as type string as argument"},
>>>>>>> 50ee5a2... accounts/abi: need to figure out what we're doing with bytes but 2D arrays/slices work now type creation wise.
		{"bytes32[]", [][32]byte{{}}, ""},
		{"function", [24]byte{}, ""},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatal("unexpected parse error:", err)
<<<<<<< HEAD
=======
		} else if err != nil && len(test.err) != 0 {
			if err.Error() != test.err {
				t.Errorf("%d failed. Expected err: '%v' got err: '%v'", i, test.err, err)
			}
			continue
>>>>>>> 50ee5a2... accounts/abi: need to figure out what we're doing with bytes but 2D arrays/slices work now type creation wise.
		}

		err = typeCheck(typ, reflect.ValueOf(test.input))
		if err != nil && len(test.err) == 0 {
			t.Errorf("%d failed. Expected no err but got: %v", i, err)
			continue
		}
		if err == nil && len(test.err) != 0 {
			t.Errorf("%d failed. Expected err: %v but got none", i, test.err)
			continue
		}

		if err != nil && len(test.err) != 0 && err.Error() != test.err {
			t.Errorf("%d failed. Expected err: '%v' got err: '%v'", i, test.err, err)
		}
	}
}
