// Copyright 2015 The go-ethereum Authors
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

package common

import "testing"

func TestBytesConversion(t *testing.T) {
	bytes := []byte{5}
	hash := BytesToHash(bytes)

	var exp Hash
	exp[31] = 5

	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

// Tests whether addresses are correctly matched against allowed form and data
// content.
func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		{"", false},                                             // Empty, without optional 0x prefix
		{"0x", false},                                           // Empty, with optional 0x prefix
		{"00", false},                                           // Too short, without optional 0x prefix
		{"0x00", false},                                         // Too short, with optional 0x prefix
		{"00000000000000000000000000000000000000", false},       // Too short (even), without optional 0x prefix
		{"0x00000000000000000000000000000000000000", false},     // Too short (even), with optional 0x prefix
		{"000000000000000000000000000000000000000", false},      // Too short (odd), without optional 0x prefix
		{"0x000000000000000000000000000000000000000", false},    // Too short (odd), with optional 0x prefix
		{"0000000000000000000000000000000000000000", true},      // Valid, without optional 0x prefix
		{"0x0000000000000000000000000000000000000000", true},    // Valid, with optional 0x prefix
		{"0x00000000000000000000000000000000000000", false},     // Length / prefix combo invalidity
		{"00x0000000000000000000000000000000000000", false},     // Invalid content, without optional 0x prefix
		{"0x0x00000000000000000000000000000000000000", false},   // Invalid content, with optional 0x prefix
		{"abcdefghijklmnopqrstuvwxyz0123456789xxxx", false},     // Invalid content, without optional 0x prefix
		{"0xabcdefghijklmnopqrstuvwxyz0123456789xxxx", false},   // Invalid content, with optional 0x prefix
		{"00000000000000000000000000000000000000000", false},    // Too long (odd), without optional 0x prefix
		{"0x00000000000000000000000000000000000000000", false},  // Too long (odd), with optional 0x prefix
		{"000000000000000000000000000000000000000000", false},   // Too long (even), without optional 0x prefix
		{"0x000000000000000000000000000000000000000000", false}, // Too long (even), with optional 0x prefix
	}

	for i, tt := range tests {
		if valid := IsHexAddress(tt.address); valid != tt.valid {
			t.Errorf("test %d: address validity mismatch: have %v, want %v", i, valid, tt.valid)
		}
	}
}
