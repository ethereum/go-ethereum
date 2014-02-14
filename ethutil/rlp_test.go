package ethutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestRlpValueEncoding(t *testing.T) {
	val := EmptyRlpValue()
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

func TestEncode(t *testing.T) {
	strRes := "\x83dog"
	bytes := Encode("dog")

	str := string(bytes)
	if str != strRes {
		t.Error(fmt.Sprintf("Expected %q, got %q", strRes, str))
	}

	sliceRes := "\xcc\x83dog\x83god\x83cat"
	strs := []interface{}{"dog", "god", "cat"}
	bytes = Encode(strs)
	slice := string(bytes)
	if slice != sliceRes {
		t.Error(fmt.Sprintf("Expected %q, got %q", sliceRes, slice))
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
	fmt.Println(value.BigInt(), bigInt)
	if value.BigInt().Cmp(bigInt) != 0 {
		t.Errorf("Expected %v, got %v", bigInt, value.BigInt())
	}

	dec, _ := hex.DecodeString("52f4fc1e")
	fmt.Println(NewValueFromBytes(dec).BigInt())
}

func TestEncodeDecodeBytes(t *testing.T) {
	b := NewValue([]interface{}{[]byte{1, 2, 3, 4, 5}, byte(6)})
	val := NewValueFromBytes(b.Encode())
	if !b.Cmp(val) {
		t.Errorf("Expected %v, got %v", val, b)
	}
}

/*
var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = Sha3Bin(Encode([]interface{}{}))

var GenisisHeader = []interface{}{
	// Previous hash (none)
	//"",
	ZeroHash256,
	// Sha of uncles
	Sha3Bin(Encode([]interface{}{})),
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	// Sha of transactions
	//EmptyShaList,
	Sha3Bin(Encode([]interface{}{})),
	// Difficulty
	BigPow(2, 22),
	// Time
	//big.NewInt(0),
	int64(0),
	// extra
	"",
	// Nonce
	big.NewInt(42),
}

func TestEnc(t *testing.T) {
	//enc := Encode(GenisisHeader)
	//fmt.Printf("%x (%d)\n", enc, len(enc))
	h, _ := hex.DecodeString("f8a0a00000000000000000000000000000000000000000000000000000000000000000a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347940000000000000000000000000000000000000000a06d076baa9c4074fb2df222dd16a96b0155a1e6686b3e5748b4e9ca0a208a425ca01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d493478340000080802a")
	fmt.Printf("%x\n", Sha3Bin(h))
}
*/

func BenchmarkEncodeDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bytes := Encode([]interface{}{"dog", "god", "cat"})
		Decode(bytes, 0)
	}
}
