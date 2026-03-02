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
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestPayloadStatusSSZRoundTrip(t *testing.T) {
	hash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	errMsg := "something went wrong"

	tests := []struct {
		name string
		ps   PayloadStatusV1
	}{
		{
			name: "valid with hash",
			ps: PayloadStatusV1{
				Status:          VALID,
				LatestValidHash: &hash,
			},
		},
		{
			name: "invalid with error",
			ps: PayloadStatusV1{
				Status:          INVALID,
				LatestValidHash: &hash,
				ValidationError: &errMsg,
			},
		},
		{
			name: "syncing no hash",
			ps: PayloadStatusV1{
				Status: SYNCING,
			},
		},
		{
			name: "accepted",
			ps: PayloadStatusV1{
				Status: ACCEPTED,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodePayloadStatusSSZ(&tt.ps)
			decoded, err := DecodePayloadStatusSSZ(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if decoded.Status != tt.ps.Status {
				t.Errorf("status mismatch: got %s, want %s", decoded.Status, tt.ps.Status)
			}
			if (decoded.LatestValidHash == nil) != (tt.ps.LatestValidHash == nil) {
				t.Errorf("hash nil mismatch")
			}
			if decoded.LatestValidHash != nil && *decoded.LatestValidHash != *tt.ps.LatestValidHash {
				t.Errorf("hash mismatch")
			}
			if tt.ps.ValidationError != nil {
				if decoded.ValidationError == nil || *decoded.ValidationError != *tt.ps.ValidationError {
					t.Errorf("validation error mismatch")
				}
			}
		})
	}
}

func TestForkchoiceStateSSZRoundTrip(t *testing.T) {
	fcs := &ForkchoiceStateV1{
		HeadBlockHash:      common.HexToHash("0xaaaa"),
		SafeBlockHash:      common.HexToHash("0xbbbb"),
		FinalizedBlockHash: common.HexToHash("0xcccc"),
	}

	encoded := EncodeForkchoiceStateSSZ(fcs)
	if len(encoded) != 96 {
		t.Fatalf("expected 96 bytes, got %d", len(encoded))
	}

	decoded, err := DecodeForkchoiceStateSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.HeadBlockHash != fcs.HeadBlockHash {
		t.Errorf("head hash mismatch")
	}
	if decoded.SafeBlockHash != fcs.SafeBlockHash {
		t.Errorf("safe hash mismatch")
	}
	if decoded.FinalizedBlockHash != fcs.FinalizedBlockHash {
		t.Errorf("finalized hash mismatch")
	}
}

func TestForkChoiceResponseSSZRoundTrip(t *testing.T) {
	hash := common.HexToHash("0x1234")
	pid := PayloadID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	tests := []struct {
		name string
		resp ForkChoiceResponse
	}{
		{
			name: "with payload id",
			resp: ForkChoiceResponse{
				PayloadStatus: PayloadStatusV1{
					Status:          VALID,
					LatestValidHash: &hash,
				},
				PayloadID: &pid,
			},
		},
		{
			name: "without payload id",
			resp: ForkChoiceResponse{
				PayloadStatus: PayloadStatusV1{
					Status: SYNCING,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeForkChoiceResponseSSZ(&tt.resp)
			decoded, err := DecodeForkChoiceResponseSSZ(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if decoded.PayloadStatus.Status != tt.resp.PayloadStatus.Status {
				t.Errorf("status mismatch: got %s, want %s", decoded.PayloadStatus.Status, tt.resp.PayloadStatus.Status)
			}
			if (decoded.PayloadID == nil) != (tt.resp.PayloadID == nil) {
				t.Errorf("payloadID nil mismatch: got %v, want %v", decoded.PayloadID, tt.resp.PayloadID)
			}
			if decoded.PayloadID != nil && *decoded.PayloadID != *tt.resp.PayloadID {
				t.Errorf("payloadID mismatch: got %x, want %x", decoded.PayloadID, tt.resp.PayloadID)
			}
		})
	}
}

func TestCapabilitiesSSZRoundTrip(t *testing.T) {
	caps := []string{"engine_newPayloadV1", "engine_forkchoiceUpdatedV1", "engine_getPayloadV1"}
	encoded := EncodeCapabilitiesSSZ(caps)
	decoded, err := DecodeCapabilitiesSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != len(caps) {
		t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(caps))
	}
	for i, c := range caps {
		if decoded[i] != c {
			t.Errorf("capability[%d] mismatch: got %s, want %s", i, decoded[i], c)
		}
	}
}

func TestCommunicationChannelsSSZRoundTrip(t *testing.T) {
	channels := []CommunicationChannel{
		{Protocol: "json_rpc", URL: "localhost:8551"},
		{Protocol: "ssz_rest", URL: "http://localhost:8552"},
	}
	encoded := EncodeCommunicationChannelsSSZ(channels)
	decoded, err := DecodeCommunicationChannelsSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != len(channels) {
		t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(channels))
	}
	for i, ch := range channels {
		if decoded[i].Protocol != ch.Protocol {
			t.Errorf("channel[%d].Protocol mismatch: got %s, want %s", i, decoded[i].Protocol, ch.Protocol)
		}
		if decoded[i].URL != ch.URL {
			t.Errorf("channel[%d].URL mismatch: got %s, want %s", i, decoded[i].URL, ch.URL)
		}
	}
}

func TestClientVersionSSZRoundTrip(t *testing.T) {
	cv := &ClientVersionV1{
		Code:    "GE",
		Name:    "go-ethereum",
		Version: "1.14.0",
		Commit:  "0xdeadbeef",
	}
	encoded := EncodeClientVersionSSZ(cv)
	decoded, err := DecodeClientVersionSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded.Code != cv.Code || decoded.Name != cv.Name || decoded.Version != cv.Version || decoded.Commit != cv.Commit {
		t.Errorf("mismatch: got %+v, want %+v", decoded, cv)
	}
}

func TestClientVersionsSSZRoundTrip(t *testing.T) {
	versions := []ClientVersionV1{
		{Code: "GE", Name: "go-ethereum", Version: "1.14.0", Commit: "0xabcd"},
		{Code: "ER", Name: "erigon", Version: "2.59.0", Commit: "0x1234"},
	}
	encoded := EncodeClientVersionsSSZ(versions)
	decoded, err := DecodeClientVersionsSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != len(versions) {
		t.Fatalf("length mismatch")
	}
	for i := range versions {
		if decoded[i].Code != versions[i].Code || decoded[i].Name != versions[i].Name {
			t.Errorf("version[%d] mismatch", i)
		}
	}
}

func TestExecutableDataSSZRoundTripV1(t *testing.T) {
	baseFee := big.NewInt(7)
	ed := &ExecutableData{
		ParentHash:    common.HexToHash("0x1111"),
		FeeRecipient:  common.HexToAddress("0x2222"),
		StateRoot:     common.HexToHash("0x3333"),
		ReceiptsRoot:  common.HexToHash("0x4444"),
		LogsBloom:     make([]byte, 256),
		Random:        common.HexToHash("0x5555"),
		Number:        100,
		GasLimit:      30000000,
		GasUsed:       21000,
		Timestamp:     1700000000,
		ExtraData:     []byte("hello"),
		BaseFeePerGas: baseFee,
		BlockHash:     common.HexToHash("0x6666"),
		Transactions:  [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}},
	}

	encoded := EncodeExecutableDataSSZ(ed, 1)
	decoded, err := DecodeExecutableDataSSZ(encoded, 1)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.ParentHash != ed.ParentHash {
		t.Errorf("ParentHash mismatch")
	}
	if decoded.Number != ed.Number {
		t.Errorf("Number mismatch: got %d, want %d", decoded.Number, ed.Number)
	}
	if decoded.BaseFeePerGas.Cmp(ed.BaseFeePerGas) != 0 {
		t.Errorf("BaseFeePerGas mismatch: got %v, want %v", decoded.BaseFeePerGas, ed.BaseFeePerGas)
	}
	if len(decoded.Transactions) != len(ed.Transactions) {
		t.Fatalf("Transactions length mismatch: got %d, want %d", len(decoded.Transactions), len(ed.Transactions))
	}
	for i := range ed.Transactions {
		if !bytes.Equal(decoded.Transactions[i], ed.Transactions[i]) {
			t.Errorf("Transaction[%d] mismatch", i)
		}
	}
	if !bytes.Equal(decoded.ExtraData, ed.ExtraData) {
		t.Errorf("ExtraData mismatch")
	}
}

func TestExecutableDataSSZRoundTripV3(t *testing.T) {
	baseFee := big.NewInt(1000000000)
	blobGasUsed := uint64(131072)
	excessBlobGas := uint64(262144)

	addr := common.HexToAddress("0xdead")
	ed := &ExecutableData{
		ParentHash:    common.HexToHash("0xaa"),
		FeeRecipient:  addr,
		StateRoot:     common.HexToHash("0xbb"),
		ReceiptsRoot:  common.HexToHash("0xcc"),
		LogsBloom:     make([]byte, 256),
		Random:        common.HexToHash("0xdd"),
		Number:        200,
		GasLimit:      30000000,
		GasUsed:       42000,
		Timestamp:     1700000001,
		ExtraData:     []byte{},
		BaseFeePerGas: baseFee,
		BlockHash:     common.HexToHash("0xee"),
		Transactions:  [][]byte{},
		Withdrawals: []*types.Withdrawal{
			{Index: 0, Validator: 1, Address: addr, Amount: 1000},
			{Index: 1, Validator: 2, Address: addr, Amount: 2000},
		},
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	encoded := EncodeExecutableDataSSZ(ed, 3)
	decoded, err := DecodeExecutableDataSSZ(encoded, 3)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.Number != ed.Number {
		t.Errorf("Number mismatch")
	}
	if *decoded.BlobGasUsed != *ed.BlobGasUsed {
		t.Errorf("BlobGasUsed mismatch")
	}
	if *decoded.ExcessBlobGas != *ed.ExcessBlobGas {
		t.Errorf("ExcessBlobGas mismatch")
	}
	if len(decoded.Withdrawals) != len(ed.Withdrawals) {
		t.Fatalf("Withdrawals length mismatch")
	}
	for i := range ed.Withdrawals {
		if decoded.Withdrawals[i].Index != ed.Withdrawals[i].Index {
			t.Errorf("Withdrawal[%d].Index mismatch", i)
		}
		if decoded.Withdrawals[i].Amount != ed.Withdrawals[i].Amount {
			t.Errorf("Withdrawal[%d].Amount mismatch", i)
		}
	}
}

func TestNewPayloadRequestSSZRoundTripV4(t *testing.T) {
	baseFee := big.NewInt(1000000000)
	blobGasUsed := uint64(131072)
	excessBlobGas := uint64(0)

	addr := common.HexToAddress("0xbeef")
	ep := &ExecutableData{
		ParentHash:    common.HexToHash("0x01"),
		FeeRecipient:  addr,
		StateRoot:     common.HexToHash("0x02"),
		ReceiptsRoot:  common.HexToHash("0x03"),
		LogsBloom:     make([]byte, 256),
		Random:        common.HexToHash("0x04"),
		Number:        300,
		GasLimit:      30000000,
		GasUsed:       63000,
		Timestamp:     1700000002,
		ExtraData:     []byte("test"),
		BaseFeePerGas: baseFee,
		BlockHash:     common.HexToHash("0x05"),
		Transactions:  [][]byte{{0xf8}},
		Withdrawals:   []*types.Withdrawal{},
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	blobHashes := []common.Hash{common.HexToHash("0xb10b")}
	beaconRoot := common.HexToHash("0xbeac")
	execRequests := [][]byte{
		{0x00, 0x01, 0x02}, // deposits
		{0x01, 0x03, 0x04}, // withdrawals
	}

	encoded := EncodeNewPayloadRequestSSZ(ep, blobHashes, &beaconRoot, execRequests, 4)
	decEp, decHashes, decRoot, decReqs, err := DecodeNewPayloadRequestSSZ(encoded, 4)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decEp.Number != ep.Number {
		t.Errorf("Number mismatch")
	}
	if len(decHashes) != 1 || decHashes[0] != blobHashes[0] {
		t.Errorf("blob hashes mismatch")
	}
	if *decRoot != beaconRoot {
		t.Errorf("beacon root mismatch")
	}
	// Structured requests: deposits and withdrawals should be present
	if len(decReqs) < 2 {
		t.Fatalf("expected at least 2 execution requests, got %d", len(decReqs))
	}
}

func TestGetBlobsRequestSSZRoundTrip(t *testing.T) {
	hashes := []common.Hash{
		common.HexToHash("0x1111"),
		common.HexToHash("0x2222"),
		common.HexToHash("0x3333"),
	}
	encoded := EncodeGetBlobsRequestSSZ(hashes)
	decoded, err := DecodeGetBlobsRequestSSZ(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != len(hashes) {
		t.Fatalf("length mismatch")
	}
	for i := range hashes {
		if decoded[i] != hashes[i] {
			t.Errorf("hash[%d] mismatch", i)
		}
	}
}

func TestUint256SSZRoundTrip(t *testing.T) {
	tests := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(1000000000),
		new(big.Int).SetBytes(common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")),
	}

	for _, val := range tests {
		encoded := uint256ToSSZBytes(val)
		decoded := sszBytesToUint256(encoded)
		if decoded.Cmp(val) != 0 {
			t.Errorf("uint256 roundtrip failed for %v: got %v", val, decoded)
		}
	}
}

func TestEngineStatusSSZConversion(t *testing.T) {
	statuses := []string{VALID, INVALID, SYNCING, ACCEPTED, "INVALID_BLOCK_HASH"}
	for _, s := range statuses {
		ssz := EngineStatusToSSZ(s)
		back := SSZToEngineStatus(ssz)
		if back != s {
			t.Errorf("status roundtrip failed: %s -> %d -> %s", s, ssz, back)
		}
	}
}
