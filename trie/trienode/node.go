// Copyright 2023 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package trienode

import "github.com/ethereum/go-ethereum/common"

// Node is a wrapper which contains the encoded blob of the trie node and its
// unique hash identifier. It is general enough that can be used to represent
// trie nodes corresponding to different trie implementations.
type Node struct {
	Hash common.Hash // Node hash, empty for deleted node
	Blob []byte      // Encoded node blob, nil for the deleted node
}

// Size returns the total memory size used by this node.
func (n *Node) Size() int {
	return len(n.Blob) + common.HashLength
}

// IsDeleted returns the indicator if the node is marked as deleted.
func (n *Node) IsDeleted() bool {
	return n.Hash == (common.Hash{})
}

// WithPrev wraps the Node with the previous node value attached.
type WithPrev struct {
	*Node
	Prev []byte // Encoded original value, nil means it's non-existent
}

// Unwrap returns the internal Node object.
func (n *WithPrev) Unwrap() *Node {
	return n.Node
}

// Size returns the total memory size used by this node. It overloads
// the function in Node by counting the size of previous value as well.
func (n *WithPrev) Size() int {
	return n.Node.Size() + len(n.Prev)
}

// New constructs a node with provided node information.
func New(hash common.Hash, blob []byte) *Node {
	return &Node{Hash: hash, Blob: blob}
}

// NewWithPrev constructs a node with provided node information.
func NewWithPrev(hash common.Hash, blob []byte, prev []byte) *WithPrev {
	return &WithPrev{
		Node: New(hash, blob),
		Prev: prev,
	}
}
