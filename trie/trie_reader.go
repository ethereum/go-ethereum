// Copyright 2022 The go-ethereum Authors
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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Reader wraps the Node and NodeBlob method of a backing trie store.
type Reader interface {
	// Node retrieves the trie node with the provided trie identifier, hexary
	// node path and the corresponding node hash.
	// No error will be returned if the node is not found.
	Node(owner common.Hash, path []byte, hash common.Hash) (node, error)

	// NodeBlob retrieves the RLP-encoded trie node blob with the provided trie
	// identifier, hexary node path and the corresponding node hash.
	// No error will be returned if the node is not found.
	NodeBlob(owner common.Hash, path []byte, hash common.Hash) ([]byte, error)
}

// NodeReader wraps all the necessary functions for accessing trie node.
type NodeReader interface {
	// GetReader returns a reader for accessing all trie nodes with provided
	// state root. Nil is returned in case the state is not available.
	GetReader(root common.Hash) Reader
}

// trieReader is a wrapper of the underlying node reader. It's not safe
// for concurrent usage.
type trieReader struct {
	owner  common.Hash
	reader Reader
	banned map[string]struct{} // Marker to prevent node from being accessed, for tests
}

// newTrieReader initializes the trie reader with the given node reader.
func newTrieReader(stateRoot, owner common.Hash, db NodeReader) (*trieReader, error) {
	reader := db.GetReader(stateRoot)
	if reader == nil {
		return nil, fmt.Errorf("state not found #%x", stateRoot)
	}
	return &trieReader{owner: owner, reader: reader}, nil
}

// newEmptyReader initializes the pure in-memory reader. All read operations
// should be forbidden and returns the MissingNodeError.
func newEmptyReader() *trieReader {
	return &trieReader{}
}

// node retrieves the trie node with the provided trie node information.
// An MissingNodeError will be returned in case the node is not found or
// any error is encountered.
func (r *trieReader) node(path []byte, hash common.Hash) (node, error) {
	// Perform the logics in tests for preventing trie node access.
	if r.banned != nil {
		if _, ok := r.banned[string(path)]; ok {
			return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path}
		}
	}
	if r.reader == nil {
		return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path}
	}
	node, err := r.reader.Node(r.owner, path, hash)
	if err != nil || node == nil {
		return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path, err: err}
	}
	return node, nil
}

// node retrieves the rlp-encoded trie node with the provided trie node
// information. An MissingNodeError will be returned in case the node is
// not found or any error is encountered.
func (r *trieReader) nodeBlob(path []byte, hash common.Hash) ([]byte, error) {
	// Perform the logics in tests for preventing trie node access.
	if r.banned != nil {
		if _, ok := r.banned[string(path)]; ok {
			return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path}
		}
	}
	if r.reader == nil {
		return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path}
	}
	blob, err := r.reader.NodeBlob(r.owner, path, hash)
	if err != nil || len(blob) == 0 {
		return nil, &MissingNodeError{Owner: r.owner, NodeHash: hash, Path: path, err: err}
	}
	return blob, nil
}
