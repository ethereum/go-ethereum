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
)

// Reader wraps the Node method of a backing trie reader.
type Reader interface {
	// Node retrieves the trie node blob with the provided trie identifier,
	// node path and the corresponding node hash. No error will be returned
	// if the node is not found.
	Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error)
}

// PreimageStore wraps the methods of a backing store for reading and writing
// trie node preimages.
type PreimageStore interface {
	// Preimage retrieves the preimage of the specified hash.
	Preimage(hash common.Hash) []byte

	// InsertPreimage commits a set of preimages along with their hashes.
	InsertPreimage(preimages map[common.Hash][]byte)
}

// Database wraps the methods of a backing trie store.
type Database interface {
	PreimageStore

	// Reader returns a node reader associated with the specific state.
	// An error will be returned if the specified state is not available.
	Reader(stateRoot common.Hash) (Reader, error)
}
