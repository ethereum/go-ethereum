// Copyright 2024 The go-ethereum Authors
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

package exex

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/exex"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// stateAdapter is an adapter to convert Geth's internal state db (unstable
// and legacy API) into the exex state interface (stable API).
type stateAdapter struct {
	root  common.Hash    // Root hash of the state being accessed
	state vm.StateDB     // State database backed by a trie
	snaps *snapshot.Tree // State database backed by a snapshot
}

// wrapState wraps a Geth internal state object into an exex stable API.
func wrapState(root common.Hash, state vm.StateDB, snaps *snapshot.Tree) exex.State {
	return &stateAdapter{
		root:  root,
		state: state,
		snaps: snaps,
	}
}

// Balance retrieves the balance of the given account, or 0 if the account is
// not found in the state.
func (a *stateAdapter) Balance(addr common.Address) *uint256.Int {
	return a.state.GetBalance(addr)
}

// Nonce retrieves the nonce of the given account, or 0 if the account is not
// found in the state.
func (a *stateAdapter) Nonce(addr common.Address) uint64 {
	return a.state.GetNonce(addr)
}

// Code retrieves the bytecode associated with the given account, or a nil slice
// if the account is not found.
func (a *stateAdapter) Code(addr common.Address) []byte {
	return common.CopyBytes(a.state.GetCode(addr))
}

// Storage retrieves the value associated with a specific storage slot key within
// a specific account.
func (a *stateAdapter) Storage(addr common.Address, slot common.Hash) common.Hash {
	return a.state.GetState(addr, slot)
}

// AccountIterator retrieves an iterator to walk across all the known accounts in
// the Ethereum state trie from a starting position.
func (a *stateAdapter) AccountIterator(seek common.Hash) snapshot.AccountIterator {
	if a.snaps == nil {
		return nil
	}
	it, err := a.snaps.AccountIterator(a.root, seek)
	if err != nil {
		return nil
	}
	return it
}

// StorageIterator retrieves an iterator to walk across all the known storage slots
// the Ethereum state trie of a given account, from a starting position.
//
// The iteration order is the Merkle-Patricia order (storage slot hash alphabetically).
//
// Note, the account is the hash of the address. This is due to the AccountIterator
// also walking the state in hash order, not address order.
func (a *stateAdapter) StorageIterator(account common.Hash, seek common.Hash) snapshot.StorageIterator {
	if a.snaps == nil {
		return nil
	}
	it, err := a.snaps.StorageIterator(a.root, account, seek)
	if err != nil {
		return nil
	}
	return it
}
