// Copyright 2023 The go-ethereum Authors
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

package apitypes

import "testing"

func TestIsPrimitive(t *testing.T) {
	t.Parallel()
	// Expected positives
	for i, tc := range []string{
		"int24", "int24[]", "int[]", "int[2]", "uint88", "uint88[]", "uint", "uint[]", "uint[2]", "int256", "int256[]",
		"uint96", "uint96[]", "int96", "int96[]", "bytes17[]", "bytes17", "address[2]", "bool[4]", "string[5]", "bytes[2]",
		"bytes32", "bytes32[]", "bytes32[4]",
	} {
		if !isPrimitiveTypeValid(tc) {
			t.Errorf("test %d: expected '%v' to be a valid primitive", i, tc)
		}
	}
	// Expected negatives
	for i, tc := range []string{
		"int257", "int257[]", "uint88 ", "uint88 []", "uint257", "uint-1[]",
		"uint0", "uint0[]", "int95", "int95[]", "uint1", "uint1[]", "bytes33[]", "bytess",
	} {
		if isPrimitiveTypeValid(tc) {
			t.Errorf("test %d: expected '%v' to not be a valid primitive", i, tc)
		}
	}
}

func TestType_IsArray(t *testing.T) {
	t.Parallel()
	// Expected positives
	for i, tc := range []Type{
		{
			Name: "type1",
			Type: "int24[]",
		},
		{
			Name: "type2",
			Type: "int24[2]",
		},
		{
			Name: "type3",
			Type: "int24[2][2][2]",
		},
	} {
		if !tc.isArray() {
			t.Errorf("test %d: expected '%v' to be an array", i, tc)
		}
	}
	// Expected negatives
	for i, tc := range []Type{
		{
			Name: "type1",
			Type: "int24",
		},
		{
			Name: "type2",
			Type: "uint88",
		},
		{
			Name: "type3",
			Type: "bytes32",
		},
	} {
		if tc.isArray() {
			t.Errorf("test %d: expected '%v' to not be an array", i, tc)
		}
	}
}

func TestType_TypeName(t *testing.T) {
	t.Parallel()

	for i, tc := range []struct {
		Input    Type
		Expected string
	}{
		{
			Input: Type{
				Name: "type1",
				Type: "int24[]",
			},
			Expected: "int24",
		},
		{
			Input: Type{
				Name: "type2",
				Type: "int26[2][2][2]",
			},
			Expected: "int26",
		},
		{
			Input: Type{
				Name: "type3",
				Type: "int24",
			},
			Expected: "int24",
		},
		{
			Input: Type{
				Name: "type4",
				Type: "uint88",
			},
			Expected: "uint88",
		},
		{
			Input: Type{
				Name: "type5",
				Type: "bytes32[2]",
			},
			Expected: "bytes32",
		},
	} {
		if tc.Input.typeName() != tc.Expected {
			t.Errorf("test %d: expected typeName value of '%v' to be '%v'", i, tc.Input, tc.Expected)
		}
	}
}
