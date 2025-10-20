// Copyright 2025 The go-ethereum Authors
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

package bintrie

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

var errIteratorEnd = errors.New("end of iteration")

type binaryNodeIteratorState struct {
	Node  BinaryNode
	Index int
}

type binaryNodeIterator struct {
	trie    *BinaryTrie
	current BinaryNode
	lastErr error

	stack []binaryNodeIteratorState
}

func newBinaryNodeIterator(t *BinaryTrie, _ []byte) (trie.NodeIterator, error) {
	if t.Hash() == zero {
		return &binaryNodeIterator{trie: t, lastErr: errIteratorEnd}, nil
	}
	it := &binaryNodeIterator{trie: t, current: t.root}
	// it.err = it.seek(start)
	return it, nil
}

// Next moves the iterator to the next node. If the parameter is false, any child
// nodes will be skipped.
func (it *binaryNodeIterator) Next(descend bool) bool {
	if it.lastErr == errIteratorEnd {
		it.lastErr = errIteratorEnd
		return false
	}

	if len(it.stack) == 0 {
		it.stack = append(it.stack, binaryNodeIteratorState{Node: it.trie.root})
		it.current = it.trie.root

		return true
	}

	switch node := it.current.(type) {
	case *InternalNode:
		// index: 0 = nothing visited, 1=left visited, 2=right visited
		context := &it.stack[len(it.stack)-1]

		// recurse into both children
		if context.Index == 0 {
			if _, isempty := node.left.(Empty); node.left != nil && !isempty {
				it.stack = append(it.stack, binaryNodeIteratorState{Node: node.left})
				it.current = node.left
				return it.Next(descend)
			}

			context.Index++
		}

		if context.Index == 1 {
			if _, isempty := node.right.(Empty); node.right != nil && !isempty {
				it.stack = append(it.stack, binaryNodeIteratorState{Node: node.right})
				it.current = node.right
				return it.Next(descend)
			}

			context.Index++
		}

		// Reached the end of this node, go back to the parent, if
		// this isn't root.
		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)
	case *StemNode:
		// Look for the next non-empty value
		for i := it.stack[len(it.stack)-1].Index; i < 256; i++ {
			if node.Values[i] != nil {
				it.stack[len(it.stack)-1].Index = i + 1
				return true
			}
		}

		// go back to parent to get the next leaf
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)
	case HashedNode:
		// resolve the node
		data, err := it.trie.nodeResolver(it.Path(), common.Hash(node))
		if err != nil {
			panic(err)
		}
		it.current, err = DeserializeNode(data, len(it.stack)-1)
		if err != nil {
			panic(err)
		}

		// update the stack and parent with the resolved node
		it.stack[len(it.stack)-1].Node = it.current
		parent := &it.stack[len(it.stack)-2]
		if parent.Index == 0 {
			parent.Node.(*InternalNode).left = it.current
		} else {
			parent.Node.(*InternalNode).right = it.current
		}
		return it.Next(descend)
	case Empty:
		// do nothing
		return false
	default:
		panic("invalid node type")
	}
}

// Error returns the error status of the iterator.
func (it *binaryNodeIterator) Error() error {
	if it.lastErr == errIteratorEnd {
		return nil
	}
	return it.lastErr
}

// Hash returns the hash of the current node.
func (it *binaryNodeIterator) Hash() common.Hash {
	return it.current.Hash()
}

// Parent returns the hash of the parent of the current node. The hash may be the one
// grandparent if the immediate parent is an internal node with no hash.
func (it *binaryNodeIterator) Parent() common.Hash {
	return it.stack[len(it.stack)-1].Node.Hash()
}

// Path returns the hex-encoded path to the current node.
// Callers must not retain references to the return value after calling Next.
// For leaf nodes, the last element of the path is the 'terminator symbol' 0x10.
func (it *binaryNodeIterator) Path() []byte {
	if it.Leaf() {
		return it.LeafKey()
	}
	var path []byte
	for i, state := range it.stack {
		// skip the last byte
		if i >= len(it.stack)-1 {
			break
		}
		path = append(path, byte(state.Index))
	}
	return path
}

// NodeBlob returns the serialized bytes of the current node.
func (it *binaryNodeIterator) NodeBlob() []byte {
	return SerializeNode(it.current)
}

// Leaf returns true iff the current node is a leaf node.
func (it *binaryNodeIterator) Leaf() bool {
	_, ok := it.current.(*StemNode)
	return ok
}

// LeafKey returns the key of the leaf. The method panics if the iterator is not
// positioned at a leaf. Callers must not retain references to the value after
// calling Next.
func (it *binaryNodeIterator) LeafKey() []byte {
	leaf, ok := it.current.(*StemNode)
	if !ok {
		panic("Leaf() called on an binary node iterator not at a leaf location")
	}
	return leaf.Key(it.stack[len(it.stack)-1].Index - 1)
}

// LeafBlob returns the content of the leaf. The method panics if the iterator
// is not positioned at a leaf. Callers must not retain references to the value
// after calling Next.
func (it *binaryNodeIterator) LeafBlob() []byte {
	leaf, ok := it.current.(*StemNode)
	if !ok {
		panic("LeafBlob() called on an binary node iterator not at a leaf location")
	}
	return leaf.Values[it.stack[len(it.stack)-1].Index-1]
}

// LeafProof returns the Merkle proof of the leaf. The method panics if the
// iterator is not positioned at a leaf. Callers must not retain references
// to the value after calling Next.
func (it *binaryNodeIterator) LeafProof() [][]byte {
	sn, ok := it.current.(*StemNode)
	if !ok {
		panic("LeafProof() called on an binary node iterator not at a leaf location")
	}

	proof := make([][]byte, 0, len(it.stack)+NodeWidth)

	// Build proof by walking up the stack and collecting sibling hashes
	for i := range it.stack[:len(it.stack)-2] {
		state := it.stack[i]
		internalNode := state.Node.(*InternalNode) // should panic if the node isn't an InternalNode

		// Add the sibling hash to the proof
		if state.Index == 0 {
			// We came from left, so include right sibling
			proof = append(proof, internalNode.right.Hash().Bytes())
		} else {
			// We came from right, so include left sibling
			proof = append(proof, internalNode.left.Hash().Bytes())
		}
	}

	// Add the stem and siblings
	proof = append(proof, sn.Stem)
	for _, v := range sn.Values {
		proof = append(proof, v)
	}

	return proof
}

// AddResolver sets an intermediate database to use for looking up trie nodes
// before reaching into the real persistent layer.
//
// This is not required for normal operation, rather is an optimization for
// cases where trie nodes can be recovered from some external mechanism without
// reading from disk. In those cases, this resolver allows short circuiting
// accesses and returning them from memory.
//
// Before adding a similar mechanism to any other place in Geth, consider
// making trie.Database an interface and wrapping at that level. It's a huge
// refactor, but it could be worth it if another occurrence arises.
func (it *binaryNodeIterator) AddResolver(trie.NodeResolver) {
	// Not implemented, but should not panic
}
