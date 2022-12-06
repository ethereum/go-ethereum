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
				Types: []TypeAnnotation{{input: 0, output: 0, maxStackHeight: 0}},
				Code:  [][]byte{common.Hex2Bytes("604200")},
				Data:  []byte{0x01, 0x02, 0x03},
			},
		},
		{
			want: Container{
				Types: []TypeAnnotation{
					{input: 0, output: 0, maxStackHeight: 0},
					{input: 2, output: 3, maxStackHeight: 32},
					{input: 1, output: 1, maxStackHeight: 1023},
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
		if err := got.UnmarshalBinary(b); err != nil && err != test.err {
			t.Fatalf("test %d: got error \"%v\", want \"%v\"", i, err, test.err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("test %d: got %+v, want %+v", i, got, test.want)
		}
	}
}
