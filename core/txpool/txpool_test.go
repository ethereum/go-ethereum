// Copyright 2021 The go-ethereum Authors
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

package txpool

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

func TestInvalidTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)

	pool.currentState.AddBalance(from, big.NewInt(1))
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrInsufficientFunds) {
		t.Error("expected", core.ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice()))
	pool.currentState.AddBalance(from, balance)
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrIntrinsicGas) {
		t.Error("expected", core.ErrIntrinsicGas)
	}

	pool.currentState.SetNonce(from, 1)
	pool.currentState.AddBalance(from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, 100000, key)
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrNonceTooLow) {
		t.Error("expected", core.ErrNonceTooLow)
	}
	// Test negative value
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(-1), 100, big.NewInt(1), nil), types.HomesteadSigner{}, key)
	from2, _ := deriveSender(tx2)
	pool.currentState.AddBalance(from2, big.NewInt(1))
	if err := pool.AddRemotes([]*types.Transaction{tx2}); err[0] != core.ErrNegativeValue {
		t.Error("expected", core.ErrNegativeValue, "got", err[0])
	}

	tx = transaction(1, 100000, key)
	pool.config.minGasPrice = big.NewInt(1000)
	if err := pool.AddRemotes([]*types.Transaction{tx}); err[0] != core.ErrUnderpriced {
		t.Error("expected", core.ErrUnderpriced)
	}
	if err := pool.AddLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
	validateTxPoolInternals(pool)
}

func TestTransactionMissingNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(100000000000000))
	tx := transaction(1, 100000, key)
	if err := pool.AddRemotes([]*types.Transaction{tx}); err[0] != nil {
		t.Error("didn't expect error", err)
	}
	if pool.remoteTxs.Len() != 0 {
		t.Error("expected 0 pending transactions, got", pool.remoteTxs.Len())
	}
	if pool.gappedTxs[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.gappedTxs[addr].Len())
	}
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
	validateTxPoolInternals(pool)
}

// Tests that the pool rejects duplicate transactions.
func TestTransactionDeduplication(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 1000000, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	pool.currentState.AddBalance(crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Create a batch of transactions and add a few of them
	txs := make([]*types.Transaction, 16)
	for i := 0; i < len(txs); i++ {
		txs[i] = pricedTransaction(uint64(i), 100000, big.NewInt(1), key)
	}
	var firsts []*types.Transaction
	for i := 0; i < len(txs); i += 2 {
		firsts = append(firsts, txs[i])
	}
	errs := pool.AddRemotesSync(firsts)
	if len(errs) != len(firsts) {
		t.Fatalf("first add mismatching result count: have %d, want %d", len(errs), len(firsts))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued := pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != len(txs)/2-1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, len(txs)/2-1)
	}
	// Try to add all of them now and ensure previous ones error out as knowns
	errs = pool.AddRemotesSync(txs)
	if len(errs) != len(txs) {
		t.Fatalf("all add mismatching result count: have %d, want %d", len(errs), len(txs))
	}
	for i, err := range errs {
		if i%2 == 0 && err == nil {
			t.Errorf("add %d succeeded, should have failed as known", i)
		}
		if i%2 == 1 && err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued = pool.Stats()
	if pending != len(txs) {
		_, remote := pool.PendingBlock()
		for _, t := range remote {
			fmt.Printf("%v\n", t.Nonce())
		}
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, len(txs))
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement transactions that don't meet the minimum
// price bump required.
func TestTransactionReplacement(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 1000000, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	pool.currentState.AddBalance(crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	price := int64(100)
	threshold := (price * (100 + int64(testTxPoolConfig.PriceBump))) / 100

	if err := pool.AddRemotesSync([]*types.Transaction{pricedTransaction(0, 100000, big.NewInt(1), key)}); err[0] != nil {
		t.Fatalf("failed to add original cheap pending transaction: %v", err[0])
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(0, 100001, big.NewInt(1), key)}); err[0] != core.ErrReplaceUnderpriced {
		t.Fatalf("original cheap pending transaction replacement error mismatch: have %v, want %v", err[0], core.ErrReplaceUnderpriced)
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(0, 100000, big.NewInt(2), key)}); err[0] != nil {
		t.Fatalf("failed to replace original cheap pending transaction: %v", err[0])
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("cheap replacement event firing failed: %v", err)
	}

	if err := pool.AddRemotesSync([]*types.Transaction{pricedTransaction(0, 100000, big.NewInt(price), key)}); err[0] != nil {
		t.Fatalf("failed to add original proper pending transaction: %v", err[0])
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(0, 100001, big.NewInt(threshold-1), key)}); err[0] != core.ErrReplaceUnderpriced {
		t.Fatalf("original proper pending transaction replacement error mismatch: have %v, want %v", err[0], core.ErrReplaceUnderpriced)
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(0, 100000, big.NewInt(threshold+1), key)}); err[0] != nil {
		t.Fatalf("failed to replace original proper pending transaction: %v", err[0])
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("proper replacement event firing failed: %v", err)
	}

	// Add queued transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100000, big.NewInt(1), key)}); err[0] != nil {
		t.Fatalf("failed to add original cheap queued transaction: %v", err[0])
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100001, big.NewInt(1), key)}); err[0] != core.ErrReplaceUnderpriced {
		t.Fatalf("original cheap queued transaction replacement error mismatch: have %v, want %v", err[0], core.ErrReplaceUnderpriced)
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100000, big.NewInt(2), key)}); err[0] != nil {
		t.Fatalf("failed to replace original cheap queued transaction: %v", err[0])
	}

	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100000, big.NewInt(price), key)}); err[0] != nil {
		t.Fatalf("failed to add original proper queued transaction: %v", err[0])
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100001, big.NewInt(threshold-1), key)}); err[0] != core.ErrReplaceUnderpriced {
		t.Fatalf("original proper queued transaction replacement error mismatch: have %v, want %v", err[0], core.ErrReplaceUnderpriced)
	}
	if err := pool.AddRemotes([]*types.Transaction{pricedTransaction(2, 100000, big.NewInt(threshold+1), key)}); err[0] != nil {
		t.Fatalf("failed to replace original proper queued transaction: %v", err[0])
	}

	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("queued replacement event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

func TestTransactionQueue(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)
	pool.currentState.AddBalance(from, big.NewInt(100000))
	pool.runReorg(nil, nil)

	entry1, err := pool.txToTxEntry(tx)
	if err != nil {
		t.Fatal(err)
	}
	pool.addGapped(entry1, false)
	if pending, queued := pool.Stats(); pending != 0 || queued != 1 {
		t.Errorf("expected valid txs to be 1 is %v , pending is %v", queued, pending)
	}

	tx = transaction(1, 21000, key)
	entry, err := pool.txToTxEntry(tx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := pool.add(entry, false); err != nil {
		t.Fatal(err)
	}

	// Request promotion
	pool.addUngappedTx(0, from)

	var found bool
	pool.remoteTxs.Delete(func(entry *txEntry) bool {
		if entry.tx.Nonce() == tx.Nonce() {
			found = true
		}
		return false
	})

	if !found {
		t.Error("expected transaction to be in tx pool")
	}
	if pending, queued := pool.Stats(); pending != 2 || queued != 0 {
		t.Errorf("expected valid txs to be 2 is %v , queued is %v", pending, queued)
	}
}

func TestTransactionQueue2(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	txs := []*types.Transaction{
		transaction(0, 100, key),
		transaction(10, 100, key),
		transaction(11, 100, key),
	}
	from, _ := deriveSender(txs[0])
	pool.currentState.AddBalance(from, big.NewInt(1000))
	pool.reset(nil, nil)

	for _, tx := range txs {
		entry, err := pool.txToTxEntry(tx)
		if err != nil {
			t.Fatal(err)
		}
		pool.addGapped(entry, false)
	}

	pool.addUngappedTx(0, from)

	pending, queued := pool.Stats()
	if pending != 1 {
		t.Error("expected pending length to be 1, got", pending)
	}
	if queued != 2 {
		t.Error("expected len(queue) == 2, got", queued)
	}
}

func TestTransactionChainFork(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		statedb.AddBalance(addr, big.NewInt(10000000000000000))

		pool.chain = &testBlockChain{statedb, 1000000, new(event.Feed)}
	}
	resetState()
	tx, _ := pool.txToTxEntry(transaction(0, 100000, key))
	if _, _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.removeTx(tx.tx.Hash())

	// reset the pool's internal state
	resetState()
	if _, _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}

}

func TestStateChangeDuringTransactionPoolReset(t *testing.T) {
	t.Parallel()

	var (
		key, _     = crypto.GenerateKey()
		address    = crypto.PubkeyToAddress(key.PublicKey)
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		trigger    = false
	)

	// setup pool with 2 transaction in it
	statedb.SetBalance(address, new(big.Int).SetUint64(params.Ether))
	blockchain := &testChain{&testBlockChain{statedb, 1000000000, new(event.Feed)}, address, &trigger}

	tx0 := transaction(0, 100000, key)
	tx1 := transaction(1, 100000, key)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	nonce := pool.Nonce(address)
	if nonce != 0 {
		t.Fatalf("Invalid nonce, want 0, got %d", nonce)
	}

	pool.AddRemotesSync([]*types.Transaction{tx0, tx1})

	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}

	// trigger state change in the background
	trigger = true
	pool.runReorg(nil, nil)

	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}
}

// TestPruning adds enough transactions to trigger the pruning,
// then removes transactions until the unpruning is triggered.
func TestPruning(t *testing.T) {
	// Generate a batch of transactions to enqueue into the pool
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 10000000, new(event.Feed)}

	key, _ := crypto.GenerateKey()
	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)

	defer pool.Stop()
	size := pool.config.MaxTxCount * 2
	pool.config.pendingBlockSize = uint64(size)
	account := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(account, big.NewInt(1000000))

	batch := make(types.Transactions, size)
	for j := 0; j < size; j++ {
		batch[j] = transaction(uint64(j), 100000, key)
	}
	for _, err := range pool.AddRemotesSync(batch) {
		if err != nil {
			t.Error(err)
		}
	}
	// Pruning should be triggered right now
	time.Sleep(500 * time.Millisecond)
	// Pending block triggers unpruning, but we have enough tx's in memory already.
	_, remotes := pool.PendingBlock()
	if len(remotes) != 750 {
		t.Errorf("Not enough tx retrieved, got: %v want: %v", len(remotes), 750)
	}
	// Delete the remotes from the pool
	for _, t := range remotes {
		from, _ := types.Sender(pool.signer, t)
		statedb.SetNonce(from, t.Nonce()+1)
	}
	pool.runReorg(nil, nil)
	time.Sleep(500 * time.Millisecond)
	// Pending block triggers unpruning, but we have enough tx's in memory already.
	_, remotes = pool.PendingBlock()
	if len(remotes) != 750 {
		t.Errorf("Not enough tx retrieved, got: %v want: %v", len(remotes), 750)
	}
	// Delete the remotes from the pool
	for _, t := range remotes {
		from, _ := types.Sender(pool.signer, t)
		statedb.SetNonce(from, t.Nonce()+1)
	}
	pool.runReorg(nil, nil)
	time.Sleep(500 * time.Millisecond)
	// Pending block triggers unpruning, but we have enough tx's in memory already.
	_, remotes = pool.PendingBlock()
	if len(remotes) != size-2*750 {
		t.Errorf("Not enough tx retrieved, got: %v want: %v", len(remotes), size-2*750)
	}
	validateTxPoolInternals(pool)
}

// validateTxPoolInternals checks various consistency invariants within the pool.
func validateTxPoolInternals(pool *TxPool) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Ensure the total transaction set is consistent with pending + queued
	pending, queued := pool.Stats()
	if total := pool.all.Count(); total != pending+queued {
		return fmt.Errorf("total transaction count %d != %d pending + %d queued", total, pending, queued)
	}

	// Ensure the next nonce to assign is the correct one
	highestNonce := make(map[common.Address]uint64)
	remotes := pool.remoteTxs.Peek(pool.remoteTxs.Len())
	for _, tx := range remotes {
		// Find the last transaction
		sender, err := types.Sender(types.HomesteadSigner{}, tx)
		if err != nil {
			panic(err)
		}
		if highestNonce[sender] < tx.Nonce() {
			highestNonce[sender] = tx.Nonce()
		}
	}
	for addr, last := range highestNonce {
		if nonce := pool.pendingNonces.get(addr); nonce != last+1 {
			return fmt.Errorf("pending nonce mismatch: have %v, want %v", nonce, last+1)
		}
	}
	return nil
}

// validateEvents checks that the correct number of transaction addition events
// were fired on the pool's event feed.
func validateEvents(events chan core.NewTxsEvent, count int) error {
	var received []*types.Transaction

	for len(received) < count {
		select {
		case ev := <-events:
			received = append(received, ev.Txs...)
		case <-time.After(time.Second):
			return fmt.Errorf("event #%d not fired", len(received))
		}
	}
	if len(received) > count {
		return fmt.Errorf("more than %d events fired: %v", count, received[count:])
	}
	select {
	case ev := <-events:
		return fmt.Errorf("more than %d events fired: %v", count, ev.Txs)

	case <-time.After(50 * time.Millisecond):
		// This branch should be "default", but it's a data race between goroutines,
		// reading the event channel and pushing into it, so better wait a bit ensuring
		// really nothing gets injected.
	}
	return nil
}

func deriveSender(tx *types.Transaction) (common.Address, error) {
	return types.Sender(types.HomesteadSigner{}, tx)
}

type testChain struct {
	*testBlockChain
	address common.Address
	trigger *bool
}

// testChain.State() is used multiple times to reset the pending state.
// when simulate is true it will create a state that indicates
// that tx0 and tx1 are included in the chain.
func (c *testChain) State() (*state.StateDB, error) {
	// delay "state change" by one. The tx pool fetches the
	// state multiple times and by delaying it a bit we simulate
	// a state change between those fetches.
	stdb := c.statedb
	if *c.trigger {
		c.statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		// simulate that the new head block included tx0 and tx1
		c.statedb.SetNonce(c.address, 2)
		c.statedb.SetBalance(c.address, new(big.Int).SetUint64(params.Ether))
		*c.trigger = false
	}
	return stdb, nil
}
