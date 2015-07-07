// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestNonInterfaceSlice(t *testing.T) {
	vala := []string{"value1", "value2", "value3"}
	valb := []interface{}{"value1", "value2", "value3"}
	resa := Encode(vala)
	resb := Encode(valb)
	if !bytes.Equal(resa, resb) {
		t.Errorf("expected []string & []interface{} to be equal")
	}
}

func TestRlpValueEncoding(t *testing.T) {
	val := EmptyValue()
	val.AppendList().Append(byte(1)).Append(byte(2)).Append(byte(3))
	val.Append("4").AppendList().Append(byte(5))

	res, err := rlp.EncodeToBytes(val)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	exp := Encode([]interface{}{[]interface{}{1, 2, 3}, "4", []interface{}{5}})
	if bytes.Compare(res, exp) != 0 {
		t.Errorf("expected %x, got %x", exp, res)
	}
}

func TestValueSlice(t *testing.T) {
	val := []interface{}{
		"value1",
		"valeu2",
		"value3",
	}

	value := NewValue(val)
	splitVal := value.SliceFrom(1)

	if splitVal.Len() != 2 {
		t.Error("SliceFrom: Expected len", 2, "got", splitVal.Len())
	}

	splitVal = value.SliceTo(2)
	if splitVal.Len() != 2 {
		t.Error("SliceTo: Expected len", 2, "got", splitVal.Len())
	}

	splitVal = value.SliceFromTo(1, 3)
	if splitVal.Len() != 2 {
		t.Error("SliceFromTo: Expected len", 2, "got", splitVal.Len())
	}
}

func TestLargeData(t *testing.T) {
	data := make([]byte, 100000)
	enc := Encode(data)
	value := NewValueFromBytes(enc)
	if value.Len() != len(data) {
		t.Error("Expected data to be", len(data), "got", value.Len())
	}
}

func TestValue(t *testing.T) {
	value := NewValueFromBytes([]byte("\xcd\x83dog\x83god\x83cat\x01"))
	if value.Get(0).Str() != "dog" {
		t.Errorf("expected '%v', got '%v'", value.Get(0).Str(), "dog")
	}

	if value.Get(3).Uint() != 1 {
		t.Errorf("expected '%v', got '%v'", value.Get(3).Uint(), 1)
	}
}

func TestEncode(t *testing.T) {
	strRes := "\x83dog"
	bytes := Encode("dog")

	str := string(bytes)
	if str != strRes {
		t.Errorf("Expected %q, got %q", strRes, str)
	}

	sliceRes := "\xcc\x83dog\x83god\x83cat"
	strs := []interface{}{"dog", "god", "cat"}
	bytes = Encode(strs)
	slice := string(bytes)
	if slice != sliceRes {
		t.Error("Expected %q, got %q", sliceRes, slice)
	}

	intRes := "\x82\x04\x00"
	bytes = Encode(1024)
	if string(bytes) != intRes {
		t.Errorf("Expected %q, got %q", intRes, bytes)
	}
}

func TestDecode(t *testing.T) {
	single := []byte("\x01")
	b, _ := Decode(single, 0)

	if b.(uint8) != 1 {
		t.Errorf("Expected 1, got %q", b)
	}

	str := []byte("\x83dog")
	b, _ = Decode(str, 0)
	if bytes.Compare(b.([]byte), []byte("dog")) != 0 {
		t.Errorf("Expected dog, got %q", b)
	}

	slice := []byte("\xcc\x83dog\x83god\x83cat")
	res := []interface{}{"dog", "god", "cat"}
	b, _ = Decode(slice, 0)
	if reflect.DeepEqual(b, res) {
		t.Errorf("Expected %q, got %q", res, b)
	}
}

func TestEncodeDecodeBigInt(t *testing.T) {
	bigInt := big.NewInt(1391787038)
	encoded := Encode(bigInt)

	value := NewValueFromBytes(encoded)
	if value.BigInt().Cmp(bigInt) != 0 {
		t.Errorf("Expected %v, got %v", bigInt, value.BigInt())
	}
}

func TestEncodeDecodeBytes(t *testing.T) {
	bv := NewValue([]interface{}{[]byte{1, 2, 3, 4, 5}, []byte{6}})
	b, _ := rlp.EncodeToBytes(bv)
	val := NewValueFromBytes(b)
	if !bv.Cmp(val) {
		t.Errorf("Expected %#v, got %#v", bv, val)
	}
}

func TestEncodeZero(t *testing.T) {
	b, _ := rlp.EncodeToBytes(NewValue(0))
	exp := []byte{0xc0}
	if bytes.Compare(b, exp) == 0 {
		t.Error("Expected", exp, "got", b)
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bytes := Encode([]interface{}{"dog", "god", "cat"})
		Decode(bytes, 0)
	}
}
