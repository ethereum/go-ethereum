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

package bigint

import (
	"bytes"
	"math/big"
	"testing"
)

func TestModExp(t *testing.T) {
	tests := []struct {
		name string
		base string
		exp  string
		mod  string
		want string
	}{
		{
			name: "simple",
			base: "2",
			exp:  "10",
			mod:  "1000",
			want: "24",
		},
		{
			name: "zero_exponent",
			base: "12345",
			exp:  "0",
			mod:  "67890",
			want: "1",
		},
		{
			name: "base_one",
			base: "1",
			exp:  "999999",
			mod:  "1000",
			want: "1",
		},
		{
			name: "zero_modulus",
			base: "2",
			exp:  "10",
			mod:  "0",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseBig := new(big.Int)
			baseBig.SetString(tt.base, 10)
			expBig := new(big.Int)
			expBig.SetString(tt.exp, 10)
			modBig := new(big.Int)
			modBig.SetString(tt.mod, 10)
			wantBig := new(big.Int)
			if tt.want != "" {
				wantBig.SetString(tt.want, 10)
			}

			result, err := ModExp(baseBig.Bytes(), expBig.Bytes(), modBig.Bytes())
			if err != nil {
				t.Fatalf("ModExp error: %v", err)
			}

			if !bytes.Equal(result, wantBig.Bytes()) {
				t.Errorf("ModExp result mismatch: got %x, want %x", result, wantBig.Bytes())
			}
		})
	}
}
