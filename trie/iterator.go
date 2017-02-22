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
	"github.com/ethereum/go-ethereum/common"
)

// Iterator is a key-value trie iterator that traverses a Trie.
type Iterator struct {
	nodeIt NodeIterator

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
}

// NewIterator creates a new key-value iterator.
func NewIterator(trie *Trie) *Iterator {
	return &Iterator{
		nodeIt: NewNodeIterator(trie),
	}
}

// FromNodeIterator creates a new key-value iterator from a node iterator
func NewIteratorFromNodeIterator(it NodeIterator) *Iterator {
	return &Iterator{
		nodeIt: it,
	}
}

// Next moves the iterator forward one key-value entry.
func (it *Iterator) Next() bool {
	for it.nodeIt.Next(true) {
		if it.nodeIt.Leaf() {
			it.Key = decodeCompact(it.nodeIt.Path())
			it.Value = it.nodeIt.LeafBlob()
			return true
		}
	}
	it.Key = nil
	it.Value = nil
	return false
}

// NodeIterator is an iterator to traverse the trie pre-order.
type NodeIterator interface {
	// Hash returns the hash of the current node
	Hash() common.Hash
	// Parent returns the hash of the parent of the current node
	Parent() common.Hash
	// Leaf returns true iff the current node is a leaf node.
	Leaf() bool
	// LeafBlob returns the contents of the node, if it is a leaf.
	// Callers must not retain references to the return value after calling Next()
	LeafBlob() []byte
	// Path returns the hex-encoded path to the current node.
	// Callers must not retain references to the return value after calling Next()
	Path() []byte
	// Next moves the iterator to the next node. If the parameter is false, any child
	// nodes will be skipped.
	Next(bool) bool
	// Error returns the error status of the iterator.
	Error() error
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

type nodeIterator struct {
	trie  *Trie                // Trie being iterated
	stack []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state

	err error // Failure set in case of an internal error in the iterator

	path []byte // Path to the current node
}

// NewNodeIterator creates an post-order trie iterator.
func NewNodeIterator(trie *Trie) NodeIterator {
	if trie.Hash() == emptyState {
		return new(nodeIterator)
	}
	return &nodeIterator{trie: trie}
}

// Hash returns the hash of the current node
func (it *nodeIterator) Hash() common.Hash {
	if len(it.stack) == 0 {
		return common.Hash{}
	}

	return it.stack[len(it.stack)-1].hash
}

// Parent returns the hash of the parent node
func (it *nodeIterator) Parent() common.Hash {
	if len(it.stack) == 0 {
		return common.Hash{}
	}

	return it.stack[len(it.stack)-1].parent
}

// Leaf returns true if the current node is a leaf
func (it *nodeIterator) Leaf() bool {
	if len(it.stack) == 0 {
		return false
	}

	_, ok := it.stack[len(it.stack)-1].node.(valueNode)
	return ok
}

// LeafBlob returns the data for the current node, if it is a leaf
func (it *nodeIterator) LeafBlob() []byte {
	if len(it.stack) == 0 {
		return nil
	}

	if node, ok := it.stack[len(it.stack)-1].node.(valueNode); ok {
		return []byte(node)
	}
	return nil
}

// Path returns the hex-encoded path to the current node
func (it *nodeIterator) Path() []byte {
	return it.path
}

// Error returns the error set in case of an internal error in the iterator
func (it *nodeIterator) Error() error {
	return it.err
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes. In case of an internal error this method returns false and
// sets the Error field to the encountered failure. If `descend` is false,
// skips iterating over any subnodes of the current node.
func (it *nodeIterator) Next(descend bool) bool {
	// If the iterator failed previously, don't do anything
	if it.err != nil {
		return false
	}
	// Otherwise step forward with the iterator and report any errors
	if err := it.step(descend); err != nil {
		it.err = err
		return false
	}
	return it.trie != nil
}

// step moves the iterator to the next node of the trie.
func (it *nodeIterator) step(descend bool) error {
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

	if !descend {
		// If we're skipping children, pop the current node first
		it.path = it.path[:it.stack[len(it.stack)-1].pathlen]
		it.stack = it.stack[:len(it.stack)-1]
	}

	// Continue iteration to the next child
outer:
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
			for parent.child++; parent.child < len(node.Children); parent.child++ {
				child := node.Children[parent.child]
				if child != nil {
					hash, _ := child.cache()
					it.stack = append(it.stack, &nodeIteratorState{
						hash:    common.BytesToHash(hash),
						node:    child,
						parent:  ancestor,
						child:   -1,
						pathlen: len(it.path),
					})
					it.path = append(it.path, byte(parent.child))
					break outer
				}
			}
		} else if node, ok := parent.node.(*shortNode); ok {
			// Short node, return the pointer singleton child
			if parent.child < 0 {
				parent.child++
				hash, _ := node.Val.cache()
				it.stack = append(it.stack, &nodeIteratorState{
					hash:    common.BytesToHash(hash),
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

type differenceIterator struct {
	a, b  NodeIterator // Nodes returned are those in b - a.
	eof   bool         // Indicates a has run out of elements
	count int          // Number of nodes scanned on either trie
}

// NewDifferenceIterator constructs a NodeIterator that iterates over elements in b that
// are not in a. Returns the iterator, and a pointer to an integer recording the number
// of nodes seen.
func NewDifferenceIterator(a, b NodeIterator) (NodeIterator, *int) {
	a.Next(true)
	it := &differenceIterator{
		a: a,
		b: b,
	}
	return it, &it.count
}

func (it *differenceIterator) Hash() common.Hash {
	return it.b.Hash()
}

func (it *differenceIterator) Parent() common.Hash {
	return it.b.Parent()
}

func (it *differenceIterator) Leaf() bool {
	return it.b.Leaf()
}

func (it *differenceIterator) LeafBlob() []byte {
	return it.b.LeafBlob()
}

func (it *differenceIterator) Path() []byte {
	return it.b.Path()
}

func (it *differenceIterator) Next(bool) bool {
	// Invariants:
	// - We always advance at least one element in b.
	// - At the start of this function, a's path is lexically greater than b's.
	if !it.b.Next(true) {
		return false
	}
	it.count += 1

	if it.eof {
		// a has reached eof, so we just return all elements from b
		return true
	}

	for {
		apath, bpath := it.a.Path(), it.b.Path()
		switch bytes.Compare(apath, bpath) {
		case -1:
			// b jumped past a; advance a
			if !it.a.Next(true) {
				it.eof = true
				return true
			}
			it.count += 1
		case 1:
			// b is before a
			return true
		case 0:
			if it.a.Hash() != it.b.Hash() || it.a.Leaf() != it.b.Leaf() {
				// Keys are identical, but hashes or leaf status differs
				return true
			}
			if it.a.Leaf() && it.b.Leaf() && !bytes.Equal(it.a.LeafBlob(), it.b.LeafBlob()) {
				// Both are leaf nodes, but with different values
				return true
			}

			// a and b are identical; skip this whole subtree if the nodes have hashes
			hasHash := it.a.Hash() == common.Hash{}
			if !it.b.Next(hasHash) {
				return false
			}
			it.count += 1
			if !it.a.Next(hasHash) {
				it.eof = true
				return true
			}
			it.count += 1
		}
	}
}

func (it *differenceIterator) Error() error {
	if err := it.a.Error(); err != nil {
		return err
	}
	return it.b.Error()
}
