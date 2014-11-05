package ethutil

import (
	"bytes"
	"testing"
)

func TestParseData(t *testing.T) {
	data := ParseData("hello", "world", "0x0106")
	exp := "68656c6c6f000000000000000000000000000000000000000000000000000000776f726c640000000000000000000000000000000000000000000000000000000106000000000000000000000000000000000000000000000000000000000000"
	if bytes.Compare(data, Hex2Bytes(exp)) != 0 {
		t.Error("Error parsing data")
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
