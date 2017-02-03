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
	"reflect"
	"testing"
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
		{"int", Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}},
		{"int8", Type{Kind: reflect.Int8, Type: big_t, Size: 8, T: IntTy, stringKind: "int8"}},
		{"int256", Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}},
		{"int[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[]"}},
		{"int[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, Elem: &Type{Kind: reflect.Ptr, Type: big_t, Size: 256, T: IntTy, stringKind: "int256"}, stringKind: "int256[2]"}},
		{"int32[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Int32, Type: big_t, Size: 32, T: IntTy, Elem: &Type{Kind: reflect.Int32, Type: big_t, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[]"}},
		{"int32[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Int32, Type: big_t, Size: 32, T: IntTy, Elem: &Type{Kind: reflect.Int32, Type: big_t, Size: 32, T: IntTy, stringKind: "int32"}, stringKind: "int32[2]"}},
		{"uint", Type{Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, stringKind: "uint256"}},
		{"uint8", Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}},
		{"uint256", Type{Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, stringKind: "uint256"}},
		{"uint[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, Elem: &Type{Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[]"}},
		{"uint[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, Elem: &Type{Kind: reflect.Ptr, Type: ubig_t, Size: 256, T: UintTy, stringKind: "uint256"}, stringKind: "uint256[2]"}},
		{"uint32[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Uint32, Type: ubig_t, Size: 32, T: UintTy, Elem: &Type{Kind: reflect.Uint32, Type: big_t, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[]"}},
		{"uint32[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Uint32, Type: ubig_t, Size: 32, T: UintTy, Elem: &Type{Kind: reflect.Uint32, Type: big_t, Size: 32, T: UintTy, stringKind: "uint32"}, stringKind: "uint32[2]"}},
		{"bytes", Type{IsSlice: true, SliceSize: -1, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: BytesTy, stringKind: "bytes"}},
		{"bytes32", Type{IsArray: true, SliceSize: 32, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: FixedBytesTy, stringKind: "bytes32"}},
		{"bytes[]", Type{IsSlice: true, SliceSize: -1, Elem: &Type{IsSlice: true, SliceSize: -1, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: BytesTy, stringKind: "bytes"}, stringKind: "bytes[]"}},
		{"bytes[2]", Type{IsArray: true, SliceSize: 2, Elem: &Type{IsSlice: true, SliceSize: -1, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: BytesTy, stringKind: "bytes"}, stringKind: "bytes[2]"}},
		{"bytes32[]", Type{IsSlice: true, SliceSize: -1, Elem: &Type{IsArray: true, SliceSize: 32, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: FixedBytesTy, stringKind: "bytes32"}, stringKind: "bytes32[]"}},
		{"bytes32[2]", Type{IsArray: true, SliceSize: 2, Elem: &Type{IsArray: true, SliceSize: 32, Elem: &Type{Kind: reflect.Uint8, Type: ubig_t, Size: 8, T: UintTy, stringKind: "uint8"}, T: FixedBytesTy, stringKind: "bytes32"}, stringKind: "bytes32[2]"}},
		{"string", Type{Kind: reflect.String, Size: -1, T: StringTy, stringKind: "string"}},
		{"string[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.String, T: StringTy, Size: -1, Elem: &Type{Kind: reflect.String, T: StringTy, Size: -1, stringKind: "string"}, stringKind: "string[]"}},
		{"string[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.String, T: StringTy, Size: -1, Elem: &Type{Kind: reflect.String, T: StringTy, Size: -1, stringKind: "string"}, stringKind: "string[2]"}},
		{"address", Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}},
		{"address[]", Type{IsSlice: true, SliceSize: -1, Kind: reflect.Array, Type: address_t, T: AddressTy, Size: 20, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[]"}},
		{"address[2]", Type{IsArray: true, SliceSize: 2, Kind: reflect.Array, Type: address_t, T: AddressTy, Size: 20, Elem: &Type{Kind: reflect.Array, Type: address_t, Size: 20, T: AddressTy, stringKind: "address"}, stringKind: "address[2]"}},

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
