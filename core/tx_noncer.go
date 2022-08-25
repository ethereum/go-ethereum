// Copyright 2019 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	lru "github.com/hashicorp/golang-lru"
)

const nonceCacheLimit = 1024 * 50

// txNoncer is a LRU cache to manage the executable nonces of
// accounts in the pool, falling back to reading from a real state database if
// an account is unknown.
type txNoncer struct {
	fallback *state.StateDB
	nonces   *lru.Cache
	lock     sync.Mutex
}

// newTxNoncer creates a new LRU cache to track the pool nonces.
func newTxNoncer(statedb *state.StateDB) *txNoncer {
	cache, _ := lru.New(nonceCacheLimit)
	return &txNoncer{
		fallback: statedb.Copy(),
		nonces:   cache,
	}
}

// get returns the current nonce of an account, falling back to a real state
// database if the account is unknown.
func (txn *txNoncer) get(addr common.Address) uint64 {
	// We use mutex for get operation is the underlying
	// state will mutate db even for read access.
	if _, ok := txn.nonces.Get(addr); !ok {
		txn.lock.Lock()
		txn.nonces.Add(addr, txn.fallback.GetNonce(addr))
		txn.lock.Unlock()
	}
	nonce, _ := txn.nonces.Get(addr)
	return nonce.(uint64)
}

// set inserts a new virtual nonce into the LRU cache to be returned
// whenever the pool requests it instead of reaching into the real state database.
func (txn *txNoncer) set(addr common.Address, nonce uint64) {
	txn.nonces.Add(addr, nonce)
}

// setIfLower updates a new virtual nonce into the LRU cache if the
// the new one is lower.
func (txn *txNoncer) setIfLower(addr common.Address, nonce uint64) {
	if _, ok := txn.nonces.Get(addr); !ok {
		txn.lock.Lock()
		txn.nonces.Add(addr, txn.fallback.GetNonce(addr))
		txn.lock.Unlock()
	}

	cachedNonce, _ := txn.nonces.Get(addr)
	if cachedNonce.(uint64) <= nonce {
		return
	}
	txn.nonces.Add(addr, nonce)
}

// setAll sets the nonces for all accounts to the given map.
func (txn *txNoncer) setAll(all map[common.Address]uint64) {
	for addr, nonce := range all {
		txn.nonces.Add(addr, nonce)
	}
}
