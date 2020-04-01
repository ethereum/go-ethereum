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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/sha3"
)

type BinaryTrie struct {
	left  *BinaryTrie
	right *BinaryTrie
	value []byte
	db    ethdb.Database
}

func NewBinary(db ethdb.Database) (*BinaryTrie, error) {
	if db == nil {
		return nil, fmt.Errorf("trie.NewBinary called without a database")
	}

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
	bi := (by >> uint(depth%8)) & 1
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

func (t *BinaryTrie) TryUpdate(key, value []byte) error {
	// TODO check key depth
	err := t.insert(0, key, value)
	return err
}

func (t *BinaryTrie) insert(depth int, key, value []byte) error {
	// TODO hash intermediate nodes
	by := key[depth/8]
	bi := (by >> uint(depth%8)) & 1
	if bi == 0 {
		if t.left == nil {
			if depth == len(key)*8-2 {
				t.left = &BinaryTrie{nil, nil, value, t.db}
				return nil
			} else {
				t.left = &BinaryTrie{nil, nil, nil, t.db}
			}
		}
		return t.left.insert(depth+1, key, value)
	} else {
		if t.right == nil {
			if depth == len(key)*8-2 {
				t.right = &BinaryTrie{nil, nil, value, t.db}
				return nil
			} else {
				t.right = &BinaryTrie{nil, nil, nil, t.db}
			}
		}
		return t.right.insert(depth+1, key, value)
	}
}

func (t *BinaryTrie) Commit() ([]byte, error) {
	var payload [3][]byte
	var err error
	if t.left != nil {
		payload[0], err = t.left.Commit()
		if err != nil {
			return nil, err
		}
	}
	if t.right != nil {
		payload[1], err = t.right.Commit()
		if err != nil {
			return nil, err
		}
	}
	hasher := sha3.NewLegacyKeccak256()
	if t.value != nil {
		hasher.Write(t.value)
		hasher.Sum(payload[2][:])
	}

	hasher.Reset()
	hasher.Write(payload[0])
	hasher.Write(payload[1])
	hasher.Write(payload[2])
	h := hasher.Sum(nil)
	data := make([]byte, len(payload[0])+len(payload[1])+len(payload[2])+3)
	data[0] = byte(len(payload[0]))
	copy(data[1:], payload[0])
	data[len(payload[0])+1] = byte(len(payload[1]))
	copy(data[2+len(payload[0]):], payload[1])
	data[len(payload[0])+len(payload[1])+2] = byte(len(payload[2]))
	copy(data[2+len(payload[0])+len(payload[1]):], payload[2])

	t.db.Put(h, data)

	return h, err
}
