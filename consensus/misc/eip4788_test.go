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
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type tcInput struct {
	HeaderTime       uint64
	HeaderBeaconRoot common.Hash

	TimeKey common.Hash
	Time    common.Hash
	RootKey common.Hash
	Root    common.Hash
}

func TestCalcBeaconRootIndices(t *testing.T) {
	data, err := os.ReadFile("./testdata/eip4788_beaconroot.json")
	if err != nil {
		t.Fatal(err)
	}
	var tests []tcInput
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}
	for i, tc := range tests {
		header := types.Header{Time: tc.HeaderTime, BeaconRoot: &tc.HeaderBeaconRoot}
		haveTimeKey, haveTime, haveRootKey, haveRoot := calcBeaconRootIndices(&header)
		if haveTimeKey != tc.TimeKey {
			t.Errorf("test %d: invalid time key: \nhave %v\nwant %v", i, haveTimeKey, tc.TimeKey)
		}
		if haveTime != tc.Time {
			t.Errorf("test %d: invalid time: \nhave %v\nwant %v", i, haveTime, tc.Time)
		}
		if haveRootKey != tc.RootKey {
			t.Errorf("test %d: invalid root key: \nhave %v\nwant %v", i, haveRootKey, tc.RootKey)
		}
		if haveRoot != tc.Root {
			t.Errorf("test %d: invalid root: \nhave %v\nwant %v", i, haveRoot, tc.Root)
		}
	}
}
