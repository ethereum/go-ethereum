// Copyright 2016 The go-ethereum Authors
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

package legacypool

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// Tests that transactions can be added to strict lists and list contents and
// nonce boundaries are correctly maintained.
func TestStrictListAdd(t *testing.T) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 1024)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newList(true)
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultConfig.PriceBump)
	}
	// Verify internal state
	if len(list.txs.items) != len(txs) {
		t.Errorf("transaction count mismatch: have %d, want %d", len(list.txs.items), len(txs))
	}
	for i, tx := range txs {
		if list.txs.items[tx.Nonce()] != tx {
			t.Errorf("item %d: transaction mismatch: have %v, want %v", i, list.txs.items[tx.Nonce()], tx)
		}
	}
}

// TestListAddVeryExpensive tests adding txs which exceed 256 bits in cost. It is
// expected that the list does not panic.
func TestListAddVeryExpensive(t *testing.T) {
	key, _ := crypto.GenerateKey()
	list := newList(true)
	for i := 0; i < 3; i++ {
		value := big.NewInt(100)
		gasprice, _ := new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
		gaslimit := uint64(i)
		tx, _ := types.SignTx(types.NewTransaction(uint64(i), common.Address{}, value, gaslimit, gasprice, nil), types.HomesteadSigner{}, key)
		t.Logf("cost: %x bitlen: %d\n", tx.Cost(), tx.Cost().BitLen())
		list.Add(tx, DefaultConfig.PriceBump)
	}
}

func TestListAddReplacementAvoidsIntermediateOverflow(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	max := new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	oldPrice := new(big.Int).Sub(new(big.Int).Rsh(new(big.Int).Set(max), 1), big.NewInt(100))
	newPrice := new(big.Int).Add(oldPrice, common.Big1)

	oldTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, common.Big0, 1, oldPrice, nil), types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatalf("failed to sign old tx: %v", err)
	}
	newTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, common.Big0, 1, newPrice, nil), types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatalf("failed to sign replacement tx: %v", err)
	}

	list := newList(true)
	inserted, _ := list.Add(oldTx, 0)
	if !inserted {
		t.Fatal("failed to insert baseline transaction")
	}
	inserted, replaced := list.Add(newTx, 0)
	if !inserted {
		t.Fatal("replacement transaction should not overflow after subtracting old cost")
	}
	if replaced == nil || replaced.Hash() != oldTx.Hash() {
		t.Fatal("expected old transaction to be replaced")
	}
	want, overflow := uint256.FromBig(newTx.Cost())
	if overflow {
		t.Fatal("replacement tx cost overflowed uint256 in test setup")
	}
	if list.totalcost.Cmp(want) != 0 {
		t.Fatalf("totalcost mismatch after replacement: have %v want %v", list.totalcost, want)
	}
	if tx := list.txs.Get(newTx.Nonce()); tx == nil || tx.Hash() != newTx.Hash() {
		t.Fatal("replacement transaction was not stored in list")
	}
	list.Forward(1)
	if list.totalcost.Sign() != 0 {
		t.Fatalf("totalcost should be zero after removal, have %v", list.totalcost)
	}
}

// TestPriceHeapCmp tests that the price heap comparison function works as intended.
// It also tests combinations where the basefee is higher than the gas fee cap, which
// are useful to sort in the mempool to support basefee changes.
func TestPriceHeapCmp(t *testing.T) {
	key, _ := crypto.GenerateKey()
	txs := []*types.Transaction{
		// nonce, gaslimit, gasfee, gastip
		dynamicFeeTx(0, 1000, big.NewInt(2), big.NewInt(1), key),
		dynamicFeeTx(0, 1000, big.NewInt(1), big.NewInt(2), key),
		dynamicFeeTx(0, 1000, big.NewInt(1), big.NewInt(1), key),
		dynamicFeeTx(0, 1000, big.NewInt(1), big.NewInt(0), key),
	}

	// create priceHeap
	ph := &priceHeap{}

	// now set the basefee on the heap
	for _, basefee := range []uint64{0, 1, 2, 3} {
		ph.baseFee = uint256.NewInt(basefee)

		for i := 0; i < len(txs); i++ {
			for j := 0; j < len(txs); j++ {
				switch {
				case i == j:
					if c := ph.cmp(txs[i], txs[j]); c != 0 {
						t.Errorf("tx %d should be equal priority to tx %d with basefee %d (cmp=%d)", i, j, basefee, c)
					}
				case i < j:
					if c := ph.cmp(txs[i], txs[j]); c != 1 {
						t.Errorf("tx %d vs tx %d comparison inconsistent with basefee %d (cmp=%d)", i, j, basefee, c)
					}
				}
			}
		}
	}
}

func BenchmarkListAdd(b *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 100000)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	priceLimit := uint256.NewInt(DefaultConfig.PriceLimit)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := newList(true)
		for _, v := range rand.Perm(len(txs)) {
			list.Add(txs[v], DefaultConfig.PriceBump)
			list.Filter(priceLimit, DefaultConfig.PriceBump)
		}
	}
}

func BenchmarkListCapOneTx(b *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 32)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := newList(true)
		// Insert the transactions in a random order
		for _, v := range rand.Perm(len(txs)) {
			list.Add(txs[v], DefaultConfig.PriceBump)
		}
		b.StartTimer()
		list.Cap(list.Len() - 1)
		b.StopTimer()
	}
}
