// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	legacyReceipt = &Receipt{
		Status:            ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
	}
	accessListReceipt = &Receipt{
		Status:            ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: AccessListTxType,
	}
	eip1559Receipt = &Receipt{
		Status:            ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: DynamicFeeTxType,
	}
)

func TestDecodeEmptyTypedReceipt(t *testing.T) {
	input := []byte{0x80}
	var r Receipt
	err := rlp.DecodeBytes(input, &r)
	if err != errShortTypedReceipt {
		t.Fatal("wrong error:", err)
	}
}

func TestLegacyReceiptDecoding(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*Receipt) ([]byte, error)
	}{
		{
			"ReceiptForStorage",
			encodeAsReceiptForStorage,
		},
		{
			"StoredReceiptRLP",
			encodeAsStoredReceiptRLP,
		},
		{
			"V4StoredReceiptRLP",
			encodeAsV4StoredReceiptRLP,
		},
		{
			"V3StoredReceiptRLP",
			encodeAsV3StoredReceiptRLP,
		},
	}

	tx := NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), 1, big.NewInt(1), nil)
	receipt := &Receipt{
		Status:            ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		TxHash:          tx.Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}
	receipt.Bloom = CreateBloom(Receipts{receipt})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := tc.encode(receipt)
			if err != nil {
				t.Fatalf("Error encoding receipt: %v", err)
			}
			var dec ReceiptForStorage
			if err := rlp.DecodeBytes(enc, &dec); err != nil {
				t.Fatalf("Error decoding RLP receipt: %v", err)
			}
			// Check whether all consensus fields are correct.
			if dec.Status != receipt.Status {
				t.Fatalf("Receipt status mismatch, want %v, have %v", receipt.Status, dec.Status)
			}
			if dec.CumulativeGasUsed != receipt.CumulativeGasUsed {
				t.Fatalf("Receipt CumulativeGasUsed mismatch, want %v, have %v", receipt.CumulativeGasUsed, dec.CumulativeGasUsed)
			}
			if dec.Bloom != receipt.Bloom {
				t.Fatalf("Bloom data mismatch, want %v, have %v", receipt.Bloom, dec.Bloom)
			}
			if len(dec.Logs) != len(receipt.Logs) {
				t.Fatalf("Receipt log number mismatch, want %v, have %v", len(receipt.Logs), len(dec.Logs))
			}
			for i := 0; i < len(dec.Logs); i++ {
				if dec.Logs[i].Address != receipt.Logs[i].Address {
					t.Fatalf("Receipt log %d address mismatch, want %v, have %v", i, receipt.Logs[i].Address, dec.Logs[i].Address)
				}
				if !reflect.DeepEqual(dec.Logs[i].Topics, receipt.Logs[i].Topics) {
					t.Fatalf("Receipt log %d topics mismatch, want %v, have %v", i, receipt.Logs[i].Topics, dec.Logs[i].Topics)
				}
				if !bytes.Equal(dec.Logs[i].Data, receipt.Logs[i].Data) {
					t.Fatalf("Receipt log %d data mismatch, want %v, have %v", i, receipt.Logs[i].Data, dec.Logs[i].Data)
				}
			}
		})
	}
}

func encodeAsReceiptForStorage(want *Receipt) ([]byte, error) {
	return rlp.EncodeToBytes((*ReceiptForStorage)(want))
}

func encodeAsStoredReceiptRLP(want *Receipt) ([]byte, error) {
	stored := &storedReceiptRLP{
		PostStateOrStatus: want.statusEncoding(),
		CumulativeGasUsed: want.CumulativeGasUsed,
		Logs:              make([]*LogForStorage, len(want.Logs)),
	}
	for i, log := range want.Logs {
		stored.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.EncodeToBytes(stored)
}

func encodeAsV4StoredReceiptRLP(want *Receipt) ([]byte, error) {
	stored := &v4StoredReceiptRLP{
		PostStateOrStatus: want.statusEncoding(),
		CumulativeGasUsed: want.CumulativeGasUsed,
		TxHash:            want.TxHash,
		ContractAddress:   want.ContractAddress,
		Logs:              make([]*LogForStorage, len(want.Logs)),
		GasUsed:           want.GasUsed,
	}
	for i, log := range want.Logs {
		stored.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.EncodeToBytes(stored)
}

func encodeAsV3StoredReceiptRLP(want *Receipt) ([]byte, error) {
	stored := &v3StoredReceiptRLP{
		PostStateOrStatus: want.statusEncoding(),
		CumulativeGasUsed: want.CumulativeGasUsed,
		Bloom:             want.Bloom,
		TxHash:            want.TxHash,
		ContractAddress:   want.ContractAddress,
		Logs:              make([]*LogForStorage, len(want.Logs)),
		GasUsed:           want.GasUsed,
	}
	for i, log := range want.Logs {
		stored.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.EncodeToBytes(stored)
}

// Tests that receipt data can be correctly derived from the contextual infos
func TestDeriveFields(t *testing.T) {
	// Create a few transactions to have receipts for
	to2 := common.HexToAddress("0x2")
	to3 := common.HexToAddress("0x3")
	txs := Transactions{
		NewTx(&LegacyTx{
			Nonce:    1,
			Value:    big.NewInt(1),
			Gas:      1,
			GasPrice: big.NewInt(1),
		}),
		NewTx(&LegacyTx{
			To:       &to2,
			Nonce:    2,
			Value:    big.NewInt(2),
			Gas:      2,
			GasPrice: big.NewInt(2),
		}),
		NewTx(&AccessListTx{
			To:       &to3,
			Nonce:    3,
			Value:    big.NewInt(3),
			Gas:      3,
			GasPrice: big.NewInt(3),
		}),
	}
	// Create the corresponding receipts
	receipts := Receipts{
		&Receipt{
			Status:            ReceiptStatusFailed,
			CumulativeGasUsed: 1,
			Logs: []*Log{
				{Address: common.BytesToAddress([]byte{0x11})},
				{Address: common.BytesToAddress([]byte{0x01, 0x11})},
			},
			TxHash:          txs[0].Hash(),
			ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
			GasUsed:         1,
		},
		&Receipt{
			PostState:         common.Hash{2}.Bytes(),
			CumulativeGasUsed: 3,
			Logs: []*Log{
				{Address: common.BytesToAddress([]byte{0x22})},
				{Address: common.BytesToAddress([]byte{0x02, 0x22})},
			},
			TxHash:          txs[1].Hash(),
			ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
			GasUsed:         2,
		},
		&Receipt{
			Type:              AccessListTxType,
			PostState:         common.Hash{3}.Bytes(),
			CumulativeGasUsed: 6,
			Logs: []*Log{
				{Address: common.BytesToAddress([]byte{0x33})},
				{Address: common.BytesToAddress([]byte{0x03, 0x33})},
			},
			TxHash:          txs[2].Hash(),
			ContractAddress: common.BytesToAddress([]byte{0x03, 0x33, 0x33}),
			GasUsed:         3,
		},
	}
	// Clear all the computed fields and re-derive them
	number := big.NewInt(1)
	hash := common.BytesToHash([]byte{0x03, 0x14})

	clearComputedFieldsOnReceipts(t, receipts)
	if err := receipts.DeriveFields(params.TestChainConfig, hash, number.Uint64(), txs); err != nil {
		t.Fatalf("DeriveFields(...) = %v, want <nil>", err)
	}
	// Iterate over all the computed fields and check that they're correct
	signer := MakeSigner(params.TestChainConfig, number)

	logIndex := uint(0)
	for i := range receipts {
		if receipts[i].Type != txs[i].Type() {
			t.Errorf("receipts[%d].Type = %d, want %d", i, receipts[i].Type, txs[i].Type())
		}
		if receipts[i].TxHash != txs[i].Hash() {
			t.Errorf("receipts[%d].TxHash = %s, want %s", i, receipts[i].TxHash.String(), txs[i].Hash().String())
		}
		if receipts[i].BlockHash != hash {
			t.Errorf("receipts[%d].BlockHash = %s, want %s", i, receipts[i].BlockHash.String(), hash.String())
		}
		if receipts[i].BlockNumber.Cmp(number) != 0 {
			t.Errorf("receipts[%c].BlockNumber = %s, want %s", i, receipts[i].BlockNumber.String(), number.String())
		}
		if receipts[i].TransactionIndex != uint(i) {
			t.Errorf("receipts[%d].TransactionIndex = %d, want %d", i, receipts[i].TransactionIndex, i)
		}
		if receipts[i].GasUsed != txs[i].Gas() {
			t.Errorf("receipts[%d].GasUsed = %d, want %d", i, receipts[i].GasUsed, txs[i].Gas())
		}
		if txs[i].To() != nil && receipts[i].ContractAddress != (common.Address{}) {
			t.Errorf("receipts[%d].ContractAddress = %s, want %s", i, receipts[i].ContractAddress.String(), (common.Address{}).String())
		}
		from, _ := Sender(signer, txs[i])
		contractAddress := crypto.CreateAddress(from, txs[i].Nonce())
		if txs[i].To() == nil && receipts[i].ContractAddress != contractAddress {
			t.Errorf("receipts[%d].ContractAddress = %s, want %s", i, receipts[i].ContractAddress.String(), contractAddress.String())
		}
		for j := range receipts[i].Logs {
			if receipts[i].Logs[j].BlockNumber != number.Uint64() {
				t.Errorf("receipts[%d].Logs[%d].BlockNumber = %d, want %d", i, j, receipts[i].Logs[j].BlockNumber, number.Uint64())
			}
			if receipts[i].Logs[j].BlockHash != hash {
				t.Errorf("receipts[%d].Logs[%d].BlockHash = %s, want %s", i, j, receipts[i].Logs[j].BlockHash.String(), hash.String())
			}
			if receipts[i].Logs[j].TxHash != txs[i].Hash() {
				t.Errorf("receipts[%d].Logs[%d].TxHash = %s, want %s", i, j, receipts[i].Logs[j].TxHash.String(), txs[i].Hash().String())
			}
			if receipts[i].Logs[j].TxIndex != uint(i) {
				t.Errorf("receipts[%d].Logs[%d].TransactionIndex = %d, want %d", i, j, receipts[i].Logs[j].TxIndex, i)
			}
			if receipts[i].Logs[j].Index != logIndex {
				t.Errorf("receipts[%d].Logs[%d].Index = %d, want %d", i, j, receipts[i].Logs[j].Index, logIndex)
			}
			logIndex++
		}
	}
}

// TestTypedReceiptEncodingDecoding reproduces a flaw that existed in the receipt
// rlp decoder, which failed due to a shadowing error.
func TestTypedReceiptEncodingDecoding(t *testing.T) {
	var payload = common.FromHex("f9043eb9010c01f90108018262d4b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0b9010c01f901080182cd14b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0b9010d01f901090183013754b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0b9010d01f90109018301a194b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0")
	check := func(bundle []*Receipt) {
		t.Helper()
		for i, receipt := range bundle {
			if got, want := receipt.Type, uint8(1); got != want {
				t.Fatalf("bundle %d: got %x, want %x", i, got, want)
			}
		}
	}
	{
		var bundle []*Receipt
		rlp.DecodeBytes(payload, &bundle)
		check(bundle)
	}
	{
		var bundle []*Receipt
		r := bytes.NewReader(payload)
		s := rlp.NewStream(r, uint64(len(payload)))
		if err := s.Decode(&bundle); err != nil {
			t.Fatal(err)
		}
		check(bundle)
	}
}

func TestReceiptMarshalBinary(t *testing.T) {
	// Legacy Receipt
	legacyReceipt.Bloom = CreateBloom(Receipts{legacyReceipt})
	have, err := legacyReceipt.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal binary error: %v", err)
	}
	legacyReceipts := Receipts{legacyReceipt}
	buf := new(bytes.Buffer)
	legacyReceipts.EncodeIndex(0, buf)
	haveEncodeIndex := buf.Bytes()
	if !bytes.Equal(have, haveEncodeIndex) {
		t.Errorf("BinaryMarshal and EncodeIndex mismatch, got %x want %x", have, haveEncodeIndex)
	}
	buf.Reset()
	if err := legacyReceipt.EncodeRLP(buf); err != nil {
		t.Fatalf("encode rlp error: %v", err)
	}
	haveRLPEncode := buf.Bytes()
	if !bytes.Equal(have, haveRLPEncode) {
		t.Errorf("BinaryMarshal and EncodeRLP mismatch for legacy tx, got %x want %x", have, haveRLPEncode)
	}
	legacyWant := common.FromHex("f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	if !bytes.Equal(have, legacyWant) {
		t.Errorf("encoded RLP mismatch, got %x want %x", have, legacyWant)
	}

	// 2930 Receipt
	buf.Reset()
	accessListReceipt.Bloom = CreateBloom(Receipts{accessListReceipt})
	have, err = accessListReceipt.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal binary error: %v", err)
	}
	accessListReceipts := Receipts{accessListReceipt}
	accessListReceipts.EncodeIndex(0, buf)
	haveEncodeIndex = buf.Bytes()
	if !bytes.Equal(have, haveEncodeIndex) {
		t.Errorf("BinaryMarshal and EncodeIndex mismatch, got %x want %x", have, haveEncodeIndex)
	}
	accessListWant := common.FromHex("01f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	if !bytes.Equal(have, accessListWant) {
		t.Errorf("encoded RLP mismatch, got %x want %x", have, accessListWant)
	}

	// 1559 Receipt
	buf.Reset()
	eip1559Receipt.Bloom = CreateBloom(Receipts{eip1559Receipt})
	have, err = eip1559Receipt.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal binary error: %v", err)
	}
	eip1559Receipts := Receipts{eip1559Receipt}
	eip1559Receipts.EncodeIndex(0, buf)
	haveEncodeIndex = buf.Bytes()
	if !bytes.Equal(have, haveEncodeIndex) {
		t.Errorf("BinaryMarshal and EncodeIndex mismatch, got %x want %x", have, haveEncodeIndex)
	}
	eip1559Want := common.FromHex("02f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	if !bytes.Equal(have, eip1559Want) {
		t.Errorf("encoded RLP mismatch, got %x want %x", have, eip1559Want)
	}
}

func TestReceiptUnmarshalBinary(t *testing.T) {
	// Legacy Receipt
	legacyBinary := common.FromHex("f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	gotLegacyReceipt := new(Receipt)
	if err := gotLegacyReceipt.UnmarshalBinary(legacyBinary); err != nil {
		t.Fatalf("unmarshal binary error: %v", err)
	}
	legacyReceipt.Bloom = CreateBloom(Receipts{legacyReceipt})
	if !reflect.DeepEqual(gotLegacyReceipt, legacyReceipt) {
		t.Errorf("receipt unmarshalled from binary mismatch, got %v want %v", gotLegacyReceipt, legacyReceipt)
	}

	// 2930 Receipt
	accessListBinary := common.FromHex("01f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	gotAccessListReceipt := new(Receipt)
	if err := gotAccessListReceipt.UnmarshalBinary(accessListBinary); err != nil {
		t.Fatalf("unmarshal binary error: %v", err)
	}
	accessListReceipt.Bloom = CreateBloom(Receipts{accessListReceipt})
	if !reflect.DeepEqual(gotAccessListReceipt, accessListReceipt) {
		t.Errorf("receipt unmarshalled from binary mismatch, got %v want %v", gotAccessListReceipt, accessListReceipt)
	}

	// 1559 Receipt
	eip1559RctBinary := common.FromHex("02f901c58001b9010000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000010000080000000000000000000004000000000000000000000000000040000000000000000000000000000800000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000f8bef85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100fff85d940000000000000000000000000000000000000111f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff")
	got1559Receipt := new(Receipt)
	if err := got1559Receipt.UnmarshalBinary(eip1559RctBinary); err != nil {
		t.Fatalf("unmarshal binary error: %v", err)
	}
	eip1559Receipt.Bloom = CreateBloom(Receipts{eip1559Receipt})
	if !reflect.DeepEqual(got1559Receipt, eip1559Receipt) {
		t.Errorf("receipt unmarshalled from binary mismatch, got %v want %v", got1559Receipt, eip1559Receipt)
	}
}

func clearComputedFieldsOnReceipts(t *testing.T, receipts Receipts) {
	t.Helper()

	for _, receipt := range receipts {
		clearComputedFieldsOnReceipt(t, receipt)
	}
}

func clearComputedFieldsOnReceipt(t *testing.T, receipt *Receipt) {
	t.Helper()

	receipt.TxHash = common.Hash{}
	receipt.BlockHash = common.Hash{}
	receipt.BlockNumber = big.NewInt(math.MaxUint32)
	receipt.TransactionIndex = math.MaxUint32
	receipt.ContractAddress = common.Address{}
	receipt.GasUsed = 0

	clearComputedFieldsOnLogs(t, receipt.Logs)
}

func clearComputedFieldsOnLogs(t *testing.T, logs []*Log) {
	t.Helper()

	for _, log := range logs {
		clearComputedFieldsOnLog(t, log)
	}
}

func clearComputedFieldsOnLog(t *testing.T, log *Log) {
	t.Helper()

	log.BlockNumber = math.MaxUint32
	log.BlockHash = common.Hash{}
	log.TxHash = common.Hash{}
	log.TxIndex = math.MaxUint32
	log.Index = math.MaxUint32
}
