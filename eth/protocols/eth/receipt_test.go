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

func TestReceiptList69(t *testing.T) {
	for i, test := range receiptsTests {
		// encode receipts from types.ReceiptForStorage object.
		canonDB, _ := rlp.EncodeToBytes(test.input)

		// encode block body from types object.
		blockBody := types.Body{Transactions: test.txs}
		canonBody, _ := rlp.EncodeToBytes(blockBody)

		// convert from storage encoding to network encoding
		network, err := blockReceiptsToNetwork69(canonDB, canonBody)
		if err != nil {
			t.Fatalf("test[%d]: blockReceiptsToNetwork69 error: %v", i, err)
		}

		// parse as Receipts response list from network encoding
		var rl ReceiptList69
		if err := rlp.DecodeBytes(network, &rl); err != nil {
			t.Fatalf("test[%d]: can't decode network receipts: %v", i, err)
		}
		rlStorageEnc := rl.EncodeForStorage()
		if !bytes.Equal(rlStorageEnc, canonDB) {
			t.Fatalf("test[%d]: re-encoded receipts not equal\nhave: %x\nwant: %x", i, rlStorageEnc, canonDB)
		}
		rlNetworkEnc, _ := rlp.EncodeToBytes(&rl)
		if !bytes.Equal(rlNetworkEnc, network) {
			t.Fatalf("test[%d]: re-encoded network receipt list not equal\nhave: %x\nwant: %x", i, rlNetworkEnc, network)
		}

		// compute root hash from ReceiptList69 and compare.
		responseHash := types.DeriveSha(&rl, trie.NewStackTrie(nil))
		if responseHash != test.root {
			t.Fatalf("test[%d]: wrong root hash from ReceiptList69\nhave: %v\nwant: %v", i, responseHash, test.root)
		}
	}
}

func TestReceiptList68(t *testing.T) {
	for i, test := range receiptsTests {
		// encode receipts from types.ReceiptForStorage object.
		canonDB, _ := rlp.EncodeToBytes(test.input)

		// encode block body from types object.
		blockBody := types.Body{Transactions: test.txs}
		canonBody, _ := rlp.EncodeToBytes(blockBody)

		// convert from storage encoding to network encoding
		network, err := blockReceiptsToNetwork68(canonDB, canonBody)
		if err != nil {
			t.Fatalf("test[%d]: blockReceiptsToNetwork68 error: %v", i, err)
		}

		// parse as Receipts response list from network encoding
		var rl ReceiptList68
		if err := rlp.DecodeBytes(network, &rl); err != nil {
			t.Fatalf("test[%d]: can't decode network receipts: %v", i, err)
		}
		rlStorageEnc := rl.EncodeForStorage()
		if !bytes.Equal(rlStorageEnc, canonDB) {
			t.Fatalf("test[%d]: re-encoded receipts not equal\nhave: %x\nwant: %x", i, rlStorageEnc, canonDB)
		}
		rlNetworkEnc, _ := rlp.EncodeToBytes(&rl)
		if !bytes.Equal(rlNetworkEnc, network) {
			t.Fatalf("test[%d]: re-encoded network receipt list not equal\nhave: %x\nwant: %x", i, rlNetworkEnc, network)
		}

		// compute root hash from ReceiptList68 and compare.
		responseHash := types.DeriveSha(&rl, trie.NewStackTrie(nil))
		if responseHash != test.root {
			t.Fatalf("test[%d]: wrong root hash from ReceiptList68\nhave: %v\nwant: %v", i, responseHash, test.root)
		}
	}
}
