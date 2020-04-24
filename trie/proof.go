// Copyright 2015 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *Trie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	// Collect all nodes on the path to key.
	key = keybytesToHex(key)
	var nodes []node
	tn := t.root
	for len(key) > 0 && tn != nil {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				// The trie doesn't contain the key.
				tn = nil
			} else {
				tn = n.Val
				key = key[len(n.Key):]
			}
			nodes = append(nodes, n)
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
			nodes = append(nodes, n)
		case hashNode:
			var err error
			tn, err = t.resolveHash(n, nil)
			if err != nil {
				log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
				return err
			}
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	hasher := newHasher(false)
	defer returnHasherToPool(hasher)

	for i, n := range nodes {
		if fromLevel > 0 {
			fromLevel--
			continue
		}
		var hn node
		n, hn = hasher.proofHash(n)
		if hash, ok := hn.(hashNode); ok || i == 0 {
			// If the node's database encoding is a hash (or is the
			// root node), it becomes a proof element.
			enc, _ := rlp.EncodeToBytes(n)
			if !ok {
				hash = hasher.hashData(enc)
			}
			proofDb.Put(hash, enc)
		}
	}
	return nil
}

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *SecureTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	return t.trie.Prove(key, fromLevel, proofDb)
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProof(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader) (value []byte, err error) {
	key = keybytesToHex(key)
	wantHash := rootHash
	for i := 0; ; i++ {
		buf, _ := proofDb.Get(wantHash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash)
		}
		n, err := decodeNode(wantHash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err)
		}
		keyrest, cld := get(n, key, true)
		switch cld := cld.(type) {
		case nil:
			// The trie doesn't contain the key.
			return nil, nil
		case hashNode:
			key = keyrest
			copy(wantHash[:], cld)
		case valueNode:
			return cld, nil
		}
	}
}

// proofToPath converts a merkle proof to trie node path.
// The main purpose of this function is recovering a node
// path from the merkle proof stream. All necessary nodes
// will be resolved and leave the remaining as hashnode.
func proofToPath(rootHash common.Hash, root node, key []byte, proofDb ethdb.KeyValueReader) (node, error) {
	// resolveNode retrieves and resolves trie node from merkle proof stream
	resolveNode := func(hash common.Hash) (node, error) {
		buf, _ := proofDb.Get(hash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
		}
		n, err := decodeNode(hash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %v", err)
		}
		return n, err
	}
	// If the root node is empty, resolve it first
	if root == nil {
		n, err := resolveNode(rootHash)
		if err != nil {
			return nil, err
		}
		root = n
	}
	var (
		err           error
		child, parent node
		keyrest       []byte
		terminate     bool
	)
	key, parent = keybytesToHex(key), root
	for {
		keyrest, child = get(parent, key, false)
		switch cld := child.(type) {
		case nil:
			// The trie doesn't contain the key.
			return nil, errors.New("the node is not contained in trie")
		case *shortNode:
			key, parent = keyrest, child // Already resolved
			continue
		case *fullNode:
			key, parent = keyrest, child // Already resolved
			continue
		case hashNode:
			child, err = resolveNode(common.BytesToHash(cld))
			if err != nil {
				return nil, err
			}
		case valueNode:
			terminate = true
		}
		// Link the parent and child.
		switch pnode := parent.(type) {
		case *shortNode:
			pnode.Val = child
		case *fullNode:
			pnode.Children[key[0]] = child
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", pnode, pnode))
		}
		if terminate {
			return root, nil // The whole path is resolved
		}
		key, parent = keyrest, child
	}
}

// unsetInternal removes all internal node references(hashnode, embedded node).
// It should be called after a trie is constructed with two edge proofs. Also
// the given boundary keys must be the one used to construct the edge proofs.
//
// It's the key step for range proof. All visited nodes should be marked dirty
// since the node content might be modified. Besides it can happen that some
// fullnodes only have one child which is disallowed. But if the proof is valid,
// the missing children will be filled, otherwise it will be thrown anyway.
func unsetInternal(node node, left []byte, right []byte) error {
	left, right = keybytesToHex(left), keybytesToHex(right)

	// todo(rjl493456442) different length edge keys should be supported
	if len(left) != len(right) {
		return errors.New("inconsistent edge path")
	}
	// Step down to the fork point
	prefix, pos := prefixLen(left, right), 0
	for {
		if pos >= prefix {
			break
		}
		switch n := (node).(type) {
		case *shortNode:
			if len(left)-pos < len(n.Key) || !bytes.Equal(n.Key, left[pos:pos+len(n.Key)]) {
				return errors.New("invalid edge path")
			}
			n.flags = nodeFlag{dirty: true}
			node, pos = n.Val, pos+len(n.Key)
		case *fullNode:
			n.flags = nodeFlag{dirty: true}
			node, pos = n.Children[left[pos]], pos+1
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", node, node))
		}
	}
	fn, ok := node.(*fullNode)
	if !ok {
		return errors.New("the fork point must be a fullnode")
	}
	// Find the fork point! Unset all intermediate references
	for i := left[prefix] + 1; i < right[prefix]; i++ {
		fn.Children[i] = nil
	}
	fn.flags = nodeFlag{dirty: true}
	unset(fn.Children[left[prefix]], left[prefix+1:], false)
	unset(fn.Children[right[prefix]], right[prefix+1:], true)
	return nil
}

// unset removes all internal node references either the left most or right most.
func unset(root node, rest []byte, removeLeft bool) {
	switch rn := root.(type) {
	case *fullNode:
		if removeLeft {
			for i := 0; i < int(rest[0]); i++ {
				rn.Children[i] = nil
			}
			rn.flags = nodeFlag{dirty: true}
		} else {
			for i := rest[0] + 1; i < 16; i++ {
				rn.Children[i] = nil
			}
			rn.flags = nodeFlag{dirty: true}
		}
		unset(rn.Children[rest[0]], rest[1:], removeLeft)
	case *shortNode:
		rn.flags = nodeFlag{dirty: true}
		if _, ok := rn.Val.(valueNode); ok {
			rn.Val = nilValueNode
			return
		}
		unset(rn.Val, rest[len(rn.Key):], removeLeft)
	case hashNode, nil, valueNode:
		panic("it shouldn't happen")
	}
}

// VerifyRangeProof checks whether the given leave nodes and edge proofs
// can prove the given trie leaves range is matched with given root hash
// and the range is consecutive(no gap inside).
func VerifyRangeProof(rootHash common.Hash, keys [][]byte, values [][]byte, firstProof ethdb.KeyValueReader, lastProof ethdb.KeyValueReader) error {
	if len(keys) != len(values) {
		return fmt.Errorf("inconsistent proof data, keys: %d, values: %d", len(keys), len(values))
	}
	if len(keys) == 0 {
		return fmt.Errorf("nothing to verify")
	}
	if len(keys) == 1 {
		value, err := VerifyProof(rootHash, keys[0], firstProof)
		if err != nil {
			return err
		}
		if !bytes.Equal(value, values[0]) {
			return fmt.Errorf("correct proof but invalid data")
		}
		return nil
	}
	// Convert the edge proofs to edge trie paths. Then we can
	// have the same tree architecture with the original one.
	root, err := proofToPath(rootHash, nil, keys[0], firstProof)
	if err != nil {
		return err
	}
	// Pass the root node here, the second path will be merged
	// with the first one.
	root, err = proofToPath(rootHash, root, keys[len(keys)-1], lastProof)
	if err != nil {
		return err
	}
	// Remove all internal references. All the removed parts should
	// be re-filled(or re-constructed) by the given leaves range.
	if err := unsetInternal(root, keys[0], keys[len(keys)-1]); err != nil {
		return err
	}
	// Rebuild the trie with the leave stream, the shape of trie
	// should be same with the original one.
	newtrie := &Trie{root: root, db: NewDatabase(memorydb.New())}
	for index, key := range keys {
		newtrie.TryUpdate(key, values[index])
	}
	if newtrie.Hash() != rootHash {
		return fmt.Errorf("invalid proof, wanthash %x, got %x", rootHash, newtrie.Hash())
	}
	return nil
}

// get returns the child of the given node. Return nil if the
// node with specified key doesn't exist at all.
//
// There is an additional flag `skipResolved`. If it's set then
// all resolved nodes won't be returned.
func get(tn node, key []byte, skipResolved bool) ([]byte, node) {
	for {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.Val
			key = key[len(n.Key):]
			if !skipResolved {
				return key, tn
			}
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
			if !skipResolved {
				return key, tn
			}
		case hashNode:
			return key, n
		case nil:
			return key, nil
		case valueNode:
			return nil, n
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
}
