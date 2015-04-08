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
	return NewTxPool(&m, func() *state.StateDB { return statedb }), key
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

	pool.currentState().AddBalance(from, big.NewInt(100*100))
	err = pool.Add(tx)
	if err != ErrIntrinsicGas {
		t.Error("expected", ErrIntrinsicGas)
	}

	pool.currentState().SetNonce(from, 1)
	pool.currentState().AddBalance(from, big.NewInt(0xffffffffffffff))
	tx.GasLimit = big.NewInt(100000)
	tx.Price = big.NewInt(1)
	tx.SignECDSA(key)

	err = pool.Add(tx)
	if err != ErrImpossibleNonce {
		t.Error("expected", ErrImpossibleNonce)
	}
}
