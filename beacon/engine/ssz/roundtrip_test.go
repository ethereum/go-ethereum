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
