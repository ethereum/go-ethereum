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

package miner

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestTransactionPriceNonceSortLegacy(t *testing.T) {
	t.Parallel()
	testTransactionPriceNonceSort(t, nil)
}

func TestTransactionPriceNonceSort1559(t *testing.T) {
	t.Parallel()
	testTransactionPriceNonceSort(t, big.NewInt(0))
	testTransactionPriceNonceSort(t, big.NewInt(5))
	testTransactionPriceNonceSort(t, big.NewInt(50))
}

// Tests that transactions can be correctly sorted according to their price in
// decreasing order, but at the same time with increasing nonces when issued by
// the same account.
func testTransactionPriceNonceSort(t *testing.T, baseFee *big.Int) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 25)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := types.LatestSignerForChainID(common.Big1)

	// Generate a batch of transactions with overlapping values, but shifted nonces
	groups := map[common.Address][]*txpool.LazyTransaction{}
	expectedCount := 0
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		count := 25
		for i := 0; i < 25; i++ {
			var tx *types.Transaction
			gasFeeCap := rand.Intn(50)
			if baseFee == nil {
				tx = types.NewTx(&types.LegacyTx{
					Nonce:    uint64(start + i),
					To:       &common.Address{},
					Value:    big.NewInt(100),
					Gas:      100,
					GasPrice: big.NewInt(int64(gasFeeCap)),
					Data:     nil,
				})
			} else {
				tx = types.NewTx(&types.DynamicFeeTx{
					Nonce:     uint64(start + i),
					To:        &common.Address{},
					Value:     big.NewInt(100),
					Gas:       100,
					GasFeeCap: big.NewInt(int64(gasFeeCap)),
					GasTipCap: big.NewInt(int64(rand.Intn(gasFeeCap + 1))),
					Data:      nil,
				})
				if count == 25 && int64(gasFeeCap) < baseFee.Int64() {
					count = i
				}
			}
			tx, err := types.SignTx(tx, signer, key)
			if err != nil {
				t.Fatalf("failed to sign tx: %s", err)
			}
			groups[addr] = append(groups[addr], &txpool.LazyTransaction{
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
				GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		}
		expectedCount += count
	}
	// Sort the transactions and cross check the nonce ordering
	txset := newTransactionsByPriceAndNonce(signer, groups, baseFee)

	txs := types.Transactions{}
	for tx, _ := txset.Peek(); tx != nil; tx, _ = txset.Peek() {
		txs = append(txs, tx.Tx)
		txset.Shift()
	}
	if len(txs) != expectedCount {
		t.Errorf("expected %d transactions, found %d", expectedCount, len(txs))
	}
	for i, txi := range txs {
		fromi, _ := types.Sender(signer, txi)

		// Make sure the nonce order is valid
		for j, txj := range txs[i+1:] {
			fromj, _ := types.Sender(signer, txj)
			if fromi == fromj && txi.Nonce() > txj.Nonce() {
				t.Errorf("invalid nonce ordering: tx #%d (A=%x N=%v) < tx #%d (A=%x N=%v)", i, fromi[:4], txi.Nonce(), i+j, fromj[:4], txj.Nonce())
			}
		}
		// If the next tx has different from account, the price must be lower than the current one
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := types.Sender(signer, next)
			tip, err := txi.EffectiveGasTip(baseFee)
			nextTip, nextErr := next.EffectiveGasTip(baseFee)
			if err != nil || nextErr != nil {
				t.Errorf("error calculating effective tip: %v, %v", err, nextErr)
			}
			if fromi != fromNext && tip.Cmp(nextTip) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
		}
	}
}

// TestTransactionLookahead verifies that a sender with a low-tip head transaction
// followed by a high-tip transaction is promoted above a sender whose single
// transaction has a tip between the two. Without look-ahead scoring the low-tip
// head would bury the high-value transaction.
func TestTransactionLookahead(t *testing.T) {
	t.Parallel()

	signer := types.LatestSignerForChainID(common.Big1)
	baseFee := big.NewInt(10)

	keyA, _ := crypto.GenerateKey()
	keyB, _ := crypto.GenerateKey()
	addrA := crypto.PubkeyToAddress(keyA.PublicKey)
	addrB := crypto.PubkeyToAddress(keyB.PublicKey)

	// Sender A: nonce 0 has tip 1, nonce 1 has tip 100.
	// Head-only score = 1, but look-ahead average ≈ 50.
	txA0, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     0,
		To:        &common.Address{},
		Gas:       100,
		GasFeeCap: big.NewInt(11), // baseFee + 1
		GasTipCap: big.NewInt(1),
	}), signer, keyA)
	txA1, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     1,
		To:        &common.Address{},
		Gas:       100,
		GasFeeCap: big.NewInt(110), // baseFee + 100
		GasTipCap: big.NewInt(100),
	}), signer, keyA)

	// Sender B: single tx with tip 20.
	// Without look-ahead, B (tip 20) would rank above A (tip 1).
	// With look-ahead, A's score (~50) should rank above B (20).
	txB0, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     0,
		To:        &common.Address{},
		Gas:       100,
		GasFeeCap: big.NewInt(30), // baseFee + 20
		GasTipCap: big.NewInt(20),
	}), signer, keyB)

	now := time.Now()
	groups := map[common.Address][]*txpool.LazyTransaction{
		addrA: {
			{Hash: txA0.Hash(), Tx: txA0, Time: now, GasFeeCap: uint256.NewInt(11), GasTipCap: uint256.NewInt(1), Gas: 100},
			{Hash: txA1.Hash(), Tx: txA1, Time: now, GasFeeCap: uint256.NewInt(110), GasTipCap: uint256.NewInt(100), Gas: 100},
		},
		addrB: {
			{Hash: txB0.Hash(), Tx: txB0, Time: now, GasFeeCap: uint256.NewInt(30), GasTipCap: uint256.NewInt(20), Gas: 100},
		},
	}

	txset := newTransactionsByPriceAndNonce(signer, groups, baseFee)

	// First tx out should be A's nonce 0 (sender A ranked higher due to look-ahead).
	first, _ := txset.Peek()
	if first == nil {
		t.Fatal("expected a transaction")
	}
	if first.Hash != txA0.Hash() {
		t.Errorf("expected sender A's tx first (look-ahead should promote it), got sender B")
	}
	txset.Shift()

	// Second should be A's nonce 1 (tip 100 > B's tip 20).
	second, _ := txset.Peek()
	if second == nil {
		t.Fatal("expected a transaction")
	}
	if second.Hash != txA1.Hash() {
		t.Errorf("expected sender A's nonce 1 second, got %s", second.Hash)
	}
	txset.Shift()

	// Third should be B's tx.
	third, _ := txset.Peek()
	if third == nil {
		t.Fatal("expected a transaction")
	}
	if third.Hash != txB0.Hash() {
		t.Errorf("expected sender B's tx third, got %s", third.Hash)
	}
	txset.Shift()

	// Should be empty now.
	if last, _ := txset.Peek(); last != nil {
		t.Error("expected no more transactions")
	}
}

// Tests that if multiple transactions have the same price, the ones seen earlier
// are prioritized to avoid network spam attacks aiming for a specific ordering.
func TestTransactionTimeSort(t *testing.T) {
	t.Parallel()
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := types.HomesteadSigner{}

	// Generate a batch of transactions with overlapping prices, but different creation times
	groups := map[common.Address][]*txpool.LazyTransaction{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)

		tx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100, big.NewInt(1), nil), signer, key)
		tx.SetTime(time.Unix(0, int64(len(keys)-start)))

		groups[addr] = append(groups[addr], &txpool.LazyTransaction{
			Hash:      tx.Hash(),
			Tx:        tx,
			Time:      tx.Time(),
			GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
			GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
			Gas:       tx.Gas(),
			BlobGas:   tx.BlobGas(),
		})
	}
	// Sort the transactions and cross check the nonce ordering
	txset := newTransactionsByPriceAndNonce(signer, groups, nil)

	txs := types.Transactions{}
	for tx, _ := txset.Peek(); tx != nil; tx, _ = txset.Peek() {
		txs = append(txs, tx.Tx)
		txset.Shift()
	}
	if len(txs) != len(keys) {
		t.Errorf("expected %d transactions, found %d", len(keys), len(txs))
	}
	for i, txi := range txs {
		fromi, _ := types.Sender(signer, txi)
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := types.Sender(signer, next)

			if txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
			// Make sure time order is ascending if the txs have the same gas price
			if txi.GasPrice().Cmp(next.GasPrice()) == 0 && txi.Time().After(next.Time()) {
				t.Errorf("invalid received time ordering: tx #%d (A=%x T=%v) > tx #%d (A=%x T=%v)", i, fromi[:4], txi.Time(), i+1, fromNext[:4], next.Time())
			}
		}
	}
}
