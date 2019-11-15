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

	"github.com/davecgh/go-spew/spew"
	"github.com/maticnetwork/bor/common"
)

// typeWithoutStringer is a alias for the Type type which simply doesn't implement
// the stringer interface to allow printing type details in the tests below.
type typeWithoutStringer Type

// Tests that all allowed types get recognized by the type parser.
func TestTypeRegexp(t *testing.T) {
	tests := []struct {
		blob       string
		components []ArgumentMarshaling
		kind       Type
	}{
		{"bool", nil, Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}},
		{"bool[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]bool(nil)), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[]"}},
		{"bool[2]", nil, Type{Size: 2, Kind: reflect.Array, T: ArrayTy, Type: reflect.TypeOf([2]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[2]"}},
		{"bool[2][]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([][2]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][]"}},
		{"bool[][]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([][]bool{}), Elem: &Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][]"}},
		{"bool[][2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][]bool{}), Elem: &Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][2]"}},
		{"bool[2][2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][2]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][2]"}},
		{"bool[2][][2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][][2]bool{}), Elem: &Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([][2]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][]"}, stringKind: "bool[2][][2]"}},
		{"bool[2][2][2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][2][2]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][2]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[2]"}, stringKind: "bool[2][2]"}, stringKind: "bool[2][2][2]"}},
		{"bool[][][]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([][][]bool{}), Elem: &Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([][]bool{}), Elem: &Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][]"}, stringKind: "bool[][][]"}},
		{"bool[][2][]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([][2][]bool{}), Elem: &Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][]bool{}), Elem: &Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]bool{}), Elem: &Type{Kind: reflect.Bool, T: BoolTy, Type: reflect.TypeOf(bool(false)), stringKind: "bool"}, stringKind: "bool[]"}, stringKind: "bool[][2]"}, stringKind: "bool[][2][]"}},
		{"int8", nil, Type{Kind: reflect.Int8, Type: int8T, Size: 8, T: IntTy, stringKind: "int8"}},
		{"int16", nil, Type{Kind: reflect.Int16, Type: int16T, Size: 16, T: IntTy, stringKind: "int16"}},
		{"int32", nil, Type{Kind: reflect.Int32, Type: int32T, Size: 32, T: IntTy, stringKind: "int32"}},
		{"int64", nil, Type{Kind: reflect.Int64, Type: int64T, Size: 64, T: IntTy, stringKind: "int64"}},
		{"int256", nil, Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: IntTy, stringKind: "int256"}},
		{"int8[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]int8{}), Elem: &Type{Kind: reflect.Int8, Type: int8T, Size: 8, T: IntTy, stringKind: "int8"}, stringKind: "int8[]"}},
		{"int8[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]int8{}), Elem: &Type{Kind: reflect.Int8, Type: int8T, Size: 8, T: IntTy, stringKind: "int8"}, stringKind: "int8[2]"}},
		{"int16[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]int16{}), Elem: &Type{Kind: reflect.Int16, Type: int16T, Size: 16, T: IntTy, stringKind: "int16"}, stringKind: "int16[]"}},
		{"int16[2]", nil, Type{Size: 2, Kind: reflect.Array, T: ArrayTy, Type: reflect.TypeOf([2]int16{}), Elem: &Type{Kind: reflect.Int16, Type: int16T, Size: 16, T: IntTy, stringKind: "int16"}, stringKind: "int16[2]"}},
		{"int32[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]int32{}), Elem: &Type{Kind: reflect.Int32, Type: int32T, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[]"}},
		{"int32[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]int32{}), Elem: &Type{Kind: reflect.Int32, Type: int32T, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[2]"}},
		{"int64[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]int64{}), Elem: &Type{Kind: reflect.Int64, Type: int64T, Size: 64, T: IntTy, stringKind: "int64"}, stringKind: "int64[]"}},
		{"int64[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]int64{}), Elem: &Type{Kind: reflect.Int64, Type: int64T, Size: 64, T: IntTy, stringKind: "int64"}, stringKind: "int64[2]"}},
		{"int256[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]*big.Int{}), Elem: &Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[]"}},
		{"int256[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]*big.Int{}), Elem: &Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[2]"}},
		{"uint8", nil, Type{Kind: reflect.Uint8, Type: uint8T, Size: 8, T: UintTy, stringKind: "uint8"}},
		{"uint16", nil, Type{Kind: reflect.Uint16, Type: uint16T, Size: 16, T: UintTy, stringKind: "uint16"}},
		{"uint32", nil, Type{Kind: reflect.Uint32, Type: uint32T, Size: 32, T: UintTy, stringKind: "uint32"}},
		{"uint64", nil, Type{Kind: reflect.Uint64, Type: uint64T, Size: 64, T: UintTy, stringKind: "uint64"}},
		{"uint256", nil, Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: UintTy, stringKind: "uint256"}},
		{"uint8[]", nil, Type{Kind: reflect.Slice, T: SliceTy, Type: reflect.TypeOf([]uint8{}), Elem: &Type{Kind: reflect.Uint8, Type: uint8T, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "uint8[]"}},
		{"uint8[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]uint8{}), Elem: &Type{Kind: reflect.Uint8, Type: uint8T, Size: 8, T: UintTy, stringKind: "uint8"}, stringKind: "uint8[2]"}},
		{"uint16[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]uint16{}), Elem: &Type{Kind: reflect.Uint16, Type: uint16T, Size: 16, T: UintTy, stringKind: "uint16"}, stringKind: "uint16[]"}},
		{"uint16[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]uint16{}), Elem: &Type{Kind: reflect.Uint16, Type: uint16T, Size: 16, T: UintTy, stringKind: "uint16"}, stringKind: "uint16[2]"}},
		{"uint32[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]uint32{}), Elem: &Type{Kind: reflect.Uint32, Type: uint32T, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[]"}},
		{"uint32[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]uint32{}), Elem: &Type{Kind: reflect.Uint32, Type: uint32T, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[2]"}},
		{"uint64[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]uint64{}), Elem: &Type{Kind: reflect.Uint64, Type: uint64T, Size: 64, T: UintTy, stringKind: "uint64"}, stringKind: "uint64[]"}},
		{"uint64[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]uint64{}), Elem: &Type{Kind: reflect.Uint64, Type: uint64T, Size: 64, T: UintTy, stringKind: "uint64"}, stringKind: "uint64[2]"}},
		{"uint256[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]*big.Int{}), Elem: &Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[]"}},
		{"uint256[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Type: reflect.TypeOf([2]*big.Int{}), Size: 2, Elem: &Type{Kind: reflect.Ptr, Type: bigT, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[2]"}},
		{"bytes32", nil, Type{Kind: reflect.Array, T: FixedBytesTy, Size: 32, Type: reflect.TypeOf([32]byte{}), stringKind: "bytes32"}},
		{"bytes[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([][]byte{}), Elem: &Type{Kind: reflect.Slice, Type: reflect.TypeOf([]byte{}), T: BytesTy, stringKind: "bytes"}, stringKind: "bytes[]"}},
		{"bytes[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][]byte{}), Elem: &Type{T: BytesTy, Type: reflect.TypeOf([]byte{}), Kind: reflect.Slice, stringKind: "bytes"}, stringKind: "bytes[2]"}},
		{"bytes32[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([][32]byte{}), Elem: &Type{Kind: reflect.Array, Type: reflect.TypeOf([32]byte{}), T: FixedBytesTy, Size: 32, stringKind: "bytes32"}, stringKind: "bytes32[]"}},
		{"bytes32[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2][32]byte{}), Elem: &Type{Kind: reflect.Array, T: FixedBytesTy, Size: 32, Type: reflect.TypeOf([32]byte{}), stringKind: "bytes32"}, stringKind: "bytes32[2]"}},
		{"string", nil, Type{Kind: reflect.String, T: StringTy, Type: reflect.TypeOf(""), stringKind: "string"}},
		{"string[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]string{}), Elem: &Type{Kind: reflect.String, Type: reflect.TypeOf(""), T: StringTy, stringKind: "string"}, stringKind: "string[]"}},
		{"string[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]string{}), Elem: &Type{Kind: reflect.String, T: StringTy, Type: reflect.TypeOf(""), stringKind: "string"}, stringKind: "string[2]"}},
		{"address", nil, Type{Kind: reflect.Array, Type: addressT, Size: 20, T: AddressTy, stringKind: "address"}},
		{"address[]", nil, Type{T: SliceTy, Kind: reflect.Slice, Type: reflect.TypeOf([]common.Address{}), Elem: &Type{Kind: reflect.Array, Type: addressT, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[]"}},
		{"address[2]", nil, Type{Kind: reflect.Array, T: ArrayTy, Size: 2, Type: reflect.TypeOf([2]common.Address{}), Elem: &Type{Kind: reflect.Array, Type: addressT, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[2]"}},
		// TODO when fixed types are implemented properly
		// {"fixed", nil, Type{}},
		// {"fixed128x128", nil, Type{}},
		// {"fixed[]", nil, Type{}},
		// {"fixed[2]", nil, Type{}},
		// {"fixed128x128[]", nil, Type{}},
		// {"fixed128x128[2]", nil, Type{}},
		{"tuple", []ArgumentMarshaling{{Name: "a", Type: "int64"}}, Type{Kind: reflect.Struct, T: TupleTy, Type: reflect.TypeOf(struct {
			A int64 `json:"a"`
		}{}), stringKind: "(int64)",
			TupleElems: []*Type{{Kind: reflect.Int64, T: IntTy, Type: reflect.TypeOf(int64(0)), Size: 64, stringKind: "int64"}}, TupleRawNames: []string{"a"}}},
		{"tuple with long name", []ArgumentMarshaling{{Name: "aTypicalParamName", Type: "int64"}}, Type{Kind: reflect.Struct, T: TupleTy, Type: reflect.TypeOf(struct {
			ATypicalParamName int64 `json:"aTypicalParamName"`
		}{}), stringKind: "(int64)",
			TupleElems: []*Type{{Kind: reflect.Int64, T: IntTy, Type: reflect.TypeOf(int64(0)), Size: 64, stringKind: "int64"}}, TupleRawNames: []string{"aTypicalParamName"}}},
	}

	for _, tt := range tests {
		typ, err := NewType(tt.blob, tt.components)
		if err != nil {
			t.Errorf("type %q: failed to parse type string: %v", tt.blob, err)
		}
		if !reflect.DeepEqual(typ, tt.kind) {
			t.Errorf("type %q: parsed type mismatch:\nGOT %s\nWANT %s ", tt.blob, spew.Sdump(typeWithoutStringer(typ)), spew.Sdump(typeWithoutStringer(tt.kind)))
		}
	}
}

func TestTypeCheck(t *testing.T) {
	for i, test := range []struct {
		typ        string
		components []ArgumentMarshaling
		input      interface{}
		err        string
	}{
		{"uint", nil, big.NewInt(1), "unsupported arg type: uint"},
		{"int", nil, big.NewInt(1), "unsupported arg type: int"},
		{"uint256", nil, big.NewInt(1), ""},
		{"uint256[][3][]", nil, [][3][]*big.Int{{{}}}, ""},
		{"uint256[][][3]", nil, [3][][]*big.Int{{{}}}, ""},
		{"uint256[3][][]", nil, [][][3]*big.Int{{{}}}, ""},
		{"uint256[3][3][3]", nil, [3][3][3]*big.Int{{{}}}, ""},
		{"uint8[][]", nil, [][]uint8{}, ""},
		{"int256", nil, big.NewInt(1), ""},
		{"uint8", nil, uint8(1), ""},
		{"uint16", nil, uint16(1), ""},
		{"uint32", nil, uint32(1), ""},
		{"uint64", nil, uint64(1), ""},
		{"int8", nil, int8(1), ""},
		{"int16", nil, int16(1), ""},
		{"int32", nil, int32(1), ""},
		{"int64", nil, int64(1), ""},
		{"uint24", nil, big.NewInt(1), ""},
		{"uint40", nil, big.NewInt(1), ""},
		{"uint48", nil, big.NewInt(1), ""},
		{"uint56", nil, big.NewInt(1), ""},
		{"uint72", nil, big.NewInt(1), ""},
		{"uint80", nil, big.NewInt(1), ""},
		{"uint88", nil, big.NewInt(1), ""},
		{"uint96", nil, big.NewInt(1), ""},
		{"uint104", nil, big.NewInt(1), ""},
		{"uint112", nil, big.NewInt(1), ""},
		{"uint120", nil, big.NewInt(1), ""},
		{"uint128", nil, big.NewInt(1), ""},
		{"uint136", nil, big.NewInt(1), ""},
		{"uint144", nil, big.NewInt(1), ""},
		{"uint152", nil, big.NewInt(1), ""},
		{"uint160", nil, big.NewInt(1), ""},
		{"uint168", nil, big.NewInt(1), ""},
		{"uint176", nil, big.NewInt(1), ""},
		{"uint184", nil, big.NewInt(1), ""},
		{"uint192", nil, big.NewInt(1), ""},
		{"uint200", nil, big.NewInt(1), ""},
		{"uint208", nil, big.NewInt(1), ""},
		{"uint216", nil, big.NewInt(1), ""},
		{"uint224", nil, big.NewInt(1), ""},
		{"uint232", nil, big.NewInt(1), ""},
		{"uint240", nil, big.NewInt(1), ""},
		{"uint248", nil, big.NewInt(1), ""},
		{"int24", nil, big.NewInt(1), ""},
		{"int40", nil, big.NewInt(1), ""},
		{"int48", nil, big.NewInt(1), ""},
		{"int56", nil, big.NewInt(1), ""},
		{"int72", nil, big.NewInt(1), ""},
		{"int80", nil, big.NewInt(1), ""},
		{"int88", nil, big.NewInt(1), ""},
		{"int96", nil, big.NewInt(1), ""},
		{"int104", nil, big.NewInt(1), ""},
		{"int112", nil, big.NewInt(1), ""},
		{"int120", nil, big.NewInt(1), ""},
		{"int128", nil, big.NewInt(1), ""},
		{"int136", nil, big.NewInt(1), ""},
		{"int144", nil, big.NewInt(1), ""},
		{"int152", nil, big.NewInt(1), ""},
		{"int160", nil, big.NewInt(1), ""},
		{"int168", nil, big.NewInt(1), ""},
		{"int176", nil, big.NewInt(1), ""},
		{"int184", nil, big.NewInt(1), ""},
		{"int192", nil, big.NewInt(1), ""},
		{"int200", nil, big.NewInt(1), ""},
		{"int208", nil, big.NewInt(1), ""},
		{"int216", nil, big.NewInt(1), ""},
		{"int224", nil, big.NewInt(1), ""},
		{"int232", nil, big.NewInt(1), ""},
		{"int240", nil, big.NewInt(1), ""},
		{"int248", nil, big.NewInt(1), ""},
		{"uint30", nil, uint8(1), "abi: cannot use uint8 as type ptr as argument"},
		{"uint8", nil, uint16(1), "abi: cannot use uint16 as type uint8 as argument"},
		{"uint8", nil, uint32(1), "abi: cannot use uint32 as type uint8 as argument"},
		{"uint8", nil, uint64(1), "abi: cannot use uint64 as type uint8 as argument"},
		{"uint8", nil, int8(1), "abi: cannot use int8 as type uint8 as argument"},
		{"uint8", nil, int16(1), "abi: cannot use int16 as type uint8 as argument"},
		{"uint8", nil, int32(1), "abi: cannot use int32 as type uint8 as argument"},
		{"uint8", nil, int64(1), "abi: cannot use int64 as type uint8 as argument"},
		{"uint16", nil, uint16(1), ""},
		{"uint16", nil, uint8(1), "abi: cannot use uint8 as type uint16 as argument"},
		{"uint16[]", nil, []uint16{1, 2, 3}, ""},
		{"uint16[]", nil, [3]uint16{1, 2, 3}, ""},
		{"uint16[]", nil, []uint32{1, 2, 3}, "abi: cannot use []uint32 as type [0]uint16 as argument"},
		{"uint16[3]", nil, [3]uint32{1, 2, 3}, "abi: cannot use [3]uint32 as type [3]uint16 as argument"},
		{"uint16[3]", nil, [4]uint16{1, 2, 3}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"uint16[3]", nil, []uint16{1, 2, 3}, ""},
		{"uint16[3]", nil, []uint16{1, 2, 3, 4}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"address[]", nil, []common.Address{{1}}, ""},
		{"address[1]", nil, []common.Address{{1}}, ""},
		{"address[1]", nil, [1]common.Address{{1}}, ""},
		{"address[2]", nil, [1]common.Address{{1}}, "abi: cannot use [1]array as type [2]array as argument"},
		{"bytes32", nil, [32]byte{}, ""},
		{"bytes31", nil, [31]byte{}, ""},
		{"bytes30", nil, [30]byte{}, ""},
		{"bytes29", nil, [29]byte{}, ""},
		{"bytes28", nil, [28]byte{}, ""},
		{"bytes27", nil, [27]byte{}, ""},
		{"bytes26", nil, [26]byte{}, ""},
		{"bytes25", nil, [25]byte{}, ""},
		{"bytes24", nil, [24]byte{}, ""},
		{"bytes23", nil, [23]byte{}, ""},
		{"bytes22", nil, [22]byte{}, ""},
		{"bytes21", nil, [21]byte{}, ""},
		{"bytes20", nil, [20]byte{}, ""},
		{"bytes19", nil, [19]byte{}, ""},
		{"bytes18", nil, [18]byte{}, ""},
		{"bytes17", nil, [17]byte{}, ""},
		{"bytes16", nil, [16]byte{}, ""},
		{"bytes15", nil, [15]byte{}, ""},
		{"bytes14", nil, [14]byte{}, ""},
		{"bytes13", nil, [13]byte{}, ""},
		{"bytes12", nil, [12]byte{}, ""},
		{"bytes11", nil, [11]byte{}, ""},
		{"bytes10", nil, [10]byte{}, ""},
		{"bytes9", nil, [9]byte{}, ""},
		{"bytes8", nil, [8]byte{}, ""},
		{"bytes7", nil, [7]byte{}, ""},
		{"bytes6", nil, [6]byte{}, ""},
		{"bytes5", nil, [5]byte{}, ""},
		{"bytes4", nil, [4]byte{}, ""},
		{"bytes3", nil, [3]byte{}, ""},
		{"bytes2", nil, [2]byte{}, ""},
		{"bytes1", nil, [1]byte{}, ""},
		{"bytes32", nil, [33]byte{}, "abi: cannot use [33]uint8 as type [32]uint8 as argument"},
		{"bytes32", nil, common.Hash{1}, ""},
		{"bytes31", nil, common.Hash{1}, "abi: cannot use common.Hash as type [31]uint8 as argument"},
		{"bytes31", nil, [32]byte{}, "abi: cannot use [32]uint8 as type [31]uint8 as argument"},
		{"bytes", nil, []byte{0, 1}, ""},
		{"bytes", nil, [2]byte{0, 1}, "abi: cannot use array as type slice as argument"},
		{"bytes", nil, common.Hash{1}, "abi: cannot use array as type slice as argument"},
		{"string", nil, "hello world", ""},
		{"string", nil, string(""), ""},
		{"string", nil, []byte{}, "abi: cannot use slice as type string as argument"},
		{"bytes32[]", nil, [][32]byte{{}}, ""},
		{"function", nil, [24]byte{}, ""},
		{"bytes20", nil, common.Address{}, ""},
		{"address", nil, [20]byte{}, ""},
		{"address", nil, common.Address{}, ""},
		{"bytes32[]]", nil, "", "invalid arg type in abi"},
		{"invalidType", nil, "", "unsupported arg type: invalidType"},
		{"invalidSlice[]", nil, "", "unsupported arg type: invalidSlice"},
		// simple tuple
		{"tuple", []ArgumentMarshaling{{Name: "a", Type: "uint256"}, {Name: "b", Type: "uint256"}}, struct {
			A *big.Int
			B *big.Int
		}{}, ""},
		// tuple slice
		{"tuple[]", []ArgumentMarshaling{{Name: "a", Type: "uint256"}, {Name: "b", Type: "uint256"}}, []struct {
			A *big.Int
			B *big.Int
		}{}, ""},
		// tuple array
		{"tuple[2]", []ArgumentMarshaling{{Name: "a", Type: "uint256"}, {Name: "b", Type: "uint256"}}, []struct {
			A *big.Int
			B *big.Int
		}{{big.NewInt(0), big.NewInt(0)}, {big.NewInt(0), big.NewInt(0)}}, ""},
	} {
		typ, err := NewType(test.typ, test.components)
		if err != nil && len(test.err) == 0 {
			t.Fatal("unexpected parse error:", err)
		} else if err != nil && len(test.err) != 0 {
			if err.Error() != test.err {
				t.Errorf("%d failed. Expected err: '%v' got err: '%v'", i, test.err, err)
			}
			continue
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
