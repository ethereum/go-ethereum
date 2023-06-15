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
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/gballet/go-verkle"
)

type verkleNodeIteratorState struct {
	Node  verkle.VerkleNode
	Index int
}

type verkleNodeIterator struct {
	trie    *VerkleTrie
	current verkle.VerkleNode
	lastErr error

	stack []verkleNodeIteratorState
}

func newVerkleNodeIterator(trie *VerkleTrie, start []byte) NodeIterator {
	if trie.Hash() == emptyState {
		return new(nodeIterator)
	}
	it := &verkleNodeIterator{trie: trie, current: trie.root}
	// it.err = it.seek(start)
	return it
}

// Next moves the iterator to the next node. If the parameter is false, any child
// nodes will be skipped.
func (it *verkleNodeIterator) Next(descend bool) bool {
	if it.lastErr == errIteratorEnd {
		it.lastErr = errIteratorEnd
		return false
	}

	if len(it.stack) == 0 {
		it.stack = append(it.stack, verkleNodeIteratorState{Node: it.trie.root, Index: 0})
		it.current = it.trie.root

		return true
	}

	switch node := it.current.(type) {
	case *verkle.InternalNode:
		context := &it.stack[len(it.stack)-1]

		// Look for the next non-empty child
		children := node.Children()
		for ; context.Index < len(children); context.Index++ {
			if _, ok := children[context.Index].(verkle.Empty); !ok {
				it.stack = append(it.stack, verkleNodeIteratorState{Node: children[context.Index], Index: 0})
				it.current = children[context.Index]
				return it.Next(descend)
			}
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
	case *verkle.LeafNode:
		// Look for the next non-empty value
		for i := it.stack[len(it.stack)-1].Index; i < 256; i++ {
			if node.Value(i) != nil {
				it.stack[len(it.stack)-1].Index = i + 1
				return true
			}
		}

		// go back to parent to get the next leaf
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)
	case *verkle.HashedNode:
		// resolve the node
		data, err := it.trie.db.diskdb.Get(nodeToDBKey(node))
		if err != nil {
			panic(err)
		}
		it.current, err = verkle.ParseNode(data, byte(len(it.stack)-1), nodeToDBKey(node))
		if err != nil {
			panic(err)
		}

		// update the stack and parent with the resolved node
		it.stack[len(it.stack)-1].Node = it.current
		parent := &it.stack[len(it.stack)-2]
		parent.Node.(*verkle.InternalNode).SetChild(parent.Index, it.current)
		return true
	default:
		panic("invalid node type")
	}
}

// Error returns the error status of the iterator.
func (it *verkleNodeIterator) Error() error {
	if it.lastErr == errIteratorEnd {
		return nil
	}
	return it.lastErr
}

// Hash returns the hash of the current node.
func (it *verkleNodeIterator) Hash() common.Hash {
	return it.current.Commit().Bytes()
}

// Parent returns the hash of the parent of the current node. The hash may be the one
// grandparent if the immediate parent is an internal node with no hash.
func (it *verkleNodeIterator) Parent() common.Hash {
	return it.stack[len(it.stack)-1].Node.Commit().Bytes()
}

// Path returns the hex-encoded path to the current node.
// Callers must not retain references to the return value after calling Next.
// For leaf nodes, the last element of the path is the 'terminator symbol' 0x10.
func (it *verkleNodeIterator) Path() []byte {
	if it.Leaf() {
		return it.LeafKey()
	}
	var path []byte
	for i, state := range it.stack {
		// skip the last byte
		if i <= len(it.stack)-1 {
			break
		}
		path = append(path, byte(state.Index))
	}
	return path
}

func (it *verkleNodeIterator) NodeBlob() []byte {
	panic("not completely implemented")
}

// Leaf returns true iff the current node is a leaf node.
func (it *verkleNodeIterator) Leaf() bool {
	_, ok := it.current.(*verkle.LeafNode)
	return ok
}

// LeafKey returns the key of the leaf. The method panics if the iterator is not
// positioned at a leaf. Callers must not retain references to the value after
// calling Next.
func (it *verkleNodeIterator) LeafKey() []byte {
	leaf, ok := it.current.(*verkle.LeafNode)
	if !ok {
		panic("Leaf() called on an verkle node iterator not at a leaf location")
	}

	return leaf.Key(it.stack[len(it.stack)-1].Index - 1)
}

// LeafBlob returns the content of the leaf. The method panics if the iterator
// is not positioned at a leaf. Callers must not retain references to the value
// after calling Next.
func (it *verkleNodeIterator) LeafBlob() []byte {
	leaf, ok := it.current.(*verkle.LeafNode)
	if !ok {
		panic("LeafBlob() called on an verkle node iterator not at a leaf location")
	}

	return leaf.Value(it.stack[len(it.stack)-1].Index - 1)
}

// LeafProof returns the Merkle proof of the leaf. The method panics if the
// iterator is not positioned at a leaf. Callers must not retain references
// to the value after calling Next.
func (it *verkleNodeIterator) LeafProof() [][]byte {
	_, ok := it.current.(*verkle.LeafNode)
	if !ok {
		panic("LeafProof() called on an verkle node iterator not at a leaf location")
	}

	// return it.trie.Prove(leaf.Key())
	panic("not completely implemented")
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
func (it *verkleNodeIterator) AddResolver(ethdb.KeyValueReader) {
	// Not implemented, but should not panic
}
