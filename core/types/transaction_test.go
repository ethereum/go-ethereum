// Copyright 2014 The go-ethereum Authors
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
	"crypto/ecdsa"
	"encoding/json"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// The values in those tests are from the Transaction Tests
// at github.com/ethereum/tests.
var (
	emptyTx = NewTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), 0, big.NewInt(0),
		nil,
		nil,
		nil,
	)

	rightvrsTx, _ = NewTransaction(
		3,
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
		nil,
		nil,
	).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	noRctNoSigTx = NewContractCreation(
		3,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
		nil,
		nil,
	)

	noSignatureTx = NewTransaction(
		3,
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
		nil,
		nil,
	)

	eip1559Tx, _ = NewTransaction(
		3,
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(10),
		2000,
		nil,
		common.FromHex("5544"),
		big.NewInt(200000),
		big.NewInt(800000),
	).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	eip1559NoRctNoSigTx = NewContractCreation(
		3,
		big.NewInt(10),
		2000,
		nil,
		common.FromHex("5544"),
		big.NewInt(200000),
		big.NewInt(800000),
	)

	eip1559NoSignatureTx = NewTransaction(
		3,
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(10),
		2000,
		nil,
		common.FromHex("5544"),
		big.NewInt(200000),
		big.NewInt(800000),
	)
)

func TestTransactionSigHash(t *testing.T) {
	var homestead HomesteadSigner
	if homestead.Hash(emptyTx) != common.HexToHash("c775b99e7ad12f50d819fcd602390467e28141316969f4b57f0626f74fe3b386") {
		t.Errorf("empty transaction hash mismatch, got %x", homestead.Hash(emptyTx))
	}
	if homestead.Hash(rightvrsTx) != common.HexToHash("fe7a79529ed5f7c3375d06b26b186a8644e0e16c373d7a12be41c62d6042b77a") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", homestead.Hash(rightvrsTx))
	}
	if homestead.Hash(eip1559Tx) != common.HexToHash("33bf0422a1819f7b82ae6cba37ae0169a37ec05ccb6d5a9963fe48cf765fe97f") {
		t.Errorf("eip1559Tx transaction hash mismatch, got %x", homestead.Hash(eip1559Tx))
	}
}

func TestTransactionEncode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(rightvrsTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	should := common.FromHex("f86103018207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a8255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(txb, should) {
		t.Errorf("encoded RLP mismatch, got %x", txb)
	}
}

func TestEIP1159TransactionEncode(t *testing.T) {
	tx1559b, err := rlp.EncodeToBytes(eip1559Tx)
	if err != nil {
		t.Fatalf("EIP1559 tx encode error: %v", err)
	}
	tx1559bshould := common.FromHex("f86903808207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a82554483030d40830c35001ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(tx1559b, tx1559bshould) {
		t.Errorf("EIP1559 tx encoded RLP mismatch, got %x", tx1559b)
	}
}

func TestEIP1159TransactionDecode(t *testing.T) {
	tx, err := decodeTx(common.Hex2Bytes("f86903808207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a82554483030d40830c35001ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3"))
	if err != nil {
		t.Fatal(err)
	}
	if tx.data.FeeCap == nil || tx.data.FeeCap.Cmp(eip1559Tx.data.FeeCap) != 0 {
		t.Fatal("unexpected FeeCap")
	}
	if tx.data.GasPremium == nil || tx.data.GasPremium.Cmp(eip1559Tx.data.GasPremium) != 0 {
		t.Fatal("unexpected GasPremium")
	}
	if tx.data.Price != nil {
		t.Fatal("expected GasPrice to be nil")
	}
	if tx.data.Hash != nil {
		t.Fatal("expected tx Hash to be nil")
	}
	if tx.data.GasLimit != eip1559Tx.data.GasLimit {
		t.Fatal("unexpected GasLimit")
	}
	if tx.data.Amount == nil || tx.data.Amount.Cmp(eip1559Tx.data.Amount) != 0 {
		t.Fatal("unexpected Amount")
	}
	if tx.data.Recipient == nil || !bytes.Equal(tx.data.Recipient.Bytes(), eip1559Tx.data.Recipient.Bytes()) {
		t.Fatal("unexpected Recipient")
	}
	if !bytes.Equal(tx.data.Payload, eip1559Tx.data.Payload) {
		t.Fatal("unexpected Payload")
	}
	if tx.data.AccountNonce != eip1559Tx.data.AccountNonce {
		t.Fatal("unexpected AccountNonce")
	}
	if tx.data.R == nil || tx.data.R.Cmp(eip1559Tx.data.R) != 0 {
		t.Fatal("unexpected R")
	}
	if tx.data.V == nil || tx.data.V.Cmp(eip1559Tx.data.V) != 0 {
		t.Fatal("unexpected V")
	}
	if tx.data.S == nil || tx.data.S.Cmp(eip1559Tx.data.S) != 0 {
		t.Fatal("unexpected S")
	}
}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	t, err := &tx, rlp.Decode(bytes.NewReader(data), &tx)

	return t, err
}

func defaultTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func TestRecipientEmpty(t *testing.T) {
	key, addr := defaultTestKey()
	tx, err := decodeTx(common.Hex2Bytes("f8498080808080011ca09b16de9d5bdee2cf56c28d16275a4da68cd30273e2525f3959f5d62557489921a0372ebd8fb3345f7db7b5a86d42e24d36e983e259b0664ceb8c227ec9af572f3d"))
	if err != nil {
		t.Fatal(err)
	}
	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}

	tx2, err := SignTx(noRctNoSigTx, HomesteadSigner{}, key)
	if err != nil {
		log.Fatal(err)
	}
	from2, err := Sender(HomesteadSigner{}, tx2)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from2 {
		t.Fatal("derived address doesn't match")
	}
}

func TestEIP1559RecipientEmpty(t *testing.T) {
	key, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f85503808207d0800a82554483030d40830c35001ca09de1afa53c1d66d6d759d28c1f2f48e3073de340e0b0966bd3b72e86c61dc40ca0120a40bca942f11fc1c00ac52f499acb006dd7cb70e3dbf41bae82e4fbf373c3"))
	if err != nil {
		t.Fatal(err)
	}
	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}

	tx2, err := SignTx(eip1559NoRctNoSigTx, HomesteadSigner{}, key)
	if err != nil {
		log.Fatal(err)
	}
	from2, err := Sender(HomesteadSigner{}, tx2)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from2 {
		t.Fatal("derived address doesn't match")
	}
}

func TestRecipientNormal(t *testing.T) {
	key, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f85d80808094000000000000000000000000000000000000000080011ca0527c0d8f5c63f7b9f41324a7c8a563ee1190bcbf0dac8ab446291bdbf32f5c79a0552c4ef0a09a04395074dab9ed34d3fbfb843c2f2546cc30fe89ec143ca94ca6"))
	if err != nil {
		t.Fatal(err)
	}
	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}

	tx2, err := SignTx(noSignatureTx, HomesteadSigner{}, key)
	if err != nil {
		log.Fatal(err)
	}
	from2, err := Sender(HomesteadSigner{}, tx2)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from2 {
		t.Fatal("derived address doesn't match")
	}
}

func TestEIP1559RecipientNormal(t *testing.T) {
	key, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f86903808207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a82554483030d40830c35001ba0612a9c962f0ac2841c671c021e45aeaa23f2892bf34da5d32d7948754cf078bda03a350e0e4e1ff5299228eb921af7c0435dbabd5b3d17f79c925864192ca9d126"))
	if err != nil {
		t.Fatal(err)
	}
	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}

	tx2, err := SignTx(eip1559NoSignatureTx, HomesteadSigner{}, key)
	if err != nil {
		log.Fatal(err)
	}
	from2, err := Sender(HomesteadSigner{}, tx2)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from2 {
		t.Fatal("derived address doesn't match")
	}
}

// Tests that transactions can be correctly sorted according to their price in
// decreasing order, but at the same time with increasing nonces when issued by
// the same account.
func TestTransactionPriceNonceSort(t *testing.T) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 25)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := HomesteadSigner{}

	// Generate a batch of transactions with overlapping values, but shifted nonces
	groups := map[common.Address]Transactions{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for i := 0; i < 25; i++ {
			tx, _ := SignTx(NewTransaction(uint64(start+i), common.Address{}, big.NewInt(100), 100, big.NewInt(int64(start+i)), nil, nil, nil), signer, key)
			groups[addr] = append(groups[addr], tx)
		}
	}
	// Sort the transactions and cross check the nonce ordering
	txset := NewTransactionsByPriceAndNonce(signer, groups)

	txs := Transactions{}
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		txs = append(txs, tx)
		txset.Shift()
	}
	if len(txs) != 25*25 {
		t.Errorf("expected %d transactions, found %d", 25*25, len(txs))
	}
	for i, txi := range txs {
		fromi, _ := Sender(signer, txi)

		// Make sure the nonce order is valid
		for j, txj := range txs[i+1:] {
			fromj, _ := Sender(signer, txj)
			if fromi == fromj && txi.Nonce() > txj.Nonce() {
				t.Errorf("invalid nonce ordering: tx #%d (A=%x N=%v) < tx #%d (A=%x N=%v)", i, fromi[:4], txi.Nonce(), i+j, fromj[:4], txj.Nonce())
			}
		}
		// If the next tx has different from account, the price must be lower than the current one
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := Sender(signer, next)
			if fromi != fromNext && txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
		}
	}
}

// Tests that if multiple transactions have the same price, the ones seen earlier
// are prioritized to avoid network spam attacks aiming for a specific ordering.
func TestTransactionTimeSort(t *testing.T) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := HomesteadSigner{}

	// Generate a batch of transactions with overlapping prices, but different creation times
	groups := map[common.Address]Transactions{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)

		tx, _ := SignTx(NewTransaction(0, common.Address{}, big.NewInt(100), 100, big.NewInt(1), nil), signer, key)
		tx.time = time.Unix(0, int64(len(keys)-start))

		groups[addr] = append(groups[addr], tx)
	}
	// Sort the transactions and cross check the nonce ordering
	txset := NewTransactionsByPriceAndNonce(signer, groups)

	txs := Transactions{}
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		txs = append(txs, tx)
		txset.Shift()
	}
	if len(txs) != len(keys) {
		t.Errorf("expected %d transactions, found %d", len(keys), len(txs))
	}
	for i, txi := range txs {
		fromi, _ := Sender(signer, txi)
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := Sender(signer, next)

			if txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
			// Make sure time order is ascending if the txs have the same gas price
			if txi.GasPrice().Cmp(next.GasPrice()) == 0 && txi.time.After(next.time) {
				t.Errorf("invalid received time ordering: tx #%d (A=%x T=%v) > tx #%d (A=%x T=%v)", i, fromi[:4], txi.time, i+1, fromNext[:4], next.time)
			}
		}
	}
}

// TestTransactionJSON tests serializing/de-serializing to/from JSON.
func TestTransactionJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewEIP155Signer(common.Big1)

	transactions := make([]*Transaction, 0, 50)
	for i := uint64(0); i < 25; i++ {
		var tx *Transaction
		switch i % 2 {
		case 0:
			tx = NewTransaction(i, common.Address{1}, common.Big0, 1, common.Big2, []byte("abcdef"), nil, nil)
		case 1:
			tx = NewContractCreation(i, common.Big0, 1, common.Big2, []byte("abcdef"), nil, nil)
		}
		transactions = append(transactions, tx)

		signedTx, err := SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}

		transactions = append(transactions, signedTx)
	}

	for _, tx := range transactions {
		data, err := json.Marshal(tx)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var parsedTx *Transaction
		if err := json.Unmarshal(data, &parsedTx); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
		if tx.Hash() != parsedTx.Hash() {
			t.Errorf("parsed tx differs from original tx, want %v, got %v", tx, parsedTx)
		}
		if tx.ChainId().Cmp(parsedTx.ChainId()) != 0 {
			t.Errorf("invalid chain id, want %d, got %d", tx.ChainId(), parsedTx.ChainId())
		}
	}
}

// TestEIP1559TransactionJSON tests serializing/de-serializing to/from JSON.
func TestEIP1559TransactionJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewEIP155Signer(common.Big1)

	transactions := make([]*Transaction, 0, 50)
	for i := uint64(0); i < 25; i++ {
		var tx *Transaction
		switch i % 2 {
		case 0:
			tx = NewTransaction(i, common.Address{1}, common.Big0, 1, nil, []byte("abcdef"), big.NewInt(200000), big.NewInt(800000))
		case 1:
			tx = NewContractCreation(i, common.Big0, 1, nil, []byte("abcdef"), big.NewInt(200000), big.NewInt(800000))
		}
		transactions = append(transactions, tx)

		signedTx, err := SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}

		transactions = append(transactions, signedTx)
	}

	for _, tx := range transactions {
		data, err := json.Marshal(tx)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var parsedTx *Transaction
		if err := json.Unmarshal(data, &parsedTx); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
		if tx.Hash() != parsedTx.Hash() {
			t.Errorf("parsed tx differs from original tx, want %v, got %v", tx, parsedTx)
		}
		if tx.ChainId().Cmp(parsedTx.ChainId()) != 0 {
			t.Errorf("invalid chain id, want %d, got %d", tx.ChainId(), parsedTx.ChainId())
		}
	}
}
