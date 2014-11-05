package ethutil

import (
	"bytes"
	"testing"
)

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
