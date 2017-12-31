// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type unpackTest struct {
	def  string      // ABI definition JSON
	enc  string      // evm return data
	want interface{} // the expected output
	err  string      // empty or error if expected
}

func (test unpackTest) checkError(err error) error {
	if err != nil {
		if len(test.err) == 0 {
			return fmt.Errorf("expected no err but got: %v", err)
		} else if err.Error() != test.err {
			return fmt.Errorf("expected err: '%v' got err: %q", test.err, err)
		}
	} else if len(test.err) > 0 {
		return fmt.Errorf("expected err: %v but got none", test.err)
	}
	return nil
}

var unpackTests = []unpackTest{
	{
		def:  `[{ "type": "bool" }]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: true,
	},
	{
		def:  `[{"type": "uint32"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: uint32(1),
	},
	{
		def:  `[{"type": "uint32"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: uint16(0),
		err:  "abi: cannot unmarshal uint32 in to uint16",
	},
	{
		def:  `[{"type": "uint17"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: uint16(0),
		err:  "abi: cannot unmarshal *big.Int in to uint16",
	},
	{
		def:  `[{"type": "uint17"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: big.NewInt(1),
	},
	{
		def:  `[{"type": "int32"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: int32(1),
	},
	{
		def:  `[{"type": "int32"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: int16(0),
		err:  "abi: cannot unmarshal int32 in to int16",
	},
	{
		def:  `[{"type": "int17"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: int16(0),
		err:  "abi: cannot unmarshal *big.Int in to int16",
	},
	{
		def:  `[{"type": "int17"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: big.NewInt(1),
	},
	{
		def:  `[{"type": "address"}]`,
		enc:  "0000000000000000000000000100000000000000000000000000000000000000",
		want: common.Address{1},
	},
	{
		def:  `[{"type": "bytes32"}]`,
		enc:  "0100000000000000000000000000000000000000000000000000000000000000",
		want: [32]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	{
		def:  `[{"type": "bytes"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200100000000000000000000000000000000000000000000000000000000000000",
		want: common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
	},
	{
		def:  `[{"type": "bytes"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200100000000000000000000000000000000000000000000000000000000000000",
		want: [32]byte{},
		err:  "abi: cannot unmarshal []uint8 in to [32]uint8",
	},
	{
		def:  `[{"type": "bytes32"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200100000000000000000000000000000000000000000000000000000000000000",
		want: []byte(nil),
		err:  "abi: cannot unmarshal [32]uint8 in to []uint8",
	},
	{
		def:  `[{"type": "bytes32"}]`,
		enc:  "0100000000000000000000000000000000000000000000000000000000000000",
		want: common.HexToHash("0100000000000000000000000000000000000000000000000000000000000000"),
	},
	{
		def:  `[{"type": "function"}]`,
		enc:  "0100000000000000000000000000000000000000000000000000000000000000",
		want: [24]byte{1},
	},
	// slices
	{
		def:  `[{"type": "uint8[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []uint8{1, 2},
	},
	{
		def:  `[{"type": "uint8[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]uint8{1, 2},
	},
	// multi dimensional, if these pass, all types that don't require length prefix should pass
	{
		def:  `[{"type": "uint8[][]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000E0000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [][]uint8{{1, 2}, {1, 2}},
	},
	{
		def:  `[{"type": "uint8[2][2]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2][2]uint8{{1, 2}, {1, 2}},
	},
	{
		def:  `[{"type": "uint8[][2]"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001",
		want: [2][]uint8{{1}, {1}},
	},
	{
		def:  `[{"type": "uint8[2][]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [][2]uint8{{1, 2}},
	},
	{
		def:  `[{"type": "uint16[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []uint16{1, 2},
	},
	{
		def:  `[{"type": "uint16[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]uint16{1, 2},
	},
	{
		def:  `[{"type": "uint32[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []uint32{1, 2},
	},
	{
		def:  `[{"type": "uint32[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]uint32{1, 2},
	},
	{
		def:  `[{"type": "uint64[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []uint64{1, 2},
	},
	{
		def:  `[{"type": "uint64[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]uint64{1, 2},
	},
	{
		def:  `[{"type": "uint256[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []*big.Int{big.NewInt(1), big.NewInt(2)},
	},
	{
		def:  `[{"type": "uint256[3]"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003",
		want: [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
	},
	{
		def:  `[{"type": "int8[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []int8{1, 2},
	},
	{
		def:  `[{"type": "int8[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]int8{1, 2},
	},
	{
		def:  `[{"type": "int16[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []int16{1, 2},
	},
	{
		def:  `[{"type": "int16[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]int16{1, 2},
	},
	{
		def:  `[{"type": "int32[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []int32{1, 2},
	},
	{
		def:  `[{"type": "int32[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]int32{1, 2},
	},
	{
		def:  `[{"type": "int64[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []int64{1, 2},
	},
	{
		def:  `[{"type": "int64[2]"}]`,
		enc:  "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: [2]int64{1, 2},
	},
	{
		def:  `[{"type": "int256[]"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002",
		want: []*big.Int{big.NewInt(1), big.NewInt(2)},
	},
	{
		def:  `[{"type": "int256[3]"}]`,
		enc:  "000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003",
		want: [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
	},
}

func TestUnpack(t *testing.T) {
	for i, test := range unpackTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			def := fmt.Sprintf(`[{ "name" : "method", "outputs": %s}]`, test.def)
			abi, err := JSON(strings.NewReader(def))
			if err != nil {
				t.Fatalf("invalid ABI definition %s: %v", def, err)
			}
			encb, err := hex.DecodeString(test.enc)
			if err != nil {
				t.Fatalf("invalid hex: %s" + test.enc)
			}
			outptr := reflect.New(reflect.TypeOf(test.want))
			err = abi.Unpack(outptr.Interface(), "method", encb)
			if err := test.checkError(err); err != nil {
				t.Errorf("test %d (%v) failed: %v", i, test.def, err)
				return
			}
			out := outptr.Elem().Interface()
			if !reflect.DeepEqual(test.want, out) {
				t.Errorf("test %d (%v) failed: expected %v, got %v", i, test.def, test.want, out)
			}
		})
	}
}

var unpackMobileTests = []unpackTest{
	{
		def:  `[{"type": "uint256"}]`,
		enc:  "0000000000000000000000000000000000000000000000000000000000000001",
		want: big.NewInt(1),
	},
}

func TestUnpackMobileOnly(t *testing.T) {
	for i, test := range unpackMobileTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			def := fmt.Sprintf(`[{ "name" : "method", "outputs": %s}]`, test.def)
			abi, err := JSON(strings.NewReader(def))
			if err != nil {
				t.Fatalf("invalid ABI definition %s: %v", def, err)
			}
			encb, err := hex.DecodeString(test.enc)
			if err != nil {
				t.Fatalf("invalid hex: %s" + test.enc)
			}
			outptr := reflect.New(reflect.TypeOf(test.want))
			results := make([]interface{}, 1)
			copy(results, []interface{}{outptr.Interface()})
			err = abi.Unpack(&results, "method", encb)
			if err := test.checkError(err); err != nil {
				t.Errorf("test %d (%v) failed: %v", i, test.def, err)
				return
			}
			out := outptr.Elem().Interface()
			if !reflect.DeepEqual(test.want, out) {
				t.Errorf("test %d (%v) failed: expected %v, got %v", i, test.def, test.want, out)
			}
		})
	}
}

type methodMultiOutput struct {
	Int    *big.Int
	String string
}

func methodMultiReturn(require *require.Assertions) (ABI, []byte, methodMultiOutput) {
	const definition = `[
	{ "name" : "multi", "constant" : false, "outputs": [ { "name": "Int", "type": "uint256" }, { "name": "String", "type": "string" } ] }]`
	var expected = methodMultiOutput{big.NewInt(1), "hello"}

	abi, err := JSON(strings.NewReader(definition))
	require.NoError(err)
	// using buff to make the code readable
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005"))
	buff.Write(common.RightPadBytes([]byte(expected.String), 32))
	return abi, buff.Bytes(), expected
}

func TestMethodMultiReturn(t *testing.T) {
	type reversed struct {
		String string
		Int    *big.Int
	}

	abi, data, expected := methodMultiReturn(require.New(t))
	bigint := new(big.Int)
	var testCases = []struct {
		dest     interface{}
		expected interface{}
		error    string
		name     string
	}{{
		&methodMultiOutput{},
		&expected,
		"",
		"Can unpack into structure",
	}, {
		&reversed{},
		&reversed{expected.String, expected.Int},
		"",
		"Can unpack into reversed structure",
	}, {
		&[]interface{}{&bigint, new(string)},
		&[]interface{}{&expected.Int, &expected.String},
		"",
		"Can unpack into a slice",
	}, {
		&[2]interface{}{&bigint, new(string)},
		&[2]interface{}{&expected.Int, &expected.String},
		"",
		"Can unpack into an array",
	}, {
		&[]interface{}{new(int), new(int)},
		&[]interface{}{&expected.Int, &expected.String},
		"abi: cannot unmarshal *big.Int in to int",
		"Can not unpack into a slice with wrong types",
	}, {
		&[]interface{}{new(int)},
		&[]interface{}{},
		"abi: insufficient number of elements in the list/array for unpack, want 2, got 1",
		"Can not unpack into a slice with wrong types",
	}}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			err := abi.Unpack(tc.dest, "multi", data)
			if tc.error == "" {
				require.Nil(err, "Should be able to unpack method outputs.")
				require.Equal(tc.expected, tc.dest)
			} else {
				require.EqualError(err, tc.error)
			}
		})
	}
}

func TestMultiReturnWithArray(t *testing.T) {
	const definition = `[{"name" : "multi", "outputs": [{"type": "uint64[3]"}, {"type": "uint64"}]}]`
	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000090000000000000000000000000000000000000000000000000000000000000009"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008"))

	ret1, ret1Exp := new([3]uint64), [3]uint64{9, 9, 9}
	ret2, ret2Exp := new(uint64), uint64(8)
	if err := abi.Unpack(&[]interface{}{ret1, ret2}, "multi", buff.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*ret1, ret1Exp) {
		t.Error("array result", *ret1, "!= Expected", ret1Exp)
	}
	if *ret2 != ret2Exp {
		t.Error("int result", *ret2, "!= Expected", ret2Exp)
	}
}

func TestUnmarshal(t *testing.T) {
	const definition = `[
	{ "name" : "int", "constant" : false, "outputs": [ { "type": "uint256" } ] },
	{ "name" : "bool", "constant" : false, "outputs": [ { "type": "bool" } ] },
	{ "name" : "bytes", "constant" : false, "outputs": [ { "type": "bytes" } ] },
	{ "name" : "fixed", "constant" : false, "outputs": [ { "type": "bytes32" } ] },
	{ "name" : "multi", "constant" : false, "outputs": [ { "type": "bytes" }, { "type": "bytes" } ] },
	{ "name" : "intArraySingle", "constant" : false, "outputs": [ { "type": "uint256[3]" } ] },
	{ "name" : "addressSliceSingle", "constant" : false, "outputs": [ { "type": "address[]" } ] },
	{ "name" : "addressSliceDouble", "constant" : false, "outputs": [ { "name": "a", "type": "address[]" }, { "name": "b", "type": "address[]" } ] },
	{ "name" : "mixedBytes", "constant" : true, "outputs": [ { "name": "a", "type": "bytes" }, { "name": "b", "type": "bytes32" } ] }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)

	// marshall mixed bytes (mixedBytes)
	p0, p0Exp := []byte{}, common.Hex2Bytes("01020000000000000000")
	p1, p1Exp := [32]byte{}, common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000ddeeff")
	mixedBytes := []interface{}{&p0, &p1}

	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000ddeeff"))
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000a"))
	buff.Write(common.Hex2Bytes("0102000000000000000000000000000000000000000000000000000000000000"))

	err = abi.Unpack(&mixedBytes, "mixedBytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	} else {
		if !bytes.Equal(p0, p0Exp) {
			t.Errorf("unexpected value unpacked: want %x, got %x", p0Exp, p0)
		}

		if !bytes.Equal(p1[:], p1Exp) {
			t.Errorf("unexpected value unpacked: want %x, got %x", p1Exp, p1)
		}
	}

	// marshal int
	var Int *big.Int
	err = abi.Unpack(&Int, "int", common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	if err != nil {
		t.Error(err)
	}

	if Int == nil || Int.Cmp(big.NewInt(1)) != 0 {
		t.Error("expected Int to be 1 got", Int)
	}

	// marshal bool
	var Bool bool
	err = abi.Unpack(&Bool, "bool", common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	if err != nil {
		t.Error(err)
	}

	if !Bool {
		t.Error("expected Bool to be true")
	}

	// marshal dynamic bytes max length 32
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	bytesOut := common.RightPadBytes([]byte("hello"), 32)
	buff.Write(bytesOut)

	var Bytes []byte
	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(Bytes, bytesOut) {
		t.Errorf("expected %x got %x", bytesOut, Bytes)
	}

	// marshall dynamic bytes max length 64
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	bytesOut = common.RightPadBytes([]byte("hello"), 64)
	buff.Write(bytesOut)

	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(Bytes, bytesOut) {
		t.Errorf("expected %x got %x", bytesOut, Bytes)
	}

	// marshall dynamic bytes max length 64
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000003f"))
	bytesOut = common.RightPadBytes([]byte("hello"), 64)
	buff.Write(bytesOut)

	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(Bytes, bytesOut[:len(bytesOut)-1]) {
		t.Errorf("expected %x got %x", bytesOut[:len(bytesOut)-1], Bytes)
	}

	// marshal dynamic bytes output empty
	err = abi.Unpack(&Bytes, "bytes", nil)
	if err == nil {
		t.Error("expected error")
	}

	// marshal dynamic bytes length 5
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005"))
	buff.Write(common.RightPadBytes([]byte("hello"), 32))

	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(Bytes, []byte("hello")) {
		t.Errorf("expected %x got %x", bytesOut, Bytes)
	}

	// marshal dynamic bytes length 5
	buff.Reset()
	buff.Write(common.RightPadBytes([]byte("hello"), 32))

	var hash common.Hash
	err = abi.Unpack(&hash, "fixed", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	helloHash := common.BytesToHash(common.RightPadBytes([]byte("hello"), 32))
	if hash != helloHash {
		t.Errorf("Expected %x to equal %x", hash, helloHash)
	}

	// marshal error
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err == nil {
		t.Error("expected error")
	}

	err = abi.Unpack(&Bytes, "multi", make([]byte, 64))
	if err == nil {
		t.Error("expected error")
	}

	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003"))
	// marshal int array
	var intArray [3]*big.Int
	err = abi.Unpack(&intArray, "intArraySingle", buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	var testAgainstIntArray [3]*big.Int
	testAgainstIntArray[0] = big.NewInt(1)
	testAgainstIntArray[1] = big.NewInt(2)
	testAgainstIntArray[2] = big.NewInt(3)

	for i, Int := range intArray {
		if Int.Cmp(testAgainstIntArray[i]) != 0 {
			t.Errorf("expected %v, got %v", testAgainstIntArray[i], Int)
		}
	}
	// marshal address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"))

	var outAddr []common.Address
	err = abi.Unpack(&outAddr, "addressSliceSingle", buff.Bytes())
	if err != nil {
		t.Fatal("didn't expect error:", err)
	}

	if len(outAddr) != 1 {
		t.Fatal("expected 1 item, got", len(outAddr))
	}

	if outAddr[0] != (common.Address{1}) {
		t.Errorf("expected %x, got %x", common.Address{1}, outAddr[0])
	}

	// marshal multiple address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000200000000000000000000000000000000000000"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000300000000000000000000000000000000000000"))

	var outAddrStruct struct {
		A []common.Address
		B []common.Address
	}
	err = abi.Unpack(&outAddrStruct, "addressSliceDouble", buff.Bytes())
	if err != nil {
		t.Fatal("didn't expect error:", err)
	}

	if len(outAddrStruct.A) != 1 {
		t.Fatal("expected 1 item, got", len(outAddrStruct.A))
	}

	if outAddrStruct.A[0] != (common.Address{1}) {
		t.Errorf("expected %x, got %x", common.Address{1}, outAddrStruct.A[0])
	}

	if len(outAddrStruct.B) != 2 {
		t.Fatal("expected 1 item, got", len(outAddrStruct.B))
	}

	if outAddrStruct.B[0] != (common.Address{2}) {
		t.Errorf("expected %x, got %x", common.Address{2}, outAddrStruct.B[0])
	}
	if outAddrStruct.B[1] != (common.Address{3}) {
		t.Errorf("expected %x, got %x", common.Address{3}, outAddrStruct.B[1])
	}

	// marshal invalid address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000100"))

	err = abi.Unpack(&outAddr, "addressSliceSingle", buff.Bytes())
	if err == nil {
		t.Fatal("expected error:", err)
	}
}
