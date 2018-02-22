// Copyright 2017 The go-ethereum Authors
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
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPack(t *testing.T) {
	for i, test := range []struct {
		typ string

		input  interface{}
		output []byte
	}{
		{
			"uint8",
			uint8(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint8[]",
			[]uint8{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16",
			uint16(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16[]",
			[]uint16{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32",
			uint32(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32[]",
			[]uint32{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64",
			uint64(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64[]",
			[]uint64{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256",
			big.NewInt(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256[]",
			[]*big.Int{big.NewInt(1), big.NewInt(2)},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8",
			int8(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8[]",
			[]int8{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16",
			int16(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16[]",
			[]int16{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32",
			int32(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32[]",
			[]int32{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64",
			int64(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64[]",
			[]int64{1, 2},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256",
			big.NewInt(2),
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256[]",
			[]*big.Int{big.NewInt(1), big.NewInt(2)},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"bytes1",
			[1]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes2",
			[2]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes3",
			[3]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes4",
			[4]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes5",
			[5]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes6",
			[6]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes7",
			[7]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes8",
			[8]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes9",
			[9]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes10",
			[10]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes11",
			[11]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes12",
			[12]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes13",
			[13]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes14",
			[14]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes15",
			[15]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes16",
			[16]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes17",
			[17]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes18",
			[18]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes19",
			[19]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes20",
			[20]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes21",
			[21]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes22",
			[22]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes23",
			[23]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			[24]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			[24]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes25",
			[25]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes26",
			[26]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes27",
			[27]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes28",
			[28]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes29",
			[29]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes30",
			[30]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes31",
			[31]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes32",
			[32]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"address[]",
			[]common.Address{{1}, {2}},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000"),
		},
		{
			"bytes32[]",
			[]common.Hash{{1}, {2}},
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000201000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"function",
			[24]byte{1},
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"string",
			"foobar",
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000006666f6f6261720000000000000000000000000000000000000000000000000000"),
		},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatalf("%v failed. Unexpected parse error: %v", i, err)
		}

		output, err := typ.pack(reflect.ValueOf(test.input))
		if err != nil {
			t.Fatalf("%v failed. Unexpected pack error: %v", i, err)
		}

		if !bytes.Equal(output, test.output) {
			t.Errorf("%d failed. Expected bytes: '%x' Got: '%x'", i, test.output, output)
		}
	}
}

func TestMethodPack(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Fatal(err)
	}

	sig := abi.Methods["slice"].Id()
	sig = append(sig, common.LeftPadBytes([]byte{1}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)

	packed, err := abi.Pack("slice", []uint32{1, 2})
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	var addrA, addrB = common.Address{1}, common.Address{2}
	sig = abi.Methods["sliceAddress"].Id()
	sig = append(sig, common.LeftPadBytes([]byte{32}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)
	sig = append(sig, common.LeftPadBytes(addrA[:], 32)...)
	sig = append(sig, common.LeftPadBytes(addrB[:], 32)...)

	packed, err = abi.Pack("sliceAddress", []common.Address{addrA, addrB})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	var addrC, addrD = common.Address{3}, common.Address{4}
	sig = abi.Methods["sliceMultiAddress"].Id()
	sig = append(sig, common.LeftPadBytes([]byte{64}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{160}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)
	sig = append(sig, common.LeftPadBytes(addrA[:], 32)...)
	sig = append(sig, common.LeftPadBytes(addrB[:], 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)
	sig = append(sig, common.LeftPadBytes(addrC[:], 32)...)
	sig = append(sig, common.LeftPadBytes(addrD[:], 32)...)

	packed, err = abi.Pack("sliceMultiAddress", []common.Address{addrA, addrB}, []common.Address{addrC, addrD})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	sig = abi.Methods["slice256"].Id()
	sig = append(sig, common.LeftPadBytes([]byte{1}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)

	packed, err = abi.Pack("slice256", []*big.Int{big.NewInt(1), big.NewInt(2)})
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}
}

func TestPackNumber(t *testing.T) {
	tests := []struct {
		value  reflect.Value
		packed []byte
	}{
		// Protocol limits
		{reflect.ValueOf(0), common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")},
		{reflect.ValueOf(1), common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")},
		{reflect.ValueOf(-1), common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")},

		// Type corner cases
		{reflect.ValueOf(uint8(math.MaxUint8)), common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000ff")},
		{reflect.ValueOf(uint16(math.MaxUint16)), common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000ffff")},
		{reflect.ValueOf(uint32(math.MaxUint32)), common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000ffffffff")},
		{reflect.ValueOf(uint64(math.MaxUint64)), common.Hex2Bytes("000000000000000000000000000000000000000000000000ffffffffffffffff")},

		{reflect.ValueOf(int8(math.MaxInt8)), common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000007f")},
		{reflect.ValueOf(int16(math.MaxInt16)), common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000007fff")},
		{reflect.ValueOf(int32(math.MaxInt32)), common.Hex2Bytes("000000000000000000000000000000000000000000000000000000007fffffff")},
		{reflect.ValueOf(int64(math.MaxInt64)), common.Hex2Bytes("0000000000000000000000000000000000000000000000007fffffffffffffff")},

		{reflect.ValueOf(int8(math.MinInt8)), common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80")},
		{reflect.ValueOf(int16(math.MinInt16)), common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8000")},
		{reflect.ValueOf(int32(math.MinInt32)), common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffff80000000")},
		{reflect.ValueOf(int64(math.MinInt64)), common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffff8000000000000000")},
	}
	for i, tt := range tests {
		packed := packNum(tt.value)
		if !bytes.Equal(packed, tt.packed) {
			t.Errorf("test %d: pack mismatch: have %x, want %x", i, packed, tt.packed)
		}
	}
}
