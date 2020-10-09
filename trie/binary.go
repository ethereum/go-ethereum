// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/log"
)

type BinaryNode interface {
	Hash() []byte
	Commit() error
	//tryGet(key []byte, depth int) ([]byte, error)
}

// BinaryHashPreimage represents a tuple of a hash and its preimage
type BinaryHashPreimage struct {
	Key   []byte
	Value []byte
}

type binkey []byte

type storeSlot struct {
	key   binkey
	value []byte
}

type store []storeSlot

type BinaryTrie struct {
	root  BinaryNode
	store store
}

// All known implementations of binaryNode
type (
	// branch is a node with two children ("left" and "right")
	// It can be prefixed by bits that are common to all subtrie
	// keys and it can also hold a value.
	branch struct {
		left  BinaryNode
		right BinaryNode

		key   []byte // TODO split into leaf and branch
		value []byte

		// Used to send (hash, preimage) pairs when hashing
		CommitCh chan BinaryHashPreimage

		// This is the binary equivalent of "extension nodes":
		// binary nodes can have a prefix that is common to all
		// subtrees.
		prefix binkey
	}

	hashBinaryNode []byte

	empty struct{}
)

var (
	errInsertIntoHash    = errors.New("trying to insert into a hash")
	errReadFromHash      = errors.New("trying to read from a hash")
	errReadFromEmptyTree = errors.New("reached an empty subtree")
	errKeyNotPresent     = errors.New("trie doesn't contain key")

	// 0_32
	zero32 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

func newBinKey(key []byte) binkey {
	bits := make([]byte, 8*len(key))
	for i := range bits {

		if key[i/8]&(1<<(7-i%8)) == 0 {
			bits[i] = 0
		} else {
			bits[i] = 1
		}
	}

	return binkey(bits)
}
func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
func (b binkey) commonLength(other binkey) int {
	for i := 0; i < len(b) && i < len(other); i++ {
		if b[i] != other[i] {
			return i
		}
	}
	return min(len(b), len(other))
}
func (b binkey) samePrefix(other binkey, off int) bool {
	return bytes.Equal(b[off:off+len(other)], other[:])
}

func (s store) Len() int { return len(s) }
func (s store) Less(i, j int) bool {
	for b := 0; b < len(s[i].key) && b < len(s[j].key); b++ {
		if s[i].key[b] != s[j].key[b] {
			// if s[j].key.Bit(b) is true, then it is
			// the greater value of the two.
			return s[j].key[b] == 1
		}
	}

	// Keys are equal on their common length, the shortest
	// is the smaller one.
	return len(s[i].key) < len(s[j].key)
}
func (s store) Swap(i, j int) {
	temp := s[i]
	s[i] = s[j]
	s[j] = temp
}

func NewBinTrie() BinaryNode {
	return empty(struct{}{})
}

func (bt *BinaryTrie) TryGet(key []byte) ([]byte, error) {
	bk := newBinKey(key)
	off := 0

	var currentNode *branch
	switch bt.root.(type) {
	case empty:
		return nil, errReadFromEmptyTree
	case *branch:
		currentNode = bt.root.(*branch)
	case hashBinaryNode:
		return nil, errReadFromHash
	}

	for {
		fmt.Printf("%x\n", currentNode.prefix)
		if !bk.samePrefix(currentNode.prefix, off) {
			return nil, errKeyNotPresent
		}

		// XXX if it is a leaf node, then the value needs
		// to be returned right there and then.

		mustBreak := false
		// XXX ca va creer le node s'il existe pas, et on ne
		// veut pas de ca.
		if bk[off+len(currentNode.prefix)] == 0 {
			currentNode, mustBreak = bt.resolveNode(currentNode.left, bk, off, nil)
		} else {
			currentNode, mustBreak = bt.resolveNode(currentNode.right, bk, off, nil)
		}
		// XXX j'aime vraiment pas ca, le mieux serait de sortir
		// la creation du leaf dans TryUpdate pour pas qu'on melange
		// les deux, et du coup on se debarasse de mustBreak
		if mustBreak {
			if currentNode == nil {
				return nil, errKeyNotPresent
			}
			return currentNode.value, nil
		}
		off += len(currentNode.prefix) + 1
	}
}

// Hash calculates the hash of an expanded (i.e. not already
// hashed) node.
func (br *branch) Hash() []byte {
	hasher := newHasher(false)
	defer returnHasherToPool(hasher)
	hasher.sha.Reset()

	// Check that either value is set or left+right are

	hash := make([]byte, 32)
	if br.value == nil {
		// This is a branch node, so the rule is
		// branch_hash = hash(left_root_hash || right_root_hash)
		hasher.sha.Write(br.left.Hash())
		hasher.sha.Write(br.right.Hash())
		hasher.sha.Read(hash)
		hasher.sha.Reset()
	} else {
		// This is a leaf node, so the hashing rule is
		// leaf_hash = hash(hash(key) || hash(leaf_value))
		var kh [32]byte
		hasher.sha.Write(br.key)
		hasher.sha.Read(kh[:])
		hasher.sha.Reset()

		hasher.sha.Write(br.value)
		hasher.sha.Read(hash)
		hasher.sha.Reset()

		hasher.sha.Write(kh[:])
		hasher.sha.Write(hash)
		hasher.sha.Read(hash)
		hasher.sha.Reset()
	}

	if len(br.prefix) > 0 {
		hasher.sha.Write([]byte{byte(len(br.prefix) - 1)})
		hasher.sha.Write(zero32[:31])
		hasher.sha.Write(hash)
		hasher.sha.Read(hash)
		hasher.sha.Reset()
	}

	return hash
}

func NewBinaryTrie() *BinaryTrie {
	return &BinaryTrie{
		root:  empty(struct{}{}),
		store: store(nil),
	}
}

func (t *BinaryTrie) Hash() []byte {
	return t.root.Hash()
}

func (t *BinaryTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (bt *BinaryTrie) subTreeFromKey(path binkey) *branch {
	subtrie := NewBinaryTrie()
	for _, keyval := range bt.store {
		// keyval.key is a full key from the store,
		// path is smaller - so the comparison only
		// happens on the length of path.
		if bytes.Equal(path, keyval.key[:len(path)]) {
			subtrie.TryUpdate(keyval.key, keyval.value)
		}
	}
	// NOTE panics if there are no entries for this
	// range in the store. This case must have been
	// caught with the empty check. Panicking is in
	// order otherwise. Which will lead whoever ran
	// into this bug to this comment. Hello :)
	rootbranch := subtrie.root.(*branch)
	// Remove the part of the prefix that is redundant
	// with the path, as it will be part of the resulting
	// root node's prefix.
	rootbranch.prefix = rootbranch.prefix[:len(path)]
	return rootbranch
}

func (bt *BinaryTrie) resolveNode(childNode BinaryNode, bk binkey, off int, value []byte) (*branch, bool) {

	// Check if the node has already been resolved,
	// otherwise, resolve it.
	switch childNode.(type) {
	case empty:
		if value == nil {
			return nil, true
		}

		// This child does not exist, create it
		return &branch{
			prefix: bk[off:],
			left:   hashBinaryNode(emptyRoot[:]),
			right:  hashBinaryNode(emptyRoot[:]),
			value:  value,
		}, true
	case hashBinaryNode:
		// empty root ?
		if bytes.Equal(childNode.(hashBinaryNode)[:], emptyRoot[:]) {
			if value != nil {
				return &branch{
					prefix: bk[off:],
					left:   hashBinaryNode(emptyRoot[:]),
					right:  hashBinaryNode(emptyRoot[:]),
					value:  value,
				}, true
			}
			return nil, true
		}

		// The whole section of the store has to be
		// hashed in order to produce the correct
		// subtrie.
		return bt.subTreeFromKey(bk[:off]), false
	}

	// Nothing to be done
	return childNode.(*branch), false
}

func (bt *BinaryTrie) TryUpdate(key, value []byte) error {
	bk := newBinKey(key)
	off := 0 // Number of key bits that've been walked at current iteration

	// Go through the storage, find the parent node to
	// insert this (key, value) into.
	var currentNode *branch
	switch bt.root.(type) {
	case empty:
		// This is when the trie hasn't been inserted
		// into, so initialize the root as a branch
		// node (a value, really).
		bt.root = &branch{
			prefix: bk,
			value:  value,
			left:   hashBinaryNode(emptyRoot[:]),
			right:  hashBinaryNode(emptyRoot[:]),
			key:    key,
		}
		bt.store = append(bt.store, storeSlot{key: bk, value: value})
		sort.Sort(bt.store)

		return nil
	case *branch:
		currentNode = bt.root.(*branch)
	case hashBinaryNode:
		return errInsertIntoHash
	}
	for {
		if bk.samePrefix(currentNode.prefix, off) {
			// The key matches the full node prefix, iterate
			// at  the child's level.
			var childNode *branch
			var mustBreak bool
			off += len(currentNode.prefix)
			if bk[off] == 0 {
				childNode, mustBreak = bt.resolveNode(currentNode.left, bk, off+1, value)
			} else {
				childNode, mustBreak = bt.resolveNode(currentNode.right, bk, off+1, value)
			}
			if mustBreak {
				break
			}

			// Update the parent node's reference
			if bk[off] == 0 {
				currentNode.left = childNode
			} else {
				currentNode.right = childNode
			}
			currentNode = childNode
			off += 1
		} else {
			split := bk[off:].commonLength(currentNode.prefix)
			// If the split is on either the first or last bit,
			// there is no need to create an intermediate node.

			// A split is needed
			midNode := &branch{
				prefix: currentNode.prefix[split+1:],
				left:   currentNode.left,
				right:  currentNode.right,
				key:    currentNode.key,
				value:  currentNode.value,
			}
			currentNode.prefix = currentNode.prefix[:split]
			currentNode.value = nil
			if bk[off+split] == 1 {
				// New node goes on the right
				currentNode.left = midNode
				currentNode.right = &branch{
					prefix: bk[off+split+1:],
					left:   hashBinaryNode(emptyRoot[:]),
					right:  hashBinaryNode(emptyRoot[:]),
					value:  value,
					key:    key,
				}
			} else {
				// New node goes on the left
				currentNode.right = midNode
				currentNode.left = &branch{
					prefix: bk[off+split+1:],
					left:   hashBinaryNode(emptyRoot[:]),
					right:  hashBinaryNode(emptyRoot[:]),
					value:  value,
					key:    key,
				}
			}
			break
		}
	}

	// Add the node to the store and make sure it's
	// sorted.
	bt.store = append(bt.store, storeSlot{
		key:   bk,
		value: value,
	})
	sort.Sort(bt.store)

	return nil
}

// Starting from the following context:
//
//          ...
// parent <                     child1
//          [ a b c d e ... ] <
//                ^             child2
//                |
//             cut-off
//
// This needs to be turned into:
//
//          ...                    child1
// parent <          [ d e ... ] <
//          [ a b ] <              child2
//                    child3
//
// where `c` determines which child is left
// or right.
//
// Both [ a b ] and [ d e ... ] can be empty
// prefixes.

//func dotHelper(prefix string, t *branch) ([]string, []string) {
//p := []byte{}
//for i := 0; i < t.getPrefixLen(); i++ {
//if t.getPrefixBit(i) {
//p = append(p, []byte("1")...)
//} else {
//p = append(p, []byte("0")...)
//}
//}
//typ := "node"
//if t.left == nil && t.right == nil {
//typ = "leaf"
//}
//nodeName := fmt.Sprintf("bin%s%s_%s", typ, prefix, p)
//nodes := []string{nodeName}
//links := []string{}
//if t.left != nil {
//if left, ok := t.left.(*branch); ok {
//n, l := dotHelper(fmt.Sprintf("%s%s%d", prefix, p, 0), left)
//nodes = append(nodes, n...)
//links = append(links, fmt.Sprintf("%s -> %s", nodeName, n[0]))
//links = append(links, l...)
//} else {
//nodes = append(nodes, fmt.Sprintf("hash%s", prefix))
//}
//}
//if t.right != nil {
//if right, ok := t.right.(*branch); ok {
//n, l := dotHelper(fmt.Sprintf("%s%s%d", prefix, p, 1), right)
//nodes = append(nodes, n...)
//links = append(links, fmt.Sprintf("%s -> %s", nodeName, n[0]))
//links = append(links, l...)
//} else {
//nodes = append(nodes, fmt.Sprintf("hash%s", prefix))
//}
//}
//return nodes, links
//}

// toDot creates a graphviz representation of the binary trie
//func (t *branch) toDot() string {
//nodes, links := dotHelper("", t)
//return fmt.Sprintf("digraph D {\nnode [shape=rect]\n%s\n%s\n}", strings.Join(nodes, "\n"), strings.Join(links, "\n"))
//}

// Commit stores all the values in the binary trie into the database.
// This version does not perform any caching, it is intended to perform
// the conversion from hexary to binary.
// It basically performs a hash, except that it makes sure that there is
// a channel to stream the intermediate (hash, preimage) values to.
func (t *branch) Commit() error {
	if t.CommitCh == nil {
		return fmt.Errorf("commit channel missing")
	}
	t.Hash()
	return nil
}

// Commit does not commit anything, because a hash doesn't have
// its accompanying preimage.
func (h hashBinaryNode) Commit() error {
	return nil
}

// Hash returns itself
func (h hashBinaryNode) Hash() []byte {
	return h
}

func (h hashBinaryNode) tryGet(key []byte, depth int) ([]byte, error) {
	if depth >= 8*len(key) {
		return []byte(h), nil
	}
	return nil, errReadFromEmptyTree
}

func (e empty) Hash() []byte {
	return emptyRoot[:]
}

func (e empty) Commit() error {
	return errors.New("can not commit empty node")
}

func (e empty) tryGet(key []byte, depth int) ([]byte, error) {
	return nil, errReadFromEmptyTree
}
