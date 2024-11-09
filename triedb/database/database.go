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

package database

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NodeReader wraps the Node method of a backing trie reader.
type NodeReader interface {
	// Node retrieves the trie node blob with the provided trie identifier,
	// node path and the corresponding node hash. No error will be returned
	// if the node is not found.
	//
	// Don't modify the returned byte slice since it's not deep-copied and
	// still be referenced by database.
	Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error)
}

// NodeDatabase wraps the methods of a backing trie store.
type NodeDatabase interface {
	// NodeReader returns a node reader associated with the specific state.
	// An error will be returned if the specified state is not available.
	NodeReader(stateRoot common.Hash) (NodeReader, error)
}

// StateReader wraps the Account and Storage method of a backing state reader.
type StateReader interface {
	// Account directly retrieves the account associated with a particular hash in
	// the slim data format. An error will be returned if the read operation exits
	// abnormally. Specifically, if the layer is already stale.
	//
	// Note:
	// - the returned account object is safe to modify
	// - no error will be returned if the requested account is not found in database
	Account(hash common.Hash) (*types.SlimAccount, error)

	// Storage directly retrieves the storage data associated with a particular hash,
	// within a particular account. An error will be returned if the read operation
	// exits abnormally.
	//
	// Note:
	// - the returned storage data is not a copy, please don't modify it
	// - no error will be returned if the requested slot is not found in database
	Storage(accountHash, storageHash common.Hash) ([]byte, error)
}

// StateDatabase wraps the methods of a backing state store.
type StateDatabase interface {
	// StateReader returns a state reader associated with the specific state.
	// An error will be returned if the specified state is not available.
	StateReader(stateRoot common.Hash) (StateReader, error)
}
