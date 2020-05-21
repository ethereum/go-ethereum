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
func proofToPath(rootHash common.Hash, root node, key []byte, proofDb ethdb.KeyValueReader, allowNonExistent bool) (node, error) {
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
	// If the root node is empty, resolve it first.
	// Root node must be included in the proof.
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
			// The trie doesn't contain the key. It's possible
			// the proof is a non-existing proof, but at least
			// we can prove all resolved nodes are correct, it's
			// enough for us to prove range.
			if allowNonExistent {
				return root, nil
			}
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
func unsetInternal(n node, left []byte, right []byte) error {
	left, right = keybytesToHex(left), keybytesToHex(right)

	// todo(rjl493456442) different length edge keys should be supported
	if len(left) != len(right) {
		return errors.New("inconsistent edge path")
	}
	// Step down to the fork point. There are two scenarios can happen:
	// - the fork point is a shortnode: the left proof MUST point to a
	//   non-existent key and the key doesn't match with the shortnode
	// - the fork point is a fullnode: the left proof can point to an
	//   existent key or not.
	var (
		pos    = 0
		parent node
	)
findFork:
	for {
		switch rn := (n).(type) {
		case *shortNode:
			// The right proof must point to an existent key.
			if len(right)-pos < len(rn.Key) || !bytes.Equal(rn.Key, right[pos:pos+len(rn.Key)]) {
				return errors.New("invalid edge path")
			}
			rn.flags = nodeFlag{dirty: true}
			// Special case, the non-existent proof points to the same path
			// as the existent proof, but the path of existent proof is longer.
			// In this case, the fork point is this shortnode.
			if len(left)-pos < len(rn.Key) || !bytes.Equal(rn.Key, left[pos:pos+len(rn.Key)]) {
				break findFork
			}
			parent = n
			n, pos = rn.Val, pos+len(rn.Key)
		case *fullNode:
			leftnode, rightnode := rn.Children[left[pos]], rn.Children[right[pos]]
			// The right proof must point to an existent key.
			if rightnode == nil {
				return errors.New("invalid edge path")
			}
			rn.flags = nodeFlag{dirty: true}
			if leftnode != rightnode {
				break findFork
			}
			parent = n
			n, pos = rn.Children[left[pos]], pos+1
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", n, n))
		}
	}
	switch rn := n.(type) {
	case *shortNode:
		if _, ok := rn.Val.(valueNode); ok {
			parent.(*fullNode).Children[right[pos-1]] = nil
			return nil
		}
		return unset(rn, rn.Val, right[pos:], len(rn.Key), true)
	case *fullNode:
		for i := left[pos] + 1; i < right[pos]; i++ {
			rn.Children[i] = nil
		}
		if err := unset(rn, rn.Children[left[pos]], left[pos:], 1, false); err != nil {
			return err
		}
		if err := unset(rn, rn.Children[right[pos]], right[pos:], 1, true); err != nil {
			return err
		}
		return nil
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// unset removes all internal node references either the left most or right most.
// If we try to unset all right most references, it can meet these scenarios:
//
// - The given path is existent in the trie, unset the associated shortnode
// - The given path is non-existent in the trie
//   - the fork point is a fullnode, the corresponding child pointed by path
//     is nil, return
//   - the fork point is a shortnode, the key of shortnode is less than path,
//     keep the entire branch and return.
//   - the fork point is a shortnode, the key of shortnode is greater than path,
//     unset the entire branch.
//
// If we try to unset all left most references, then the given path should
// be existent.
func unset(parent node, child node, key []byte, pos int, removeLeft bool) error {
	switch cld := child.(type) {
	case *fullNode:
		if removeLeft {
			for i := 0; i < int(key[pos]); i++ {
				cld.Children[i] = nil
			}
			cld.flags = nodeFlag{dirty: true}
		} else {
			for i := key[pos] + 1; i < 16; i++ {
				cld.Children[i] = nil
			}
			cld.flags = nodeFlag{dirty: true}
		}
		return unset(cld, cld.Children[key[pos]], key, pos+1, removeLeft)
	case *shortNode:
		if len(key[pos:]) < len(cld.Key) || !bytes.Equal(cld.Key, key[pos:pos+len(cld.Key)]) {
			// Find the fork point, it's an non-existent branch.
			if removeLeft {
				return errors.New("invalid right edge proof")
			}
			if bytes.Compare(cld.Key, key[pos:]) > 0 {
				// The key of fork shortnode is greater than the
				// path(it belongs to the range), unset the entrie
				// branch. The parent must be a fullnode.
				fn := parent.(*fullNode)
				fn.Children[key[pos-1]] = nil
			} else {
				// The key of fork shortnode is less than the
				// path(it doesn't belong to the range), keep
				// it with the cached hash available.
			}
			return nil
		}
		if _, ok := cld.Val.(valueNode); ok {
			fn := parent.(*fullNode)
			fn.Children[key[pos-1]] = nil
			return nil
		}
		cld.flags = nodeFlag{dirty: true}
		return unset(cld, cld.Val, key, pos+len(cld.Key), removeLeft)
	case nil:
		// If the node is nil, it's a child of the fork point
		// fullnode(it's an non-existent branch).
		if removeLeft {
			return errors.New("invalid right edge proof")
		}
		return nil
	default:
		panic("it shouldn't happen") // hashNode, valueNode
	}
}

// VerifyRangeProof checks whether the given leaf nodes and edge proofs
// can prove the given trie leaves range is matched with given root hash
// and the range is consecutive(no gap inside).
//
// Note the given first edge proof can be non-existing proof. For example
// the first proof is for an non-existent values 0x03. The given batch
// leaves are [0x04, 0x05, .. 0x09]. It's still feasible to prove. But the
// last edge proof should always be an existent proof.
//
// The firstKey is paired with firstProof, not necessarily the same as keys[0]
// (unless firstProof is an existent proof).
//
// Expect the normal case, this function can also be used to verify the following
// range proofs:
//
// - All elements proof. In this case the left and right proof can be nil, but the
//   range should be all the leaves in the trie.
//
// - Zero element proof(left edge proof should be a non-existent proof). In this
//   case if there are still some other leaves available on the right side, then
//   an error will be returned.
//
// - One element proof. In this case no matter the left edge proof is a non-existent
//   proof or not, we can always verify the correctness of the proof.
func VerifyRangeProof(rootHash common.Hash, firstKey []byte, keys [][]byte, values [][]byte, firstProof ethdb.KeyValueReader, lastProof ethdb.KeyValueReader) error {
	if len(keys) != len(values) {
		return fmt.Errorf("inconsistent proof data, keys: %d, values: %d", len(keys), len(values))
	}
	// Special case, there is no edge proof at all. The given range is expected
	// to be the whole leaf-set in the trie.
	if firstProof == nil && lastProof == nil {
		emptytrie, err := New(common.Hash{}, NewDatabase(memorydb.New()))
		if err != nil {
			return err
		}
		for index, key := range keys {
			emptytrie.TryUpdate(key, values[index])
		}
		if emptytrie.Hash() != rootHash {
			return fmt.Errorf("invalid proof, want hash %x, got %x", rootHash, emptytrie.Hash())
		}
		return nil
	}
	// Special case, there is a provided non-existence proof and zero key/value
	// pairs, meaning there are no more accounts / slots in the trie.
	if len(keys) == 0 {
		// Recover the non-existent proof to a path, ensure there is nothing left
		root, err := proofToPath(rootHash, nil, firstKey, firstProof, true)
		if err != nil {
			return err
		}
		node, pos, firstKey := root, 0, keybytesToHex(firstKey)
		for node != nil {
			switch rn := node.(type) {
			case *fullNode:
				for i := firstKey[pos] + 1; i < 16; i++ {
					if rn.Children[i] != nil {
						return errors.New("more leaves available")
					}
				}
				node, pos = rn.Children[firstKey[pos]], pos+1
			case *shortNode:
				if len(firstKey)-pos < len(rn.Key) || !bytes.Equal(rn.Key, firstKey[pos:pos+len(rn.Key)]) {
					if bytes.Compare(rn.Key, firstKey[pos:]) < 0 {
						node = nil
						continue
					} else {
						return errors.New("more leaves available")
					}
				}
				node, pos = rn.Val, pos+len(rn.Key)
			case valueNode, hashNode:
				return errors.New("more leaves available")
			}
		}
		// Yeah, although we receive nothing, but we can prove
		// there is no more leaf in the trie, return nil.
		return nil
	}
	// Special case, there is only one element and left edge
	// proof is an existent one.
	if len(keys) == 1 && bytes.Equal(keys[0], firstKey) {
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
	// For the first edge proof, non-existent proof is allowed.
	root, err := proofToPath(rootHash, nil, firstKey, firstProof, true)
	if err != nil {
		return err
	}
	// Pass the root node here, the second path will be merged
	// with the first one. For the last edge proof, non-existent
	// proof is not allowed.
	root, err = proofToPath(rootHash, root, keys[len(keys)-1], lastProof, false)
	if err != nil {
		return err
	}
	// Remove all internal references. All the removed parts should
	// be re-filled(or re-constructed) by the given leaves range.
	if err := unsetInternal(root, firstKey, keys[len(keys)-1]); err != nil {
		return err
	}
	// Rebuild the trie with the leave stream, the shape of trie
	// should be same with the original one.
	newtrie := &Trie{root: root, db: NewDatabase(memorydb.New())}
	for index, key := range keys {
		newtrie.TryUpdate(key, values[index])
	}
	if newtrie.Hash() != rootHash {
		return fmt.Errorf("invalid proof, want hash %x, got %x", rootHash, newtrie.Hash())
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
