// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
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
	"maps"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestDepositTxRLP(t *testing.T) {
	deposits := []*DepositTx{
		{
			Version:         DepositTxVersion,
			SourceBlockHash: common.Hash{0xaa},
			SourceLogIndex:  7,
			From:            common.Address{0x01},
			To:              &common.Address{0x02},
			Value:           uint256.NewInt(1000),
			GasPrice:        uint256.NewInt(10),
			Gas:             9000,
			IsMove:          true,
			Data:            []byte{},
		},
		{
			Version:         DepositTxVersion,
			SourceBlockHash: common.Hash{0xbb},
			SourceLogIndex:  math.MaxUint64,
			From:            common.Address{0x03},
			To:              &common.Address{0x04},
			Value:           new(uint256.Int).SetAllOne(),
			GasPrice:        new(uint256.Int),
			Gas:             math.MaxUint64,
			Data:            bytes.Repeat([]byte{0xff}, 100),
		},
		{
			// A deposit creating a contract: a nil To, which no source shard
			// sends for now.
			Version:         DepositTxVersion,
			SourceBlockHash: common.Hash{0xcc},
			SourceLogIndex:  1,
			From:            common.Address{0x05},
			To:              nil,
			Value:           uint256.NewInt(1),
			GasPrice:        uint256.NewInt(1),
			Gas:             9000,
			Data:            []byte{0x60, 0x00},
		},
	}
	enc, err := rlp.EncodeToBytes(deposits)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	var dec []*DepositTx
	if err := rlp.DecodeBytes(enc, &dec); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !reflect.DeepEqual(dec, deposits) {
		t.Errorf("decoded %+v, want %+v", dec, deposits)
	}
	// Pin the encoding: the fields, in order, as an RLP list.
	want := common.FromHex("0xf856" + "80" + "a0" + "aa" + "00000000000000000000000000000000000000000000000000000000000000" + "07" +
		"94" + "0100000000000000000000000000000000000000" + "94" + "0200000000000000000000000000000000000000" +
		"8203e8" + "0a" + "822328" + "01" + "80")
	if enc, _ := rlp.EncodeToBytes(deposits[0]); !bytes.Equal(enc, want) {
		t.Errorf("encoding %x, want %x", enc, want)
	}
}

func TestDepositTxTransaction(t *testing.T) {
	deposit := &DepositTx{
		Version:         DepositTxVersion,
		SourceBlockHash: common.Hash{0xaa},
		SourceLogIndex:  3,
		From:            common.Address{0x01},
		To:              &common.Address{0x02},
		Value:           uint256.NewInt(1000),
		GasPrice:        uint256.NewInt(10),
		Gas:             50000,
		IsMove:          true,
		Data:            []byte{0xca, 0xfe},
	}
	tx := NewTx(deposit)
	enc, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	if enc[0] != DepositTxType {
		t.Fatalf("encoded type %#x, want %#x", enc[0], DepositTxType)
	}
	dec := new(Transaction)
	if err := dec.UnmarshalBinary(enc); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if dec.Hash() != tx.Hash() || !reflect.DeepEqual(dec.inner, deposit) {
		t.Fatalf("decoded %+v, want %+v", dec.inner, deposit)
	}
	if have := dec.Deposit(); !reflect.DeepEqual(have, deposit) {
		t.Fatalf("deposit copy %+v, want %+v", have, deposit)
	}
	// The deposit pays its gas price, whatever the base fee.
	if dec.Type() != DepositTxType || dec.Gas() != 50000 || dec.Nonce() != 0 || *dec.To() != *deposit.To ||
		dec.Value().Cmp(big.NewInt(1000)) != 0 || !bytes.Equal(dec.Data(), deposit.Data) {
		t.Errorf("decoded transaction fields do not match the deposit")
	}
	for _, price := range []*big.Int{dec.GasPrice(), dec.GasFeeCap(), dec.GasTipCap()} {
		if price.Cmp(big.NewInt(10)) != 0 {
			t.Errorf("gas price %v, want 10", price)
		}
	}
	for _, baseFee := range []*big.Int{nil, big.NewInt(1), big.NewInt(1000)} {
		if have := dec.inner.effectiveGasPrice(new(big.Int), baseFee); have.Cmp(big.NewInt(10)) != 0 {
			t.Errorf("base fee %v: effective gas price %v, want 10", baseFee, have)
		}
	}
	// The sender is From, without any signature.
	if from, err := Sender(LatestSignerForChainID(big.NewInt(1)), dec); err != nil || from != deposit.From {
		t.Errorf("sender %v (err %v), want %v", from, err, deposit.From)
	}
	// Only the known version decodes.
	deposit.Version = DepositTxVersion + 1
	if enc, _ := NewTx(deposit).MarshalBinary(); new(Transaction).UnmarshalBinary(enc) == nil {
		t.Error("decoded a deposit of an unknown version")
	}
	key, _ := crypto.GenerateKey()
	if _, err := SignTx(dec, LatestSignerForChainID(big.NewInt(1)), key); err == nil {
		t.Error("signed a deposit")
	}
}

func TestDepositTxReceipt(t *testing.T) {
	receipt := &Receipt{
		Type:              DepositTxType,
		Status:            ReceiptStatusSuccessful,
		CumulativeGasUsed: 9000,
		Logs:              []*Log{{Address: common.Address{0x01}, Topics: []common.Hash{{0x02}}, Data: []byte{0x03}}},
	}
	receipt.Bloom = CreateBloom(receipt)
	enc, err := receipt.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	var index bytes.Buffer
	Receipts{receipt}.EncodeIndex(0, &index)
	if !bytes.Equal(index.Bytes(), enc) {
		t.Errorf("receipt trie encoding %x, want %x", index.Bytes(), enc)
	}
	dec := new(Receipt)
	if err := dec.UnmarshalBinary(enc); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if dec.Type != DepositTxType || dec.Status != receipt.Status || dec.CumulativeGasUsed != receipt.CumulativeGasUsed ||
		dec.Bloom != receipt.Bloom || len(dec.Logs) != 1 {
		t.Errorf("decoded %+v, want %+v", dec, receipt)
	}
}

func TestDepositTxJSON(t *testing.T) {
	tx := NewTx(&DepositTx{
		Version:         DepositTxVersion,
		SourceBlockHash: common.Hash{0xaa},
		SourceLogIndex:  3,
		From:            common.Address{0x01},
		To:              &common.Address{0x02},
		Value:           uint256.NewInt(1000),
		GasPrice:        uint256.NewInt(10),
		Gas:             50000,
		IsMove:          true,
		Data:            []byte{0xca, 0xfe},
	})
	enc, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(enc, &fields); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]any{
		"type":            "0x7e",
		"from":            "0x0100000000000000000000000000000000000000",
		"to":              "0x0200000000000000000000000000000000000000",
		"version":         "0x0",
		"sourceBlockHash": "0xaa00000000000000000000000000000000000000000000000000000000000000",
		"sourceLogIndex":  "0x3",
		"isMove":          true,
		"gas":             "0xc350",
		"gasPrice":        "0xa",
		"value":           "0x3e8",
		"input":           "0xcafe",
		"hash":            tx.Hash().Hex(),
		"nonce":           nil,
		"v":               nil,
	} {
		if fields[name] != want {
			t.Errorf("field %s: have %v, want %v", name, fields[name], want)
		}
	}
	dec := new(Transaction)
	if err := json.Unmarshal(enc, dec); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if dec.Hash() != tx.Hash() {
		t.Errorf("decoded %+v, want %+v", dec.inner, tx.inner)
	}
	// The RPC encoding, with a zero nonce and signature, decodes too.
	fields["nonce"], fields["v"], fields["r"], fields["s"] = "0x0", "0x0", "0x0", "0x0"
	if rpc, _ := json.Marshal(fields); json.Unmarshal(rpc, new(Transaction)) != nil {
		t.Error("failed to decode the RPC encoding")
	}
	for name, edit := range map[string]func(map[string]any){
		"signed":          func(f map[string]any) { f["r"] = "0x1" },
		"chain id":        func(f map[string]any) { f["chainId"] = "0x1" },
		"unknown version": func(f map[string]any) { f["version"] = "0x1" },
		"no version":      func(f map[string]any) { delete(f, "version") },
		"no source":       func(f map[string]any) { delete(f, "sourceBlockHash") },
		"no log index":    func(f map[string]any) { delete(f, "sourceLogIndex") },
		"no move flag":    func(f map[string]any) { delete(f, "isMove") },
		"no sender":       func(f map[string]any) { delete(f, "from") },
		"no recipient":    func(f map[string]any) { delete(f, "to") },
		"no gas price":    func(f map[string]any) { delete(f, "gasPrice") },
	} {
		invalid := maps.Clone(fields)
		edit(invalid)
		data, _ := json.Marshal(invalid)
		if err := json.Unmarshal(data, new(Transaction)); err == nil {
			t.Errorf("%s: decoded an invalid deposit", name)
		}
	}
}
