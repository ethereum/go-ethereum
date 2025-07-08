package gmp

import (
	"bytes"
	"math/big"
	"testing"
)

// TestModExpAgainstBigInt tests our GMP implementation against Go's math/big
func TestModExpAgainstBigInt(t *testing.T) {
	tests := []struct {
		name string
		base string
		exp  string
		mod  string
	}{
		{
			name: "small_numbers",
			base: "2",
			exp:  "10",
			mod:  "1000",
		},
		{
			name: "medium_numbers",
			base: "123",
			exp:  "456",
			mod:  "789",
		},
		{
			name: "large_base",
			base: "123456789012345678901234567890",
			exp:  "2",
			mod:  "1000000007",
		},
		{
			name: "large_exponent",
			base: "2",
			exp:  "123456789012345678901234567890",
			mod:  "1000000007",
		},
		{
			name: "all_large",
			base: "123456789012345678901234567890",
			exp:  "987654321098765432109876543210",
			mod:  "111111111111111111111111111111",
		},
		{
			name: "prime_modulus",
			base: "12345",
			exp:  "67890",
			mod:  "2147483647", // 2^31 - 1 (Mersenne prime)
		},
		{
			name: "fermat_little_theorem",
			base: "3",
			exp:  "16",
			mod:  "17",
		},
		{
			name: "carmichael_number",
			base: "2",
			exp:  "560",
			mod:  "561", // First Carmichael number
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GMP calculation
			gmpBase := NewInt()
			gmpBase.SetString(tt.base, 10)
			
			gmpExp := NewInt()
			gmpExp.SetString(tt.exp, 10)
			
			gmpMod := NewInt()
			gmpMod.SetString(tt.mod, 10)
			
			gmpResult := NewInt()
			gmpResult.ExpMod(gmpBase, gmpExp, gmpMod)
			
			// math/big calculation
			bigBase := new(big.Int)
			bigBase.SetString(tt.base, 10)
			
			bigExp := new(big.Int)
			bigExp.SetString(tt.exp, 10)
			
			bigMod := new(big.Int)
			bigMod.SetString(tt.mod, 10)
			
			bigResult := new(big.Int)
			bigResult.Exp(bigBase, bigExp, bigMod)
			
			// Compare results
			if gmpResult.String() != bigResult.String() {
				t.Errorf("ModExp mismatch:\n  GMP: %s\n  big: %s", 
					gmpResult.String(), bigResult.String())
			}
		})
	}
}

// TestSetBytesAgainstBigInt tests byte conversion against math/big
func TestSetBytesAgainstBigInt(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
	}{
		{"empty", []byte{}},
		{"single_byte", []byte{0x42}},
		{"two_bytes", []byte{0x12, 0x34}},
		{"four_bytes", []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{"eight_bytes", []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}},
		{"all_zeros", []byte{0x00, 0x00, 0x00, 0x00}},
		{"leading_zeros", []byte{0x00, 0x00, 0x12, 0x34}},
		{"all_ones", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GMP SetBytes
			gmpInt := NewInt()
			gmpInt.SetBytes(tt.bytes)
			
			// math/big SetBytes
			bigInt := new(big.Int)
			bigInt.SetBytes(tt.bytes)
			
			// Compare string representations
			if gmpInt.String() != bigInt.String() {
				t.Errorf("SetBytes mismatch for %x:\n  GMP: %s\n  big: %s",
					tt.bytes, gmpInt.String(), bigInt.String())
			}
			
			// Test round trip
			gmpBytes := gmpInt.Bytes()
			bigBytes := bigInt.Bytes()
			
			// Handle empty/zero cases
			// Note: both empty input and all-zeros should produce empty output
			if len(bigBytes) == 0 {
				if len(gmpBytes) != 0 {
					t.Errorf("Expected empty bytes, got %x", gmpBytes)
				}
				return
			}
			
			// Compare bytes (handling leading zeros)
			if !bytesEqual(gmpBytes, bigBytes) {
				t.Errorf("Bytes() mismatch:\n  GMP: %x\n  big: %x",
					gmpBytes, bigBytes)
			}
		})
	}
}

// TestModExpByteArrays tests modular exponentiation with byte arrays
func TestModExpByteArrays(t *testing.T) {
	// Test case: RSA-like encryption with byte arrays
	msgBytes := []byte("Hello!")
	
	// Convert to numbers
	gmpMsg := NewInt()
	gmpMsg.SetBytes(msgBytes)
	
	bigMsg := new(big.Int)
	bigMsg.SetBytes(msgBytes)
	
	// Public exponent (common RSA value)
	e := "65537"
	
	// Small modulus for testing
	n := "12345678901234567890"
	
	// GMP calculation
	gmpE := NewInt()
	gmpE.SetString(e, 10)
	
	gmpN := NewInt()
	gmpN.SetString(n, 10)
	
	gmpResult := NewInt()
	gmpResult.ExpMod(gmpMsg, gmpE, gmpN)
	
	// math/big calculation
	bigE := new(big.Int)
	bigE.SetString(e, 10)
	
	bigN := new(big.Int)
	bigN.SetString(n, 10)
	
	bigResult := new(big.Int)
	bigResult.Exp(bigMsg, bigE, bigN)
	
	// Compare
	if gmpResult.String() != bigResult.String() {
		t.Errorf("Byte array ModExp mismatch:\n  GMP: %s\n  big: %s",
			gmpResult.String(), bigResult.String())
	}
	
	// Verify we can convert back to bytes
	resultBytes := gmpResult.Bytes()
	if len(resultBytes) == 0 {
		t.Error("Result bytes should not be empty")
	}
}

// Benchmark against math/big
func BenchmarkModExpGMP(b *testing.B) {
	base := NewInt()
	base.SetString("123456789012345678901234567890123456789012345678901234567890", 10)
	
	exp := NewInt()
	exp.SetString("987654321098765432109876543210987654321098765432109876543210", 10)
	
	mod := NewInt()
	mod.SetString("111111111111111111111111111111111111111111111111111111111111", 10)
	
	result := NewInt()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.ExpMod(base, exp, mod)
	}
}

func BenchmarkModExpBigInt(b *testing.B) {
	base := new(big.Int)
	base.SetString("123456789012345678901234567890123456789012345678901234567890", 10)
	
	exp := new(big.Int)
	exp.SetString("987654321098765432109876543210987654321098765432109876543210", 10)
	
	mod := new(big.Int)
	mod.SetString("111111111111111111111111111111111111111111111111111111111111", 10)
	
	result := new(big.Int)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.Exp(base, exp, mod)
	}
}

// Helper function
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}