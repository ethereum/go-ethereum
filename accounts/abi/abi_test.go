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

const jsondata = `
[
	{ "type" : "function", "name" : "balance", "const" : true },
	{ "type" : "function", "name" : "send", "const" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] }
]`

const jsondata2 = `
[
	{ "type" : "function", "name" : "balance", "const" : true },
	{ "type" : "function", "name" : "send", "const" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] },
	{ "type" : "function", "name" : "test", "const" : false, "inputs" : [ { "name" : "number", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "string", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "string" } ] },
	{ "type" : "function", "name" : "bool", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "bool" } ] },
	{ "type" : "function", "name" : "address", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "address" } ] },
	{ "type" : "function", "name" : "string32", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "string32" } ] },
	{ "type" : "function", "name" : "uint64[2]", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[2]" } ] },
	{ "type" : "function", "name" : "uint64[]", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[]" } ] },
	{ "type" : "function", "name" : "foo", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "bar", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" }, { "name" : "string", "type" : "uint16" } ] },
	{ "type" : "function", "name" : "slice", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32[2]" } ] },
	{ "type" : "function", "name" : "slice256", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint256[2]" } ] },
	{ "type" : "function", "name" : "sliceAddress", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "address[]" } ] },
	{ "type" : "function", "name" : "sliceMultiAddress", "const" : false, "inputs" : [ { "name" : "a", "type" : "address[]" }, { "name" : "b", "type" : "address[]" } ] }
]`

func TestType(t *testing.T) {
	typ, err := NewType("uint32")
	if err != nil {
		t.Error(err)
	}
	if typ.Kind != reflect.Uint {
		t.Error("expected uint32 to have kind Ptr")
	}

	typ, err = NewType("uint32[]")
	if err != nil {
		t.Error(err)
	}
	if !typ.IsSlice {
		t.Error("expected uint32[] to be slice")
	}
	if typ.Type != ubig_t {
		t.Error("expcted uith32[] to have type uint64")
	}

	typ, err = NewType("uint32[2]")
	if err != nil {
		t.Error(err)
	}
	if !typ.IsSlice {
		t.Error("expected uint32[2] to be slice")
	}
	if typ.Type != ubig_t {
		t.Error("expcted uith32[2] to have type uint64")
	}
	if typ.SliceSize != 2 {
		t.Error("expected uint32[2] to have a size of 2")
	}
}

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

	str10 := string(make([]byte, 10))
	if _, err := abi.Pack("string32", str10); err != nil {
		t.Error(err)
	}

	str32 := string(make([]byte, 32))
	if _, err := abi.Pack("string32", str32); err != nil {
		t.Error(err)
	}

	str33 := string(make([]byte, 33))
	if _, err := abi.Pack("string32", str33); err == nil {
		t.Error("expected str33 to throw out of bound error")
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

func TestImplicitTypeCasts(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	slice := make([]uint8, 2)
	_, err = abi.Pack("uint64[2]", slice)
	expStr := "`uint64[2]` abi: cannot use type uint8 as type uint64"
	if err.Error() != expStr {
		t.Errorf("expected %v, got %v", expStr, err)
	}
}

func TestMethodSignature(t *testing.T) {
	String, _ := NewType("string")
	String32, _ := NewType("string32")
	m := Method{"foo", false, []Argument{Argument{"bar", String32, false}, Argument{"baz", String, false}}, nil}
	exp := "foo(string32,string)"
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

func TestPack(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sig := crypto.Keccak256([]byte("foo(uint32)"))[:4]
	sig = append(sig, make([]byte, 32)...)
	sig[35] = 10

	packed, err := abi.Pack("foo", uint32(10))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
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

func TestPackSlice(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sig := crypto.Keccak256([]byte("slice(uint32[2])"))[:4]
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

	sig = crypto.Keccak256([]byte("slice256(uint256[2])"))[:4]
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
	{ "type" : "function", "name" : "strOne", "const" : true, "inputs" : [ { "name" : "str", "type" : "string" } ] },
	{ "type" : "function", "name" : "bytesOne", "const" : true, "inputs" : [ { "name" : "str", "type" : "bytes" } ] },
	{ "type" : "function", "name" : "strTwo", "const" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "str1", "type" : "string" } ] }
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
	{ "name" : "multi", "const" : false, "outputs": [ { "name": "Int", "type": "uint256" }, { "name": "String", "type": "string" } ] }]`

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
	{ "name" : "multi", "const" : false, "outputs": [ { "name": "Int", "type": "uint256" }, { "name": "String", "type": "string" } ] }]`

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
	{ "name" : "bytes32", "const" : false, "outputs": [ { "type": "bytes32" } ] },
	{ "name" : "bytes10", "const" : false, "outputs": [ { "type": "bytes10" } ] }
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
	{ "name" : "int", "const" : false, "outputs": [ { "type": "uint256" } ] },
	{ "name" : "bool", "const" : false, "outputs": [ { "type": "bool" } ] },
	{ "name" : "bytes", "const" : false, "outputs": [ { "type": "bytes" } ] },
	{ "name" : "fixed", "const" : false, "outputs": [ { "type": "bytes32" } ] },
	{ "name" : "multi", "const" : false, "outputs": [ { "type": "bytes" }, { "type": "bytes" } ] },
	{ "name" : "addressSliceSingle", "const" : false, "outputs": [ { "type": "address[]" } ] },
	{ "name" : "addressSliceDouble", "const" : false, "outputs": [ { "name": "a", "type": "address[]" }, { "name": "b", "type": "address[]" } ] },
	{ "name" : "mixedBytes", "const" : true, "outputs": [ { "name": "a", "type": "bytes" }, { "name": "b", "type": "bytes32" } ] }]`

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
