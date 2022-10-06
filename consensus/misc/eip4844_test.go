// Copyright 2022 The go-ethereum Authors
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

package misc

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestFakeExponential(t *testing.T) {
	var tests = []struct {
		factor, num, denom int64
		want               int64
	}{
		{1, 2, 1, 6}, // approximate 7.389
		{1, 4, 2, 6},
		{1, 3, 1, 16}, // approximate 20.09
		{1, 6, 2, 18},
		{1, 4, 1, 49}, // approximate 54.60
		{1, 8, 2, 50},
		{10, 8, 2, 542}, // approximate 540.598
		{11, 8, 2, 596}, // approximate 600.58
		{1, 5, 1, 136},  // approximate 148.4
		{1, 5, 2, 11},   // approximate 12.18
		{2, 5, 2, 23},   // approximate 24.36
	}

	for _, tt := range tests {
		factor := big.NewInt(tt.factor)
		num := big.NewInt(tt.num)
		denom := big.NewInt(tt.denom)
		result := FakeExponential(factor, num, denom)
		//t.Logf("%v*e^(%v/%v): %v", factor, num, denom, result)
		if tt.want != result.Int64() {
			t.Errorf("got %v want %v", result, tt.want)
		}
	}
}

func TestCalcExcessDataGas(t *testing.T) {
	var tests = []struct {
		parentExcessDataGas int64
		newBlobs            int
		want                int64
	}{
		{0, 0, 0},
		{0, 1, 0},
		{0, params.TargetDataGasPerBlock / params.DataGasPerBlob, 0},
		{0, (params.TargetDataGasPerBlock / params.DataGasPerBlob) + 1, params.DataGasPerBlob},
		{100000, (params.TargetDataGasPerBlock / params.DataGasPerBlob) + 1, params.DataGasPerBlob + 100000},
		{params.TargetDataGasPerBlock, 1, params.DataGasPerBlob},
		{params.TargetDataGasPerBlock, 0, 0},
		{params.TargetDataGasPerBlock, (params.TargetDataGasPerBlock / params.DataGasPerBlob), params.TargetDataGasPerBlock},
	}

	for _, tt := range tests {
		parentExcessDataGas := big.NewInt(tt.parentExcessDataGas)
		result := CalcExcessDataGas(parentExcessDataGas, tt.newBlobs)
		if tt.want != result.Int64() {
			t.Errorf("got %v want %v", result, tt.want)
		}
	}

	// Test nil value for parentExcessDataGas
	result := CalcExcessDataGas(nil, (params.TargetDataGasPerBlock/params.DataGasPerBlob)+1)
	if result.Int64() != params.DataGasPerBlob {
		t.Errorf("got %v want %v", result, params.DataGasPerBlob)
	}
}
