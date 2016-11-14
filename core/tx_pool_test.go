// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

func transaction(nonce uint64, gaslimit *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, big.NewInt(1), nil).SignECDSA(types.HomesteadSigner{}, key)
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)

	key, _ := crypto.GenerateKey()
	newPool := NewTxPool(testChainConfig(), new(event.TypeMux), func() (*state.StateDB, error) { return statedb, nil }, func() *big.Int { return big.NewInt(1000000) })
	newPool.resetState()

	return newPool, key
}

func deriveSender(tx *types.Transaction) (common.Address, error) {
	return types.Sender(types.HomesteadSigner{}, tx)
}

func TestInvalidTransactions(t *testing.T) {
	pool, key := setupTxPool()

	tx := transaction(0, big.NewInt(100), key)
	if err := pool.Add(tx); err != ErrNonExistentAccount {
		t.Error("expected", ErrNonExistentAccount)
	}

	from, _ := deriveSender(tx)
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1))
	if err := pool.Add(tx); err != ErrInsufficientFunds {
		t.Error("expected", ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(tx.Gas(), tx.GasPrice()))
	currentState.AddBalance(from, balance)
	if err := pool.Add(tx); err != ErrIntrinsicGas {
		t.Error("expected", ErrIntrinsicGas, "got", err)
	}

	currentState.SetNonce(from, 1)
	currentState.AddBalance(from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, big.NewInt(100000), key)
	if err := pool.Add(tx); err != ErrNonce {
		t.Error("expected", ErrNonce)
	}

	tx = transaction(1, big.NewInt(100000), key)
	pool.minGasPrice = big.NewInt(1000)
	if err := pool.Add(tx); err != ErrCheap {
		t.Error("expected", ErrCheap, "got", err)
	}

	pool.SetLocal(tx)
	if err := pool.Add(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

func TestTransactionQueue(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(100), key)
	from, _ := deriveSender(tx)
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000))
	pool.enqueueTx(tx.Hash(), tx)

	pool.promoteExecutables()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction(1, big.NewInt(100), key)
	from, _ = deriveSender(tx)
	currentState.SetNonce(from, 2)
	pool.enqueueTx(tx.Hash(), tx)
	pool.promoteExecutables()
	if _, ok := pool.pending[from].txs.items[tx.Nonce()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

	if len(pool.queue) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue))
	}

	pool, key = setupTxPool()
	tx1 := transaction(0, big.NewInt(100), key)
	tx2 := transaction(10, big.NewInt(100), key)
	tx3 := transaction(11, big.NewInt(100), key)
	from, _ = deriveSender(tx1)
	currentState, _ = pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000))
	pool.enqueueTx(tx1.Hash(), tx1)
	pool.enqueueTx(tx2.Hash(), tx2)
	pool.enqueueTx(tx3.Hash(), tx3)

	pool.promoteExecutables()

	if len(pool.pending) != 1 {
		t.Error("expected tx pool to be 1, got", len(pool.pending))
	}
	if pool.queue[from].Len() != 2 {
		t.Error("expected len(queue) == 2, got", pool.queue[from].Len())
	}
}

func TestRemoveTx(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(100), key)
	from, _ := deriveSender(tx)
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1))

	pool.enqueueTx(tx.Hash(), tx)
	pool.promoteTx(from, tx.Hash(), tx)
	if len(pool.queue) != 1 {
		t.Error("expected queue to be 1, got", len(pool.queue))
	}
	if len(pool.pending) != 1 {
		t.Error("expected pending to be 1, got", len(pool.pending))
	}
	pool.Remove(tx.Hash())
	if len(pool.queue) > 0 {
		t.Error("expected queue to be 0, got", len(pool.queue))
	}
	if len(pool.pending) > 0 {
		t.Error("expected pending to be 0, got", len(pool.pending))
	}
}

func TestNegativeValue(t *testing.T) {
	pool, key := setupTxPool()

	tx, _ := types.NewTransaction(0, common.Address{}, big.NewInt(-1), big.NewInt(100), big.NewInt(1), nil).SignECDSA(types.HomesteadSigner{}, key)
	from, _ := deriveSender(tx)
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1))
	if err := pool.Add(tx); err != ErrNegativeValue {
		t.Error("expected", ErrNegativeValue, "got", err)
	}
}

func TestTransactionChainFork(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb, _ := state.New(common.Hash{}, db)
		pool.currentState = func() (*state.StateDB, error) { return statedb, nil }
		currentState, _ := pool.currentState()
		currentState.AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	tx := transaction(0, big.NewInt(100000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.RemoveBatch([]*types.Transaction{tx})

	// reset the pool's internal state
	resetState()
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestTransactionDoubleNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb, _ := state.New(common.Hash{}, db)
		pool.currentState = func() (*state.StateDB, error) { return statedb, nil }
		currentState, _ := pool.currentState()
		currentState.AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	signer := types.HomesteadSigner{}
	tx1, _ := types.NewTransaction(0, common.Address{}, big.NewInt(100), big.NewInt(100000), big.NewInt(1), nil).SignECDSA(signer, key)
	tx2, _ := types.NewTransaction(0, common.Address{}, big.NewInt(100), big.NewInt(1000000), big.NewInt(2), nil).SignECDSA(signer, key)
	tx3, _ := types.NewTransaction(0, common.Address{}, big.NewInt(100), big.NewInt(1000000), big.NewInt(1), nil).SignECDSA(signer, key)

	// Add the first two transaction, ensure higher priced stays only
	if err := pool.add(tx1); err != nil {
		t.Error("didn't expect error", err)
	}
	if err := pool.add(tx2); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.promoteExecutables()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	// Add the thid transaction and ensure it's not saved (smaller price)
	if err := pool.add(tx3); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.promoteExecutables()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	// Ensure the total transaction count is correct
	if len(pool.all) != 1 {
		t.Error("expected 1 total transactions, got", len(pool.all))
	}
}

func TestMissingNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	currentState, _ := pool.currentState()
	currentState.AddBalance(addr, big.NewInt(100000000000000))
	tx := transaction(1, big.NewInt(100000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
	if pool.queue[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.queue[addr].Len())
	}
	if len(pool.all) != 1 {
		t.Error("expected 1 total transactions, got", len(pool.all))
	}
}

func TestNonceRecovery(t *testing.T) {
	const n = 10
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	currentState, _ := pool.currentState()
	currentState.SetNonce(addr, n)
	currentState.AddBalance(addr, big.NewInt(100000000000000))
	pool.resetState()
	tx := transaction(n, big.NewInt(100000), key)
	if err := pool.Add(tx); err != nil {
		t.Error(err)
	}
	// simulate some weird re-order of transactions and missing nonce(s)
	currentState.SetNonce(addr, n-1)
	pool.resetState()
	if fn := pool.pendingState.GetNonce(addr); fn != n+1 {
		t.Errorf("expected nonce to be %d, got %d", n+1, fn)
	}
}

func TestRemovedTxEvent(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(1000000), key)
	from, _ := deriveSender(tx)
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000000000000))
	pool.eventMux.Post(RemovedTransactionEvent{types.Transactions{tx}})
	pool.eventMux.Post(ChainHeadEvent{nil})
	if pool.pending[from].Len() != 1 {
		t.Error("expected 1 pending tx, got", pool.pending[from].Len())
	}
	if len(pool.all) != 1 {
		t.Error("expected 1 total transactions, got", len(pool.all))
	}
}

// Tests that if an account runs out of funds, any pending and queued transactions
// are dropped.
func TestTransactionDropping(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000))

	// Add some pending and some queued transactions
	var (
		tx0  = transaction(0, big.NewInt(100), key)
		tx1  = transaction(1, big.NewInt(200), key)
		tx10 = transaction(10, big.NewInt(100), key)
		tx11 = transaction(11, big.NewInt(200), key)
	)
	pool.promoteTx(account, tx0.Hash(), tx0)
	pool.promoteTx(account, tx1.Hash(), tx1)
	pool.enqueueTx(tx10.Hash(), tx10)
	pool.enqueueTx(tx11.Hash(), tx11)

	// Check that pre and post validations leave the pool as is
	if pool.pending[account].Len() != 2 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 2)
	}
	if pool.queue[account].Len() != 2 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 2)
	}
	if len(pool.all) != 4 {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), 4)
	}
	pool.resetState()
	if pool.pending[account].Len() != 2 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 2)
	}
	if pool.queue[account].Len() != 2 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 2)
	}
	if len(pool.all) != 4 {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), 4)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	state.AddBalance(account, big.NewInt(-750))
	pool.resetState()

	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
	if _, ok := pool.queue[account].txs.items[tx10.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx11.Nonce()]; ok {
		t.Errorf("out-of-fund queued transaction present: %v", tx11)
	}
	if len(pool.all) != 2 {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), 2)
	}
}

// Tests that if a transaction is dropped from the current pending pool (e.g. out
// of fund), all consecutive (still valid, but not executable) transactions are
// postponed back into the future queue to prevent broadcasting them.
func TestTransactionPostponing(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000))

	// Add a batch consecutive pending transactions for validation
	txns := []*types.Transaction{}
	for i := 0; i < 100; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = transaction(uint64(i), big.NewInt(100), key)
		} else {
			tx = transaction(uint64(i), big.NewInt(500), key)
		}
		pool.promoteTx(account, tx.Hash(), tx)
		txns = append(txns, tx)
	}
	// Check that pre and post validations leave the pool as is
	if pool.pending[account].Len() != len(txns) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), len(txns))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 0)
	}
	if len(pool.all) != len(txns) {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), len(txns))
	}
	pool.resetState()
	if pool.pending[account].Len() != len(txns) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), len(txns))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 0)
	}
	if len(pool.all) != len(txns) {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), len(txns))
	}
	// Reduce the balance of the account, and check that transactions are reorganised
	state.AddBalance(account, big.NewInt(-750))
	pool.resetState()

	if _, ok := pool.pending[account].txs.items[txns[0].Nonce()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txns[0])
	}
	if _, ok := pool.queue[account].txs.items[txns[0].Nonce()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txns[0])
	}
	for i, tx := range txns[1:] {
		if i%2 == 1 {
			if _, ok := pool.pending[account].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: valid but future transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[account].txs.items[tx.Nonce()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", i+1, tx)
			}
		} else {
			if _, ok := pool.pending[account].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[account].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", i+1, tx)
			}
		}
	}
	if len(pool.all) != len(txns)/2 {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), len(txns)/2)
	}
}

// Tests that if the transaction count belonging to a single account goes above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestTransactionQueueAccountLimiting(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= maxQueuedPerAccount+5; i++ {
		if err := pool.Add(transaction(i, big.NewInt(100000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
		if i <= maxQueuedPerAccount {
			if pool.queue[account].Len() != int(i) {
				t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), i)
			}
		} else {
			if pool.queue[account].Len() != int(maxQueuedPerAccount) {
				t.Errorf("tx %d: queue limit mismatch: have %d, want %d", i, pool.queue[account].Len(), maxQueuedPerAccount)
			}
		}
	}
	if len(pool.all) != int(maxQueuedPerAccount) {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), maxQueuedPerAccount)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestTransactionQueueGlobalLimiting(t *testing.T) {
	// Reduce the queue limits to shorten test time
	defer func(old uint64) { maxQueuedInTotal = old }(maxQueuedInTotal)
	maxQueuedInTotal = maxQueuedPerAccount * 3

	// Create the pool to test the limit enforcement with
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)

	pool := NewTxPool(testChainConfig(), new(event.TypeMux), func() (*state.StateDB, error) { return statedb, nil }, func() *big.Int { return big.NewInt(1000000) })
	pool.resetState()

	// Create a number of test accounts and fund them
	state, _ := pool.currentState()

	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		state.AddBalance(crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := make(types.Transactions, 0, 3*maxQueuedInTotal)
	for len(txs) < cap(txs) {
		key := keys[rand.Intn(len(keys))]
		addr := crypto.PubkeyToAddress(key.PublicKey)

		txs = append(txs, transaction(nonces[addr]+1, big.NewInt(100000), key))
		nonces[addr]++
	}
	// Import the batch and verify that limits have been enforced
	pool.AddBatch(txs)

	queued := 0
	for addr, list := range pool.queue {
		if list.Len() > int(maxQueuedPerAccount) {
			t.Errorf("addr %x: queued accounts overflown allowance: %d > %d", addr, list.Len(), maxQueuedPerAccount)
		}
		queued += list.Len()
	}
	if queued > int(maxQueuedInTotal) {
		t.Fatalf("total transactions overflow allowance: %d > %d", queued, maxQueuedInTotal)
	}
}

// Tests that if an account remains idle for a prolonged amount of time, any
// non-executable transactions queued up are dropped to prevent wasting resources
// on shuffling them around.
func TestTransactionQueueTimeLimiting(t *testing.T) {
	// Reduce the queue limits to shorten test time
	defer func(old time.Duration) { maxQueuedLifetime = old }(maxQueuedLifetime)
	defer func(old time.Duration) { evictionInterval = old }(evictionInterval)
	maxQueuedLifetime = time.Second
	evictionInterval = time.Second

	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	// Queue up a batch of transactions
	for i := uint64(1); i <= maxQueuedPerAccount; i++ {
		if err := pool.Add(transaction(i, big.NewInt(100000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
	}
	// Wait until at least two expiration cycles hit and make sure the transactions are gone
	time.Sleep(2 * evictionInterval)
	if len(pool.queue) > 0 {
		t.Fatalf("old transactions remained after eviction")
	}
}

// Tests that even if the transaction count belonging to a single account goes
// above some threshold, as long as the transactions are executable, they are
// accepted.
func TestTransactionPendingLimiting(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < maxQueuedPerAccount+5; i++ {
		if err := pool.Add(transaction(i, big.NewInt(100000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if pool.pending[account].Len() != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, pool.pending[account].Len(), i+1)
		}
		if len(pool.queue) != 0 {
			t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), 0)
		}
	}
	if len(pool.all) != int(maxQueuedPerAccount+5) {
		t.Errorf("total transaction mismatch: have %d, want %d", len(pool.all), maxQueuedPerAccount+5)
	}
}

// Tests that the transaction limits are enforced the same way irrelevant whether
// the transactions are added one by one or in batches.
func TestTransactionQueueLimitingEquivalency(t *testing.T)   { testTransactionLimitingEquivalency(t, 1) }
func TestTransactionPendingLimitingEquivalency(t *testing.T) { testTransactionLimitingEquivalency(t, 0) }

func testTransactionLimitingEquivalency(t *testing.T, origin uint64) {
	// Add a batch of transactions to a pool one by one
	pool1, key1 := setupTxPool()
	account1, _ := deriveSender(transaction(0, big.NewInt(0), key1))
	state1, _ := pool1.currentState()
	state1.AddBalance(account1, big.NewInt(1000000))

	for i := uint64(0); i < maxQueuedPerAccount+5; i++ {
		if err := pool1.Add(transaction(origin+i, big.NewInt(100000), key1)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
	}
	// Add a batch of transactions to a pool in one big batch
	pool2, key2 := setupTxPool()
	account2, _ := deriveSender(transaction(0, big.NewInt(0), key2))
	state2, _ := pool2.currentState()
	state2.AddBalance(account2, big.NewInt(1000000))

	txns := []*types.Transaction{}
	for i := uint64(0); i < maxQueuedPerAccount+5; i++ {
		txns = append(txns, transaction(origin+i, big.NewInt(100000), key2))
	}
	pool2.AddBatch(txns)

	// Ensure the batch optimization honors the same pool mechanics
	if len(pool1.pending) != len(pool2.pending) {
		t.Errorf("pending transaction count mismatch: one-by-one algo: %d, batch algo: %d", len(pool1.pending), len(pool2.pending))
	}
	if len(pool1.queue) != len(pool2.queue) {
		t.Errorf("queued transaction count mismatch: one-by-one algo: %d, batch algo: %d", len(pool1.queue), len(pool2.queue))
	}
	if len(pool1.all) != len(pool2.all) {
		t.Errorf("total transaction count mismatch: one-by-one algo %d, batch algo %d", len(pool1.all), len(pool2.all))
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, the higher transactions are dropped to prevent DOS
// attacks.
func TestTransactionPendingGlobalLimiting(t *testing.T) {
	// Reduce the queue limits to shorten test time
	defer func(old uint64) { maxPendingTotal = old }(maxPendingTotal)
	maxPendingTotal = minPendingPerAccount * 10

	// Create the pool to test the limit enforcement with
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)

	pool := NewTxPool(testChainConfig(), new(event.TypeMux), func() (*state.StateDB, error) { return statedb, nil }, func() *big.Int { return big.NewInt(1000000) })
	pool.resetState()

	// Create a number of test accounts and fund them
	state, _ := pool.currentState()

	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		state.AddBalance(crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(maxPendingTotal)/len(keys)*2; j++ {
			txs = append(txs, transaction(nonces[addr], big.NewInt(100000), key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddBatch(txs)

	pending := 0
	for _, list := range pool.pending {
		pending += list.Len()
	}
	if pending > int(maxPendingTotal) {
		t.Fatalf("total pending transactions overflow allowance: %d > %d", pending, maxPendingTotal)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, if they are under the minimum guaranteed slot count then
// the transactions are still kept.
func TestTransactionPendingMinimumAllowance(t *testing.T) {
	// Reduce the queue limits to shorten test time
	defer func(old uint64) { maxPendingTotal = old }(maxPendingTotal)
	maxPendingTotal = 0

	// Create the pool to test the limit enforcement with
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)

	pool := NewTxPool(testChainConfig(), new(event.TypeMux), func() (*state.StateDB, error) { return statedb, nil }, func() *big.Int { return big.NewInt(1000000) })
	pool.resetState()

	// Create a number of test accounts and fund them
	state, _ := pool.currentState()

	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		state.AddBalance(crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(minPendingPerAccount)*2; j++ {
			txs = append(txs, transaction(nonces[addr], big.NewInt(100000), key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddBatch(txs)

	for addr, list := range pool.pending {
		if list.Len() != int(minPendingPerAccount) {
			t.Errorf("addr %x: total pending transactions mismatch: have %d, want %d", addr, list.Len(), minPendingPerAccount)
		}
	}
}

// Benchmarks the speed of validating the contents of the pending queue of the
// transaction pool.
func BenchmarkPendingDemotion100(b *testing.B)   { benchmarkPendingDemotion(b, 100) }
func BenchmarkPendingDemotion1000(b *testing.B)  { benchmarkPendingDemotion(b, 1000) }
func BenchmarkPendingDemotion10000(b *testing.B) { benchmarkPendingDemotion(b, 10000) }

func benchmarkPendingDemotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(i), big.NewInt(100000), key)
		pool.promoteTx(account, tx.Hash(), tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.demoteUnexecutables()
	}
}

// Benchmarks the speed of scheduling the contents of the future queue of the
// transaction pool.
func BenchmarkFuturePromotion100(b *testing.B)   { benchmarkFuturePromotion(b, 100) }
func BenchmarkFuturePromotion1000(b *testing.B)  { benchmarkFuturePromotion(b, 1000) }
func BenchmarkFuturePromotion10000(b *testing.B) { benchmarkFuturePromotion(b, 10000) }

func benchmarkFuturePromotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(1+i), big.NewInt(100000), key)
		pool.enqueueTx(tx.Hash(), tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.promoteExecutables()
	}
}

// Benchmarks the speed of iterative transaction insertion.
func BenchmarkPoolInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	txs := make(types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		txs[i] = transaction(uint64(i), big.NewInt(100000), key)
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for _, tx := range txs {
		pool.Add(tx)
	}
}

// Benchmarks the speed of batched transaction insertion.
func BenchmarkPoolBatchInsert100(b *testing.B)   { benchmarkPoolBatchInsert(b, 100) }
func BenchmarkPoolBatchInsert1000(b *testing.B)  { benchmarkPoolBatchInsert(b, 1000) }
func BenchmarkPoolBatchInsert10000(b *testing.B) { benchmarkPoolBatchInsert(b, 10000) }

func benchmarkPoolBatchInsert(b *testing.B, size int) {
	// Generate a batch of transactions to enqueue into the pool
	pool, key := setupTxPool()
	account, _ := deriveSender(transaction(0, big.NewInt(0), key))
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	batches := make([]types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		batches[i] = make(types.Transactions, size)
		for j := 0; j < size; j++ {
			batches[i][j] = transaction(uint64(size*i+j), big.NewInt(100000), key)
		}
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for _, batch := range batches {
		pool.AddBatch(batch)
	}
}
