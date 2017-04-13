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
	"container/heap"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var iteratorEnd = errors.New("end of iteration")

// Iterator is a key-value trie iterator that traverses a Trie.
type Iterator struct {
	nodeIt NodeIterator

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
}

// NewIterator creates a new key-value iterator from a node iterator
func NewIterator(it NodeIterator) *Iterator {
	return &Iterator{
		nodeIt: it,
	}
}

// Next moves the iterator forward one key-value entry.
func (it *Iterator) Next() bool {
	for it.nodeIt.Next(true) {
		if it.nodeIt.Leaf() {
			it.Key = hexToKeybytes(it.nodeIt.Path())
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
	index   int         // Child to be processed next
	pathlen int         // Length of the path to this node
}

type nodeIterator struct {
	trie  *Trie                // Trie being iterated
	stack []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state
	err   error                // Failure set in case of an internal error in the iterator
	path  []byte               // Path to the current node
}

func newNodeIterator(trie *Trie, start []byte) NodeIterator {
	if trie.Hash() == emptyState {
		return new(nodeIterator)
	}
	it := &nodeIterator{trie: trie}
	it.seek(start)
	return it
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
	if it.err == iteratorEnd {
		return nil
	}
	return it.err
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes. In case of an internal error this method returns false and
// sets the Error field to the encountered failure. If `descend` is false,
// skips iterating over any subnodes of the current node.
func (it *nodeIterator) Next(descend bool) bool {
	if it.err != nil {
		return false
	}
	// Otherwise step forward with the iterator and report any errors
	state, parentIndex, path, err := it.peek(descend)
	if err != nil {
		it.err = err
		return false
	}
	it.push(state, parentIndex, path)
	return true
}

func (it *nodeIterator) seek(prefix []byte) {
	// The path we're looking for is the hex encoded key without terminator.
	key := keybytesToHex(prefix)
	key = key[:len(key)-1]
	// Move forward until we're just before the closest match to key.
	for {
		state, parentIndex, path, err := it.peek(bytes.HasPrefix(key, it.path))
		if err != nil || bytes.Compare(path, key) >= 0 {
			it.err = err
			return
		}
		it.push(state, parentIndex, path)
	}
}

// peek creates the next state of the iterator.
func (it *nodeIterator) peek(descend bool) (*nodeIteratorState, *int, []byte, error) {
	if len(it.stack) == 0 {
		// Initialize the iterator if we've just started.
		root := it.trie.Hash()
		state := &nodeIteratorState{node: it.trie.root, index: -1}
		if root != emptyRoot {
			state.hash = root
		}
		return state, nil, nil, nil
	}
	if !descend {
		// If we're skipping children, pop the current node first
		it.pop()
	}

	// Continue iteration to the next child
	for {
		if len(it.stack) == 0 {
			return nil, nil, nil, iteratorEnd
		}
		parent := it.stack[len(it.stack)-1]
		ancestor := parent.hash
		if (ancestor == common.Hash{}) {
			ancestor = parent.parent
		}
		if node, ok := parent.node.(*fullNode); ok {
			// Full node, move to the first non-nil child.
			for i := parent.index + 1; i < len(node.Children); i++ {
				child := node.Children[i]
				if child != nil {
					hash, _ := child.cache()
					state := &nodeIteratorState{
						hash:    common.BytesToHash(hash),
						node:    child,
						parent:  ancestor,
						index:   -1,
						pathlen: len(it.path),
					}
					path := append(it.path, byte(i))
					parent.index = i - 1
					return state, &parent.index, path, nil
				}
			}
		} else if node, ok := parent.node.(*shortNode); ok {
			// Short node, return the pointer singleton child
			if parent.index < 0 {
				hash, _ := node.Val.cache()
				state := &nodeIteratorState{
					hash:    common.BytesToHash(hash),
					node:    node.Val,
					parent:  ancestor,
					index:   -1,
					pathlen: len(it.path),
				}
				var path []byte
				if hasTerm(node.Key) {
					path = append(it.path, node.Key[:len(node.Key)-1]...)
				} else {
					path = append(it.path, node.Key...)
				}
				return state, &parent.index, path, nil
			}
		} else if hash, ok := parent.node.(hashNode); ok {
			// Hash node, resolve the hash child from the database
			if parent.index < 0 {
				node, err := it.trie.resolveHash(hash, nil, nil)
				if err != nil {
					return it.stack[len(it.stack)-1], &parent.index, it.path, err
				}
				state := &nodeIteratorState{
					hash:    common.BytesToHash(hash),
					node:    node,
					parent:  ancestor,
					index:   -1,
					pathlen: len(it.path),
				}
				return state, &parent.index, it.path, nil
			}
		}
		// No more child nodes, move back up.
		it.pop()
	}
}

func (it *nodeIterator) push(state *nodeIteratorState, parentIndex *int, path []byte) {
	it.path = path
	it.stack = append(it.stack, state)
	if parentIndex != nil {
		*parentIndex += 1
	}
}

func (it *nodeIterator) pop() {
	parent := it.stack[len(it.stack)-1]
	it.path = it.path[:parent.pathlen]
	it.stack = it.stack[:len(it.stack)-1]
}

func compareNodes(a, b NodeIterator) int {
	cmp := bytes.Compare(a.Path(), b.Path())
	if cmp != 0 {
		return cmp
	}

	if a.Leaf() && !b.Leaf() {
		return -1
	} else if b.Leaf() && !a.Leaf() {
		return 1
	}

	cmp = bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes())
	if cmp != 0 {
		return cmp
	}

	return bytes.Compare(a.LeafBlob(), b.LeafBlob())
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
		switch compareNodes(it.a, it.b) {
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

type nodeIteratorHeap []NodeIterator

func (h nodeIteratorHeap) Len() int            { return len(h) }
func (h nodeIteratorHeap) Less(i, j int) bool  { return compareNodes(h[i], h[j]) < 0 }
func (h nodeIteratorHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *nodeIteratorHeap) Push(x interface{}) { *h = append(*h, x.(NodeIterator)) }
func (h *nodeIteratorHeap) Pop() interface{} {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

type unionIterator struct {
	items *nodeIteratorHeap // Nodes returned are the union of the ones in these iterators
	count int               // Number of nodes scanned across all tries
	err   error             // The error, if one has been encountered
}

// NewUnionIterator constructs a NodeIterator that iterates over elements in the union
// of the provided NodeIterators. Returns the iterator, and a pointer to an integer
// recording the number of nodes visited.
func NewUnionIterator(iters []NodeIterator) (NodeIterator, *int) {
	h := make(nodeIteratorHeap, len(iters))
	copy(h, iters)
	heap.Init(&h)

	ui := &unionIterator{
		items: &h,
	}
	return ui, &ui.count
}

func (it *unionIterator) Hash() common.Hash {
	return (*it.items)[0].Hash()
}

func (it *unionIterator) Parent() common.Hash {
	return (*it.items)[0].Parent()
}

func (it *unionIterator) Leaf() bool {
	return (*it.items)[0].Leaf()
}

func (it *unionIterator) LeafBlob() []byte {
	return (*it.items)[0].LeafBlob()
}

func (it *unionIterator) Path() []byte {
	return (*it.items)[0].Path()
}

// Next returns the next node in the union of tries being iterated over.
//
// It does this by maintaining a heap of iterators, sorted by the iteration
// order of their next elements, with one entry for each source trie. Each
// time Next() is called, it takes the least element from the heap to return,
// advancing any other iterators that also point to that same element. These
// iterators are called with descend=false, since we know that any nodes under
// these nodes will also be duplicates, found in the currently selected iterator.
// Whenever an iterator is advanced, it is pushed back into the heap if it still
// has elements remaining.
//
// In the case that descend=false - eg, we're asked to ignore all subnodes of the
// current node - we also advance any iterators in the heap that have the current
// path as a prefix.
func (it *unionIterator) Next(descend bool) bool {
	if len(*it.items) == 0 {
		return false
	}

	// Get the next key from the union
	least := heap.Pop(it.items).(NodeIterator)

	// Skip over other nodes as long as they're identical, or, if we're not descending, as
	// long as they have the same prefix as the current node.
	for len(*it.items) > 0 && ((!descend && bytes.HasPrefix((*it.items)[0].Path(), least.Path())) || compareNodes(least, (*it.items)[0]) == 0) {
		skipped := heap.Pop(it.items).(NodeIterator)
		// Skip the whole subtree if the nodes have hashes; otherwise just skip this node
		if skipped.Next(skipped.Hash() == common.Hash{}) {
			it.count += 1
			// If there are more elements, push the iterator back on the heap
			heap.Push(it.items, skipped)
		}
	}

	if least.Next(descend) {
		it.count += 1
		heap.Push(it.items, least)
	}

	return len(*it.items) > 0
}

func (it *unionIterator) Error() error {
	for i := 0; i < len(*it.items); i++ {
		if err := (*it.items)[i].Error(); err != nil {
			return err
		}
	}
	return nil
}
