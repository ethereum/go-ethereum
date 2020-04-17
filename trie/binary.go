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
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/sha3"
)

type binaryNode interface {
	Hash() []byte
	Commit() error
	insert(depth int, key, value []byte) error
	tryGet(key []byte, depth int) ([]byte, error)
}

type (
	BinaryTrie struct {
		left  binaryNode
		right binaryNode
		value []byte
		db    ethdb.Database

		// This is the binary equivalent of "extension nodes":
		// binary nodes can have a prefix that is common to all
		// subtrees. The prefix is defined by a series of bytes,
		// and two offsets marking the start bit and the end bit
		// of the range.
		prefix   []byte
		startBit int
		endBit   int
	}
	hashBinaryNode []byte
)

func NewBinary(db ethdb.Database) (*BinaryTrie, error) {

	return &BinaryTrie{db: db}, nil
}

func (t *BinaryTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

func (t *BinaryTrie) TryGet(key []byte) ([]byte, error) {
	value, err := t.tryGet(key, 0)
	return value, err
}

func (t *BinaryTrie) getPrefixLen() int {
	if t.endBit > t.startBit {
		return t.endBit - t.startBit
	}
	return 0
}

func getBit(key []byte, off int) bool {
	mask := byte(1) << (7 - uint(off)%8)

	return byte(key[uint(off)/8])&mask != byte(0)
}

func (t *BinaryTrie) getPrefixBit(bitnum int) bool {
	if bitnum > t.getPrefixLen() {
		panic(fmt.Sprintf("Trying to get bit #%d in a %d bit-long bitfield", bitnum, t.getPrefixLen()))
	}
	return getBit(t.prefix, t.startBit+bitnum)
}

func (t *BinaryTrie) tryGet(key []byte, depth int) ([]byte, error) {
	// Compare the key and the prefix. If they represent the
	// same bitfield, recurse. Otherwise, raise an error as
	// the value isn't present in this trie.
	var i int
	for i = 0; i < t.getPrefixLen(); i++ {
		if getBit(key, depth+i) != t.getPrefixBit(i) {
			return nil, fmt.Errorf("Key %v isn't present in this trie", key)
		}
	}

	// Exit condition: has the length of the key been reached?
	if depth+i == 8*len(key) {
		if t.value == nil {
			return nil, fmt.Errorf("Key %v isn't present in this trie", key)
		}
		return t.value, nil
	}

	// End of the key hasn't been reached, recurse into left or right
	// if the corresponding node is available.
	child := t.left
	isRight := getBit(key, depth+i)
	if isRight {
		child = t.right
	}

	if child == nil {
		if depth+i < len(key)*8-1 || t.value == nil {
			return nil, fmt.Errorf("could not find key 0x%s in trie %v %v %v", common.ToHex(key), depth+i, len(key), t.value)
		}
		return t.value, nil
	}
	return child.tryGet(key, depth+i+1)
}

func (t *BinaryTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (t *BinaryTrie) Hash() []byte {
	var payload bytes.Buffer

	var lh []byte
	if t.left != nil {
		lh = t.left.Hash()
	}
	payload.Write(lh)
	t.left = hashBinaryNode(lh)

	var rh []byte
	if t.right != nil {
		rh = t.right.Hash()
	}
	payload.Write(rh)
	t.right = hashBinaryNode(rh)

	hasher := sha3.NewLegacyKeccak256()
	if t.value != nil {
		hasher.Write(t.value)
		hv := hasher.Sum(nil)
		payload.Write(hv)
	}

	hasher.Reset()
	io.Copy(hasher, &payload)
	return hasher.Sum(nil)
}

func (t *BinaryTrie) TryUpdate(key, value []byte) error {
	// TODO check key depth
	err := t.insert(0, key, value)
	return err
}

func (t *BinaryTrie) insert(depth int, key, value []byte) error {
	by := key[depth/8]
	bi := (by >> uint(7-depth%8)) & 1
	if bi == 0 {
		if t.left == nil {
			switch depth {
			case len(key)*8 - 1:
				t.value = value
				return nil
			case len(key)*8 - 2:
				t.left = &BinaryTrie{nil, nil, value, t.db, nil, 0, 0}
				return nil
			default:
				t.left = &BinaryTrie{nil, nil, nil, t.db, nil, 0, 0}
			}
		}
		return t.left.insert(depth+1, key, value)
	} else {
		if t.right == nil {
			// Free the space taken by left branch as insert
			// will no longer visit it.
			if t.left != nil {
				h := t.left.Hash()
				t.left = hashBinaryNode(h)
			}

			switch depth {
			case len(key)*8 - 1:
				t.value = value
				return nil
			case len(key)*8 - 2:
				t.right = &BinaryTrie{nil, nil, value, t.db, nil, 0, 0}
				return nil
			default:
				t.right = &BinaryTrie{nil, nil, nil, t.db, nil, 0, 0}
			}
		}
		return t.right.insert(depth+1, key, value)
	}
}

func (t *BinaryTrie) Commit() error {
	var payload bytes.Buffer
	var err error

	var lh []byte
	if t.left != nil {
		lh = t.left.Hash()
		err := t.left.Commit()
		if err != nil {
			return err
		}
	}
	payload.Write(lh)
	t.left = hashBinaryNode(lh)

	var rh []byte
	if t.right != nil {
		rh = t.right.Hash()
		err := t.right.Commit()
		if err != nil {
			return err
		}
	}
	payload.Write(rh)
	t.right = hashBinaryNode(rh)

	hasher := sha3.NewLegacyKeccak256()
	if t.value != nil {
		hasher.Write(t.value)
		hv := hasher.Sum(nil)
		payload.Write(hv)
	}

	hasher.Reset()
	io.Copy(hasher, &payload)
	h := hasher.Sum(nil)

	err = t.db.Put(h, payload.Bytes())

	return err
}

func (h hashBinaryNode) Commit() error {
	return nil
}

func (h hashBinaryNode) Hash() []byte {
	return h
}

func (h hashBinaryNode) insert(depth int, key, value []byte) error {
	return fmt.Errorf("trying to insert into a hash")
}

func (h hashBinaryNode) tryGet(key []byte, depth int) ([]byte, error) {
	if depth == 2*len(key) {
		return []byte(h), nil
	}
	return nil, fmt.Errorf("reached an empty branch")
}
