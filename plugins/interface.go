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

package plugins

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// Plugin is an interface that allows 3rd-party developers to build plugins
// for go-ethereum which can be used to add additional functionality to geth
// without modifying the chain logic.
type Plugin interface {
	// Setup is called on startup when the Plugin is initialized.
	Setup(chain Chain)
	// Close is called when the geth node is torn down.
	Close()
	// NewHead is called when a new head is set.
	NewHead()
}

// The Chain interface allows for interacting with the chain from a plugin.
type Chain interface {
	// Head returns the number of the current head and finalized block.
	Head() (uint64, uint64)
	// Header returns a header in the canonical chain.
	Header(number uint64) *types.Header
	// Block returns a block in the canonical chain.
	Block(number uint64) *types.Block
	// Receipts returns the receipts of a block in the canonical chain.
	Receipts(number uint64) types.Receipts
	// State returns the state at a certain root.
	State(root common.Hash) State
}

// The State interface allows for interacting with a specific state from a plugin.
// Please note that State might hold internal references which interferes with garbage collection.
// Make sure to not hold references to State for long.
type State interface {
	// Account retrieves an account from the state.
	Account(addr common.Address) Account
	// AccountIterator creates an iterator to iterate over accounts.
	AccountIterator(seek common.Hash) snapshot.AccountIterator
	// NewAccount interprets an rlp slim account as an Account.
	NewAccount(addr common.Address, account []byte) Account
}

// The Account interface allows for interacting with a specific account from a plugin.
// Please note that Account might hold internal references which interferes with garbage collection.
// Make sure to not hold references to Account for long.
type Account interface {
	// Balance returns the balance of an account.
	Balance() *uint256.Int
	// Nonce returns the nonce of an account.
	Nonce() uint64
	// Code returns the code of an account.
	Code() []byte
	// Storage returns a storage slot.
	Storage(slot common.Hash) common.Hash
	// StorageIterator creates an iterator over the storage slots of an account.
	StorageIterator(seek common.Hash) snapshot.StorageIterator
}
