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
	pool.queueTx(tx)

	pool.checkQueue()
	if len(pool.txs) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.txs))
	}

	tx = transaction()
	tx.SignECDSA(key)
	from, _ = tx.From()
	pool.currentState().SetNonce(from, 10)
	tx.SetNonce(1)
	pool.queueTx(tx)
	pool.checkQueue()
	if _, ok := pool.txs[tx.Hash()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

	if len(pool.queue[from]) != 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
	}

	pool, key = setupTxPool()
	tx1, tx2, tx3 := transaction(), transaction(), transaction()
	tx2.SetNonce(10)
	tx3.SetNonce(11)
	tx1.SignECDSA(key)
	tx2.SignECDSA(key)
	tx3.SignECDSA(key)
	pool.queueTx(tx1)
	pool.queueTx(tx2)
	pool.queueTx(tx3)
	from, _ = tx1.From()
	pool.checkQueue()

	if len(pool.txs) != 1 {
		t.Error("expected tx pool to be 1 =")
	}

	if len(pool.queue[from]) != 3 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
	}
}

func TestRemoveTx(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction()
	tx.SignECDSA(key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	pool.queueTx(tx)
	pool.addTx(tx)
	if len(pool.queue) != 1 {
		t.Error("expected queue to be 1, got", len(pool.queue))
	}

	if len(pool.txs) != 1 {
		t.Error("expected txs to be 1, got", len(pool.txs))
	}

	pool.removeTx(tx.Hash())

	if len(pool.queue) > 0 {
		t.Error("expected queue to be 0, got", len(pool.queue))
	}

	if len(pool.txs) > 0 {
		t.Error("expected txs to be 0, got", len(pool.txs))
	}
}
