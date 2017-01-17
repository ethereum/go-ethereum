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

import "github.com/ethereum/go-ethereum/common"

// Iterator is a key-value trie iterator that traverses a Trie.
type Iterator struct {
	trie   *Trie
	nodeIt *NodeIterator

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
}

// NewIterator creates a new key-value iterator.
func NewIterator(trie *Trie) *Iterator {
	return &Iterator{
		trie:   trie,
		nodeIt: NewNodeIterator(trie),
		Key:    nil,
	}
}

// Next moves the iterator forward one key-value entry.
func (it *Iterator) Next() bool {
	for it.nodeIt.Next(true) {
		if it.nodeIt.Leaf {
			it.Key = decodeCompact(it.nodeIt.path)
			it.Value = it.nodeIt.LeafBlob
			return true
		}
	}
	it.Key = nil
	it.Value = nil
	return false
}

// nodeIteratorState represents the iteration state at one particular node of the
// trie, which can be resumed at a later invocation.
type nodeIteratorState struct {
	hash    common.Hash // Hash of the node being iterated (nil if not standalone)
	node    node        // Trie node being iterated
	parent  common.Hash // Hash of the first full ancestor node (nil if current is the root)
	child   int         // Child to be processed next
	pathlen int         // Length of the path to this node
}

// NodeIterator is an iterator to traverse the trie post-order.
type NodeIterator struct {
	trie  *Trie                // Trie being iterated
	stack []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state

	Hash     common.Hash // Hash of the current node being iterated (nil if not standalone)
	Node     node        // Current node being iterated (internal representation)
	Parent   common.Hash // Hash of the first full ancestor node (nil if current is the root)
	Leaf     bool        // Flag whether the current node is a value (data) node
	LeafBlob []byte      // Data blob contained within a leaf (otherwise nil)
	path     []byte      // Path to the current node

	Error error // Failure set in case of an internal error in the iterator
}

// NewNodeIterator creates an post-order trie iterator.
func NewNodeIterator(trie *Trie) *NodeIterator {
	if trie.Hash() == emptyState {
		return new(NodeIterator)
	}
	return &NodeIterator{trie: trie}
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes. In case of an internal error this method returns false and
// sets the Error field to the encountered failure. If `children` is false,
// skips iterating over any subnodes of the current node.
func (it *NodeIterator) Next(children bool) bool {
	// If the iterator failed previously, don't do anything
	if it.Error != nil {
		return false
	}
	// Otherwise step forward with the iterator and report any errors
	if err := it.step(children); err != nil {
		it.Error = err
		return false
	}
	return it.retrieve()
}

// step moves the iterator to the next node of the trie.
func (it *NodeIterator) step(children bool) error {
	if it.trie == nil {
		// Abort if we reached the end of the iteration
		return nil
	}
	if len(it.stack) == 0 {
		// Initialize the iterator if we've just started.
		root := it.trie.Hash()
		state := &nodeIteratorState{node: it.trie.root, child: -1}
		if root != emptyRoot {
			state.hash = root
		}
		it.stack = append(it.stack, state)
		return nil
	}

	if !children {
		// If we're skipping children, pop the current node first
		it.path = it.path[:it.stack[len(it.stack)-1].pathlen]
		it.stack = it.stack[:len(it.stack)-1]
	}

	// Continue iteration to the next child
	for {
		if len(it.stack) == 0 {
			it.trie = nil
			return nil
		}
		parent := it.stack[len(it.stack)-1]
		ancestor := parent.hash
		if (ancestor == common.Hash{}) {
			ancestor = parent.parent
		}
		if node, ok := parent.node.(*fullNode); ok {
			// Full node, iterate over children
			parent.child++
			if parent.child < len(node.Children) {
				it.stack = append(it.stack, &nodeIteratorState{
					hash:    common.BytesToHash(node.flags.hash),
					node:    node.Children[parent.child],
					parent:  ancestor,
					child:   -1,
					pathlen: len(it.path),
				})
				it.path = append(it.path, byte(parent.child))
				break
			}
		} else if node, ok := parent.node.(*shortNode); ok {
			// Short node, return the pointer singleton child
			if parent.child < 0 {
				parent.child++
				it.stack = append(it.stack, &nodeIteratorState{
					hash:    common.BytesToHash(node.flags.hash),
					node:    node.Val,
					parent:  ancestor,
					child:   -1,
					pathlen: len(it.path),
				})
				if hasTerm(node.Key) {
					it.path = append(it.path, node.Key[:len(node.Key)-1]...)
				} else {
					it.path = append(it.path, node.Key...)
				}
				break
			}
		} else if hash, ok := parent.node.(hashNode); ok {
			// Hash node, resolve the hash child from the database
			if parent.child < 0 {
				parent.child++
				node, err := it.trie.resolveHash(hash, nil, nil)
				if err != nil {
					return err
				}
				it.stack = append(it.stack, &nodeIteratorState{
					hash:    common.BytesToHash(hash),
					node:    node,
					parent:  ancestor,
					child:   -1,
					pathlen: len(it.path),
				})
				break
			}
		}
		it.path = it.path[:parent.pathlen]
		it.stack = it.stack[:len(it.stack)-1]
	}
	return nil
}

// retrieve pulls and caches the current trie node the iterator is traversing.
// In case of a value node, the additional leaf blob is also populated with the
// data contents for external interpretation.
//
// The method returns whether there are any more data left for inspection.
func (it *NodeIterator) retrieve() bool {
	// Clear out any previously set values
	it.Hash, it.Node, it.Parent, it.Leaf, it.LeafBlob = common.Hash{}, nil, common.Hash{}, false, nil

	// If the iteration's done, return no available data
	if it.trie == nil {
		return false
	}
	// Otherwise retrieve the current node and resolve leaf accessors
	state := it.stack[len(it.stack)-1]

	it.Hash, it.Node, it.Parent = state.hash, state.node, state.parent
	if value, ok := it.Node.(valueNode); ok {
		it.Leaf, it.LeafBlob = true, []byte(value)
	}
	return true
}
