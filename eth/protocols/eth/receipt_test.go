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
)

func TestTransformReceipts(t *testing.T) {
	tests := []struct {
		input  []types.ReceiptForStorage
		txs    []*types.Transaction
		output []Receipt
	}{
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.LegacyTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: nil}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.DynamicFeeTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: nil, TxType: 2}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: nil}},
			txs:    []*types.Transaction{types.NewTx(&types.AccessListTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: nil, TxType: 1}},
		},
		{
			input:  []types.ReceiptForStorage{{CumulativeGasUsed: 123, Status: 1, Logs: []*types.Log{{Address: common.Address{1}, Topics: []common.Hash{{1}}}}}},
			txs:    []*types.Transaction{types.NewTx(&types.AccessListTx{})},
			output: []Receipt{{GasUsed: 123, PostStateOrStatus: []byte{1}, Logs: []*types.Log{{Address: common.Address{1}, Topics: []common.Hash{{1}}}}, Type: 1}},
		},
	}

	for i, test := range tests {
		in, _ := rlp.EncodeToBytes(test.input)
		have := blockReceiptsToNetwork(in, test.txs)
		out, _ := rlp.EncodeToBytes(test.output)
		if !bytes.Equal(have, out) {
			t.Fatalf("transforming receipt mismatch, test %v: want %v have %v", i, out, have)
		}
	}
}
