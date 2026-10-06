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

package eth

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// miniDeriveFields derives the necessary receipt fields to make types.DeriveSha work.
func miniDeriveFields(r *types.Receipt, txType byte) {
	r.Type = txType
	r.Bloom = types.CreateBloom(r)
}

var receiptsTestLogs1 = []*types.Log{{Address: common.Address{1}, Topics: []common.Hash{{1}}}}
var receiptsTestLogs2 = []*types.Log{
	{Address: common.Address{2}, Topics: []common.Hash{{21}, {22}}, Data: []byte{2, 2, 32, 32}},
	{Address: common.Address{3}, Topics: []common.Hash{{31}, {32}}, Data: []byte{3, 3, 32, 32}},
}

var receiptsTestPayer = common.Address{0xaa}

var receiptsTests = []struct {
	input []types.ReceiptForStorage
	txs   []*types.Transaction
	root  common.Hash
}{
	{
		input: []types.ReceiptForStorage{{CumulativeGasUsed: 555, Status: 1, Logs: nil}},
		txs:   []*types.Transaction{types.NewTx(&types.LegacyTx{})},
	},
	{
		input: []types.ReceiptForStorage{{CumulativeGasUsed: 555, Status: 1, Logs: nil}},
		txs:   []*types.Transaction{types.NewTx(&types.DynamicFeeTx{})},
	},
	{
		input: []types.ReceiptForStorage{{CumulativeGasUsed: 555, Status: 1, Logs: nil}},
		txs:   []*types.Transaction{types.NewTx(&types.AccessListTx{})},
	},
	{
		input: []types.ReceiptForStorage{{CumulativeGasUsed: 555, Status: 1, Logs: receiptsTestLogs1}},
		txs:   []*types.Transaction{types.NewTx(&types.LegacyTx{})},
	},
	{
		input: []types.ReceiptForStorage{{CumulativeGasUsed: 555, Status: 1, Logs: receiptsTestLogs2}},
		txs:   []*types.Transaction{types.NewTx(&types.AccessListTx{})},
	},
	{
		input: []types.ReceiptForStorage{
			{CumulativeGasUsed: 111, PostState: common.HexToHash("0x1111").Bytes(), Logs: receiptsTestLogs1},
			{CumulativeGasUsed: 222, Status: 0, Logs: receiptsTestLogs2},
			{CumulativeGasUsed: 333, Status: 1, Logs: nil},
		},
		txs: []*types.Transaction{
			types.NewTx(&types.LegacyTx{}),
			types.NewTx(&types.AccessListTx{}),
			types.NewTx(&types.DynamicFeeTx{}),
		},
	},
	{
		input: []types.ReceiptForStorage{
			{CumulativeGasUsed: 21000, Status: 1, Logs: receiptsTestLogs1},
			{
				CumulativeGasUsed: 90000,
				Payer:             &receiptsTestPayer,
				FrameReceipts: []types.FrameReceipt{
					{Status: types.ReceiptStatusSuccessful, GasUsed: 30000, StateGasUsed: 100, Logs: receiptsTestLogs1},
					{Status: types.ReceiptStatusFailed, GasUsed: 20000, Logs: receiptsTestLogs2},
					{Status: types.ReceiptStatusSuccessful, GasUsed: 19000},
				},
			},
			{CumulativeGasUsed: 111000, Status: 0, Logs: receiptsTestLogs2},
		},
		txs: []*types.Transaction{
			types.NewTx(&types.DynamicFeeTx{}),
			types.NewTx(&types.FrameTx{}),
			types.NewTx(&types.LegacyTx{}),
		},
	},
	{
		input: []types.ReceiptForStorage{
			{CumulativeGasUsed: 50000, Payer: &receiptsTestPayer, FrameReceipts: []types.FrameReceipt{}},
		},
		txs: []*types.Transaction{types.NewTx(&types.FrameTx{})},
	},
}

// receiptsLogsSize returns the expected ReceiptList.LogsSize of receipts.
func receiptsLogsSize(receipts []types.ReceiptForStorage) uint64 {
	contentSize := func(logs []*types.Log) uint64 {
		if logs == nil {
			logs = []*types.Log{}
		}
		enc, _ := rlp.EncodeToBytes(logs)
		content, _, _ := rlp.SplitList(enc)
		return uint64(len(content))
	}
	var size uint64
	for _, r := range receipts {
		if r.Type != types.FrameTxType {
			size += contentSize(r.Logs)
			continue
		}
		for _, fr := range r.FrameReceipts {
			size += contentSize(fr.Logs)
		}
	}
	return size
}

func init() {
	for i := range receiptsTests {
		// derive basic fields
		for j := range receiptsTests[i].input {
			r := (*types.Receipt)(&receiptsTests[i].input[j])
			txType := receiptsTests[i].txs[j].Type()
			miniDeriveFields(r, txType)
		}
		// compute expected root
		receipts := make(types.Receipts, len(receiptsTests[i].input))
		for j, sr := range receiptsTests[i].input {
			r := types.Receipt(sr)
			receipts[j] = &r
		}
		receiptsTests[i].root = types.DeriveSha(receipts, trie.NewStackTrie(nil))
	}
}

func TestReceiptList(t *testing.T) {
	for i, test := range receiptsTests {
		// encode receipts from types.ReceiptForStorage object.
		canonDB, _ := rlp.EncodeToBytes(test.input)

		// encode block body from types object.
		blockBody := types.Body{Transactions: test.txs}
		canonBody, _ := rlp.EncodeToBytes(blockBody)

		// convert from storage encoding to network encoding
		network, incomplete, err := blockReceiptsToNetwork(canonDB, canonBody, receiptQueryParams{})
		if err != nil {
			t.Fatalf("test[%d]: blockReceiptsToNetwork error: %v", i, err)
		}
		if incomplete {
			t.Fatalf("test[%d]: blockReceiptsToNetwork returned incomplete == true", i)
		}

		// parse as Receipts response list from network encoding
		var rl ReceiptList
		if err := rlp.DecodeBytes(network, &rl); err != nil {
			t.Fatalf("test[%d]: can't decode network receipts: %v", i, err)
		}
		rlStorageEnc, err := rl.EncodeForStorage()
		if err != nil {
			t.Fatalf("test[%d]: error from EncodeForStorage: %v", i, err)
		}
		if !bytes.Equal(rlStorageEnc, canonDB) {
			t.Fatalf("test[%d]: re-encoded receipts not equal\nhave: %x\nwant: %x", i, rlStorageEnc, canonDB)
		}
		rlNetworkEnc, _ := rlp.EncodeToBytes(&rl)
		if !bytes.Equal(rlNetworkEnc, network) {
			t.Fatalf("test[%d]: re-encoded network receipt list not equal\nhave: %x\nwant: %x", i, rlNetworkEnc, network)
		}

		// check log size accounting.
		if size, err := rl.LogsSize(); err != nil || size != receiptsLogsSize(test.input) {
			t.Fatalf("test[%d]: wrong logs size: have %d (err %v), want %d", i, size, err, receiptsLogsSize(test.input))
		}

		// compute root hash from ReceiptList and compare.
		responseHash := types.DeriveSha(rl.Derivable(), trie.NewStackTrie(nil))
		if responseHash != test.root {
			t.Fatalf("test[%d]: wrong root hash from ReceiptList\nhave: %v\nwant: %v", i, responseHash, test.root)
		}
	}
}

func TestFrameReceiptDecodeMalformed(t *testing.T) {
	validFrame := []any{types.ReceiptStatusSuccessful, []uint64{30000, 100}, receiptsTestLogs1}
	tests := []struct {
		name   string
		payer  []byte
		frames []any
	}{
		{"short payer", receiptsTestPayer[:19], []any{validFrame}},
		{"status is a list", receiptsTestPayer[:], []any{validFrame, []any{[]uint64{1}, []uint64{30000, 100}, receiptsTestLogs1}}},
		{"gas is a scalar", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), uint64(30000), receiptsTestLogs1}}},
		{"gas has one item", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), []uint64{30000}, receiptsTestLogs1}}},
		{"gas has 3 items", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), []uint64{30000, 100, 7}, receiptsTestLogs1}}},
		{"logs is a string", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), []uint64{30000, 100}, []byte{1}}}},
		{"missing logs", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), []uint64{30000, 100}}}},
		{"junk after logs", receiptsTestPayer[:], []any{validFrame, []any{uint64(1), []uint64{30000, 100}, receiptsTestLogs1, uint64(9)}}},
		{"frame is a string", receiptsTestPayer[:], []any{validFrame, []byte{1}}},
	}
	for _, test := range tests {
		encoded, _ := rlp.EncodeToBytes([]any{uint64(types.FrameTxType), uint64(90000), test.payer, test.frames})
		var r Receipt
		if err := r.decode(encoded); err == nil {
			t.Errorf("%s: decode accepted malformed frame receipt", test.name)
		}
		if test.name == "short payer" {
			continue // LogsSize does not inspect the payer
		}
		list, _ := rlp.EncodeToBytes([]rlp.RawValue{encoded})
		var rl ReceiptList
		if err := rlp.DecodeBytes(list, &rl); err != nil {
			t.Fatalf("%s: can't decode receipt list: %v", test.name, err)
		}
		if _, err := rl.LogsSize(); err == nil {
			t.Errorf("%s: LogsSize accepted malformed frame receipt", test.name)
		}
	}
}
