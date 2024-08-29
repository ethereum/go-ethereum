// Copyright 2022 The go-ethereum Authors
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

package vm

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEOFMarshaling(t *testing.T) {
	for i, test := range []struct {
		want Container
		err  error
	}{
		{
			want: Container{
				Types:    []*FunctionMetadata{{Input: 0, Output: 0x80, MaxStackHeight: 1}},
				Code:     [][]byte{common.Hex2Bytes("604200")},
				Data:     []byte{0x01, 0x02, 0x03},
				DataSize: 3,
			},
		},
		{
			want: Container{
				Types:    []*FunctionMetadata{{Input: 0, Output: 0x80, MaxStackHeight: 1}},
				Code:     [][]byte{common.Hex2Bytes("604200")},
				Data:     []byte{0x01, 0x02, 0x03},
				DataSize: 3,
			},
		},
		{
			want: Container{
				Types: []*FunctionMetadata{
					{Input: 0, Output: 0x80, MaxStackHeight: 1},
					{Input: 2, Output: 3, MaxStackHeight: 4},
					{Input: 1, Output: 1, MaxStackHeight: 1},
				},
				Code: [][]byte{
					common.Hex2Bytes("604200"),
					common.Hex2Bytes("6042604200"),
					common.Hex2Bytes("00"),
				},
				Data: []byte{},
			},
		},
	} {
		var (
			b   = test.want.MarshalBinary()
			got Container
		)
		t.Logf("b: %#x", b)
		if err := got.UnmarshalBinary(b, true); err != nil && err != test.err {
			t.Fatalf("test %d: got error \"%v\", want \"%v\"", i, err, test.err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("test %d: got %+v, want %+v", i, got, test.want)
		}
	}
}

func TestEOFSubcontainer(t *testing.T) {
	var subcontainer = new(Container)
	if err := subcontainer.UnmarshalBinary(common.Hex2Bytes("ef000101000402000100010400000000800000fe"), true); err != nil {
		t.Fatal(err)
	}
	container := Container{
		Types:             []*FunctionMetadata{{Input: 0, Output: 0x80, MaxStackHeight: 1}},
		Code:              [][]byte{common.Hex2Bytes("604200")},
		ContainerSections: []*Container{subcontainer},
		Data:              []byte{0x01, 0x02, 0x03},
		DataSize:          3,
	}
	var (
		b   = container.MarshalBinary()
		got Container
	)
	if err := got.UnmarshalBinary(b, true); err != nil {
		t.Fatal(err)
	}
	fmt.Print(got)
	if res := got.MarshalBinary(); !reflect.DeepEqual(res, b) {
		t.Fatalf("invalid marshalling, want %v got %v", b, res)
	}
}

func TestMarshaling(t *testing.T) {
	tests := []string{
		"EF000101000402000100040400000000800000E0000000",
		"ef0001010004020001000d04000000008000025fe100055f5fe000035f600100",
	}
	for i, test := range tests {
		s, err := hex.DecodeString(test)
		if err != nil {
			t.Fatalf("test %d: error decoding: %v", i, err)
		}
		var got Container
		if err := got.UnmarshalBinary(s, true); err != nil {
			t.Fatalf("test %d: got error %v", i, err)
		}
	}
}
