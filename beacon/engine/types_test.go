// Copyright 2025 The go-ethereum Authors
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

package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

func TestBlobs(t *testing.T) {
	var (
		emptyBlob          = new(kzg4844.Blob)
		emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
		emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
		emptyCellProof, _  = kzg4844.ComputeCellProofs(emptyBlob)
	)
	header := types.Header{}
	block := types.NewBlock(&header, &types.Body{}, nil, nil)

	sidecarWithoutCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, []kzg4844.Proof{emptyBlobProof})
	env := BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithoutCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 1 {
		t.Fatalf("Expect 1 proof in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}

	sidecarWithCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, emptyCellProof)
	env = BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 128 {
		t.Fatalf("Expect 128 proofs in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}
}

// TestPayloadAttributesJSON verifies the JSON encoding of PayloadAttributes,
// in particular that the amsterdam targetGasLimit field survives a round trip
// and that attributes as sent by a consensus client on forkchoiceUpdatedV4
// decode correctly.
func TestPayloadAttributesJSON(t *testing.T) {
	// PayloadAttributesV4 as sent by a Gloas consensus client.
	input := `{
		"timestamp": "0x64",
		"prevRandao": "0x0202020202020202020202020202020202020202020202020202020202020202",
		"suggestedFeeRecipient": "0x0101010101010101010101010101010101010101",
		"withdrawals": [],
		"parentBeaconBlockRoot": "0x0303030303030303030303030303030303030303030303030303030303030303",
		"slotNumber": "0x10",
		"targetGasLimit": "0x11e1a300"
	}`
	var attr PayloadAttributes
	if err := json.Unmarshal([]byte(input), &attr); err != nil {
		t.Fatalf("failed to unmarshal payload attributes: %v", err)
	}
	if attr.SlotNumber == nil || *attr.SlotNumber != 16 {
		t.Fatalf("wrong slotNumber: %v", attr.SlotNumber)
	}
	if attr.TargetGasLimit == nil || *attr.TargetGasLimit != 300_000_000 {
		t.Fatalf("wrong targetGasLimit: %v", attr.TargetGasLimit)
	}
	// Round trip.
	encoded, err := json.Marshal(&attr)
	if err != nil {
		t.Fatalf("failed to marshal payload attributes: %v", err)
	}
	var decoded PayloadAttributes
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to unmarshal re-encoded payload attributes: %v", err)
	}
	if !reflect.DeepEqual(attr, decoded) {
		t.Fatalf("payload attributes changed in round trip:\nbefore: %+v\nafter: %+v", attr, decoded)
	}
}
