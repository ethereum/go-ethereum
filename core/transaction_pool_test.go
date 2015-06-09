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

func transaction() *types.Transaction {
	return types.NewTransactionMessage(common.Address{}, big.NewInt(100), big.NewInt(100), big.NewInt(100), nil)
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	db, _ := ethdb.NewMemDatabase()
	statedb := state.New(common.Hash{}, db)

	var m event.TypeMux
	key, _ := crypto.GenerateKey()
	return NewTxPool(&m, func() *state.StateDB { return statedb }, func() *big.Int { return big.NewInt(1000000) }), key
}

func TestInvalidTransactions(t *testing.T) {
	pool, key := setupTxPool()

	tx := transaction()
	tx.SignECDSA(key)
	err := pool.Add(tx)
	if err != ErrNonExistentAccount {
		t.Error("expected", ErrNonExistentAccount)
	}

	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	err = pool.Add(tx)
	if err != ErrInsufficientFunds {
		t.Error("expected", ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(tx.Gas(), tx.GasPrice()))
	pool.currentState().AddBalance(from, balance)
	err = pool.Add(tx)
	if err != ErrIntrinsicGas {
		t.Error("expected", ErrIntrinsicGas, "got", err)
	}

	pool.currentState().SetNonce(from, 1)
	pool.currentState().AddBalance(from, big.NewInt(0xffffffffffffff))
	tx.GasLimit = big.NewInt(100000)
	tx.Price = big.NewInt(1)
	tx.SignECDSA(key)

	err = pool.Add(tx)
	if err != ErrNonce {
		t.Error("expected", ErrNonce)
	}
}

func TestTransactionQueue(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction()
	tx.SignECDSA(key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	pool.queueTx(tx.Hash(), tx)

	pool.checkQueue()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction()
	tx.SetNonce(1)
	tx.SignECDSA(key)
	from, _ = tx.From()
	pool.currentState().SetNonce(from, 2)
	pool.queueTx(tx.Hash(), tx)
	pool.checkQueue()
	if _, ok := pool.pending[tx.Hash()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

	if len(pool.queue[from]) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
	}

	pool, key = setupTxPool()
	tx1, tx2, tx3 := transaction(), transaction(), transaction()
	tx2.SetNonce(10)
	tx3.SetNonce(11)
	tx1.SignECDSA(key)
	tx2.SignECDSA(key)
	tx3.SignECDSA(key)
	pool.queueTx(tx1.Hash(), tx1)
	pool.queueTx(tx2.Hash(), tx2)
	pool.queueTx(tx3.Hash(), tx3)
	from, _ = tx1.From()

	pool.checkQueue()

	if len(pool.pending) != 1 {
		t.Error("expected tx pool to be 1 =")
	}
	if len(pool.queue[from]) != 2 {
		t.Error("expected len(queue) == 2, got", len(pool.queue[from]))
	}
}

func TestRemoveTx(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction()
	tx.SignECDSA(key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	pool.queueTx(tx.Hash(), tx)
	pool.addTx(tx.Hash(), from, tx)
	if len(pool.queue) != 1 {
		t.Error("expected queue to be 1, got", len(pool.queue))
	}

	if len(pool.pending) != 1 {
		t.Error("expected txs to be 1, got", len(pool.pending))
	}

	pool.removeTx(tx.Hash())

	if len(pool.queue) > 0 {
		t.Error("expected queue to be 0, got", len(pool.queue))
	}

	if len(pool.pending) > 0 {
		t.Error("expected txs to be 0, got", len(pool.pending))
	}
}

func TestNegativeValue(t *testing.T) {
	pool, key := setupTxPool()

	tx := transaction()
	tx.Value().Set(big.NewInt(-1))
	tx.SignECDSA(key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	err := pool.Add(tx)
	if err != ErrNegativeValue {
		t.Error("expected", ErrNegativeValue, "got", err)
	}
}

func TestTransactionChainFork(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		pool.currentState = func() *state.StateDB { return statedb }
		pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	tx := transaction()
	tx.GasLimit = big.NewInt(100000)
	tx.SignECDSA(key)

	err := pool.add(tx)
	if err != nil {
		t.Error("didn't expect error", err)
	}
	pool.RemoveTransactions([]*types.Transaction{tx})

	// reset the pool's internal state
	resetState()
	err = pool.add(tx)
	if err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestTransactionDoubleNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		pool.currentState = func() *state.StateDB { return statedb }
		pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	tx := transaction()
	tx.GasLimit = big.NewInt(100000)
	tx.SignECDSA(key)

	err := pool.add(tx)
	if err != nil {
		t.Error("didn't expect error", err)
	}

	tx2 := transaction()
	tx2.GasLimit = big.NewInt(1000000)
	tx2.SignECDSA(key)

	err = pool.add(tx2)
	if err != nil {
		t.Error("didn't expect error", err)
	}

	if len(pool.pending) != 2 {
		t.Error("expected 2 pending txs. Got", len(pool.pending))
	}
}

func TestMissingNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
	tx := transaction()
	tx.AccountNonce = 1
	tx.GasLimit = big.NewInt(100000)
	tx.SignECDSA(key)

	err := pool.add(tx)
	if err != nil {
		t.Error("didn't expect error", err)
	}

	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}

	if len(pool.queue[addr]) != 1 {
		t.Error("expected 1 queued transaction, got", len(pool.queue[addr]))
	}
}
