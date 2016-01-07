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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

func transaction(nonce uint64, gaslimit *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, big.NewInt(1), nil).SignECDSA(key)
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)

	var m event.TypeMux
	key, _ := crypto.GenerateKey()
	newPool := NewTxPool(&m, func() (*state.StateDB, error) { return statedb, nil }, func() *big.Int { return big.NewInt(1000000) })
	newPool.resetState()
	return newPool, key
}

func TestInvalidTransactions(t *testing.T) {
	pool, key := setupTxPool()

	tx := transaction(0, big.NewInt(100), key)
	if err := pool.Add(tx); err != ErrNonExistentAccount {
		t.Error("expected", ErrNonExistentAccount)
	}

	from, _ := tx.From()
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
	from, _ := tx.From()
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000))
	pool.queueTx(tx.Hash(), tx)

	pool.checkQueue()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction(1, big.NewInt(100), key)
	from, _ = tx.From()
	currentState.SetNonce(from, 2)
	pool.queueTx(tx.Hash(), tx)
	pool.checkQueue()
	if _, ok := pool.pending[tx.Hash()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

	if len(pool.queue[from]) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
	}

	pool, key = setupTxPool()
	tx1 := transaction(0, big.NewInt(100), key)
	tx2 := transaction(10, big.NewInt(100), key)
	tx3 := transaction(11, big.NewInt(100), key)
	from, _ = tx1.From()
	currentState, _ = pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000))
	pool.queueTx(tx1.Hash(), tx1)
	pool.queueTx(tx2.Hash(), tx2)
	pool.queueTx(tx3.Hash(), tx3)

	pool.checkQueue()

	if len(pool.pending) != 1 {
		t.Error("expected tx pool to be 1, got", len(pool.pending))
	}
	if len(pool.queue[from]) != 2 {
		t.Error("expected len(queue) == 2, got", len(pool.queue[from]))
	}
}

func TestRemoveTx(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(100), key)
	from, _ := tx.From()
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1))
	pool.queueTx(tx.Hash(), tx)
	pool.addTx(tx.Hash(), from, tx)
	if len(pool.queue) != 1 {
		t.Error("expected queue to be 1, got", len(pool.queue))
	}

	if len(pool.pending) != 1 {
		t.Error("expected txs to be 1, got", len(pool.pending))
	}

	pool.RemoveTx(tx.Hash())

	if len(pool.queue) > 0 {
		t.Error("expected queue to be 0, got", len(pool.queue))
	}

	if len(pool.pending) > 0 {
		t.Error("expected txs to be 0, got", len(pool.pending))
	}
}

func TestNegativeValue(t *testing.T) {
	pool, key := setupTxPool()

	tx, _ := types.NewTransaction(0, common.Address{}, big.NewInt(-1), big.NewInt(100), big.NewInt(1), nil).SignECDSA(key)
	from, _ := tx.From()
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
	pool.RemoveTransactions([]*types.Transaction{tx})

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

	tx := transaction(0, big.NewInt(100000), key)
	tx2 := transaction(0, big.NewInt(1000000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	if err := pool.add(tx2); err != nil {
		t.Error("didn't expect error", err)
	}

	pool.checkQueue()
	if len(pool.pending) != 2 {
		t.Error("expected 2 pending txs. Got", len(pool.pending))
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
	if len(pool.queue[addr]) != 1 {
		t.Error("expected 1 queued transaction, got", len(pool.queue[addr]))
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
	from, _ := tx.From()
	currentState, _ := pool.currentState()
	currentState.AddBalance(from, big.NewInt(1000000000000))
	pool.eventMux.Post(RemovedTransactionEvent{types.Transactions{tx}})
	pool.eventMux.Post(ChainHeadEvent{nil})
	if len(pool.pending) != 1 {
		t.Error("expected 1 pending tx, got", len(pool.pending))
	}
}

// Tests that if an account runs out of funds, any pending and queued transactions
// are dropped.
func TestTransactionDropping(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000))

	// Add some pending and some queued transactions
	var (
		tx0  = transaction(0, big.NewInt(100), key)
		tx1  = transaction(1, big.NewInt(200), key)
		tx10 = transaction(10, big.NewInt(100), key)
		tx11 = transaction(11, big.NewInt(200), key)
	)
	pool.addTx(tx0.Hash(), account, tx0)
	pool.addTx(tx1.Hash(), account, tx1)
	pool.queueTx(tx10.Hash(), tx10)
	pool.queueTx(tx11.Hash(), tx11)

	// Check that pre and post validations leave the pool as is
	if len(pool.pending) != 2 {
		t.Errorf("pending transaction mismatch: have %d, want %d", len(pool.pending), 2)
	}
	if len(pool.queue[account]) != 2 {
		t.Errorf("queued transaction mismatch: have %d, want %d", len(pool.queue), 2)
	}
	pool.resetState()
	if len(pool.pending) != 2 {
		t.Errorf("pending transaction mismatch: have %d, want %d", len(pool.pending), 2)
	}
	if len(pool.queue[account]) != 2 {
		t.Errorf("queued transaction mismatch: have %d, want %d", len(pool.queue), 2)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	state.AddBalance(account, big.NewInt(-750))
	pool.resetState()

	if _, ok := pool.pending[tx0.Hash()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[tx1.Hash()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
	if _, ok := pool.queue[account][tx10.Hash()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account][tx11.Hash()]; ok {
		t.Errorf("out-of-fund queued transaction present: %v", tx11)
	}
}

// Tests that if a transaction is dropped from the current pending pool (e.g. out
// of fund), all consecutive (still valid, but not executable) transactions are
// postponed back into the future queue to prevent broadcating them.
func TestTransactionPostponing(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()

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
		pool.addTx(tx.Hash(), account, tx)
		txns = append(txns, tx)
	}
	// Check that pre and post validations leave the pool as is
	if len(pool.pending) != len(txns) {
		t.Errorf("pending transaction mismatch: have %d, want %d", len(pool.pending), len(txns))
	}
	if len(pool.queue[account]) != 0 {
		t.Errorf("queued transaction mismatch: have %d, want %d", len(pool.queue), 0)
	}
	pool.resetState()
	if len(pool.pending) != len(txns) {
		t.Errorf("pending transaction mismatch: have %d, want %d", len(pool.pending), len(txns))
	}
	if len(pool.queue[account]) != 0 {
		t.Errorf("queued transaction mismatch: have %d, want %d", len(pool.queue), 0)
	}
	// Reduce the balance of the account, and check that transactions are reorganized
	state.AddBalance(account, big.NewInt(-750))
	pool.resetState()

	if _, ok := pool.pending[txns[0].Hash()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txns[0])
	}
	if _, ok := pool.queue[account][txns[0].Hash()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txns[0])
	}
	for i, tx := range txns[1:] {
		if i%2 == 1 {
			if _, ok := pool.pending[tx.Hash()]; ok {
				t.Errorf("tx %d: valid but future transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[account][tx.Hash()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", i+1, tx)
			}
		} else {
			if _, ok := pool.pending[tx.Hash()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[account][tx.Hash()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", i+1, tx)
			}
		}
	}
}

// Tests that if the transaction count belonging to a single account goes above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestTransactionQueueLimiting(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= maxQueued+5; i++ {
		if err := pool.Add(transaction(i, big.NewInt(100000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
		if i <= maxQueued {
			if len(pool.queue[account]) != int(i) {
				t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, len(pool.queue[account]), i)
			}
		} else {
			if len(pool.queue[account]) != maxQueued {
				t.Errorf("tx %d: queue limit mismatch: have %d, want %d", i, len(pool.queue[account]), maxQueued)
			}
		}
	}
}

// Tests that even if the transaction count belonging to a single account goes
// above some threshold, as long as the transactions are executable, they are
// accepted.
func TestTransactionPendingLimiting(t *testing.T) {
	// Create a test account and fund it
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()

	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < maxQueued+5; i++ {
		if err := pool.Add(transaction(i, big.NewInt(100000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if len(pool.pending) != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), i+1)
		}
		if len(pool.queue[account]) != 0 {
			t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, len(pool.queue[account]), 0)
		}
	}
}

// Tests that the transaction limits are enforced the same way irrelevant whether
// the transactions are added one by one or in batches.
func TestTransactionQueueLimitingEquivalency(t *testing.T)   { testTransactionLimitingEquivalency(t, 1) }
func TestTransactionPendingLimitingEquivalency(t *testing.T) { testTransactionLimitingEquivalency(t, 0) }

func testTransactionLimitingEquivalency(t *testing.T, origin uint64) {
	// Add a batch of transactions to a pool one by one
	pool1, key1 := setupTxPool()
	account1, _ := transaction(0, big.NewInt(0), key1).From()
	state1, _ := pool1.currentState()
	state1.AddBalance(account1, big.NewInt(1000000))

	for i := uint64(0); i < maxQueued+5; i++ {
		if err := pool1.Add(transaction(origin+i, big.NewInt(100000), key1)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
	}
	// Add a batch of transactions to a pool in one bit batch
	pool2, key2 := setupTxPool()
	account2, _ := transaction(0, big.NewInt(0), key2).From()
	state2, _ := pool2.currentState()
	state2.AddBalance(account2, big.NewInt(1000000))

	txns := []*types.Transaction{}
	for i := uint64(0); i < maxQueued+5; i++ {
		txns = append(txns, transaction(origin+i, big.NewInt(100000), key2))
	}
	pool2.AddTransactions(txns)

	// Ensure the batch optimization honors the same pool mechanics
	if len(pool1.pending) != len(pool2.pending) {
		t.Errorf("pending transaction count mismatch: one-by-one algo: %d, batch algo: %d", len(pool1.pending), len(pool2.pending))
	}
	if len(pool1.queue[account1]) != len(pool2.queue[account2]) {
		t.Errorf("queued transaction count mismatch: one-by-one algo: %d, batch algo: %d", len(pool1.queue[account1]), len(pool2.queue[account2]))
	}
}

// Benchmarks the speed of validating the contents of the pending queue of the
// transaction pool.
func BenchmarkValidatePool100(b *testing.B)   { benchmarkValidatePool(b, 100) }
func BenchmarkValidatePool1000(b *testing.B)  { benchmarkValidatePool(b, 1000) }
func BenchmarkValidatePool10000(b *testing.B) { benchmarkValidatePool(b, 10000) }

func benchmarkValidatePool(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(i), big.NewInt(100000), key)
		pool.addTx(tx.Hash(), account, tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.validatePool()
	}
}

// Benchmarks the speed of scheduling the contents of the future queue of the
// transaction pool.
func BenchmarkCheckQueue100(b *testing.B)   { benchmarkCheckQueue(b, 100) }
func BenchmarkCheckQueue1000(b *testing.B)  { benchmarkCheckQueue(b, 1000) }
func BenchmarkCheckQueue10000(b *testing.B) { benchmarkCheckQueue(b, 10000) }

func benchmarkCheckQueue(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	account, _ := transaction(0, big.NewInt(0), key).From()
	state, _ := pool.currentState()
	state.AddBalance(account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(1+i), big.NewInt(100000), key)
		pool.queueTx(tx.Hash(), tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.checkQueue()
	}
}
