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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

var errIteratorEnd = errors.New("end of iteration")

type binaryNodeIteratorState struct {
	Ref   NodeRef
	Index int
}

type binaryNodeIterator struct {
	trie    *BinaryTrie
	current NodeRef
	lastErr error

	stack []binaryNodeIteratorState
}

func newBinaryNodeIterator(t *BinaryTrie, _ []byte) (trie.NodeIterator, error) {
	if t.Hash() == zero {
		return &binaryNodeIterator{trie: t, lastErr: errIteratorEnd}, nil
	}
	it := &binaryNodeIterator{trie: t, current: t.root}
	return it, nil
}

// Next moves the iterator to the next node.
func (it *binaryNodeIterator) Next(descend bool) bool {
	if it.lastErr == errIteratorEnd {
		return false
	}

	if len(it.stack) == 0 {
		it.stack = append(it.stack, binaryNodeIteratorState{Ref: it.trie.root})
		it.current = it.trie.root
		return true
	}

	switch it.current.Kind() {
	case KindInternal:
		n := it.trie.store.getInternal(it.current.Index())
		context := &it.stack[len(it.stack)-1]

		if context.Index == 0 {
			if !n.left.IsEmpty() {
				it.stack = append(it.stack, binaryNodeIteratorState{Ref: n.left})
				it.current = n.left
				return it.Next(descend)
			}
			context.Index++
		}

		if context.Index == 1 {
			if !n.right.IsEmpty() {
				it.stack = append(it.stack, binaryNodeIteratorState{Ref: n.right})
				it.current = n.right
				return it.Next(descend)
			}
			context.Index++
		}

		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Ref
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case KindStem:
		sn := it.trie.store.getStem(it.current.Index())
		for i := it.stack[len(it.stack)-1].Index; i < 256; i++ {
			if sn.Values[i] != nil {
				it.stack[len(it.stack)-1].Index = i + 1
				return true
			}
		}

		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Ref
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)

	case KindHashed:
		hn := it.trie.store.getHashed(it.current.Index())
		data, err := it.trie.nodeResolver(it.Path(), hn.hash)
		if err != nil {
			it.lastErr = fmt.Errorf("iterator resolve error: %w", err)
			return false
		}
		resolved, err := it.trie.store.DeserializeNodeWithHash(data, len(it.stack)-1, hn.hash)
		if err != nil {
			it.lastErr = fmt.Errorf("iterator deserialize error: %w", err)
			return false
		}
		it.current = resolved

		// update the stack and parent with the resolved node
		it.stack[len(it.stack)-1].Ref = resolved
		parent := &it.stack[len(it.stack)-2]
		parentNode := it.trie.store.getInternal(parent.Ref.Index())
		if parent.Index == 0 {
			parentNode.left = resolved
		} else {
			parentNode.right = resolved
		}
		return it.Next(descend)

	case KindEmpty:
		return false

	default:
		it.lastErr = errors.New("invalid node type in iterator")
		return false
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
	return it.trie.store.ComputeHash(it.current)
}

// Parent returns the hash of the parent of the current node.
func (it *binaryNodeIterator) Parent() common.Hash {
	return it.trie.store.ComputeHash(it.stack[len(it.stack)-1].Ref)
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
	return it.trie.store.SerializeNode(it.current)
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
	sn := it.trie.store.getStem(it.current.Index())
	currentValueIndex := idx - 1
	return sn.Values[currentValueIndex] != nil
}

// LeafKey returns the key of the leaf.
func (it *binaryNodeIterator) LeafKey() []byte {
	if it.current.Kind() != KindStem {
		panic("LeafKey() called on an iterator not at a leaf location")
	}
	sn := it.trie.store.getStem(it.current.Index())
	return sn.Key(it.stack[len(it.stack)-1].Index - 1)
}

// LeafBlob returns the content of the leaf.
func (it *binaryNodeIterator) LeafBlob() []byte {
	if it.current.Kind() != KindStem {
		panic("LeafBlob() called on an iterator not at a leaf location")
	}
	sn := it.trie.store.getStem(it.current.Index())
	return sn.Values[it.stack[len(it.stack)-1].Index-1]
}

// LeafProof returns the Merkle proof of the leaf.
func (it *binaryNodeIterator) LeafProof() [][]byte {
	if it.current.Kind() != KindStem {
		panic("LeafProof() called on an iterator not at a leaf location")
	}
	sn := it.trie.store.getStem(it.current.Index())

	proof := make([][]byte, 0, len(it.stack)+StemNodeWidth)

	for i := range it.stack[:len(it.stack)-2] {
		state := it.stack[i]
		internalNode := it.trie.store.getInternal(state.Ref.Index())

		if state.Index == 0 {
			proof = append(proof, it.trie.store.ComputeHash(internalNode.right).Bytes())
		} else {
			proof = append(proof, it.trie.store.ComputeHash(internalNode.left).Bytes())
		}
	}

	proof = append(proof, sn.Stem)
	for _, v := range sn.Values {
		proof = append(proof, v)
	}

	return proof
}

// AddResolver sets an intermediate database to use for looking up trie nodes.
func (it *binaryNodeIterator) AddResolver(trie.NodeResolver) {
	// Not implemented, but should not panic
}
