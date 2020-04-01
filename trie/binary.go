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

func (t *BinaryTrie) tryGet(key []byte, depth int) ([]byte, error) {
	by := key[depth/8]
	bi := (by >> uint(7-depth%8)) & 1
	if bi == 0 {
		if t.left == nil {
			if depth < len(key)*8-1 || t.value == nil {
				return nil, fmt.Errorf("could not find key %s in trie", common.ToHex(key))
			}
			return t.value, nil
		}
		return t.left.tryGet(key, depth+1)
	} else {
		if t.right == nil {
			if depth < len(key)*8-1 || t.value == nil {
				return nil, fmt.Errorf("could not find key 0x%x in trie", key)
			}
			return t.value, nil
		}
		return t.right.tryGet(key, depth+1)
	}
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
				t.left = &BinaryTrie{nil, nil, value, t.db}
				return nil
			default:
				t.left = &BinaryTrie{nil, nil, nil, t.db}
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
				t.right = &BinaryTrie{nil, nil, value, t.db}
				return nil
			default:
				t.right = &BinaryTrie{nil, nil, nil, t.db}
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
