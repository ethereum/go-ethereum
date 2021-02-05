package abi

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEncodePacked(t *testing.T) {

	Uint8ArrT, _ := NewType("uint8[3]", "", nil)
	int16T, _ := NewType("int16", "", nil)
	bytes1T, _ := NewType("bytes1", "", nil)
	bytesT, _ := NewType("bytes", "", nil)

	type test struct {
		name    string
		encoded string
		types   []Type
		values  []interface{}
	}

	tests := []test{
		{
			// Test example from docs.soliditylang.org
			// int16(-1), bytes1(0x42), uint16(0x03), string("Hello, world!")
			name:    "TestSolidityExample",
			encoded: "ffff42000348656c6c6f2c20776f726c6421",
			types: []Type{
				int16T,
				bytes1T,
				Uint16,
				String,
			},
			values: []interface{}{
				int16(-1),
				[]byte{0x42},
				uint16(0x03),
				"Hello, world!",
			},
		},
		{
			name: "TestArray",
			encoded: "0000000000000000000000000000000000000000000000000000000000000001" +
				"0000000000000000000000000000000000000000000000000000000000000002" +
				"0000000000000000000000000000000000000000000000000000000000000042" +
				"41414141" +
				"42424242",
			types: []Type{
				Uint8ArrT,
				String,
				String,
			},
			values: []interface{}{
				uint8(0x1),
				uint8(0x2),
				uint8(0x42),
				"AAAA",
				"BBBB",
			},
		},
		{
			name:    "TestBytesBool",
			encoded: "414141410100",
			types: []Type{
				bytesT,
				Bool,
				Bool,
			},
			values: []interface{}{
				[]byte{0x41, 0x41, 0x41, 0x41},
				true,
				false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := common.Hex2Bytes(tt.encoded)
			encoded, err := EncodePacked(tt.types, tt.values)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, want) {
				t.Errorf("Could not properly encode using EncodePacked: got %v, want %v", encoded, want)
			}
		})
	}
}
