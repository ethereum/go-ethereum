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
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
)

// ZkTrie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a ZkTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// ZkTrie is not safe for concurrent use.
type ZkTrie struct {
	tree *ZkTrieImpl
}

// NewSecure creates a trie
// SecureBinaryTrie bypasses all the buffer mechanism in *Database, it directly uses the
// underlying diskdb
func NewZkTrie(root common.Hash, db *ZktrieDatabase) (*ZkTrie, error) {
	rootHash, err := zkt.NewHashFromBytes(root.Bytes())
	if err != nil {
		return nil, err
	}
	tree, err := NewZkTrieImplWithRoot((db), rootHash, 256)
	if err != nil {
		return nil, err
	}
	return &ZkTrie{
		tree: tree,
	}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *ZkTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *ZkTrie) TryGet(key []byte) ([]byte, error) {
	word := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := word.Hash()
	if err != nil {
		return nil, err
	}

	return t.tree.TryGet(k.Bytes())
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *ZkTrie) TryGetNode(path []byte) ([]byte, int, error) {
	panic("unimplemented")
}

func (t *ZkTrie) updatePreimage(preimage []byte, hashField *big.Int) {
	db := t.tree.db.db
	if db.preimages != nil { // Ugly direct check but avoids the below write lock
		db.lock.Lock()
		// we must copy the input key
		db.insertPreimage(common.BytesToHash(hashField.Bytes()), common.CopyBytes(preimage))
		db.lock.Unlock()
	}
}

// TryUpdateAccount will abstract the write of an account to the
// secure trie.
func (t *ZkTrie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	keyPreimage := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := keyPreimage.Hash()
	if err != nil {
		return err
	}
	t.updatePreimage(key, k)
	return t.tree.TryUpdateAccount(k.Bytes(), acc)
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *ZkTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
//
// NOTE: value is restricted to length of bytes32.
func (t *ZkTrie) TryUpdate(key, value []byte) error {
	keyPreimage := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := keyPreimage.Hash()
	if err != nil {
		return err
	}
	t.updatePreimage(key, k)
	return t.tree.TryUpdate(k.Bytes(), value)
}

// Delete removes any existing value for key from the trie.
func (t *ZkTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *ZkTrie) TryDelete(key []byte) error {
	keyPreimage := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := keyPreimage.Hash()
	if err != nil {
		return err
	}

	//mitigate the create-delete issue: do not delete unexisted key
	if r := t.tree.Get(k.Bytes()); r == nil {
		return nil
	}

	// FIXME: when tryDelete get more solid test, use it instead
	return t.tree.tryDeleteLite(zkt.NewHashFromBigInt(k))
}

// GetKey returns the preimage of a hashed key that was
// previously used to store a value.
func (t *ZkTrie) GetKey(kHashBytes []byte) []byte {
	// TODO: use a kv cache in memory
	k, err := zkt.NewBigIntFromHashBytes(kHashBytes)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}

	return t.tree.db.db.preimage(common.BytesToHash(k.Bytes()))

}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *ZkTrie) Commit(LeafCallback) (common.Hash, int, error) {
	// in current implmentation, every update of trie already writes into database
	// so Commmit does nothing
	return t.Hash(), 0, nil
}

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *ZkTrie) Hash() common.Hash {
	var hash common.Hash
	hash.SetBytes(t.tree.rootKey.Bytes())
	return hash
}

// Copy returns a copy of SecureBinaryTrie.
func (t *ZkTrie) Copy() *ZkTrie {
	cpy, err := NewZkTrieImplWithRoot(t.tree.db, t.tree.rootKey, t.tree.maxLevels)
	if err != nil {
		panic("clone trie failed")
	}
	return &ZkTrie{
		tree: cpy,
	}
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *ZkTrie) NodeIterator(start []byte) NodeIterator {
	/// FIXME
	panic("not implemented")
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
/*func (t *ZkTrie) hashKey(key []byte) []byte {
	if len(key) != 32 {
		panic("non byte32 input to hashKey")
	}
	low16 := new(big.Int).SetBytes(key[:16])
	high16 := new(big.Int).SetBytes(key[16:])
	hash, err := poseidon.Hash([]*big.Int{low16, high16})
	if err != nil {
		panic(err)
	}
	return hash.Bytes()
}
*/

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *ZkTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	word := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := word.Hash()
	if err != nil {
		return err
	}
	err = t.tree.prove(zkt.NewHashFromBigInt(k), fromLevel, func(n *Node) error {
		key, err := n.Key()
		if err != nil {
			return err
		}

		if n.Type == NodeTypeLeaf {
			preImage := t.GetKey(n.NodeKey.Bytes())
			if len(preImage) > 0 {
				n.KeyPreimage = &zkt.Byte32{}
				copy(n.KeyPreimage[:], preImage)
				//return fmt.Errorf("key preimage not found for [%x] ref %x", n.NodeKey.Bytes(), k.Bytes())
			}
		}
		return proofDb.Put(key.Bytes(), n.Value())
	})
	if err != nil {
		return err
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, magicSMTBytes)
}
