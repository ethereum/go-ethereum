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

package ssz

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

func encDec[T ssz.Object](t *testing.T, obj T, newEmpty func() T) {
	t.Helper()
	size := ssz.Size(obj)
	buf := make([]byte, size)
	if err := ssz.EncodeToBytes(buf, obj); err != nil {
		t.Fatalf("encode: %v", err)
	}
	got := newEmpty()
	if err := ssz.DecodeFromBytes(buf, got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// SSZ decode of an empty list yields []T{}, while encoding either nil
	// or []T{} produces the same bytes. Compare via re-encode rather than
	// DeepEqual to avoid spurious nil-vs-empty failures.
	rebuf := make([]byte, ssz.Size(got))
	if err := ssz.EncodeToBytes(rebuf, got); err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if !reflect.DeepEqual(buf, rebuf) {
		t.Fatalf("round-trip mismatch\nwant bytes: %x\n got bytes: %x", buf, rebuf)
	}
}

func TestWithdrawalRoundtrip(t *testing.T) {
	encDec(t, &Withdrawal{
		Index:          7,
		ValidatorIndex: 42,
		Address:        common.Address{0xaa, 0xbb},
		Amount:         123456,
	}, func() *Withdrawal { return new(Withdrawal) })
}

func TestForkchoiceStateRoundtrip(t *testing.T) {
	encDec(t, &ForkchoiceState{
		HeadBlockHash:      common.Hash{0x01},
		SafeBlockHash:      common.Hash{0x02},
		FinalizedBlockHash: common.Hash{0x03},
	}, func() *ForkchoiceState { return new(ForkchoiceState) })
}

func TestPayloadStatusRoundtrip(t *testing.T) {
	h := common.Hash{0xab}
	encDec(t, &PayloadStatus{
		Status:          StatusValid,
		LatestValidHash: []common.Hash{h},
		ValidationError: [][]byte{[]byte("err detail")},
	}, func() *PayloadStatus { return new(PayloadStatus) })

	// Absent optionals.
	encDec(t, &PayloadStatus{Status: StatusSyncing}, func() *PayloadStatus { return new(PayloadStatus) })
}

// encDecOnFork is the fork-aware sibling of encDec for monolith types.
func encDecOnFork[T ssz.Object](t *testing.T, obj T, fork ssz.Fork, newEmpty func() T) {
	t.Helper()
	size := ssz.SizeOnFork(obj, fork)
	buf := make([]byte, size)
	if err := ssz.EncodeToBytesOnFork(buf, obj, fork); err != nil {
		t.Fatalf("encode: %v", err)
	}
	got := newEmpty()
	if err := ssz.DecodeFromBytesOnFork(buf, got, fork); err != nil {
		t.Fatalf("decode: %v", err)
	}
	rebuf := make([]byte, ssz.SizeOnFork(got, fork))
	if err := ssz.EncodeToBytesOnFork(rebuf, got, fork); err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if !reflect.DeepEqual(buf, rebuf) {
		t.Fatalf("round-trip mismatch\nwant bytes: %x\n got bytes: %x", buf, rebuf)
	}
}

func TestExecutionPayloadAmsterdamRoundtrip(t *testing.T) {
	blob, excess, slot := uint64(131072), uint64(0), uint64(200)
	p := &ExecutionPayload{
		ParentHash:      common.Hash{0x11},
		FeeRecipient:    common.Address{0x22},
		StateRoot:       common.Hash{0x33},
		ReceiptsRoot:    common.Hash{0x44},
		PrevRandao:      common.Hash{0x55},
		BlockNumber:     100,
		GasLimit:        30000000,
		GasUsed:         21000,
		Timestamp:       1700000000,
		ExtraData:       []byte("ext"),
		BaseFeePerGas:   uint256.NewInt(7e9),
		BlockHash:       common.Hash{0x66},
		Transactions:    [][]byte{{0x02, 0x03}, {0x04}},
		Withdrawals:     []*Withdrawal{{Index: 1, ValidatorIndex: 2, Amount: 3}},
		BlobGasUsed:     &blob,
		ExcessBlobGas:   &excess,
		BlockAccessList: []byte{0xde, 0xad, 0xbe, 0xef},
		SlotNumber:      &slot,
	}
	encDecOnFork(t, p, forkAmsterdam, func() *ExecutionPayload { return new(ExecutionPayload) })
}

func TestForkchoiceUpdateAmsterdamRoundtrip(t *testing.T) {
	// With payload_attributes and custody_columns present.
	bits := &Bitvector128{Bytes: make([]byte, CellsPerExtBlob/8)}
	bits.Bytes[0] = 0x01
	root := common.Hash{0x77}
	slot, tgl := uint64(42), uint64(30000000)
	attrs := &PayloadAttributes{
		Timestamp:             1700000000,
		PrevRandao:            common.Hash{0x55},
		SuggestedFeeRecipient: common.Address{0x66},
		Withdrawals:           []*Withdrawal{},
		ParentBeaconBlockRoot: &root,
		SlotNumber:            &slot,
		TargetGasLimit:        &tgl,
	}
	fcu := &ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &ForkchoiceState{
			HeadBlockHash:      common.Hash{0xaa},
			SafeBlockHash:      common.Hash{0xbb},
			FinalizedBlockHash: common.Hash{0xcc},
		},
		PayloadAttributes: []*PayloadAttributes{attrs},
		CustodyColumns:    []*Bitvector128{bits},
	}
	encDecOnFork(t, fcu, forkAmsterdam, func() *ForkchoiceUpdateAmsterdam { return new(ForkchoiceUpdateAmsterdam) })

	// Without optionals.
	bare := &ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &ForkchoiceState{
			HeadBlockHash: common.Hash{0xdd},
		},
	}
	encDecOnFork(t, bare, forkAmsterdam, func() *ForkchoiceUpdateAmsterdam { return new(ForkchoiceUpdateAmsterdam) })
}

// minimalPayload builds an ExecutionPayload carrying exactly the fields the
// given fork's wire shape requires (BaseFeePerGas is always present; blob-gas
// pointers from Cancun; slot from Amsterdam), so it round-trips cleanly at fork.
func minimalPayload(fork ssz.Fork) *ExecutionPayload {
	p := &ExecutionPayload{BaseFeePerGas: uint256.NewInt(7e9)}
	if fork >= ssz.ForkDencun {
		blob, excess := uint64(0), uint64(0)
		p.BlobGasUsed, p.ExcessBlobGas = &blob, &excess
	}
	if fork >= forkAmsterdam {
		slot := uint64(0)
		p.SlotNumber = &slot
	}
	return p
}

// allEngineForks enumerates the codec forks the Engine API v2 spec covers
// (Paris onward), paired with the spec's fixed-part sizes for the envelope and
// built-payload wrappers.
var allEngineForks = []struct {
	name            string
	fork            ssz.Fork
	envelopeFixed   uint32 // ExecutionPayloadEnvelope fixed part
	builtPayloadFix uint32 // BuiltPayload fixed part
	hasBeaconRoot   bool   // Cancun+
	hasRequests     bool   // Prague+
	hasBundle       bool   // Cancun+
	bundleIsV2      bool   // Osaka+
}{
	// envelope fixed: offset(payload)=4 [+root 32 from Cancun] [+offset(requests) 4 from Prague]
	// built fixed:    offset(payload)=4 + value 32 [+offset(bundle) 4 + override 1 from Cancun] [+offset(requests) 4 from Prague]
	{"paris", ssz.ForkParis, 4, 36, false, false, false, false},
	{"shanghai", ssz.ForkShapella, 4, 36, false, false, false, false},
	{"cancun", ssz.ForkDencun, 36, 41, true, false, true, false},
	{"prague", ssz.ForkPectra, 40, 45, true, true, true, false},
	{"osaka", forkOsaka, 40, 45, true, true, true, true},
	{"amsterdam", forkAmsterdam, 40, 45, true, true, true, true},
}

// TestExecutionPayloadEnvelopePerFork verifies the envelope wire shape matches
// the per-fork ExecutionPayloadEnvelope catalogue in refactor-ssz.md for every
// fork the spec covers: bare payload for Paris/Shanghai, +parent_beacon_block_root
// from Cancun, +execution_requests from Prague.
func TestExecutionPayloadEnvelopePerFork(t *testing.T) {
	for _, tc := range allEngineForks {
		t.Run(tc.name, func(t *testing.T) {
			env := &ExecutionPayloadEnvelopeAmsterdam{Payload: minimalPayload(tc.fork)}
			if tc.hasBeaconRoot {
				env.ParentBeaconBlockRoot = &common.Hash{0x55}
			}
			if tc.hasRequests {
				env.ExecutionRequests = [][]byte{{0x01, 0x02}}
			}
			encDecOnFork(t, env, tc.fork, func() *ExecutionPayloadEnvelopeAmsterdam {
				return new(ExecutionPayloadEnvelopeAmsterdam)
			})

			// Pre-Cancun must not carry a beacon root; pre-Prague must not carry
			// requests — the codec drops them, so a decoded copy stays empty.
			buf := make([]byte, ssz.SizeOnFork(env, tc.fork))
			if err := ssz.EncodeToBytesOnFork(buf, env, tc.fork); err != nil {
				t.Fatalf("encode: %v", err)
			}
			// payload is the first (dynamic) field, so its offset equals the
			// fixed-part length — the spec's per-fork envelope size.
			if off := readOffset(buf); off != tc.envelopeFixed {
				t.Errorf("envelope fixed size = %d, want %d", off, tc.envelopeFixed)
			}
			got := new(ExecutionPayloadEnvelopeAmsterdam)
			if err := ssz.DecodeFromBytesOnFork(buf, got, tc.fork); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !tc.hasBeaconRoot && got.ParentBeaconBlockRoot != nil {
				t.Errorf("%s: beacon root decoded but fork predates Cancun", tc.name)
			}
			if tc.hasBeaconRoot && got.ParentBeaconBlockRoot == nil {
				t.Errorf("%s: beacon root missing after roundtrip", tc.name)
			}
			if !tc.hasRequests && len(got.ExecutionRequests) > 0 {
				t.Errorf("%s: requests decoded but fork predates Prague", tc.name)
			}
		})
	}
}

// TestBuiltPayloadPerFork verifies the BuiltPayload wire shape matches the
// per-fork BuiltPayload catalogue in refactor-ssz.md: {payload, block_value}
// for Paris/Shanghai, +blobs_bundle(V1)+should_override_builder from Cancun,
// +execution_requests from Prague, blobs_bundle→V2 from Osaka.
func TestBuiltPayloadPerFork(t *testing.T) {
	for _, tc := range allEngineForks {
		t.Run(tc.name, func(t *testing.T) {
			bp := &BuiltPayloadAmsterdam{
				Payload:    minimalPayload(tc.fork),
				BlockValue: uint256.NewInt(1000),
			}
			if tc.hasBundle {
				bundle := func() ([][48]byte, [][48]byte, []*Blob) {
					return [][48]byte{{0x01}}, [][48]byte{{0x02}}, []*Blob{{Bytes: make([]byte, BytesPerBlob)}}
				}
				c, p, b := bundle()
				if tc.bundleIsV2 {
					bp.BlobsBundleV2 = &BlobsBundleV2{Commitments: c, Proofs: p, Blobs: b}
				} else {
					bp.BlobsBundleV1 = &BlobsBundleV1{Commitments: c, Proofs: p, Blobs: b}
				}
				override := true
				bp.ShouldOverrideBuilder = &override
			}
			if tc.hasRequests {
				bp.ExecutionRequests = [][]byte{{0xaa}}
			}
			encDecOnFork(t, bp, tc.fork, func() *BuiltPayloadAmsterdam { return new(BuiltPayloadAmsterdam) })

			buf := make([]byte, ssz.SizeOnFork(bp, tc.fork))
			if err := ssz.EncodeToBytesOnFork(buf, bp, tc.fork); err != nil {
				t.Fatalf("encode: %v", err)
			}
			// payload is the first (dynamic) field, so its offset equals the
			// fixed-part length — the spec's per-fork built-payload size.
			if off := readOffset(buf); off != tc.builtPayloadFix {
				t.Errorf("built-payload fixed size = %d, want %d", off, tc.builtPayloadFix)
			}
			got := new(BuiltPayloadAmsterdam)
			if err := ssz.DecodeFromBytesOnFork(buf, got, tc.fork); err != nil {
				t.Fatalf("decode: %v", err)
			}
			// Exactly the right bundle revision is populated (or neither pre-Cancun).
			if tc.bundleIsV2 && got.BlobsBundleV1 != nil {
				t.Errorf("%s: V1 bundle decoded for an Osaka+ fork", tc.name)
			}
			if tc.hasBundle && !tc.bundleIsV2 && got.BlobsBundleV2 != nil {
				t.Errorf("%s: V2 bundle decoded for a pre-Osaka fork", tc.name)
			}
			if !tc.hasBundle && (got.BlobsBundleV1 != nil || got.BlobsBundleV2 != nil) {
				t.Errorf("%s: bundle decoded but fork predates Cancun", tc.name)
			}
			if !tc.hasBundle && got.ShouldOverrideBuilder != nil {
				t.Errorf("%s: should_override_builder decoded but fork predates Cancun", tc.name)
			}
			if !tc.hasRequests && len(got.ExecutionRequests) > 0 {
				t.Errorf("%s: requests decoded but fork predates Prague", tc.name)
			}
		})
	}
}

// readOffset reads the first 4-byte little-endian SSZ offset from buf. For a
// container whose first field is dynamic, this offset equals the length of the
// fixed part (where the variable region begins).
func readOffset(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf[:4])
}
