package types

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestUint32RLP(t *testing.T) {
	v := Uint32(0x12345678)
	encoded, err := rlp.EncodeToBytes(&v)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: 0x84 + big-endian uint32
	want := []byte{0x84, 0x12, 0x34, 0x56, 0x78}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("encode: want %x, got %x", want, encoded)
	}
	var v2 Uint32
	if err := rlp.DecodeBytes(encoded, &v2); err != nil {
		t.Fatal(err)
	}
	if v2 != v {
		t.Fatalf("decode: want %d, got %d", v, v2)
	}
}
