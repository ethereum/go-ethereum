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
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/fxamacker/cbor"
	"github.com/hashicorp/golang-lru"
)

// BinaryNode represents any node in a binary trie.
type BinaryNode interface {
	Hash() []byte
	HashM4() []byte
	hash(off int) []byte
	Commit() error

	gv(string) (string, string)
}

// BinaryHashPreimage represents a tuple of a hash and its preimage
type BinaryHashPreimage struct {
	Key   []byte
	Value []byte
}

type binkey []byte

type hashType int

const (
	typeKeccak256 hashType = iota
	typeBlake2b
)

var blake2bEmptyRoot = common.FromHex("45b0cfc220ceec5b7c1c62c4d4193d38e4eba48e8815729ce75f9c0ab0e4c1c0")

// BinaryTrie represents a multi-level binary trie.
//
// Nodes with only one child are compacted into a "prefix"
// for the first node that has two children.
type BinaryTrie struct {
	root     BinaryNode
	db       btDatabase
	hashType hashType
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

		hType hashType
	}

	hashBinaryNode []byte

	empty struct{}
)

type accountDirties struct {
	flags   byte
	balance *big.Int
	nonce   uint64
	code    []byte

	// list of dirty slots. The key to the slot
	// needs to be hashed.
	dirties map[common.Hash]common.Hash
}

type btDatabase struct {
	// dirty accounts and slot caches.
	dirties map[common.Hash]*accountDirties

	cache *lru.Cache

	diskdb ethdb.KeyValueStore

	lock sync.RWMutex
}

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
	for i, kb := range key {
		// might be best to have this statement first, as compiler bounds-checking hint
		bits[8*i+7] = kb & 0x1
		bits[8*i] = (kb >> 7) & 0x1
		bits[8*i+1] = (kb >> 6) & 0x1
		bits[8*i+2] = (kb >> 5) & 0x1
		bits[8*i+3] = (kb >> 4) & 0x1
		bits[8*i+4] = (kb >> 3) & 0x1
		bits[8*i+5] = (kb >> 2) & 0x1
		bits[8*i+6] = (kb >> 1) & 0x1
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
	length := min(len(b), len(other))
	for i := 0; i < length; i++ {
		if b[i] != other[i] {
			return i
		}
	}
	return length
}
func (b binkey) samePrefix(other binkey, off int) bool {
	return bytes.Equal(b[off:off+len(other)], other[:])
}

// TryGet returns the value for a key stored in the trie.
func (bt *BinaryTrie) TryGet(key []byte) ([]byte, error) {
	bk := newBinKey(key)
	off := 0

	var currentNode *branch
	switch bt.root.(type) {
	case empty:
		return nil, errKeyNotPresent
	case *branch:
		currentNode = bt.root.(*branch)
	case hashBinaryNode:
		return nil, errReadFromHash
	}

	for {
		if !bk.samePrefix(currentNode.prefix, off) {
			return nil, errKeyNotPresent
		}

		// If it is a leaf node, then the a leaf node
		// has been reached, and the value can be returned
		// right away.
		if currentNode.value != nil {
			return currentNode.value, nil
		}

		// This node is a fork, get the child node
		var childNode *branch
		if bk[off+len(currentNode.prefix)] == 0 {
			childNode = bt.resolveNode(currentNode.left, bk, off+1)
		} else {
			childNode = bt.resolveNode(currentNode.right, bk, off+1)
		}

		// if no child node could be found, the key
		// isn't present in the trie.
		if childNode == nil {
			return nil, errKeyNotPresent
		}
		off += len(currentNode.prefix) + 1
		currentNode = childNode
	}
}

func (bt *BinaryTrie) ToGraphViz() string {
	content, _ := bt.root.gv("")
	return fmt.Sprintf("digraph D {\nnode [shape=box]\n%s\n}", content)
}

func newBranchNode(prefix binkey, key []byte, value []byte, ht hashType) *branch {
	return &branch{
		prefix: prefix,
		left:   empty(struct{}{}),
		right:  empty(struct{}{}),
		key:    key,
		value:  value,
		hType:  ht,
	}
}

// Hash calculates the hash of an expanded (i.e. not already
// hashed) node.
func (br *branch) Hash() []byte {
	return br.hash(0)
}

func (br *branch) getHasher() *hasher {
	var hasher *hasher
	if br.hType == typeBlake2b {
		hasher = newB2Hasher(false)
	} else {
		hasher = newHasher(false)
	}
	hasher.sha.Reset()
	return hasher
}

func (br *branch) putHasher(hasher *hasher) {
	if br.hType == typeBlake2b {
		returnHasherToB2Pool(hasher)
	} else {
		returnHasherToPool(hasher)
	}
}

func (br *branch) hash(off int) []byte {
	var hasher *hasher
	var hash []byte
	if br.value == nil {
		// This is a branch node, so the rule is
		// branch_hash = hash(left_root_hash || right_root_hash)
		lh := br.left.hash(off + len(br.prefix) + 1)
		rh := br.right.hash(off + len(br.prefix) + 1)
		hasher = br.getHasher()
		defer br.putHasher(hasher)
		hasher.sha.Write(lh)
		hasher.sha.Write(rh)
		hash = hasher.sha.Sum(nil)
	} else {
		hasher = br.getHasher()
		defer br.putHasher(hasher)
		// This is a leaf node, so the hashing rule is
		// leaf_hash = hash(hash(key) || hash(leaf_value))
		hasher.sha.Write(br.key)
		kh := hasher.sha.Sum(nil)
		hasher.sha.Reset()

		hasher.sha.Write(br.value)
		hash = hasher.sha.Sum(nil)
		hasher.sha.Reset()

		hasher.sha.Write(kh)
		hasher.sha.Write(hash)
		hash = hasher.sha.Sum(nil)
	}

	if len(br.prefix) > 0 {
		hasher.sha.Reset()
		fpLen := len(br.prefix) + off
		hasher.sha.Write([]byte{byte(fpLen), byte(fpLen >> 8)})
		hasher.sha.Write(zero32[:30])
		hasher.sha.Write(hash)
		hash = hasher.sha.Sum(nil)
	}

	return hash
}

func (br *branch) HashM4() []byte {
	var hasher *hasher
	var hash []byte
	if br.value == nil {
		// This is a branch node, so the rule is
		// branch_hash = hash(left_root_hash || right_root_hash)
		lh := br.left.HashM4()
		rh := br.right.HashM4()
		hasher = br.getHasher()
		defer br.putHasher(hasher)
		hasher.sha.Write(lh)
		hasher.sha.Write(rh)
		hash = hasher.sha.Sum(nil)
	} else {
		hasher = br.getHasher()
		defer br.putHasher(hasher)
		// This is a leaf node, so the hashing rule is
		// leaf_hash = hash(0 || hash(leaf_value))
		hasher.sha.Write(br.value)
		hash = hasher.sha.Sum(nil)
		hasher.sha.Reset()

		hasher.sha.Write(zero32)
		hasher.sha.Write(hash)
		hash = hasher.sha.Sum(nil)
	}

	if len(br.prefix) > 0 {
		for i := range br.prefix {
			hasher.sha.Reset()
			if br.prefix[len(br.prefix)-1-i] != 0 {
				hasher.sha.Write(emptyRoot[:])
			}
			hasher.sha.Write(hash)
			if br.prefix[len(br.prefix)-1-i] == 0 {
				hasher.sha.Write(emptyRoot[:])
			}
			hash = hasher.sha.Sum(nil)
		}
	}

	return hash
}

func (br *branch) gv(path string) (string, string) {
	me := fmt.Sprintf("br%s", path)
	var l, r string
	switch br.left.(type) {
	case empty:
	default:
		leftPath := fmt.Sprintf("%s%x0", path, br.prefix)
		leftGV, leftName := br.left.gv(leftPath)
		l = fmt.Sprintf("%s -> %s\n%s", me, leftName, leftGV)
	}
	switch br.right.(type) {
	case empty:
	default:
		rightPath := fmt.Sprintf("%s%x1", path, br.prefix)
		rightGV, rightName := br.right.gv(rightPath)
		r = fmt.Sprintf("%s -> %s\n%s", me, rightName, rightGV)
	}
	return fmt.Sprintf("%s [label=\"%x:%x\"]\n%s%s", me, br.prefix, br.value, l, r), me
}

func (db *btDatabase) insert(key, value []byte) error {
	if len(key) != 32 && len(key) != 64 {
		return errors.New("bintrie: can only insert a value at depth 32 or 64 bytes")
	}

	// the only value associated to a key length of 512 bits is
	// if the subtree selector is 0b11.
	itemSelector := key[31] & 3
	if len(key) == 64 && itemSelector != 3 {
		return errors.New("bintrie: trying to write at an invalid depth")
	}

	var accountKey common.Hash
	copy(accountKey[:], key[:32])
	accountKey[31] &= 0xFC
	account, ok := db.dirties[accountKey]
	if !ok {
		account = &accountDirties{
			balance: big.NewInt(0),
			dirties: make(map[common.Hash]common.Hash),
		}
		db.dirties[accountKey] = account
	}

	switch itemSelector {
	case 0: // Balance
		account.balance.SetBytes(value)
	case 1: // nonce
		if len(value) > 8 {
			return errors.New("bintrie: tried to write a nonce larger than u64")
		}
		account.nonce = binary.BigEndian.Uint64(value)
	case 2: // code
		account.code = value
	case 3:
		if len(value) != 32 {
			return errors.New("bintrie: invalid value length in slot write")
		}
		slotKey := common.BytesToHash(key[32:64])
		account.dirties[slotKey] = common.BytesToHash(value)
	default:
		return errors.New("bintrie: range reserved for future use")
	}

	account.flags |= (1 << itemSelector)

	return nil
}

func binCacheEvictionCallback(db *btDatabase) func(interface{}, interface{}) {
	return func(key interface{}, value interface{}) {
		// XXX vérifier que la clé est 32 ou 64 bytes et pas 256 ou 512

		switch t := value.(type) {
		case *branch:
		default:
			panic(fmt.Sprintf("attempting to insert non-branch into the database, type = %v", t))
		}

		// Write the data to disk
		payload, err := cbor.Marshal(value, cbor.CanonicalEncOptions())
		if err != nil {
			panic(err)
		}
		db.diskdb.Put(key.([]byte), payload)
	}
}

// NewBinaryTrie creates a binary trie using Keccak256 for hashing.
func NewBinaryTrie() *BinaryTrie {
	bt := &BinaryTrie{
		root: empty(struct{}{}),
		db: btDatabase{
			diskdb:  rawdb.NewMemoryDatabase(),
			dirties: make(map[common.Hash]*accountDirties),
		},
		hashType: typeKeccak256,
	}
	bt.db.cache, _ = lru.NewWithEvict(1000, binCacheEvictionCallback(&bt.db))
	return bt
}

func NewBinaryTrieWithRawDB(db ethdb.KeyValueStore) *BinaryTrie {
	bt := &BinaryTrie{
		root: empty(struct{}{}),
		db: btDatabase{
			diskdb:  db,
			dirties: make(map[common.Hash]*accountDirties),
		},
		hashType: typeKeccak256,
	}
	bt.db.cache, _ = lru.NewWithEvict(1000, binCacheEvictionCallback(&bt.db))
	return bt
}

// NewBinaryTrieWithBlake2b creates a binary trie using Blake2b for hashing.
func NewBinaryTrieWithBlake2b() *BinaryTrie {
	bt := &BinaryTrie{
		root: empty(struct{}{}),
		db: btDatabase{
			diskdb:  rawdb.NewMemoryDatabase(),
			dirties: make(map[common.Hash]*accountDirties),
		},
		hashType: typeBlake2b,
	}
	bt.db.cache, _ = lru.NewWithEvict(1000, binCacheEvictionCallback(&bt.db))
	return bt
}

// Hash returns the root hash of the binary trie, with the merkelization
// rule described in EIP-3102.
func (bt *BinaryTrie) Hash() []byte {
	return bt.root.Hash()
}

// HashM4 returns the root hash of the binary trie, with the alternative
// merkelization rule M4.
func (bt *BinaryTrie) HashM4() []byte {
	return bt.root.HashM4()
}

// Update does the same thing as TryUpdate except it panics if it encounters
// an error.
func (bt *BinaryTrie) Update(key, value []byte) {
	if err := bt.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// subTreeFromPath rebuilds the subtrie rooted at path `path` from the db.
func (bt *BinaryTrie) subTreeFromPath(path binkey) *branch {
	subtrie := NewBinaryTrie()
	it := bt.db.diskdb.NewIterator(path, nil)
	for it.Next() {
		// TODO check if the length is 32, the last two
		// bits 00, and if so, insert the special account
		// node, in order to save disk space and memory.
		subtrie.TryUpdate(it.Key(), it.Value())
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

func (bt *BinaryTrie) resolveNode(childNode BinaryNode, bk binkey, off int) *branch {

	// Check if the node has already been resolved,
	// otherwise, resolve it.
	switch childNode := childNode.(type) {
	case empty:
		return nil
	case hashBinaryNode:
		// empty root ?
		if bytes.Equal(childNode[:], emptyRoot[:]) {
			return nil
		}

		// look for the hash in the cache, otherwise load
		// it from disk.
		if v, ok := bt.db.cache.Get([]byte(childNode)); ok {
			return v.(*branch)
		}

		p, err := bt.db.diskdb.Get([]byte(childNode))
		if err != nil {
			panic(fmt.Errorf("error reading key %x from the db: %v", childNode, err))
		}

		var b branch
		cbor.Unmarshal(p, &b)

		bt.db.cache.Add([]byte(childNode), b)
		return &b
	}

	// Nothing to be done
	return childNode.(*branch)
}

// TryUpdate will set the trie's leaf at `key` to `value`. If there is
// no leaf at `key`, it will be created.
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
		bt.root = newBranchNode(bk, key, value, bt.hashType)
		bt.db.insert(key, value)

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
			off += len(currentNode.prefix)
			if bk[off] == 0 {
				childNode = bt.resolveNode(currentNode.left, bk, off+1)
			} else {
				childNode = bt.resolveNode(currentNode.right, bk, off+1)
			}
			var isLeaf bool
			if childNode == nil {
				childNode = newBranchNode(bk[off+1:], nil, value, bt.hashType)
				isLeaf = true
			}

			// Update the parent node's reference
			if bk[off] == 0 {
				currentNode.left = childNode
			} else {
				currentNode.right = childNode
			}

			if isLeaf {
				break
			}
			currentNode = childNode
			off++
		} else {
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
			split := bk[off:].commonLength(currentNode.prefix)

			// A split is needed
			midNode := &branch{
				prefix: currentNode.prefix[split+1:],
				left:   currentNode.left,
				right:  currentNode.right,
				key:    currentNode.key,
				value:  currentNode.value,
				hType:  bt.hashType,
			}
			currentNode.prefix = currentNode.prefix[:split]
			currentNode.value = nil
			childNode := newBranchNode(bk[off+split+1:], key, value, bt.hashType)
			if bk[off+split] == 1 {
				// New node goes on the right
				currentNode.left = midNode
				currentNode.right = childNode
			} else {
				// New node goes on the left
				currentNode.right = midNode
				currentNode.left = childNode
			}

			bt.db.cache.Add(bk[:off+split+1], childNode)
			break
		}
	}

	// Update the list of dirty values.
	bt.db.insert(key, value)

	return nil
}

// Commit stores all the values in the binary trie into the database.
// This version does not perform any caching, it is intended to perform
// the conversion from hexary to binary.
// It basically performs a hash, except that it makes sure that there is
// a channel to stream the intermediate (hash, preimage) values to.
func (br *branch) Commit() error {
	if br.CommitCh == nil {
		return fmt.Errorf("commit channel missing")
	}
	br.Hash()
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
func (h hashBinaryNode) HashM4() []byte {
	return h
}

func (h hashBinaryNode) hash(off int) []byte {
	return h
}

func (h hashBinaryNode) tryGet(key []byte, depth int) ([]byte, error) {
	if depth >= 8*len(key) {
		return []byte(h), nil
	}
	return nil, errReadFromEmptyTree
}

func (h hashBinaryNode) gv(path string) (string, string) {
	me := fmt.Sprintf("h%s", path)
	return fmt.Sprintf("%s [label=\"H\"]\n", me), me
}

func (e empty) Hash() []byte {
	return emptyRoot[:]
}

func (e empty) HashM4() []byte {
	return emptyRoot[:]
}

func (e empty) hash(off int) []byte {
	return emptyRoot[:]
}

func (e empty) Commit() error {
	return errors.New("can not commit empty node")
}

func (e empty) tryGet(key []byte, depth int) ([]byte, error) {
	return nil, errReadFromEmptyTree
}

func (e empty) gv(path string) (string, string) {
	me := fmt.Sprintf("e%s", path)
	return fmt.Sprintf("%s [label=\"∅\"]\n", me), me
}
