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

func TestHashJsonValidation(t *testing.T) {
	var h Hash
	var tests = []struct {
		Prefix string
		Size   int
		Error  error
	}{
		{"", 2, hashJsonLengthErr},
		{"", 62, hashJsonLengthErr},
		{"", 66, hashJsonLengthErr},
		{"", 65, hashJsonLengthErr},
		{"0X", 64, nil},
		{"0x", 64, nil},
		{"0x", 62, hashJsonLengthErr},
	}
	for i, test := range tests {
		if err := h.UnmarshalJSON(append([]byte(test.Prefix), make([]byte, test.Size)...)); err != test.Error {
			t.Error(i, "expected", test.Error, "got", err)
		}
	}
}
