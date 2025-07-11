// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gmp

import (
	"bytes"
	"math/big"
	"testing"
)

// TestWrapperVsBigInt compares the wrapper with Go's math/big
func TestWrapperVsBigInt(t *testing.T) {
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
			base: "123456789",
			exp:  "987654321",
			mod:  "1000000007",
		},
		{
			name: "large_numbers",
			base: "123456789012345678901234567890",
			exp:  "987654321098765432109876543210",
			mod:  "111111111111111111111111111111",
		},
		{
			name: "zero_exponent",
			base: "12345",
			exp:  "0",
			mod:  "67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse numbers
			baseBig := new(big.Int)
			baseBig.SetString(tt.base, 10)
			expBig := new(big.Int)
			expBig.SetString(tt.exp, 10)
			modBig := new(big.Int)
			modBig.SetString(tt.mod, 10)

			// Test with wrapper
			baseBytes := baseBig.Bytes()
			expBytes := expBig.Bytes()
			modBytes := modBig.Bytes()

			wrapperResult, err := ModExp(baseBytes, expBytes, modBytes)
			if err != nil {
				t.Fatalf("Wrapper error: %v", err)
			}

			// Test with math/big
			bigResult := new(big.Int)
			bigResult.Exp(baseBig, expBig, modBig)

			expectedResult := bigResult.Bytes()

			// Compare results
			if !bytes.Equal(wrapperResult, expectedResult) {
				t.Errorf("Results differ:\nWrapper:  %x\nExpected: %x",
					wrapperResult, expectedResult)
			}
		})
	}
}

// BenchmarkWrapperVsExisting compares performance
func BenchmarkModExp(b *testing.B) {
	base := make([]byte, 60)
	exp := make([]byte, 60)
	mod := make([]byte, 60)

	for i := range base {
		base[i] = byte(i * 17)
		exp[i] = byte(i * 31)
		mod[i] = byte(255 - i)
	}
	mod[59] |= 0x01

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ModExp(base, exp, mod)
	}
}

func BenchmarkModExpBigInt(b *testing.B) {
	baseBytes := make([]byte, 60)
	expBytes := make([]byte, 60)
	modBytes := make([]byte, 60)

	for i := range baseBytes {
		baseBytes[i] = byte(i * 17)
		expBytes[i] = byte(i * 31)
		modBytes[i] = byte(255 - i)
	}
	modBytes[59] |= 0x01

	base := new(big.Int).SetBytes(baseBytes)
	exp := new(big.Int).SetBytes(expBytes)
	mod := new(big.Int).SetBytes(modBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := new(big.Int)
		result.Exp(base, exp, mod)
		_ = result.Bytes()
	}
}

// TestLeadingZeros tests that leading zeros are handled correctly
func TestLeadingZeros(t *testing.T) {
	tests := []struct {
		name     string
		base     []byte
		exp      []byte
		mod      []byte
		expected []byte
	}{
		{
			name:     "base_with_leading_zeros",
			base:     []byte{0, 0, 0, 0, 2},
			exp:      []byte{3},
			mod:      []byte{7},
			expected: []byte{1}, // 2^3 mod 7 = 8 mod 7 = 1
		},
		{
			name:     "exp_with_leading_zeros",
			base:     []byte{2},
			exp:      []byte{0, 0, 0, 3},
			mod:      []byte{7},
			expected: []byte{1}, // 2^3 mod 7 = 8 mod 7 = 1
		},
		{
			name:     "mod_with_leading_zeros",
			base:     []byte{2},
			exp:      []byte{3},
			mod:      []byte{0, 0, 0, 0, 7},
			expected: []byte{1}, // 2^3 mod 7 = 8 mod 7 = 1
		},
		{
			name:     "all_with_leading_zeros",
			base:     []byte{0, 0, 2},
			exp:      []byte{0, 0, 3},
			mod:      []byte{0, 0, 7},
			expected: []byte{1}, // 2^3 mod 7 = 8 mod 7 = 1
		},
		{
			name:     "base_all_zeros",
			base:     []byte{0, 0, 0},
			exp:      []byte{5},
			mod:      []byte{7},
			expected: []byte{}, // 0^5 mod 7 = 0
		},
		{
			name:     "exp_all_zeros",
			base:     []byte{5},
			exp:      []byte{0, 0, 0},
			mod:      []byte{7},
			expected: []byte{1}, // 5^0 mod 7 = 1
		},
		{
			name:     "base_and_exp_all_zeros",
			base:     []byte{0, 0},
			exp:      []byte{0, 0},
			mod:      []byte{7},
			expected: []byte{1}, // 0^0 mod 7 = 1 (by convention)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ModExp(tt.base, tt.exp, tt.mod)
			if err != nil {
				t.Fatalf("ModExp error: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Results differ:\nGot:      %x\nExpected: %x",
					result, tt.expected)
			}
		})
	}
}

// TestSpecialCases tests the special case optimizations
func TestSpecialCases(t *testing.T) {
	tests := []struct {
		name     string
		base     []byte
		exp      []byte
		mod      []byte
		expected []byte
	}{
		{
			name:     "base_one_mod_large",
			base:     []byte{1},
			exp:      []byte{255, 255, 255, 255}, // large exponent
			mod:      []byte{100},
			expected: []byte{1},
		},
		{
			name:     "base_one_mod_one",
			base:     []byte{1},
			exp:      []byte{10},
			mod:      []byte{1},
			expected: []byte{}, // 1 mod 1 = 0
		},
		{
			name:     "zero_modulus",
			base:     []byte{2},
			exp:      []byte{3},
			mod:      []byte{},
			expected: []byte{},
		},
		{
			name:     "all_zero_modulus",
			base:     []byte{2},
			exp:      []byte{3},
			mod:      []byte{0, 0, 0},
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ModExp(tt.base, tt.exp, tt.mod)
			if err != nil {
				t.Fatalf("ModExp error: %v", err)
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Results differ:\nGot:      %x\nExpected: %x",
					result, tt.expected)
			}
		})
	}
}
