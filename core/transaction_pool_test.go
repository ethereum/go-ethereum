package core

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/state"
)

// State query interface
type stateQuery struct{}

func (self stateQuery) GetAccount(addr []byte) *state.StateObject {
	return state.NewStateObject(addr)
}

func transaction() *types.Transaction {
	return types.NewTransactionMessage(make([]byte, 20), ethutil.Big0, ethutil.Big0, ethutil.Big0, nil)
}

func setup() (*TxPool, *ecdsa.PrivateKey) {
	var m event.TypeMux
	key, _ := crypto.GenerateKey()
	return NewTxPool(&m), key
}

func TestTxAdding(t *testing.T) {
	pool, key := setup()
	tx1 := transaction()
	tx1.SignECDSA(key)
	err := pool.Add(tx1)
	if err != nil {
		t.Error(err)
	}

	err = pool.Add(tx1)
	if err == nil {
		t.Error("added tx twice")
	}
}

func TestAddInvalidTx(t *testing.T) {
	pool, _ := setup()
	tx1 := transaction()
	err := pool.Add(tx1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestRemoveSet(t *testing.T) {
	pool, _ := setup()
	tx1 := transaction()
	pool.pool.Add(tx1)
	pool.RemoveSet(types.Transactions{tx1})
	if pool.Size() > 0 {
		t.Error("expected pool size to be 0")
	}
}

func TestRemoveInvalid(t *testing.T) {
	pool, key := setup()
	tx1 := transaction()
	pool.pool.Add(tx1)
	pool.RemoveInvalid(stateQuery{})
	if pool.Size() > 0 {
		t.Error("expected pool size to be 0")
	}

	tx1.SetNonce(1)
	tx1.SignECDSA(key)
	pool.pool.Add(tx1)
	pool.RemoveInvalid(stateQuery{})
	if pool.Size() != 1 {
		t.Error("expected pool size to be 1, is", pool.Size())
	}
}
