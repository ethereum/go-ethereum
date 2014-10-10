package ethtrie

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCompactEncode(t *testing.T) {
	test1 := []byte{1, 2, 3, 4, 5}
	if res := CompactEncode(test1); res != "\x11\x23\x45" {
		t.Error(fmt.Sprintf("even compact encode failed. Got: %q", res))
	}

	test2 := []byte{0, 1, 2, 3, 4, 5}
	if res := CompactEncode(test2); res != "\x00\x01\x23\x45" {
		t.Error(fmt.Sprintf("odd compact encode failed. Got: %q", res))
	}

	test3 := []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	if res := CompactEncode(test3); res != "\x20\x0f\x1c\xb8" {
		t.Error(fmt.Sprintf("odd terminated compact encode failed. Got: %q", res))
	}

	test4 := []byte{15, 1, 12, 11, 8 /*term*/, 16}
	if res := CompactEncode(test4); res != "\x3f\x1c\xb8" {
		t.Error(fmt.Sprintf("even terminated compact encode failed. Got: %q", res))
	}
}

func TestCompactHexDecode(t *testing.T) {
	exp := []byte{7, 6, 6, 5, 7, 2, 6, 2, 16}
	res := CompactHexDecode("verb")

	if !bytes.Equal(res, exp) {
		t.Error("Error compact hex decode. Expected", exp, "got", res)
	}
}

func TestCompactDecode(t *testing.T) {
	exp := []byte{1, 2, 3, 4, 5}
	res := CompactDecode("\x11\x23\x45")

	if !bytes.Equal(res, exp) {
		t.Error("odd compact decode. Expected", exp, "got", res)
	}

	exp = []byte{0, 1, 2, 3, 4, 5}
	res = CompactDecode("\x00\x01\x23\x45")

	if !bytes.Equal(res, exp) {
		t.Error("even compact decode. Expected", exp, "got", res)
	}

	exp = []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	res = CompactDecode("\x20\x0f\x1c\xb8")

	if !bytes.Equal(res, exp) {
		t.Error("even terminated compact decode. Expected", exp, "got", res)
	}

	exp = []byte{15, 1, 12, 11, 8 /*term*/, 16}
	res = CompactDecode("\x3f\x1c\xb8")

	if !bytes.Equal(res, exp) {
		t.Error("even terminated compact decode. Expected", exp, "got", res)
	}
}
