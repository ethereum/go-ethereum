package abi

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEncodePackedExample(t *testing.T) {
	// Test example from docs.soliditylang.org
	// int16(-1), bytes1(0x42), uint16(0x03), string("Hello, world!")
	int16T, _ := NewType("int16", "", nil)
	bytes1T, _ := NewType("bytes1", "", nil)
	want := common.Hex2Bytes("ffff42000348656c6c6f2c20776f726c6421")
	fmt.Print(want)

	types := []Type{
		int16T,
		bytes1T,
		Uint16,
		String,
	}
	values := []interface{}{
		int16(-1),
		[]byte{0x42},
		uint16(0x03),
		"Hello, world!",
	}
	encoded, err := EncodePacked(types, values)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, want) {
		t.Errorf("Could not properly encode using EncodePacked: got %v, want %v", encoded, want)
	}
}
