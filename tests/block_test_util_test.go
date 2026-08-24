// Copyright 2026 The go-ethereum Authors
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

package tests

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestValidateHeaderBaseFee(t *testing.T) {
	jsonZero := mustHexOrDecimal256(t, "0x0")
	jsonZeroPad := mustHexOrDecimal256(t, "0x00")
	jsonOne := mustHexOrDecimal256(t, "0x1")
	rlpZero := mustRLPBigInt(t, []byte{0x80})
	one := big.NewInt(1)

	if reflect.DeepEqual(jsonZero, rlpZero) {
		t.Fatal("test setup: JSON 0x0 and RLP 0 are unexpectedly DeepEqual")
	}

	tests := []struct {
		name    string
		want    *big.Int
		have    *big.Int
		wantErr bool
	}{
		{name: "both nil", want: nil, have: nil},
		{name: "json 0x0 vs rlp 0", want: jsonZero, have: rlpZero},
		{name: "json 0x00 vs rlp 0", want: jsonZeroPad, have: rlpZero},
		{name: "same non-zero", want: jsonOne, have: new(big.Int).Set(one)},
		{name: "nil vs zero", want: nil, have: rlpZero, wantErr: true},
		{name: "zero vs nil", want: jsonZero, have: nil, wantErr: true},
		{name: "different values", want: jsonOne, have: rlpZero, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeader(
				&btHeader{Number: one, Difficulty: one, BaseFeePerGas: tt.want},
				&types.Header{Number: one, Difficulty: one, BaseFee: tt.have},
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mustHexOrDecimal256(t *testing.T, text string) *big.Int {
	t.Helper()
	var n math.HexOrDecimal256
	if err := n.UnmarshalText([]byte(text)); err != nil {
		t.Fatal(err)
	}
	return (*big.Int)(&n)
}

func mustRLPBigInt(t *testing.T, enc []byte) *big.Int {
	t.Helper()
	var n big.Int
	if err := rlp.DecodeBytes(enc, &n); err != nil {
		t.Fatal(err)
	}
	return &n
}
