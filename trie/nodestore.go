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
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

// errUnexpectedNode is returned if the requested node with specified path is
// not hash matched or marked as deleted.
var errUnexpectedNode = errors.New("unexpected node")

// memoryNode is all the information we know about a single cached trie node
// in the memory.
type memoryNode struct {
	hash common.Hash // Node hash, computed by hashing rlp value
	size uint16      // Byte size of the useful cached data
	node node        // Cached collapsed trie node, or raw rlp data
}

// memoryNodeSize is the raw size of a memoryNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var memoryNodeSize = int(reflect.TypeOf(memoryNode{}).Size())

// rlp returns the raw rlp encoded blob of the cached trie node, either directly
// from the cache, or by regenerating it from the collapsed node.
func (n *memoryNode) rlp() []byte {
	if n.node == nil {
		return nil
	}
	if node, ok := n.node.(rawNode); ok {
		return node
	}
	return nodeToBytes(n.node)
}

// obj returns the decoded and expanded trie node, either directly from the cache,
// or by regenerating it from the rlp encoded blob.
func (n *memoryNode) obj() node {
	if n.node == nil {
		return nil
	}
	if node, ok := n.node.(rawNode); ok {
		return mustDecodeNode(n.hash[:], node)
	}
	return expandNode(n.hash[:], n.node)
}

// memorySize returns the total memory size used by this node.
func (n *memoryNode) memorySize(key int) int {
	return int(n.size) + memoryNodeSize + key
}

// nodeStore is built on the underlying node database with an additional
// node cache. The dirty nodes will be cached here whenever trie commit
// is performed to make them accessible. Nodes are keyed by node path
// which is unique in the trie.
//
// nodeStore is not safe for concurrent use.
type nodeStore struct {
	db    *Database
	nodes map[string]*memoryNode
}

// readNode retrieves the node in canonical representation.
// Returns an MissingNodeError error if the node is not found.
func (s *nodeStore) readNode(owner common.Hash, hash common.Hash, path []byte) (node, error) {
	// Load the node from the local cache first.
	mn, ok := s.nodes[string(path)]
	if ok {
		if mn.hash == hash {
			return mn.obj(), nil
		}
		// Bubble up an error if the trie node is not hash matched.
		// It shouldn't happen at all.
		return nil, fmt.Errorf("%w %x!=%x(%x %v)", errUnexpectedNode, mn.hash, hash, owner, path)
	}
	// Load the node from the underlying database then
	if s.db == nil {
		return nil, &MissingNodeError{Owner: owner, NodeHash: hash, Path: path}
	}
	n := s.db.node(hash)
	if n != nil {
		return n, nil
	}
	return nil, &MissingNodeError{Owner: owner, NodeHash: hash, Path: path}
}

// readBlob retrieves the node in rlp-encoded representation.
// Returns an MissingNodeError error if the node is not found.
func (s *nodeStore) readBlob(owner common.Hash, hash common.Hash, path []byte) ([]byte, error) {
	// Load the node from the local cache first
	mn, ok := s.nodes[string(path)]
	if ok {
		if mn.hash == hash {
			return mn.rlp(), nil
		}
		// Bubble up an error if the trie node is not hash matched.
		// It shouldn't happen at all.
		return nil, fmt.Errorf("%w %x!=%x(%x %v)", errUnexpectedNode, mn.hash, hash, owner, path)
	}
	// Load the node from the underlying database then
	if s.db == nil {
		return nil, &MissingNodeError{Owner: owner, NodeHash: hash, Path: path}
	}
	blob, err := s.db.Node(hash)
	if err == nil {
		return blob, nil
	}
	return nil, &MissingNodeError{Owner: owner, NodeHash: hash, Path: path, err: err}
}

// write inserts a dirty node into the store. It happens in trie commit procedure.
func (s *nodeStore) write(path string, node *memoryNode) {
	s.nodes[path] = node
}

// copy deep copies the nodeStore and returns an independent handler but with
// same content cached inside.
func (s *nodeStore) copy() *nodeStore {
	nodes := make(map[string]*memoryNode)
	for k, n := range s.nodes {
		nodes[k] = n
	}
	return &nodeStore{
		db:    s.db, // safe to copy directly.
		nodes: nodes,
	}
}

// size returns the total memory usage used by caching nodes internally.
func (s *nodeStore) size() common.StorageSize {
	var size common.StorageSize
	for k, n := range s.nodes {
		size += common.StorageSize(n.memorySize(len(k)))
	}
	return size
}

// newNodeStore initializes the nodeStore with the given node reader.
func newNodeStore(db *Database) (*nodeStore, error) {
	return &nodeStore{
		db:    db,
		nodes: make(map[string]*memoryNode),
	}, nil
}

// newMemoryStore initializes the pure in-memory store.
func newMemoryStore() *nodeStore {
	return &nodeStore{nodes: make(map[string]*memoryNode)}
}
