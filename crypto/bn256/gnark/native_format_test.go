package bn256

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

func TestNativeGnarkFormatIncompatibility(t *testing.T) {
	// Use official gnark serialization
	_, _, g1Gen, _ := bn254.Generators()
	wrongSer := g1Gen.Bytes()

	var evmG1 G1
	_, err := evmG1.Unmarshal(wrongSer[:])
	if err == nil {
		t.Fatalf("points serialized using the official bn254 serialization algorithm, should not work with the evm format")
	}
}

func TestSerRoundTrip(t *testing.T) {
	_, _, g1Gen, g2Gen := bn254.Generators()

	expectedG1 := G1{inner: g1Gen}
	bytesG1 := expectedG1.Marshal()

	expectedG2 := G2{inner: g2Gen}
	bytesG2 := expectedG2.Marshal()

	var gotG1 G1
	gotG1.Unmarshal(bytesG1)

	var gotG2 G2
	gotG2.Unmarshal(bytesG2)

	if !expectedG1.inner.Equal(&gotG1.inner) {
		t.Errorf("serialization roundtrip failed for G1")
	}
	if !expectedG2.inner.Equal(&gotG2.inner) {
		t.Errorf("serialization roundtrip failed for G2")
	}
}
