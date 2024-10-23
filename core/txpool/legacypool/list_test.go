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
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
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

func TestFilterTxConditionalKnownAccounts(t *testing.T) {
	t.Parallel()

	// Create an in memory state db to test against.
	memDb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memDb)
	state, _ := state.New(common.Hash{}, db, nil)

	header := &types.Header{
		Number: big.NewInt(0),
	}

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
	drops := list.FilterTxConditional(state, header)

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

	state.AddBalance(common.Address{19: 1}, uint256.NewInt(1000), tracing.BalanceChangeTransfer)

	trie, _ := state.StorageTrie(common.Address{19: 1})
	fmt.Println("before", trie)

	state.SetState(common.Address{19: 1}, common.Hash{}, common.Hash{30: 1})

	state.Finalise(true)

	trie, _ = state.StorageTrie(common.Address{19: 1})
	fmt.Println("after", trie.Hash())

	tx2.PutOptions(&options)
	list.Add(tx2, DefaultConfig.PriceBump)

	// There should still be no drops as no state has been modified.
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Set state that conflicts with tx2's policy
	state.SetState(common.Address{19: 1}, common.Hash{}, common.Hash{31: 1})

	state.Finalise(true)

	trie, _ = state.StorageTrie(common.Address{19: 1})
	fmt.Println("after2", trie.Hash())

	// tx2 should be the single transaction filtered out
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 1, count, "got %d filtered by TxOptions when there should be a single one", count)

	require.Equal(t, tx2, drops[0], "Got %x, expected %x", drops[0].Hash(), tx2.Hash())
}

func TestFilterTxConditionalBlockNumber(t *testing.T) {
	t.Parallel()

	// Create an in memory state db to test against.
	memDb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memDb)
	state, _ := state.New(common.Hash{}, db, nil)

	header := &types.Header{
		Number: big.NewInt(100),
	}

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
	drops := list.FilterTxConditional(state, header)

	count := len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Create another transaction with a block number option and add to the list.
	tx2 := transaction(1, 1000, key)

	var options types.OptionsPIP15

	options.BlockNumberMin = big.NewInt(90)
	options.BlockNumberMax = big.NewInt(110)

	tx2.PutOptions(&options)
	list.Add(tx2, DefaultConfig.PriceBump)

	// There should still be no drops as no state has been modified.
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Set block number that conflicts with tx2's policy
	header.Number = big.NewInt(120)

	// tx2 should be the single transaction filtered out
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 1, count, "got %d filtered by TxOptions when there should be a single one", count)

	require.Equal(t, tx2, drops[0], "Got %x, expected %x", drops[0].Hash(), tx2.Hash())
}

func TestFilterTxConditionalTimestamp(t *testing.T) {
	t.Parallel()

	// Create an in memory state db to test against.
	memDb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memDb)
	state, _ := state.New(common.Hash{}, db, nil)

	header := &types.Header{
		Number: big.NewInt(0),
		Time:   100,
	}

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
	drops := list.FilterTxConditional(state, header)

	count := len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Create another transaction with a timestamp option and add to the list.
	tx2 := transaction(1, 1000, key)

	var options types.OptionsPIP15

	minTimestamp := uint64(90)
	maxTimestamp := uint64(110)

	options.TimestampMin = &minTimestamp
	options.TimestampMax = &maxTimestamp

	tx2.PutOptions(&options)
	list.Add(tx2, DefaultConfig.PriceBump)

	// There should still be no drops as no state has been modified.
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 0, count, "got %d filtered by TxOptions when there should not be any", count)

	// Set timestamp that conflicts with tx2's policy
	header.Time = 120

	// tx2 should be the single transaction filtered out
	drops = list.FilterTxConditional(state, header)

	count = len(drops)
	require.Equal(t, 1, count, "got %d filtered by TxOptions when there should be a single one", count)

	require.Equal(t, tx2, drops[0], "Got %x, expected %x", drops[0].Hash(), tx2.Hash())
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
