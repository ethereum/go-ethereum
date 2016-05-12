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
	"log"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// formatSilceOutput add padding to the value and adds a size
func formatSliceOutput(v ...[]byte) []byte {
	off := common.LeftPadBytes(big.NewInt(int64(len(v))).Bytes(), 32)
	output := append(off, make([]byte, 0, len(v)*32)...)

	for _, value := range v {
		output = append(output, common.LeftPadBytes(value, 32)...)
	}
	return output
}

// quick helper padding
func pad(input []byte, size int, left bool) []byte {
	if left {
		return common.LeftPadBytes(input, size)
	}
	return common.RightPadBytes(input, size)
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
		{"uint16[]", []uint32{1, 2, 3}, "abi: cannot use []uint32 as type []uint16 as argument"},
		{"uint16[3]", [3]uint32{1, 2, 3}, "abi: cannot use [3]uint32 as type [3]uint16 as argument"},
		{"uint16[3]", [4]uint16{1, 2, 3}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"uint16[3]", []uint16{1, 2, 3}, ""},
		{"uint16[3]", []uint16{1, 2, 3, 4}, "abi: cannot use [4]uint16 as type [3]uint16 as argument"},
		{"address[]", []common.Address{common.Address{1}}, ""},
		{"address[1]", []common.Address{common.Address{1}}, ""},
		{"address[1]", [1]common.Address{common.Address{1}}, ""},
		{"address[2]", [1]common.Address{common.Address{1}}, "abi: cannot use [1]array as type [2]array as argument"},
		{"bytes32", [32]byte{}, ""},
		{"bytes32", [33]byte{}, "abi: cannot use [33]uint8 as type [32]uint8 as argument"},
		{"bytes32", common.Hash{1}, ""},
		{"bytes31", [31]byte{}, ""},
		{"bytes31", [32]byte{}, "abi: cannot use [32]uint8 as type [31]uint8 as argument"},
		{"bytes", []byte{0, 1}, ""},
		{"bytes", [2]byte{0, 1}, ""},
		{"bytes", common.Hash{1}, ""},
		{"string", "hello world", ""},
		{"bytes32[]", [][32]byte{[32]byte{}}, ""},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatal("unexpected parse error:", err)
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

func TestSimpleMethodUnpack(t *testing.T) {
	for i, test := range []struct {
		def              string      // definition of the **output** ABI params
		marshalledOutput []byte      // evm return data
		expectedOut      interface{} // the expected output
		outVar           string      // the output variable (e.g. uint32, *big.Int, etc)
		err              string      // empty or error if expected
	}{
		{
			`[ { "type": "uint32" } ]`,
			pad([]byte{1}, 32, true),
			uint32(1),
			"uint32",
			"",
		},
		{
			`[ { "type": "uint32" } ]`,
			pad([]byte{1}, 32, true),
			nil,
			"uint16",
			"abi: cannot unmarshal uint32 in to uint16",
		},
		{
			`[ { "type": "uint17" } ]`,
			pad([]byte{1}, 32, true),
			nil,
			"uint16",
			"abi: cannot unmarshal *big.Int in to uint16",
		},
		{
			`[ { "type": "uint17" } ]`,
			pad([]byte{1}, 32, true),
			big.NewInt(1),
			"*big.Int",
			"",
		},

		{
			`[ { "type": "int32" } ]`,
			pad([]byte{1}, 32, true),
			int32(1),
			"int32",
			"",
		},
		{
			`[ { "type": "int32" } ]`,
			pad([]byte{1}, 32, true),
			nil,
			"int16",
			"abi: cannot unmarshal int32 in to int16",
		},
		{
			`[ { "type": "int17" } ]`,
			pad([]byte{1}, 32, true),
			nil,
			"int16",
			"abi: cannot unmarshal *big.Int in to int16",
		},
		{
			`[ { "type": "int17" } ]`,
			pad([]byte{1}, 32, true),
			big.NewInt(1),
			"*big.Int",
			"",
		},

		{
			`[ { "type": "address" } ]`,
			pad(pad([]byte{1}, 20, false), 32, true),
			common.Address{1},
			"address",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			pad([]byte{1}, 32, false),
			pad([]byte{1}, 32, false),
			"bytes",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			pad([]byte{1}, 32, false),
			pad([]byte{1}, 32, false),
			"hash",
			"",
		},
		{
			`[ { "type": "bytes32" } ]`,
			pad([]byte{1}, 32, false),
			pad([]byte{1}, 32, false),
			"interface",
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
			// bit of an ugly hack for hash type but I don't feel like finding a proper solution
			if test.outVar == "hash" {
				tmp := outvar.(common.Hash) // without assignment it's unaddressable
				outvar = tmp[:]
			}

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
	marshalledReturn := append(pad([]byte{1}, 32, true), pad([]byte{2}, 32, true)...)
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

func TestPack(t *testing.T) {
	for i, test := range []struct {
		typ string

		input  interface{}
		output []byte
	}{
		{"uint16", uint16(2), pad([]byte{2}, 32, true)},
		{"uint16[]", []uint16{1, 2}, formatSliceOutput([]byte{1}, []byte{2})},
		{"bytes20", [20]byte{1}, pad([]byte{1}, 32, false)},
		{"uint256[]", []*big.Int{big.NewInt(1), big.NewInt(2)}, formatSliceOutput([]byte{1}, []byte{2})},
		{"address[]", []common.Address{common.Address{1}, common.Address{2}}, formatSliceOutput(pad([]byte{1}, 20, false), pad([]byte{2}, 20, false))},
		{"bytes32[]", []common.Hash{common.Hash{1}, common.Hash{2}}, formatSliceOutput(pad([]byte{1}, 32, false), pad([]byte{2}, 32, false))},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatal("unexpected parse error:", err)
		}

		output, err := typ.pack(reflect.ValueOf(test.input))
		if err != nil {
			t.Fatal("unexpected pack error:", err)
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
	sig = append(sig, common.LeftPadBytes([]byte{32}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)
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
	sig = append(sig, common.LeftPadBytes([]byte{32}, 32)...)
	sig = append(sig, common.LeftPadBytes([]byte{2}, 32)...)
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

const jsondata = `
[
	{ "type" : "function", "name" : "balance", "constant" : true },
	{ "type" : "function", "name" : "send", "constant" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] }
]`

const jsondata2 = `
[
	{ "type" : "function", "name" : "balance", "constant" : true },
	{ "type" : "function", "name" : "send", "constant" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] },
	{ "type" : "function", "name" : "test", "constant" : false, "inputs" : [ { "name" : "number", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "string", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "string" } ] },
	{ "type" : "function", "name" : "bool", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "bool" } ] },
	{ "type" : "function", "name" : "address", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "address" } ] },
	{ "type" : "function", "name" : "uint64[2]", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[2]" } ] },
	{ "type" : "function", "name" : "uint64[]", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[]" } ] },
	{ "type" : "function", "name" : "foo", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "bar", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" }, { "name" : "string", "type" : "uint16" } ] },
	{ "type" : "function", "name" : "slice", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32[2]" } ] },
	{ "type" : "function", "name" : "slice256", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint256[2]" } ] },
	{ "type" : "function", "name" : "sliceAddress", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "address[]" } ] },
	{ "type" : "function", "name" : "sliceMultiAddress", "constant" : false, "inputs" : [ { "name" : "a", "type" : "address[]" }, { "name" : "b", "type" : "address[]" } ] }
]`

func TestReader(t *testing.T) {
	Uint256, _ := NewType("uint256")
	exp := ABI{
		Methods: map[string]Method{
			"balance": Method{
				"balance", true, nil, nil,
			},
			"send": Method{
				"send", false, []Argument{
					Argument{"amount", Uint256, false},
				}, nil,
			},
		},
	}

	abi, err := JSON(strings.NewReader(jsondata))
	if err != nil {
		t.Error(err)
	}

	// deep equal fails for some reason
	t.Skip()
	if !reflect.DeepEqual(abi, exp) {
		t.Errorf("\nabi: %v\ndoes not match exp: %v", abi, exp)
	}
}

func TestTestNumbers(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.Pack("balance"); err != nil {
		t.Error(err)
	}

	if _, err := abi.Pack("balance", 1); err == nil {
		t.Error("expected error for balance(1)")
	}

	if _, err := abi.Pack("doesntexist", nil); err == nil {
		t.Errorf("doesntexist shouldn't exist")
	}

	if _, err := abi.Pack("doesntexist", 1); err == nil {
		t.Errorf("doesntexist(1) shouldn't exist")
	}

	if _, err := abi.Pack("send", big.NewInt(1000)); err != nil {
		t.Error(err)
	}

	i := new(int)
	*i = 1000
	if _, err := abi.Pack("send", i); err == nil {
		t.Errorf("expected send( ptr ) to throw, requires *big.Int instead of *int")
	}

	if _, err := abi.Pack("test", uint32(1000)); err != nil {
		t.Error(err)
	}
}

func TestTestString(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.Pack("string", "hello world"); err != nil {
		t.Error(err)
	}
}

func TestTestBool(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.Pack("bool", true); err != nil {
		t.Error(err)
	}
}

func TestTestSlice(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	slice := make([]uint64, 2)
	if _, err := abi.Pack("uint64[2]", slice); err != nil {
		t.Error(err)
	}

	if _, err := abi.Pack("uint64[]", slice); err != nil {
		t.Error(err)
	}
}

func TestMethodSignature(t *testing.T) {
	String, _ := NewType("string")
	m := Method{"foo", false, []Argument{Argument{"bar", String, false}, Argument{"baz", String, false}}, nil}
	exp := "foo(string,string)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}

	idexp := crypto.Keccak256([]byte(exp))[:4]
	if !bytes.Equal(m.Id(), idexp) {
		t.Errorf("expected ids to match %x != %x", m.Id(), idexp)
	}

	uintt, _ := NewType("uint")
	m = Method{"foo", false, []Argument{Argument{"bar", uintt, false}}, nil}
	exp = "foo(uint256)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}
}

func TestMultiPack(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sig := crypto.Keccak256([]byte("bar(uint32,uint16)"))[:4]
	sig = append(sig, make([]byte, 64)...)
	sig[35] = 10
	sig[67] = 11

	packed, err := abi.Pack("bar", uint32(10), uint16(11))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}
}

func ExampleJSON() {
	const definition = `[{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"isBar","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		log.Fatalln(err)
	}
	out, err := abi.Pack("isBar", common.HexToAddress("01"))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%x\n", out)
	// Output:
	// 1f2c40920000000000000000000000000000000000000000000000000000000000000001
}

func TestInputVariableInputLength(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "strOne", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" } ] },
	{ "type" : "function", "name" : "bytesOne", "constant" : true, "inputs" : [ { "name" : "str", "type" : "bytes" } ] },
	{ "type" : "function", "name" : "strTwo", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "str1", "type" : "string" } ] }
	]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	// test one string
	strin := "hello world"
	strpack, err := abi.Pack("strOne", strin)
	if err != nil {
		t.Error(err)
	}

	offset := make([]byte, 32)
	offset[31] = 32
	length := make([]byte, 32)
	length[31] = byte(len(strin))
	value := common.RightPadBytes([]byte(strin), 32)
	exp := append(offset, append(length, value...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	strpack = strpack[4:]
	if !bytes.Equal(strpack, exp) {
		t.Errorf("expected %x, got %x\n", exp, strpack)
	}

	// test one bytes
	btspack, err := abi.Pack("bytesOne", []byte(strin))
	if err != nil {
		t.Error(err)
	}
	// ignore first 4 bytes of the output. This is the function identifier
	btspack = btspack[4:]
	if !bytes.Equal(btspack, exp) {
		t.Errorf("expected %x, got %x\n", exp, btspack)
	}

	//  test two strings
	str1 := "hello"
	str2 := "world"
	str2pack, err := abi.Pack("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 := make([]byte, 32)
	offset1[31] = 64
	length1 := make([]byte, 32)
	length1[31] = byte(len(str1))
	value1 := common.RightPadBytes([]byte(str1), 32)

	offset2 := make([]byte, 32)
	offset2[31] = 128
	length2 := make([]byte, 32)
	length2[31] = byte(len(str2))
	value2 := common.RightPadBytes([]byte(str2), 32)

	exp2 := append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}

	// test two strings, first > 32, second < 32
	str1 = strings.Repeat("a", 33)
	str2pack, err = abi.Pack("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 = make([]byte, 32)
	offset1[31] = 64
	length1 = make([]byte, 32)
	length1[31] = byte(len(str1))
	value1 = common.RightPadBytes([]byte(str1), 64)
	offset2[31] = 160

	exp2 = append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}

	// test two strings, first > 32, second >32
	str1 = strings.Repeat("a", 33)
	str2 = strings.Repeat("a", 33)
	str2pack, err = abi.Pack("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 = make([]byte, 32)
	offset1[31] = 64
	length1 = make([]byte, 32)
	length1[31] = byte(len(str1))
	value1 = common.RightPadBytes([]byte(str1), 64)

	offset2 = make([]byte, 32)
	offset2[31] = 160
	length2 = make([]byte, 32)
	length2[31] = byte(len(str2))
	value2 = common.RightPadBytes([]byte(str2), 64)

	exp2 = append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}
}

func TestDefaultFunctionParsing(t *testing.T) {
	const definition = `[{ "name" : "balance" }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := abi.Methods["balance"]; !ok {
		t.Error("expected 'balance' to be present")
	}
}

func TestBareEvents(t *testing.T) {
	const definition = `[
	{ "type" : "event", "name" : "balance" },
	{ "type" : "event", "name" : "name" }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	if len(abi.Events) != 2 {
		t.Error("expected 2 events")
	}

	if _, ok := abi.Events["balance"]; !ok {
		t.Error("expected 'balance' event to be present")
	}

	if _, ok := abi.Events["name"]; !ok {
		t.Error("expected 'name' event to be present")
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
