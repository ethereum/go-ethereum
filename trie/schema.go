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

package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	HashScheme = "hashScheme" // Identifier of hash based node scheme

	// Path-based scheme will be introduced in the following PRs.
	// PathScheme = "pathScheme" // Identifier of path based node scheme
)

// NodeScheme describes the scheme for interacting nodes in disk.
type NodeScheme interface {
	// Name returns the identifier of node scheme.
	Name() string

	// HasTrieNode checks the trie node presence with the provided node info and
	// the associated node hash.
	HasTrieNode(db ethdb.KeyValueReader, owner common.Hash, path []byte, hash common.Hash) bool

	// ReadTrieNode retrieves the trie node from database with the provided node
	// info and the associated node hash.
	ReadTrieNode(db ethdb.KeyValueReader, owner common.Hash, path []byte, hash common.Hash) []byte

	// WriteTrieNode writes the trie node into database with the provided node
	// info and associated node hash.
	WriteTrieNode(db ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash, node []byte)

	// DeleteTrieNode deletes the trie node from database with the provided node
	// info and associated node hash.
	DeleteTrieNode(db ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash)

	// IsTrieNode returns an indicator if the given database key is the key of
	// trie node according to the scheme.
	IsTrieNode(key []byte) (bool, []byte)
}

type hashScheme struct{}

// Name returns the identifier of hash based scheme.
func (scheme *hashScheme) Name() string {
	return HashScheme
}

// HasTrieNode checks the trie node presence with the provided node info and
// the associated node hash.
func (scheme *hashScheme) HasTrieNode(db ethdb.KeyValueReader, owner common.Hash, path []byte, hash common.Hash) bool {
	return rawdb.HasTrieNode(db, hash)
}

// ReadTrieNode retrieves the trie node from database with the provided node info
// and associated node hash.
func (scheme *hashScheme) ReadTrieNode(db ethdb.KeyValueReader, owner common.Hash, path []byte, hash common.Hash) []byte {
	return rawdb.ReadTrieNode(db, hash)
}

// WriteTrieNode writes the trie node into database with the provided node info
// and associated node hash.
func (scheme *hashScheme) WriteTrieNode(db ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash, node []byte) {
	rawdb.WriteTrieNode(db, hash, node)
}

// DeleteTrieNode deletes the trie node from database with the provided node info
// and associated node hash.
func (scheme *hashScheme) DeleteTrieNode(db ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash) {
	rawdb.DeleteTrieNode(db, hash)
}

// IsTrieNode returns an indicator if the given database key is the key of trie
// node according to the scheme.
func (scheme *hashScheme) IsTrieNode(key []byte) (bool, []byte) {
	if len(key) == common.HashLength {
		return true, key
	}
	return false, nil
}
