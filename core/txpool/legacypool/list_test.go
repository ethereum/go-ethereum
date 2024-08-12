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

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Tests that transactions can be added to strict lists and list contents and
// nonce boundaries are correctly maintained.
func TestStrictListAdd(t *testing.T) {
	t.Parallel()

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

func BenchmarkListAdd(b *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 100000)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	priceLimit := big.NewInt(int64(DefaultConfig.PriceLimit))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := newList(true)
		for _, v := range rand.Perm(len(txs)) {
			list.Add(txs[v], DefaultConfig.PriceBump)
			list.Filter(priceLimit, DefaultConfig.PriceBump)
		}
	}
}

func TestFilterTxConditional(t *testing.T) {
	t.Parallel()

	// Create an in memory state db to test against.
	memDb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memDb)
	state, _ := state.New(common.Hash{}, db, nil)

	// Create a private key to sign transactions.
	key, _ := crypto.GenerateKey()

	// Create a list.
	list := newList(true)

	// Create a transaction with no defined tx options
	// and add to the list.
	tx := transaction(0, 1000, key)
	list.Add(tx, DefaultConfig.PriceBump)

	// There should be no drops at this point.
	// No state has been modified.
	drops := list.FilterTxConditional(state)

	count := len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Create another transaction with a known account storage root tx option
	// and add to the list.
	tx2 := transaction(1, 1000, key)

	var options types.OptionsPIP15

	options.KnownAccounts = types.KnownAccounts{
		common.Address{19: 1}: &types.Value{
			Single: common.HexToRefHash("0xe734938daf39aae1fa4ee64dc3155d7c049f28b57a8ada8ad9e86832e0253bef"),
		},
	}

	state.SetState(common.Address{19: 1}, common.Hash{}, common.Hash{30: 1})
	tx2.PutOptions(&options)
	list.Add(tx2, DefaultConfig.PriceBump)

	// There should still be no drops as no state has been modified.
	drops = list.FilterTxConditional(state)

	count = len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Set state that conflicts with tx2's policy
	state.SetState(common.Address{19: 1}, common.Hash{}, common.Hash{31: 1})

	// tx2 should be the single transaction filtered out
	drops = list.FilterTxConditional(state)

	count = len(drops)
	require.Equal(t, 1, count, "got %d filtered by TxOptions when there should be a single one", count)

	require.Equal(t, tx2, drops[0], "Got %x, expected %x", drops[0].Hash(), tx2.Hash())
}
