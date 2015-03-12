package rpc

import (
	"bytes"
	"testing"
)

//fromHex
func TestFromHex(t *testing.T) {
	input := "0x01"
	expected := []byte{1}
	result := fromHex(input)
	if bytes.Compare(expected, result) != 0 {
		t.Errorf("Expected % x got % x", expected, result)
	}
}

func TestFromHexOddLength(t *testing.T) {
	input := "0x1"
	expected := []byte{1}
	result := fromHex(input)
	if bytes.Compare(expected, result) != 0 {
		t.Errorf("Expected % x got % x", expected, result)
	}
}
