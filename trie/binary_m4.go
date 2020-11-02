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

	"github.com/ethereum/go-ethereum/log"
)

type BinaryTrieM4 struct {
	root     BinaryNode
	store    store
	hashType hashType
}

// All known implementations of binaryNode
type (
	// branch is a node with two children ("left" and "right")
	branchM4 struct {
		left  BinaryNode
		right BinaryNode

		key   []byte // TODO split into leaf and branch
		value []byte

		hType hashType
	}

	valueNodeM4 []byte

	emptyM4 struct{}
)

func (bt *BinaryTrieM4) TryGet(key []byte) ([]byte, error) {
	bk := newBinKey(key)
	off := 0

	var currentNode *branchM4
	switch bt.root.(type) {
	case emptyM4:
		return nil, errKeyNotPresent
	case *branch:
		currentNode = bt.root.(*branchM4)
	case hashBinaryNode:
		return nil, errReadFromHash
	}

	for {
		// If it is a leaf node, then the a leaf node
		// has been reached, and the value can be returned
		// right away.
		if currentNode.value != nil {
			return currentNode.value, nil
		}

		// This node is a fork, get the child node
		var childNode *branchM4
		if bk[off+1] == 0 {
			switch currentNode.left.(type) {
			case emptyM4:
				return nil, errKeyNotPresent
			case *branchM4:
				childNode = currentNode.left.(*branchM4)
			default:
				panic("not implemented")
			}
		} else {
			switch currentNode.right.(type) {
			case emptyM4:
				return nil, errKeyNotPresent
			case *branchM4:
				childNode = currentNode.right.(*branchM4)
			default:
				panic("not implemented")
			}
		}

		off++
		currentNode = childNode
	}
}

// Hash calculates the hash of an expanded (i.e. not already
// hashed) node.
func (br *branchM4) Hash() []byte {
	return br.hash(0)
}

func (br *branchM4) hash(off int) []byte {
	var hasher *hasher
	if br.hType == typeBlake2b {
		hasher = newB2Hasher(false)
		defer returnHasherToB2Pool(hasher)
	} else {
		hasher = newHasher(false)
		defer returnHasherToPool(hasher)
	}
	hasher.sha.Reset()

	// This is a branch node, so the rule is
	// branch_hash = hash(left_root_hash || right_root_hash)
	hasher.sha.Write(br.left.hash(0))
	hasher.sha.Write(br.right.hash(0))
	hash := hasher.sha.Sum(nil)
	hasher.sha.Reset()

	//fmt.Printf("hash %x\n", hash)
	return hash
}

func (vn valueNodeM4) Hash() []byte {
	return vn.hash(0)
}

func (vn valueNodeM4) hash(off int) []byte {
	hasher := newHasher(false)
	defer returnHasherToPool(hasher)
	hasher.sha.Reset()

	// This is a leaf node, so the hashing rule is
	// leaf_hash = hash(0 || hash(leaf_value))
	hasher.sha.Write(vn)
	hash := hasher.sha.Sum(nil)
	hasher.sha.Reset()
	//fmt.Printf("leaf hash = %x\n", hash)

	hasher.sha.Write(zero32)
	hasher.sha.Write(hash)
	hash = hasher.sha.Sum(nil)
	hasher.sha.Reset()

	return hash
}

func (vn valueNodeM4) Commit() error {
	return errors.New("not implemented")
}

func NewM4BinaryTrie() *BinaryTrieM4 {
	return &BinaryTrieM4{
		root:     emptyM4(struct{}{}),
		store:    store(nil),
		hashType: typeKeccak256,
	}
}

func NewM4BinaryTrieWithBlake2b() *BinaryTrieM4 {
	return &BinaryTrieM4{
		root:     emptyM4(struct{}{}),
		store:    store(nil),
		hashType: typeBlake2b,
	}
}

func (t *BinaryTrieM4) Hash() []byte {
	return t.root.Hash()
}

func (t *BinaryTrieM4) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (bt *BinaryTrieM4) subTreeFromKey(path binkey) *branch {
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

func (bt *BinaryTrieM4) TryUpdate(key, value []byte) error {
	bk := newBinKey(key)
	off := 0 // Number of key bits that've been walked at current iteration

	// Go through the storage, find the parent node to
	// insert this (key, value) into.
	var currentNode BinaryNode
	switch bt.root.(type) {
	case emptyM4:
		// This is when the trie hasn't been inserted
		// into, so initialize the root as a branch
		// node (a value, really).
		var childNode BinaryNode = valueNodeM4(value)
		for i := len(bk) - 1; i >= 0; i-- {
			if bk[i] == 0 {
				childNode = &branchM4{
					left:  childNode,
					right: emptyM4(struct{}{}),
					hType: bt.hashType,
				}
			} else {
				childNode = &branchM4{
					right: childNode,
					left:  emptyM4(struct{}{}),
					hType: bt.hashType,
				}
			}
		}

		bt.root = childNode
		return nil
	case *branchM4:
		currentNode = bt.root
	case hashBinaryNode:
		return errInsertIntoHash
	default:
		panic("unsupported type")
	}

	var parent *branchM4
	// Walk the trie until the first value / empty node
	currentBranch, ok := currentNode.(*branchM4)
	for ; ok; off++ {
		parent = currentBranch
		if bk[off] == 0 {
			currentNode = currentBranch.left
		} else {
			currentNode = currentBranch.right
		}
		currentBranch, ok = currentNode.(*branchM4)
	}

	switch currentNode.(type) {
	case valueNodeM4:
		panic("attempting to overwrite value")
	case emptyM4:
		var childNode BinaryNode = valueNodeM4(value)
		for i := len(bk) - 1; i >= off; i-- {
			if bk[i] == 0 {
				childNode = &branchM4{
					left:  childNode,
					right: emptyM4(struct{}{}),
					hType: bt.hashType,
				}
			} else {
				childNode = &branchM4{
					right: childNode,
					left:  emptyM4(struct{}{}),
					hType: bt.hashType,
				}
			}
		}

		if bk[off-1] == 0 {
			parent.left = childNode
		} else {
			parent.right = childNode
		}
		return nil
	}

	// Add the node to the store and make sure it's
	// sorted.
	//bt.store = append(bt.store, storeSlot{
	//key:   bk,
	//value: value,
	//})
	//sort.Sort(bt.store)

	return nil
}

// Commit stores all the values in the binary trie into the database.
// This version does not perform any caching, it is intended to perform
// the conversion from hexary to binary.
// It basically performs a hash, except that it makes sure that there is
// a channel to stream the intermediate (hash, preimage) values to.
func (t *branchM4) Commit() error {
	return errors.New("not implemented")
}

func (e emptyM4) Hash() []byte {
	return emptyRoot[:]
}

func (e emptyM4) hash(off int) []byte {
	return emptyRoot[:]
}

func (e emptyM4) Commit() error {
	return errors.New("can not commit empty node")
}

func (e emptyM4) tryGet(key []byte, depth int) ([]byte, error) {
	return nil, errReadFromEmptyTree
}
