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

package txpool

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// noncer is a tiny virtual state database to manage the executable nonces of
// accounts in the pool, falling back to reading from a real state database if
// an account is unknown.
type noncer struct {
	fallback *state.StateDB
	nonces   sync.Map
}

// newNoncer creates a new virtual state database to track the pool nonces.
func newNoncer(statedb *state.StateDB) *noncer {
	return &noncer{
		fallback: statedb.Copy(),
	}
}

// get returns the current nonce of an account, falling back to a real state
// database if the account is unknown.
func (txn *noncer) get(addr common.Address) uint64 {
	// We use mutex for get operation is the underlying
	// state will mutate db even for read access.

	if n, ok := txn.nonces.Load(addr); !ok {
		if nonce := txn.fallback.GetNonce(addr); nonce != 0 {
			txn.nonces.Store(addr, nonce)
			return nonce
		}
		return 0
	} else {
		return n.(uint64)
	}
}

// set inserts a new virtual nonce into the virtual state database to be returned
// whenever the pool requests it instead of reaching into the real state database.
func (txn *noncer) set(addr common.Address, nonce uint64) {
	txn.nonces.Store(addr, nonce)
}

// setIfLower updates a new virtual nonce into the virtual state database if the
// new one is lower.
func (txn *noncer) setIfLower(addr common.Address, nonce uint64) {
	if n, ok := txn.nonces.Load(addr); !ok {
		if fn := txn.fallback.GetNonce(addr); fn <= nonce {
			txn.nonces.Store(addr, fn)
			return
		}
	} else {
		if n.(uint64) <= nonce {
			return
		}
	}
	txn.nonces.Store(addr, nonce)
}

// setAll sets the nonces for all accounts to the given map.
func (txn *noncer) setAll(all map[common.Address]uint64) {
	for addr, nonce := range all {
		txn.nonces.Store(addr, nonce)
	}
}
