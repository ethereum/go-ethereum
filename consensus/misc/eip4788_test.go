// Copyright 2023 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestCalcBeaconRootIndices(t *testing.T) {
	tests := []struct {
		Header  types.Header
		TimeKey common.Hash
		Time    common.Hash
		RootKey common.Hash
		Root    common.Hash
	}{
		{
			Header:  types.Header{Time: 0, BeaconRoot: &common.Hash{1}},
			TimeKey: common.Hash{},
			Time:    common.Hash{},
			RootKey: common.BigToHash(big.NewInt(int64(params.HistoricalRootsModulus))),
			Root:    common.Hash{1},
		},
		{
			Header:  types.Header{Time: 120265298769267, BeaconRoot: &common.Hash{0xff, 0xfe, 0xfc}},
			TimeKey: common.BytesToHash([]byte{0xf5, 0x73}),                         // 120265298769267 % params.HistoricalRootsModulus
			Time:    common.BytesToHash([]byte{0x6D, 0x61, 0x72, 0x69, 0x75, 0x73}), // 120265298769267 -> hex
			RootKey: common.BytesToHash([]byte{0x02, 0x75, 0x73}),                   // params.HistoricalRootsModulus + 0xf5 0x73
			Root:    common.Hash{0xff, 0xfe, 0xfc},
		},
	}

	for _, test := range tests {
		timeKey, time, rootKey, root := calcBeaconRootIndices(&test.Header)
		if timeKey != test.TimeKey {
			t.Fatalf("invalid time key: got %v want %v", timeKey, test.TimeKey)
		}
		if time != test.Time {
			t.Fatalf("invalid time: got %v want %v", time, test.Time)
		}
		if rootKey != test.RootKey {
			t.Fatalf("invalid time key: got %v want %v", rootKey, test.RootKey)
		}
		if root != test.Root {
			t.Fatalf("invalid time key: got %v want %v", root, test.Root)
		}
	}
}
