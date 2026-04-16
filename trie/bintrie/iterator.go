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
	"bytes"
	"errors"
	"fmt"

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

func newBinaryNodeIterator(t *BinaryTrie, start []byte) (trie.NodeIterator, error) {
	if t.Hash() == zero {
		return &binaryNodeIterator{trie: t, lastErr: errIteratorEnd}, nil
	}
	it := &binaryNodeIterator{trie: t, current: t.root}
	if len(start) > 0 {
		if err := it.seek(start); err != nil {
			return nil, err
		}
	}
	return it, nil
}

// seek positions the iterator so that the next call to Next(true) advances to
// the first leaf with key >= start. It walks down the trie following start's
// bit path, building the iterator stack along the way. When the chosen path
// dead-ends (Empty, missing child, or a stem strictly less than start), the
// implementation backtracks through the existing stack to find the next
// in-order subtree and descends to its leftmost leaf.
//
// A nil/empty start is a no-op; iteration begins at the trie root as usual.
//
// This is required for resumable bintrie generators (snapshot generation,
// pathdb flat-state population) so that an interrupted run can pick up where
// it left off after a crash or graceful shutdown.
func (it *binaryNodeIterator) seek(start []byte) error {
	if len(start) == 0 {
		return nil
	}
	// Pad start to a 32-byte key (the trie's natural key length).
	var key [32]byte
	copy(key[:], start)

	// Reset state
	it.stack = it.stack[:0]
	it.current = nil
	it.lastErr = nil

	root := it.trie.root
	if root == nil {
		it.lastErr = errIteratorEnd
		return nil
	}
	if _, isEmpty := root.(Empty); isEmpty {
		it.lastErr = errIteratorEnd
		return nil
	}

	// Resolve the root if it's a HashedNode
	resolved, err := it.resolveIfHashed(root, nil, 0)
	if err != nil {
		return err
	}
	if resolved == nil {
		it.lastErr = errIteratorEnd
		return nil
	}
	if resolved != root {
		it.trie.root = resolved
		root = resolved
	}

	return it.seekDescend(root, key[:])
}

// seekDescend walks down from `node` following key's bit path. For each
// InternalNode encountered, it pushes the node onto the stack with Index set
// to the bit it descended into (0 for left, 1 for right) and recurses into
// the chosen child. On a StemNode it positions at the appropriate value
// offset and returns. On a dead end (Empty, nil, stem < key), it delegates
// to seekBacktrack to find the next valid subtree.
func (it *binaryNodeIterator) seekDescend(node BinaryNode, key []byte) error {
	for {
		switch n := node.(type) {
		case *InternalNode:
			depth := n.depth
			if depth >= 31*8 {
				return errors.New("seek: internal node too deep")
			}
			bit := key[depth/8] >> (7 - uint(depth%8)) & 1

			// Push this internal node with Index = chosen bit. The Next()
			// loop interprets Index as "the side currently being explored",
			// so this is consistent with normal iteration state.
			it.stack = append(it.stack, binaryNodeIteratorState{Node: n, Index: int(bit)})
			it.current = n

			var child BinaryNode
			if bit == 0 {
				child = n.left
			} else {
				child = n.right
			}
			if child == nil {
				return it.seekBacktrack()
			}
			if _, isEmpty := child.(Empty); isEmpty {
				return it.seekBacktrack()
			}
			// Resolve a hashed child using the current key as the path source.
			resolved, err := it.resolveIfHashed(child, key, depth+1)
			if err != nil {
				return err
			}
			if resolved == nil {
				return it.seekBacktrack()
			}
			if resolved != child {
				if bit == 0 {
					n.left = resolved
				} else {
					n.right = resolved
				}
			}
			node = resolved

		case *StemNode:
			cmp := bytes.Compare(n.Stem, key[:StemSize])
			if cmp < 0 {
				// Stem is strictly before our target. Don't push it; backtrack
				// to find the next subtree to the right.
				return it.seekBacktrack()
			}
			startOffset := 0
			if cmp == 0 {
				startOffset = int(key[StemSize])
			}
			it.stack = append(it.stack, binaryNodeIteratorState{Node: n, Index: startOffset})
			it.current = n
			return nil

		default:
			return fmt.Errorf("seek: unexpected node type %T", node)
		}
	}
}

// seekBacktrack walks the existing stack backward looking for the first
// InternalNode whose right subtree hasn't been considered yet. If found, it
// flips that node's Index to 1 and descends into the leftmost leaf of the
// right subtree. If no such ancestor exists, it sets errIteratorEnd.
func (it *binaryNodeIterator) seekBacktrack() error {
	for len(it.stack) > 0 {
		top := &it.stack[len(it.stack)-1]
		n, ok := top.Node.(*InternalNode)
		if !ok {
			// Not an InternalNode (e.g., a StemNode pushed elsewhere). Pop and
			// continue. seekDescend never pushes non-internal nodes before
			// returning, so this is a defensive fallback.
			it.stack = it.stack[:len(it.stack)-1]
			continue
		}
		if top.Index == 0 {
			// We were positioned in the left subtree. Try the right sibling.
			top.Index = 1
			right := n.right
			if right == nil {
				it.stack = it.stack[:len(it.stack)-1]
				continue
			}
			if _, isEmpty := right.(Empty); isEmpty {
				it.stack = it.stack[:len(it.stack)-1]
				continue
			}
			// Resolve the right child if it's hashed. Use a synthetic path
			// where the bit at this depth is 1 (we're descending right).
			resolved, err := it.resolveRightChild(n)
			if err != nil {
				return err
			}
			if resolved == nil {
				it.stack = it.stack[:len(it.stack)-1]
				continue
			}
			if resolved != right {
				n.right = resolved
				right = resolved
			}
			it.current = right
			return it.seekLeftmost(right)
		}
		// Index == 1: we were already in the right subtree. Both subtrees of
		// this internal node have been considered. Pop and try higher.
		it.stack = it.stack[:len(it.stack)-1]
	}
	it.lastErr = errIteratorEnd
	return nil
}

// seekLeftmost descends into the leftmost leaf of the subtree rooted at
// `node`, pushing internal nodes onto the stack with Index = 0 (left first).
// It positions the iterator at a StemNode with Index = 0, ready to scan
// values from offset 0.
func (it *binaryNodeIterator) seekLeftmost(node BinaryNode) error {
	for {
		switch n := node.(type) {
		case *InternalNode:
			it.stack = append(it.stack, binaryNodeIteratorState{Node: n, Index: 0})
			it.current = n

			child := n.left
			pickedRight := false
			if child == nil {
				child = n.right
				pickedRight = true
			}
			if child != nil {
				if _, isEmpty := child.(Empty); isEmpty {
					if !pickedRight {
						child = n.right
						pickedRight = true
					}
					if child != nil {
						if _, isEmpty2 := child.(Empty); isEmpty2 {
							child = nil
						}
					}
				}
			}
			if child == nil {
				// Both children are empty/nil — degenerate. Pop and let seek
				// backtrack handle it. (This shouldn't normally happen for a
				// well-formed trie because internal nodes always have at least
				// two non-empty children at construction time.)
				it.stack = it.stack[:len(it.stack)-1]
				return it.seekBacktrack()
			}
			if pickedRight {
				it.stack[len(it.stack)-1].Index = 1
			}
			// Resolve hashed child
			resolved, err := it.resolveIfHashed(child, nil, n.depth+1)
			if err != nil {
				return err
			}
			if resolved == nil {
				// Resolution failed; treat as empty and try the other side.
				if pickedRight {
					// Already tried right; nothing left.
					it.stack = it.stack[:len(it.stack)-1]
					return it.seekBacktrack()
				}
				// Try right
				right := n.right
				if right == nil {
					it.stack = it.stack[:len(it.stack)-1]
					return it.seekBacktrack()
				}
				if _, isEmpty := right.(Empty); isEmpty {
					it.stack = it.stack[:len(it.stack)-1]
					return it.seekBacktrack()
				}
				it.stack[len(it.stack)-1].Index = 1
				resolved, err = it.resolveIfHashed(right, nil, n.depth+1)
				if err != nil {
					return err
				}
				if resolved == nil {
					it.stack = it.stack[:len(it.stack)-1]
					return it.seekBacktrack()
				}
				n.right = resolved
				node = resolved
				continue
			}
			if resolved != child {
				if pickedRight {
					n.right = resolved
				} else {
					n.left = resolved
				}
			}
			node = resolved

		case *StemNode:
			it.stack = append(it.stack, binaryNodeIteratorState{Node: n, Index: 0})
			it.current = n
			return nil

		default:
			return fmt.Errorf("seekLeftmost: unexpected node type %T", node)
		}
	}
}

// resolveIfHashed checks whether the given node is a HashedNode and, if so,
// uses the trie's nodeResolver to load and deserialize the underlying node.
// Returns the resolved node or the original if no resolution was needed.
// Returns (nil, nil) if the resolver returned no data (e.g., zero hash).
//
// keyForPath supplies the bit path used to address the node; for the root
// this is unused (path is empty). depth is the depth of the node being
// resolved, used for the deserialized node's internal depth field.
func (it *binaryNodeIterator) resolveIfHashed(node BinaryNode, keyForPath []byte, depth int) (BinaryNode, error) {
	hn, ok := node.(HashedNode)
	if !ok {
		return node, nil
	}
	var path []byte
	if depth > 0 && keyForPath != nil {
		var err error
		path, err = keyToPath(depth-1, keyForPath)
		if err != nil {
			return nil, err
		}
	}
	data, err := it.trie.nodeResolver(path, common.Hash(hn))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	resolved, err := DeserializeNodeWithHash(data, depth, common.Hash(hn))
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

// resolveRightChild resolves the right child of an InternalNode using a
// synthetic path that ends in bit=1. This is used by seekBacktrack when
// flipping from left to right exploration.
func (it *binaryNodeIterator) resolveRightChild(parent *InternalNode) (BinaryNode, error) {
	right := parent.right
	if _, ok := right.(HashedNode); !ok {
		return right, nil
	}
	// Build a 32-byte key whose bit at parent.depth is 1; rest doesn't matter
	// for the path computation.
	var key [32]byte
	key[parent.depth/8] |= 1 << (7 - uint(parent.depth%8))
	return it.resolveIfHashed(right, key[:], parent.depth+1)
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
		// Check if we're at the root before popping
		if len(it.stack) == 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)
	case HashedNode:
		// resolve the node
		resolverPath := it.Path()
		data, err := it.trie.nodeResolver(resolverPath, common.Hash(node))
		if err != nil {
			panic(err)
		}
		if data == nil {
			// Empty/nil node — treat as Empty, backtrack
			it.current = Empty{}
			it.stack[len(it.stack)-1].Node = it.current
			return it.Next(descend)
		}
		it.current, err = DeserializeNodeWithHash(data, len(it.stack)-1, common.Hash(node))
		if err != nil {
			panic(err)
		}

		// update the stack and parent with the resolved node
		it.stack[len(it.stack)-1].Node = it.current
		if len(it.stack) >= 2 {
			parent := &it.stack[len(it.stack)-2]
			if parent.Index == 0 {
				parent.Node.(*InternalNode).left = it.current
			} else {
				parent.Node.(*InternalNode).right = it.current
			}
		}
		return it.Next(descend)
	case Empty:
		// Empty node - go back to parent and continue
		if len(it.stack) <= 1 {
			it.lastErr = errIteratorEnd
			return false
		}
		it.stack = it.stack[:len(it.stack)-1]
		it.current = it.stack[len(it.stack)-1].Node
		it.stack[len(it.stack)-1].Index++
		return it.Next(descend)
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
// In a Binary Trie, a StemNode contains up to 256 leaf values.
// The iterator is only considered to be "at a leaf" when it's positioned
// at a specific non-nil value within the StemNode, not just at the StemNode itself.
func (it *binaryNodeIterator) Leaf() bool {
	sn, ok := it.current.(*StemNode)
	if !ok {
		return false
	}

	// Check if we have a valid stack position
	if len(it.stack) == 0 {
		return false
	}

	// The Index in the stack state points to the NEXT position after the current value.
	// So if Index is 0, we haven't started iterating through the values yet.
	// If Index is 5, we're currently at value[4] (the 5th value, 0-indexed).
	idx := it.stack[len(it.stack)-1].Index
	if idx == 0 || idx > 256 {
		return false
	}

	// Check if there's actually a value at the current position
	currentValueIndex := idx - 1
	return sn.Values[currentValueIndex] != nil
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

	proof := make([][]byte, 0, len(it.stack)+StemNodeWidth)

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
