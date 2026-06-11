package serialize

import (
	"math/big"
	"testing"
)

// TestFixedSizeUintRejectsNegative guards against silent sign loss: because
// big.Int.Bytes() returns the absolute value, a negative Uint128/Uint256 would
// otherwise serialize identically to its positive counterpart. It must error
// instead.
func TestFixedSizeUintRejectsNegative(t *testing.T) {
	cases := []struct {
		name string
		ser  func() error
	}{
		{"Uint128", func() error { var w []byte; return (&Uint128{Value: big.NewInt(-1)}).Serialize(&w) }},
		{"Uint256", func() error { var w []byte; return (&Uint256{Value: big.NewInt(-1)}).Serialize(&w) }},
	}
	for _, tc := range cases {
		if err := tc.ser(); err == nil {
			t.Fatalf("%s: expected an error serializing a negative value, got nil (sign silently dropped)", tc.name)
		}
	}

	// Non-negative values still serialize to their fixed width.
	var w []byte
	if err := (&Uint256{Value: big.NewInt(1)}).Serialize(&w); err != nil {
		t.Fatalf("positive Uint256 rejected: %v", err)
	}
	if len(w) != 32 {
		t.Fatalf("Uint256 should serialize to 32 bytes, got %d", len(w))
	}
}
