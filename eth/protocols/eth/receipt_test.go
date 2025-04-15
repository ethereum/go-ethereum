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

		have, err := blockReceiptsToNetwork(in, encBlockBody)
		if err != nil {
			t.Fatalf("test[%d]: blockReceiptsToNetwork error: %v", i, err)
		}
		out, _ := rlp.EncodeToBytes(test.output)
		if !bytes.Equal(have, out) {
			t.Fatalf("test[%d]: blockReceiptsToNetwork mismatch\nhave: %x\nwant: %x\n  in: %x", i, out, have, in)
		}

		var rl ReceiptList69
		if err := rlp.DecodeBytes(out, &rl); err != nil {
			t.Fatalf("test[%d]: can't decode network receipts: %v", i, err)
		}
		storageEnc := rl.toStorageReceiptsRLP()
		if !bytes.Equal(storageEnc, in) {
			t.Fatalf("test[%d]: re-encoded receipts not equal\nhave: %x\nwant: %x", i, storageEnc, in)
		}
	}
}

func TestReceiptsMessage69(t *testing.T) {
	msg := common.FromHex("f9037c880e17dea70d94cb05f90370e7e680a0c97ed91127dd6f6a099889686685b3b204a5888a9d84431747324654eca53e4c825208c0e7e680a00ba7c3d24be245396fc5d3f80d8a69813637c63b2750a0aecca0ecf1893ae6de825208c0e7e680a0b1bdb7832a63083237ae909f02b5290e5ead11e947f90ad406ce58af42d9cbf2825208c0e7e680a0a8db4c173896d1961aab985d4084276cfbb5b460184c72ed9148093427813e50825208c0e7e680a02ac83e5239694103e4b8bfe88055573eb6947eb9ec456bd1ecd40c3a390c1184825208c0e7e680a05d61474327cd80ea904886eb01497db86aa4e1826f60ba8041b342b7e3271451825208c0e7e680a0c586c7bffdaa7eafd1e64755f6438d4737d3da328e4bf15bf36f425e35dcba57825208c0e7e680a0db8c7c749080c80efc9b8fec0036b7aebe1fdb130cc0529cd41658f0e4073f32825208c0e7e680a052820bf047785bce72e004175cf20430228ed6e3aafd2f0dc9d881ab4b8247b0825208c0e7e680a00b8fa31c4ba7486ba43779175b05f07e9a913d97f0d784294283c67ca08ecade825208c0e7e680a059d6258b1d014be483b4393850a81fa109d80d0d222acc0ca4897332839f8760825208c0e7e680a0128b6226c363fe1fe0a94bc576b46569ee4d59cfcff913df74165294bb1478de825208c0e7e680a0525f09191eb703dce45154db01474a84191b45bddd36e99b6cbf59a5a40ee935825208c0e7e680a032664bd056b1e2ae4dd7293e578b5d7300f97e1a1a81432d310d7100d1931c2d825208c0e7e680a07d29340b854047e215e787d04073bbf4451a4fbdeaaaac8ef67934eb2612d264825208c0e7e680a054b35228ab2fd0a096976f3c144cb86bed4f48e952ac78b4c4bd8ee377ce686a8256d0c0e7e680a0bf5fb6d0da92232bc4b0a5c8bc6675a9f298c23ce747877074fe24f6e2d75dfa825208c0e7e680a04fbf751cb29c49f34c9da0d8fe456afc8672b2e6049d51b01ff40456661456e9825208c0e7e680a050397035c265d9d628bf382e072fec34e72c98a86f48f7c6afdd7bbb128ea7e2825208c0e7e680a0ed10826649c2bdb34763b3f293cc18564cc883d6b6cbb84df75e7e328d3e2594825208c0e7e680a02ca0ecfc7149619fac0b5376a8a6fd1b0da6371ad875d50603d2adb667f12c2f825208c0e7e680a0801743bd8c5e43b7b77024ff80c4e437d8b505f8295dfb179f667b22c0762637825208c0")
	var dec ReceiptsPacket69
	if err := rlp.DecodeBytes(msg, &dec); err != nil {
		t.Fatal(err)
	}
	if len(dec.List) != 22 {
		t.Fatalf("wrong list length %d", len(dec.List))
	}
}
