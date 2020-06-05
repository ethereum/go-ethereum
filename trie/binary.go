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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/crypto/sha3"
)

type binaryNode interface {
	Hash() []byte
	Commit() error
	insert(depth int, key, value []byte, hashLeft bool) error
	tryGet(key []byte, depth int) ([]byte, error)
}

// BinaryHashPreimage represents a tuple of a hash and its preimage
type BinaryHashPreimage struct {
	Key   []byte
	Value []byte
}

// All known implementations of binaryNode
type (
	// BinaryTrie is a node with two children ("left" and "right")
	// It can be prefixed by bits that are common to all subtrie
	// keys and it can also hold a value.
	BinaryTrie struct {
		left  binaryNode
		right binaryNode
		value []byte

		// Used to send (hash, preimage) pairs when hashing
		CommitCh chan BinaryHashPreimage

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

// getPrefixLen returns the bit length of the current node's prefix
func (t *BinaryTrie) getPrefixLen() int {
	if t.endBit > t.startBit {
		return t.endBit - t.startBit
	}
	return 0
}

// getBit returns the boolean value of bit at offset `off` in
// the byte key `key`
func getBit(key []byte, off int) bool {
	mask := byte(1) << (7 - uint(off)%8)

	return byte(key[uint(off)/8])&mask != byte(0)
}

// getPrefixBit returns the boolean value of bit number `bitnum`
// in the prefix of the current node.
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

// bitPrefix add the "bit prefix" to a key / extension node
func (t *BinaryTrie) bitPrefix() []byte {
	bp := make([]byte, 1+(t.getPrefixLen()+7)/8)
	for i := 0; i < t.getPrefixLen(); i++ {
		if t.getPrefixBit(i) {
			by := i / 8
			bi := 1 << uint(7-i%8)
			bp[1+by] |= byte(bi)
		}
	}
	if t.getPrefixLen() > 0 {
		bp[0] = byte(t.endBit-t.startBit) % 8
	}

	return bp
}

func Prefix2Bitfield(payload []byte) *BinaryTrie {
	return &BinaryTrie{
		prefix: payload[1:],
		endBit: int(payload[0]),
	}
}

func CheckKey(db *leveldb.DB, key, root []byte, depth int, value []byte) bool {
	node, err := db.Get(root, nil)
	if err != nil {
		log.Error("could not find the node!", "error", err)
		return false
	}

	var out BinaryDBNode
	err = rlp.DecodeBytes(node, &out)

	bt := Prefix2Bitfield(out.Bitprefix)
	fulldepth := depth
	if len(bt.prefix) > 0 {
		fulldepth += 8*len(bt.prefix) - (8-bt.endBit)%8
	}

	if fulldepth < 8*len(key) {
		by := key[fulldepth/8]
		bi := (by>>uint(7-(fulldepth%8)))&1 == 0
		if bi {
			if len(out.Left) == 0 {
				log.Error("key could not be found !")
				return false
			}

			return CheckKey(db, key, out.Left, fulldepth+1, value)
		} else {
			if len(out.Right) == 0 {
				log.Error("key could not be found ?")
				return false
			}

			return CheckKey(db, key, out.Right, fulldepth+1, value)
		}
	}

	return true // bytes.Equal(out.Value, value)
}

// Hash calculates the hash of an expanded (i.e. not already
// hashed) node.
func (t *BinaryTrie) Hash() []byte {
	return t.hash()
}

// BinaryDBNode represents a binary node as it is stored
// inside the DB.
type BinaryDBNode struct {
	Bitprefix []byte
	Left      []byte
	Right     []byte
	Value     []byte
}

// hash is a a helper function that is shared between Hash and
// Commit. If t.CommitCh is not nil, then its behavior will be
// that of Commit, and that of Hash otherwise.
func (t *BinaryTrie) hash() []byte {
	var payload bytes.Buffer

	// Calculate the hash of both subtrees
	var dbnode BinaryDBNode
	if t.left != nil {
		dbnode.Left = t.left.Hash()
		t.left = hashBinaryNode(dbnode.Left)
	}
	if t.right != nil {
		dbnode.Right = t.right.Hash()
		t.right = hashBinaryNode(dbnode.Right)
	}

	dbnode.Value = t.value
	dbnode.Bitprefix = t.bitPrefix()

	// Create the "bitprefix" which indicates which are the start and
	// end bit inside the prefix value.
	rlp.Encode(&payload, dbnode)
	value := payload.Bytes()

	hasher := sha3.NewLegacyKeccak256()
	io.Copy(hasher, &payload)
	h := hasher.Sum(nil)
	if t.CommitCh != nil {
		t.CommitCh <- BinaryHashPreimage{Key: h, Value: value}
	}
	return h
}

// TryUpdate inserts a (key, value) pair into the binary trie,
// and expects values to be inserted in order as inserting to
// the right of a node will cause the left node to be hashed.
func (t *BinaryTrie) TryUpdate(key, value []byte) error {
	// TODO check key depth
	err := t.insert(0, key, value, true)
	return err
}

// insert is a recursive helper function that inserts a (key, value) pair at
// a given depth. If hashLeft is true, inserting a key into a right subnode
// will cause the left subnode to be hashed.
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
			oldChild := new(BinaryTrie)
			oldChild.prefix = t.prefix
			oldChild.startBit = depth + i + 1
			oldChild.endBit = t.endBit
			oldChild.left = t.left
			oldChild.right = t.right
			oldChild.value = t.value
			oldChild.CommitCh = t.CommitCh

			// Create the child3 part
			newChild := new(BinaryTrie)
			newChild.prefix = key
			newChild.startBit = depth + i + 1
			newChild.endBit = len(key) * 8
			newChild.value = value
			newChild.CommitCh = t.CommitCh

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
				// if asked to, hash the left subtrie to free
				// up memory.
				if hashLeft {
					t.left = hashBinaryNode(oldChild.hash())
				} else {
					t.left = oldChild
				}
				t.right = newChild
			}
			t.value = nil

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
		*child = &BinaryTrie{nil, nil, value, nil, key, depth + i + 1, 8 * len(key)}
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

// toDot creates a graphviz representation of the binary trie
func (t *BinaryTrie) toDot() string {
	nodes, links := dotHelper("", t)
	return fmt.Sprintf("digraph D {\nnode [shape=rect]\n%s\n%s\n}", strings.Join(nodes, "\n"), strings.Join(links, "\n"))
}

// Commit stores all the values in the binary trie into the database.
// This version does not perform any caching, it is intended to perform
// the conversion from hexary to binary.
// It basically performs a hash, except that it makes sure that there is
// a channel to stream the intermediate (hash, preimage) values to.
func (t *BinaryTrie) Commit() error {
	if t.CommitCh == nil {
		return fmt.Errorf("commit channel missing")
	}
	t.hash()
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

func (h hashBinaryNode) insert(depth int, key, value []byte, hashLeft bool) error {
	return errInsertIntoHash
}

func (h hashBinaryNode) tryGet(key []byte, depth int) ([]byte, error) {
	if depth >= 8*len(key) {
		return []byte(h), nil
	}
	return nil, errReadFromEmptyTree
}
