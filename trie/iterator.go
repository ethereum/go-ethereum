// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Iterator is a key-value trie iterator to traverse the data contents.
type Iterator struct {
	trie *Trie

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
}

// NewIterator creates a new key-value iterator.
func NewIterator(trie *Trie) *Iterator {
	return &Iterator{trie: trie, Key: nil}
}

// Next moves the iterator forward with one key-value entry.
func (self *Iterator) Next() bool {
	isIterStart := false
	if self.Key == nil {
		isIterStart = true
		self.Key = make([]byte, 32)
	}

	key := remTerm(compactHexDecode(self.Key))
	k := self.next(self.trie.root, key, isIterStart)

	self.Key = []byte(decodeCompact(k))

	return len(k) > 0
}

func (self *Iterator) next(node interface{}, key []byte, isIterStart bool) []byte {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case fullNode:
		if len(key) > 0 {
			k := self.next(node[key[0]], key[1:], isIterStart)
			if k != nil {
				return append([]byte{key[0]}, k...)
			}
		}

		var r byte
		if len(key) > 0 {
			r = key[0] + 1
		}

		for i := r; i < 16; i++ {
			k := self.key(node[i])
			if k != nil {
				return append([]byte{i}, k...)
			}
		}

	case shortNode:
		k := remTerm(node.Key)
		if vnode, ok := node.Val.(valueNode); ok {
			switch bytes.Compare([]byte(k), key) {
			case 0:
				if isIterStart {
					self.Value = vnode
					return k
				}
			case 1:
				self.Value = vnode
				return k
			}
		} else {
			cnode := node.Val

			var ret []byte
			skey := key[len(k):]
			if bytes.HasPrefix(key, k) {
				ret = self.next(cnode, skey, isIterStart)
			} else if bytes.Compare(k, key[:len(k)]) > 0 {
				return self.key(node)
			}

			if ret != nil {
				return append(k, ret...)
			}
		}

	case hashNode:
		rn, err := self.trie.resolveHash(node, nil, nil)
		if err != nil && glog.V(logger.Error) {
			glog.Errorf("Unhandled trie error: %v", err)
		}
		return self.next(rn, key, isIterStart)
	}
	return nil
}

func (self *Iterator) key(node interface{}) []byte {
	switch node := node.(type) {
	case shortNode:
		// Leaf node
		k := remTerm(node.Key)
		if vnode, ok := node.Val.(valueNode); ok {
			self.Value = vnode
			return k
		}
		return append(k, self.key(node.Val)...)
	case fullNode:
		if node[16] != nil {
			self.Value = node[16].(valueNode)
			return []byte{16}
		}
		for i := 0; i < 16; i++ {
			k := self.key(node[i])
			if k != nil {
				return append([]byte{byte(i)}, k...)
			}
		}
	case hashNode:
		rn, err := self.trie.resolveHash(node, nil, nil)
		if err != nil && glog.V(logger.Error) {
			glog.Errorf("Unhandled trie error: %v", err)
		}
		return self.key(rn)
	}
	return nil
}

// nodeIteratorState represents the iteration state at one particular node of the
// trie, which can be resumed at a later invocation.
type nodeIteratorState struct {
	hash  common.Hash // Hash of the node being iterated (nil if not standalone)
	node  node        // Trie node being iterated
	child int         // Child to be processed next
}

// NodeIterator is an iterator to traverse the trie post-order.
type NodeIterator struct {
	trie  *Trie                // Trie being iterated
	stack []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state

	Hash     common.Hash // Hash of the current node being iterated (nil if not standalone)
	Node     node        // Current node being iterated (internal representation)
	Leaf     bool        // Flag whether the current node is a value (data) node
	LeafBlob []byte      // Data blob contained within a leaf (otherwise nil)
}

// NewNodeIterator creates an post-order trie iterator.
func NewNodeIterator(trie *Trie) *NodeIterator {
	if bytes.Compare(trie.Root(), emptyRoot.Bytes()) == 0 {
		return new(NodeIterator)
	}
	return &NodeIterator{trie: trie}
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes.
func (it *NodeIterator) Next() bool {
	it.step()
	return it.retrieve()
}

// step moves the iterator to the next node of the trie.
func (it *NodeIterator) step() {
	// Abort if we reached the end of the iteration
	if it.trie == nil {
		return
	}
	// Initialize the iterator if we've just started, or pop off the old node otherwise
	if len(it.stack) == 0 {
		it.stack = append(it.stack, &nodeIteratorState{node: it.trie.root, child: -1})
		if it.stack[0].node == nil {
			panic(fmt.Sprintf("root node missing: %x", it.trie.Root()))
		}
	} else {
		it.stack = it.stack[:len(it.stack)-1]
		if len(it.stack) == 0 {
			it.trie = nil
			return
		}
	}
	// Continue iteration to the next child
	for {
		parent := it.stack[len(it.stack)-1]
		if node, ok := parent.node.(fullNode); ok {
			// Full node, traverse all children, then the node itself
			if parent.child >= len(node) {
				break
			}
			for parent.child++; parent.child < len(node); parent.child++ {
				if current := node[parent.child]; current != nil {
					it.stack = append(it.stack, &nodeIteratorState{node: current, child: -1})
					break
				}
			}
		} else if node, ok := parent.node.(shortNode); ok {
			// Short node, traverse the pointer singleton child, then the node itself
			if parent.child >= 0 {
				break
			}
			parent.child++
			it.stack = append(it.stack, &nodeIteratorState{node: node.Val, child: -1})
		} else if hash, ok := parent.node.(hashNode); ok {
			// Hash node, resolve the hash child from the database, then the node itself
			if parent.child >= 0 {
				break
			}
			parent.child++

			node, err := it.trie.resolveHash(hash, nil, nil)
			if err != nil {
				panic(err)
			}
			it.stack = append(it.stack, &nodeIteratorState{hash: common.BytesToHash(hash), node: node, child: -1})
		} else {
			break
		}
	}
}

// retrieve pulls and caches the current trie node the iterator is traversing.
// In case of a value node, the additional leaf blob is also populated with the
// data contents for external interpretation.
//
// The method returns whether there are any more data left for inspection.
func (it *NodeIterator) retrieve() bool {
	// Clear out any previously set values
	it.Hash, it.Node, it.Leaf, it.LeafBlob = common.Hash{}, nil, false, nil

	// If the iteration's done, return no available data
	if it.trie == nil {
		return false
	}
	// Otherwise retrieve the current node and resolve leaf accessors
	state := it.stack[len(it.stack)-1]

	it.Hash, it.Node = state.hash, state.node
	if value, ok := it.Node.(valueNode); ok {
		it.Leaf, it.LeafBlob = true, []byte(value)
	}
	return true
}
