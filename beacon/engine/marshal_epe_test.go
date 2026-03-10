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

// canonicalEnvelope is a reference type for ExecutionPayloadEnvelope that uses
// standard json.Marshal (no custom MarshalJSON). It mirrors the gencodec type
// overrides so its output matches what the generated code would produce.
type canonicalEnvelope struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"`
	BlockValue       *hexutil.Big    `json:"blockValue"`
	BlobsBundle      *BlobsBundle    `json:"blobsBundle"`
	Requests         []hexutil.Bytes `json:"executionRequests"`
	Override         bool            `json:"shouldOverrideBuilder"`
	Witness          *hexutil.Bytes  `json:"witness,omitempty"`
}

func toCanonical(e *ExecutionPayloadEnvelope) *canonicalEnvelope {
	c := &canonicalEnvelope{
		ExecutionPayload: e.ExecutionPayload,
		BlockValue:       (*hexutil.Big)(e.BlockValue),
		BlobsBundle:      e.BlobsBundle,
		Override:         e.Override,
		Witness:          e.Witness,
	}
	if e.Requests != nil {
		c.Requests = make([]hexutil.Bytes, len(e.Requests))
		for i, r := range e.Requests {
			c.Requests[i] = r
		}
	}
	return c
}

// compactJSON returns the compacted form of a JSON byte slice.
func compactJSON(data []byte) []byte {
	var buf bytes.Buffer
	json.Compact(&buf, data)
	return buf.Bytes()
}

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

func TestMarshalJSON(t *testing.T) {
	witness := hexutil.Bytes{0xde, 0xad}
	tests := []struct {
		name string
		env  ExecutionPayloadEnvelope
	}{
		{
			name: "full envelope with blobs",
			env: ExecutionPayloadEnvelope{
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
			},
		},
		{
			name: "nil BlobsBundle",
			env: ExecutionPayloadEnvelope{
				ExecutionPayload: makeTestPayload(),
				BlockValue:       big.NewInt(0),
			},
		},
		{
			name: "nil Requests",
			env: ExecutionPayloadEnvelope{
				ExecutionPayload: makeTestPayload(),
				BlockValue:       big.NewInt(1),
				Requests:         nil,
			},
		},
		{
			name: "empty Requests",
			env: ExecutionPayloadEnvelope{
				ExecutionPayload: makeTestPayload(),
				BlockValue:       big.NewInt(1),
				Requests:         [][]byte{},
			},
		},
		{
			name: "nil Witness",
			env: ExecutionPayloadEnvelope{
				ExecutionPayload: makeTestPayload(),
				BlockValue:       big.NewInt(1),
				Witness:          nil,
			},
		},
		{
			name: "empty blobs bundle arrays",
			env: ExecutionPayloadEnvelope{
				ExecutionPayload: makeTestPayload(),
				BlockValue:       big.NewInt(1),
				BlobsBundle: &BlobsBundle{
					Commitments: []hexutil.Bytes{},
					Proofs:      []hexutil.Bytes{},
					Blobs:       []hexutil.Bytes{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hand-rolled marshal.
			got, err := tt.env.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}

			// Canonical marshal via reference struct.
			want, err := json.Marshal(toCanonical(&tt.env))
			if err != nil {
				t.Fatalf("canonical marshal error: %v", err)
			}

			if !bytes.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
			}
		})
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
	if err := decoded.UnmarshalJSON(data); err != nil {
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
