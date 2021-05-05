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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type transactionOrNumber struct {
	tx     *types.Transaction
	number *uint64
}

func (t *transactionOrNumber) Tx() (*types.Transaction, bool) {
	return t.tx, t.tx != nil
}

func (t *transactionOrNumber) Number() (uint64, bool) {
	if t.number != nil {
		return 0, false
	}
	return *t.number, false
}

// txLookup is used internally by TxPool to track transactions while allowing
// lookup without mutex contention.
//
// Note, although this type is properly protected against concurrent access, it
// is **not** a type that should ever be mutated or even exposed outside of the
// transaction pool, since its internal state is tightly coupled with the pools
// internal mechanisms. The sole purpose of the type is to permit out-of-bound
// peeking into the pool in TxPool.Get without having to acquire the widely scoped
// TxPool.mu mutex.
//
// This lookup set combines the notion of "local transactions", which is useful
// to build upper-level structure.
type txLookup struct {
	slots   int
	lock    sync.RWMutex
	locals  map[common.Hash]*transactionOrNumber
	remotes map[common.Hash]*transactionOrNumber
}

// newTxLookup returns a new txLookup structure.
func newTxLookup() *txLookup {
	return &txLookup{
		locals:  make(map[common.Hash]*transactionOrNumber),
		remotes: make(map[common.Hash]*transactionOrNumber),
	}
}

// Has returns true if a transaction exists in the lookup
func (t *txLookup) Has(hash common.Hash) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if tx := t.remotes[hash]; tx != nil {
		return true
	}
	if tx := t.locals[hash]; tx != nil {
		return true
	}
	return false
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash common.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.remotes[hash]; tx != nil {
		tx, ok := tx.Tx()
		if !ok {
			panic("TODO: lookup tx in db")
		}
		return tx
	}
	if tx := t.locals[hash]; tx != nil {
		tx, ok := tx.Tx()
		if !ok {
			panic("TODO: lookup tx in db")
		}
		return tx
	}
	return nil
}

// GetLocal returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetLocal(hash common.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if tx := t.locals[hash]; tx != nil {
		tx, ok := tx.Tx()
		if !ok {
			panic("TODO: lookup tx in db")
		}
		return tx
	}
	return nil
}

// GetRemote returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetRemote(hash common.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.remotes[hash]; tx != nil {
		tx, ok := tx.Tx()
		if !ok {
			panic("TODO: lookup tx in db")
		}
		return tx
	}
	return nil
}

// Count returns the current number of transactions in the lookup.
func (t *txLookup) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals) + len(t.remotes)
}

// LocalCount returns the current number of local transactions in the lookup.
func (t *txLookup) LocalCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals)
}

// RemoteCount returns the current number of remote transactions in the lookup.
func (t *txLookup) RemoteCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.remotes)
}

// Slots returns the current number of slots used in the lookup.
func (t *txLookup) Slots() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.slots
}

// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx *types.Transaction, local bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.slots += numSlots(tx)

	if local {
		t.locals[tx.Hash()] = &transactionOrNumber{tx: tx}
	} else {
		t.remotes[tx.Hash()] = &transactionOrNumber{tx: tx}
	}
}

// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash common.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tx, ok := t.locals[hash]
	if !ok {
		tx, ok = t.remotes[hash]
	}
	if !ok {
		log.Error("No transaction found to be deleted", "hash", hash)
		return
	}
	var resTx *types.Transaction
	if t, ok := tx.Tx(); ok {
		resTx = t
	} else {
		panic("TODO: delete tx from disk")
	}
	t.slots -= numSlots(resTx)

	delete(t.locals, hash)
	delete(t.remotes, hash)
}

// numSlots calculates the number of slots needed for a single transaction.
func numSlots(tx *types.Transaction) int {
	return int((tx.Size() + txSlotSize - 1) / txSlotSize)
}

type senderSet struct {
	accounts map[common.Address]struct{}
}

func newSenderSet() *senderSet {
	return &senderSet{
		make(map[common.Address]struct{}),
	}
}

// contains checks if a given address is contained within the set.
func (as *senderSet) contains(addr common.Address) bool {
	_, exist := as.accounts[addr]
	return exist
}
