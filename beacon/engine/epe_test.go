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

package engine

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func makeTestPayload() *ExecutableData {
	return &ExecutableData{
		ParentHash:    common.HexToHash("0x01"),
		FeeRecipient:  common.HexToAddress("0x02"),
		StateRoot:     common.HexToHash("0x03"),
		ReceiptsRoot:  common.HexToHash("0x04"),
		LogsBloom:     make([]byte, 256),
		Random:        common.HexToHash("0x05"),
		Number:        100,
		GasLimit:      1000000,
		GasUsed:       500000,
		Timestamp:     1234567890,
		ExtraData:     []byte("extra"),
		BaseFeePerGas: big.NewInt(7),
		BlockHash:     common.HexToHash("0x08"),
		Transactions:  [][]byte{{0xaa, 0xbb}},
	}
}

func TestMarshalJSONRoundtrip(t *testing.T) {
	witness := hexutil.Bytes{0xde, 0xad}
	original := ExecutionPayloadEnvelope{
		ExecutionPayload: makeTestPayload(),
		BlockValue:       big.NewInt(12345),
		BlobsBundle: &BlobsBundle{
			Commitments: []hexutil.Bytes{{0x01, 0x02}},
			Proofs:      []hexutil.Bytes{{0x03, 0x04}},
			Blobs:       []hexutil.Bytes{{0x05, 0x06}},
		},
		Requests: [][]byte{{0xaa}, {0xbb, 0xcc}},
		Override: true,
		Witness:  &witness,
	}

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var decoded ExecutionPayloadEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	if decoded.ExecutionPayload.Number != original.ExecutionPayload.Number {
		t.Error("ExecutionPayload.Number mismatch")
	}
	if decoded.BlockValue.Cmp(original.BlockValue) != 0 {
		t.Errorf("BlockValue mismatch: got %v, want %v", decoded.BlockValue, original.BlockValue)
	}
	if len(decoded.BlobsBundle.Blobs) != len(original.BlobsBundle.Blobs) {
		t.Error("BlobsBundle.Blobs length mismatch")
	}
	if len(decoded.Requests) != len(original.Requests) {
		t.Error("Requests length mismatch")
	}
	if decoded.Override != original.Override {
		t.Error("Override mismatch")
	}
	if !bytes.Equal(*decoded.Witness, *original.Witness) {
		t.Error("Witness mismatch")
	}
}

func TestMarshalJSONNilPayload(t *testing.T) {
	env := ExecutionPayloadEnvelope{
		ExecutionPayload: nil,
		BlockValue:       big.NewInt(1),
	}
	_, err := env.MarshalJSON()
	if err == nil {
		t.Fatal("expected error for nil ExecutionPayload")
	}
}

// TestExecutionPayloadEnvelopeFieldCoverage guards against structural drift.
// If a field is added to or removed from ExecutionPayloadEnvelope, this test
// fails, reminding the developer to update MarshalJSON in marshal_epe.go.
func TestExecutionPayloadEnvelopeFieldCoverage(t *testing.T) {
	expected := []string{
		"ExecutionPayload",
		"BlockValue",
		"BlobsBundle",
		"Requests",
		"Override",
		"Witness",
	}
	typ := reflect.TypeOf(ExecutionPayloadEnvelope{})
	if typ.NumField() != len(expected) {
		t.Fatalf("ExecutionPayloadEnvelope has %d fields, expected %d — update MarshalJSON in marshal_epe.go",
			typ.NumField(), len(expected))
	}
	for i, name := range expected {
		if typ.Field(i).Name != name {
			t.Errorf("field %d: got %q, want %q — update MarshalJSON in marshal_epe.go",
				i, typ.Field(i).Name, name)
		}
	}
}
