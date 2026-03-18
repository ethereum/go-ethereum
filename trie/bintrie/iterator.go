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
	Node  NodeRef
	Index int
}

type binaryNodeIterator struct {
	trie    *BinaryTrie
	store   *NodeStore
	current NodeRef
	lastErr error

	stack []binaryNodeIteratorState
}

func newBinaryNodeIterator(t *BinaryTrie, _ []byte) (trie.NodeIterator, error) {
	if t.Hash() == zero {
		return &binaryNodeIterator{trie: t, store: t.store, lastErr: errIteratorEnd}, nil
	}
	it := &binaryNodeIterator{trie: t, store: t.store, current: t.store.Root()}
	return it, nil
}

// Next moves the iterator to the next node.
func (it *binaryNodeIterator) Next(descend bool) bool {
	if it.lastErr == errIteratorEnd {
		return false
	}

	if len(it.stack) == 0 {
		it.stack = append(it.stack, binaryNodeIteratorState{Node: it.trie.store.Root()})
		it.current = it.trie.store.Root()
		return true
	}

	switch it.current.Kind() {
	case KindInternal:
		node := it.store.getInternal(it.current.Index())
		context := &it.stack[len(it.stack)-1]

		if !descend {
			// Skip children: pop this node and advance parent
			if len(it.stack) == 1 {
				it.lastErr = errIteratorEnd
				return false
			}
			it.stack = it.stack[:len(it.stack)-1]
			it.current = it.stack[len(it.stack)-1].Node
			it.stack[len(it.stack)-1].Index++
			return it.Next(true)
		}

		if context.Index == 0 {
			if !node.left.IsEmpty() {
				it.stack = append(it.stack, binaryNodeIteratorState{Node: node.left})
				it.current = node.left
				return it.Next(descend)
			}
			context.Index++
		}

		if context.Index == 1 {
			if !node.right.IsEmpty() {
				it.stack = append(it.stack, binaryNodeIteratorState{Node: node.right})
				it.current = node.right
				return it.Next(descend)
			}
			context.Index++
		}

		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case KindStem:
		sn := it.store.getStem(it.current.Index())
		for i := it.stack[len(it.stack)-1].Index; i < 256; i++ {
			if sn.hasValue(byte(i)) {
				it.stack[len(it.stack)-1].Index = i + 1
				return true
			}
		}

		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case KindHashed:
		if len(it.stack) < 2 {
			it.lastErr = errors.New("cannot resolve hashed root during iteration")
			return false
		}
		hn := it.store.getHashed(it.current.Index())
		data, err := it.trie.nodeResolver(it.Path(), hn.hash)
		if err != nil {
			it.lastErr = err
			return false
		}
		resolved, err := it.store.DeserializeNodeWithHash(data, len(it.stack)-1, hn.hash)
		if err != nil {
			it.lastErr = err
			return false
		}

		// Update the stack and parent with the resolved node
		it.current = resolved
		it.stack[len(it.stack)-1].Node = resolved
		parent := &it.stack[len(it.stack)-2]
		parentNode := it.store.getInternal(parent.Node.Index())
		if parent.Index == 0 {
			parentNode.left = resolved
		} else {
			parentNode.right = resolved
		}
		return it.Next(descend)

	case KindEmpty:
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
	return it.store.ComputeHash(it.current)
}

// Parent returns the hash of the parent of the current node.
func (it *binaryNodeIterator) Parent() common.Hash {
	return it.store.ComputeHash(it.stack[len(it.stack)-1].Node)
}

// Path returns the hex-encoded path to the current node.
func (it *binaryNodeIterator) Path() []byte {
	if it.Leaf() {
		return it.LeafKey()
	}
	var path []byte
	for i, state := range it.stack {
		if i >= len(it.stack)-1 {
			break
		}
		path = append(path, byte(state.Index))
	}
	return path
}

// NodeBlob returns the serialized bytes of the current node.
func (it *binaryNodeIterator) NodeBlob() []byte {
	return it.store.SerializeNode(it.current, MaxGroupDepth)
}

// Leaf returns true iff the current node is a leaf node.
func (it *binaryNodeIterator) Leaf() bool {
	if it.current.Kind() != KindStem {
		return false
	}

	if len(it.stack) == 0 {
		return false
	}

	idx := it.stack[len(it.stack)-1].Index
	if idx == 0 || idx > 256 {
		return false
	}

	sn := it.store.getStem(it.current.Index())
	currentValueIndex := idx - 1
	return sn.hasValue(byte(currentValueIndex))
}

// LeafKey returns the key of the leaf.
func (it *binaryNodeIterator) LeafKey() []byte {
	if it.current.Kind() != KindStem {
		panic("Leaf() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())
	return sn.Key(it.stack[len(it.stack)-1].Index - 1)
}

// LeafBlob returns the content of the leaf.
func (it *binaryNodeIterator) LeafBlob() []byte {
	if it.current.Kind() != KindStem {
		panic("LeafBlob() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())
	return sn.getValue(byte(it.stack[len(it.stack)-1].Index - 1))
}

// LeafProof returns the Merkle proof of the leaf.
func (it *binaryNodeIterator) LeafProof() [][]byte {
	if it.current.Kind() != KindStem {
		panic("LeafProof() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())

	proof := make([][]byte, 0, len(it.stack)+StemNodeWidth)

	for i := range it.stack[:len(it.stack)-2] {
		state := it.stack[i]
		internalNode := it.store.getInternal(state.Node.Index())

		if state.Index == 0 {
			rh := it.store.ComputeHash(internalNode.right)
			proof = append(proof, rh.Bytes())
		} else {
			lh := it.store.ComputeHash(internalNode.left)
			proof = append(proof, lh.Bytes())
		}
	}

	// Add the stem and siblings
	proof = append(proof, sn.Stem[:])
	allVals := sn.allValues()
	for _, v := range allVals {
		proof = append(proof, v)
	}

	return proof
}

// AddResolver sets an intermediate database to use for looking up trie nodes
// before reaching into the real persistent layer.
func (it *binaryNodeIterator) AddResolver(trie.NodeResolver) {
	// Not implemented, but should not panic
}
