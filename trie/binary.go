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
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

type binaryNode interface {
	Hash() []byte
	Commit() error
	insert(depth int, key, value []byte, hashLeft bool) error
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
		endBit   int // Technically, this is the "1st bit past the end"
	}
	hashBinaryNode []byte
)

var (
	errInsertIntoHash    = errors.New("trying to insert into a hash")
	errReadFromEmptyTree = errors.New("reached an empty subtree")
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
			return nil, fmt.Errorf("could not find key %s in trie depth=%d keylen=%d value=%v", common.ToHex(key), depth+i, len(key), t.value)
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

// Hash calculates the hash of an expanded (i.e. not already
// hashed) node.
func (t *BinaryTrie) Hash() []byte {
	return t.hash()
}

func (t *BinaryTrie) hash() []byte {
	var payload bytes.Buffer

	// Calculate the hash of both subtrees
	var lh, rh []byte
	if t.left != nil {
		lh = t.left.Hash()
	}
	t.left = hashBinaryNode(lh)
	if t.right != nil {
		rh = t.right.Hash()
	}
	t.right = hashBinaryNode(rh)

	// Create the "bitprefix" which indicates which are the start and
	// end bit inside the prefix value.
	rlp.Encode(&payload, []interface{}{t.bitPrefix(), lh, rh, t.value})

	hasher := sha3.NewLegacyKeccak256()
	io.Copy(hasher, &payload)
	return hasher.Sum(nil)
}

func (t *BinaryTrie) TryUpdate(key, value []byte) error {
	// TODO check key depth
	err := t.insert(0, key, value, true)
	return err
}

func (t *BinaryTrie) insert(depth int, key, value []byte, hashLeft bool) error {
	// Special case: the trie is empty
	if depth == 0 && t.left == nil && t.right == nil && len(t.prefix) == 0 {
		t.prefix = key
		t.value = value
		t.startBit = 0
		t.endBit = 8 * len(key)
		return nil
	}

	// Compare the current segment of the key with the prefix,
	// create an intermediate node if they are different.
	var i int
	for i = 0; i < t.getPrefixLen(); i++ {
		if getBit(key, depth+i) != t.getPrefixBit(i) {
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

			// Create the [ d e ... ] part
			oldChild, _ := NewBinary(t.db)
			oldChild.prefix = t.prefix
			oldChild.startBit = depth + i + 1
			oldChild.endBit = t.endBit
			oldChild.left = t.left
			oldChild.right = t.right

			// Create the child3 part
			newChild, _ := NewBinary(t.db)
			newChild.prefix = key
			newChild.startBit = depth + i + 1
			newChild.endBit = len(key) * 8
			newChild.value = value

			// reconfigure the [ a b ] part by just specifying
			// which one is the endbit (which could lead to a
			// 0-length [ a b ] part) and also which one of the
			// two children are left and right.
			t.endBit = depth + i
			if t.getPrefixBit(i) {
				// if the prefix is 1 then the new
				// child goes left and the old one
				// goes right.
				t.left = newChild
				t.right = oldChild
			} else {
				if hashLeft {
					oldChild.Commit()
					t.left = hashBinaryNode(oldChild.Hash())
				} else {
					t.left = oldChild
				}
				t.right = newChild
			}

			return nil
		}
	}

	if depth+i >= 8*len(key)-1 {
		t.value = value
		return nil
	}

	// No break in the middle of the extension prefix,
	// recurse into one of the children.
	child := &t.left
	isRight := getBit(key, depth+i)
	if isRight {
		child = &t.right

		// Free the space taken by the left branch as insert
		// will no longer visit it, this will free memory.
		if t.left != nil {
			t.left = hashBinaryNode(t.left.Hash())
		}
	}

	// Create the child if it doesn't exist, otherwise recurse
	if *child == nil {
		*child = &BinaryTrie{nil, nil, value, t.db, key, depth + i + 1, 8 * len(key)}
		return nil
	}
	return (*child).insert(depth+1+i, key, value, hashLeft)
}

func dotHelper(prefix string, t *BinaryTrie) ([]string, []string) {
	p := []byte{}
	for i := 0; i < t.getPrefixLen(); i++ {
		if t.getPrefixBit(i) {
			p = append(p, []byte("1")...)
		} else {
			p = append(p, []byte("0")...)
		}
	}
	typ := "node"
	if t.left == nil && t.right == nil {
		typ = "leaf"
	}
	nodeName := fmt.Sprintf("bin%s%s_%s", typ, prefix, p)
	nodes := []string{nodeName}
	links := []string{}
	if t.left != nil {
		if left, ok := t.left.(*BinaryTrie); ok {
			n, l := dotHelper(fmt.Sprintf("%s%s%d", prefix, p, 0), left)
			nodes = append(nodes, n...)
			links = append(links, fmt.Sprintf("%s -> %s", nodeName, n[0]))
			links = append(links, l...)
		} else {
			nodes = append(nodes, fmt.Sprintf("hash%s", prefix))
		}
	}
	if t.right != nil {
		if right, ok := t.right.(*BinaryTrie); ok {
			n, l := dotHelper(fmt.Sprintf("%s%s%d", prefix, p, 1), right)
			nodes = append(nodes, n...)
			links = append(links, fmt.Sprintf("%s -> %s", nodeName, n[0]))
			links = append(links, l...)
		} else {
			nodes = append(nodes, fmt.Sprintf("hash%s", prefix))
		}
	}
	return nodes, links
}

func (t *BinaryTrie) toDot() string {
	nodes, links := dotHelper("", t)
	return fmt.Sprintf("digraph D {\nnode [shape=rect]\n%s\n%s\n}", strings.Join(nodes, "\n"), strings.Join(links, "\n"))
}

func (t *BinaryTrie) Commit() error {
	var payload bytes.Buffer
	var err error

	payload.Write(t.prefix)

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

func (h hashBinaryNode) insert(depth int, key, value []byte, hashLeft bool) error {
	return errInsertIntoHash
}

func (h hashBinaryNode) tryGet(key []byte, depth int) ([]byte, error) {
	if depth >= 8*len(key) {
		return []byte(h), nil
	}
	return nil, errReadFromEmptyTree
}
