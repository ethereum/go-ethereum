// Copyright 2026 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// errIteratorEnd is stored in nodeIterator.err when iteration is done.
var errIteratorEnd = errors.New("end of iteration")

// nodeIterator walks the trie in pre-order; leaves surface one at a time in
// ascending key order (bit-lexicographic, which per zone equals byte order).
type nodeIterator struct {
	trie  *BinaryTrie
	stack []*iterState
	err   error
}

type iterState struct {
	node    binaryNode
	walk    bitstr // bits consumed above the node
	pos     int    // == walk.n
	valIdx  int    // next value index within a group (-1 before descent)
	visited bool   // node itself already emitted
}

// NodeIterator returns an iterator over the trie's nodes and leaves,
// starting at the first leaf with key >= start.
func (t *BinaryTrie) NodeIterator(start []byte) (trie.NodeIterator, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	it := &nodeIterator{trie: t}
	it.stack = append(it.stack, &iterState{node: t.root, valIdx: -1})
	if len(start) > 0 {
		it.seek(start)
	}
	return it, nil
}

// seek discards subtrees strictly before the start key.
func (it *nodeIterator) seek(start []byte) {
	// The root frame is on the stack; descend greedily, pruning left
	// subtrees whose keys all sort below start.
	for {
		st := it.stack[len(it.stack)-1]
		switch n := st.node.(type) {
		case hashedNode:
			resolved, err := it.trie.resolve(n, st.walk)
			if err != nil {
				it.err = err
				return
			}
			st.node = resolved
		case *groupNode:
			// Position valIdx at the first value with key >= start.
			st.visited = true
			st.valIdx = len(n.subs)
			for i, sub := range n.subs {
				key := append(append([]byte{}, n.stem...), sub)
				if bytes.Compare(key, start) >= 0 {
					st.valIdx = i
					break
				}
			}
			return
		case *branchNode:
			st.visited = true
			split := st.pos + n.prefix.n
			// Compare start against the subtree's key range: if the walked
			// prefix already exceeds start, the whole subtree qualifies; if
			// it sorts below, only the right side may qualify.
			m := n.prefix.matchKey(start, min(st.pos, 8*len(start)))
			if st.pos >= 8*len(start) || m < n.prefix.n && n.prefix.bit(m) == 1 {
				// Subtree keys sort >= start entirely: emit from the left.
				it.push(n, st.walk, split, 0)
				continue
			}
			if m < n.prefix.n {
				// Subtree sorts strictly below start: nothing here.
				it.pop()
				if len(it.stack) == 0 {
					it.err = errIteratorEnd
					return
				}
				continue
			}
			if 8*len(start) <= split {
				it.push(n, st.walk, split, 0)
				continue
			}
			if bitAt(start, split) == 0 {
				it.push(n, st.walk, split, 0)
			} else {
				// Left subtree is entirely below start.
				it.push(n, st.walk, split, 1)
			}
		case empty:
			it.pop()
			if len(it.stack) == 0 {
				it.err = errIteratorEnd
				return
			}
			return
		default:
			return
		}
	}
}

func (it *nodeIterator) push(n *branchNode, walk bitstr, split int, side byte) {
	child := n.left
	if side == 1 {
		child = n.right
	}
	it.stack = append(it.stack, &iterState{
		node:   child,
		walk:   walk.append(n.prefix).concat(side, bitstr{}),
		pos:    split + 1,
		valIdx: -1,
	})
	// Remember which side we descended, so backtracking knows whether the
	// right side is still pending.
	it.stack[len(it.stack)-2].valIdx = int(side)
}

func (it *nodeIterator) pop() {
	it.stack = it.stack[:len(it.stack)-1]
}

// Next moves the iterator to the next node or leaf. If descend is false,
// the current subtree is skipped.
func (it *nodeIterator) Next(descend bool) bool {
	if it.err != nil {
		return false
	}
	if !descend && len(it.stack) > 0 {
		// Skip the current subtree: drop the frame outright.
		it.pop()
		if len(it.stack) == 0 {
			it.err = errIteratorEnd
			return false
		}
	}
	for {
		if len(it.stack) == 0 {
			it.err = errIteratorEnd
			return false
		}
		st := it.stack[len(it.stack)-1]
		switch n := st.node.(type) {
		case empty:
			it.pop()
		case hashedNode:
			resolved, err := it.trie.resolve(n, st.walk)
			if err != nil {
				it.err = err
				return false
			}
			st.node = resolved
		case *groupNode:
			if !st.visited {
				st.visited = true
				st.valIdx = -1
				return true // surface the group record itself
			}
			st.valIdx++
			if st.valIdx < len(n.subs) {
				return true // surface one leaf
			}
			it.pop()
		case *branchNode:
			if !st.visited {
				st.visited = true
				st.valIdx = -1
				return true // surface the branch itself
			}
			split := st.pos + n.prefix.n
			switch st.valIdx {
			case -1:
				it.push(n, st.walk, split, 0)
			case 0:
				it.push(n, st.walk, split, 1)
			default:
				it.pop()
			}
		default:
			it.err = errors.New("bintrie: unknown node in iterator")
			return false
		}
	}
}

func (it *nodeIterator) Error() error {
	if it.err == errIteratorEnd {
		return nil
	}
	return it.err
}

func (it *nodeIterator) current() (*iterState, binaryNode) {
	if len(it.stack) == 0 {
		return nil, nil
	}
	st := it.stack[len(it.stack)-1]
	return st, st.node
}

// Hash returns the hash of the current node.
func (it *nodeIterator) Hash() common.Hash {
	st, n := it.current()
	if n == nil {
		return common.Hash{}
	}
	return n.hashAt(st.pos)
}

// Parent returns the hash of the parent node.
func (it *nodeIterator) Parent() common.Hash {
	if len(it.stack) < 2 {
		return common.Hash{}
	}
	st := it.stack[len(it.stack)-2]
	return st.node.hashAt(st.pos)
}

// Path returns the database path of the current node, or the full key when
// positioned on a leaf.
func (it *nodeIterator) Path() []byte {
	st, n := it.current()
	if n == nil {
		return nil
	}
	if g, ok := n.(*groupNode); ok && st.valIdx >= 0 && st.valIdx < len(g.subs) {
		return append(append([]byte{}, g.stem...), g.subs[st.valIdx])
	}
	return pathOf(st.walk)
}

// NodeBlob returns the database record of the current node.
func (it *nodeIterator) NodeBlob() []byte {
	st, n := it.current()
	if n == nil {
		return nil
	}
	switch n.(type) {
	case *groupNode, *branchNode:
		return serializeNode(n, st.pos)
	}
	return nil
}

// Leaf reports whether the iterator is positioned on a leaf.
func (it *nodeIterator) Leaf() bool {
	st, n := it.current()
	if n == nil {
		return false
	}
	g, ok := n.(*groupNode)
	return ok && st.valIdx >= 0 && st.valIdx < len(g.subs)
}

// LeafKey returns the full key of the current leaf.
func (it *nodeIterator) LeafKey() []byte {
	st, n := it.current()
	if g, ok := n.(*groupNode); ok && st.valIdx >= 0 && st.valIdx < len(g.subs) {
		return append(append([]byte{}, g.stem...), g.subs[st.valIdx])
	}
	panic("not at leaf")
}

// LeafBlob returns the value of the current leaf.
func (it *nodeIterator) LeafBlob() []byte {
	st, n := it.current()
	if g, ok := n.(*groupNode); ok && st.valIdx >= 0 && st.valIdx < len(g.subs) {
		return g.vals[st.valIdx]
	}
	panic("not at leaf")
}

// LeafProof returns the merkle proof of the current leaf: the hash
// preimages of every canonical node from the root to the leaf.
func (it *nodeIterator) LeafProof() [][]byte {
	st, n := it.current()
	g, ok := n.(*groupNode)
	if !ok || st.valIdx < 0 || st.valIdx >= len(g.subs) {
		panic("not at leaf")
	}
	key := append(append([]byte{}, g.stem...), g.subs[st.valIdx])
	collector := &proofList{}
	if err := it.trie.prove(key, collector); err != nil {
		// The signature has no room for an error and a nil proof is a legal
		// answer, so record it on the iterator: without this a failed resolve
		// is indistinguishable from a leaf that simply has no proof.
		it.err = err
		return nil
	}
	return collector.list
}

// AddResolver is accepted for interface compatibility; the trie's own
// database reader performs resolution.
func (it *nodeIterator) AddResolver(trie.NodeResolver) {}
