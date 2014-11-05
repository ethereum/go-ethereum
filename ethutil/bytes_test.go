package ethutil

import (
	"bytes"
	"testing"
)

func TestByteString(t *testing.T) {
	var data Bytes
	data = []byte{102, 111, 111}
	exp := "foo"
	res := data.String()

	if res != exp {
		t.Errorf("Expected %s got %s", exp, res)
	}
}


func TestReadVarInt(t *testing.T) {
	data8 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	data4 := []byte{1, 2, 3, 4}
	data2 := []byte{1, 2}
	data1 := []byte{1}

	exp8 := uint64(72623859790382856)
	exp4 := uint64(16909060)
	exp2 := uint64(258)
	exp1 := uint64(1)

	res8 := ReadVarInt(data8)
	res4 := ReadVarInt(data4)
	res2 := ReadVarInt(data2)
	res1 := ReadVarInt(data1)

	if res8 != exp8 {
		t.Errorf("Expected %d | Got %d", exp8, res8)
	}

	if res4 != exp4 {
		t.Errorf("Expected %d | Got %d", exp4, res4)
	}

	if res2 != exp2 {
		t.Errorf("Expected %d | Got %d", exp2, res2)
	}

	if res1 != exp1 {
		t.Errorf("Expected %d | Got %d", exp1, res1)
	}
}

func TestBinaryLength(t *testing.T) {
	data1 := 0
	data2 := 920987656789

	exp1 := 0
	exp2 := 5

	res1 := BinaryLength(data1)
	res2 := BinaryLength(data2)

	if res1 != exp1 {
		t.Errorf("Expected %d got %d", exp1, res1)
	}

	if res2 != exp2 {
		t.Errorf("Expected %d got %d", exp2, res2)
	}
}

func TestCopyBytes(t *testing.T) {
	data1 := []byte{1, 2, 3, 4}
	exp1 := []byte{1, 2, 3, 4}
	res1 := CopyBytes(data1)
	if bytes.Compare(res1, exp1) != 0 {
		t.Errorf("Expected % x got % x", exp1, res1)
	}
}

func TestIsHex(t *testing.T) {
	data1 := "a9e67e"
	exp1 := false
	res1 := IsHex(data1)
	if exp1 != res1 {
		t.Errorf("Expected % x Got % x", exp1, res1)
	}

	data2 := "0xa9e67e00"
	exp2 := true
	res2 := IsHex(data2)
	if exp2 != res2 {
		t.Errorf("Expected % x Got % x", exp2, res2)
	}
}

func TestParseDataString(t *testing.T) {
	data := ParseData("hello", "world", "0x0106")
	exp := "68656c6c6f000000000000000000000000000000000000000000000000000000776f726c640000000000000000000000000000000000000000000000000000000106000000000000000000000000000000000000000000000000000000000000"
	if bytes.Compare(data, Hex2Bytes(exp)) != 0 {
		t.Error("Error parsing data")
	}
}

func TestParseDataBytes(t *testing.T) {
	data := []byte{232, 212, 165, 16, 0}
	exp := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 232, 212, 165, 16, 0}

	res := ParseData(data)
	if bytes.Compare(res, exp) != 0 {
		t.Errorf("Expected %x got %x", exp, res)
	}
}

func TestLeftPadBytes(t *testing.T) {
	val := []byte{1, 2, 3, 4}
	exp := []byte{0, 0, 0, 0, 1, 2, 3, 4}

	resstd := LeftPadBytes(val, 8)
	if bytes.Compare(resstd, exp) != 0 {
		t.Errorf("Expected % x Got % x", exp, resstd)
	}

	resshrt := LeftPadBytes(val, 2)
	if bytes.Compare(resshrt, val) != 0 {
		t.Errorf("Expected % x Got % x", exp, resshrt)
	}
}

func TestFormatData(t *testing.T) {
	data1 := ""
	data2 := "0xa9e67e00"
	data3 := "a9e67e"
	data4 := "\"a9e67e00\""

	exp1 := []byte{}
	exp2 := []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0xa9, 0xe6, 0x7e, 00}
	exp3 := []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}
	exp4 := []byte{0x61, 0x39, 0x65, 0x36, 0x37, 0x65, 0x30, 0x30, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}

	res1 := FormatData(data1)
	res2 := FormatData(data2)
	res3 := FormatData(data3)
	res4 := FormatData(data4)

	if bytes.Compare(res1, exp1) != 0 {
		t.Errorf("Expected % x Got % x", exp1, res1)
	}

	if bytes.Compare(res2, exp2) != 0 {
		t.Errorf("Expected % x Got % x", exp2, res2)
	}

	if bytes.Compare(res3, exp3) != 0 {
		t.Errorf("Expected % x Got % x", exp3, res3)
	}

	if bytes.Compare(res4, exp4) != 0 {
		t.Errorf("Expected % x Got % x", exp4, res4)
	}
}

func TestRightPadBytes(t *testing.T) {
	val := []byte{1, 2, 3, 4}
	exp := []byte{1, 2, 3, 4, 0, 0, 0, 0}

	resstd := RightPadBytes(val, 8)
	if bytes.Compare(resstd, exp) != 0 {
		t.Errorf("Expected % x Got % x", exp, resstd)
	}

	resshrt := RightPadBytes(val, 2)
	if bytes.Compare(resshrt, val) != 0 {
		t.Errorf("Expected % x Got % x", exp, resshrt)
	}
}

func TestLeftPadString(t *testing.T) {
	val := "test"

	resstd := LeftPadString(val, 8)

	if resstd != "\x30\x30\x30\x30"+val {
		t.Errorf("Expected % x Got % x", val, resstd)
	}

	resshrt := LeftPadString(val, 2)

	if resshrt != val {
		t.Errorf("Expected % x Got % x", val, resshrt)
	}
}

func TestRightPadString(t *testing.T) {
	val := "test"

	resstd := RightPadString(val, 8)
	if resstd != val+"\x30\x30\x30\x30" {
		t.Errorf("Expected % x Got % x", val, resstd)
	}

	resshrt := RightPadString(val, 2)
	if resshrt != val {
		t.Errorf("Expected % x Got % x", val, resshrt)
	}
}
