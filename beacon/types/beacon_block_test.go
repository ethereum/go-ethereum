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
			// fulu reuses the electra block format, so an electra block
			// must decode successfully with the fulu fork name as well
			file:            "block_electra_withdrawals.json",
			version:         "fulu",
			wantSlot:        151850,
			wantBlockNumber: 141654,
			wantBlockHash:   common.HexToHash("0xf6730485a38be5ada3e110990a2c7adaabd2e8d4a49782134f1a8bfbc246a5d7"),
		},
		{
			file:            "block_electra_withdrawals.json",
			version:         "electra",
			wantSlot:        151850,
			wantBlockNumber: 141654,
			wantBlockHash:   common.HexToHash("0xf6730485a38be5ada3e110990a2c7adaabd2e8d4a49782134f1a8bfbc246a5d7"),
		},
		{
			file:            "block_electra_deposits.json",
			version:         "electra",
			wantSlot:        151016,
			wantBlockNumber: 140858,
			wantBlockHash:   common.HexToHash("0x1f2637170986346c7993d5adbadbebbf4c9ed89c6a4d2dff653db99c8c168076"),
		},
		{
			file:            "block_electra_consolidations.json",
			version:         "electra",
			wantSlot:        151717,
			wantBlockNumber: 141529,
			wantBlockHash:   common.HexToHash("0xc8807f7a1f96b0a073ff27065776dd21eff6b7e64079c60bffd33f690efbb330"),
		},
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
		t.Run(test.file+"/"+test.version, func(t *testing.T) {
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

func TestGloasBlockFromJSON(t *testing.T) {
	data := []byte(`{
		"slot":"144512",
		"proposer_index":"780",
		"parent_root":"0x0000000000000000000000000000000000000000000000000000000000000001",
		"state_root":"0x0000000000000000000000000000000000000000000000000000000000000002",
		"body":{"signed_execution_payload_bid":{"message":{"block_hash":"0x0000000000000000000000000000000000000000000000000000000000000002"}}}
	}`)
	block, err := BlockFromJSON("gloas", data)
	if err != nil {
		t.Fatal(err)
	}
	if block.Slot() != 144512 {
		t.Fatalf("slot = %d, want 144512", block.Slot())
	}
	if _, err := block.ExecutionPayload(); err == nil {
		t.Fatal("missing Gloas envelope accepted")
	}
	header := Header{
		Slot:          144512,
		ProposerIndex: 780,
		ParentRoot:    common.HexToHash("0x1"),
		StateRoot:     common.HexToHash("0x2"),
		BodyRoot:      common.HexToHash("0x3"),
	}
	blockRoot := header.Hash()
	if err := block.SetGloasHeader(blockRoot, header); err != nil {
		t.Fatalf("set Gloas header: %v\nblock header: %+v\nwant: %+v", err, block.Header(), header)
	}
	if block.Root() != blockRoot {
		t.Fatalf("root = %x, want %x", block.Root(), blockRoot)
	}
	wrongEnvelope := []byte(`{
		"beacon_block_root":"0x0000000000000000000000000000000000000000000000000000000000000002",
		"parent_beacon_block_root":"0x0000000000000000000000000000000000000000000000000000000000000001"
	}`)
	if err := block.SetGloasPayloadEnvelope(blockRoot, wrongEnvelope); err == nil {
		t.Fatal("envelope with wrong beacon root accepted")
	}
}
