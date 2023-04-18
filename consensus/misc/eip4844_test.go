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
