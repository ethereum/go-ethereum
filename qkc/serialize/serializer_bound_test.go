package serialize

import (
	"math/big"
	"testing"
)

// TestSerializeBigIntLengthBound guards the variable-length big.Int encoding:
// its length prefix is a single byte, so a value needing more than 255 bytes
// would truncate the prefix and decode to a different value. It must error
// instead, and values at/under the limit must still round-trip.
func TestSerializeBigIntLengthBound(t *testing.T) {
	// 2^2048 needs 257 bytes (> 255): must be rejected.
	tooBig := new(big.Int).Lsh(big.NewInt(1), 2048)
	var w []byte
	if err := Serialize(&w, tooBig); err == nil {
		t.Fatal("expected an error serializing a big.Int needing >255 bytes (non-roundtrippable length prefix)")
	}

	// 2^2032 needs exactly 255 bytes: the maximum that fits, must round-trip.
	maxFit := new(big.Int).Lsh(big.NewInt(1), 8*254)
	if got := len(maxFit.Bytes()); got != 255 {
		t.Fatalf("test setup: expected 255 bytes, got %d", got)
	}
	w = nil
	if err := Serialize(&w, maxFit); err != nil {
		t.Fatalf("255-byte big.Int rejected: %v", err)
	}
	var back big.Int
	if err := Deserialize(NewByteBuffer(w), &back); err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if back.Cmp(maxFit) != 0 {
		t.Fatalf("round-trip mismatch")
	}
}
