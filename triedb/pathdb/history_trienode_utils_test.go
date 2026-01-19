// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"reflect"
	"testing"
)

func TestHexPathNodeID(t *testing.T) {
	t.Parallel()

	var suites = []struct {
		input string
		exp   uint16
	}{
		{
			input: "",
			exp:   0,
		},
		{
			input: string([]byte{0x0}),
			exp:   1,
		},
		{
			input: string([]byte{0xf}),
			exp:   16,
		},
		{
			input: string([]byte{0x0, 0x0}),
			exp:   17,
		},
		{
			input: string([]byte{0x0, 0xf}),
			exp:   32,
		},
		{
			input: string([]byte{0x1, 0x0}),
			exp:   33,
		},
		{
			input: string([]byte{0x1, 0xf}),
			exp:   48,
		},
		{
			input: string([]byte{0xf, 0xf}),
			exp:   272,
		},
		{
			input: string([]byte{0xf, 0xf, 0xf}),
			exp:   4368,
		},
	}
	for _, suite := range suites {
		got := hexPathNodeID(suite.input)
		if got != suite.exp {
			t.Fatalf("Unexpected node ID for %v: got %d, want %d", suite.input, got, suite.exp)
		}
	}
}

func TestFindLeafPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  []string
		expect []string
	}{
		{
			input:  nil,
			expect: nil,
		},
		{
			input:  []string{"a"},
			expect: []string{"a"},
		},
		{
			input: []string{"", "0", "00", "01", "1"},
			expect: []string{
				"00",
				"01",
				"1",
			},
		},
		{
			input: []string{"10", "100", "11", "2"},
			expect: []string{
				"100",
				"11",
				"2",
			},
		},
		{
			input: []string{"10", "100000000", "11", "111111111", "2"},
			expect: []string{
				"100000000",
				"111111111",
				"2",
			},
		},
	}
	for _, test := range tests {
		res := findLeafPaths(test.input)
		if !reflect.DeepEqual(res, test.expect) {
			t.Fatalf("Unexpected result: %v, expected %v", res, test.expect)
		}
	}
}

func TestSplitAccountPath(t *testing.T) {
	t.Parallel()

	var suites = []struct {
		input     string
		expPrefix []string
		expID     []uint16
	}{
		// Length = 0
		{
			"", nil, nil,
		},
		// Length = 1
		{
			string([]byte{0x0}),
			[]string{
				string([]byte{0x0}),
			},
			[]uint16{
				0,
			},
		},
		{
			string([]byte{0x1}),
			[]string{
				string([]byte{0x1}),
			},
			[]uint16{
				0,
			},
		},
		{
			string([]byte{0xf}),
			[]string{
				string([]byte{0xf}),
			},
			[]uint16{
				0,
			},
		},
		// Length = 2
		{
			string([]byte{0x0, 0x0}),
			[]string{
				string([]byte{0x0}),
			},
			[]uint16{
				1,
			},
		},
		{
			string([]byte{0x0, 0x1}),
			[]string{
				string([]byte{0x0}),
			},
			[]uint16{
				2,
			},
		},
		{
			string([]byte{0x0, 0xf}),
			[]string{
				string([]byte{0x0}),
			},
			[]uint16{
				16,
			},
		},
		{
			string([]byte{0xf, 0xf}),
			[]string{
				string([]byte{0xf}),
			},
			[]uint16{
				16,
			},
		},
		// Length = 3
		{
			string([]byte{0x0, 0x0, 0x0}),
			[]string{
				string([]byte{0x0}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				1, 0,
			},
		},
		// Length = 3
		{
			string([]byte{0xf, 0xf, 0xf}),
			[]string{
				string([]byte{0xf}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				16, 0,
			},
		},
		// Length = 4
		{
			string([]byte{0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{0x0}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				1, 1,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{0xf}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				16, 16,
			},
		},
		// Length = 5
		{
			string([]byte{0x0, 0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{0x0}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				1, 17,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{0xf}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				16, 272,
			},
		},
		// Length = 6
		{
			string([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{0x0}),
				string([]byte{0x0, 0x0, 0x0}),
				string([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
			},
			[]uint16{
				1, 17, 0,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{0xf}),
				string([]byte{0xf, 0xf, 0xf}),
				string([]byte{0xf, 0xf, 0xf, 0xf, 0xf, 0xf}),
			},
			[]uint16{
				16, 272, 0,
			},
		},
	}
	for _, suite := range suites {
		prefix, id := accountIndexScheme.splitPath(suite.input)
		if !reflect.DeepEqual(prefix, suite.expPrefix) {
			t.Fatalf("Unexpected prefix for %v: got %v, want %v", suite.input, prefix, suite.expPrefix)
		}
		if !reflect.DeepEqual(id, suite.expID) {
			t.Fatalf("Unexpected ID for %v: got %v, want %v", suite.input, id, suite.expID)
		}
	}
}

func TestSplitStoragePath(t *testing.T) {
	t.Parallel()

	var suites = []struct {
		input     string
		expPrefix []string
		expID     []uint16
	}{
		// Length = 0
		{
			"",
			[]string{
				string([]byte{}),
			},
			[]uint16{
				0,
			},
		},
		// Length = 1
		{
			string([]byte{0x0}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				1,
			},
		},
		{
			string([]byte{0x1}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				2,
			},
		},
		{
			string([]byte{0xf}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				16,
			},
		},
		// Length = 2
		{
			string([]byte{0x0, 0x0}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				17,
			},
		},
		{
			string([]byte{0x0, 0x1}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				18,
			},
		},
		{
			string([]byte{0x0, 0xf}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				32,
			},
		},
		{
			string([]byte{0xf, 0xf}),
			[]string{
				string([]byte{}),
			},
			[]uint16{
				272,
			},
		},
		// Length = 3
		{
			string([]byte{0x0, 0x0, 0x0}),
			[]string{
				string([]byte{}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				17, 0,
			},
		},
		// Length = 3
		{
			string([]byte{0xf, 0xf, 0xf}),
			[]string{
				string([]byte{}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				272, 0,
			},
		},
		// Length = 4
		{
			string([]byte{0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				17, 1,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				272, 16,
			},
		},
		// Length = 5
		{
			string([]byte{0x0, 0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{}),
				string([]byte{0x0, 0x0, 0x0}),
			},
			[]uint16{
				17, 17,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{}),
				string([]byte{0xf, 0xf, 0xf}),
			},
			[]uint16{
				272, 272,
			},
		},
		// Length = 6
		{
			string([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
			[]string{
				string([]byte{}),
				string([]byte{0x0, 0x0, 0x0}),
				string([]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
			},
			[]uint16{
				17, 17, 0,
			},
		},
		{
			string([]byte{0xf, 0xf, 0xf, 0xf, 0xf, 0xf}),
			[]string{
				string([]byte{}),
				string([]byte{0xf, 0xf, 0xf}),
				string([]byte{0xf, 0xf, 0xf, 0xf, 0xf, 0xf}),
			},
			[]uint16{
				272, 272, 0,
			},
		},
	}
	for i, suite := range suites {
		prefix, id := storageIndexScheme.splitPath(suite.input)
		if !reflect.DeepEqual(prefix, suite.expPrefix) {
			t.Fatalf("Test %d, unexpected prefix for %v: got %v, want %v", i, suite.input, prefix, suite.expPrefix)
		}
		if !reflect.DeepEqual(id, suite.expID) {
			t.Fatalf("Test %d, unexpected ID for %v: got %v, want %v", i, suite.input, id, suite.expID)
		}
	}
}

func TestIsAncestor(t *testing.T) {
	suites := []struct {
		x, y uint16
		want bool
	}{
		{0, 1, true},
		{0, 16, true},
		{0, 17, true},
		{0, 272, true},

		{1, 0, false},
		{1, 2, false},
		{1, 17, true},
		{1, 18, true},
		{17, 273, true},
		{1, 1, false},
	}
	for _, tc := range suites {
		result := isAncestor(tc.x, tc.y)
		if result != tc.want {
			t.Fatalf("isAncestor(%d, %d) = %v, want %v", tc.x, tc.y, result, tc.want)
		}
	}
}

func TestBitmapSet(t *testing.T) {
	suites := []struct {
		index  int
		expect []byte
	}{
		{
			0, []byte{0b10000000, 0x0},
		},
		{
			1, []byte{0b01000000, 0x0},
		},
		{
			7, []byte{0b00000001, 0x0},
		},
		{
			8, []byte{0b00000000, 0b10000000},
		},
		{
			15, []byte{0b00000000, 0b00000001},
		},
	}
	for _, tc := range suites {
		var buf [2]byte
		setBit(buf[:], tc.index)

		if !bytes.Equal(buf[:], tc.expect) {
			t.Fatalf("bitmap = %v, want %v", buf, tc.expect)
		}
		if !isBitSet(buf[:], tc.index) {
			t.Fatal("bit is not set")
		}
	}
}

func TestBitPositions(t *testing.T) {
	suites := []struct {
		input  []byte
		expect []int
	}{
		{
			[]byte{0b10000000, 0x0}, []int{0},
		},
		{
			[]byte{0b01000000, 0x0}, []int{1},
		},
		{
			[]byte{0b00000001, 0x0}, []int{7},
		},
		{
			[]byte{0b00000000, 0b10000000}, []int{8},
		},
		{
			[]byte{0b00000000, 0b00000001}, []int{15},
		},
		{
			[]byte{0b10000000, 0b00000001}, []int{0, 15},
		},
		{
			[]byte{0b10000001, 0b00000001}, []int{0, 7, 15},
		},
		{
			[]byte{0b10000001, 0b10000001}, []int{0, 7, 8, 15},
		},
		{
			[]byte{0b11111111, 0b11111111}, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		{
			[]byte{0x0, 0x0}, nil,
		},
	}
	for _, tc := range suites {
		got := bitPosTwoBytes(tc.input)
		if !reflect.DeepEqual(got, tc.expect) {
			t.Fatalf("Unexpected position set, want: %v, got: %v", tc.expect, got)
		}
	}
}
