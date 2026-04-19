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
	Node  nodeRef
	Index int
}

type binaryNodeIterator struct {
	trie    *BinaryTrie
	store   *nodeStore
	current nodeRef
	lastErr error

	stack []binaryNodeIteratorState
}

func newBinaryNodeIterator(t *BinaryTrie, _ []byte) (trie.NodeIterator, error) {
	if t.Hash() == zero {
		return &binaryNodeIterator{trie: t, store: t.store, lastErr: errIteratorEnd}, nil
	}
	it := &binaryNodeIterator{trie: t, store: t.store, current: t.store.root}
	return it, nil
}

// Next moves the iterator to the next node. If descend is false, children of
// the current node are skipped.
func (it *binaryNodeIterator) Next(descend bool) bool {
	if it.lastErr == errIteratorEnd {
		return false
	}

	if len(it.stack) == 0 {
		it.stack = append(it.stack, binaryNodeIteratorState{Node: it.trie.store.root})
		it.current = it.trie.store.root
		return true
	}

	switch it.current.Kind() {
	case kindInternal:
		// index: 0 = nothing visited, 1 = left visited, 2 = right visited.
		node := it.store.getInternal(it.current.Index())
		context := &it.stack[len(it.stack)-1]

		if !descend {
			// Skip children: pop this node and advance parent.
			if len(it.stack) == 1 {
				it.lastErr = errIteratorEnd
				return false
			}
			it.stack = it.stack[:len(it.stack)-1]
			it.current = it.stack[len(it.stack)-1].Node
			it.stack[len(it.stack)-1].Index++
			return it.Next(true)
		}

		// Recurse into both children.
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

		// Reached the end of this node; go back to the parent unless we're at the root.
		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case kindStem:
		// Look for the next non-empty value in this stem.
		sn := it.store.getStem(it.current.Index())
		for i := it.stack[len(it.stack)-1].Index; i < 256; i++ {
			if sn.hasValue(byte(i)) {
				it.stack[len(it.stack)-1].Index = i + 1
				return true
			}
		}

		// No more values in this stem; go back to parent to get the next leaf.
		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case kindHashed:
		// Resolve the hashed node from disk, then rewire the parent to point at the
		// resolved node in place.
		if len(it.stack) < 2 {
			it.lastErr = errors.New("cannot resolve hashed root during iteration")
			return false
		}
		hn := it.store.getHashed(it.current.Index())
		data, err := it.trie.nodeResolver(it.Path(), hn.Hash())
		if err != nil {
			it.lastErr = err
			return false
		}
		resolved, err := it.store.deserializeNodeWithHash(data, len(it.stack)-1, hn.Hash())
		if err != nil {
			it.lastErr = err
			return false
		}

		oldHashedIdx := it.current.Index()
		it.current = resolved
		it.stack[len(it.stack)-1].Node = resolved
		parent := &it.stack[len(it.stack)-2]
		parentNode := it.store.getInternal(parent.Node.Index())
		if parent.Index == 0 {
			parentNode.left = resolved
		} else {
			parentNode.right = resolved
		}
		it.store.freeHashedNode(oldHashedIdx)
		return it.Next(descend)

	case kindEmpty:
		return false

	default:
		panic("invalid node type")
	}
}

func (it *binaryNodeIterator) Error() error {
	if it.lastErr == errIteratorEnd {
		return nil
	}
	return it.lastErr
}

func (it *binaryNodeIterator) Hash() common.Hash {
	return it.store.computeHash(it.current)
}

// Parent returns the hash of the current node's parent. When the immediate
// parent is an internal node whose hash has not been materialised, the
// returned hash may be the one of a grandparent instead.
func (it *binaryNodeIterator) Parent() common.Hash {
	if len(it.stack) < 2 {
		return common.Hash{}
	}
	return it.store.computeHash(it.stack[len(it.stack)-2].Node)
}

// Path returns the bit-path to the current node.
// Callers must not retain references to the returned slice after calling Next.
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

func (it *binaryNodeIterator) NodeBlob() []byte {
	return it.store.serializeNode(it.current)
}

// Leaf reports whether the iterator is currently positioned at a leaf value.
// A StemNode holds up to 256 values; the iterator is only "at a leaf" when
// positioned at a specific non-nil value inside the stem, not merely at the
// StemNode itself. The stack Index points to the NEXT position after the
// current value, so Index == 0 means we haven't yielded anything yet.
func (it *binaryNodeIterator) Leaf() bool {
	if it.current.Kind() != kindStem {
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

// LeafKey returns the key of the leaf. Panics if the iterator is not
// positioned at a leaf. Callers must not retain references to the returned
// slice after calling Next.
func (it *binaryNodeIterator) LeafKey() []byte {
	if it.current.Kind() != kindStem {
		panic("Leaf() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())
	return sn.Key(it.stack[len(it.stack)-1].Index - 1)
}

// LeafBlob returns the leaf value. Panics if the iterator is not positioned
// at a leaf. Callers must not retain references to the returned slice after
// calling Next.
func (it *binaryNodeIterator) LeafBlob() []byte {
	if it.current.Kind() != kindStem {
		panic("LeafBlob() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())
	return sn.getValue(byte(it.stack[len(it.stack)-1].Index - 1))
}

// LeafProof returns the Merkle proof of the leaf. Panics if the iterator is
// not positioned at a leaf. Callers must not retain references to the
// returned slices after calling Next.
func (it *binaryNodeIterator) LeafProof() [][]byte {
	if it.current.Kind() != kindStem {
		panic("LeafProof() called on an binary node iterator not at a leaf location")
	}
	sn := it.store.getStem(it.current.Index())

	proof := make([][]byte, 0, len(it.stack)+StemNodeWidth)

	if len(it.stack) < 2 {
		proof = append(proof, sn.Stem[:])
		proof = append(proof, sn.allValues()...)
		return proof
	}

	for i := range it.stack[:len(it.stack)-2] {
		state := it.stack[i]
		internalNode := it.store.getInternal(state.Node.Index())

		if state.Index == 0 {
			rh := it.store.computeHash(internalNode.right)
			proof = append(proof, rh.Bytes())
		} else {
			lh := it.store.computeHash(internalNode.left)
			proof = append(proof, lh.Bytes())
		}
	}

	// Add the stem and siblings
	proof = append(proof, sn.Stem[:])
	proof = append(proof, sn.allValues()...)

	return proof
}

// AddResolver is a no-op (satisfies the NodeIterator interface).
func (it *binaryNodeIterator) AddResolver(trie.NodeResolver) {
	// Not implemented, but should not panic
}
