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
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSimpleMethodUnpack(t *testing.T) {
	for i, test := range []struct {
		def              string      // definition of the **output** ABI params
		marshalledOutput []byte      // evm return data
		expectedOut      interface{} // the expected output
		outVar           string      // the output variable (e.g. uint32, *big.Int, etc)
		err              string      // empty or error if expected
	}{
		{
			`[ { "type": "bool" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			bool(true),
			"bool",
			"",
		},
		{
			`[ { "type": "uint32" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			uint32(1),
			"uint32",
			"",
		},
		{
			`[ { "type": "uint32" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			nil,
			"uint16",
			"abi: cannot unmarshal uint32 in to uint16",
		},
		{
			`[ { "type": "uint17" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			nil,
			"uint16",
			"abi: cannot unmarshal *big.Int in to uint16",
		},
		{
			`[ { "type": "uint17" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			big.NewInt(1),
			"*big.Int",
			"",
		},

		{
			`[ { "type": "int32" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			int32(1),
			"int32",
			"",
		},
		{
			`[ { "type": "int32" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			nil,
			"int16",
			"abi: cannot unmarshal int32 in to int16",
		},
		{
			`[ { "type": "int17" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			nil,
			"int16",
			"abi: cannot unmarshal *big.Int in to int16",
		},
		{
			`[ { "type": "int17" } ]`,
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
			big.NewInt(1),
			"*big.Int",
			"",
		},

		{
			`[ { "type": "address" } ]`,
			common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"),
			common.Address{1},
			"address",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			"bytes",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			"hash",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			"interface",
			"",
		},
		{
			`[ { "type": "function" } ]`,
			common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000"),
			[24]byte{1},
			"function",
			"",
		},
	} {
		abiDefinition := fmt.Sprintf(`[{ "name" : "method", "outputs": %s}]`, test.def)
		abi, err := JSON(strings.NewReader(abiDefinition))
		if err != nil {
			t.Errorf("%d failed. %v", i, err)
			continue
		}

		var outvar interface{}
		switch test.outVar {
		case "bool":
			var v bool
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "uint8":
			var v uint8
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "uint16":
			var v uint16
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "uint32":
			var v uint32
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "uint64":
			var v uint64
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "int8":
			var v int8
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "int16":
			var v int16
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "int32":
			var v int32
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "int64":
			var v int64
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "*big.Int":
			var v *big.Int
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "address":
			var v common.Address
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "bytes":
			var v []byte
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "hash":
			var v common.Hash
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v.Bytes()[:]
		case "function":
			var v [24]byte
			err = abi.Unpack(&v, "method", test.marshalledOutput)
			outvar = v
		case "interface":
			err = abi.Unpack(&outvar, "method", test.marshalledOutput)
		default:
			t.Errorf("unsupported type '%v' please add it to the switch statement in this test", test.outVar)
			continue
		}

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
			continue
		}

		if err == nil {
			if !reflect.DeepEqual(test.expectedOut, outvar) {
				t.Errorf("%d failed. Output error: expected %v, got %v", i, test.expectedOut, outvar)
			}
		}
	}
}

func TestUnpackSetInterfaceSlice(t *testing.T) {
	var (
		var1 = new(uint8)
		var2 = new(uint8)
	)
	out := []interface{}{var1, var2}
	abi, err := JSON(strings.NewReader(`[{"type":"function", "name":"ints", "outputs":[{"type":"uint8"}, {"type":"uint8"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	marshalledReturn := append(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"), common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")...)
	err = abi.Unpack(&out, "ints", marshalledReturn)
	if err != nil {
		t.Fatal(err)
	}
	if *var1 != 1 {
		t.Error("expected var1 to be 1, got", *var1)
	}
	if *var2 != 2 {
		t.Error("expected var2 to be 2, got", *var2)
	}

	out = []interface{}{var1}
	err = abi.Unpack(&out, "ints", marshalledReturn)

	expErr := "abi: cannot marshal in to slices of unequal size (require: 2, got: 1)"
	if err == nil || err.Error() != expErr {
		t.Error("expected err:", expErr, "Got:", err)
	}
}

func TestUnpackSetInterfaceArrayOutput(t *testing.T) {
	var (
		var1 = new([1]uint32)
		var2 = new([1]uint32)
	)
	out := []interface{}{var1, var2}
	abi, err := JSON(strings.NewReader(`[{"type":"function", "name":"ints", "outputs":[{"type":"uint32[1]"}, {"type":"uint32[1]"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	marshalledReturn := append(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"), common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")...)
	err = abi.Unpack(&out, "ints", marshalledReturn)
	if err != nil {
		t.Fatal(err)
	}

	if *var1 != [1]uint32{1} {
		t.Error("expected var1 to be [1], got", *var1)
	}
	if *var2 != [1]uint32{2} {
		t.Error("expected var2 to be [2], got", *var2)
	}
}

func TestMultiReturnWithStruct(t *testing.T) {
	const definition = `[
	{ "name" : "multi", "constant" : false, "outputs": [ { "name": "Int", "type": "uint256" }, { "name": "String", "type": "string" } ] }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	// using buff to make the code readable
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005"))
	stringOut := "hello"
	buff.Write(common.RightPadBytes([]byte(stringOut), 32))

	var inter struct {
		Int    *big.Int
		String string
	}
	err = abi.Unpack(&inter, "multi", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if inter.Int == nil || inter.Int.Cmp(big.NewInt(1)) != 0 {
		t.Error("expected Int to be 1 got", inter.Int)
	}

	if inter.String != stringOut {
		t.Error("expected String to be", stringOut, "got", inter.String)
	}

	var reversed struct {
		String string
		Int    *big.Int
	}

	err = abi.Unpack(&reversed, "multi", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if reversed.Int == nil || reversed.Int.Cmp(big.NewInt(1)) != 0 {
		t.Error("expected Int to be 1 got", reversed.Int)
	}

	if reversed.String != stringOut {
		t.Error("expected String to be", stringOut, "got", reversed.String)
	}
}

func TestMultiReturnWithSlice(t *testing.T) {
	const definition = `[
	{ "name" : "multi", "constant" : false, "outputs": [ { "name": "Int", "type": "uint256" }, { "name": "String", "type": "string" } ] }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	// using buff to make the code readable
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005"))
	stringOut := "hello"
	buff.Write(common.RightPadBytes([]byte(stringOut), 32))

	var inter []interface{}
	err = abi.Unpack(&inter, "multi", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if len(inter) != 2 {
		t.Fatal("expected 2 results got", len(inter))
	}

	if num, ok := inter[0].(*big.Int); !ok || num.Cmp(big.NewInt(1)) != 0 {
		t.Error("expected index 0 to be 1 got", num)
	}

	if str, ok := inter[1].(string); !ok || str != stringOut {
		t.Error("expected index 1 to be", stringOut, "got", str)
	}
}

func TestMarshalArrays(t *testing.T) {
	const definition = `[
	{ "name" : "bytes32", "constant" : false, "outputs": [ { "type": "bytes32" } ] },
	{ "name" : "bytes10", "constant" : false, "outputs": [ { "type": "bytes10" } ] }
	]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	output := common.LeftPadBytes([]byte{1}, 32)

	var bytes10 [10]byte
	err = abi.Unpack(&bytes10, "bytes32", output)
	if err == nil || err.Error() != "abi: cannot unmarshal src (len=32) in to dst (len=10)" {
		t.Error("expected error or bytes32 not be assignable to bytes10:", err)
	}

	var bytes32 [32]byte
	err = abi.Unpack(&bytes32, "bytes32", output)
	if err != nil {
		t.Error("didn't expect error:", err)
	}
	if !bytes.Equal(bytes32[:], output) {
		t.Error("expected bytes32[31] to be 1 got", bytes32[31])
	}

	type (
		B10 [10]byte
		B32 [32]byte
	)

	var b10 B10
	err = abi.Unpack(&b10, "bytes32", output)
	if err == nil || err.Error() != "abi: cannot unmarshal src (len=32) in to dst (len=10)" {
		t.Error("expected error or bytes32 not be assignable to bytes10:", err)
	}

	var b32 B32
	err = abi.Unpack(&b32, "bytes32", output)
	if err != nil {
		t.Error("didn't expect error:", err)
	}
	if !bytes.Equal(b32[:], output) {
		t.Error("expected bytes32[31] to be 1 got", bytes32[31])
	}

	output[10] = 1
	var shortAssignLong [32]byte
	err = abi.Unpack(&shortAssignLong, "bytes10", output)
	if err != nil {
		t.Error("didn't expect error:", err)
	}
	if !bytes.Equal(output, shortAssignLong[:]) {
		t.Errorf("expected %x to be %x", shortAssignLong, output)
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

	// marshall dynamic bytes max length 63
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000003f"))
	bytesOut = common.RightPadBytes([]byte("hello"), 63)
	buff.Write(bytesOut)

	err = abi.Unpack(&Bytes, "bytes", buff.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(Bytes, bytesOut) {
		t.Errorf("expected %x got %x", bytesOut, Bytes)
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

	// marshal mixed bytes
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	fixed := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	buff.Write(fixed)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	bytesOut = common.RightPadBytes([]byte("hello"), 32)
	buff.Write(bytesOut)

	var out []interface{}
	err = abi.Unpack(&out, "mixedBytes", buff.Bytes())
	if err != nil {
		t.Fatal("didn't expect error:", err)
	}

	if !bytes.Equal(bytesOut, out[0].([]byte)) {
		t.Errorf("expected %x, got %x", bytesOut, out[0])
	}

	if !bytes.Equal(fixed, out[1].([]byte)) {
		t.Errorf("expected %x, got %x", fixed, out[1])
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
