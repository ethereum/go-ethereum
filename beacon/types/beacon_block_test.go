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

package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBlockFromJSON(t *testing.T) {
	type blocktest struct {
		file            string
		version         string
		wantSlot        uint64
		wantBlockNumber uint64
		wantBlockHash   common.Hash
	}
	tests := []blocktest{
		{
			file:            "block_deneb.json",
			version:         "deneb",
			wantSlot:        8631513,
			wantBlockNumber: 19431837,
			wantBlockHash:   common.HexToHash("0x4cf7d9108fc01b50023ab7cab9b372a96068fddcadec551630393b65acb1f34c"),
		},
		{
			file:            "block_capella.json",
			version:         "capella",
			wantSlot:        7378495,
			wantBlockNumber: 18189758,
			wantBlockHash:   common.HexToHash("0x802acf5c350f4252e31d83c431fcb259470250fa0edf49e8391cfee014239820"),
		},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", test.file))
			if err != nil {
				t.Fatal(err)
			}
			beaconBlock, err := BlockFromJSON(test.version, data)
			if err != nil {
				t.Fatal(err)
			}
			if beaconBlock.Slot() != test.wantSlot {
				t.Errorf("wrong slot number %d", beaconBlock.Slot())
			}
			execBlock, err := beaconBlock.ExecutionPayload()
			if err != nil {
				t.Fatalf("payload extraction failed: %v", err)
			}
			if execBlock.NumberU64() != test.wantBlockNumber {
				t.Errorf("wrong block number: %v", execBlock.NumberU64())
			}
			if execBlock.Hash() != test.wantBlockHash {
				t.Errorf("wrong block hash: %v", execBlock.Hash())
			}
		})
	}
}
