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

func TestReceiptList69(t *testing.T) {
	logs := []*types.Log{{Address: common.Address{1}, Topics: []common.Hash{{1}}}}
	encLogs, _ := rlp.EncodeToBytes(logs)

	tests := []struct {
		input  []types.ReceiptForStorage
		txs    []*types.Transaction
		output []Receipt
	}{
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.LegacyTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: rlp.EmptyList}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.DynamicFeeTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: rlp.EmptyList, TxType: 2}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.AccessListTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: rlp.EmptyList, TxType: 1}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: logs}},
			txs:    []*types.Transaction{types.NewTx(&types.AccessListTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: encLogs, TxType: 1}},
		},
	}

	for i, test := range tests {
		// encode receipts from types object.
		in, _ := rlp.EncodeToBytes(test.input)

		// encode block body from types object.
		blockBody := types.Body{Transactions: test.txs}
		encBlockBody, _ := rlp.EncodeToBytes(blockBody)

		// convert from storage encoding to network encoding
		network, err := blockReceiptsToNetwork69(in, encBlockBody)
		if err != nil {
			t.Fatalf("test[%d]: blockReceiptsToNetwork error: %v", i, err)
		}

		// parse as Receipts response list from network encoding
		var rl ReceiptList69
		if err := rlp.DecodeBytes(network, &rl); err != nil {
			t.Fatalf("test[%d]: can't decode network receipts: %v", i, err)
		}
		storageEnc := rl.EncodeForStorage()
		if !bytes.Equal(storageEnc, in) {
			t.Fatalf("test[%d]: re-encoded receipts not equal\nhave: %x\nwant: %x", i, storageEnc, in)
		}

		// compute expected root hash
		receipts := make(types.Receipts, len(test.input))
		for i := range test.input {
			r := types.Receipt(test.input[i])
			miniDeriveFields(&r, test.txs[i].Type())
			receipts[i] = &r
		}
		expectedHash := types.DeriveSha(receipts, trie.NewStackTrie(nil))

		// compute root hash from ReceiptList69 and compare.
		responseHash := types.DeriveSha(&rl, trie.NewStackTrie(nil))
		if responseHash != expectedHash {
			t.Fatalf("test[%d]: wrong root hash from ReceiptList69\nhave: %v\nwant: %v", i, responseHash, expectedHash)
		}
	}
}
