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
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/holiman/uint256"
)

// State provides read access to Geth's internal state object.
type State interface {
	// Balance retrieves the balance of the given account, or 0 if the account
	// is not found in the state.
	Balance(addr common.Address) *uint256.Int

	// Nonce retrieves the nonce of the given account, or 0 if the account is
	// not found in the state.
	Nonce(addr common.Address) uint64

	// Code retrieves the bytecode associated with the given account, or a nil
	// slice if the account is not found.
	Code(addr common.Address) []byte

	// Storage retrieves the value associated with a specific storage slot key
	// within a specific account.
	Storage(addr common.Address, slot common.Hash) common.Hash

	// AccountIterator retrieves an iterator to walk across all the known accounts
	// in the Ethereum state trie from a starting position, or returns nil if the
	// requested state is unavailable in snapshot (accelerated access) form.
	//
	// Iteration is in Merkle-Patricia order (address hash alphabetically).
	AccountIterator(seek common.Hash) snapshot.AccountIterator

	// StorageIterator retrieves an iterator to walk across all the known storage
	// slots the Ethereum state trie of a given account, from a starting position,
	// or returns nil if the requested state is unavailable in snapshot (accelerated
	// access) form.
	//
	// Iteration is in Merkle-Patricia order (storage slot hash alphabetically).
	//
	// The account is the hash of the address. This is due to the AccountIterator
	// also walking the state in hash order, not address order.
	StorageIterator(account common.Hash, seek common.Hash) snapshot.StorageIterator
}
