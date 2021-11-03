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

package lendingstate

import (
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
)

type exchanges struct {
	stateObject *lendingExchangeState
	nstart      uint64
	nonces      []bool
}

type LendingManagedState struct {
	*LendingStateDB
	mu         sync.RWMutex
	lenddinges map[common.Hash]*exchanges
}

// LendingManagedState returns a new managed state with the statedb as it's backing layer
func ManageState(statedb *LendingStateDB) *LendingManagedState {
	return &LendingManagedState{
		LendingStateDB: statedb.Copy(),
		lenddinges:     make(map[common.Hash]*exchanges),
	}
}

// SetState sets the backing layer of the managed state
func (ms *LendingManagedState) SetState(statedb *LendingStateDB) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.LendingStateDB = statedb
}

// RemoveNonce removed the nonce from the managed state and all future pending nonces
func (ms *LendingManagedState) RemoveNonce(addr common.Hash, n uint64) {
	if ms.hasAccount(addr) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		account := ms.getAccount(addr)
		if n-account.nstart <= uint64(len(account.nonces)) {
			reslice := make([]bool, n-account.nstart)
			copy(reslice, account.nonces[:n-account.nstart])
			account.nonces = reslice
		}
	}
}

// NewNonce returns the new canonical nonce for the managed tradeId
func (ms *LendingManagedState) NewNonce(addr common.Hash) uint64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	account := ms.getAccount(addr)
	for i, nonce := range account.nonces {
		if !nonce {
			return account.nstart + uint64(i)
		}
	}
	account.nonces = append(account.nonces, true)

	return uint64(len(account.nonces)-1) + account.nstart
}

// GetNonce returns the canonical nonce for the managed or unmanaged tradeId.
//
// Because GetNonce mutates the DB, we must take a write lock.
func (ms *LendingManagedState) GetNonce(addr common.Hash) uint64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.hasAccount(addr) {
		account := ms.getAccount(addr)
		return uint64(len(account.nonces)) + account.nstart
	} else {
		return ms.LendingStateDB.GetNonce(addr)
	}
}

// SetNonce sets the new canonical nonce for the managed state
func (ms *LendingManagedState) SetNonce(addr common.Hash, nonce uint64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	so := ms.GetOrNewLendingExchangeObject(addr)
	so.setNonce(nonce)

	ms.lenddinges[addr] = newAccount(so)
}

// HasAccount returns whether the given address is managed or not
func (ms *LendingManagedState) HasAccount(addr common.Hash) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.hasAccount(addr)
}

func (ms *LendingManagedState) hasAccount(addr common.Hash) bool {
	_, ok := ms.lenddinges[addr]
	return ok
}

// populate the managed state
func (ms *LendingManagedState) getAccount(addr common.Hash) *exchanges {
	if account, ok := ms.lenddinges[addr]; !ok {
		so := ms.GetOrNewLendingExchangeObject(addr)
		ms.lenddinges[addr] = newAccount(so)
	} else {
		// Always make sure the state tradeId nonce isn't actually higher
		// than the tracked one.
		so := ms.LendingStateDB.getLendingExchange(addr)
		if so != nil && uint64(len(account.nonces))+account.nstart < so.Nonce() {
			ms.lenddinges[addr] = newAccount(so)
		}

	}

	return ms.lenddinges[addr]
}

func newAccount(so *lendingExchangeState) *exchanges {
	return &exchanges{so, so.Nonce(), nil}
}
