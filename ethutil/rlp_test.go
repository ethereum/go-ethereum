package ethutil

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestRlpValueEncoding(t *testing.T) {
	val := EmptyValue()
	val.AppendList().Append(1).Append(2).Append(3)
	val.Append("4").AppendList().Append(5)

	res := val.Encode()
	exp := Encode([]interface{}{[]interface{}{1, 2, 3}, "4", []interface{}{5}})
	if bytes.Compare(res, exp) != 0 {
		t.Errorf("expected %q, got %q", res, exp)
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

func TestValue(t *testing.T) {
	value := NewValueFromBytes([]byte("\xcd\x83dog\x83god\x83cat\x01"))
	if value.Get(0).Str() != "dog" {
		t.Errorf("expected '%v', got '%v'", value.Get(0).Str(), "dog")
	}

	if value.Get(3).Uint() != 1 {
		t.Errorf("expected '%v', got '%v'", value.Get(3).Uint(), 1)
	}
}

func TestEncodeDecodeMaran(t *testing.T) {
	b := NewValue([]interface{}{"dog", 15, []interface{}{"cat", "cat", []interface{}{}}, 1024, "tachikoma"})
	a := b.Encode()
	fmt.Println("voor maran", a)
	f, i := Decode(a, 0)
	fmt.Println("voor maran 2", f)
	fmt.Println(i)
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
	b := NewValue([]interface{}{[]byte{1, 2, 3, 4, 5}, byte(6)})
	val := NewValueFromBytes(b.Encode())
	if !b.Cmp(val) {
		t.Errorf("Expected %v, got %v", val, b)
	}
}

func TestEncodeZero(t *testing.T) {
	b := NewValue(0).Encode()
	fmt.Println(b)
}

func BenchmarkEncodeDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bytes := Encode([]interface{}{"dog", "god", "cat"})
		Decode(bytes, 0)
	}
}
