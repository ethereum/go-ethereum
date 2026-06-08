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

func TestExecutionPayloadAmsterdamRoundtrip(t *testing.T) {
	p := &ExecutionPayloadAmsterdam{
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
		BlobGasUsed:     131072,
		ExcessBlobGas:   0,
		BlockAccessList: []byte{0xde, 0xad, 0xbe, 0xef},
		SlotNumber:      200,
	}
	encDec(t, p, func() *ExecutionPayloadAmsterdam { return new(ExecutionPayloadAmsterdam) })
}

func TestForkchoiceUpdateAmsterdamRoundtrip(t *testing.T) {
	// With payload_attributes and custody_columns present.
	bits := &Bitvector128{Bytes: make([]byte, CellsPerExtBlob/8)}
	bits.Bytes[0] = 0x01
	attrs := &PayloadAttributesAmsterdam{
		Timestamp:             1700000000,
		PrevRandao:            common.Hash{0x55},
		SuggestedFeeRecipient: common.Address{0x66},
		Withdrawals:           []*Withdrawal{},
		ParentBeaconBlockRoot: common.Hash{0x77},
		SlotNumber:            42,
		TargetGasLimit:        30000000,
	}
	fcu := &ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &ForkchoiceState{
			HeadBlockHash:      common.Hash{0xaa},
			SafeBlockHash:      common.Hash{0xbb},
			FinalizedBlockHash: common.Hash{0xcc},
		},
		PayloadAttributes: []*PayloadAttributesAmsterdam{attrs},
		CustodyColumns:    []*Bitvector128{bits},
	}
	encDec(t, fcu, func() *ForkchoiceUpdateAmsterdam { return new(ForkchoiceUpdateAmsterdam) })

	// Without optionals.
	bare := &ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &ForkchoiceState{
			HeadBlockHash: common.Hash{0xdd},
		},
	}
	encDec(t, bare, func() *ForkchoiceUpdateAmsterdam { return new(ForkchoiceUpdateAmsterdam) })
}
