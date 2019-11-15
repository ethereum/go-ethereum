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
	"log"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/crypto"
)

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
	{ "type" : "function", "name" : "sliceMultiAddress", "constant" : false, "inputs" : [ { "name" : "a", "type" : "address[]" }, { "name" : "b", "type" : "address[]" } ] },
	{ "type" : "function", "name" : "nestedArray", "constant" : false, "inputs" : [ { "name" : "a", "type" : "uint256[2][2]" }, { "name" : "b", "type" : "address[]" } ] },
	{ "type" : "function", "name" : "nestedArray2", "constant" : false, "inputs" : [ { "name" : "a", "type" : "uint8[][2]" } ] },
	{ "type" : "function", "name" : "nestedSlice", "constant" : false, "inputs" : [ { "name" : "a", "type" : "uint8[][]" } ] }
]`

func TestReader(t *testing.T) {
	Uint256, _ := NewType("uint256", nil)
	exp := ABI{
		Methods: map[string]Method{
			"balance": {
				"balance", true, nil, nil,
			},
			"send": {
				"send", false, []Argument{
					{"amount", Uint256, false},
				}, nil,
			},
		},
	}

	abi, err := JSON(strings.NewReader(jsondata))
	if err != nil {
		t.Error(err)
	}

	// deep equal fails for some reason
	for name, expM := range exp.Methods {
		gotM, exist := abi.Methods[name]
		if !exist {
			t.Errorf("Missing expected method %v", name)
		}
		if !reflect.DeepEqual(gotM, expM) {
			t.Errorf("\nGot abi method: \n%v\ndoes not match expected method\n%v", gotM, expM)
		}
	}

	for name, gotM := range abi.Methods {
		expM, exist := exp.Methods[name]
		if !exist {
			t.Errorf("Found extra method %v", name)
		}
		if !reflect.DeepEqual(gotM, expM) {
			t.Errorf("\nGot abi method: \n%v\ndoes not match expected method\n%v", gotM, expM)
		}
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
	String, _ := NewType("string", nil)
	m := Method{"foo", false, []Argument{{"bar", String, false}, {"baz", String, false}}, nil}
	exp := "foo(string,string)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}

	idexp := crypto.Keccak256([]byte(exp))[:4]
	if !bytes.Equal(m.Id(), idexp) {
		t.Errorf("expected ids to match %x != %x", m.Id(), idexp)
	}

	uintt, _ := NewType("uint256", nil)
	m = Method{"foo", false, []Argument{{"bar", uintt, false}}, nil}
	exp = "foo(uint256)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}

	// Method with tuple arguments
	s, _ := NewType("tuple", []ArgumentMarshaling{
		{Name: "a", Type: "int256"},
		{Name: "b", Type: "int256[]"},
		{Name: "c", Type: "tuple[]", Components: []ArgumentMarshaling{
			{Name: "x", Type: "int256"},
			{Name: "y", Type: "int256"},
		}},
		{Name: "d", Type: "tuple[2]", Components: []ArgumentMarshaling{
			{Name: "x", Type: "int256"},
			{Name: "y", Type: "int256"},
		}},
	})
	m = Method{"foo", false, []Argument{{"s", s, false}, {"bar", String, false}}, nil}
	exp = "foo((int256,int256[],(int256,int256)[],(int256,int256)[2]),string)"
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

func TestInputFixedArrayAndVariableInputLength(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "fixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr", "type" : "uint256[2]" } ] },
	{ "type" : "function", "name" : "fixedArrBytes", "constant" : true, "inputs" : [ { "name" : "str", "type" : "bytes" }, { "name" : "fixedArr", "type" : "uint256[2]" } ] },
    { "type" : "function", "name" : "mixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr", "type": "uint256[2]" }, { "name" : "dynArr", "type": "uint256[]" } ] },
    { "type" : "function", "name" : "doubleFixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr1", "type": "uint256[2]" }, { "name" : "fixedArr2", "type": "uint256[3]" } ] },
    { "type" : "function", "name" : "multipleMixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr1", "type": "uint256[2]" }, { "name" : "dynArr", "type" : "uint256[]" }, { "name" : "fixedArr2", "type" : "uint256[3]" } ] }
	]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Error(err)
	}

	// test string, fixed array uint256[2]
	strin := "hello world"
	arrin := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedArrStrPack, err := abi.Pack("fixedArrStr", strin, arrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	offset := make([]byte, 32)
	offset[31] = 96
	length := make([]byte, 32)
	length[31] = byte(len(strin))
	strvalue := common.RightPadBytes([]byte(strin), 32)
	arrinvalue1 := common.LeftPadBytes(arrin[0].Bytes(), 32)
	arrinvalue2 := common.LeftPadBytes(arrin[1].Bytes(), 32)
	exp := append(offset, arrinvalue1...)
	exp = append(exp, arrinvalue2...)
	exp = append(exp, append(length, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	fixedArrStrPack = fixedArrStrPack[4:]
	if !bytes.Equal(fixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, fixedArrStrPack)
	}

	// test byte array, fixed array uint256[2]
	bytesin := []byte(strin)
	arrin = [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedArrBytesPack, err := abi.Pack("fixedArrBytes", bytesin, arrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	offset = make([]byte, 32)
	offset[31] = 96
	length = make([]byte, 32)
	length[31] = byte(len(strin))
	strvalue = common.RightPadBytes([]byte(strin), 32)
	arrinvalue1 = common.LeftPadBytes(arrin[0].Bytes(), 32)
	arrinvalue2 = common.LeftPadBytes(arrin[1].Bytes(), 32)
	exp = append(offset, arrinvalue1...)
	exp = append(exp, arrinvalue2...)
	exp = append(exp, append(length, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	fixedArrBytesPack = fixedArrBytesPack[4:]
	if !bytes.Equal(fixedArrBytesPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, fixedArrBytesPack)
	}

	// test string, fixed array uint256[2], dynamic array uint256[]
	strin = "hello world"
	fixedarrin := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	dynarrin := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	mixedArrStrPack, err := abi.Pack("mixedArrStr", strin, fixedarrin, dynarrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset := make([]byte, 32)
	stroffset[31] = 128
	strlength := make([]byte, 32)
	strlength[31] = byte(len(strin))
	strvalue = common.RightPadBytes([]byte(strin), 32)
	fixedarrinvalue1 := common.LeftPadBytes(fixedarrin[0].Bytes(), 32)
	fixedarrinvalue2 := common.LeftPadBytes(fixedarrin[1].Bytes(), 32)
	dynarroffset := make([]byte, 32)
	dynarroffset[31] = byte(160 + ((len(strin)/32)+1)*32)
	dynarrlength := make([]byte, 32)
	dynarrlength[31] = byte(len(dynarrin))
	dynarrinvalue1 := common.LeftPadBytes(dynarrin[0].Bytes(), 32)
	dynarrinvalue2 := common.LeftPadBytes(dynarrin[1].Bytes(), 32)
	dynarrinvalue3 := common.LeftPadBytes(dynarrin[2].Bytes(), 32)
	exp = append(stroffset, fixedarrinvalue1...)
	exp = append(exp, fixedarrinvalue2...)
	exp = append(exp, dynarroffset...)
	exp = append(exp, append(strlength, strvalue...)...)
	dynarrarg := append(dynarrlength, dynarrinvalue1...)
	dynarrarg = append(dynarrarg, dynarrinvalue2...)
	dynarrarg = append(dynarrarg, dynarrinvalue3...)
	exp = append(exp, dynarrarg...)

	// ignore first 4 bytes of the output. This is the function identifier
	mixedArrStrPack = mixedArrStrPack[4:]
	if !bytes.Equal(mixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, mixedArrStrPack)
	}

	// test string, fixed array uint256[2], fixed array uint256[3]
	strin = "hello world"
	fixedarrin1 := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedarrin2 := [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	doubleFixedArrStrPack, err := abi.Pack("doubleFixedArrStr", strin, fixedarrin1, fixedarrin2)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset = make([]byte, 32)
	stroffset[31] = 192
	strlength = make([]byte, 32)
	strlength[31] = byte(len(strin))
	strvalue = common.RightPadBytes([]byte(strin), 32)
	fixedarrin1value1 := common.LeftPadBytes(fixedarrin1[0].Bytes(), 32)
	fixedarrin1value2 := common.LeftPadBytes(fixedarrin1[1].Bytes(), 32)
	fixedarrin2value1 := common.LeftPadBytes(fixedarrin2[0].Bytes(), 32)
	fixedarrin2value2 := common.LeftPadBytes(fixedarrin2[1].Bytes(), 32)
	fixedarrin2value3 := common.LeftPadBytes(fixedarrin2[2].Bytes(), 32)
	exp = append(stroffset, fixedarrin1value1...)
	exp = append(exp, fixedarrin1value2...)
	exp = append(exp, fixedarrin2value1...)
	exp = append(exp, fixedarrin2value2...)
	exp = append(exp, fixedarrin2value3...)
	exp = append(exp, append(strlength, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	doubleFixedArrStrPack = doubleFixedArrStrPack[4:]
	if !bytes.Equal(doubleFixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, doubleFixedArrStrPack)
	}

	// test string, fixed array uint256[2], dynamic array uint256[], fixed array uint256[3]
	strin = "hello world"
	fixedarrin1 = [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	dynarrin = []*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedarrin2 = [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	multipleMixedArrStrPack, err := abi.Pack("multipleMixedArrStr", strin, fixedarrin1, dynarrin, fixedarrin2)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset = make([]byte, 32)
	stroffset[31] = 224
	strlength = make([]byte, 32)
	strlength[31] = byte(len(strin))
	strvalue = common.RightPadBytes([]byte(strin), 32)
	fixedarrin1value1 = common.LeftPadBytes(fixedarrin1[0].Bytes(), 32)
	fixedarrin1value2 = common.LeftPadBytes(fixedarrin1[1].Bytes(), 32)
	dynarroffset = U256(big.NewInt(int64(256 + ((len(strin)/32)+1)*32)))
	dynarrlength = make([]byte, 32)
	dynarrlength[31] = byte(len(dynarrin))
	dynarrinvalue1 = common.LeftPadBytes(dynarrin[0].Bytes(), 32)
	dynarrinvalue2 = common.LeftPadBytes(dynarrin[1].Bytes(), 32)
	fixedarrin2value1 = common.LeftPadBytes(fixedarrin2[0].Bytes(), 32)
	fixedarrin2value2 = common.LeftPadBytes(fixedarrin2[1].Bytes(), 32)
	fixedarrin2value3 = common.LeftPadBytes(fixedarrin2[2].Bytes(), 32)
	exp = append(stroffset, fixedarrin1value1...)
	exp = append(exp, fixedarrin1value2...)
	exp = append(exp, dynarroffset...)
	exp = append(exp, fixedarrin2value1...)
	exp = append(exp, fixedarrin2value2...)
	exp = append(exp, fixedarrin2value3...)
	exp = append(exp, append(strlength, strvalue...)...)
	dynarrarg = append(dynarrlength, dynarrinvalue1...)
	dynarrarg = append(dynarrarg, dynarrinvalue2...)
	exp = append(exp, dynarrarg...)

	// ignore first 4 bytes of the output. This is the function identifier
	multipleMixedArrStrPack = multipleMixedArrStrPack[4:]
	if !bytes.Equal(multipleMixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, multipleMixedArrStrPack)
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
	{ "type" : "event", "name" : "anon", "anonymous" : true},
	{ "type" : "event", "name" : "args", "inputs" : [{ "indexed":false, "name":"arg0", "type":"uint256" }, { "indexed":true, "name":"arg1", "type":"address" }] },
	{ "type" : "event", "name" : "tuple", "inputs" : [{ "indexed":false, "name":"t", "type":"tuple", "components":[{"name":"a", "type":"uint256"}] }, { "indexed":true, "name":"arg1", "type":"address" }] }
	]`

	arg0, _ := NewType("uint256", nil)
	arg1, _ := NewType("address", nil)
	tuple, _ := NewType("tuple", []ArgumentMarshaling{{Name: "a", Type: "uint256"}})

	expectedEvents := map[string]struct {
		Anonymous bool
		Args      []Argument
	}{
		"balance": {false, nil},
		"anon":    {true, nil},
		"args": {false, []Argument{
			{Name: "arg0", Type: arg0, Indexed: false},
			{Name: "arg1", Type: arg1, Indexed: true},
		}},
		"tuple": {false, []Argument{
			{Name: "t", Type: tuple, Indexed: false},
			{Name: "arg1", Type: arg1, Indexed: true},
		}},
	}

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	if len(abi.Events) != len(expectedEvents) {
		t.Fatalf("invalid number of events after parsing, want %d, got %d", len(expectedEvents), len(abi.Events))
	}

	for name, exp := range expectedEvents {
		got, ok := abi.Events[name]
		if !ok {
			t.Errorf("could not found event %s", name)
			continue
		}
		if got.Anonymous != exp.Anonymous {
			t.Errorf("invalid anonymous indication for event %s, want %v, got %v", name, exp.Anonymous, got.Anonymous)
		}
		if len(got.Inputs) != len(exp.Args) {
			t.Errorf("invalid number of args, want %d, got %d", len(exp.Args), len(got.Inputs))
			continue
		}
		for i, arg := range exp.Args {
			if arg.Name != got.Inputs[i].Name {
				t.Errorf("events[%s].Input[%d] has an invalid name, want %s, got %s", name, i, arg.Name, got.Inputs[i].Name)
			}
			if arg.Indexed != got.Inputs[i].Indexed {
				t.Errorf("events[%s].Input[%d] has an invalid indexed indication, want %v, got %v", name, i, arg.Indexed, got.Inputs[i].Indexed)
			}
			if arg.Type.T != got.Inputs[i].Type.T {
				t.Errorf("events[%s].Input[%d] has an invalid type, want %x, got %x", name, i, arg.Type.T, got.Inputs[i].Type.T)
			}
		}
	}
}

// TestUnpackEvent is based on this contract:
//    contract T {
//      event received(address sender, uint amount, bytes memo);
//      event receivedAddr(address sender);
//      function receive(bytes memo) external payable {
//        received(msg.sender, msg.value, memo);
//        receivedAddr(msg.sender);
//      }
//    }
// When receive("X") is called with sender 0x00... and value 1, it produces this tx receipt:
//   receipt{status=1 cgas=23949 bloom=00000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000040200000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 logs=[log: b6818c8064f645cd82d99b59a1a267d6d61117ef [75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed] 000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158 9ae378b6d4409eada347a5dc0c180f186cb62dc68fcc0f043425eb917335aa28 0 95d429d309bb9d753954195fe2d69bd140b4ae731b9b5b605c34323de162cf00 0]}
func TestUnpackEvent(t *testing.T) {
	const abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"receive","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"receivedAddr","type":"event"}]`
	abi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}

	const hexdata = `000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158`
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}

	type ReceivedEvent struct {
		Sender common.Address
		Amount *big.Int
		Memo   []byte
	}
	var ev ReceivedEvent

	err = abi.Unpack(&ev, "received", data)
	if err != nil {
		t.Error(err)
	}

	type ReceivedAddrEvent struct {
		Sender common.Address
	}
	var receivedAddrEv ReceivedAddrEvent
	err = abi.Unpack(&receivedAddrEv, "receivedAddr", data)
	if err != nil {
		t.Error(err)
	}
}

func TestUnpackEventIntoMap(t *testing.T) {
	const abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"receive","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"receivedAddr","type":"event"}]`
	abi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}

	const hexdata = `000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158`
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}

	receivedMap := map[string]interface{}{}
	expectedReceivedMap := map[string]interface{}{
		"sender": common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
		"amount": big.NewInt(1),
		"memo":   []byte{88},
	}
	if err := abi.UnpackIntoMap(receivedMap, "received", data); err != nil {
		t.Error(err)
	}
	if len(receivedMap) != 3 {
		t.Error("unpacked `received` map expected to have length 3")
	}
	if receivedMap["sender"] != expectedReceivedMap["sender"] {
		t.Error("unpacked `received` map does not match expected map")
	}
	if receivedMap["amount"].(*big.Int).Cmp(expectedReceivedMap["amount"].(*big.Int)) != 0 {
		t.Error("unpacked `received` map does not match expected map")
	}
	if !bytes.Equal(receivedMap["memo"].([]byte), expectedReceivedMap["memo"].([]byte)) {
		t.Error("unpacked `received` map does not match expected map")
	}

	receivedAddrMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(receivedAddrMap, "receivedAddr", data); err != nil {
		t.Error(err)
	}
	if len(receivedAddrMap) != 1 {
		t.Error("unpacked `receivedAddr` map expected to have length 1")
	}
	if receivedAddrMap["sender"] != expectedReceivedMap["sender"] {
		t.Error("unpacked `receivedAddr` map does not match expected map")
	}
}

func TestUnpackMethodIntoMap(t *testing.T) {
	const abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"receive","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[],"name":"send","outputs":[{"name":"amount","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"get","outputs":[{"name":"hash","type":"bytes"}],"payable":true,"stateMutability":"payable","type":"function"}]`
	abi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	const hexdata = `00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000015800000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000158000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000001580000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000015800000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000158`
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 != 0 {
		t.Errorf("len(data) is %d, want a multiple of 32", len(data))
	}

	// Tests a method with no outputs
	receiveMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(receiveMap, "receive", data); err != nil {
		t.Error(err)
	}
	if len(receiveMap) > 0 {
		t.Error("unpacked `receive` map expected to have length 0")
	}

	// Tests a method with only outputs
	sendMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(sendMap, "send", data); err != nil {
		t.Error(err)
	}
	if len(sendMap) != 1 {
		t.Error("unpacked `send` map expected to have length 1")
	}
	if sendMap["amount"].(*big.Int).Cmp(big.NewInt(1)) != 0 {
		t.Error("unpacked `send` map expected `amount` value of 1")
	}

	// Tests a method with outputs and inputs
	getMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(getMap, "get", data); err != nil {
		t.Error(err)
	}
	if len(getMap) != 1 {
		t.Error("unpacked `get` map expected to have length 1")
	}
	expectedBytes := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 88, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 88, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 88, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 88, 0}
	if !bytes.Equal(getMap["hash"].([]byte), expectedBytes) {
		t.Errorf("unpacked `get` map expected `hash` value of %v", expectedBytes)
	}
}

func TestUnpackIntoMapNamingConflict(t *testing.T) {
	// Two methods have the same name
	var abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"get","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[],"name":"send","outputs":[{"name":"amount","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"get","outputs":[{"name":"hash","type":"bytes"}],"payable":true,"stateMutability":"payable","type":"function"}]`
	abi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	var hexdata = `00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158`
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}
	getMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(getMap, "get", data); err == nil {
		t.Error("naming conflict between two methods; error expected")
	}

	// Two events have the same name
	abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"receive","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"received","type":"event"}]`
	abi, err = JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	hexdata = `000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158`
	data, err = hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}
	receivedMap := map[string]interface{}{}
	if err = abi.UnpackIntoMap(receivedMap, "received", data); err != nil {
		t.Error("naming conflict between two events; no error expected")
	}

	// Method and event have the same name
	abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"received","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"receivedAddr","type":"event"}]`
	abi, err = JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}
	if err = abi.UnpackIntoMap(receivedMap, "received", data); err == nil {
		t.Error("naming conflict between an event and a method; error expected")
	}

	// Conflict is case sensitive
	abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"received","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"Received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"receivedAddr","type":"event"}]`
	abi, err = JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%32 == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}
	expectedReceivedMap := map[string]interface{}{
		"sender": common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
		"amount": big.NewInt(1),
		"memo":   []byte{88},
	}
	if err = abi.UnpackIntoMap(receivedMap, "Received", data); err != nil {
		t.Error(err)
	}
	if len(receivedMap) != 3 {
		t.Error("unpacked `received` map expected to have length 3")
	}
	if receivedMap["sender"] != expectedReceivedMap["sender"] {
		t.Error("unpacked `received` map does not match expected map")
	}
	if receivedMap["amount"].(*big.Int).Cmp(expectedReceivedMap["amount"].(*big.Int)) != 0 {
		t.Error("unpacked `received` map does not match expected map")
	}
	if !bytes.Equal(receivedMap["memo"].([]byte), expectedReceivedMap["memo"].([]byte)) {
		t.Error("unpacked `received` map does not match expected map")
	}
}

func TestABI_MethodById(t *testing.T) {
	const abiJSON = `[
		{"type":"function","name":"receive","constant":false,"inputs":[{"name":"memo","type":"bytes"}],"outputs":[],"payable":true,"stateMutability":"payable"},
		{"type":"event","name":"received","anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}]},
		{"type":"function","name":"fixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr","type":"uint256[2]"}]},
		{"type":"function","name":"fixedArrBytes","constant":true,"inputs":[{"name":"str","type":"bytes"},{"name":"fixedArr","type":"uint256[2]"}]},
		{"type":"function","name":"mixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr","type":"uint256[2]"},{"name":"dynArr","type":"uint256[]"}]},
		{"type":"function","name":"doubleFixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr1","type":"uint256[2]"},{"name":"fixedArr2","type":"uint256[3]"}]},
		{"type":"function","name":"multipleMixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr1","type":"uint256[2]"},{"name":"dynArr","type":"uint256[]"},{"name":"fixedArr2","type":"uint256[3]"}]},
		{"type":"function","name":"balance","constant":true},
		{"type":"function","name":"send","constant":false,"inputs":[{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"test","constant":false,"inputs":[{"name":"number","type":"uint32"}]},
		{"type":"function","name":"string","constant":false,"inputs":[{"name":"inputs","type":"string"}]},
		{"type":"function","name":"bool","constant":false,"inputs":[{"name":"inputs","type":"bool"}]},
		{"type":"function","name":"address","constant":false,"inputs":[{"name":"inputs","type":"address"}]},
		{"type":"function","name":"uint64[2]","constant":false,"inputs":[{"name":"inputs","type":"uint64[2]"}]},
		{"type":"function","name":"uint64[]","constant":false,"inputs":[{"name":"inputs","type":"uint64[]"}]},
		{"type":"function","name":"foo","constant":false,"inputs":[{"name":"inputs","type":"uint32"}]},
		{"type":"function","name":"bar","constant":false,"inputs":[{"name":"inputs","type":"uint32"},{"name":"string","type":"uint16"}]},
		{"type":"function","name":"_slice","constant":false,"inputs":[{"name":"inputs","type":"uint32[2]"}]},
		{"type":"function","name":"__slice256","constant":false,"inputs":[{"name":"inputs","type":"uint256[2]"}]},
		{"type":"function","name":"sliceAddress","constant":false,"inputs":[{"name":"inputs","type":"address[]"}]},
		{"type":"function","name":"sliceMultiAddress","constant":false,"inputs":[{"name":"a","type":"address[]"},{"name":"b","type":"address[]"}]}
	]
`
	abi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	for name, m := range abi.Methods {
		a := fmt.Sprintf("%v", m)
		m2, err := abi.MethodById(m.Id())
		if err != nil {
			t.Fatalf("Failed to look up ABI method: %v", err)
		}
		b := fmt.Sprintf("%v", m2)
		if a != b {
			t.Errorf("Method %v (id %v) not 'findable' by id in ABI", name, common.ToHex(m.Id()))
		}
	}
	// Also test empty
	if _, err := abi.MethodById([]byte{0x00}); err == nil {
		t.Errorf("Expected error, too short to decode data")
	}
	if _, err := abi.MethodById([]byte{}); err == nil {
		t.Errorf("Expected error, too short to decode data")
	}
	if _, err := abi.MethodById(nil); err == nil {
		t.Errorf("Expected error, nil is short to decode data")
	}
}

func TestABI_EventById(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		event string
	}{
		{
			name: "",
			json: `[
			{"type":"event","name":"received","anonymous":false,"inputs":[
				{"indexed":false,"name":"sender","type":"address"},
				{"indexed":false,"name":"amount","type":"uint256"},
				{"indexed":false,"name":"memo","type":"bytes"}
				]
			}]`,
			event: "received(address,uint256,bytes)",
		}, {
			name: "",
			json: `[
				{ "constant": true, "inputs": [], "name": "name", "outputs": [ { "name": "", "type": "string" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "constant": false, "inputs": [ { "name": "_spender", "type": "address" }, { "name": "_value", "type": "uint256" } ], "name": "approve", "outputs": [ { "name": "", "type": "bool" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
				{ "constant": true, "inputs": [], "name": "totalSupply", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "constant": false, "inputs": [ { "name": "_from", "type": "address" }, { "name": "_to", "type": "address" }, { "name": "_value", "type": "uint256" } ], "name": "transferFrom", "outputs": [ { "name": "", "type": "bool" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
				{ "constant": true, "inputs": [], "name": "decimals", "outputs": [ { "name": "", "type": "uint8" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "constant": true, "inputs": [ { "name": "_owner", "type": "address" } ], "name": "balanceOf", "outputs": [ { "name": "balance", "type": "uint256" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "constant": true, "inputs": [], "name": "symbol", "outputs": [ { "name": "", "type": "string" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "constant": false, "inputs": [ { "name": "_to", "type": "address" }, { "name": "_value", "type": "uint256" } ], "name": "transfer", "outputs": [ { "name": "", "type": "bool" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
				{ "constant": true, "inputs": [ { "name": "_owner", "type": "address" }, { "name": "_spender", "type": "address" } ], "name": "allowance", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "view", "type": "function" },
				{ "payable": true, "stateMutability": "payable", "type": "fallback" },
				{ "anonymous": false, "inputs": [ { "indexed": true, "name": "owner", "type": "address" }, { "indexed": true, "name": "spender", "type": "address" }, { "indexed": false, "name": "value", "type": "uint256" } ], "name": "Approval", "type": "event" },
				{ "anonymous": false, "inputs": [ { "indexed": true, "name": "from", "type": "address" }, { "indexed": true, "name": "to", "type": "address" }, { "indexed": false, "name": "value", "type": "uint256" } ], "name": "Transfer", "type": "event" }
			]`,
			event: "Transfer(address,address,uint256)",
		},
	}

	for testnum, test := range tests {
		abi, err := JSON(strings.NewReader(test.json))
		if err != nil {
			t.Error(err)
		}

		topic := test.event
		topicID := crypto.Keccak256Hash([]byte(topic))

		event, err := abi.EventByID(topicID)
		if err != nil {
			t.Fatalf("Failed to look up ABI method: %v, test #%d", err, testnum)
		}
		if event == nil {
			t.Errorf("We should find a event for topic %s, test #%d", topicID.Hex(), testnum)
		}

		if event.Id() != topicID {
			t.Errorf("Event id %s does not match topic %s, test #%d", event.Id().Hex(), topicID.Hex(), testnum)
		}

		unknowntopicID := crypto.Keccak256Hash([]byte("unknownEvent"))
		unknownEvent, err := abi.EventByID(unknowntopicID)
		if err == nil {
			t.Errorf("EventByID should return an error if a topic is not found, test #%d", testnum)
		}
		if unknownEvent != nil {
			t.Errorf("We should not find any event for topic %s, test #%d", unknowntopicID.Hex(), testnum)
		}
	}
}

func TestDuplicateMethodNames(t *testing.T) {
	abiJSON := `[{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}],"name":"transfer","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"customFallback","type":"string"}],"name":"transfer","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	contractAbi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := contractAbi.Methods["transfer"]; !ok {
		t.Fatalf("Could not find original method")
	}
	if _, ok := contractAbi.Methods["transfer0"]; !ok {
		t.Fatalf("Could not find duplicate method")
	}
	if _, ok := contractAbi.Methods["transfer1"]; !ok {
		t.Fatalf("Could not find duplicate method")
	}
	if _, ok := contractAbi.Methods["transfer2"]; ok {
		t.Fatalf("Should not have found extra method")
	}
}

// TestDoubleDuplicateMethodNames checks that if transfer0 already exists, there won't be a name
// conflict and that the second transfer method will be renamed transfer1.
func TestDoubleDuplicateMethodNames(t *testing.T) {
	abiJSON := `[{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}],"name":"transfer0","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"customFallback","type":"string"}],"name":"transfer","outputs":[{"name":"ok","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	contractAbi, err := JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := contractAbi.Methods["transfer"]; !ok {
		t.Fatalf("Could not find original method")
	}
	if _, ok := contractAbi.Methods["transfer0"]; !ok {
		t.Fatalf("Could not find duplicate method")
	}
	if _, ok := contractAbi.Methods["transfer1"]; !ok {
		t.Fatalf("Could not find duplicate method")
	}
	if _, ok := contractAbi.Methods["transfer2"]; ok {
		t.Fatalf("Should not have found extra method")
	}
}
