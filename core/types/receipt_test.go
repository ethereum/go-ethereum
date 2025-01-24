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
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/kylelemons/godebug/diff"
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

	// Create a few transactions to have receipts for
	to2 = common.HexToAddress("0x2")
	to3 = common.HexToAddress("0x3")
	to4 = common.HexToAddress("0x4")
	to5 = common.HexToAddress("0x5")
	to6 = common.HexToAddress("0x6")
	to7 = common.HexToAddress("0x7")
	txs = Transactions{
		NewTx(&LegacyTx{
			Nonce:    1,
			Value:    big.NewInt(1),
			Gas:      1,
			GasPrice: big.NewInt(11),
		}),
		NewTx(&LegacyTx{
			To:       &to2,
			Nonce:    2,
			Value:    big.NewInt(2),
			Gas:      2,
			GasPrice: big.NewInt(22),
		}),
		NewTx(&AccessListTx{
			To:       &to3,
			Nonce:    3,
			Value:    big.NewInt(3),
			Gas:      3,
			GasPrice: big.NewInt(33),
		}),
		// EIP-1559 transactions.
		NewTx(&DynamicFeeTx{
			To:        &to4,
			Nonce:     4,
			Value:     big.NewInt(4),
			Gas:       4,
			GasTipCap: big.NewInt(44),
			GasFeeCap: big.NewInt(1044),
		}),
		NewTx(&DynamicFeeTx{
			To:        &to5,
			Nonce:     5,
			Value:     big.NewInt(5),
			Gas:       5,
			GasTipCap: big.NewInt(55),
			GasFeeCap: big.NewInt(1055),
		}),
	}

	blockNumber = big.NewInt(1)
	blockTime   = uint64(2)
	blockHash   = common.BytesToHash([]byte{0x03, 0x14})

	// Create the corresponding receipts
	receipts = Receipts{
		&Receipt{
			Status:            ReceiptStatusFailed,
			CumulativeGasUsed: 1,
			Logs: []*Log{
				{
					Address: common.BytesToAddress([]byte{0x11}),
					Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[0].Hash(),
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       0,
				},
				{
					Address: common.BytesToAddress([]byte{0x01, 0x11}),
					Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[0].Hash(),
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       1,
				},
			},
			// derived fields:
			TxHash:            txs[0].Hash(),
			ContractAddress:   common.HexToAddress("0x5a443704dd4b594b382c22a083e2bd3090a6fef3"),
			GasUsed:           1,
			EffectiveGasPrice: big.NewInt(11),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  0,
		},
		&Receipt{
			PostState:         common.Hash{2}.Bytes(),
			CumulativeGasUsed: 3,
			Logs: []*Log{
				{
					Address: common.BytesToAddress([]byte{0x22}),
					Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[1].Hash(),
					TxIndex:     1,
					BlockHash:   blockHash,
					Index:       2,
				},
				{
					Address: common.BytesToAddress([]byte{0x02, 0x22}),
					Topics:  []common.Hash{common.HexToHash("dead"), common.HexToHash("beef")},
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[1].Hash(),
					TxIndex:     1,
					BlockHash:   blockHash,
					Index:       3,
				},
			},
			// derived fields:
			TxHash:            txs[1].Hash(),
			GasUsed:           2,
			EffectiveGasPrice: big.NewInt(22),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  1,
		},
		&Receipt{
			Type:              AccessListTxType,
			PostState:         common.Hash{3}.Bytes(),
			CumulativeGasUsed: 6,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[2].Hash(),
			GasUsed:           3,
			EffectiveGasPrice: big.NewInt(33),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  2,
		},
		&Receipt{
			Type:              DynamicFeeTxType,
			PostState:         common.Hash{4}.Bytes(),
			CumulativeGasUsed: 10,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[3].Hash(),
			GasUsed:           4,
			EffectiveGasPrice: big.NewInt(1044),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  3,
		},
		&Receipt{
			Type:              DynamicFeeTxType,
			PostState:         common.Hash{5}.Bytes(),
			CumulativeGasUsed: 15,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[4].Hash(),
			GasUsed:           5,
			EffectiveGasPrice: big.NewInt(1055),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  4,
		},
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

// Test that we can marshal/unmarshal receipts to/from json without errors.
// This also confirms that our test receipts contain all the required fields.
func TestReceiptJSON(t *testing.T) {
	for i := range receipts {
		b, err := receipts[i].MarshalJSON()
		if err != nil {
			t.Fatal("error marshaling receipt to json:", err)
		}
		r := Receipt{}
		err = r.UnmarshalJSON(b)
		if err != nil {
			t.Fatal("error unmarshaling receipt from json:", err)
		}
	}
}

// Test we can still parse receipt without EffectiveGasPrice for backwards compatibility, even
// though it is required per the spec.
func TestEffectiveGasPriceNotRequired(t *testing.T) {
	r := *receipts[0]
	r.EffectiveGasPrice = nil
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatal("error marshaling receipt to json:", err)
	}
	r2 := Receipt{}
	err = r2.UnmarshalJSON(b)
	if err != nil {
		t.Fatal("error unmarshaling receipt from json:", err)
	}
}

func TestLegacyReceiptDecoding(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*Receipt) ([]byte, error)
	}{
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
	to4 := common.HexToAddress("0x4")
	to5 := common.HexToAddress("0x5")
	txs := Transactions{
		NewTx(&LegacyTx{
			Nonce:    1,
			Value:    big.NewInt(1),
			Gas:      1,
			GasPrice: big.NewInt(11),
		}),
		NewTx(&LegacyTx{
			To:       &to2,
			Nonce:    2,
			Value:    big.NewInt(2),
			Gas:      2,
			GasPrice: big.NewInt(22),
		}),
		NewTx(&AccessListTx{
			To:       &to3,
			Nonce:    3,
			Value:    big.NewInt(3),
			Gas:      3,
			GasPrice: big.NewInt(33),
		}),
		// EIP-1559 transactions.
		NewTx(&DynamicFeeTx{
			To:        &to4,
			Nonce:     4,
			Value:     big.NewInt(4),
			Gas:       4,
			GasTipCap: big.NewInt(44),
			GasFeeCap: big.NewInt(1045),
		}),
		NewTx(&DynamicFeeTx{
			To:        &to5,
			Nonce:     5,
			Value:     big.NewInt(5),
			Gas:       5,
			GasTipCap: big.NewInt(56),
			GasFeeCap: big.NewInt(1055),
		}),
	}

	blockNumber := big.NewInt(1)
	blockHash := common.BytesToHash([]byte{0x03, 0x14})

	// Create the corresponding receipts
	receipts := Receipts{
		&Receipt{
			Status:            ReceiptStatusFailed,
			CumulativeGasUsed: 1,
			Logs: []*Log{
				{
					Address: common.BytesToAddress([]byte{0x11}),
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[0].Hash(),
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       0,
				},
				{
					Address: common.BytesToAddress([]byte{0x01, 0x11}),
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[0].Hash(),
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       1,
				},
			},
			// derived fields:
			TxHash:            txs[0].Hash(),
			ContractAddress:   common.HexToAddress("0x5a443704dd4b594b382c22a083e2bd3090a6fef3"),
			GasUsed:           1,
			EffectiveGasPrice: big.NewInt(11),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  0,
		},
		&Receipt{
			PostState:         common.Hash{2}.Bytes(),
			CumulativeGasUsed: 3,
			Logs: []*Log{
				{
					Address: common.BytesToAddress([]byte{0x22}),
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[1].Hash(),
					TxIndex:     1,
					BlockHash:   blockHash,
					Index:       2,
				},
				{
					Address: common.BytesToAddress([]byte{0x02, 0x22}),
					// derived fields:
					BlockNumber: blockNumber.Uint64(),
					TxHash:      txs[1].Hash(),
					TxIndex:     1,
					BlockHash:   blockHash,
					Index:       3,
				},
			},
			// derived fields:
			TxHash:            txs[1].Hash(),
			GasUsed:           2,
			EffectiveGasPrice: big.NewInt(22),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  1,
		},
		&Receipt{
			Type:              AccessListTxType,
			PostState:         common.Hash{3}.Bytes(),
			CumulativeGasUsed: 6,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[2].Hash(),
			GasUsed:           3,
			EffectiveGasPrice: big.NewInt(33),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  2,
		},
		&Receipt{
			Type:              DynamicFeeTxType,
			PostState:         common.Hash{4}.Bytes(),
			CumulativeGasUsed: 10,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[3].Hash(),
			GasUsed:           4,
			EffectiveGasPrice: big.NewInt(1044),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  3,
		},
		&Receipt{
			Type:              DynamicFeeTxType,
			PostState:         common.Hash{5}.Bytes(),
			CumulativeGasUsed: 15,
			Logs:              []*Log{},
			// derived fields:
			TxHash:            txs[4].Hash(),
			GasUsed:           5,
			EffectiveGasPrice: big.NewInt(1055),
			BlockHash:         blockHash,
			BlockNumber:       blockNumber,
			TransactionIndex:  4,
		},
	}

	// Re-derive receipts.
	basefee := big.NewInt(1000)
	derivedReceipts := clearComputedFieldsOnReceipts(receipts)
	err := Receipts(derivedReceipts).DeriveFields(params.TestChainConfig, blockHash, blockNumber.Uint64(), basefee, txs)
	if err != nil {
		t.Fatalf("DeriveFields(...) = %v, want <nil>", err)
	}

	// Check diff of receipts against derivedReceipts.
	r1, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		t.Fatal("error marshaling input receipts:", err)
	}
	r2, err := json.MarshalIndent(derivedReceipts, "", "  ")
	if err != nil {
		t.Fatal("error marshaling derived receipts:", err)
	}
	d := diff.Diff(string(r1), string(r2))
	if d != "" {
		t.Fatal("receipts differ:", d)
	}
}

func clearComputedFieldsOnReceipts(receipts []*Receipt) []*Receipt {
	r := make([]*Receipt, len(receipts))
	for i, receipt := range receipts {
		r[i] = clearComputedFieldsOnReceipt(receipt)
	}
	return r
}

func clearComputedFieldsOnReceipt(receipt *Receipt) *Receipt {
	cpy := *receipt
	cpy.TxHash = common.Hash{0xff, 0xff, 0x11}
	cpy.BlockHash = common.Hash{0xff, 0xff, 0x22}
	cpy.BlockNumber = big.NewInt(math.MaxUint32)
	cpy.TransactionIndex = math.MaxUint32
	cpy.ContractAddress = common.Address{0xff, 0xff, 0x33}
	cpy.GasUsed = 0xffffffff
	cpy.Logs = clearComputedFieldsOnLogs(receipt.Logs)
	return &cpy
}

func clearComputedFieldsOnLogs(logs []*Log) []*Log {
	l := make([]*Log, len(logs))
	for i, log := range logs {
		cpy := *log
		cpy.BlockNumber = math.MaxUint32
		cpy.BlockHash = common.Hash{}
		cpy.TxHash = common.Hash{}
		cpy.TxIndex = math.MaxUint32
		cpy.Index = math.MaxUint32
		l[i] = &cpy
	}
	return l
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
