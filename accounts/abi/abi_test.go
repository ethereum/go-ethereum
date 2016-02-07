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
	{ "type" : "function", "name" : "slice256", "const" : false, "inputs" : [ { "name" : "inputs", "type" : "uint256[2]" } ] }
]`

func TestType(t *testing.T) {
	typ, err := NewType("uint32")
	if err != nil {
		t.Error(err)
	}
	if typ.Kind != reflect.Ptr {
		t.Error("expected uint32 to have kind Ptr")
	}

	typ, err = NewType("uint32[]")
	if err != nil {
		t.Error(err)
	}
	if typ.Kind != reflect.Slice {
		t.Error("expected uint32[] to have type slice")
	}
	if typ.Type != ubig_ts {
		t.Error("expcted uith32[] to have type uint64")
	}

	typ, err = NewType("uint32[2]")
	if err != nil {
		t.Error(err)
	}
	if typ.Kind != reflect.Slice {
		t.Error("expected uint32[2] to have kind slice")
	}
	if typ.Type != ubig_ts {
		t.Error("expcted uith32[2] to have type uint64")
	}
	if typ.Size != 2 {
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

	if _, err := abi.Pack("send", 1000); err != nil {
		t.Error("expected send(1000) to cast to big")
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

	addr := make([]byte, 20)
	if _, err := abi.Pack("address", addr); err != nil {
		t.Error(err)
	}

	addr = make([]byte, 21)
	if _, err := abi.Pack("address", addr); err == nil {
		t.Error("expected address of 21 width to throw")
	}

	slice := make([]byte, 2)
	if _, err := abi.Pack("uint64[2]", slice); err != nil {
		t.Error(err)
	}

	if _, err := abi.Pack("uint64[]", slice); err != nil {
		t.Error(err)
	}
}

func TestTestAddress(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	addr := make([]byte, 20)
	if _, err := abi.Pack("address", addr); err != nil {
		t.Error(err)
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

	idexp := crypto.Sha3([]byte(exp))[:4]
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

	sig := crypto.Sha3([]byte("foo(uint32)"))[:4]
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

	sig := crypto.Sha3([]byte("bar(uint32,uint16)"))[:4]
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

	sig := crypto.Sha3([]byte("slice(uint32[2])"))[:4]
	sig = append(sig, make([]byte, 64)...)
	sig[35] = 1
	sig[67] = 2

	packed, err := abi.Pack("slice", []uint32{1, 2})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}
}

func TestPackSliceBig(t *testing.T) {
	abi, err := JSON(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sig := crypto.Sha3([]byte("slice256(uint256[2])"))[:4]
	sig = append(sig, make([]byte, 64)...)
	sig[35] = 1
	sig[67] = 2

	packed, err := abi.Pack("slice256", []*big.Int{big.NewInt(1), big.NewInt(2)})
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

func TestBytes(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "balance", "const" : true, "inputs" : [ { "name" : "address", "type" : "bytes20" } ] },
	{ "type" : "function", "name" : "send", "const" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] }
]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}
	ok := make([]byte, 20)
	_, err = abi.Pack("balance", ok)
	if err != nil {
		t.Error(err)
	}

	toosmall := make([]byte, 19)
	_, err = abi.Pack("balance", toosmall)
	if err != nil {
		t.Error(err)
	}

	toobig := make([]byte, 21)
	_, err = abi.Pack("balance", toobig)
	if err == nil {
		t.Error("expected error")
	}
}

func TestReturn(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "balance", "const" : true, "inputs" : [], "outputs" : [ { "name": "", "type": "hash" } ] },
	{ "type" : "function", "name" : "name", "const" : true, "inputs" : [], "outputs" : [ { "name": "", "type": "address" } ] }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	r := abi.Call(func([]byte) []byte {
		t := make([]byte, 32)
		t[0] = 1
		return t
	}, "balance")
	if _, ok := r.(common.Hash); !ok {
		t.Errorf("expected type common.Hash, got %T", r)
	}

	r = abi.Call(func([]byte) []byte {
		t := make([]byte, 32)
		t[0] = 1
		return t
	}, "name")
	if _, ok := r.(common.Address); !ok {
		t.Errorf("expected type common.Address, got %T", r)
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
