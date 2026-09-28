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

		// compute root hash from ReceiptList and compare.
		responseHash := types.DeriveSha(rl.Derivable(), trie.NewStackTrie(nil))
		if responseHash != test.root {
			t.Fatalf("test[%d]: wrong root hash from ReceiptList\nhave: %v\nwant: %v", i, responseHash, test.root)
		}
	}
}

func TestFrameReceiptListRoundTrip(t *testing.T) {
	payer := common.Address{0xaa}
	frameReceipt := &types.Receipt{
		Type:              types.FrameTxType,
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 90000,
		Payer:             &payer,
		FrameReceipts: []types.FrameReceipt{
			{Status: types.ReceiptStatusSuccessful, GasUsed: 30000, StateGasUsed: 100, Logs: receiptsTestLogs1},
			{Status: types.ReceiptStatusFailed, GasUsed: 20000, StateGasUsed: 0, Logs: receiptsTestLogs2},
		},
	}
	frameReceipt.Logs = append(append([]*types.Log{}, receiptsTestLogs1...), receiptsTestLogs2...)
	frameReceipt.Bloom = types.CreateBloom(frameReceipt)
	legacyReceipt := &types.Receipt{Type: types.DynamicFeeTxType, Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 21000, Logs: receiptsTestLogs1}
	legacyReceipt.Bloom = types.CreateBloom(legacyReceipt)
	receipts := types.Receipts{legacyReceipt, frameReceipt}
	wantRoot := types.DeriveSha(receipts, trie.NewStackTrie(nil))

	network, err := rlp.EncodeToBytes(NewReceiptList(receipts))
	if err != nil {
		t.Fatalf("can't encode network receipts: %v", err)
	}
	storageReceipts := []*types.ReceiptForStorage{(*types.ReceiptForStorage)(legacyReceipt), (*types.ReceiptForStorage)(frameReceipt)}
	canonDB, _ := rlp.EncodeToBytes(storageReceipts)
	canonBody, _ := rlp.EncodeToBytes(types.Body{Transactions: []*types.Transaction{types.NewTx(&types.DynamicFeeTx{}), types.NewTx(&types.FrameTx{})}})
	served, _, err := blockReceiptsToNetwork(canonDB, canonBody, receiptQueryParams{})
	if err != nil {
		t.Fatalf("blockReceiptsToNetwork error: %v", err)
	}
	if !bytes.Equal(network, served) {
		t.Fatalf("network encoding differs from served receipts\nhave: %x\nwant: %x", network, served)
	}

	var decoded ReceiptList
	if err := rlp.DecodeBytes(network, &decoded); err != nil {
		t.Fatalf("can't decode network receipts: %v", err)
	}
	legacyLogs, _ := rlp.EncodeToBytes(legacyReceipt.Logs)
	frameLogs, _ := rlp.EncodeToBytes(frameReceipt.Logs)
	legacyLogsContent, _, _ := rlp.SplitList(legacyLogs)
	frameLogsContent, _, _ := rlp.SplitList(frameLogs)
	wantLogsSize := uint64(len(legacyLogsContent) + len(frameLogsContent))
	if haveLogsSize, err := decoded.LogsSize(); err != nil || haveLogsSize != wantLogsSize {
		t.Fatalf("wrong logs size: have %d (err %v), want %d", haveLogsSize, err, wantLogsSize)
	}
	reencoded, _ := rlp.EncodeToBytes(&decoded)
	if !bytes.Equal(reencoded, network) {
		t.Fatalf("re-encoded network receipt list not equal\nhave: %x\nwant: %x", reencoded, network)
	}
	if haveRoot := types.DeriveSha(decoded.Derivable(), trie.NewStackTrie(nil)); haveRoot != wantRoot {
		t.Fatalf("wrong root hash from ReceiptList\nhave: %v\nwant: %v", haveRoot, wantRoot)
	}
}

func TestFrameReceiptDecodeMalformed(t *testing.T) {
	payer := common.Address{0xaa}.Bytes()
	validFrame := []any{types.ReceiptStatusSuccessful, []uint64{30000, 100}, receiptsTestLogs1}
	tests := map[string][]any{
		"valid":             {validFrame},
		"status is a list":  {validFrame, []any{[]uint64{1}, []uint64{30000, 100}, receiptsTestLogs1}},
		"gas is a scalar":   {validFrame, []any{uint64(1), uint64(30000), receiptsTestLogs1}},
		"gas has one item":  {validFrame, []any{uint64(1), []uint64{30000}, receiptsTestLogs1}},
		"gas has 3 items":   {validFrame, []any{uint64(1), []uint64{30000, 100, 7}, receiptsTestLogs1}},
		"logs is a string":  {validFrame, []any{uint64(1), []uint64{30000, 100}, []byte{1}}},
		"missing logs":      {validFrame, []any{uint64(1), []uint64{30000, 100}}},
		"junk after logs":   {validFrame, []any{uint64(1), []uint64{30000, 100}, receiptsTestLogs1, uint64(9)}},
		"frame is a string": {validFrame, []byte{1}},
	}
	for name, frames := range tests {
		encoded, err := rlp.EncodeToBytes([]any{uint64(types.FrameTxType), uint64(90000), payer, frames})
		if err != nil {
			t.Fatalf("%s: can't encode receipt: %v", name, err)
		}
		var r Receipt
		decodeErr := r.decode(encoded)
		list, _ := rlp.EncodeToBytes([]rlp.RawValue{encoded})
		var rl ReceiptList
		if err := rlp.DecodeBytes(list, &rl); err != nil {
			t.Fatalf("%s: can't decode receipt list: %v", name, err)
		}
		_, logsSizeErr := rl.LogsSize()
		if name == "valid" {
			if decodeErr != nil || logsSizeErr != nil {
				t.Fatalf("%s: unexpected errors: decode %v, logs size %v", name, decodeErr, logsSizeErr)
			}
			continue
		}
		if decodeErr == nil {
			t.Errorf("%s: decode accepted malformed frame receipt", name)
		}
		if logsSizeErr == nil {
			t.Errorf("%s: LogsSize accepted malformed frame receipt", name)
		}
	}
}
